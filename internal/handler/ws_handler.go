package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"stock/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub *ws.Hub
}

func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, map[string]interface{}{"code": 401, "message": "token required"})
		return
	}

	ws.ServeWS(h.hub, c.Writer, c.Request)
}

type WSMessage struct {
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

func (h *WSHandler) broadcastSnapshot(data interface{}) {
	msg := WSMessage{
		Type:      "pool_snapshot",
		Timestamp: time.Now().Unix(),
	}
	body, _ := json.Marshal(data)
	msg.Data = body

	msgBytes, _ := json.Marshal(msg)
	h.hub.Broadcast(msgBytes)
}

func (h *WSHandler) broadcastDiff(data interface{}) {
	msg := WSMessage{
		Type:      "pool_diff",
		Timestamp: time.Now().Unix(),
	}
	body, _ := json.Marshal(data)
	msg.Data = body

	msgBytes, _ := json.Marshal(msg)
	h.hub.Broadcast(msgBytes)
}

func (h *WSHandler) broadcastAlert(data interface{}) {
	msg := WSMessage{
		Type:      "strategy_alert",
		Timestamp: time.Now().Unix(),
	}
	body, _ := json.Marshal(data)
	msg.Data = body

	msgBytes, _ := json.Marshal(msg)
	h.hub.Broadcast(msgBytes)
}
