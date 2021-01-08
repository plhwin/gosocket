package tcpsocket

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"time"

	"github.com/plhwin/gosocket/conf"

	"github.com/plhwin/gosocket"

	"github.com/plhwin/gosocket/protocol"
)

const (
	msgEnd byte = '\n'
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
	c.LeaveAll()
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
	ticker := time.NewTicker(time.Duration(conf.Acceptor.Heartbeat.PingInterval) * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		// Give a signal to the sender(Emit)
		// Here is the consumer program of the channel c.Out()
		// Can not close c.Out() here
		// c.Out() channel must be close by it's sender
		close(c.StopOut())
	}()

	for {
		select {
		case msg, ok := <-c.Out():
			if !ok {
				log.Println("[TCPSocket][client][write] msg send channel has been closed:", string(msg), c.Id(), c.RemoteAddr())
				return
			}

			// for test start: simulate sending delay
			//if strings.HasPrefix(c.RemoteAddr().String(), "127.0.0.1") {
			//	time.Sleep(time.Second * 5)
			//}
			// for test end

			if n, err := c.conn.Write(append(msg, msgEnd)); err != nil {
				log.Println("[TCPSocket][client][write] error:", err, n, string(msg), c.Id(), c.RemoteAddr())
				return
			}
		case <-ticker.C:
			// when the socket server sends `ping` messages for x consecutive times
			// but does not receive any` pong` messages back,
			// the server will actively disconnect from this client
			pings := c.Ping()
			if len(pings) >= conf.Acceptor.Heartbeat.PingMaxTimes {
				// close the connection
				log.Println("[TCPSocket][client][write] miss pong reply:", c.Id(), c.RemoteAddr(), len(pings))
				return
			}
			timeNow := time.Now()
			millisecond := timeNow.UnixNano() / int64(time.Millisecond)
			if msg, err := protocol.Encode(gosocket.EventPing, millisecond, ""); err == nil {
				if _, err := c.conn.Write(append(msg, msgEnd)); err != nil {
					return
				}
				c.SetPing(millisecond, true)
			}
			if conf.Acceptor.Logs.Heartbeat.PingSend && c.Delay() >= conf.Acceptor.Logs.Heartbeat.PingSendPrintDelay {
				log.Println("[heartbeat][TCPSocket][ping]:", c.Id(), c.RemoteAddr(), millisecond, timeNow.Format("2006-01-02 15:04:05.999"), len(pings), c.Delay())
			}
		}
	}
}

func (c *Client) read(face ClientFace) {
	defer func() {
		c.Close(face)
	}()
	// tcp sticky packet: use bufio NewReader, specific characters \n separated
	reader := bufio.NewReader(c.conn)

	// Tolerate one heartbeat cycle
	wait := time.Duration((conf.Acceptor.Heartbeat.PingMaxTimes+2)*conf.Acceptor.Heartbeat.PingInterval) * time.Second
	for {
		if wait > 0 {
			c.conn.SetReadDeadline(time.Now().Add(wait))
		}
		msg, err := reader.ReadBytes(msgEnd)
		if err != nil {
			log.Println("[TCPSocket][client][read] go away:", err, c.Id(), c.RemoteAddr())
			break
		}
		// parse the message to determine what the client connection wants to do
		// remove msgEnd
		msg = bytes.TrimSuffix(msg, []byte{msgEnd})
		message, err := protocol.Decode(msg)
		if err != nil {
			log.Println("[TCPSocket][client][read] msg decode error:", err, msg, c.Id(), c.RemoteAddr())
			continue
		}

		// bind function handler
		c.Acceptor().CallEvent(face, message)
	}
}
