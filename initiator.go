package gosocket

import "time"

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
}

func (i *Initiator) SetConn(c ConnFace) {
	i.conn = c
	i.alive = true
}

func (i *Initiator) onDisConn(interface{}) {
	i.alive = false
}

func (i *Initiator) Alive() bool {
	return i.alive
}

func (i *Initiator) Emit(event string, args interface{}) {
	if i.alive {
		i.conn.Emit(event, args)
	}
}

// After receive the SocketId event, call OnConnection
func (i *Initiator) socketId(c ConnFace, id string) {
	c.SetId(id)
	c.Initiator().CallGivenEvent(c, OnConnection)
}

// the server initiate a ping and the client reply a pong
func (i *Initiator) ping(c ConnFace, arg int64) {
	c.Emit(EventPong, arg)
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
	return
}
