package gosocket

import (
	"time"
)

// the client initiate a ping and the server reply a pong
func ping(c ClientFace, arg int64) {
	// reply pong
	c.Emit("pong", arg)
	return
}

// the client reply a pong, and the server initiate a ping
func pong(c ClientFace, arg int64) {
	if _, ok := c.Ping()[arg]; ok {
		millisecond := time.Now().UnixNano() / int64(time.Millisecond)
		// to achieve a "continuous" effect, clear the container immediately after receiving any response
		c.SetPing(make(map[int64]bool))
		// update the value of delay
		c.SetDelay(millisecond - arg)
	}
	return
}
