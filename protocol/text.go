package protocol

import (
	"errors"
	"strings"
)

// TextProtocol defines a common text of protocol interface.
type TextProtocol interface {
	Encode(string, string, string) []byte
	Decode([]byte, *Message) error
}

// DefaultTextProtocol implements default text of protocol
type DefaultTextProtocol struct {
}

func (p *DefaultTextProtocol) Encode(event string, data, id string) []byte {
	body := `"` + event + `"`
	if data != "" {
		body += `,` + data
	}
	if id != "" {
		body += `,"` + id + `"`
	}
	return []byte(`[` + body + `]`)
}

// Decode Parse message as ["$event",$args,"$identity"]
func (p *DefaultTextProtocol) Decode(text []byte, msg *Message) (err error) {
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
	if strings.HasSuffix(msg.Args, `"`) {
		if left, right, cutErr := cutFromRight(msg.Args); cutErr == nil {
			if msg.Args != `"`+right+`"` {
				msg.Args = left
				msg.Id = right
			}
		}
	}
	return
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
	if (c.end+1 > c.start-1) || (c.rest < 1) {
		err = errors.New("wrong message len")
		return
	}
	left = string(bytes[:c.rest-1])
	right = string(bytes[c.end+1 : c.start-1])
	return
}
