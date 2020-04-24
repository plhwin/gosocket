package gosocket

import (
	"log"
	"net/http"
	"time"

	"github.com/plhwin/gosocket/protocol"

	"github.com/gorilla/websocket"
)

type WebsocketClientFace interface {
	ClientFace
	InitWebsocket(*websocket.Conn, *Server) // init the client
	Conn() *websocket.Conn                  // get *websocket.Conn
}

type WebsocketClient struct {
	Client
	conn *websocket.Conn // websocket conn
}

var upgrader = websocket.Upgrader{
	//ReadBufferSize:  4096,
	//WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *WebsocketClient) InitWebsocket(conn *websocket.Conn, s *Server) {
	c.conn = conn
	c.remoteAddr = c.conn.RemoteAddr()
	c.Init(s)
}

func (c *WebsocketClient) Conn() *websocket.Conn {
	return c.conn
}

func (c *WebsocketClient) Close() {
	c.LeaveAll()
	c.conn.Close()
}

// handles websocket requests from the peer.
func ServeWs(s *Server, w http.ResponseWriter, r *http.Request, c WebsocketClientFace) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade websocket: ", err)
		return
	}

	c.InitWebsocket(conn, s)

	log.Println("new connection incoming:", c.Id(), c.RemoteAddr())

	// write message to client
	go writeWs(c)

	// read message from client
	go readWs(c)
}

func writeWs(c WebsocketClientFace) {
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
			//c.Conn().SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// the hub closed the channel.
				c.Conn().WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn().WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
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
				c.Conn().WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			timeNow := time.Now()
			millisecond := timeNow.UnixNano() / int64(time.Millisecond)
			if msg, err := protocol.Encode("ping", millisecond); err == nil {
				if err := c.Conn().WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					return
				}
				c.Ping()[millisecond] = true
			}
			log.Println("heartbeat:", c.Id(), c.RemoteAddr(), millisecond, timeNow.Format("2006-01-02 15:04:05.999"), c.Delay())
		}
	}
}

func readWs(c WebsocketClientFace) {
	defer func() {
		c.Close()
	}()
	for {
		_, msg, err := c.Conn().ReadMessage()
		if err != nil {
			log.Println("client go away:", err, c.Id(), c.RemoteAddr())
			break
			// error reading the message, break out of the loop,
			// the function of defer will executes the instruction to disconnect the client
		}
		processWs(c, string(msg))
	}
}

func processWs(c WebsocketClientFace, msg string) {
	// parse the message to determine what the client connection wants to do
	message, err := protocol.Decode(msg)
	if err != nil {
		log.Println("msg parse error:", err, msg)
		return
	}
	c.Server().CallEvent(c, message)
}
