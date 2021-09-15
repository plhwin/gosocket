package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/plhwin/gosocket/conf"

	"github.com/plhwin/gosocket"
	"github.com/plhwin/gosocket/protocol"
)

type ConnFace interface {
	gosocket.ConnFace
	init(*websocket.Conn, *gosocket.Initiator) // init to conn
	read(ConnFace)
	write()
}

type Conn struct {
	gosocket.Conn
	conn *websocket.Conn // websocket conn
}

func (c *Conn) init(conn *websocket.Conn, i *gosocket.Initiator) {
	c.conn = conn
	c.SetRemoteAddr(conn.RemoteAddr())
	c.Init(i)
}

func (c *Conn) Close(face ConnFace) {
	c.conn.Close()
	c.Initiator().CallGivenEvent(face, gosocket.OnDisconnection)
}

func Dial(urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	return websocket.DefaultDialer.Dial(urlStr, requestHeader)
}

// Receive as an initiator, receive message from websocket server
// Receive is a non-blocking method.
// The blocking process is managed by the outer layer.
// For example, the outer layer can freely control the automatic reconnection after the connection is disconnected.
// Therefore, the outer method cannot close the websocket connection.
// The closing of the connection will be handled here
func Receive(i *gosocket.Initiator, conn *websocket.Conn, c ConnFace) {
	c.init(conn, i)
	// After receive the SocketId event, then call OnConnection, see sponsor.go
	go c.write()
	go c.read(c)
}

func (c *Conn) read(face ConnFace) {
	defer c.Close(face)
	for {
		messageType, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("[WebSocket][conn][read] connection read error:", err, c.conn.LocalAddr(), "|", messageType, "|", msg, "|", string(msg), "|", c.Id(), c.RemoteAddr())
			break
		}
		message, decodeErr := protocol.Decode(msg, conf.Initiator.Transport.Receive.Serialize, conf.Initiator.Transport.Receive.Compress)
		if decodeErr != nil {
			log.Println("[WebSocket][conn][read] protocol Decode error:", decodeErr, msg, string(msg), c.Id(), c.RemoteAddr())
			continue
		}
		// bind function handler
		c.Initiator().CallEvent(face, message)
	}
}

func (c *Conn) write() {
	defer c.conn.Close()
	messageType := websocket.TextMessage
	if conf.Initiator.Websocket.MessageType == conf.WebsocketMessageTypeBinary {
		messageType = websocket.BinaryMessage
	}
	for msg := range c.Out() {
		if err := c.conn.WriteMessage(messageType, msg); err != nil {
			log.Println("[WebSocket][conn][write] error:", err, msg, string(msg), c.Id(), c.RemoteAddr())
			break
		}
	}
}
