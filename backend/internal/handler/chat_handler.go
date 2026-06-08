package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"nexus-forum-backend/internal/dto"
	"nexus-forum-backend/internal/model"

	"github.com/gin-gonic/gin"
)

// ================= Chat Handlers =================

func (h *Handlers) CreateChatRoom(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.CreateChatRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room, err := h.ChatService.CreateRoom(uid, req.Participants, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, room)
}

func (h *Handlers) GetChatRooms(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	rooms, err := h.ChatService.GetRoomsByUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rooms)
}

func (h *Handlers) GetMessages(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	roomID, ok := parseID(c, "id")
	if !ok {
		return
	}

	// Verify room participation
	room, err := h.ChatService.GetRoom(roomID, uid)
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
		if pid == uid {
			isParticipant = true
			break
		}
	}

	roleVal, _ := c.Get("role")
	roleStr, _ := roleVal.(string)
	if !isParticipant && roleStr != "admin" && roleStr != "moderator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a participant of this chat room"})
		return
	}

	msgs, err := h.ChatService.GetMessages(roomID, uid, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, msgs)
}

func (h *Handlers) SendMessage(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	roomID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req dto.CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Content == "" && req.AttachmentURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message content or attachment is required"})
		return
	}

	// 1. Verify room participation
	room, err := h.ChatService.GetRoom(roomID, uid)
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
		if pid == uid {
			isParticipant = true
			break
		}
	}

	roleVal, _ := c.Get("role")
	roleStr, _ := roleVal.(string)
	if !isParticipant && roleStr != "admin" && roleStr != "moderator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a participant of this chat room"})
		return
	}

	// 2. Check if other participant is currently connected to room to mark delivered
	h.WSHub.mu.RLock()
	roomClients := h.WSHub.rooms[roomID]
	hasOtherParticipants := false
	for rc := range roomClients {
		if rc.userID != uid {
			hasOtherParticipants = true
			break
		}
	}
	h.WSHub.mu.RUnlock()

	msg, err := h.ChatService.SendMessageWithAttachment(uid, roomID, req.Content, req.AttachmentURL, req.AttachmentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Printf("MESSAGE CREATED ID = %d", msg.ID)

	if hasOtherParticipants {
		msg.IsDelivered = true
		if h.WSHub.db != nil {
			h.WSHub.db.Model(&model.Message{}).Where("id = ?", msg.ID).Update("is_delivered", true)
		}
	}

	// 3. Broadcast message via WebSocket
	payload, _ := json.Marshal(WSMessage{
		ID:             msg.ID,
		Type:           "message",
		RoomID:         roomID,
		SenderID:       uid,
		SenderName:     msg.SenderUsername,
		Content:        msg.Content,
		IsRead:         msg.IsRead,
		IsDelivered:    msg.IsDelivered,
		AttachmentURL:  msg.AttachmentURL,
		AttachmentType: msg.AttachmentType,
		Timestamp:      msg.CreatedAt,
	})

	log.Printf("WS SEND ID = %d", msg.ID)

	h.WSHub.Broadcast(roomID, payload)

	c.JSON(http.StatusOK, msg)
}

func (h *Handlers) UpdateChatRoom(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	roomID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Mark read if requested
	if unreadVal, exists := req["unread_count"]; exists {
		if val, ok := unreadVal.(float64); ok && val == 0 {
			_ = h.ChatService.MarkRoomAsRead(roomID, uid)
		}
	}

	// 2. Fetch room to update details
	room, err := h.ChatService.GetRoom(roomID, uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	if nameVal, exists := req["name"]; exists {
		if nameStr, ok := nameVal.(string); ok {
			room.Name = nameStr
		}
	}
	if lastMsgVal, exists := req["last_message"]; exists {
		if lastMsgStr, ok := lastMsgVal.(string); ok {
			room.LastMessage = lastMsgStr
			room.LastMessageAt = time.Now()
		}
	}

	_ = h.ChatService.UpdateRoom(room)

	c.JSON(http.StatusOK, room)
}

func (h *Handlers) DeleteMessage(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	msgID, ok := parseID(c, "id")
	if !ok {
		return
	}

	deleteType := c.DefaultQuery("type", "me")
	if deleteType != "me" && deleteType != "everyone" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid delete type"})
		return
	}

	msg, err := h.ChatService.DeleteMessage(uid, msgID, deleteType)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if deleteType == "everyone" {
		// Broadcast deleted message event to room
		payload, _ := json.Marshal(struct {
			Type      string `json:"type"`
			RoomID    uint   `json:"room_id"`
			MessageID uint   `json:"message_id"`
			Content   string `json:"content"`
		}{
			Type:      "message_deleted",
			RoomID:    msg.ChatRoomID,
			MessageID: msg.ID,
			Content:   msg.Content,
		})
		h.WSHub.Broadcast(msg.ChatRoomID, payload)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) UpdateMessage(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	msgID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.ChatService.UpdateMessage(uid, msgID, req.Content)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Broadcast updated message event to room
	payload, _ := json.Marshal(struct {
		Type      string    `json:"type"`
		RoomID    uint      `json:"room_id"`
		MessageID uint      `json:"message_id"`
		Content   string    `json:"content"`
		IsEdited  bool      `json:"is_edited"`
		Timestamp time.Time `json:"timestamp"`
	}{
		Type:      "message_edited",
		RoomID:    msg.ChatRoomID,
		MessageID: msg.ID,
		Content:   msg.Content,
		IsEdited:  msg.IsEdited,
		Timestamp: msg.CreatedAt,
	})
	h.WSHub.Broadcast(msg.ChatRoomID, payload)

	c.JSON(http.StatusOK, msg)
}

func (h *Handlers) DeleteChatRoom(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	roomID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.ChatService.DeleteRoom(uid, roomID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
