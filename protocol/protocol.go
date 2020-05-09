package protocol

import (
	"encoding/json"
	"errors"
)

type Message struct {
	Event string
	Args  string
}

// Parse out events and args from the message (if args exist)
func Decode(text string) (msg *Message, err error) {
	var start, end, rest, countQuote int
	msg = new(Message)

	for i, c := range text {
		if c == '"' {
			switch countQuote {
			case 0:
				start = i + 1
			case 1:
				end = i
				rest = i + 1
			default:
				err = errors.New("wrong msg with quote")
				return
			}
			countQuote++
		}
		if c == ',' {
			if countQuote < 2 {
				continue
			}
			rest = i + 1
			break
		}
	}
	if (end < start) || (rest >= len(text)) {
		err = errors.New("wrong msg with len")
		return
	}

	msg.Event = text[start:end]
	msg.Args = text[rest : len(text)-1]

	return
}

// The message is sent to the client in the format of the agreed protocol
// The args of the event processing function cannot be a struct
// if want json, recommended map
func Encode(event string, args interface{}) (msg string, err error) {
	body := "\"" + event + "\""
	if args != nil {
		var jsonArgs []byte
		jsonArgs, err = json.Marshal(&args)
		if err != nil {
			return
		}
		if strArgs := string(jsonArgs); strArgs != "" {
			body += "," + strArgs
		}
	}
	msg = "[" + body + "]"
	return
}
