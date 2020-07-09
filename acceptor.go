package gosocket

import (
	"log"
	"time"

	"github.com/plhwin/gosocket/conf"
)

func NewAcceptor() (a *Acceptor) {
	a = new(Acceptor)
	a.initEvents()
	a.initRooms()
	a.initClients()
	a.onConnection = a.onConn

	a.On(EventPing, a.ping)
	a.On(EventPong, a.pong)
	return
}

type Acceptor struct {
	events
	rooms   *rooms
	clients map[string]ClientFace
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
		c.SetPing(make(map[int64]bool))
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
	a.clients = make(map[string]ClientFace)
	a.join = make(chan ClientFace)
	a.leave = make(chan ClientFace)
	go a.manageClients()
}

func (a *Acceptor) manageClients() {
	for {
		select {
		case c := <-a.join:
			a.clients[c.Id()] = c
		case c := <-a.leave:
			if _, ok := a.clients[c.Id()]; ok {
				delete(a.clients, c.Id())
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

func (a *Acceptor) Leave(c ClientFace) {
	a.leave <- c
}

func (a *Acceptor) BroadcastTo(room, event string, args interface{}, id string) {
	a.rooms.BroadcastTo(room, event, args, id)
}

func (a *Acceptor) BroadcastToAll(event string, args interface{}, id string) {
	for _, client := range a.clients {
		client.Emit(event, args, id)
	}
}

func (a *Acceptor) Emit(clientId, event string, args interface{}, id string) {
	if client, ok := a.clients[clientId]; ok {
		client.Emit(event, args, id)
	}
}

func (a *Acceptor) Client(clientId string) (c ClientFace, ok bool) {
	c, ok = a.clients[clientId]
	return
}

func (a *Acceptor) Clients() map[string]ClientFace {
	return a.clients
}

func (a *Acceptor) ClientsByRoom(room string) (clientFaces []ClientFace) {
	if v, ok := a.rooms.clients.Load(room); ok {
		for client := range v.(map[*Client]bool) {
			// What we need is the clientFace that injected by the user
			if clientFace, ok := a.Client(client.id); ok {
				clientFaces = append(clientFaces, clientFace)
			}
		}
	}
	return
}
