package gosocket

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/plhwin/gosocket/conf"

	"github.com/plhwin/gosocket/protocol"
)

type ClientFace interface {
	Init(*Acceptor)                                          // init the client
	Emit(string, interface{}, string)                        // send message to socket client
	EmitByInitiator(*Initiator, string, interface{}, string) // send message to socket server by initiator instance
	Join(room string)                                        // client join a room
	Leave(room string)                                       // client leave a room
	LeaveAllRooms()                                          // client leave all of the rooms
	Id() string                                              // get the client id
	RemoteAddr() net.Addr                                    // the ip:port of client
	Acceptor() *Acceptor                                     // get *Acceptor
	Rooms() map[string]bool                                  // get all rooms joined by the client
	Ping() map[int64]bool                                    // get ping
	Delay() int64                                            // obtain a time delay that reflects the quality of the connection between the two ends
	Out() chan string                                        // get the message send channel
	SetPing(map[int64]bool)                                  // set ping
	SetDelay(int64)                                          // set delay
	SetRemoteAddr(net.Addr)                                  // set remoteAddr
}

type Client struct {
	id         string         // client id
	remoteAddr net.Addr       // client remoteAddr
	acceptor   *Acceptor      // event processing function register
	rooms      *sync.Map      // map[string]bool all rooms joined by the client, used to quickly join and leave the rooms
	out        chan string    // message send channel
	ping       map[int64]bool // ping
	delay      int64          // delay
}

func (c *Client) Init(a *Acceptor) {
	c.id = c.genId()
	c.acceptor = a
	// set a capacity N for the data transmission pipeline as a buffer. if the client has not received it, the pipeline will always keep the latest N
	c.out = make(chan string, 500)
	c.rooms = new(sync.Map)
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
	rooms := make(map[string]bool)
	c.rooms.Range(func(k, v interface{}) bool {
		rooms[k.(string)] = v.(bool)
		return true
	})
	return rooms
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

func (c *Client) Emit(event string, args interface{}, id string) {
	// This is a Insurance measures to avoid "send on closed channel" panic
	// This is a temporary measure
	// Usually due to non-compliance with the channel closing principle
	defer func() {
		if r := recover(); r != nil {
			log.Println("gosocket client emit panic: ", r, c.Id(), c.RemoteAddr())
		}
	}()
	msg, err := protocol.Encode(event, args, id)
	if err != nil {
		log.Println("[GoSocket][Emit] encode error:", err, event, args, id, c.Id(), c.RemoteAddr())
		return
	}
	c.out <- msg
}

func (c *Client) EmitByInitiator(i *Initiator, event string, args interface{}, id string) {
	// Similar to the OSI network model,
	// add the socket client ID to re-packet args here, then send message to server by initiator instance
	var req ArgsRequest
	req.Id = c.Id()
	req.Args = args
	// send to socket server
	i.Emit(event, req, id)
}

func (c *Client) Join(room string) {
	c.acceptor.rooms.join <- roomClient{room, c}
	if conf.Acceptor.Logs.Room.Join {
		log.Println("[room][join]:", room, c.Id(), c.RemoteAddr())
	}
}

func (c *Client) Leave(room string) {
	c.acceptor.rooms.leave <- roomClient{room, c}
	if conf.Acceptor.Logs.Room.Leave {
		log.Println("[room][Leave]:", room, c.Id(), c.RemoteAddr())
	}
}

func (c *Client) LeaveAllRooms() {
	c.acceptor.rooms.leaveAll <- c
	if conf.Acceptor.Logs.Room.Leave {
		log.Println("[room][LeaveAll]:", c.Id(), c.RemoteAddr())
	}
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
