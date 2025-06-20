package test

import (
	"bytes"
	"encoding/json"
	fmt "fmt"
	"os"
	"strings"
	"testing"

	"github.com/plhwin/gosocket/protocol"

	"github.com/plhwin/gosocket/conf"

	"google.golang.org/protobuf/proto"
)

const (
	msgEnd byte = '\n'
)

type ArgsRequest struct {
	Id   string `json:"id"`
	Args Args   `json:"args"`
}

type ArgsCommon struct {
	Server int64  `json:"server"`
	Token  string `json:"token"`
}

type Args struct {
	ArgsCommon
	Symbol string `json:"symbol"`
	Period string `json:"period"`
	From   int64  `json:"from"`
	To     int64  `json:"to"`
	Count  int    `json:"count"`
}

func TestProtobuf(t *testing.T) {
	msg := &protocol.Message{
		Event: "event:req",
		Args:  `{"result":true,"message":"ok","data":{"a":1,"b":"b","c":1.02,"d":false}}`,
		Id:    "0OZAnMtGfvroGnDCusTPdiM7r1CPocUv",
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		fmt.Println("marshal error:", err.Error())
		os.Exit(1)
	}
	fmt.Println("marshal:", data)

	data = append(data, msgEnd)

	fmt.Println("dataadd:", data)

	data = bytes.TrimSuffix(data, []byte{msgEnd})
	fmt.Println("datarem:", data)

	msgNew := new(protocol.Message)
	err = proto.Unmarshal(data, msgNew)
	if err != nil {
		fmt.Println("unmarshal err:", err)
		os.Exit(2)
	}
	fmt.Println("unmarshal:", msgNew)
}

func TestEncodeAndDecode(t *testing.T) {
	var p protocol.Protocol
	p.SetProtocol(new(protocol.DefaultTextProtocol))

	// "kline",{"id":"VsL7ZQOTe60hM_GYvtc8","args":{"server":1,"token":"","symbol":"EURUSD","period":"M1","from":1590422400,"to":0,"count":300}},"DtUwOg67X4V6AHHXOPxvYwWwM6we3RWU"
	event := "kline"
	args := `{"server":1,"token":"","symbol":"EURUSD","period":"M1","from":1590422400,"to":0,"count":300}`
	id := "DtUwOg67X4V6AHHXOPxvYwWwM6we3RWU"

	var err error
	var req ArgsRequest
	req.Id = "VsL7ZQOTe60hM_GYvtc8"
	if err = json.Unmarshal([]byte(args), &req.Args); err != nil {
		fmt.Println("args Unmarshal error:", err)
		os.Exit(1)
	}

	fmt.Printf("req: %+v %+v %+v \n\n", req, event, id)

	var msg []byte
	msg, err = p.Encode(event, req, id, conf.TransportSerializeProtobuf, conf.TransportCompressNone)
	if err != nil {
		fmt.Println("Encode error:", err, event, req, id)
		os.Exit(2)
	}

	fmt.Println("msg Encode:", msg)

	msg = append(msg, msgEnd)
	fmt.Println("msgadd:", msg)

	msg = bytes.TrimSuffix(msg, []byte{msgEnd})
	fmt.Println("msgrem:", msg)

	var message *protocol.Message
	message, err = p.Decode(msg, conf.TransportSerializeProtobuf, conf.TransportCompressNone)
	if err != nil {
		fmt.Println("Decode error:", err, event, req, id)
		os.Exit(3)
	}
	fmt.Println("msg Decode:", message)

	var reqNew ArgsRequest
	if message.Args != "" {
		message.Args = strings.Trim(message.Args, " ")
		if err := json.Unmarshal([]byte(message.Args), &reqNew); err != nil {
			fmt.Println("json decode error:", message.Args, reqNew)
			os.Exit(4)
		}
	}

	fmt.Printf("req: %+v \n\n", reqNew)

}
