package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"nexus-forum-backend/internal/dto"
	"nexus-forum-backend/internal/model"

	"github.com/gin-gonic/gin"
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

	var pollOptionsStr string
	if req.PollOptions != nil {
		pollBytes, _ := json.Marshal(req.PollOptions)
		pollOptionsStr = string(pollBytes)
	}

	post := &model.Post{
		CommunityID: req.CommunityID,
		AuthorID:    uid,
		Title:       req.Title,
		Content:     req.Content,
		Type:        req.Type,
		MediaUrls:   string(mediaBytes),
		LinkUrl:     req.LinkUrl,
		Tags:        string(tagsBytes),
		PollOptions: pollOptionsStr,
		IsNSFW:      req.IsNSFW != nil && *req.IsNSFW,
		IsSpoiler:   req.IsSpoiler != nil && *req.IsSpoiler,
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

	// Verify author privacy
	author, err := h.UserService.GetByID(post.AuthorID)
	if err == nil && author.IsPrivate {
		reqUserID, isAuthenticated := getOptionalUserID(c, h.AuthService)
		isAuthorized := false
		if isAuthenticated {
			if reqUserID == post.AuthorID {
				isAuthorized = true
			} else {
				following, _ := h.UserService.IsFollowing(reqUserID, post.AuthorID)
				if following {
					isAuthorized = true
				}
			}
		}
		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "This account is private. Follow the author to view their posts."})
			return
		}
	}

	userID, authenticated := getOptionalUserID(c, h.AuthService)

	fmt.Println("===== GET POST =====")
	fmt.Println("USER:", userID)
	fmt.Println("AUTH:", authenticated)

	if authenticated {
		vote, err := h.PostService.GetVote(userID, post.ID)

		fmt.Println("VOTE:", vote)
		fmt.Println("ERR:", err)

		if err == nil {
			post.UserVote = vote.Value
			fmt.Println("SET USER_VOTE:", vote.Value)
		}
	}
	fmt.Println("JSON USER_VOTE:", post.UserVote)
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
			targetAuthorID := uint(authorID)
			filter["author_id"] = targetAuthorID

			// Verify privacy
			author, err := h.UserService.GetByID(targetAuthorID)
			if err == nil && author.IsPrivate {
				reqUserID, isAuthenticated := getOptionalUserID(c, h.AuthService)
				isAuthorized := false
				if isAuthenticated {
					if reqUserID == targetAuthorID {
						isAuthorized = true
					} else {
						following, _ := h.UserService.IsFollowing(reqUserID, targetAuthorID)
						if following {
							isAuthorized = true
						}
					}
				}
				if !isAuthorized {
					c.JSON(http.StatusOK, []*model.Post{})
					return
				}
			}
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

	userID, authenticated := getOptionalUserID(c, h.AuthService)

	println("USER:", userID)

	println("AUTH:", authenticated)

	if authenticated {
		for _, post := range posts {
			vote, err := h.PostService.GetVote(userID, post.ID)
			fmt.Println("POST:", post.ID)

			fmt.Println("ERR:", err)
			if err == nil {
				post.UserVote = vote.Value
			}
		}
	}

	c.JSON(http.StatusOK, posts)
}

func (h *Handlers) UpdatePost(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	postID, ok := parseID(c, "id")
	if !ok {
		return
	}

	post, err := h.PostService.GetByID(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	user, _ := h.UserService.GetByID(uid)
	isAdmin := user != nil && (user.Role == "admin" || user.Role == "moderator")

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Author can edit title/content, admin/mod can additionally pin
	if post.AuthorID == uid {
		if v, ok := req["title"]; ok {
			if s, ok := v.(string); ok {
				post.Title = s
			}
		}
		if v, ok := req["content"]; ok {
			if s, ok := v.(string); ok {
				post.Content = s
			}
		}
		if v, ok := req["is_nsfw"]; ok {
			if b, ok := v.(bool); ok {
				post.IsNSFW = b
			}
		}
		if v, ok := req["is_spoiler"]; ok {
			if b, ok := v.(bool); ok {
				post.IsSpoiler = b
			}
		}
	}

	// Admins/mods/community owners can pin and change status
	if isAdmin {
		if v, ok := req["is_pinned"]; ok {
			if b, ok := v.(bool); ok {
				post.IsPinned = b
			}
		}
		if v, ok := req["status"]; ok {
			if s, ok := v.(string); ok {
				post.Status = s
			}
		}
	} else {
		// Allow community owner to pin
		if v, ok := req["is_pinned"]; ok {
			if b, ok := v.(bool); ok {
				comm, err := h.CommService.GetByID(post.CommunityID)
				if err == nil && comm.OwnerID == uid {
					post.IsPinned = b
				}
			}
		}
	}

	// Also allow generic field updates for view count etc
	if v, ok := req["views"]; ok {
		if f, ok := v.(float64); ok {
			post.Views = int(f)
		}
	}

	err = h.PostService.Update(post)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, post)
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

	if req.Value != -1 && req.Value != 0 && req.Value != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "vote must be -1, 0 or 1",
		})
		return
	}

	err := h.PostService.Vote(uid, postID, req.Value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) VotePoll(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	postID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var body struct {
		OptionIndex int `json:"option_index"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.PostService.VotePoll(uid, postID, body.OptionIndex)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
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
