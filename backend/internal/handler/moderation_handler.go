package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ================= Moderation Handlers =================

func (h *Handlers) ShadowBanUser(c *gin.Context) {
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

	if err := h.ModService.ShadowBanUser(modID, targetID, body.Reason); err != nil {
		slog.Error("shadow ban failed", "moderator_id", modID, "target_id", targetID, "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	slog.Info("user shadow-banned", "moderator_id", modID, "target_id", targetID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "user shadow-banned"})
}

func (h *Handlers) UnshadowBanUser(c *gin.Context) {
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

	if err := h.ModService.UnshadowBanUser(modID, targetID, body.Reason); err != nil {
		slog.Error("unshadow ban failed", "moderator_id", modID, "target_id", targetID, "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	slog.Info("user unshadow-banned", "moderator_id", modID, "target_id", targetID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "user unshadow-banned"})
}

func (h *Handlers) AddKeywordFilter(c *gin.Context) {
	_, ok := getUserID(c)
	if !ok {
		return
	}

	var body struct {
		Pattern string `json:"pattern" binding:"required"`
		IsRegex bool   `json:"is_regex"`
		Action  string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	modID, _ := getUserID(c)
	if err := h.ModService.AddKeywordFilter(modID, body.Pattern, body.IsRegex, body.Action); err != nil {
		if err.Error() == "insufficient permissions: admin or moderator role required" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) RemoveKeywordFilter(c *gin.Context) {
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	filterID, ok := parseID(c, "id")
	if !ok {
		return
	}

	if err := h.ModService.RemoveKeywordFilter(modID, filterID); err != nil {
		if err.Error() == "insufficient permissions: admin or moderator role required" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) ListKeywordFilters(c *gin.Context) {
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	filters, err := h.ModService.ListKeywordFilters(modID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, filters)
}


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
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.ModService.GetLogs(modID, limit)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *Handlers) GetCommunityModerationLogs(c *gin.Context) {
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	commID, ok := parseID(c, "id")
	if !ok {
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.ModService.GetLogsByCommunity(modID, commID, limit)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *Handlers) GetReports(c *gin.Context) {
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	reports, err := h.ModService.GetReports(modID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	enriched := make([]gin.H, 0, len(reports))
	for _, report := range reports {
		item := gin.H{
			"id":                 report.ID,
			"reporter_id":        report.ReporterID,
			"reporter_username":  report.ReporterUsername,
			"target_id":          report.TargetID,
			"target_type":        report.TargetType,
			"reason":             report.Reason,
			"description":        report.Description,
			"status":             report.Status,
			"moderator_response": report.ModeratorResponse,
			"created_date":       report.CreatedAt,
			"target_summary":     "",
			"target_username":    "",
			"target_user_id":     uint(0),
			"post_id":            uint(0),
		}
		switch report.TargetType {
		case "post":
			if post, err := h.PostService.GetByID(report.TargetID); err == nil {
				item["target_summary"] = post.Title
				item["target_user_id"] = post.AuthorID
				item["post_id"] = post.ID
				if author, err := h.UserService.GetByID(post.AuthorID); err == nil {
					item["target_username"] = author.Username
				}
			}
		case "comment":
			if comment, err := h.CommentService.GetByID(report.TargetID); err == nil {
				summary := comment.Content
				if len(summary) > 120 {
					summary = summary[:120] + "…"
				}
				item["target_summary"] = summary
				item["target_user_id"] = comment.AuthorID
				item["post_id"] = comment.PostID
				if author, err := h.UserService.GetByID(comment.AuthorID); err == nil {
					item["target_username"] = author.Username
				}
			}
		case "user":
			if u, err := h.UserService.GetByID(report.TargetID); err == nil {
				item["target_summary"] = u.Username
				item["target_user_id"] = u.ID
				item["target_username"] = u.Username
			}
		}
		enriched = append(enriched, item)
	}

	c.JSON(http.StatusOK, enriched)
}

func (h *Handlers) UpdateReport(c *gin.Context) {
	modID, ok := getUserID(c)
	if !ok {
		return
	}

	reportID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var body struct {
		Status            string `json:"status"`
		ModeratorResponse string `json:"moderator_response"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error
	if body.Status == "resolved" {
		err = h.ModService.ResolveReport(modID, reportID, body.ModeratorResponse)
	} else if body.Status == "rejected" || body.Status == "dismissed" {
		err = h.ModService.RejectReport(modID, reportID, body.ModeratorResponse)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status, must be resolved or rejected"})
		return
	}

	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
