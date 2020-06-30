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
	init(net.Conn, *gosocket.Initiator) // init the conn
	read(ConnFace)
	write()
}

type Conn struct {
	gosocket.Conn
	conn net.Conn // tcp socket conn
}

func (c *Conn) init(conn net.Conn, i *gosocket.Initiator) {
	c.conn = conn
	c.SetRemoteAddr(conn.RemoteAddr())
	c.Init(i)
}

func (c *Conn) Close(face ConnFace) {
	c.conn.Close()
	c.Initiator().CallGivenEvent(face, gosocket.OnDisconnection)
}

// as a initiator, receive message from tcp socket server
func Receive(i *gosocket.Initiator, conn net.Conn, c ConnFace) {
	c.init(conn, i)
	// After receive the SocketId event, then call OnConnection, see sponsor.go
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
			log.Println("[TCPSocket][conn][read] error:", err, msg)
			break
		}
		// parse the message to determine what the client connection wants to do
		msg = strings.Replace(msg, string([]byte{end}), "", -1)
		message, err := protocol.Decode(msg)
		if err != nil {
			log.Println("[TCPSocket][conn][read] msg decode error:", err, msg)
			continue
		}
		// bind function handler
		c.Initiator().CallEvent(face, message)
	}
}

func (c *Conn) write() {
	defer c.conn.Close()
	for msg := range c.Out() {
		if _, err := c.conn.Write([]byte(msg + "\n")); err != nil {
			log.Println("[TCPSocket][conn][write] error:", err, msg)
			break
		}
	}
}
