package tcpsocket

import (
	"log"
	"net"
	"strings"
	"time"

	"github.com/plhwin/gosocket"

	"github.com/plhwin/gosocket/protocol"
)

type ClientFace interface {
	gosocket.ClientFace
	init(net.Conn, *gosocket.Server) // init the client
	read(ClientFace)
	write()
}

type Client struct {
	gosocket.Client
	conn net.Conn // tcp socket conn
}

func (c *Client) init(conn net.Conn, s *gosocket.Server) {
	c.conn = conn
	c.SetRemoteAddr(conn.RemoteAddr())
	c.Init(s)
}

func (c *Client) Close() {
	c.LeaveServer()
	c.LeaveAllRooms()
	c.conn.Close()
}

// handles socket requests from the peer.
func Serve(listener net.Listener, s *gosocket.Server, c ClientFace) {
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("listener.Accept error:", err, c)
			continue
		}
		go handleClient(conn, s, c)
	}
}

func handleClient(conn net.Conn, s *gosocket.Server, c ClientFace) {
	// init tcp socket
	c.init(conn, s)

	log.Println("new connection incoming:", c.Id(), c.RemoteAddr())

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
			if msg, err := protocol.Encode("ping", millisecond); err == nil {
				if _, err := c.conn.Write([]byte(msg + "\n")); err != nil {
					return
				}
				c.Ping()[millisecond] = true
			}
			log.Println("heartbeat:", c.Id(), c.RemoteAddr(), millisecond, timeNow.Format("2006-01-02 15:04:05.999"), c.Delay())
		}
	}
}

func (c *Client) read(face ClientFace) {
	defer func() {
		c.Close()
	}()
	request := make([]byte, 1024) // set maximum request length to 128B to prevent flood attack
	for {
		readLen, err := c.conn.Read(request)
		if err != nil {
			log.Println("client go away:", err, c.Id(), c.RemoteAddr())
			break
			// error reading the message, break out of the loop,
			// the function of defer will executes the instruction to disconnect the client
		}

		if readLen == 0 {
			log.Println("connection already closed by client", readLen)
			break // connection already closed by client
		}

		msg := strings.TrimSpace(string(request[:readLen]))
		c.process(face, msg)
		request = make([]byte, 1024) // clear last read content
	}
}

func (c *Client) process(face ClientFace, msg string) {
	// parse the message to determine what the client connection wants to do
	message, err := protocol.Decode(msg)
	if err != nil {
		log.Println("msg parse error:", err, msg)
		return
	}
	c.Server().CallEvent(face, message)
}
