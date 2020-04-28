package gosocket

import "log"

func NewServer() (s *Server) {
	s = new(Server)
	s.initEvents()
	s.initRooms()
	s.initClients()

	s.On("ping", ping)
	s.On("pong", pong)
	return
}

type Server struct {
	events
	rooms   *rooms
	clients map[string]*Client
	join    chan *Client
	leave   chan *Client
}

func (s *Server) initClients() {
	s.clients = make(map[string]*Client)
	s.join = make(chan *Client)
	s.leave = make(chan *Client)
	go s.admClients()
}

func (s *Server) admClients() {
	for {
		select {
		case c := <-s.join:
			s.clients[c.Id()] = c
			log.Println("a new client add to server:", c.Id(), c.RemoteAddr())
		case c := <-s.leave:
			if _, ok := s.clients[c.Id()]; ok {
				delete(s.clients, c.Id())
				log.Println("the client leave from server:", c.Id(), c.RemoteAddr())
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
