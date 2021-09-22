package gosocket

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/plhwin/gosocket/protocol"
)

const (
	OnConnection    = "connection"
	OnDisconnection = "disconnection"
	EventSocketId   = "socket:id"
	EventPing       = "ping"
	EventPong       = "pong"
)

/**
System handler function for internal event processing
*/
type systemHandler func(c interface{})

type events struct {
	messageHandlers     map[string]*caller
	messageHandlersLock sync.RWMutex
	onConnection        systemHandler
	onDisconnection     systemHandler
}

func (e *events) initEvents() {
	e.messageHandlers = make(map[string]*caller)
}

// bind the event processing function
func (e *events) On(event string, f interface{}) error {
	c, err := newCaller(f)
	if err != nil {
		return err
	}

	e.messageHandlersLock.Lock()
	defer e.messageHandlersLock.Unlock()
	e.messageHandlers[event] = c

	return nil
}

// find the event handler function from the map of event handler functions registered to the system
func (e *events) findEvent(event string) (*caller, bool) {
	e.messageHandlersLock.RLock()
	defer e.messageHandlersLock.RUnlock()

	f, ok := e.messageHandlers[event]
	return f, ok
}

func (e *events) CallGivenEvent(c interface{}, event string) {
	if e.onConnection != nil && event == OnConnection {
		e.onConnection(c)
	}
	if e.onDisconnection != nil && event == OnDisconnection {
		e.onDisconnection(c)
	}
	f, ok := e.findEvent(event)
	if !ok {
		return
	}
	f.callFunc(c, &struct{}{}, "")
}

// call event processing function by incoming message
func (e *events) CallEvent(client interface{}, msg *protocol.Message) {
	f, ok := e.findEvent(msg.Event)
	if !ok {
		// the system does not register a event process function,
		// do nothing here (equivalent to ignoring this request initiated by the client)
		return
	}

	var args interface{}
	var id string

	if f.ArgsPresent {
		// the second input parameter with registered event handler function
		// the data type of the second parameter passed by the event handler function
		args = f.getArgs()
		if msg.Args != "" {
			msg.Args = strings.Trim(msg.Args, " ")
			if err := json.Unmarshal([]byte(msg.Args), &args); err != nil {
				log.Println("json decode error:", msg.Args, args, err)
				// if decode error, not return here
				// The second parameter of the event processing function will be zero value,
				// suggest that your system handles it yourself
			}
		}
	} else {
		args = &struct{}{}
	}

	if f.IdPresent {
		// the third input parameter with registered event handler function
		id = msg.Id
	}

	// 如果服务端处理某个具体客户端的某个具体事件需要耗费大量时间，
	// 如果这里不并发处理，该客户端在事件处理完成前，会无法接受和响应客户端的其他事件（如：心跳，test等），
	// 没有及时处理客户端的心跳，则会导致该客户端重连
	// @todo 并发安全性大规模测试
	go f.callFunc(client, args, id)
}
