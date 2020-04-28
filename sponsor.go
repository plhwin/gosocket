package gosocket

import "time"

func NewSponsor(c ConnFace) (s *Sponsor) {
	s = new(Sponsor)
	s.initEvents()
	s.conn = c

	s.On(EventSocketId, s.socketId)
	s.On(EventPing, s.ping)
	s.On(EventPong, s.pong)
	return
}

type Sponsor struct {
	events
	conn ConnFace
}

func (s *Sponsor) Emit(event string, args interface{}) {
	s.conn.Emit(event, args)
}

// After receive the gosocket.SocketId event, then call OnConnection
func (s *Sponsor) socketId(c ConnFace, id string) {
	c.SetId(id)
	c.Sponsor().CallGivenEvent(c, OnConnection)
}

// the server initiate a ping and the client reply a pong
func (s *Sponsor) ping(c ConnFace, arg int64) {
	c.Emit(EventPong, arg)
	return
}

// the server reply a pong, and the client initiate a ping
func (s *Sponsor) pong(c ConnFace, arg int64) {
	if _, ok := c.Ping()[arg]; ok {
		millisecond := time.Now().UnixNano() / int64(time.Millisecond)
		// to achieve a "continuous" effect, clear the container immediately after receiving any response
		c.SetPing(make(map[int64]bool))
		// update the value of delay
		c.SetDelay(millisecond - arg)
	}
	return
}
