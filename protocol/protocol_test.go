package protocol

import (
	fmt "fmt"
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
)

func TestProtobuf(t *testing.T) {
	msg := &Message{
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

	msgNew := new(Message)
	err = proto.Unmarshal(data, msgNew)
	if err != nil {
		fmt.Println("unmarshal err:", err)
		os.Exit(2)
	}
	fmt.Println("unmarshal:", msgNew)
}
