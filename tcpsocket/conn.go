package tcpsocket

import (
	"io"
	"log"
	"net"

	"github.com/plhwin/gosocket/conf"

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

	buf := make([]byte, 0, 4096) // 临时缓冲区的buffer
	readBuf := make([]byte, 256) // 每次读取多大buffer

	for {
		n, err := c.conn.Read(readBuf)
		if err != nil {
			if err == io.EOF {
				log.Println("[TCPSocket][conn][read] connection was closed:", err, c.Id(), c.RemoteAddr())
			} else {
				log.Println("[TCPSocket][conn][read] connection read error:", err, c.Id(), c.RemoteAddr())
			}
			break
		}
		buf = append(buf, readBuf[:n]...)

		// TCP拆包：一次读取可能读到到一条或者多条符合传输协议的内容，通常readBuf设置的比协议消息体大
		// 也可能什么内容也读取不到，要等待下一次读取与本次未处理的内容拼接再试，通常readBuf设置的比协议消息体小
		var data [][]byte
		if data, err = protocol.DePack(&buf); err != nil {
			log.Println("[TCPSocket][conn][read] protocol DePack error:", err, c.Id(), c.RemoteAddr())
			continue
		}
		for _, row := range data {
			message, decodeErr := protocol.Decode(row, conf.Initiator.Transport.Receive.Serialize, conf.Initiator.Transport.Receive.Compress)
			if decodeErr != nil {
				log.Println("[TCPSocket][conn][read] protocol Decode error:", decodeErr, row, string(row), c.Id(), c.RemoteAddr())
				continue
			}
			// bind function handler
			c.Initiator().CallEvent(face, message)
		}
	}
}

func (c *Conn) write() {
	defer c.conn.Close()
	for msg := range c.Out() {
		pkg, err := protocol.EnPack(msg)
		if err != nil {
			log.Println("[TCPSocket][conn][write] protocol EnPack error:", err, string(msg), c.Id(), c.RemoteAddr())
			break
		}
		if _, err := c.conn.Write(pkg.Bytes()); err != nil {
			log.Println("[TCPSocket][conn][write] error:", err, msg, string(msg), c.Id(), c.RemoteAddr())
			break
		}
	}
}
