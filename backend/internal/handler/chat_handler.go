package handler

import (
	"net/http"

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
