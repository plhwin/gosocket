package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/plhwin/gosocket"

	"github.com/plhwin/gosocket/protocol"

	"github.com/gorilla/websocket"
)

type ClientFace interface {
	gosocket.ClientFace
	init(*websocket.Conn, *gosocket.Acceptor) // init the client
	read(ClientFace)
	write()
}

type Client struct {
	gosocket.Client
	conn *websocket.Conn // websocket conn
}

var upgrader = websocket.Upgrader{
	//ReadBufferSize:  4096,
	//WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *Client) init(conn *websocket.Conn, a *gosocket.Acceptor) {
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

// handles websocket requests from the peer
func Serve(a *gosocket.Acceptor, w http.ResponseWriter, r *http.Request, c ClientFace) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[upgrade][ws] error: ", err)
		return
	}

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
			//c.Conn().SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// the hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				return
			}
		case <-ticker.C:
			//c.Conn().SetWriteDeadline(time.Now().Add(writeWait))
			//if err := c.Conn().WriteMessage(websocket.PingMessage, nil); err != nil {
			//	return
			//}

			// when the Websocket server sends `ping` messages for x consecutive times
			// but does not receive any` pong` messages back,
			// the server will actively disconnect from this client
			if len(c.Ping()) >= 5 {
				// close the connection
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			timeNow := time.Now()
			millisecond := timeNow.UnixNano() / int64(time.Millisecond)
			if msg, err := protocol.Encode(gosocket.EventPing, millisecond); err == nil {
				if err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					return
				}
				c.Ping()[millisecond] = true
			}
			log.Println("[heartbeat][ws][ping]:", c.Id(), c.RemoteAddr(), millisecond, timeNow.Format("2006-01-02 15:04:05.999"), c.Delay())
		}
	}
}

func (c *Client) read(face ClientFace) {
	defer func() {
		c.Close(face)
	}()
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("[client][ws] go away:", err, c.Id(), c.RemoteAddr())
			break
			// error reading the message, break out of the loop,
			// the function of defer will executes the instruction to disconnect the client
		}
		c.process(face, string(msg))
	}
}

func (c *Client) process(face ClientFace, msg string) {
	// parse the message to determine what the client connection wants to do
	message, err := protocol.Decode(msg)
	if err != nil {
		log.Println("[client][ws] msg parse error:", err, msg)
		return
	}
	c.Acceptor().CallEvent(face, message)
}
