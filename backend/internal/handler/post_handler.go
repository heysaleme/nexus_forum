package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/dto"
	"nexus-forum-backend/internal/model"
)

// ================= Post Handlers =================

func (h *Handlers) CreatePost(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mediaBytes, _ := json.Marshal(req.MediaUrls)
	tagsBytes, _ := json.Marshal(req.Tags)

	post := &model.Post{
		CommunityID: req.CommunityID,
		AuthorID:    uid,
		Title:       req.Title,
		Content:     req.Content,
		Type:        req.Type,
		MediaUrls:   string(mediaBytes),
		LinkUrl:     req.LinkUrl,
		Tags:        string(tagsBytes),
		PollOptions: req.PollOptions,
	}

	err := h.PostService.Create(post)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, post)
}

func (h *Handlers) GetPostByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	post, err := h.PostService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, post)
}

func (h *Handlers) ListPosts(c *gin.Context) {
	sortSpec := c.Query("sort")

	// Check query filters
	filter := make(map[string]interface{})
	filter["status"] = "published"

	if commIDStr := c.Query("community_id"); commIDStr != "" {
		commID, err := strconv.ParseUint(commIDStr, 10, 32)
		if err == nil {
			filter["community_id"] = uint(commID)
		}
	}

	if authorIDStr := c.Query("author_id"); authorIDStr != "" {
		authorID, err := strconv.ParseUint(authorIDStr, 10, 32)
		if err == nil {
			filter["author_id"] = uint(authorID)
		}
	}

	var posts []*model.Post
	var err error
	if len(filter) > 1 { // more than just status
		posts, err = h.PostService.Filter(filter, sortSpec, 50)
	} else {
		posts, err = h.PostService.List(sortSpec, 50)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, posts)
}

func (h *Handlers) DeletePost(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	postID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.PostService.Delete(uid, postID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) VotePost(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	postID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req dto.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.PostService.Vote(uid, postID, req.Value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) SavePost(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	postID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.PostService.SavePost(uid, postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) UnsavePost(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	postID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.PostService.UnsavePost(uid, postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) GetSavedPosts(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	saved, err := h.PostService.GetSavedByUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, saved)
}
