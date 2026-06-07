package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/dto"
	"nexus-forum-backend/internal/model"
)

// ================= Comment Handlers =================

func (h *Handlers) CreateComment(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := &model.Comment{
		PostID:   req.PostID,
		ParentID: req.ParentID,
		AuthorID: uid,
		Content:  req.Content,
	}

	err := h.CommentService.Create(comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comment)
}

func (h *Handlers) ListComments(c *gin.Context) {
	postIDStr := c.Query("post_id")
	if postIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post_id query parameter required"})
		return
	}

	postID, err := strconv.ParseUint(postIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post_id"})
		return
	}

	comments, err := h.CommentService.GetByPostID(uint(postID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comments)
}

func (h *Handlers) VoteComment(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	commentID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req dto.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.CommentService.Vote(uid, commentID, req.Value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
