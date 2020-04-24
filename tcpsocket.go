package gosocket

import (
	"log"
	"net"
	"strings"
	"time"

	"github.com/plhwin/gosocket/protocol"
)

type TCPClientFace interface {
	ClientFace
	InitTCPSocket(net.Conn, *Server) // init the client
	Conn() net.Conn                  // get *websocket.Conn
}

type TCPClient struct {
	Client
	conn net.Conn // tcp socket conn
}

func (c *TCPClient) InitTCPSocket(conn net.Conn, s *Server) {
	c.conn = conn
	c.remoteAddr = c.conn.RemoteAddr()
	c.Init(s)
}

func (c *TCPClient) Conn() net.Conn {
	return c.conn
}

func (c *TCPClient) Close() {
	c.LeaveAll()
	c.conn.Close()
}

// handles websocket requests from the peer.
func ServeTs(listener net.Listener, s *Server, c TCPClientFace) {
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

func handleClient(conn net.Conn, s *Server, c TCPClientFace) {
	// init tcp socket
	c.InitTCPSocket(conn, s)

	log.Println("new connection incoming:", c.Id(), c.RemoteAddr())

	// write message to client
	go writeTs(c)

	// read message from client
	go readTs(c)
}

func writeTs(c TCPClientFace) {
	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn().Close()
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
			if _, err := c.Conn().Write([]byte(msg + "\n")); err != nil {
				return
			}
		case <-ticker.C:
			// when the Websocket server sends `ping` messages for x consecutive times
			// but does not receive any` pong` messages back,
			// the server will actively disconnect from this client
			if len(c.Ping()) >= 5 {
				// close the connection
				return
			}
			timeNow := time.Now()
			millisecond := timeNow.UnixNano() / int64(time.Millisecond)
			if msg, err := protocol.Encode("ping", millisecond); err == nil {
				if _, err := c.Conn().Write([]byte(msg + "\n")); err != nil {
					return
				}
				c.Ping()[millisecond] = true
			}
			log.Println("heartbeat:", c.Id(), c.RemoteAddr(), millisecond, timeNow.Format("2006-01-02 15:04:05.999"), c.Delay())
		}
	}
}

func readTs(c TCPClientFace) {
	defer func() {
		c.Close()
	}()
	request := make([]byte, 1024) // set maximum request length to 128B to prevent flood attack
	for {
		readLen, err := c.Conn().Read(request)
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
		processTs(c, msg)
		request = make([]byte, 1024) // clear last read content
	}
}

func processTs(c TCPClientFace, msg string) {
	// parse the message to determine what the client connection wants to do
	message, err := protocol.Decode(msg)
	if err != nil {
		log.Println("msg parse error:", err, msg)
		return
	}
	c.Server().CallEvent(c, message)
}
