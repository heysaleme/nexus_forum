package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/service"
)

// WSMessage is the envelope sent over WebSocket connections.
type WSMessage struct {
	Type           string          `json:"type"`     // "message", "ping", "pong", "error", "typing", "read", "online_status", "delivery_status"
	RoomID         uint            `json:"room_id"`
	SenderID       uint            `json:"sender_id"`
	SenderName     string          `json:"sender_name"`
	Content        string          `json:"content"`
	IsRead         bool            `json:"is_read"`
	IsDelivered    bool            `json:"is_delivered"`
	AttachmentURL  string          `json:"attachment_url,omitempty"`
	AttachmentType string          `json:"attachment_type,omitempty"`
	Timestamp      time.Time       `json:"timestamp"`
	Raw            json.RawMessage `json:"data,omitempty"`
}

// wsClient represents one live WebSocket connection.
type wsClient struct {
	roomID uint
	userID uint
	conn   *websocket.Conn
	send   chan []byte
	hub    *WSHub
}

// WSHub manages all active WebSocket connections organized by chat room.
type WSHub struct {
	mu          sync.RWMutex
	rooms       map[uint]map[*wsClient]bool // roomID -> set of clients
	onlineUsers map[uint]int                // userID -> connection count
	join        chan *wsClient
	leave       chan *wsClient
	broadcast   chan broadcastMsg
	db          *gorm.DB
}

type broadcastMsg struct {
	roomID  uint
	payload []byte
}

// NewWSHub creates and starts a new hub. Must be called once during startup.
func NewWSHub(db *gorm.DB) *WSHub {
	h := &WSHub{
		rooms:       make(map[uint]map[*wsClient]bool),
		onlineUsers: make(map[uint]int),
		join:        make(chan *wsClient, 64),
		leave:       make(chan *wsClient, 64),
		broadcast:   make(chan broadcastMsg, 256),
		db:          db,
	}
	go h.run()
	return h
}

// IsUserOnline returns true if the user has any active WebSocket connections.
func (h *WSHub) IsUserOnline(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.onlineUsers[userID] > 0
}

func (h *WSHub) run() {
	for {
		select {
		case client := <-h.join:
			h.mu.Lock()
			if h.rooms[client.roomID] == nil {
				h.rooms[client.roomID] = make(map[*wsClient]bool)
			}
			h.rooms[client.roomID][client] = true
			h.onlineUsers[client.userID]++
			isNewOnline := h.onlineUsers[client.userID] == 1
			h.mu.Unlock()

			slog.Info("ws client joined room", "room_id", client.roomID, "user_id", client.userID, "is_new_online", isNewOnline)

			if isNewOnline && h.db != nil {
				go func(uid uint) {
					h.db.Model(&model.User{}).Where("id = ?", uid).Update("last_seen_at", time.Now())
				}(client.userID)
			}

			// Broadcast online status to other room participants
			if client.roomID != 0 {
				onlineMsg, _ := json.Marshal(WSMessage{
					Type:     "online_status",
					RoomID:   client.roomID,
					SenderID: client.userID,
					Content:  "online",
				})
				h.BroadcastExcept(client.roomID, onlineMsg, client)

				// Send current online status of other participants in this room to the joining client
				h.mu.RLock()
				for otherClient := range h.rooms[client.roomID] {
					if otherClient.userID != client.userID {
						statusMsg, _ := json.Marshal(WSMessage{
							Type:     "online_status",
							RoomID:   client.roomID,
							SenderID: otherClient.userID,
							Content:  "online",
						})
						client.send <- statusMsg
					}
				}
				h.mu.RUnlock()
			}

		case client := <-h.leave:
			h.mu.Lock()
			if room, ok := h.rooms[client.roomID]; ok {
				delete(room, client)
				if len(room) == 0 {
					delete(h.rooms, client.roomID)
				}
			}
			h.onlineUsers[client.userID]--
			isNowOffline := h.onlineUsers[client.userID] <= 0
			if isNowOffline {
				h.onlineUsers[client.userID] = 0
			}
			h.mu.Unlock()

			slog.Info("ws client left room", "room_id", client.roomID, "user_id", client.userID, "is_now_offline", isNowOffline)

			if isNowOffline && h.db != nil {
				go func(uid uint) {
					h.db.Model(&model.User{}).Where("id = ?", uid).Update("last_seen_at", time.Now())
				}(client.userID)
			}

			// Broadcast offline status to other room participants
			if client.roomID != 0 {
				offlineMsg, _ := json.Marshal(WSMessage{
					Type:     "online_status",
					RoomID:   client.roomID,
					SenderID: client.userID,
					Content:  "offline",
				})
				h.BroadcastExcept(client.roomID, offlineMsg, client)
			}

			close(client.send)

		case msg := <-h.broadcast:
			h.mu.RLock()
			room := h.rooms[msg.roomID]
			h.mu.RUnlock()
			for client := range room {
				select {
				case client.send <- msg.payload:
				default:
					// Slow client — drop and disconnect
					h.leave <- client
				}
			}
		}
	}
}

// Broadcast sends a message payload to all clients in a room.
func (h *WSHub) Broadcast(roomID uint, payload []byte) {
	h.broadcast <- broadcastMsg{roomID: roomID, payload: payload}
}

// BroadcastExcept sends a payload to all clients in a room except the specified client.
func (h *WSHub) BroadcastExcept(roomID uint, payload []byte, exclude *wsClient) {
	h.mu.RLock()
	room := h.rooms[roomID]
	h.mu.RUnlock()
	for client := range room {
		if client == exclude {
			continue
		}
		select {
		case client.send <- payload:
		default:
			h.leave <- client
		}
	}
}

// SendToUser sends a payload to all global connections of a specific user (room 0).
func (h *WSHub) SendToUser(userID uint, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if globalRoom, ok := h.rooms[0]; ok {
		for client := range globalRoom {
			if client.userID == userID {
				select {
				case client.send <- payload:
				default:
				}
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"Bearer"},
}

// ServeWS is the Gin handler that upgrades an HTTP request to a WebSocket
// connection and attaches the client to a hub room.
//
// Route: GET /api/ws/chat/:id
func ServeWS(hub *WSHub, chatSvc service.ChatService, authSvc service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string
		protocolHeader := c.GetHeader("Sec-WebSocket-Protocol")
		if protocolHeader != "" {
			parts := strings.Split(protocolHeader, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			} else if len(parts) == 1 {
				tokenStr = parts[0]
			}
		}

		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenStr = authHeader[7:]
			}
		}

		claims, err := authSvc.ValidateToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		roomID, ok := parseID(c, "id")
		if !ok {
			return
		}

		// Verify room participation
		room, err := chatSvc.GetRoom(roomID, claims.UserID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "chat room not found"})
			return
		}

		var pids []uint
		if err := json.Unmarshal([]byte(room.Participants), &pids); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse room participants"})
			return
		}

		isParticipant := false
		for _, pid := range pids {
			if pid == claims.UserID {
				isParticipant = true
				break
			}
		}

		if !isParticipant && claims.Role != "admin" && claims.Role != "moderator" {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a participant of this chat room"})
			return
		}

		// Prepare response headers for Sec-WebSocket-Protocol matching
		responseHeader := http.Header{}
		if protocolHeader != "" {
			responseHeader.Set("Sec-WebSocket-Protocol", "Bearer")
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
		if err != nil {
			slog.Error("ws upgrade failed", "error", err)
			return
		}

		client := &wsClient{
			roomID: roomID,
			userID: claims.UserID,
			conn:   conn,
			send:   make(chan []byte, 64),
			hub:    hub,
		}

		hub.join <- client

		// Start writer and reader goroutines
		go client.writePump()
		go client.readPump(chatSvc, claims.Username)
	}
}

// ServeGlobalWS upgrades an HTTP request to a WebSocket connection for global/notifications.
// Route: GET /api/ws/global
func ServeGlobalWS(hub *WSHub, authSvc service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string
		protocolHeader := c.GetHeader("Sec-WebSocket-Protocol")
		if protocolHeader != "" {
			parts := strings.Split(protocolHeader, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			} else if len(parts) == 1 {
				tokenStr = parts[0]
			}
		}

		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenStr = authHeader[7:]
			}
		}

		claims, err := authSvc.ValidateToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Prepare response headers for Sec-WebSocket-Protocol matching
		responseHeader := http.Header{}
		if protocolHeader != "" {
			responseHeader.Set("Sec-WebSocket-Protocol", "Bearer")
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
		if err != nil {
			slog.Error("global ws upgrade failed", "error", err)
			return
		}

		client := &wsClient{
			roomID: 0, // 0 is global/notifications
			userID: claims.UserID,
			conn:   conn,
			send:   make(chan []byte, 64),
			hub:    hub,
		}

		hub.join <- client

		// Send initial unread notification count
		var count int64
		if hub.db != nil {
			hub.db.Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", claims.UserID, false).Count(&count)
		}
		unreadMsg, _ := json.Marshal(struct {
			Type  string `json:"type"`
			Count int64  `json:"count"`
		}{
			Type:  "unread_count",
			Count: count,
		})
		client.send <- unreadMsg

		// Start writer and reader goroutines
		go client.writePump()
		go client.readPump(nil, claims.Username)
	}
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// readPump reads messages from the WebSocket and broadcasts them to the room.
func (c *wsClient) readPump(chatSvc service.ChatService, senderName string) {
	defer func() {
		c.hub.leave <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("ws unexpected close", "error", err, "user_id", c.userID)
			}
			return
		}

		var incoming WSMessage
		if err := json.Unmarshal(raw, &incoming); err != nil {
			continue
		}

		if incoming.Type == "ping" {
			pong, _ := json.Marshal(WSMessage{Type: "pong", Timestamp: time.Now()})
			c.send <- pong
			continue
		}

		if incoming.Type == "typing" {
			if c.roomID == 0 {
				continue
			}
			// Broadcast typing status to everyone else in the room
			out := WSMessage{
				Type:     "typing",
				RoomID:   c.roomID,
				SenderID: c.userID,
				Content:  incoming.Content, // "true" or "false"
			}
			payload, _ := json.Marshal(out)
			c.hub.BroadcastExcept(c.roomID, payload, c)
			continue
		}

		if incoming.Type == "read" {
			if c.roomID == 0 {
				continue
			}
			// Update messages in this room as read up to the given message ID
			msgID, err := strconv.ParseUint(incoming.Content, 10, 32)
			if err == nil && c.hub.db != nil {
				go func(roomId, userId, maxMsgId uint) {
					c.hub.db.Model(&model.Message{}).Where("chat_room_id = ? AND sender_id != ? AND id <= ?", roomId, userId, maxMsgId).Update("is_read", true)
				}(c.roomID, c.userID, uint(msgID))
			}

			// Broadcast read status to all room participants
			out := WSMessage{
				Type:     "read",
				RoomID:   c.roomID,
				SenderID: c.userID,
				Content:  incoming.Content, // the last read message ID
			}
			payload, _ := json.Marshal(out)
			c.hub.Broadcast(c.roomID, payload)
			continue
		}

		if incoming.Type == "message" {
			if c.roomID == 0 || chatSvc == nil {
				continue
			}
			// Check if any other participant is online and in the room to set is_delivered
			c.hub.mu.RLock()
			roomClients := c.hub.rooms[c.roomID]
			hasOtherParticipants := false
			for rc := range roomClients {
				if rc.userID != c.userID {
					hasOtherParticipants = true
					break
				}
			}
			c.hub.mu.RUnlock()

			// Persist the message via ChatService
			msg, err := chatSvc.SendMessage(c.userID, c.roomID, incoming.Content)
			if err != nil {
				slog.Warn("ws failed to persist message", "error", err)
				continue
			}

			if hasOtherParticipants {
				msg.IsDelivered = true
				if c.hub.db != nil {
					go func(msgId uint) {
						c.hub.db.Model(&model.Message{}).Where("id = ?", msgId).Update("is_delivered", true)
					}(msg.ID)
				}
			}

			// Build outgoing envelope and broadcast to all room members
			out := WSMessage{
				Type:        "message",
				RoomID:      c.roomID,
				SenderID:    c.userID,
				SenderName:  senderName,
				Content:     incoming.Content,
				IsRead:      msg.IsRead,
				IsDelivered: msg.IsDelivered,
				Timestamp:   msg.CreatedAt,
			}
			payload, _ := json.Marshal(out)
			c.hub.Broadcast(c.roomID, payload)
			continue
		}
	}
}

// writePump writes messages from the send channel to the WebSocket.
func (c *wsClient) writePump() {
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
				// Hub closed the channel
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
