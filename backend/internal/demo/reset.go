package demo

import (
	"encoding/json"
	"log"
	"time"

	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/search"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ResetMinimalDemo backs up are handled by the caller. This resets DB content to a minimal demo dataset.
func ResetMinimalDemo(db *gorm.DB) error {
	tables := []string{
		"messages", "chat_rooms", "notifications", "votes", "comments",
		"saved_posts", "reports", "moderation_logs", "post_search_tokens",
		"analytics_events", "user_follows", "community_members", "posts", "communities",
		"refresh_tokens", "password_reset_tokens", "email_verifications", "users",
	}
	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			return err
		}
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	passStr := string(hashedPassword)

	admin := model.User{Username: "amira", Email: "amira@example.com", PasswordHash: passStr, Bio: "Администратор Nexus Forum.", Role: "admin", ProfileTheme: "sunset", AllowDMs: true}
	mod := model.User{Username: "moduser", Email: "moderator@example.com", PasswordHash: passStr, Bio: "Модератор платформы.", Role: "moderator", ProfileTheme: "forest", AllowDMs: true}
	user := model.User{Username: "kaizer", Email: "kai@example.com", PasswordHash: passStr, Bio: "Обычный пользователь для демо.", Role: "user", ProfileTheme: "ocean", AllowDMs: true}
	if err := db.Create(&admin).Error; err != nil {
		return err
	}
	if err := db.Create(&mod).Error; err != nil {
		return err
	}
	if err := db.Create(&user).Error; err != nil {
		return err
	}

	comm := model.Community{Name: "Nexus Anime", Slug: "nexus-anime", Description: "Демо-сообщество для тестов.", Visibility: "public", OwnerID: admin.ID, MemberCount: 3, PostCount: 5}
	if err := db.Create(&comm).Error; err != nil {
		return err
	}
	for _, u := range []model.User{admin, mod, user} {
		role := "member"
		if u.ID == admin.ID {
			role = "owner"
		}
		if err := db.Create(&model.CommunityMember{UserID: u.ID, CommunityID: comm.ID, Role: role}).Error; err != nil {
			return err
		}
	}

	future := time.Now().Add(48 * time.Hour)
	past := time.Now().Add(-2 * time.Minute)

	posts := []model.Post{
		{CommunityID: comm.ID, AuthorID: admin.ID, Title: "Текстовый демо-пост", Content: "Пример текстовой публикации с нормальными абзацами.\n\nВторой абзац для проверки читаемости.", Type: "text", Status: "published", Tags: `["demo"]`, MediaUrls: `[]`},
		{CommunityID: comm.ID, AuthorID: user.ID, Title: "Пост с изображением", Content: "Демонстрация image-поста.", Type: "image", Status: "published", Tags: `["demo","image"]`, MediaUrls: `["https://picsum.photos/seed/nexus-demo/800/500"]`},
		{CommunityID: comm.ID, AuthorID: mod.ID, Title: "Пост с видео", Content: "Демонстрация video-поста.", Type: "video", Status: "published", Tags: `["demo","video"]`, MediaUrls: `["https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4"]`},
		{CommunityID: comm.ID, AuthorID: admin.ID, Title: "Отложенная публикация", Content: "Будет опубликован позже.", Type: "text", Status: "scheduled", PublishAt: &future, Tags: `["scheduled"]`, MediaUrls: `[]`},
		{CommunityID: comm.ID, AuthorID: admin.ID, Title: "Черновик админа", Content: "Не опубликован.", Type: "text", Status: "draft", Tags: `[]`, MediaUrls: `[]`},
	}
	for i := range posts {
		if err := db.Create(&posts[i]).Error; err != nil {
			return err
		}
		if posts[i].Status == "published" {
			_ = search.SyncPostIndex(db, posts[i].ID, posts[i].Title, posts[i].Content, posts[i].Tags)
		}
	}

	// Publish one scheduled post in the past to demonstrate worker output
	dueScheduled := model.Post{CommunityID: comm.ID, AuthorID: user.ID, Title: "Просроченный отложенный пост", Content: "Должен опубликоваться воркером.", Type: "text", Status: "scheduled", PublishAt: &past, Tags: `[]`, MediaUrls: `[]`}
	if err := db.Create(&dueScheduled).Error; err != nil {
		return err
	}

	parent := model.Comment{PostID: posts[0].ID, AuthorID: user.ID, Content: "Обычный комментарий к демо-посту."}
	if err := db.Create(&parent).Error; err != nil {
		return err
	}
	reply := model.Comment{PostID: posts[0].ID, ParentID: &parent.ID, AuthorID: admin.ID, Content: "Вложенный ответ на комментарий."}
	if err := db.Create(&reply).Error; err != nil {
		return err
	}
	modComment := model.Comment{PostID: posts[0].ID, AuthorID: user.ID, Content: "[удалено модератором]", IsDeleted: true}
	if err := db.Create(&modComment).Error; err != nil {
		return err
	}

	_ = db.Create(&model.UserFollow{FollowerID: user.ID, FollowingID: admin.ID, Status: "accepted"})
	_ = db.Create(&model.SavedPost{UserID: admin.ID, PostID: posts[1].ID})
	_ = db.Create(&model.Report{ReporterID: user.ID, ReporterUsername: user.Username, TargetID: posts[0].ID, TargetType: "post", Reason: "spam", Description: "Демо-жалоба на пост.", Status: "pending"})

	pBytes, _ := json.Marshal([]uint{admin.ID, user.ID})
	room := model.ChatRoom{Name: "Amira & Kai", Type: "direct", Participants: string(pBytes), LastMessage: "Привет из демо-чата!"}
	if err := db.Create(&room).Error; err != nil {
		return err
	}
	_ = db.Create(&model.Message{ChatRoomID: room.ID, SenderID: user.ID, SenderUsername: user.Username, Content: "Привет из демо-чата!"})
	_ = db.Create(&model.Message{ChatRoomID: room.ID, SenderID: admin.ID, SenderUsername: admin.Username, Content: "Сообщения и вложения работают."})

	// Sync follower counts from real relations
	for _, uid := range []uint{admin.ID, mod.ID, user.ID} {
		var followers, following int64
		_ = db.Model(&model.UserFollow{}).Where("following_id = ? AND status = ?", uid, "accepted").Count(&followers)
		_ = db.Model(&model.UserFollow{}).Where("follower_id = ? AND status = ?", uid, "accepted").Count(&following)
		_ = db.Model(&model.User{}).Where("id = ?", uid).Updates(map[string]interface{}{"followers_count": followers, "following_count": following})
	}

	// Sync post comment counts (includes replies; matches post page comment list length)
	for i := range posts {
		var commentCount int64
		_ = db.Model(&model.Comment{}).Where("post_id = ?", posts[i].ID).Count(&commentCount)
		_ = db.Model(&model.Post{}).Where("id = ?", posts[i].ID).UpdateColumn("comment_count", commentCount)
	}

	log.Println("demo database reset to minimal dataset complete")
	return nil
}
