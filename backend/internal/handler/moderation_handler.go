package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ================= Moderation Handlers =================

func (h *Handlers) BanUser(c *gin.Context) {
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	targetID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	if err := h.ModService.BanUser(modID, targetID, body.Reason); err != nil {
		slog.Error("ban user failed", "moderator_id", modID, "target_id", targetID, "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	slog.Info("user banned", "moderator_id", modID, "target_id", targetID, "reason", body.Reason)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "user banned"})
}

func (h *Handlers) UnbanUser(c *gin.Context) {
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	targetID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	if err := h.ModService.UnbanUser(modID, targetID, body.Reason); err != nil {
		slog.Error("unban user failed", "moderator_id", modID, "target_id", targetID, "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	slog.Info("user unbanned", "moderator_id", modID, "target_id", targetID, "reason", body.Reason)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "user unbanned"})
}

func (h *Handlers) RemovePost(c *gin.Context) {
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	postID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	if err := h.ModService.RemovePost(modID, postID, body.Reason); err != nil {
		slog.Error("post removal failed", "moderator_id", modID, "post_id", postID, "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	slog.Info("post removed by moderator", "moderator_id", modID, "post_id", postID, "reason", body.Reason)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "post removed"})
}

func (h *Handlers) RemoveComment(c *gin.Context) {
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	commentID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	if err := h.ModService.RemoveComment(modID, commentID, body.Reason); err != nil {
		slog.Error("comment removal failed", "moderator_id", modID, "comment_id", commentID, "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	slog.Info("comment removed by moderator", "moderator_id", modID, "comment_id", commentID, "reason", body.Reason)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "comment removed"})
}

func (h *Handlers) GetModerationLogs(c *gin.Context) {
	_, ok := getUserID(c)
	if !ok {
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.ModService.GetLogs(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *Handlers) GetCommunityModerationLogs(c *gin.Context) {
	_, ok := getUserID(c)
	if !ok {
		return
	}

	commID, ok := parseID(c, "id")
	if !ok {
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.ModService.GetLogsByCommunity(commID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}
