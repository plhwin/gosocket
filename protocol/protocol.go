package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/plhwin/gosocket/conf"
)

type counter struct {
	start      int
	end        int
	rest       int
	countQuote int
}

func (c *counter) count(i int, s int32) (done bool, err error) {
	if s == '"' {
		switch c.countQuote {
		case 0:
			c.start = i + 1
		case 1:
			c.end = i
			c.rest = i + 1
		default:
			err = errors.New("wrong message quote")
			return
		}
		c.countQuote++
	}
	if s == ',' {
		if c.countQuote == 2 {
			c.rest = i + 1
			done = true
		}
	}
	return
}

func cutFromLeft(text string) (left, right string, err error) {
	var c counter
	// loop from left to right
	for i, s := range text {
		var done bool
		if done, err = c.count(i, s); err != nil {
			return
		}
		if done {
			break
		}
	}
	if (c.end < c.start) || (c.rest >= len(text)) {
		err = errors.New("wrong message len")
		return
	}
	left = text[c.start:c.end]
	right = text[c.rest : len(text)-1]
	return
}

func cutFromRight(text string) (left, right string, err error) {
	var c counter
	bytes := []rune(text)
	// loop from right to left
	for i := len(bytes) - 1; i >= 0; i-- {
		var done bool
		if done, err = c.count(i, bytes[i]); err != nil {
			return
		}
		if done {
			break
		}
	}
	if (c.end+1 > c.start-1) || (c.rest < 1) {
		err = errors.New("wrong message len")
		return
	}
	left = string(bytes[:c.rest-1])
	right = string(bytes[c.end+1 : c.start-1])
	return
}

// Parse message ["$event",$args,"$identity"]
func Decode(text []byte, serializeType string) (msg *Message, err error) {
	msg = new(Message)

	if serializeType == conf.TransportSerializeProtobuf {
		// transport serialize - Protobuf
		if err = proto.Unmarshal(text, msg); err != nil {
			return
		}
	} else {
		// transport serialize - Text
		var event, args string
		if event, args, err = cutFromLeft(string(text)); err != nil {
			return
		}
		if event == "" {
			err = errors.New("wrong message format")
			return
		}
		msg.Event, msg.Args = event, args
		// if end with "(quote), it is possible to carry the client id
		// it means that the data type of the client id must be a string
		if strings.HasSuffix(msg.Args, "\"") {
			if left, right, cutErr := cutFromRight(msg.Args); cutErr == nil {
				if msg.Args != "\""+right+"\"" {
					msg.Args = left
					msg.Id = right
				}
			}
		}
	}
	return
}

// The message is sent to the client in the format of the agreed protocol
func Encode(event string, args interface{}, id, serializeType string) (msg []byte, err error) {
	if event == "" {
		err = errors.New("event can not be empty")
		return
	}

	// args: interface to json string
	argStr := ""
	if args != nil {
		var argBytes []byte
		if argBytes, err = json.Marshal(&args); err != nil {
			return
		}
		argStr = string(argBytes)
	}

	if serializeType == conf.TransportSerializeProtobuf {
		// transport serialize - Protobuf
		message := new(Message)
		message.Event = event
		message.Args = argStr
		message.Id = id

		if msg, err = proto.Marshal(message); err != nil {
			return
		}
	} else {
		// transport serialize - Text
		body := "\"" + event + "\""
		if argStr != "" {
			body += "," + argStr
		}
		if id != "" {
			body += ",\"" + id + "\""
		}
		msg = []byte("[" + body + "]")
	}
	return
}

// 数据封包 - 前4个字节是消息长度，后面是消息内容
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

// 数据解包
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
