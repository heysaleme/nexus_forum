package handler

import (
	"net/http"
	"strconv"

	"nexus-forum-backend/internal/dto"
	"nexus-forum-backend/internal/middleware"
	"nexus-forum-backend/internal/model"

	"github.com/gin-gonic/gin"
)

// ================= Comment Handlers =================

func (h *Handlers) CreateComment(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	h.ModService.RecordAction(uid, "comment")

	if user, err := h.UserService.GetByID(uid); err == nil && user.IsSuspicious {
		turnstileToken := c.GetHeader("X-Turnstile-Token")

		if ok, _ := middleware.VerifyTurnstileToken(
			h.TurnstileSecret,
			turnstileToken,
			c.ClientIP(),
		); !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "CAPTCHA verification required",
			})
			return
		}
	}

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isShadowContent := false

	if matched, action, pattern := h.ModService.CheckContent(req.Content); matched {

		if action == "block" {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "Content contains prohibited terms",
				"pattern": pattern,
			})
			return
		}

		if action == "shadow" {
			isShadowContent = true
		}
	}

	if user, err := h.UserService.GetByID(uid); err == nil && user.IsShadowBanned {
		isShadowContent = true
	}

	comment := &model.Comment{
		PostID:          req.PostID,
		ParentID:        req.ParentID,
		AuthorID:        uid,
		Content:         req.Content,
		IsShadowContent: isShadowContent,
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

	viewerID, authenticated := getOptionalUserID(c, h.AuthService)
	if !authenticated {
		viewerID = 0
	}

	comments, err := h.CommentService.GetByPostID(uint(postID), viewerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID, authenticated := getOptionalUserID(c, h.AuthService)

	if authenticated {
		for _, comment := range comments {
			vote, err := h.CommentService.GetVote(userID, comment.ID)

			if err == nil {
				comment.UserVote = vote.Value
			}
		}
	}

	c.JSON(http.StatusOK, comments)
}

func (h *Handlers) UpdateComment(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	commentID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment, err := h.CommentService.Update(uid, commentID, req.Content)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comment)
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

func (h *Handlers) DeleteComment(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	commentID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.CommentService.Delete(uid, commentID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
