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

func TestPostRepository_ListByFollowing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserFollow{}, &model.Community{}, &model.Post{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	viewer := model.User{Username: "viewer", Email: "viewer@example.com", PasswordHash: "h"}
	author := model.User{Username: "author", Email: "author@example.com", PasswordHash: "h"}
	other := model.User{Username: "other", Email: "other@example.com", PasswordHash: "h"}
	db.Create(&viewer)
	db.Create(&author)
	db.Create(&other)

	comm := model.Community{Name: "c", Slug: "c", OwnerID: author.ID}
	db.Create(&comm)

	db.Create(&model.UserFollow{FollowerID: viewer.ID, FollowingID: author.ID, Status: "accepted"})

	followedPost := model.Post{
		CommunityID: comm.ID,
		AuthorID:    author.ID,
		Title:       "from followed",
		Type:        "text",
		Status:      "published",
		MediaUrls:   "[]",
		Tags:        "[]",
	}
	otherPost := model.Post{
		CommunityID: comm.ID,
		AuthorID:    other.ID,
		Title:       "not followed",
		Type:        "text",
		Status:      "published",
		MediaUrls:   "[]",
		Tags:        "[]",
	}
	db.Create(&followedPost)
	db.Create(&otherPost)

	repo := NewPostRepository(db)
	posts, err := repo.ListByFollowing(viewer.ID, "new", 10, viewer.ID)
	if err != nil {
		t.Fatalf("ListByFollowing: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Title != "from followed" {
		t.Errorf("expected followed author post, got %q", posts[0].Title)
	}
}

func TestPostRepository_ListIncludesContent(t *testing.T) {
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

	body := "Feed preview body that must be returned by list queries."
	post := model.Post{
		CommunityID: comm.ID,
		AuthorID:    user.ID,
		Title:       "with content",
		Content:     body,
		Type:        "text",
		Status:      "published",
		MediaUrls:   "[]",
		Tags:        "[]",
	}
	db.Create(&post)

	repo := NewPostRepository(db)
	posts, err := repo.List("new", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Content != body {
		t.Errorf("expected content %q, got %q", body, posts[0].Content)
	}
}

func TestPostRepository_HydrateCommentCounts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Community{}, &model.Post{}, &model.Comment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	user := model.User{Username: "author", Email: "a@example.com", PasswordHash: "h"}
	comm := model.Community{Name: "c", Slug: "c", OwnerID: 1}
	db.Create(&user)
	comm.OwnerID = user.ID
	db.Create(&comm)

	post := model.Post{
		CommunityID:  comm.ID,
		AuthorID:     user.ID,
		Title:        "comments",
		Content:      "x",
		Type:         "text",
		Status:       "published",
		CommentCount: 0,
		MediaUrls:    "[]",
		Tags:         "[]",
	}
	db.Create(&post)

	parent := model.Comment{PostID: post.ID, AuthorID: user.ID, Content: "parent"}
	db.Create(&parent)
	reply := model.Comment{PostID: post.ID, ParentID: &parent.ID, AuthorID: user.ID, Content: "reply"}
	db.Create(&reply)

	repo := NewPostRepository(db)
	posts, err := repo.List("new", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].CommentCount != 2 {
		t.Errorf("expected hydrated comment_count 2 (parent + reply), got %d", posts[0].CommentCount)
	}
}
