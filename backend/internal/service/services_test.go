package service_test

import (
	"log/slog"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
	"nexus-forum-backend/internal/service"
)

// setupDB creates an in-memory SQLite database with all tables migrated.
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	err = db.AutoMigrate(
		&model.User{},
		&model.UserFollow{},
		&model.Community{},
		&model.CommunityMember{},
		&model.Post{},
		&model.Comment{},
		&model.Vote{},
		&model.SavedPost{},
		&model.Notification{},
		&model.ChatRoom{},
		&model.Message{},
		&model.ModerationLog{},
		&model.AnalyticsEvent{},
	)
	if err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return db
}

func init() {
	// Suppress gorm output in tests
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth Service Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_RegisterAndVerifyOTP(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, "test-secret-1234")

	email := "alice@example.com"
	pass := "securepass"

	// Register
	if err := authSvc.Register(email, pass); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Duplicate registration must fail
	if err := authSvc.Register(email, pass); err == nil {
		t.Error("expected error for duplicate registration")
	}

	// Verify OTP
	token, user, err := authSvc.VerifyOTP(email, "123456")
	if err != nil {
		t.Fatalf("VerifyOTP failed: %v", err)
	}
	if user.Email != email {
		t.Errorf("expected email %s got %s", email, user.Email)
	}
	if token == "" {
		t.Error("expected non-empty JWT token")
	}
	if user.Level != 1 || user.XP != 0 {
		t.Errorf("new user should have level=1 xp=0, got level=%d xp=%d", user.Level, user.XP)
	}
}

func TestAuthService_Login(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, "test-secret-5678")

	email := "bob@example.com"
	pass := "hunter2"

	_ = authSvc.Register(email, pass)
	_, user, _ := authSvc.VerifyOTP(email, "123456")

	// Login success
	loginToken, loginUser, err := authSvc.Login(email, pass)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loginUser.ID != user.ID {
		t.Errorf("user ID mismatch: expected %d got %d", user.ID, loginUser.ID)
	}
	if loginToken == "" {
		t.Error("expected login token")
	}

	// Wrong password
	_, _, err = authSvc.Login(email, "wrongpass")
	if err == nil {
		t.Error("expected error for wrong password")
	}

	// Banned user
	user.IsBanned = true
	_ = userRepo.Update(user)
	_, _, err = authSvc.Login(email, pass)
	if err == nil {
		t.Error("expected error for banned user login")
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, "my-secret")

	_ = authSvc.Register("carol@example.com", "pass")
	token, _, _ := authSvc.VerifyOTP("carol@example.com", "123456")

	claims, err := authSvc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.Email != "carol@example.com" {
		t.Errorf("unexpected claim email: %s", claims.Email)
	}

	// Bad token
	_, err = authSvc.ValidateToken("not.a.real.token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// User + XP/Level Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestUserService_FollowAndXP(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	followRepo := repository.NewFollowRepository(db)
	authSvc := service.NewAuthService(userRepo, "xp-secret")
	userSvc := service.NewUserService(userRepo, followRepo)

	_ = authSvc.Register("dave@example.com", "pass")
	_, user1, _ := authSvc.VerifyOTP("dave@example.com", "123456")

	user2 := &model.User{Username: "eve", Email: "eve@example.com", PasswordHash: "hashed", ProfileTheme: "default"}
	_ = userRepo.Create(user2)

	// Follow
	if err := userSvc.Follow(user1.ID, user2.ID); err != nil {
		t.Fatalf("Follow failed: %v", err)
	}

	updated, _ := userRepo.GetByID(user1.ID)
	if updated.XP != 4 {
		t.Errorf("expected 4 XP after follow, got %d", updated.XP)
	}

	// Cannot follow twice
	if err := userSvc.Follow(user1.ID, user2.ID); err == nil {
		t.Error("expected error for double follow")
	}

	// Cannot follow self
	if err := userSvc.Follow(user1.ID, user1.ID); err == nil {
		t.Error("expected error for self-follow")
	}

	// Unfollow
	if err := userSvc.Unfollow(user1.ID, user2.ID); err != nil {
		t.Fatalf("Unfollow failed: %v", err)
	}
}

func TestUserService_LevelProgression(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	followRepo := repository.NewFollowRepository(db)
	authSvc := service.NewAuthService(userRepo, "lvl-secret")
	userSvc := service.NewUserService(userRepo, followRepo)

	_ = authSvc.Register("frank@example.com", "pass")
	_, user, _ := authSvc.VerifyOTP("frank@example.com", "123456")

	// Manually set XP to 150 (should be level 2 after next XP-granting action)
	user.XP = 146
	_ = userRepo.Update(user)

	target := &model.User{Username: "grace", Email: "grace@example.com", PasswordHash: "h", ProfileTheme: "default"}
	_ = userRepo.Create(target)

	// Follow gives +4 XP, putting total at 150 => level = (150/100)+1 = 2
	_ = userSvc.Follow(user.ID, target.ID)

	updated, _ := userRepo.GetByID(user.ID)
	if updated.XP != 150 {
		t.Errorf("expected XP=150, got %d", updated.XP)
	}
	if updated.Level != 2 {
		t.Errorf("expected Level=2, got %d", updated.Level)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Post Service Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestPostService_CreateAndVote(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	commRepo := repository.NewCommunityRepository(db)
	postRepo := repository.NewPostRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	savedRepo := repository.NewSavedPostRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	authSvc := service.NewAuthService(userRepo, "post-secret")
	postSvc := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)

	_ = authSvc.Register("hannah@example.com", "pass")
	_, author, _ := authSvc.VerifyOTP("hannah@example.com", "123456")

	comm := &model.Community{Name: "TestComm", Slug: "test-comm", OwnerID: author.ID, Visibility: "public"}
	_ = commRepo.Create(comm)

	post := &model.Post{
		CommunityID: comm.ID,
		AuthorID:    author.ID,
		Title:       "Test Post",
		Content:     "Hello world!",
		Type:        "text",
		Status:      "published",
		MediaUrls:   "[]",
		Tags:        "[]",
	}

	if err := postSvc.Create(post); err != nil {
		t.Fatalf("Post.Create failed: %v", err)
	}
	if post.ID == 0 {
		t.Error("expected post ID to be set")
	}

	// Author gets +20 XP from creating a post
	updatedAuthor, _ := userRepo.GetByID(author.ID)
	if updatedAuthor.XP != 20 {
		t.Errorf("expected XP=20 after creating post, got %d", updatedAuthor.XP)
	}

	// Upvote
	voter := &model.User{Username: "ivan", Email: "ivan@example.com", PasswordHash: "h", ProfileTheme: "default"}
	_ = userRepo.Create(voter)

	if err := postSvc.Vote(voter.ID, post.ID, 1); err != nil {
		t.Fatalf("Vote +1 failed: %v", err)
	}
	voted, _ := postRepo.GetByID(post.ID)
	if voted.Upvotes != 1 || voted.Score != 1 {
		t.Errorf("expected upvotes=1 score=1, got upvotes=%d score=%d", voted.Upvotes, voted.Score)
	}

	// Cancel vote (vote same value again)
	_ = postSvc.Vote(voter.ID, post.ID, 1)
	afterCancel, _ := postRepo.GetByID(post.ID)
	if afterCancel.Upvotes != 0 || afterCancel.Score != 0 {
		t.Errorf("expected upvotes=0 score=0 after cancel, got upvotes=%d score=%d", afterCancel.Upvotes, afterCancel.Score)
	}
}

func TestPostService_SaveAndUnsave(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	commRepo := repository.NewCommunityRepository(db)
	postRepo := repository.NewPostRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	savedRepo := repository.NewSavedPostRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	authSvc := service.NewAuthService(userRepo, "save-secret")
	postSvc := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)

	_ = authSvc.Register("julia@example.com", "pass")
	_, user, _ := authSvc.VerifyOTP("julia@example.com", "123456")

	comm := &model.Community{Name: "SaveComm", Slug: "save-comm", OwnerID: user.ID, Visibility: "public"}
	_ = commRepo.Create(comm)
	post := &model.Post{CommunityID: comm.ID, AuthorID: user.ID, Title: "Saveable", Type: "text", Status: "published", MediaUrls: "[]", Tags: "[]"}
	_ = postSvc.Create(post)

	if err := postSvc.SavePost(user.ID, post.ID); err != nil {
		t.Fatalf("SavePost failed: %v", err)
	}
	// Save twice should fail
	if err := postSvc.SavePost(user.ID, post.ID); err == nil {
		t.Error("expected error for double save")
	}

	saved, _ := postSvc.GetSavedByUser(user.ID)
	if len(saved) != 1 {
		t.Errorf("expected 1 saved post, got %d", len(saved))
	}

	if err := postSvc.UnsavePost(user.ID, post.ID); err != nil {
		t.Fatalf("UnsavePost failed: %v", err)
	}
	saved2, _ := postSvc.GetSavedByUser(user.ID)
	if len(saved2) != 0 {
		t.Errorf("expected 0 saved posts after unsave, got %d", len(saved2))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Comment Service Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestCommentService_CreateAndVote(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	commRepo := repository.NewCommunityRepository(db)
	postRepo := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	savedRepo := repository.NewSavedPostRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	authSvc := service.NewAuthService(userRepo, "comment-secret")
	postSvc := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)
	commentSvc := service.NewCommentService(commentRepo, userRepo, postRepo, voteRepo, notifRepo)

	_ = authSvc.Register("kate@example.com", "pass")
	_, author, _ := authSvc.VerifyOTP("kate@example.com", "123456")
	commenter := &model.User{Username: "leo", Email: "leo@example.com", PasswordHash: "h", ProfileTheme: "default"}
	_ = userRepo.Create(commenter)

	comm := &model.Community{Name: "CmtComm", Slug: "cmt-comm", OwnerID: author.ID, Visibility: "public"}
	_ = commRepo.Create(comm)
	post := &model.Post{CommunityID: comm.ID, AuthorID: author.ID, Title: "T", Type: "text", Status: "published", MediaUrls: "[]", Tags: "[]"}
	_ = postSvc.Create(post)

	comment := &model.Comment{PostID: post.ID, AuthorID: commenter.ID, Content: "Nice post!"}
	if err := commentSvc.Create(comment); err != nil {
		t.Fatalf("Comment.Create failed: %v", err)
	}
	if comment.ID == 0 {
		t.Error("expected comment ID to be set")
	}

	// Commenter gets +8 XP
	updatedCommenter, _ := userRepo.GetByID(commenter.ID)
	if updatedCommenter.XP != 8 {
		t.Errorf("expected XP=8 after comment, got %d", updatedCommenter.XP)
	}

	// Vote on comment
	voter := &model.User{Username: "mia", Email: "mia@example.com", PasswordHash: "h", ProfileTheme: "default"}
	_ = userRepo.Create(voter)
	if err := commentSvc.Vote(voter.ID, comment.ID, 1); err != nil {
		t.Fatalf("Comment.Vote failed: %v", err)
	}
	voted, _ := commentRepo.GetByID(comment.ID)
	if voted.Score != 1 {
		t.Errorf("expected comment score=1, got %d", voted.Score)
	}

	// Unauthorized delete
	if err := commentSvc.Delete(author.ID, comment.ID); err == nil {
		t.Error("expected error when non-author deletes comment")
	}
	// Authorized delete
	if err := commentSvc.Delete(commenter.ID, comment.ID); err != nil {
		t.Fatalf("Comment.Delete failed: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Moderation Service Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestModerationService_BanAndUnban(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	commRepo := repository.NewCommunityRepository(db)
	postRepo := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	savedRepo := repository.NewSavedPostRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	modRepo := repository.NewModerationRepository(db)

	authSvc := service.NewAuthService(userRepo, "mod-secret")
	postSvc := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)
	modSvc := service.NewModerationService(modRepo, userRepo, postRepo, commentRepo)

	// Create admin
	_ = authSvc.Register("admin@example.com", "pass")
	_, admin, _ := authSvc.VerifyOTP("admin@example.com", "123456")
	admin.Role = "admin"
	_ = userRepo.Update(admin)

	// Create regular user
	target := &model.User{Username: "spammer", Email: "spam@example.com", PasswordHash: "h", ProfileTheme: "default"}
	_ = userRepo.Create(target)

	// Non-mod cannot ban
	regularUser := &model.User{Username: "reg", Email: "reg@example.com", PasswordHash: "h", ProfileTheme: "default"}
	_ = userRepo.Create(regularUser)
	if err := modSvc.BanUser(regularUser.ID, target.ID, "test"); err == nil {
		t.Error("expected error: non-mod cannot ban")
	}

	// Admin can ban
	if err := modSvc.BanUser(admin.ID, target.ID, "spam"); err != nil {
		t.Fatalf("BanUser failed: %v", err)
	}
	banned, _ := userRepo.GetByID(target.ID)
	if !banned.IsBanned {
		t.Error("expected user to be banned")
	}

	// Ban already-banned user
	if err := modSvc.BanUser(admin.ID, target.ID, "again"); err == nil {
		t.Error("expected error: user already banned")
	}

	// Unban
	if err := modSvc.UnbanUser(admin.ID, target.ID, "pardoned"); err != nil {
		t.Fatalf("UnbanUser failed: %v", err)
	}
	unbanned, _ := userRepo.GetByID(target.ID)
	if unbanned.IsBanned {
		t.Error("expected user to be unbanned")
	}

	// Check moderation logs
	logs, err := modSvc.GetLogs(10)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 mod log entries, got %d", len(logs))
	}

	// Test post removal
	comm := &model.Community{Name: "ModComm", Slug: "mod-comm", OwnerID: admin.ID, Visibility: "public"}
	_ = commRepo.Create(comm)
	post := &model.Post{CommunityID: comm.ID, AuthorID: target.ID, Title: "Bad post", Type: "text", Status: "published", MediaUrls: "[]", Tags: "[]"}
	_ = postSvc.Create(post)

	if err := modSvc.RemovePost(admin.ID, post.ID, "rule violation"); err != nil {
		t.Fatalf("RemovePost failed: %v", err)
	}
	removedPost, _ := postRepo.GetByID(post.ID)
	if removedPost.Status != "removed" {
		t.Errorf("expected post status=removed, got %s", removedPost.Status)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Analytics Service Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAnalyticsService_TrackAndDashboard(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	postRepo := repository.NewPostRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)

	analyticsSvc := service.NewAnalyticsService(analyticsRepo, userRepo, postRepo)

	// Track events
	userID := uint(1)
	_ = analyticsSvc.Track(&userID, "page_view", "post", nil, "")
	_ = analyticsSvc.Track(&userID, "page_view", "post", nil, "")
	_ = analyticsSvc.Track(nil, "page_view", "community", nil, "")

	// GetDashboard should run without error
	dashboard, err := analyticsSvc.GetDashboard()
	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}
	if dashboard == nil {
		t.Error("expected non-nil dashboard")
	}

	// Verify tracked page_view count
	count, _ := analyticsRepo.CountEvents("page_view", 0, 0)
	if count != 3 {
		t.Errorf("expected 3 page_view events, got %d", count)
	}
}
