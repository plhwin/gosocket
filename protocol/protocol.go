package protocol

import (
	"encoding/json"
	"errors"
	"strings"
)

type Message struct {
	Event string
	Args  string
	Id    string
}

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
	if (c.end+1 > c.start-1) || (c.rest < 0) {
		err = errors.New("wrong message len")
		return
	}
	left = string(bytes[:c.rest-1])
	right = string(bytes[c.end+1 : c.start-1])
	return
}

// Parse message ["$event",$args,"$identity"]
func Decode(text string) (msg *Message, err error) {
	msg = new(Message)

	var event, args string
	if event, args, err = cutFromLeft(text); err != nil {
		return
	}
	if event == "" {
		err = errors.New("wrong message format")
		return
	}
	msg.Event = event
	// if end with "(quote), it is possible to carry the client id
	// it means that the data type of the client id must be a string
	if strings.HasSuffix(args, "\"") {
		var left, id string
		if left, id, err = cutFromRight(args); err != nil {
			return
		}
		if args == "\""+id+"\"" {
			msg.Args = args
		} else {
			msg.Args = left
			msg.Id = id
		}
	} else {
		msg.Args = args
	}
	return
}

// The message is sent to the client in the format of the agreed protocol
func Encode(event string, args interface{}, id string) (msg string, err error) {
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
	if id != "" {
		body += "," + id
	}
	msg = "[" + body + "]"
	return
}
