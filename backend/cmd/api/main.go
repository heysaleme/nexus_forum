package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"nexus-forum-backend/internal/config"
	"nexus-forum-backend/internal/handler"
	"nexus-forum-backend/internal/middleware"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
	"nexus-forum-backend/internal/service"
)

func main() {
	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Init structured logger
	middleware.InitLogger()
	logger := middleware.Logger

	logger.Info("starting backend server", "port", cfg.Port, "db_type", cfg.DBType)

	// 3. Connect to Database
	var db *gorm.DB
	if cfg.DBType == "postgres" {
		db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
		if err != nil {
			logger.Warn("failed to connect to PostgreSQL, falling back to SQLite", "error", err)
			db, err = gorm.Open(sqlite.Open(cfg.SqliteDB), &gorm.Config{})
		}
	} else {
		db, err = gorm.Open(sqlite.Open(cfg.SqliteDB), &gorm.Config{})
	}

	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// 4. Auto Migrate
	err = db.AutoMigrate(
		&model.User{},
		&model.Community{},
		&model.CommunityMember{},
		&model.Post{},
		&model.Comment{},
		&model.Vote{},
		&model.PollVote{},
		&model.SavedPost{},
		&model.UserFollow{},
		&model.Notification{},
		&model.ChatRoom{},
		&model.Message{},
		&model.ModerationLog{},
		&model.AnalyticsEvent{},
		&model.Report{},
		&model.KeywordFilter{},
		&model.PasswordResetToken{},
		&model.RefreshToken{},
	)
	if err != nil {
		log.Fatalf("failed to auto migrate tables: %v", err)
	}

	// Data fix for existing follows
	db.Exec("UPDATE user_follows SET status = 'accepted' WHERE status = '' OR status IS NULL")

	logger.Info("database auto-migrations complete")

	// 5. Seed Demo Data if empty
	seedDemoData(db)

	// 6. Initialize layers
	userRepo := repository.NewUserRepository(db)
	commRepo := repository.NewCommunityRepository(db)
	postRepo := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	savedRepo := repository.NewSavedPostRepository(db)
	followRepo := repository.NewFollowRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	chatRepo := repository.NewChatRepository(db)
	modRepo := repository.NewModerationRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	keywordFilterRepo := repository.NewKeywordFilterRepository(db)
	resetRepo := repository.NewPasswordResetRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)

	authService := service.NewAuthService(userRepo, modRepo, resetRepo, refreshRepo, cfg.JWTSecret)
	userService := service.NewUserService(userRepo, followRepo, notifRepo, modRepo)
	commService := service.NewCommunityService(commRepo, userRepo)
	postService := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)
	commentService := service.NewCommentService(commentRepo, userRepo, postRepo, voteRepo, notifRepo, commRepo)
	chatService := service.NewChatService(chatRepo, userRepo)
	notifService := service.NewNotificationService(notifRepo)
	modService := service.NewModerationService(modRepo, userRepo, postRepo, commentRepo, commRepo, notifRepo, keywordFilterRepo)
	analyticsService := service.NewAnalyticsService(analyticsRepo, userRepo, postRepo)

	// WebSocket hub (starts background goroutine)
	wsHub := handler.NewWSHub(db)

	// Register global NotificationDispatcher to trigger real-time WS notifications
	repository.NotificationDispatcher = func(userID uint, notif *model.Notification) {
		payload, err := json.Marshal(struct {
			Type string             `json:"type"`
			Data *model.Notification `json:"data"`
		}{
			Type: "notification",
			Data: notif,
		})
		if err == nil {
			wsHub.SendToUser(userID, payload)
		}

		// Push updated count as well
		var count int64
		if db != nil {
			db.Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count)
			countPayload, err := json.Marshal(struct {
				Type  string `json:"type"`
				Count int64  `json:"count"`
			}{
				Type:  "unread_count",
				Count: count,
			})
			if err == nil {
				wsHub.SendToUser(userID, countPayload)
			}
		}
	}

	handlers := handler.NewHandlers(authService, userService, commService, postService, commentService, chatService, notifService, modService, analyticsService, wsHub, cfg.TurnstileSecret)

	// OAuth handler config
	oauthCfg := handler.OAuthConfig{
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		FrontendURL:        cfg.FrontendURL,
	}

	// 7. Setup Gin Router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Apply Middlewares
	r.Use(CORSMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(gin.Recovery())

	// Serve uploads statically
	r.Static("/uploads", "./uploads")

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	api := r.Group("/api")
	{
		// Auth endpoints
		api.POST("/auth/register", handlers.Register)
		api.POST("/auth/verify-otp", handlers.VerifyOTP)
		api.POST("/auth/login", handlers.Login)
		api.POST("/auth/forgot-password", handlers.ForgotPassword)
		api.POST("/auth/reset-password", handlers.ResetPassword)
		api.POST("/auth/refresh", handlers.RefreshToken)

		// OAuth endpoints (public — no JWT required)
		api.GET("/auth/oauth/config", handler.GetOAuthProviderConfig(oauthCfg))
		api.GET("/auth/oauth/google", handler.GoogleOAuthInitiate(oauthCfg))
		api.POST("/auth/oauth/google/callback", handler.GoogleOAuthCallback(oauthCfg, authService))

		// Public User details
		api.GET("/users", handlers.ListUsers)
		api.GET("/users/:id", handlers.GetUserByID)
		api.GET("/users/:id/followers", handlers.GetFollowers)
		api.GET("/users/:id/following", handlers.GetFollowing)

		// Public Community endpoints
		api.GET("/communities", handlers.ListCommunities)
		api.GET("/communities/memberships", handlers.GetCommunityMemberships)
		api.GET("/communities/:id", handlers.GetCommunityByID)
		api.GET("/communities/:id/members", handlers.GetCommunityMembers)

		// Public Post endpoints
		api.GET("/posts", handlers.ListPosts)
		api.GET("/posts/:id", handlers.GetPostByID)

		// Public Comment list
		api.GET("/comments", handlers.ListComments)

		// Search endpoint
		api.GET("/search", handlers.Search)

		// SECURE ROUTE GROUP
		secured := api.Group("")
		secured.Use(middleware.AuthMiddleware(authService))
		{
			// User/Auth actions
			secured.GET("/auth/me", handlers.GetMe)
			secured.PUT("/auth/me", handlers.UpdateMe)
			secured.POST("/auth/change-password", handlers.ChangePassword)
			secured.PUT("/users/:id", handlers.UpdateUser)

			// Follow operations
			secured.POST("/users/:id/follow", handlers.Follow)
			secured.POST("/users/:id/unfollow", handlers.Unfollow)
			secured.GET("/users/follow-requests", handlers.GetFollowRequests)
			secured.POST("/users/follow-requests/:follower_id/accept", handlers.AcceptFollowRequest)
			secured.POST("/users/follow-requests/:follower_id/reject", handlers.RejectFollowRequest)
			secured.GET("/users/:id/follow-status", handlers.GetFollowStatus)

			// Community actions
			secured.POST("/communities", handlers.CreateCommunity)
			secured.POST("/communities/:id/join", handlers.JoinCommunity)
			secured.POST("/communities/:id/leave", handlers.LeaveCommunity)
			secured.DELETE("/communities/:id", handlers.DeleteCommunity)

			// Post actions
			secured.GET("/posts/following", handlers.ListFollowingPosts)
			secured.POST("/posts", handlers.CreatePost)
			secured.PUT("/posts/:id", handlers.UpdatePost)
			secured.DELETE("/posts/:id", handlers.DeletePost)
			secured.POST("/posts/:id/vote", handlers.VotePost)
			secured.POST("/posts/:id/poll", handlers.VotePoll)
			secured.POST("/posts/:id/save", handlers.SavePost)
			secured.POST("/posts/:id/unsave", handlers.UnsavePost)
			secured.GET("/users/saved", handlers.GetSavedPosts)

			// Comment actions
			secured.POST("/comments", handlers.CreateComment)
			secured.PUT("/comments/:id", handlers.UpdateComment)
			secured.POST("/comments/:id/vote", handlers.VoteComment)
			secured.DELETE("/comments/:id", handlers.DeleteComment)

			// Chat actions
			secured.POST("/chats", handlers.CreateChatRoom)
			secured.GET("/chats", handlers.GetChatRooms)
			secured.PUT("/chats/:id", handlers.UpdateChatRoom)
			secured.DELETE("/chats/:id", handlers.DeleteChatRoom)
			secured.GET("/chats/:id/messages", handlers.GetMessages)
			secured.POST("/chats/:id/messages", handlers.SendMessage)
			secured.PUT("/messages/:id", handlers.UpdateMessage)
			secured.DELETE("/messages/:id", handlers.DeleteMessage)
			secured.POST("/upload", handlers.UploadFile)

			// Notification actions
			secured.GET("/notifications", handlers.GetNotifications)
			secured.GET("/notifications/unread-count", handlers.GetUnreadNotificationCount)
			secured.POST("/notifications/read", handlers.MarkAllNotificationsRead)
			secured.POST("/notifications/:id/read", handlers.MarkNotificationRead)
			secured.PUT("/notifications/:id", handlers.MarkNotificationRead)

			// Reports (any authenticated user)
			secured.POST("/reports", handlers.CreateReport)

			// Moderation actions (admin/moderator only, enforced inside service)
			secured.POST("/moderation/users/:id/ban", handlers.BanUser)
			secured.POST("/moderation/users/:id/unban", handlers.UnbanUser)
			secured.POST("/moderation/users/:id/shadow-ban", handlers.ShadowBanUser)
			secured.POST("/moderation/users/:id/unshadow-ban", handlers.UnshadowBanUser)
			secured.POST("/moderation/posts/:id/remove", handlers.RemovePost)
			secured.POST("/moderation/comments/:id/remove", handlers.RemoveComment)
			secured.GET("/moderation/logs", handlers.GetModerationLogs)
			secured.GET("/moderation/communities/:id/logs", handlers.GetCommunityModerationLogs)
			secured.GET("/moderation/reports", handlers.GetReports)
			secured.PUT("/moderation/reports/:id", handlers.UpdateReport)
			secured.GET("/moderation/filters", handlers.ListKeywordFilters)
			secured.POST("/moderation/filters", handlers.AddKeywordFilter)
			secured.DELETE("/moderation/filters/:id", handlers.RemoveKeywordFilter)

			// Analytics (admin only enforced by role check in middleware or service)
			secured.GET("/analytics/dashboard", handlers.GetAnalyticsDashboard)
		}
	}

	// WebSocket endpoint (authenticated via ?token= query param)
	r.GET("/api/ws/chat/:id", handler.ServeWS(wsHub, chatService, authService))
	r.GET("/api/ws/global", handler.ServeGlobalWS(wsHub, authService))

	// Public analytics event tracking (can also be called by anonymous users)
	r.POST("/api/analytics/track", handlers.TrackEvent)

	// 8. Start HTTP Server
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("server starting", "addr", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

// CORSMiddleware handles cross-origin requests
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func seedDemoData(db *gorm.DB) {
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)
	if userCount > 0 {
		return
	}

	log.Println("Seeding initial database demo records...")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	passStr := string(hashedPassword)

	// Users
	user1 := model.User{
		Username:       "amira",
		Email:          "amira@example.com",
		PasswordHash:   passStr,
		AvatarURL:      "",
		BannerURL:      "",
		Bio:            "Люблю аниме, интерфейсы и аккуратный фронтенд.",
		Role:           "admin",
		Level:          5,
		XP:             430,
		Title:          "Frontend Builder",
		ProfileTheme:   "sunset",
		FollowersCount: 12,
		FollowingCount: 5,
		AllowDMs:       true,
	}

	user2 := model.User{
		Username:       "kaizer",
		Email:          "kai@example.com",
		PasswordHash:   passStr,
		AvatarURL:      "",
		BannerURL:      "",
		Bio:            "Собираю фанатские сообщества и пишу посты.",
		Role:           "user",
		Level:          3,
		XP:             240,
		Title:          "Community Mod",
		ProfileTheme:   "ocean",
		FollowersCount: 3,
		FollowingCount: 8,
		AllowDMs:       true,
	}

	user3 := model.User{
		Username:       "moduser",
		Email:          "moderator@example.com",
		PasswordHash:   passStr,
		AvatarURL:      "",
		BannerURL:      "",
		Bio:            "Официальный модератор платформы Nexus Forum.",
		Role:           "moderator",
		Level:          4,
		XP:             320,
		Title:          "Platform Moderator",
		ProfileTheme:   "forest",
		FollowersCount: 8,
		FollowingCount: 4,
		AllowDMs:       true,
	}

	db.Create(&user1)
	db.Create(&user2)
	db.Create(&user3)

	// Communities
	comm1 := model.Community{
		Name:        "Nexus Anime",
		Slug:        "nexus-anime",
		Description: "Обсуждаем аниме, мангу и любимые фандомы.",
		Visibility:  "public",
		OwnerID:     user1.ID,
		MemberCount: 2,
		PostCount:   1,
		Rules:       `[{"title":"Уважение","description":"Без токсичности и оскорблений."}]`,
	}

	comm2 := model.Community{
		Name:        "UI Workshop",
		Slug:        "ui-workshop",
		Description: "Разбор интерфейсов, анимаций и продуктового дизайна.",
		Visibility:  "public",
		OwnerID:     user2.ID,
		MemberCount: 2,
		PostCount:   1,
		Rules:       `[{"title":"Конструктив","description":"Критикуем бережно и по делу."}]`,
	}

	comm3 := model.Community{
		Name:        "Roleplay Hub",
		Slug:        "roleplay-hub",
		Description: "Поиск игроков, сюжетов и вселенных для ролевых игр.",
		Visibility:  "public",
		OwnerID:     user1.ID,
		MemberCount: 1,
		PostCount:   1,
		Rules:       `[{"title":"18+","description":"Возрастные ограничения указываем явно."}]`,
	}

	db.Create(&comm1)
	db.Create(&comm2)
	db.Create(&comm3)

	// Members
	db.Create(&model.CommunityMember{UserID: user1.ID, CommunityID: comm1.ID, Role: "owner"})
	db.Create(&model.CommunityMember{UserID: user2.ID, CommunityID: comm1.ID, Role: "member"})
	db.Create(&model.CommunityMember{UserID: user2.ID, CommunityID: comm2.ID, Role: "owner"})
	db.Create(&model.CommunityMember{UserID: user1.ID, CommunityID: comm2.ID, Role: "member"})
	db.Create(&model.CommunityMember{UserID: user1.ID, CommunityID: comm3.ID, Role: "owner"})

	// Posts
	post1 := model.Post{
		CommunityID:  comm2.ID,
		AuthorID:     user1.ID,
		Title:        "Как вам новый дизайн ленты?",
		Content:      "Собрала первый рабочий вариант локального форума. Хочется понять, где интерфейс уже хорош, а где еще сырой.",
		Type:         "text",
		Score:        18,
		Upvotes:      18,
		CommentCount: 2,
		Status:       "published",
		Tags:         `["ui","feedback"]`,
		MediaUrls:    `[]`,
	}

	post2 := model.Post{
		CommunityID:  comm1.ID,
		AuthorID:     user2.ID,
		Title:        "Топ аниме-сообществ для новичков",
		Content:      "Сделала подборку дружелюбных тредов, где комфортно начинать общение и не бояться задавать вопросы.",
		Type:         "text",
		Score:        9,
		Upvotes:      10,
		Downvotes:    1,
		CommentCount: 1,
		Status:       "published",
		Tags:         `["anime","guide"]`,
		MediaUrls:    `[]`,
	}

	post3 := model.Post{
		CommunityID: comm3.ID,
		AuthorID:    user1.ID,
		Title:       "Ищу игроков для sci-fi RP",
		Content:     "Нужны 2-3 человека в мягкую сюжетную космооперу с упором на персонажей.",
		Type:        "text",
		Score:       6,
		Upvotes:     7,
		Downvotes:   1,
		Status:      "published",
		Tags:        `["rp","sci-fi"]`,
		MediaUrls:   `[]`,
	}

	db.Create(&post1)
	db.Create(&post2)
	db.Create(&post3)

	// Comments
	comment1 := model.Comment{
		PostID:    post1.ID,
		AuthorID:  user2.ID,
		Content:   "Мне нравится структура. Особенно хорошо сработала правая колонка с сообществами.",
		Score:     4,
		CreatedAt: time.Now().Add(-40 * time.Minute),
	}
	db.Create(&comment1)

	comment2 := model.Comment{
		PostID:    post1.ID,
		ParentID:  &comment1.ID,
		AuthorID:  user1.ID,
		Content:   "Спасибо, хочу еще доработать пустые состояния и onboarding.",
		Score:     2,
		CreatedAt: time.Now().Add(-20 * time.Minute),
	}
	db.Create(&comment2)

	comment3 := model.Comment{
		PostID:    post2.ID,
		AuthorID:  user1.ID,
		Content:   "Добавь еще раздел с рекомендациями по жанрам.",
		Score:     3,
		CreatedAt: time.Now().Add(-75 * time.Minute),
	}
	db.Create(&comment3)

	// Follow
	db.Create(&model.UserFollow{FollowerID: user2.ID, FollowingID: user1.ID})

	// Saved Posts
	db.Create(&model.SavedPost{UserID: user1.ID, PostID: post2.ID})

	// Notifications
	db.Create(&model.Notification{
		UserID: user1.ID,
		Type:   "reply",
		Title:  "Новый ответ",
		Body:   "Kai ответил на ваш комментарий.",
		IsRead: false,
	})
	db.Create(&model.Notification{
		UserID: user1.ID,
		Type:   "follow",
		Title:  "Новый подписчик",
		Body:   "Kai подписался на ваш профиль.",
		IsRead: true,
	})

	// Chat Room
	pBytes, _ := json.Marshal([]uint{user1.ID, user2.ID})
	room := model.ChatRoom{
		Name:         "Amira & Kai",
		Type:         "direct",
		Participants: string(pBytes),
		LastMessage:  "Сделала локальный REST-сервер на Go!",
	}
	db.Create(&room)

	// Messages
	db.Create(&model.Message{
		ChatRoomID:     room.ID,
		SenderID:       user2.ID,
		SenderUsername: user2.Username,
		Content:        "Какой следующий шаг по проекту?",
		IsRead:         true,
	})
	db.Create(&model.Message{
		ChatRoomID:     room.ID,
		SenderID:       user1.ID,
		SenderUsername: user1.Username,
		Content:        "Сделала локальный REST-сервер на Go!",
		IsRead:         false,
	})
}
