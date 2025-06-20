package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/plhwin/gosocket/conf"
	"google.golang.org/protobuf/proto"
)

var (
	// Compressors supported
	Compressors = map[string]Compressor{
		conf.TransportCompressNone:   new(RawDataCompressor),
		conf.TransportCompressSnappy: new(SnappyCompressor),
		conf.TransportCompressFLate:  new(FLateCompressor),
		conf.TransportCompressGzip:   new(GzipCompressor),
	}
)

type Protocol struct {
	textProtocol TextProtocol
}

func (p *Protocol) SetProtocol(textProtocol TextProtocol) {
	if textProtocol == nil {
		textProtocol = new(DefaultTextProtocol)
	}
	p.textProtocol = textProtocol
}

// Encode message before send to socket server
func (p *Protocol) Encode(event string, args interface{}, id, serializeType, compressType string) (msg []byte, err error) {
	if event == "" {
		err = errors.New("event can not be empty")
		return
	}

	data := "" // interface to json string
	if args != nil {
		var b []byte
		if b, err = json.Marshal(&args); err != nil {
			return
		}
		data = string(b)
	}

	if serializeType == conf.TransportSerializeProtobuf {
		// transport serialize - Protobuf
		message := new(Message)
		message.Event = event
		message.Args = data
		message.Id = id
		if msg, err = proto.Marshal(message); err != nil {
			return
		}
	} else {
		// transport serialize - Text
		if msg, err = p.textProtocol.Encode(event, data, id); err != nil {
			return
		}
	}

	// compress - zip: after Encode
	return Compressors[compressType].Zip(msg)
}

// Decode message after received from socket server
func (p *Protocol) Decode(text []byte, serializeType, compressType string) (msg *Message, err error) {
	msg = new(Message)

	// compress - Unzip: before Decode
	if text, err = Compressors[compressType].Unzip(text); err != nil {
		return
	}

	if serializeType == conf.TransportSerializeProtobuf {
		// transport serialize - Protobuf
		if err = proto.Unmarshal(text, msg); err != nil {
			return
		}
	} else {
		// transport serialize - Text
		if err = p.textProtocol.Decode(text, msg); err != nil {
			return
		}
	}
	return
}

// EnPack 数据封包
// 前4个字节是消息长度，后面是消息内容
func EnPack(buf []byte) (pkg *bytes.Buffer, err error) {
	pkg = new(bytes.Buffer)

	// buffer length
	var length = int32(len(buf))

	// write length into pkg header
	// BigEndian:大端网络字节序
	err = binary.Write(pkg, binary.BigEndian, length)
	if err != nil {
		return
	}
	// write message body into pkg body
	err = binary.Write(pkg, binary.BigEndian, buf)
	if err != nil {
		return
	}
	return
}

// DePack 数据解包
// 数据解包传递过来的buf来自于接收端的io字节流，这个buf存在以下两种情况：
// 情况1：本次解包data中未能解析出至少一条正常协议数据，
// 这表示当前buf尚不足以按照传输协议解析出一条完整的消息，应等待下一次读取拼接字节流后再尝试；
// 情况2：本次解包data中包含一条或多条正常的协议数据，
// 这表示当前buf包含至少一条传输协议数据且被正确解析，通常是readBuf的长度比实际协议消息长度更长，
// 也可能是上面情况一预留在缓冲区(buf)里的数据，加上本次读取数据(readBuf)产生的效果，
// 两种情况都是TCP拆包的一个正常过程，readBuf每次读取字节流的大小决定了是情况1还是情况2
func DePack(buf *[]byte) (data [][]byte, err error) {
	// 消息前4个字节存储消息长度
	byteLen := 4
	for {
		if len(*buf) < byteLen {
			// buffer长度不足4，io流读取的字节流数据截断的"很巧"
			return
		}
		// 获取前4个字节存储的消息长度
		lenBuf := bytes.NewBuffer((*buf)[0:byteLen])
		var tmpLen int32
		if err = binary.Read(lenBuf, binary.BigEndian, &tmpLen); err != nil {
			return
		}
		rowLen := int(tmpLen)
		length := byteLen + rowLen

		if len(*buf) < length {
			// buffer长度小于传输协议的消息长度，等待下一次读取的字节流
			return
		}
		row := (*buf)[byteLen:length]
		data = append(data, row)
		// 剩余的未处理的buffer
		*buf = (*buf)[length:]
	}
}
