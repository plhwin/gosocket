package protocol

import (
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
func Decode(text []byte, transportProtocol string) (msg *Message, err error) {
	msg = new(Message)

	if transportProtocol == conf.TransportProtocolBinary {
		// transport protocol - binary
		if err = proto.Unmarshal(text, msg); err != nil {
			return
		}
	} else {
		// transport protocol - text
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
func Encode(event string, args interface{}, id, transportProtocol string) (msg []byte, err error) {
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

	if transportProtocol == conf.TransportProtocolBinary {
		// transport protocol - binary
		message := new(Message)
		message.Event = event
		message.Args = argStr
		message.Id = id

		if msg, err = proto.Marshal(message); err != nil {
			return
		}
	} else {
		// transport protocol - text
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
