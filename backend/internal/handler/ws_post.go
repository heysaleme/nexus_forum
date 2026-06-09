package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type postWSClient struct {
	postID uint
	userID uint
	conn   *websocket.Conn
	send   chan []byte
	hub    *WSHub
}

func (h *WSHub) BroadcastToPost(postID uint, payload []byte) {
	h.mu.RLock()
	clients := h.postRooms[postID]
	h.mu.RUnlock()
	for c := range clients {
		select {
		case c.send <- payload:
		default:
		}
	}
}

func (h *WSHub) runPostJoin(client *postWSClient) {
	h.mu.Lock()
	if h.postRooms == nil {
		h.postRooms = make(map[uint]map[*postWSClient]bool)
	}
	if h.postRooms[client.postID] == nil {
		h.postRooms[client.postID] = make(map[*postWSClient]bool)
	}
	h.postRooms[client.postID][client] = true
	h.mu.Unlock()
}

func (h *WSHub) runPostLeave(client *postWSClient) {
	h.mu.Lock()
	if room := h.postRooms[client.postID]; room != nil {
		delete(room, client)
		if len(room) == 0 {
			delete(h.postRooms, client.postID)
		}
	}
	h.mu.Unlock()
	close(client.send)
}

func ServePostWS(hub *WSHub, authService service.AuthService) gin.HandlerFunc {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	return func(c *gin.Context) {
		postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
			return
		}
		token := c.Query("token")
		if token == "" {
			token = c.GetHeader("Sec-WebSocket-Protocol")
		}
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := &postWSClient{
			postID: uint(postID),
			userID: claims.UserID,
			conn:   conn,
			send:   make(chan []byte, 64),
			hub:    hub,
		}
		hub.runPostJoin(client)
		go client.postWritePump()
		client.postReadPump()
	}
}

func (c *postWSClient) postReadPump() {
	defer func() {
		c.hub.runPostLeave(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Debug("post ws closed", "post_id", c.postID, "error", err)
			}
			return
		}
	}
}

func (c *postWSClient) postWritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
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

func (h *Handlers) broadcastPostVote(postID uint, score int) {
	payload, _ := json.Marshal(map[string]interface{}{
		"type": "vote",
		"data": map[string]interface{}{"post_id": postID, "score": score},
	})
	h.WSHub.BroadcastToPost(postID, payload)
}

func (h *Handlers) broadcastPostComment(postID uint, comment *model.Comment) {
	payload, _ := json.Marshal(map[string]interface{}{
		"type": "comment",
		"data": comment,
	})
	h.WSHub.BroadcastToPost(postID, payload)
}
