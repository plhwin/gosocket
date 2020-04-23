package gosocket

func NewServer() (s *Server) {
	s = new(Server)
	s.initEvents()
	s.initRooms()

	s.On("ping", ping)
	s.On("pong", pong)
	return
}

type Server struct {
	events
	rooms *rooms
}

func (s *Server) initRooms() {
	s.rooms = newRooms()
	go s.rooms.Run()
}

func (s *Server) BroadcastTo(room, event string, args interface{}) {
	s.rooms.BroadcastTo(room, event, args)
}
