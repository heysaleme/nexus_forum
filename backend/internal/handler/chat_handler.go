package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/dto"
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
	_, ok := getUserID(c)
	if !ok {
		return
	}

	roomID, ok := parseID(c, "id")
	if !ok {
		return
	}

	msgs, err := h.ChatService.GetMessages(roomID, 50)
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

	msg, err := h.ChatService.SendMessage(uid, roomID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	err := h.ChatService.DeleteMessage(uid, msgID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
