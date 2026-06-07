package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/service"
)

type Handlers struct {
	AuthService    service.AuthService
	UserService    service.UserService
	CommService    service.CommunityService
	PostService    service.PostService
	CommentService service.CommentService
	ChatService    service.ChatService
	NotifService   service.NotificationService
	ModService     service.ModerationService
	Analytics      service.AnalyticsService
}

func NewHandlers(
	auth service.AuthService,
	user service.UserService,
	comm service.CommunityService,
	post service.PostService,
	comment service.CommentService,
	chat service.ChatService,
	notif service.NotificationService,
	mod service.ModerationService,
	analytics service.AnalyticsService,
) *Handlers {
	return &Handlers{
		AuthService:    auth,
		UserService:    user,
		CommService:    comm,
		PostService:    post,
		CommentService: comment,
		ChatService:    chat,
		NotifService:   notif,
		ModService:     mod,
		Analytics:      analytics,
	}
}

// Helper: parse uint ID from path
func parseID(c *gin.Context, paramName string) (uint, bool) {
	valStr := c.Param(paramName)
	val, err := strconv.ParseUint(valStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID parameter"})
		return 0, false
	}
	return uint(val), true
}

// Helper: get current authorized user ID from context
func getUserID(c *gin.Context) (uint, bool) {
	uid, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return 0, false
	}
	return uid.(uint), true
}
