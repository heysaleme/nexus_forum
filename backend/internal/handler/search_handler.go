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
	// Generic search returning posts, communities, and users matching query
	// Filters out Wiki since wiki articles are removed.
	filter := map[string]interface{}{"status": "published"}
	posts, _ := h.PostService.Filter(filter, "-score", 30)
	communities, _ := h.CommService.List("-member_count", 30)
	users, _ := h.UserService.List("-xp", 100)

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

	reqUserID, isAuthenticated := getOptionalUserID(c, h.AuthService)
	for _, u := range users {
		if strings.Contains(strings.ToLower(u.Username), term) || strings.Contains(strings.ToLower(u.Bio), term) {
			// Check privacy
			if u.IsPrivate {
				isAuthorized := false
				if isAuthenticated {
					if reqUserID == u.ID {
						isAuthorized = true
					} else {
						following, _ := h.UserService.IsFollowing(reqUserID, u.ID)
						if following {
							isAuthorized = true
						}
					}
				}
				if !isAuthorized {
					continue // skip private user if not authorized
				}
			}
			res.Users = append(res.Users, u)
		}
	}

	c.JSON(http.StatusOK, res)
}
