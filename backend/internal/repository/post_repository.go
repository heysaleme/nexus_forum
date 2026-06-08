package repository

import (
	"strings"
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type PostRepository interface {
	Create(post *model.Post) error
	GetByID(id uint) (*model.Post, error)
	Update(post *model.Post) error
	Delete(id uint) error
	List(sortSpec string, limit int, viewerID uint) ([]*model.Post, error)
	Filter(filter map[string]interface{}, sortSpec string, limit int, viewerID uint) ([]*model.Post, error)
	Search(query string, limit int, viewerID uint) ([]*model.Post, error)
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(post *model.Post) error {
	return r.db.Create(post).Error
}

func (r *postRepository) GetByID(id uint) (*model.Post, error) {
	var post model.Post
	err := r.db.First(&post, id).Error
	if err != nil {
		return nil, err
	}
	r.hydratePostFields([]*model.Post{&post})
	return &post, nil
}

func (r *postRepository) Update(post *model.Post) error {
	return r.db.Save(post).Error
}

func (r *postRepository) Delete(id uint) error {
	return r.db.Delete(&model.Post{}, id).Error
}

func (r *postRepository) List(sortSpec string, limit int, viewerID uint) ([]*model.Post, error) {
	var posts []*model.Post
	q := r.db.Order(parsePostSort(r.db.Dialector.Name(), sortSpec))
	q = r.applyShadowFilter(q, viewerID)
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
}

func (r *postRepository) Filter(filter map[string]interface{}, sortSpec string, limit int, viewerID uint) ([]*model.Post, error) {
	var posts []*model.Post
	q := r.db.Where(filter).Order(parsePostSort(r.db.Dialector.Name(), sortSpec))
	q = r.applyShadowFilter(q, viewerID)
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
}

func (r *postRepository) Search(query string, limit int, viewerID uint) ([]*model.Post, error) {
	var posts []*model.Post
	dialect := r.db.Dialector.Name()
	var q *gorm.DB
	if dialect == "postgres" {
		q = r.db.Where("status = ? AND to_tsvector('simple', title || ' ' || content) @@ plainto_tsquery('simple', ?)", "published", query)
	} else {
		likePattern := "%" + strings.ToLower(query) + "%"
		q = r.db.Where("status = ? AND (LOWER(title) LIKE ? OR LOWER(content) LIKE ?)", "published", likePattern, likePattern)
	}
	q = r.applyShadowFilter(q, viewerID)

	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
}

// applyShadowFilter filters out shadow-banned author posts and shadow content
// unless the viewer is the author themselves.
func (r *postRepository) applyShadowFilter(q *gorm.DB, viewerID uint) *gorm.DB {
	// Subquery: get IDs of shadow-banned users
	shadowBannedSubquery := r.db.Model(&model.User{}).Select("id").Where("is_shadow_banned = ?", true)
	if viewerID > 0 {
		// Hide shadow-banned author posts unless viewer IS the author
		q = q.Where("author_id NOT IN (?) OR author_id = ?", shadowBannedSubquery, viewerID)
		// Hide shadow content unless viewer IS the author
		q = q.Where("is_shadow_content = ? OR author_id = ?", false, viewerID)
	} else {
		// Anonymous viewer — hide all shadow content and shadow-banned author posts
		q = q.Where("author_id NOT IN (?)", shadowBannedSubquery)
		q = q.Where("is_shadow_content = ?", false)
	}
	return q
}

func (r *postRepository) hydratePostFields(posts []*model.Post) {
	if len(posts) == 0 {
		return
	}
	for _, post := range posts {
		var author model.User
		if err := r.db.First(&author, post.AuthorID).Error; err == nil {
			post.AuthorUsername = author.Username
			post.AuthorAvatar = author.AvatarURL
		}
		var community model.Community
		if err := r.db.First(&community, post.CommunityID).Error; err == nil {
			post.CommunityName = community.Name
			post.CommunityAvatar = community.AvatarURL
		}
	}
}
