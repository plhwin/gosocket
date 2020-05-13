package gosocket

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/plhwin/gosocket/protocol"
)

type ClientFace interface {
	Init(*Acceptor)                                  // init the client
	Emit(string, interface{})                        // send message to socket client
	EmitByInitiator(*Initiator, string, interface{}) // send message to socket server by initiator instance
	Join(room string)                                // client join a room
	Leave(room string)                               // client leave a room
	LeaveAllRooms()                                  // client leave all of the rooms
	Id() string                                      // get the client id
	RemoteAddr() net.Addr                            // the ip:port of client
	Acceptor() *Acceptor                             // get *Acceptor
	Rooms() map[string]bool                          // get all rooms joined by the client
	Ping() map[int64]bool                            // get ping
	Delay() int64                                    // obtain a time delay that reflects the quality of the connection between the two ends
	Out() chan string                                // get the message send channel
	SetPing(map[int64]bool)                          // set ping
	SetDelay(int64)                                  // set delay
	SetRemoteAddr(net.Addr)                          // set remoteAddr
}

type Client struct {
	id         string          // client id
	remoteAddr net.Addr        // client remoteAddr
	acceptor   *Acceptor       // event processing function register
	rooms      map[string]bool // all rooms joined by the client, used to quickly join and leave the rooms
	out        chan string     // message send channel
	ping       map[int64]bool  // ping
	delay      int64           // delay
}

func (c *Client) Init(a *Acceptor) {
	c.id = c.genId()
	c.acceptor = a
	// set a capacity N for the data transmission pipeline as a buffer. if the client has not received it, the pipeline will always keep the latest N
	c.out = make(chan string, 200)
	c.rooms = make(map[string]bool)
	c.ping = make(map[int64]bool)
}

func (c *Client) Id() string {
	return c.id
}

func (c *Client) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *Client) Acceptor() *Acceptor {
	return c.acceptor
}

func (c *Client) Rooms() map[string]bool {
	return c.rooms
}

func (c *Client) Ping() map[int64]bool {
	return c.ping
}

func (c *Client) Delay() int64 {
	return c.delay
}

func (c *Client) Out() chan string {
	return c.out
}

func (c *Client) SetPing(v map[int64]bool) {
	c.ping = v
}

func (c *Client) SetDelay(v int64) {
	c.delay = v
}

func (c *Client) SetRemoteAddr(v net.Addr) {
	c.remoteAddr = v
}

func (c *Client) EmitByInitiator(i *Initiator, event string, args interface{}) {
	// Similar to the OSI network model,
	// add the socket client ID to re-packet args here, then send message to server by initiator instance
	var req ArgsRequest
	req.Id = c.Id()
	req.Args = args
	// send to socket server
	i.Emit(event, req)
}

func (c *Client) Emit(event string, args interface{}) {
	msg, err := protocol.Encode(event, args)
	if err != nil {
		log.Println("Emit encode error:", err, event, args, c.Id(), c.RemoteAddr())
		return
	}
	select {
	// transfer data without blocking
	case c.out <- msg:
	default:
		log.Println("client Emit error:", c.Id(), c.RemoteAddr(), msg)
		// the capacity of the data transmission pipeline is full,
		// may be the current network connection is of poor quality,
		// remove the client from all rooms,
		// And close the data transmission channel.(let the client re-initiate a new connection)
		//c.LeaveAllRooms()

		// @todo: in large-scale data testing, LeaveAllRooms was blocked here, need further observation and testing
	}
}

func (c *Client) Join(room string) {
	c.acceptor.rooms.join <- roomClient{room, c}
	log.Println("client join room:", room, c.Id(), c.RemoteAddr())
}

func (c *Client) Leave(room string) {
	c.acceptor.rooms.leave <- roomClient{room, c}
	log.Println("client Leave room:", room, c.Id(), c.RemoteAddr())
}

func (c *Client) LeaveAllRooms() {
	c.acceptor.rooms.leaveAll <- c
	log.Println("client Leave all of the rooms:", c.Id(), c.RemoteAddr())
}

func (c *Client) genId() string {
	custom := c.remoteAddr.String()
	hash := fmt.Sprintf("%s %s %n %n", custom, time.Now(), rand.Uint32(), rand.Uint32())
	buf := bytes.NewBuffer(nil)
	sum := md5.Sum([]byte(hash))
	encoder := base64.NewEncoder(base64.URLEncoding, buf)
	encoder.Write(sum[:])
	encoder.Close()
	return buf.String()[:20]
}
