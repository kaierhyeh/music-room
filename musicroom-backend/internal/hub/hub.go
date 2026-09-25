package hub

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	sendBuffer = 16
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]map[*Client]struct{}
}

func New() *Hub {
	return &Hub{rooms: make(map[uuid.UUID]map[*Client]struct{})}
}

type Client struct {
	hub  *Hub
	room uuid.UUID
	conn *websocket.Conn
	send chan []byte
}

func (h *Hub) Add(room uuid.UUID, conn *websocket.Conn) {
	c := &Client{hub: h, room: room, conn: conn, send: make(chan []byte, sendBuffer)}
	h.mu.Lock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]struct{})
	}
	h.rooms[room][c] = struct{}{}
	h.mu.Unlock()

	go c.writePump()
	go c.readPump()
}

func (h *Hub) remove(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set, ok := h.rooms[c.room]
	if !ok {
		return
	}
	if _, ok := set[c]; ok {
		delete(set, c)
		close(c.send)
		if len(set) == 0 {
			delete(h.rooms, c.room)
		}
	}
}

func (h *Hub) Broadcast(room uuid.UUID, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[room] {
		select {
		case c.send <- msg:
		default:
		}
	}
}

func (h *Hub) CloseRoom(room uuid.UUID) {
	h.mu.Lock()
	set := h.rooms[room]
	delete(h.rooms, room)
	h.mu.Unlock()
	for c := range set {
		close(c.send)
	}
}

func (h *Hub) RoomSize(room uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room])
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.remove(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}
