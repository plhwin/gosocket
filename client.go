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
	Init(*Server)             // init the client
	Emit(string, interface{}) // send a message to the client
	Join(room string)         // client join a room
	Leave(room string)        // client leave a room
	LeaveAllRooms()           // client leave all of the rooms
	JoinServer()              // client join to server
	LeaveServer()             // client leave server
	Close()                   // client close
	Id() string               // get the client id
	RemoteAddr() net.Addr     // the ip:port of client
	Server() *Server          // get *Server
	Rooms() map[string]bool   // get all rooms joined by the client
	Ping() map[int64]bool     // get ping
	Delay() int64             // obtain a time delay that reflects the quality of the connection between the two ends
	Out() chan string         // get the message send channel
	SetPing(map[int64]bool)   // set ping
	SetDelay(int64)           // set delay
	SetRemoteAddr(net.Addr)   // set remoteAddr
}

type Client struct {
	id         string          // client id
	remoteAddr net.Addr        // client remoteAddr
	server     *Server         // event processing function register
	rooms      map[string]bool // all rooms joined by the client, used to quickly join and leave the rooms
	out        chan string     // message send channel
	ping       map[int64]bool  // ping
	delay      int64           // delay
}

func (c *Client) Init(s *Server) {
	c.id = c.genId()
	c.server = s
	// set a capacity N for the data transmission pipeline as a buffer. if the client has not received it, the pipeline will always keep the latest N
	c.out = make(chan string, 500)
	c.rooms = make(map[string]bool)
	c.ping = make(map[int64]bool)

	c.JoinServer()
}

func (c *Client) Id() string {
	return c.id
}

func (c *Client) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *Client) Server() *Server {
	return c.server
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
		log.Println("Emit send error:", c.Id(), c.RemoteAddr(), msg)
		// the capacity of the data transmission pipeline is full,
		// may be the current network connection is of poor quality,
		// remove the client from all rooms,
		// And close the data transmission channel.(let the client re-initiate a new connection)
		c.server.rooms.leaveAll <- c
	}
}

func (c *Client) Join(room string) {
	c.server.rooms.join <- roomClient{room, c}
	log.Println("client join room:", room, c.Id(), c.RemoteAddr())
}

func (c *Client) Leave(room string) {
	c.server.rooms.leave <- roomClient{room, c}
	log.Println("client Leave room:", room, c.Id(), c.RemoteAddr())
}

func (c *Client) LeaveAllRooms() {
	c.server.rooms.leaveAll <- c
	log.Println("client Leave all of the rooms:", c.Id(), c.RemoteAddr())
}

func (c *Client) JoinServer() {
	c.server.join <- c
}

func (c *Client) LeaveServer() {
	c.server.leave <- c
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
