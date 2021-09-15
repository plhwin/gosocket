package test

import (
	"bytes"
	"fmt"
	"testing"
)

func TestAppendByte(t *testing.T) {
	var msgEnd byte = '\n'

	msg := "这是test消息！"
	msgByte := []byte(msg)

	addString := msg + string(msgEnd)
	addByte := append(msgByte, msgEnd)

	fmt.Println("msg:", msg)
	fmt.Println("msgEnd:", msgEnd, string(msgEnd), string([]byte{msgEnd}))
	fmt.Println("msgByte:", msgByte, len(msgByte), string(msgByte))

	fmt.Println("addString:", addString)
	fmt.Println("addByte:", addByte, len(addByte), string(addByte))

	// remove msgEnd
	removeByte := bytes.TrimSuffix(addByte, []byte{msgEnd})
	fmt.Println("removeByte:", removeByte, len(removeByte), string(removeByte))
}
