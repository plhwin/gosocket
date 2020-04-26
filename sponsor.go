package gosocket

func NewSponsor() (s *Sponsor) {
	s = new(Sponsor)
	s.initEvents()

	return
}

type Sponsor struct {
	events
}
