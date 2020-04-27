package gosocket

func NewSponsor(c ConnFace) (s *Sponsor) {
	s = new(Sponsor)
	s.initEvents()
	s.conn = c
	return
}

type Sponsor struct {
	events
	conn ConnFace
}

func (s *Sponsor) Emit(event string, args interface{}) {
	s.conn.Emit(event, args)
}
