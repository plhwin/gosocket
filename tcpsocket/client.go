package tcpsocket

import (
	"bufio"
	"log"
	"net"
	"strings"
	"time"

	"github.com/plhwin/gosocket"

	"github.com/plhwin/gosocket/protocol"
)

type ClientFace interface {
	gosocket.ClientFace
	init(net.Conn, *gosocket.Acceptor) // init the client
	read(ClientFace)
	write()
}

type Client struct {
	gosocket.Client
	conn net.Conn // tcp socket conn
}

func (c *Client) init(conn net.Conn, a *gosocket.Acceptor) {
	c.conn = conn
	c.SetRemoteAddr(conn.RemoteAddr())
	c.Init(a)
}

func (c *Client) Close(face ClientFace) {
	c.conn.Close()
	c.LeaveAllRooms()
	c.Acceptor().Leave(c)
	c.Acceptor().CallGivenEvent(face, gosocket.OnDisconnection)
}

// handles socket requests from the peer
func Serve(conn net.Conn, a *gosocket.Acceptor, c ClientFace) {
	// init tcp socket
	c.init(conn, a)

	// add the ClientFace to acceptor
	a.Join(c)

	// trigger the event: OnConnection
	a.CallGivenEvent(c, gosocket.OnConnection)

	// write message to client
	go c.write()

	// read message from client
	go c.read(c)
}

func (c *Client) write() {
	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		// the data transmission channel (c.Out) is not closed here,
		// it will be handled uniformly by read method
	}()
	for {
		select {
		case msg, ok := <-c.Out():
			if !ok {
				// the hub closed the channel.
				return
			}
			if _, err := c.conn.Write([]byte(msg + "\n")); err != nil {
				return
			}
		case <-ticker.C:
			// when the socket server sends `ping` messages for x consecutive times
			// but does not receive any` pong` messages back,
			// the server will actively disconnect from this client
			if len(c.Ping()) >= 5 {
				// close the connection
				return
			}
			timeNow := time.Now()
			millisecond := timeNow.UnixNano() / int64(time.Millisecond)
			if msg, err := protocol.Encode(gosocket.EventPing, millisecond, ""); err == nil {
				if _, err := c.conn.Write([]byte(msg + "\n")); err != nil {
					return
				}
				c.Ping()[millisecond] = true
			}
			log.Println("[heartbeat][ts][ping]:", c.Id(), c.RemoteAddr(), millisecond, timeNow.Format("2006-01-02 15:04:05.999"), c.Delay())
		}
	}
}

func (c *Client) read(face ClientFace) {
	defer func() {
		c.Close(face)
	}()
	// tcp sticky packet: use bufio NewReader, specific characters \n separated
	reader := bufio.NewReader(c.conn)
	var end byte = '\n'
	for {
		msg, err := reader.ReadString(end)
		if err != nil {
			log.Println("[client][ts] go away:", err, c.Id(), c.RemoteAddr())
			break
		}
		// parse the message to determine what the client connection wants to do
		msg = strings.Replace(msg, string([]byte{end}), "", -1)
		message, err := protocol.Decode(msg)
		if err != nil {
			log.Println("[client][ts] msg parse error:", err, c.Id(), c.RemoteAddr(), msg)
			continue
		}

		// bind function handler
		c.Acceptor().CallEvent(face, message)
	}
}
