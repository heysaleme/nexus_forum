package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type CommentRepository interface {
	Create(comment *model.Comment) error
	GetByID(id uint) (*model.Comment, error)
	Update(comment *model.Comment) error
	Delete(id uint) error
	GetByPostID(postID uint, viewerID uint) ([]*model.Comment, error)
}

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(comment *model.Comment) error {
	return r.db.Create(comment).Error
}

func (r *commentRepository) GetByID(id uint) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	r.hydrateCommentFields([]*model.Comment{&comment})
	return &comment, nil
}

func (r *commentRepository) Update(comment *model.Comment) error {
	return r.db.Save(comment).Error
}

func (r *commentRepository) Delete(id uint) error {
	return r.db.Delete(&model.Comment{}, id).Error
}

func (r *commentRepository) GetByPostID(postID uint, viewerID uint) ([]*model.Comment, error) {
	var comments []*model.Comment

	q := r.db.Where("post_id = ?", postID)

	shadowBannedSubquery := r.db.
		Model(&model.User{}).
		Select("id").
		Where("is_shadow_banned = ?", true)

	if viewerID > 0 {
		q = q.Where(
			"(author_id NOT IN (?) OR author_id = ?)",
			shadowBannedSubquery,
			viewerID,
		)

		q = q.Where(
			"(is_shadow_content = ? OR author_id = ?)",
			false,
			viewerID,
		)
	} else {
		q = q.Where(
			"author_id NOT IN (?)",
			shadowBannedSubquery,
		)

		q = q.Where(
			"is_shadow_content = ?",
			false,
		)
	}

	err := q.Order("created_at ASC").
		Find(&comments).Error

	if err == nil {
		r.hydrateCommentFields(comments)
	}

	return comments, err
}

func (r *commentRepository) hydrateCommentFields(comments []*model.Comment) {
	if len(comments) == 0 {
		return
	}
	for _, comment := range comments {
		var author model.User
		if err := r.db.First(&author, comment.AuthorID).Error; err == nil {
			comment.AuthorUsername = author.Username
			comment.AuthorAvatar = author.AvatarURL
		}
	}
}
