package gosocket

import (
	"log"
	"time"
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
func (a *Acceptor) ping(c ClientFace, arg int64) {
	c.Emit(EventPong, arg)
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
	return
}

// When the socket connection was established,
// the server send the socket id to the client immediately
func (a *Acceptor) onConn(f interface{}) {
	c := f.(ClientFace)
	c.Emit(EventSocketId, c.Id())
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
			log.Println("[acceptor] join:", c.Id())
		case c := <-a.leave:
			if _, ok := a.clients[c.Id()]; ok {
				delete(a.clients, c.Id())
				log.Println("[acceptor] leave:", c.Id())
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

func (a *Acceptor) BroadcastTo(room, event string, args interface{}) {
	a.rooms.BroadcastTo(room, event, args)
}

func (a *Acceptor) BroadcastToAll(event string, args interface{}) {
	for _, client := range a.clients {
		client.Emit(event, args)
	}
}

func (a *Acceptor) Emit(id, event string, args interface{}) {
	if client, ok := a.clients[id]; ok {
		client.Emit(event, args)
	}
}

func (a *Acceptor) Client(id string) (c ClientFace, ok bool) {
	c, ok = a.clients[id]
	return
}

func (a *Acceptor) Clients() map[string]ClientFace {
	return a.clients
}

func (a *Acceptor) ClientsByRoom(room string) (clientFaces []ClientFace) {
	if clients, ok := a.rooms.clients[room]; ok {
		for clientFace := range clients {
			clientFaces = append(clientFaces, clientFace)
		}
	}
	return
}
