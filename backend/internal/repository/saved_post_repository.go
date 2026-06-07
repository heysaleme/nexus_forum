package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type SavedPostRepository interface {
	Save(saved *model.SavedPost) error
	Unsave(userID, postID uint) error
	GetByUser(userID uint) ([]*model.SavedPost, error)
	IsSaved(userID, postID uint) (bool, error)
}

type savedPostRepository struct {
	db *gorm.DB
}

func NewSavedPostRepository(db *gorm.DB) SavedPostRepository {
	return &savedPostRepository{db: db}
}

func (r *savedPostRepository) Save(saved *model.SavedPost) error {
	return r.db.Create(saved).Error
}

func (r *savedPostRepository) Unsave(userID, postID uint) error {
	return r.db.Where("user_id = ? AND post_id = ?", userID, postID).Delete(&model.SavedPost{}).Error
}

func (r *savedPostRepository) GetByUser(userID uint) ([]*model.SavedPost, error) {
	var saved []*model.SavedPost
	err := r.db.Where("user_id = ?", userID).Find(&saved).Error
	if err == nil {
		for _, s := range saved {
			var post model.Post
			if err := r.db.First(&post, s.PostID).Error; err == nil {
				s.PostTitle = post.Title
				var community model.Community
				if err := r.db.First(&community, post.CommunityID).Error; err == nil {
					s.PostCommunity = community.Name
				}
			}
		}
	}
	return saved, err
}

func (r *savedPostRepository) IsSaved(userID, postID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.SavedPost{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count).Error
	return count > 0, err
}
