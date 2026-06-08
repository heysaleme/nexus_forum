package repository

import (
	"testing"
	"time"

	"nexus-forum-backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPostRepository_ListHotSort(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Community{}, &model.Post{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	user := model.User{Username: "author", Email: "a@example.com", PasswordHash: "h"}
	comm := model.Community{Name: "c", Slug: "c", OwnerID: 1}
	db.Create(&user)
	comm.OwnerID = user.ID
	db.Create(&comm)

	oldHigh := model.Post{
		CommunityID: comm.ID,
		AuthorID:    user.ID,
		Title:       "old popular",
		Type:        "text",
		Score:       100,
		Status:      "published",
		MediaUrls:   "[]",
		Tags:        "[]",
		CreatedAt:   time.Now().Add(-48 * time.Hour),
	}
	newLow := model.Post{
		CommunityID: comm.ID,
		AuthorID:    user.ID,
		Title:       "new rising",
		Type:        "text",
		Score:       10,
		Status:      "published",
		MediaUrls:   "[]",
		Tags:        "[]",
		CreatedAt:   time.Now().Add(-1 * time.Hour),
	}
	db.Create(&oldHigh)
	db.Create(&newLow)

	repo := NewPostRepository(db)
	posts, err := repo.List("hot", 10, 0)
	if err != nil {
		t.Fatalf("List hot: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
	if posts[0].Title != "new rising" {
		t.Errorf("expected hot sort to prefer newer post first, got %q then %q", posts[0].Title, posts[1].Title)
	}
}
