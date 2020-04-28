package gosocket

import (
	"time"
)

func NewServer() (s *Server) {
	s = new(Server)
	s.initEvents()
	s.initRooms()
	s.initClients()
	s.onConnection = s.onConn

	s.On(EventPing, s.ping)
	s.On(EventPong, s.pong)
	return
}

type Server struct {
	events
	rooms   *rooms
	clients map[string]*Client
	join    chan *Client
	leave   chan *Client
}

// the client initiate a ping and the server reply a pong
func (s *Server) ping(c ClientFace, arg int64) {
	c.Emit(EventPong, arg)
	return
}

// the client reply a pong, and the server initiate a ping
func (s *Server) pong(c ClientFace, arg int64) {
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
func (s *Server) onConn(f interface{}) {
	c := f.(ClientFace)
	c.Emit(EventSocketId, c.Id())
}

func (s *Server) initClients() {
	s.clients = make(map[string]*Client)
	s.join = make(chan *Client)
	s.leave = make(chan *Client)
	go s.manageClients()
}

func (s *Server) manageClients() {
	for {
		select {
		case c := <-s.join:
			s.clients[c.Id()] = c
		case c := <-s.leave:
			if _, ok := s.clients[c.Id()]; ok {
				delete(s.clients, c.Id())
			}
		}
	}
}

func (s *Server) initRooms() {
	s.rooms = newRooms()
	go s.rooms.Run()
}

func (s *Server) BroadcastTo(room, event string, args interface{}) {
	s.rooms.BroadcastTo(room, event, args)
}

func (s *Server) BroadcastToAll(event string, args interface{}) {
	for _, client := range s.clients {
		client.Emit(event, args)
	}
}

func (s *Server) Emit(id, event string, args interface{}) {
	if client, ok := s.clients[id]; ok {
		client.Emit(event, args)
	}
}
