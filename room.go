package gosocket

import "sync"

type roomClient struct {
	room   string
	client *Client
}

type roomMessage struct {
	room  string
	event string
	args  interface{}
	id    string
}

// rooms maintains the set of active clients and broadcasts messages to the clients
type rooms struct {
	clients   *sync.Map        // map[string]map[*Client]bool all the clients in the rooms
	broadcast chan roomMessage // this channel is responsible for broadcasting messages to designated rooms
	join      chan roomClient  // client request to join a room
	leave     chan roomClient  // client request to leave a room
	leaveAll  chan *Client     // client request to leave all of the rooms
}

func newRooms() *rooms {
	return &rooms{
		clients:   new(sync.Map),
		broadcast: make(chan roomMessage),
		join:      make(chan roomClient),
		leave:     make(chan roomClient),
		leaveAll:  make(chan *Client),
	}
}

func (r *rooms) Run() {
	for {
		select {
		// client request to join a room
		case rc := <-r.join:
			var clients map[*Client]bool
			if v, ok := r.clients.Load(rc.room); ok {
				clients = v.(map[*Client]bool)
			} else {
				// init a new room
				clients = make(map[*Client]bool)
			}
			clients[rc.client] = true
			r.clients.Store(rc.room, clients)
			rc.client.rooms.Store(rc.room, true)
		// client request to leave a room
		case rc := <-r.leave:
			// do not close the message send channel(rc.client.out) here,may be other data to be transferred
			if v, ok := r.clients.Load(rc.room); ok {
				clients := v.(map[*Client]bool)
				delete(clients, rc.client)
				r.clients.Store(rc.room, clients)
			}
			if _, ok := rc.client.rooms.Load(rc.room); ok {
				rc.client.rooms.Delete(rc.room)
			}
		// client request to leave all of the rooms
		case client := <-r.leaveAll:
			r.Remove(client)
		// broadcasting messages to designated rooms
		case rm := <-r.broadcast:
			if v, ok := r.clients.Load(rm.room); ok {
				clients := v.(map[*Client]bool)
				for client := range clients {
					client.Emit(rm.event, rm.args, rm.id)
				}
			}
		}
	}
}

// remove the client from all the rooms, and close the message send channel
func (r *rooms) Remove(c *Client) {
	c.rooms.Range(func(k, v interface{}) bool {
		room := k.(string)
		c.rooms.Delete(room)
		// delete client from room
		if v, ok := r.clients.Load(room); ok {
			clients := v.(map[*Client]bool)
			delete(clients, c)
			r.clients.Store(room, clients)
		}
		return true
	})
}

// broadcast message to room
func (r *rooms) BroadcastTo(room, event string, args interface{}, id string) {
	r.broadcast <- roomMessage{room, event, args, id}
}
