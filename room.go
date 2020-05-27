package gosocket

type roomClient struct {
	room   string
	client *Client
}

type roomMessage struct {
	room  string
	event string
	args  interface{}
}

// rooms maintains the set of active clients and broadcasts messages to the clients
type rooms struct {
	clients   map[string]map[*Client]bool // all the clients in the rooms
	broadcast chan roomMessage            // this channel is responsible for broadcasting messages to designated rooms
	join      chan roomClient             // client request to join a room
	leave     chan roomClient             // client request to leave a room
	leaveAll  chan *Client                // client request to leave all of the rooms
}

func newRooms() *rooms {
	return &rooms{
		clients:   make(map[string]map[*Client]bool),
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
			if _, ok := r.clients[rc.room]; !ok {
				// init a new room
				r.clients[rc.room] = make(map[*Client]bool)
			}
			r.clients[rc.room][rc.client] = true
			//rc.client.rooms[rc.room] = true
			rc.client.rooms.Store(rc.room, true)
		// client request to leave a room
		case rc := <-r.leave:
			// do not close the message send channel(rc.client.out) here,may be other data to be transferred
			if _, ok := r.clients[rc.room]; ok {
				delete(r.clients[rc.room], rc.client)
			}
			if _, ok := rc.client.rooms.Load(rc.room); ok {
				//delete(rc.client.rooms, rc.room)
				rc.client.rooms.Delete(rc.room)
			}
		// client request to leave all of the rooms
		case client := <-r.leaveAll:
			r.Remove(client)
		// broadcasting messages to designated rooms
		case rm := <-r.broadcast:
			if _, ok := r.clients[rm.room]; ok {
				for client := range r.clients[rm.room] {
					client.Emit(rm.event, rm.args)
				}
			}
		}
	}
}

// remove the client from all the rooms, and close the message send channel
func (r *rooms) Remove(c *Client) {
	close(c.out)
	c.rooms.Range(func(k, v interface{}) bool {
		room := k.(string)
		c.rooms.Delete(room)
		if _, ok := r.clients[room][c]; ok {
			delete(r.clients[room], c)
		}
		return true
	})
}

// broadcast message to room
func (r *rooms) BroadcastTo(room, event string, args interface{}) {
	r.broadcast <- roomMessage{room, event, args}
}
