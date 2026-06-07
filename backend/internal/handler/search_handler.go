package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/model"
)

// ================= Search Handlers =================

func (h *Handlers) Search(c *gin.Context) {
	query := c.Query("q")
	// Generic search returning posts and communities matching query
	// Filters out Wiki since wiki articles are removed.
	filter := map[string]interface{}{"status": "published"}
	posts, _ := h.PostService.Filter(filter, "-score", 30)
	communities, _ := h.CommService.List("-member_count", 30)

	type SearchResults struct {
		Posts       []*model.Post      `json:"posts"`
		Communities []*model.Community `json:"communities"`
		Users       []*model.User      `json:"users"`
	}

	res := SearchResults{
		Posts:       []*model.Post{},
		Communities: []*model.Community{},
		Users:       []*model.User{},
	}

	term := strings.ToLower(query)
	for _, p := range posts {
		if strings.Contains(strings.ToLower(p.Title), term) || strings.Contains(strings.ToLower(p.Content), term) {
			res.Posts = append(res.Posts, p)
		}
	}

	for _, comm := range communities {
		if strings.Contains(strings.ToLower(comm.Name), term) || strings.Contains(strings.ToLower(comm.Description), term) {
			res.Communities = append(res.Communities, comm)
		}
	}

	c.JSON(http.StatusOK, res)
}
