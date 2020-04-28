package tcpsocket

import (
	"bufio"
	"log"
	"net"
	"strings"

	"github.com/plhwin/gosocket"
	"github.com/plhwin/gosocket/protocol"
)

type ConnFace interface {
	gosocket.ConnFace
	init(net.Conn, *gosocket.Sponsor) // init the conn
	read(ConnFace)
	write()
}

type Conn struct {
	gosocket.Conn
	conn net.Conn // tcp socket conn
}

func (c *Conn) init(conn net.Conn, s *gosocket.Sponsor) {
	c.conn = conn
	c.SetRemoteAddr(conn.RemoteAddr())
	c.Init(s)
}

func (c *Conn) Close(face ConnFace) {
	c.conn.Close()
	c.Sponsor().CallGivenEvent(face, gosocket.OnDisconnection)
}

// as a sponsor, receive message from tcp socket server
func Receive(s *gosocket.Sponsor, conn net.Conn, c ConnFace) {
	c.init(conn, s)
	// After receive the gosocket.SocketId event, then call OnConnection, see sponsor.go
	go c.write()
	go c.read(c)
}

func (c *Conn) read(face ConnFace) {
	defer c.Close(face)
	reader := bufio.NewReader(c.conn)
	var end byte = '\n'
	for {
		msg, err := reader.ReadString(end)
		if err != nil {
			log.Println("can not read from tcp socket server, the connection will be close right now!", err, msg)
			break
		}
		// parse the message to determine what the client connection wants to do
		msg = strings.Replace(msg, string([]byte{end}), "", -1)
		message, err := protocol.Decode(msg)
		if err != nil {
			log.Println("msg read from tcp socket server decode error:", err, msg)
			continue
		}
		// bind function handler
		c.Sponsor().CallEvent(face, message)
	}
}

func (c *Conn) write() {
	defer c.conn.Close()
	for msg := range c.Out() {
		if _, err := c.conn.Write([]byte(msg + "\n")); err != nil {
			log.Println("can not write message to the tcp socket server, the connection will be close right now!", err, msg)
			break
		}
	}
}
