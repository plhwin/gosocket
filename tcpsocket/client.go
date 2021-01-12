package tcpsocket

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/plhwin/gosocket/conf"

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

			pkg, err := protocol.EnPack(msg)
			if err != nil {
				log.Println("[TCPSocket][client][write] protocol EnPack error:", err, string(msg), c.Id(), c.RemoteAddr())
				return
			}

			if n, err := c.conn.Write(pkg.Bytes()); err != nil {
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
			if msg, err := protocol.Encode(gosocket.EventPing, millisecond, "", conf.Acceptor.TransportProtocol.Send); err == nil {
				pkg, err := protocol.EnPack(msg)
				if err != nil {
					return
				}
				if _, err := c.conn.Write(pkg.Bytes()); err != nil {
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

	buf := make([]byte, 0, 4096) // 临时缓冲区的buffer
	readBuf := make([]byte, 256) // 每次读取多大buffer

	// Tolerate one heartbeat cycle
	wait := time.Duration((conf.Acceptor.Heartbeat.PingMaxTimes+2)*conf.Acceptor.Heartbeat.PingInterval) * time.Second
	for {
		if wait > 0 {
			c.conn.SetReadDeadline(time.Now().Add(wait))
		}

		n, err := c.conn.Read(readBuf)
		if err != nil {
			if err == io.EOF {
				log.Println("[TCPSocket][client][read] connection was closed:", err, c.Id(), c.RemoteAddr())
			} else {
				log.Println("[TCPSocket][client][read] connection read error:", err, c.Id(), c.RemoteAddr())
			}
			break
		}
		buf = append(buf, readBuf[:n]...)

		// TCP拆包：一次读取可能读到到一条或者多条符合传输协议的内容，通常readBuf设置的比协议消息体大
		// 也可能什么内容也读取不到，要等待下一次读取与本次未处理的内容拼接再试，通常readBuf设置的比协议消息体小
		var data [][]byte
		if data, err = protocol.DePack(&buf); err != nil {
			log.Println("[TCPSocket][client][read] protocol DePack error:", err, c.Id(), c.RemoteAddr())
			continue
		}
		for _, row := range data {
			message, decodeErr := protocol.Decode(row, conf.Acceptor.TransportProtocol.Receive)
			if decodeErr != nil {
				log.Println("[TCPSocket][client][read] protocol Decode error:", decodeErr, row, string(row), c.Id(), c.RemoteAddr())
				continue
			}
			// bind function handler
			c.Acceptor().CallEvent(face, message)
		}
	}
}
