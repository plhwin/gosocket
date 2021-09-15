package gosocket

import (
	"log"
	"sync"
	"time"

	"github.com/plhwin/gosocket/protocol"

	"github.com/plhwin/gosocket/conf"
)

func NewAcceptor() (a *Acceptor) {
	a = new(Acceptor)
	a.initEvents()
	a.initRooms()
	a.initClients()
	a.onConnection = a.onConn
	a.SetProtocol(nil) // set default protocol

	a.On(EventPing, a.ping)
	a.On(EventPong, a.pong)
	return
}

type Acceptor struct {
	events
	protocol.Protocol
	rooms   *rooms
	clients *sync.Map // map[string]ClientFace
	join    chan ClientFace
	leave   chan ClientFace
}

// the client initiate a ping and the server reply a pong
func (a *Acceptor) ping(c ClientFace, arg int64, id string) {
	c.Emit(EventPong, arg, id)
	if conf.Acceptor.Logs.Heartbeat.PingReceive {
		log.Println("[heartbeat][acceptor][ping]:", c.Id(), c.RemoteAddr(), c.Delay())
	}
	return
}

// the client reply a pong, and the server initiate a ping
func (a *Acceptor) pong(c ClientFace, arg int64) {
	if _, ok := c.Ping()[arg]; ok {
		millisecond := time.Now().UnixNano() / int64(time.Millisecond)
		// to achieve a "continuous" effect, clear the container immediately after receiving any response
		c.ClearPing()
		// update the value of delay
		c.SetDelay(millisecond - arg)
	}
	if conf.Acceptor.Logs.Heartbeat.PongReceive {
		log.Println("[heartbeat][acceptor][pong]:", c.Id(), c.RemoteAddr(), c.Delay())
	}
	return
}

// When the socket connection was established,
// the server send the socket id to the client immediately
func (a *Acceptor) onConn(f interface{}) {
	c := f.(ClientFace)
	c.Emit(EventSocketId, c.Id(), "")
}

func (a *Acceptor) initClients() {
	a.clients = new(sync.Map)
	a.join = make(chan ClientFace)
	a.leave = make(chan ClientFace)
	go a.manageClients()
}

func (a *Acceptor) manageClients() {
	for {
		select {
		case c := <-a.join:
			a.clients.Store(c.Id(), c)
		case c := <-a.leave:
			if _, ok := a.clients.Load(c.Id()); ok {
				a.clients.Delete(c.Id())
			}
		}
	}
}

func (a *Acceptor) initRooms() {
	a.rooms = newRooms()
	go a.rooms.Run()
}

func (a *Acceptor) Join(c ClientFace) {
	a.join <- c
}

func (a *Acceptor) BroadcastTo(room, event string, args interface{}, id string) {
	a.rooms.BroadcastTo(room, event, args, id)
}

func (a *Acceptor) BroadcastToAll(event string, args interface{}, id string) {
	for _, client := range a.Clients() {
		client.Emit(event, args, id)
	}
}

func (a *Acceptor) Emit(clientId, event string, args interface{}, id string) {
	if v, ok := a.clients.Load(clientId); ok {
		v.(ClientFace).Emit(event, args, id)
	}
}

func (a *Acceptor) Client(clientId string) (c ClientFace, ok bool) {
	var v interface{}
	if v, ok = a.clients.Load(clientId); ok {
		c = v.(ClientFace)
	}
	return
}

func (a *Acceptor) Clients() map[string]ClientFace {
	clients := make(map[string]ClientFace)
	a.clients.Range(func(k, v interface{}) bool {
		clients[k.(string)] = v.(ClientFace)
		return true
	})
	return clients
}

func (a *Acceptor) ClientsByRoom(room string) (clientFaces []ClientFace) {
	if v, ok := a.rooms.clients.Load(room); ok {
		v.(*sync.Map).Range(func(k, _ interface{}) bool {
			// What we need is the clientFace that injected by the user
			if clientFace, ok := a.Client(k.(*Client).id); ok {
				clientFaces = append(clientFaces, clientFace)
			}
			return true
		})
	}
	return
}
