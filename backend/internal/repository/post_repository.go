package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type PostRepository interface {
	Create(post *model.Post) error
	GetByID(id uint) (*model.Post, error)
	Update(post *model.Post) error
	Delete(id uint) error
	List(sortSpec string, limit int) ([]*model.Post, error)
	Filter(filter map[string]interface{}, sortSpec string, limit int) ([]*model.Post, error)
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

func (r *postRepository) List(sortSpec string, limit int) ([]*model.Post, error) {
	var posts []*model.Post
	q := r.db.Order(parseSort(sortSpec))
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
}

func (r *postRepository) Filter(filter map[string]interface{}, sortSpec string, limit int) ([]*model.Post, error) {
	var posts []*model.Post
	q := r.db.Where(filter).Order(parseSort(sortSpec))
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
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
