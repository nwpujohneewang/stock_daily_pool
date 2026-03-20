package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"stock/config"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
	logger     *log.Logger
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	ID   string
}

type WSMessage struct {
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     log.Default(),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()
		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.mutex.RUnlock()
					h.mutex.Lock()
					close(client.send)
					delete(h.clients, client)
					h.mutex.Unlock()
					h.mutex.RLock()
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) BroadcastSnapshot(data []byte) {
	msg := WSMessage{
		Type:      "pool_snapshot",
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	body, _ := json.Marshal(msg)
	h.broadcast <- body
}

func (h *Hub) BroadcastDiff(data []byte) {
	msg := WSMessage{
		Type:      "pool_diff",
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	body, _ := json.Marshal(msg)
	h.broadcast <- body
}

func (h *Hub) BroadcastAlert(data []byte) {
	msg := WSMessage{
		Type:      "strategy_alert",
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	body, _ := json.Marshal(msg)
	h.broadcast <- body
}

func (h *Hub) ClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

func NewClient(hub *Hub, conn *websocket.Conn, bufSize int) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, bufSize),
		ID:   generateClientID(),
	}
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		ID:   generateClientID(),
	}

	hub.Register(client)

	go client.WritePump(config.WebSocketConfig{
		PingInterval:    50,
		PongTimeout:     60,
		WriteBufferSize: 256,
	})
	go client.ReadPump()
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) WritePump(cfg config.WebSocketConfig) {
	ticker := time.NewTicker(time.Duration(cfg.PingInterval) * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(time.Duration(cfg.WriteBufferSize) * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(time.Duration(cfg.PingInterval) * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func generateClientID() string {
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}
