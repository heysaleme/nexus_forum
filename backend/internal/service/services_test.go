package service_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"nexus-forum-backend/internal/email"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
	"nexus-forum-backend/internal/search"
	"nexus-forum-backend/internal/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
		&model.Report{},
		&model.PasswordResetToken{},
		&model.RefreshToken{},
		&model.EmailVerification{},
		&model.PostSearchToken{},
	)
	if err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	if err := search.Init(db); err != nil {
		t.Fatalf("failed to init search: %v", err)
	}
	return db
}

func init() {
	// Suppress gorm output in tests
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
}

func newTestAuthService(db *gorm.DB, secret string) service.AuthService {
	return service.NewAuthService(
		repository.NewUserRepository(db),
		repository.NewModerationRepository(db),
		repository.NewPasswordResetRepository(db),
		repository.NewRefreshTokenRepository(db),
		repository.NewEmailVerificationRepository(db),
		email.NewMailer(email.Config{}),
		secret,
		"http://localhost:5173",
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth Service Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_RegisterAndVerifyOTP(t *testing.T) {
	db := setupDB(t)
	authSvc := newTestAuthService(db, "test-secret-1234")

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
	token, refresh, user, err := authSvc.VerifyOTP(email, "123456")
	if err != nil {
		t.Fatalf("VerifyOTP failed: %v", err)
	}
	if user.Email != email {
		t.Errorf("expected email %s got %s", email, user.Email)
	}
	if token == "" {
		t.Error("expected non-empty JWT token")
	}
	if refresh == "" {
		t.Error("expected non-empty refresh token")
	}
	if user.Level != 1 || user.XP != 0 {
		t.Errorf("new user should have level=1 xp=0, got level=%d xp=%d", user.Level, user.XP)
	}
}

func TestAuthService_Login(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	authSvc := newTestAuthService(db, "test-secret-5678")

	email := "bob@example.com"
	pass := "hunter2"

	_ = authSvc.Register(email, pass)
	_, _, user, _ := authSvc.VerifyOTP(email, "123456")

	// Login success
	loginToken, _, loginUser, err := authSvc.Login(email, pass)
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
	_, _, _, err = authSvc.Login(email, "wrongpass")
	if err == nil {
		t.Error("expected error for wrong password")
	}

	// Banned user
	user.IsBanned = true
	_ = userRepo.Update(user)
	_, _, _, err = authSvc.Login(email, pass)
	if err == nil {
		t.Error("expected error for banned user login")
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	db := setupDB(t)
	authSvc := newTestAuthService(db, "refresh-secret")

	_ = authSvc.Register("refresh@example.com", "pass1234")
	_, refreshToken, _, _ := authSvc.VerifyOTP("refresh@example.com", "123456")

	newAccess, newRefresh, err := authSvc.RefreshAccessToken(refreshToken)
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Fatal("expected new access and refresh tokens")
	}

	_, _, err = authSvc.RefreshAccessToken(refreshToken)
	if err == nil {
		t.Error("expected error when reusing revoked refresh token")
	}
}

func TestAuthService_PasswordReset(t *testing.T) {
	db := setupDB(t)
	authSvc := newTestAuthService(db, "reset-secret")

	email := "reset@example.com"
	pass := "oldpass123"
	newPass := "newpass456"

	_ = authSvc.Register(email, pass)
	_, _, _, _ = authSvc.VerifyOTP(email, "123456")

	token, err := authSvc.RequestPasswordReset(email)
	if err != nil {
		t.Fatalf("RequestPasswordReset failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected reset token")
	}

	if err := authSvc.ResetPassword(token, newPass); err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}

	_, _, _, err = authSvc.Login(email, pass)
	if err == nil {
		t.Error("expected login to fail with old password")
	}

	_, _, _, err = authSvc.Login(email, newPass)
	if err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}

	if err := authSvc.ResetPassword(token, "anotherpass1"); err == nil {
		t.Error("expected error reusing reset token")
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	db := setupDB(t)
	authSvc := newTestAuthService(db, "my-secret")

	_ = authSvc.Register("carol@example.com", "pass")
	token, _, _, _ := authSvc.VerifyOTP("carol@example.com", "123456")

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
	notifRepo := repository.NewNotificationRepository(db)
	modRepo := repository.NewModerationRepository(db)
	authSvc := newTestAuthService(db, "xp-secret")
	userSvc := service.NewUserService(userRepo, followRepo, notifRepo, modRepo)

	_ = authSvc.Register("dave@example.com", "pass")
	_, _, user1, _ := authSvc.VerifyOTP("dave@example.com", "123456")

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
	notifRepo := repository.NewNotificationRepository(db)
	modRepo := repository.NewModerationRepository(db)
	authSvc := newTestAuthService(db, "lvl-secret")
	userSvc := service.NewUserService(userRepo, followRepo, notifRepo, modRepo)

	_ = authSvc.Register("frank@example.com", "pass")
	_, _, user, _ := authSvc.VerifyOTP("frank@example.com", "123456")

	// Manually set XP to 146 (should be level 2 after next XP-granting action)
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
	authSvc := newTestAuthService(db, "post-secret")
	postSvc := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)

	_ = authSvc.Register("hannah@example.com", "pass")
	_, _, author, _ := authSvc.VerifyOTP("hannah@example.com", "123456")

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
	authSvc := newTestAuthService(db, "save-secret")
	postSvc := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)

	_ = authSvc.Register("julia@example.com", "pass")
	_, _, user, _ := authSvc.VerifyOTP("julia@example.com", "123456")

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
	authSvc := newTestAuthService(db, "comment-secret")
	postSvc := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)
	commentSvc := service.NewCommentService(commentRepo, userRepo, postRepo, voteRepo, notifRepo, commRepo)

	_ = authSvc.Register("kate@example.com", "pass")
	_, _, author, _ := authSvc.VerifyOTP("kate@example.com", "123456")
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

	// Unauthorized delete (voter is unrelated to the community)
	if err := commentSvc.Delete(voter.ID, comment.ID); err == nil {
		t.Error("expected error when unrelated user deletes comment")
	}
	// Authorized delete (author is community owner so can delete)
	if err := commentSvc.Delete(author.ID, comment.ID); err != nil {
		t.Fatalf("Comment.Delete by owner failed: %v", err)
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
	keywordFilterRepo := repository.NewKeywordFilterRepository(db)

	authSvc := newTestAuthService(db, "mod-secret")
	postSvc := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, savedRepo, notifRepo)
	modSvc := service.NewModerationService(modRepo, userRepo, postRepo, commentRepo, commRepo, notifRepo, keywordFilterRepo)

	// Create admin
	_ = authSvc.Register("admin@example.com", "pass")
	_, _, admin, _ := authSvc.VerifyOTP("admin@example.com", "123456")
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

	// Non-mod cannot view logs, reports, or filters
	if _, err := modSvc.GetLogs(regularUser.ID, 10); err == nil {
		t.Error("expected error: non-mod cannot view moderation logs")
	}
	if _, err := modSvc.GetReports(regularUser.ID); err == nil {
		t.Error("expected error: non-mod cannot view reports")
	}
	if _, err := modSvc.ListKeywordFilters(regularUser.ID); err == nil {
		t.Error("expected error: non-mod cannot list keyword filters")
	}
	if err := modSvc.AddKeywordFilter(regularUser.ID, "badword", false, "block"); err == nil {
		t.Error("expected error: non-mod cannot add keyword filters")
	}

	// Check moderation logs
	logs, err := modSvc.GetLogs(admin.ID, 10)
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

	commLogs, err := modSvc.GetLogsByCommunity(admin.ID, comm.ID, 10)
	if err != nil {
		t.Fatalf("GetLogsByCommunity failed: %v", err)
	}
	if len(commLogs) != 1 {
		t.Fatalf("expected 1 community log, got %d", len(commLogs))
	}
	if commLogs[0].CommunityID == nil || *commLogs[0].CommunityID != comm.ID {
		t.Errorf("expected community_id=%d on log, got %v", comm.ID, commLogs[0].CommunityID)
	}
	if commLogs[0].TargetType != "post" || commLogs[0].TargetID != post.ID {
		t.Errorf("unexpected community log target: %s %d", commLogs[0].TargetType, commLogs[0].TargetID)
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
	userID2 := uint(2)
	_ = analyticsSvc.Track(&userID, "page_view", "post", nil, "")
	_ = analyticsSvc.Track(&userID, "page_view", "post", nil, "")
	_ = analyticsSvc.Track(&userID2, "login", "user", nil, "")
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

	dau, ok := dashboard["dau"].(int64)
	if !ok {
		t.Fatalf("expected dau int64 in dashboard, got %T", dashboard["dau"])
	}
	if dau != 2 {
		t.Errorf("expected dau=2, got %d", dau)
	}
	mau, ok := dashboard["mau"].(int64)
	if !ok {
		t.Fatalf("expected mau int64 in dashboard, got %T", dashboard["mau"])
	}
	if mau != 2 {
		t.Errorf("expected mau=2, got %d", mau)
	}

	retention, err := analyticsSvc.GetRetention()
	if err != nil {
		t.Fatalf("GetRetention failed: %v", err)
	}
	if _, ok := retention["d1"]; !ok {
		t.Error("expected d1 retention key")
	}

	engagement, err := analyticsSvc.GetEngagement()
	if err != nil {
		t.Fatalf("GetEngagement failed: %v", err)
	}
	if engagement["total_posts"] == nil {
		t.Error("expected total_posts in engagement stats")
	}
}

func TestUserService_ProfileStats(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	followRepo := repository.NewFollowRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	modRepo := repository.NewModerationRepository(db)
	userSvc := service.NewUserService(userRepo, followRepo, notifRepo, modRepo)
	commRepo := repository.NewCommunityRepository(db)
	postRepo := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)

	u1 := &model.User{Username: "stats1", Email: "s1@example.com", PasswordHash: "h", ProfileTheme: "default"}
	u2 := &model.User{Username: "stats2", Email: "s2@example.com", PasswordHash: "h", ProfileTheme: "default"}
	_ = userRepo.Create(u1)
	_ = userRepo.Create(u2)
	comm := &model.Community{Name: "Stats", Slug: "stats", OwnerID: u1.ID, Visibility: "public"}
	_ = commRepo.Create(comm)
	_ = commRepo.AddMember(&model.CommunityMember{UserID: u1.ID, CommunityID: comm.ID, Role: "owner"})
	post := &model.Post{CommunityID: comm.ID, AuthorID: u1.ID, Title: "P", Type: "text", Status: "published", MediaUrls: "[]", Tags: "[]"}
	_ = postRepo.Create(post)
	_ = commentRepo.Create(&model.Comment{PostID: post.ID, AuthorID: u2.ID, Content: "hi"})
	_ = followRepo.Follow(&model.UserFollow{FollowerID: u2.ID, FollowingID: u1.ID, Status: "accepted"})
	_ = userRepo.SyncFollowCounts(u1.ID)

	stats, err := userSvc.GetProfileStats(u1.ID)
	if err != nil {
		t.Fatalf("GetProfileStats: %v", err)
	}
	if stats.FollowersCount != 1 || stats.PostsCount != 1 || stats.CommunitiesCount != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestCommentService_CreatesNotifications(t *testing.T) {
	db := setupDB(t)
	userRepo := repository.NewUserRepository(db)
	postRepo := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	commRepo := repository.NewCommunityRepository(db)

	author := &model.User{Username: "author", Email: "author@example.com", PasswordHash: "h", ProfileTheme: "default"}
	replier := &model.User{Username: "replier", Email: "replier@example.com", PasswordHash: "h", ProfileTheme: "default"}
	mentioned := &model.User{Username: "mentioned", Email: "mentioned@example.com", PasswordHash: "h", ProfileTheme: "default"}
	_ = userRepo.Create(author)
	_ = userRepo.Create(replier)
	_ = userRepo.Create(mentioned)

	comm := &model.Community{Name: "C", Slug: "c", OwnerID: author.ID, Visibility: "public"}
	_ = commRepo.Create(comm)
	post := &model.Post{CommunityID: comm.ID, AuthorID: author.ID, Title: "Post", Type: "text", Status: "published", MediaUrls: "[]", Tags: "[]"}
	_ = postRepo.Create(post)

	commentSvc := service.NewCommentService(commentRepo, userRepo, postRepo, voteRepo, notifRepo, commRepo)

	parent := &model.Comment{PostID: post.ID, AuthorID: author.ID, Content: "parent"}
	if err := commentSvc.Create(parent); err != nil {
		t.Fatalf("create parent: %v", err)
	}

	reply := &model.Comment{PostID: post.ID, AuthorID: replier.ID, Content: "reply here", ParentID: &parent.ID}
	if err := commentSvc.Create(reply); err != nil {
		t.Fatalf("create reply: %v", err)
	}

	mention := &model.Comment{PostID: post.ID, AuthorID: replier.ID, Content: "hello @mentioned"}
	if err := commentSvc.Create(mention); err != nil {
		t.Fatalf("create mention: %v", err)
	}

	notifs, err := notifRepo.GetByUser(author.ID)
	if err != nil {
		t.Fatalf("GetByUser: %v", err)
	}
	types := map[string]int{}
	for _, n := range notifs {
		types[n.Type]++
	}
	if types["reply"] < 1 {
		t.Errorf("expected reply notification for post author, got types=%v", types)
	}

	mentionedNotifs, _ := notifRepo.GetByUser(mentioned.ID)
	foundMention := false
	for _, n := range mentionedNotifs {
		if n.Type == "mention" {
			foundMention = true
		}
	}
	if !foundMention {
		t.Error("expected mention notification for @mentioned user")
	}
}

func TestPostService_PublishDueScheduled(t *testing.T) {
	db := setupDB(t)
	postRepo := repository.NewPostRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	userRepo := repository.NewUserRepository(db)
	commRepo := repository.NewCommunityRepository(db)
	postSvc := service.NewPostService(postRepo, userRepo, commRepo, voteRepo, repository.NewSavedPostRepository(db), repository.NewNotificationRepository(db))

	user := &model.User{Username: "sched", Email: "sched@example.com", PasswordHash: "h", ProfileTheme: "default"}
	_ = userRepo.Create(user)
	comm := &model.Community{Name: "Sched", Slug: "sched", OwnerID: user.ID, Visibility: "public"}
	_ = commRepo.Create(comm)

	past := time.Now().Add(-1 * time.Minute)
	scheduled := &model.Post{
		CommunityID: comm.ID,
		AuthorID:    user.ID,
		Title:       "Future post",
		Type:        "text",
		Status:      "scheduled",
		PublishAt:   &past,
		MediaUrls:   "[]",
		Tags:        "[]",
	}
	if err := postRepo.Create(scheduled); err != nil {
		t.Fatalf("create scheduled post: %v", err)
	}

	published, err := postSvc.PublishDueScheduled()
	if err != nil {
		t.Fatalf("PublishDueScheduled: %v", err)
	}
	if len(published) != 1 {
		t.Fatalf("expected 1 published post, got %d", len(published))
	}
	updated, _ := postRepo.GetByID(scheduled.ID)
	if updated.Status != "published" {
		t.Errorf("expected status published, got %s", updated.Status)
	}
}
