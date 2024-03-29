package websocket

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/plhwin/gosocket/conf"

	"github.com/plhwin/gosocket"

	"github.com/gorilla/websocket"
)

type ClientFace interface {
	gosocket.ClientFace
	init(*websocket.Conn, *gosocket.Acceptor, *http.Request) // init the client
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

func (c *Client) init(conn *websocket.Conn, a *gosocket.Acceptor, r *http.Request) {
	c.conn = conn
	// Set remoteAddr: Consider proxy
	// Use custom header name and controlled by the developers to avoid fake IP
	// Only set the name when the proxy is turned on
	// The header value should be contains two parts, the format is ip:port
	remoteAddr := conn.RemoteAddr()
	if conf.Acceptor.Websocket.RemoteAddrHeaderName != "" {
		if remoteAddrStr := r.Header.Get(conf.Acceptor.Websocket.RemoteAddrHeaderName); remoteAddrStr != "" {
			if ss := strings.Split(remoteAddrStr, ":"); len(ss) == 2 {
				if port, err := strconv.Atoi(ss[1]); err == nil {
					remoteAddr = &net.TCPAddr{IP: net.ParseIP(ss[0]), Port: port}
				}
			}
		}
	}
	c.SetRemoteAddr(remoteAddr)
	c.Init(a)
}

func (c *Client) Close() {
	c.conn.Close()
}

// handles websocket requests from the peer
func Serve(a *gosocket.Acceptor, w http.ResponseWriter, r *http.Request, c ClientFace) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WebSocket][client][Serve] upgrade error:", err)
		return
	}

	c.init(conn, a, r)

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
		c.Close()
		// Give a signal to the sender(Emit)
		// Here is the consumer program of the channel c.Out()
		// Can not close c.Out() here
		// c.Out() channel must be close by it's sender
		close(c.StopOut())
	}()

	messageType := websocket.TextMessage
	if conf.Acceptor.Websocket.MessageType == conf.WebsocketMessageTypeBinary {
		messageType = websocket.BinaryMessage
	}

	for {
		select {
		case msg, ok := <-c.Out():
			if !ok {
				log.Println("[WebSocket][client][write] msg send channel has been closed:", msg, c.Id(), c.RemoteAddr())
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(messageType, msg); err != nil {
				return
			}
		case <-ticker.C:
			// when the Websocket server sends `ping` messages for x consecutive times
			// but does not receive any` pong` messages back,
			// the server will actively disconnect from this client
			pings := c.Ping()
			if len(pings) >= conf.Acceptor.Heartbeat.PingMaxTimes {
				// close the connection
				log.Println("[WebSocket][client][write] miss pong reply:", c.Id(), c.RemoteAddr(), len(pings))
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			timeNow := time.Now()
			millisecond := timeNow.UnixNano() / int64(time.Millisecond)
			if msg, err := c.Acceptor().Encode(gosocket.EventPing, millisecond, "", conf.Acceptor.Transport.Send.Serialize, conf.Acceptor.Transport.Send.Compress); err == nil {
				if err := c.conn.WriteMessage(messageType, msg); err != nil {
					return
				}
				c.SetPing(millisecond, true)
			}
			if conf.Acceptor.Logs.Heartbeat.PingSend && c.Delay() >= conf.Acceptor.Logs.Heartbeat.PingSendPrintDelay {
				log.Println("[heartbeat][WebSocket][ping]:", c.Id(), c.RemoteAddr(), millisecond, timeNow.Format("2006-01-02 15:04:05.999"), len(pings), c.Delay())
			}
		}
	}
}

func (c *Client) read(face ClientFace) {
	defer func() {
		c.Close()
		c.LeaveAll()
		c.Acceptor().CallGivenEvent(face, gosocket.OnDisconnection)
	}()
	// Tolerate one heartbeat cycle
	wait := time.Duration((conf.Acceptor.Heartbeat.PingMaxTimes+2)*conf.Acceptor.Heartbeat.PingInterval) * time.Second
	for {
		if wait > 0 {
			c.conn.SetReadDeadline(time.Now().Add(wait))
		}
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("[WebSocket][client][read] go away:", err, c.Id(), c.RemoteAddr())
			break
			// error reading the message, break out of the loop,
			// the function of defer will executes the instruction to disconnect the client
		}
		c.process(face, msg)
	}
}

func (c *Client) process(face ClientFace, msg []byte) {
	// parse the message to determine what the client connection wants to do
	message, err := c.Acceptor().Decode(msg, conf.Acceptor.Transport.Receive.Serialize, conf.Acceptor.Transport.Receive.Compress)
	if err != nil {
		log.Println("[WebSocket][client][read] msg decode error:", err, msg, string(msg), c.Id(), c.RemoteAddr())
		return
	}
	c.Acceptor().CallEvent(face, message)
}
