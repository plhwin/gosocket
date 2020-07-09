package gosocket

import (
	"log"
	"net"

	"github.com/plhwin/gosocket/protocol"
)

type ConnFace interface {
	Init(*Initiator)                                           // init the Conn
	Emit(string, interface{}, string)                          // send message to the Conn
	EmitSync(string, interface{}, string) (interface{}, error) // Synchronize send message to the Conn
	EmitByAcceptor(*Acceptor, string, ArgsResponse, string)    // send message to socket client by acceptor instance
	Id() string                                                // get the Conn id
	RemoteAddr() net.Addr                                      // the ip:port of Conn
	Initiator() *Initiator                                     // get *Initiator
	Ping() map[int64]bool                                      // get ping
	Delay() int64                                              // obtain a time delay that reflects the quality of the connection between the two ends
	Out() chan string                                          // get the message send channel
	SetId(string)                                              // set conn id
	SetPing(map[int64]bool)                                    // set ping
	SetDelay(int64)                                            // set delay
	SetRemoteAddr(net.Addr)                                    // set remoteAddr
}

type Conn struct {
	id         string         // Conn id
	remoteAddr net.Addr       // Conn remoteAddr
	initiator  *Initiator     // event processing function register
	out        chan string    // message send channel
	ping       map[int64]bool // ping
	delay      int64          // delay
}

func (c *Conn) Init(i *Initiator) {
	c.initiator = i
	// set a capacity N for the data transmission pipeline as a buffer. if the Conn has not received it, the pipeline will always keep the latest N
	c.out = make(chan string, 500)
	c.ping = make(map[int64]bool)
}

func (c *Conn) Id() string {
	return c.id
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *Conn) Initiator() *Initiator {
	return c.initiator
}

func (c *Conn) Ping() map[int64]bool {
	return c.ping
}

func (c *Conn) Delay() int64 {
	return c.delay
}

func (c *Conn) Out() chan string {
	return c.out
}

func (c *Conn) SetId(v string) {
	c.id = v
}

func (c *Conn) SetPing(v map[int64]bool) {
	c.ping = v
}

func (c *Conn) SetDelay(v int64) {
	c.delay = v
}

func (c *Conn) SetRemoteAddr(v net.Addr) {
	c.remoteAddr = v
}

// Asynchronous Emit
func (c *Conn) Emit(event string, args interface{}, id string) {
	// This is a Insurance measures to avoid "send on closed channel" panic
	// This is a temporary measure
	// Usually due to non-compliance with the channel closing principle
	defer func() {
		if r := recover(); r != nil {
			log.Println("gosocket conn emit panic: ", r, c.Id(), c.RemoteAddr())
		}
	}()
	msg, err := protocol.Encode(event, args, id)
	if err != nil {
		log.Println("Emit encode error:", err, event, args, id, c.Id(), c.RemoteAddr())
		return
	}
	c.out <- msg
}

// Synchronize Emit
func (c *Conn) EmitSync(event string, args interface{}, id string) (result interface{}, err error) {
	return
}

func (c *Conn) EmitByAcceptor(a *Acceptor, event string, args ArgsResponse, id string) {
	a.Emit(args.Id, event, args.Args, id)
}
