package search

import (
	"testing"

	"nexus-forum-backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSearchDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Community{}, &model.Post{}, &model.PostSearchToken{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := Init(db); err != nil {
		t.Fatalf("init search: %v", err)
	}
	return db
}

func TestUserIDsMatching_CyrillicAndLatin(t *testing.T) {
	db := setupSearchDB(t)
	users := []model.User{
		{Username: "amira", Email: "amira@example.com", PasswordHash: "x", ProfileTheme: "default"},
		{Username: "кирилл", Email: "k@example.com", PasswordHash: "x", ProfileTheme: "default", Bio: "любит аниме"},
	}
	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}

	ids, err := UserIDsMatching(db, "amira", 10)
	if err != nil {
		t.Fatalf("UserIDsMatching: %v", err)
	}
	if len(ids) != 1 || ids[0] != users[0].ID {
		t.Fatalf("expected amira id %d, got %v", users[0].ID, ids)
	}

	ids, err = UserIDsMatching(db, "кирил", 10)
	if err != nil {
		t.Fatalf("UserIDsMatching cyrillic: %v", err)
	}
	if len(ids) != 1 || ids[0] != users[1].ID {
		t.Fatalf("expected кирилл id %d, got %v", users[1].ID, ids)
	}

	ids, err = UserIDsMatching(db, "аниме", 10)
	if err != nil {
		t.Fatalf("UserIDsMatching bio: %v", err)
	}
	if len(ids) != 1 || ids[0] != users[1].ID {
		t.Fatalf("expected bio match id %d, got %v", users[1].ID, ids)
	}
}

func TestCommunityIDsMatching_Cyrillic(t *testing.T) {
	db := setupSearchDB(t)
	comm := model.Community{Name: "Аниме Лента", Slug: "anime-lenta", OwnerID: 1, Visibility: "public", Description: "сообщество фанатов"}
	if err := db.Create(&comm).Error; err != nil {
		t.Fatalf("seed community: %v", err)
	}

	ids, err := CommunityIDsMatching(db, "лент", 10)
	if err != nil {
		t.Fatalf("CommunityIDsMatching: %v", err)
	}
	if len(ids) != 1 || ids[0] != comm.ID {
		t.Fatalf("expected community id %d, got %v", comm.ID, ids)
	}
}

func TestPostIDsMatching_CyrillicTitle(t *testing.T) {
	db := setupSearchDB(t)
	post := &model.Post{
		CommunityID: 1,
		AuthorID:    1,
		Title:       "Обзор аниме ленты",
		Content:     "подробный разбор серии",
		Type:        "text",
		Status:      "published",
		MediaUrls:   "[]",
		Tags:        `["аниме"]`,
	}
	if err := db.Create(post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := SyncPostIndex(db, post.ID, post.Title, post.Content, post.Tags); err != nil {
		t.Fatalf("sync index: %v", err)
	}

	for _, q := range []string{"лент", "аниме", "обзор"} {
		ids, err := PostIDsMatching(db, q, 10)
		if err != nil {
			t.Fatalf("PostIDsMatching %q: %v", q, err)
		}
		if len(ids) != 1 || ids[0] != post.ID {
			t.Fatalf("query %q: expected post %d, got %v", q, post.ID, ids)
		}
	}
}
