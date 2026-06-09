package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"

	"nexus-forum-backend/internal/cache"
	"nexus-forum-backend/internal/config"
	"nexus-forum-backend/internal/database"
	"nexus-forum-backend/internal/demo"
	"nexus-forum-backend/internal/email"
	"nexus-forum-backend/internal/handler"
	"nexus-forum-backend/internal/middleware"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/queue"
	"nexus-forum-backend/internal/repository"
	"nexus-forum-backend/internal/search"
	"nexus-forum-backend/internal/service"
	"nexus-forum-backend/internal/storage"
	"nexus-forum-backend/internal/telemetry"
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

	ctx := context.Background()
	otelShutdown, err := telemetry.Init(ctx, cfg.OTELService, cfg.OTELEndpoint)
	if err != nil {
		log.Fatalf("failed to init opentelemetry: %v", err)
	}
	defer func() {
		_ = otelShutdown(context.Background())
	}()

	// 3. Connect to Database (PostgreSQL primary; SQLite optional dev fallback)
	var db *gorm.DB
	if cfg.DBType == "sqlite" {
		if _, statErr := os.Stat(cfg.SqliteDB); os.IsNotExist(statErr) {
			logger.Info("sqlite database file not found; a new database will be created on first migration", "path", cfg.SqliteDB)
		}
		db, err = gorm.Open(sqlite.Open(cfg.SqliteDB), &gorm.Config{})
	} else {
		db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	}
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := db.Use(tracing.NewPlugin()); err != nil {
		logger.Warn("gorm opentelemetry plugin not enabled", "error", err)
	} else {
		logger.Info("gorm opentelemetry tracing enabled")
	}
	logger.Info("database connected", "db_type", cfg.DBType)

	redisClient := cache.New(cfg.RedisURL)
	mqPublisher := queue.NewPublisher(cfg.RabbitMQURL)

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
		&model.EmailVerification{},
		&model.PostSearchToken{},
		&model.FeatureFlag{},
		&model.SearchQuery{},
		&model.PushSubscription{},
	)
	if err != nil {
		log.Fatalf("failed to auto migrate tables: %v", err)
	}

	// Data fix for existing follows
	db.Exec("UPDATE user_follows SET status = 'accepted' WHERE status = '' OR status IS NULL")
	db.Exec("UPDATE users SET email_verified = true WHERE email_verified = false")

	if search.PostgresFTSEnabled(db) {
		if err := search.InitPostgresSearch(db); err != nil {
			logger.Warn("postgres fts init failed", "error", err)
		} else {
			logger.Info("postgres full-text search enabled")
		}
	} else if err := search.Init(db); err != nil {
		logger.Warn("search index unavailable; falling back to LIKE search", "error", err)
	} else if search.FTSEnabled() {
		_ = search.ReindexAll(db)
	}
	if !search.PostgresFTSEnabled(db) {
		if err := search.ReindexPublishedPosts(db); err != nil {
			logger.Warn("post search reindex failed", "error", err)
		}
	}
	seedFeatureFlags(db)

	logger.Info("database auto-migrations complete")
	database.PurgeLegacyBase64Media(db)

	// 5. Seed minimal demo data if empty
	seedDemoDataIfEmpty(db)

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
	verifyRepo := repository.NewEmailVerificationRepository(db)
	karmaRepo := repository.NewKarmaRepository(db)
	flagRepo := repository.NewFeatureFlagRepository(db)
	searchQueryRepo := repository.NewSearchQueryRepository(db)
	pushSubRepo := repository.NewPushSubscriptionRepository(db)
	mailer := email.NewMailer(email.Config{
		Host:        cfg.SMTPHost,
		Port:        cfg.SMTPPort,
		Username:    cfg.SMTPUsername,
		Password:    cfg.SMTPPassword,
		From:        cfg.SMTPFrom,
		FrontendURL: cfg.FrontendURL,
	})

	authService := service.NewAuthService(userRepo, modRepo, resetRepo, refreshRepo, verifyRepo, mailer, cfg.JWTSecret, cfg.FrontendURL)
	userService := service.NewUserService(userRepo, followRepo, notifRepo, modRepo, karmaRepo)
	commService := service.NewCommunityService(commRepo, userRepo)
	postService := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)
	commentService := service.NewCommentService(commentRepo, userRepo, postRepo, voteRepo, notifRepo, commRepo)
	chatService := service.NewChatService(chatRepo, userRepo)
	notifService := service.NewNotificationService(notifRepo)
	modService := service.NewModerationService(modRepo, userRepo, postRepo, commentRepo, commRepo, notifRepo, keywordFilterRepo)
	analyticsService := service.NewAnalyticsService(analyticsRepo, userRepo, postRepo)
	flagService := service.NewFeatureFlagService(flagRepo)
	pushService := service.NewPushService(pushSubRepo, cfg.VAPIDPublic, cfg.VAPIDPrivate, cfg.VAPIDSubject)

	objectStore, err := storage.NewObjectStore(cfg)
	if err != nil {
		log.Fatalf("failed to initialize object storage: %v", err)
	}
	uploadService := service.NewUploadService(objectStore)
	logger.Info("upload service ready", "backend", uploadService.Backend())

	// WebSocket hub (starts background goroutine)
	wsHub := handler.NewWSHub(db)

	// Register global NotificationDispatcher to trigger real-time WS notifications
	repository.NotificationDispatcher = func(userID uint, notif *model.Notification) {
		payload, err := json.Marshal(struct {
			Type string              `json:"type"`
			Data *model.Notification `json:"data"`
		}{
			Type: "notification",
			Data: notif,
		})
		if err == nil {
			wsHub.SendToUser(userID, payload)
		}

		if recipient, err := userRepo.GetByID(userID); err == nil {
			karmaRepo.HydrateUser(recipient)
			email.MaybeNotifyForNotification(mailer, recipient, notif)
			_ = pushService.SendToUser(recipient, notif.Type, notif.Title, notif.Body)
		}
		mqPublisher.PublishNotification(queue.NotificationEvent{
			UserID: userID,
			Type:   notif.Type,
			Data:   map[string]interface{}{"title": notif.Title, "body": notif.Body},
		})

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

	handlers := handler.NewHandlers(authService, userService, commService, postService, commentService, chatService, notifService, modService, analyticsService, uploadService, pushService, flagService, searchQueryRepo, karmaRepo, wsHub, cfg.TurnstileSecret, mailer.Enabled(), cfg.FrontendURL)
	_ = redisClient

	// OAuth handler config
	oauthCfg := handler.OAuthConfig{
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		GithubClientID:     cfg.GithubClientID,
		GithubClientSecret: cfg.GithubClientSecret,
		FrontendURL:        cfg.FrontendURL,
		TurnstileSiteKey:   cfg.TurnstileSiteKey,
	}

	// 7. Setup Gin Router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Apply Middlewares
	r.Use(otelgin.Middleware(cfg.OTELService))
	r.Use(CORSMiddleware())
	r.Use(middleware.PrometheusMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(gin.Recovery())

	// Serve local uploads statically (fallback when MinIO is not used)
	r.Static("/uploads", cfg.LocalUploadDir)

	r.GET("/health", handler.Health(db))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	authRateLimit := middleware.NewRateLimiter(30, time.Minute, redisClient)
	postRateLimit := middleware.NewRateLimiter(15, time.Minute, redisClient)
	commentRateLimit := middleware.NewRateLimiter(40, time.Minute, redisClient)

	api := r.Group("/api")
	{
		// Auth endpoints
		api.POST("/auth/register", authRateLimit, handlers.Register)
		api.POST("/auth/verify-otp", authRateLimit, handlers.VerifyOTP)
		api.GET("/auth/confirm-email", handlers.ConfirmEmail)
		api.POST("/auth/confirm-email", authRateLimit, handlers.ConfirmEmail)
		api.POST("/auth/resend-otp", authRateLimit, handlers.ResendOTP)
		api.POST("/auth/login", authRateLimit, handlers.Login)
		api.POST("/auth/forgot-password", authRateLimit, handlers.ForgotPassword)
		api.POST("/auth/reset-password", authRateLimit, handlers.ResetPassword)
		api.POST("/auth/refresh", authRateLimit, handlers.RefreshToken)
		api.POST("/auth/logout", handlers.Logout)

		// OAuth endpoints (public — no JWT required)
		api.GET("/auth/oauth/config", handler.GetOAuthProviderConfig(oauthCfg))
		api.GET("/auth/oauth/google", handler.GoogleOAuthInitiate(oauthCfg))
		api.POST("/auth/oauth/google/callback", handler.GoogleOAuthCallback(oauthCfg, authService))
		api.GET("/auth/oauth/github", handler.GitHubOAuthInitiate(oauthCfg))
		api.POST("/auth/oauth/github/callback", handler.GitHubOAuthCallback(oauthCfg, authService))

		// User details (optional JWT for moderator/admin moderation fields)
		usersPublic := api.Group("")
		usersPublic.Use(middleware.OptionalAuthMiddleware(authService))
		usersPublic.GET("/users", handlers.ListUsers)
		usersPublic.GET("/users/:id", handlers.GetUserByID)
		usersPublic.GET("/users/:id/stats", handlers.GetUserProfileStats)
		usersPublic.GET("/users/:id/achievements", handlers.GetUserAchievements)
		usersPublic.GET("/users/:id/followers", handlers.GetFollowers)
		usersPublic.GET("/users/:id/following", handlers.GetFollowing)

		// Public Community endpoints
		api.GET("/communities", handlers.ListCommunities)
		api.GET("/communities/memberships", handlers.GetCommunityMemberships)
		api.GET("/communities/:id", handlers.GetCommunityByID)
		api.GET("/communities/:id/members", handlers.GetCommunityMembers)

		// Posts, search (optional JWT for privacy/shadow/draft visibility)
		publicRead := api.Group("")
		publicRead.Use(middleware.OptionalAuthMiddleware(authService))
		publicRead.GET("/posts", handlers.ListPosts)
		publicRead.GET("/posts/:id", handlers.GetPostByID)
		publicRead.GET("/search", handlers.Search)
		api.GET("/search/trending", handlers.TrendingSearches)
		api.GET("/feature-flags", handlers.GetPublicFeatureFlags)

		// Public Comment list
		api.GET("/comments", handlers.ListComments)

		// SECURE ROUTE GROUP
		secured := api.Group("")
		secured.Use(middleware.AuthMiddleware(authService))
		{
			// User/Auth actions
			secured.GET("/auth/me", handlers.GetMe)
			secured.PUT("/auth/me", handlers.UpdateMe)
			secured.GET("/auth/sessions", handlers.ListSessions)
			secured.DELETE("/auth/sessions/:id", handlers.RevokeSession)
			secured.POST("/auth/sessions/revoke-others", handlers.RevokeOtherSessions)
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
			secured.GET("/communities/:id/moderators", handlers.ListCommunityModerators)
			secured.POST("/communities/:id/moderators/:userId/promote", handlers.PromoteCommunityModerator)
			secured.POST("/communities/:id/moderators/:userId/demote", handlers.DemoteCommunityModerator)

			// Post actions
			secured.GET("/posts/following", handlers.ListFollowingPosts)
			secured.GET("/posts/following-communities", handlers.ListFollowingCommunityPosts)
			secured.POST("/posts", postRateLimit, handlers.CreatePost)
			secured.POST("/posts/crosspost", postRateLimit, handlers.CreateCrosspost)
			secured.PUT("/posts/:id", handlers.UpdatePost)
			secured.DELETE("/posts/:id", handlers.DeletePost)
			secured.POST("/posts/:id/vote", handlers.VotePost)
			secured.POST("/posts/:id/poll", handlers.VotePoll)
			secured.POST("/posts/:id/save", handlers.SavePost)
			secured.POST("/posts/:id/unsave", handlers.UnsavePost)
			secured.GET("/users/saved", handlers.GetSavedPosts)

			// Comment actions
			secured.POST("/comments", commentRateLimit, handlers.CreateComment)
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
			secured.GET("/push/vapid-public-key", handlers.GetPushPublicKey)
			secured.POST("/push/subscribe", handlers.SubscribePush)
			secured.POST("/push/unsubscribe", handlers.UnsubscribePush)

			// Reports (any authenticated user)
			secured.POST("/reports", handlers.CreateReport)

			// Moderation (admin + moderator) — route-level RBAC + service checks
			mod := secured.Group("/moderation")
			mod.Use(middleware.RequireAdminOrMod())
			{
				mod.POST("/users/:id/ban", handlers.BanUser)
				mod.POST("/users/:id/unban", handlers.UnbanUser)
				mod.POST("/users/:id/shadow-ban", handlers.ShadowBanUser)
				mod.POST("/users/:id/unshadow-ban", handlers.UnshadowBanUser)
				mod.POST("/posts/:id/remove", handlers.RemovePost)
				mod.POST("/comments/:id/remove", handlers.RemoveComment)
				mod.GET("/logs", handlers.GetModerationLogs)
				mod.GET("/communities/:id/logs", handlers.GetCommunityModerationLogs)
				mod.GET("/reports", handlers.GetReports)
				mod.PUT("/reports/:id", handlers.UpdateReport)
				mod.GET("/filters", handlers.ListKeywordFilters)
				mod.POST("/filters", handlers.AddKeywordFilter)
				mod.DELETE("/filters/:id", handlers.RemoveKeywordFilter)
			}

			// Admin-only routes
			secured.GET("/analytics/dashboard", middleware.RequireAdmin(), handlers.GetAnalyticsDashboard)
			secured.GET("/analytics/activity", middleware.RequireAdmin(), handlers.GetAnalyticsActivity)
			secured.GET("/analytics/reports", middleware.RequireAdmin(), handlers.GetAnalyticsReports)
			secured.GET("/analytics/retention", middleware.RequireAdmin(), handlers.GetAnalyticsRetention)
			secured.GET("/analytics/engagement", middleware.RequireAdmin(), handlers.GetAnalyticsEngagement)
			secured.GET("/admin/feature-flags", middleware.RequireAdmin(), handlers.ListFeatureFlags)
			secured.PUT("/admin/feature-flags/:key", middleware.RequireAdmin(), handlers.UpdateFeatureFlag)
			if os.Getenv("ENABLE_BREAKER_DEBUG") == "true" {
				secured.POST("/admin/circuit-breaker/:name/probe", middleware.RequireAdmin(), handlers.ProbeCircuitBreaker)
			}
		}
	}

	// WebSocket endpoint (authenticated via ?token= query param)
	r.GET("/api/ws/chat/:id", handler.ServeWS(wsHub, chatService, authService))
	r.GET("/api/ws/global", handler.ServeGlobalWS(wsHub, authService))
	r.GET("/api/ws/post/:id", handler.ServePostWS(wsHub, authService))

	// Public analytics event tracking (can also be called by anonymous users)
	r.POST("/api/analytics/track", handlers.TrackEvent)

	// Background scheduler for due scheduled posts
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			published, err := postService.PublishDueScheduled()
			if err != nil {
				logger.Warn("scheduled publish failed", "error", err)
				continue
			}
			if len(published) > 0 {
				logger.Info("published scheduled posts", "count", len(published))
			}
			for _, p := range published {
				notif := &model.Notification{
					UserID: p.AuthorID,
					Type:   "scheduled_published",
					Title:  "Отложенный пост опубликован",
					Body:   p.Title,
				}
				_ = notifRepo.Create(notif)
			}
		}
	}()

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

func seedFeatureFlags(db *gorm.DB) {
	flags := []model.FeatureFlag{
		{Key: "crosspost", Enabled: true, Description: "Allow crossposting"},
		{Key: "web_push", Enabled: true, Description: "Browser push notifications"},
		{Key: "live_ws", Enabled: true, Description: "Live comment/vote WebSocket"},
	}
	for _, f := range flags {
		var existing model.FeatureFlag
		if err := db.Where("key = ?", f.Key).First(&existing).Error; err != nil {
			_ = db.Create(&f)
		}
	}
}

func seedDemoDataIfEmpty(db *gorm.DB) {
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)
	if userCount > 0 {
		return
	}
	log.Println("Seeding minimal demo database...")
	if err := demo.ResetMinimalDemo(db); err != nil {
		log.Printf("demo seed failed: %v", err)
	}
}
