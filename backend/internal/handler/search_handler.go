package handler

import (
	"net/http"

	"nexus-forum-backend/internal/model"

	"github.com/gin-gonic/gin"
)

// ================= Search Handlers =================

func (h *Handlers) Search(c *gin.Context) {
	query := c.Query("q")
	reqUserID, isAuthenticated := getOptionalUserID(c, h.AuthService)

	// DB level search for posts, communities, users
	viewerID := uint(0)
	if isAuthenticated {
		viewerID = reqUserID
	}

	if query == "" {
		c.JSON(http.StatusOK, gin.H{"posts": []*model.Post{}, "communities": []*model.Community{}, "users": []*model.User{}})
		return
	}

	posts, err := h.PostService.Search(query, 30, viewerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	communities, err := h.CommService.Search(query, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	users, err := h.UserService.Search(query, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type SearchResults struct {
		Posts       []*model.Post      `json:"posts"`
		Communities []*model.Community `json:"communities"`
		Users       []*model.User      `json:"users"`
	}

	res := SearchResults{
		Posts:       []*model.Post{},
		Communities: communities,
		Users:       []*model.User{},
	}

	// 1. Filter Users
	for _, u := range users {
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
				continue // skip private user from search results if not authorized
			}
		}
		res.Users = append(res.Users, u)
	}

	// 2. Filter Posts
	for _, p := range posts {
		author, err := h.UserService.GetByID(p.AuthorID)
		if err != nil {
			res.Posts = append(res.Posts, p)
			continue
		}
		if author.IsPrivate {
			isAuthorized := false
			if isAuthenticated {
				if reqUserID == p.AuthorID {
					isAuthorized = true
				} else {
					following, _ := h.UserService.IsFollowing(reqUserID, p.AuthorID)
					if following {
						isAuthorized = true
					}
				}
			}
			if !isAuthorized {
				continue // skip private user's post if not authorized
			}
		}
		res.Posts = append(res.Posts, p)
	}

	c.JSON(http.StatusOK, res)
}
