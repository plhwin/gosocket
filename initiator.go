package gosocket

import (
	"log"
	"sync"
	"time"

	"github.com/plhwin/gosocket/conf"
)

func NewInitiator() (i *Initiator) {
	i = new(Initiator)
	i.initEvents()
	i.onDisconnection = i.onDisConn

	i.On(EventSocketId, i.socketId)
	i.On(EventPing, i.ping)
	i.On(EventPong, i.pong)
	return
}

type Initiator struct {
	events
	conn  ConnFace
	alive bool
	mu    sync.RWMutex // mutex
}

func (i *Initiator) SetConn(c ConnFace) {
	i.conn = c
	i.setAlive(true)
}

func (i *Initiator) onDisConn(interface{}) {
	i.setAlive(false)
}

func (i *Initiator) Alive() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.alive
}

func (i *Initiator) setAlive(b bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.alive = b
}

func (i *Initiator) Emit(event string, args interface{}, id string) {
	if i.Alive() {
		i.conn.Emit(event, args, id)
	}
}

// After receive the SocketId event, call OnConnection
func (i *Initiator) socketId(c ConnFace, id string) {
	c.SetId(id)
	c.Initiator().CallGivenEvent(c, OnConnection)
}

// the server initiate a ping and the client reply a pong
func (i *Initiator) ping(c ConnFace, arg int64, id string) {
	c.Emit(EventPong, arg, id)
	if conf.Initiator.Logs.Heartbeat.PingReceive {
		log.Println("[heartbeat][initiator][ping]:", c.Id(), c.RemoteAddr(), c.Delay())
	}
	return
}

// the server reply a pong, and the client initiate a ping
func (i *Initiator) pong(c ConnFace, arg int64) {
	if _, ok := c.Ping()[arg]; ok {
		millisecond := time.Now().UnixNano() / int64(time.Millisecond)
		// to achieve a "continuous" effect, clear the container immediately after receiving any response
		c.SetPing(make(map[int64]bool))
		// update the value of delay
		c.SetDelay(millisecond - arg)
	}
	if conf.Initiator.Logs.Heartbeat.PongReceive {
		log.Println("[heartbeat][initiator][pong]:", c.Id(), c.RemoteAddr(), c.Delay())
	}
	return
}
