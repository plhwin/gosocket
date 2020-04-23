package protocol

import (
	"encoding/json"
	"errors"
)

// 客户端发送过来的消息
type Message struct {
	Event string // 事件
	Args  string // 参数
}

// 从消息中解析出event和参数（如果参数存在）
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
				err = errors.New("Wrong msg")
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
		err = errors.New("Wrong msg")
		return
	}

	msg.Event = text[start:end]
	msg.Args = text[rest : len(text)-1]

	return
}

// 消息以约定协议的格式发往客户端
// 如果想以标准格式的json串发往客户端，则args可以处理成一个map传入，如果args直接传入struct，这里会解析成空不能达成目的
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
