package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type FollowRepository interface {
	Follow(follow *model.UserFollow) error
	Unfollow(followerID, followingID uint) error
	GetFollow(followerID, followingID uint) (*model.UserFollow, error)
	GetFollowers(userID uint) ([]*model.UserFollow, error)
	GetFollowing(userID uint) ([]*model.UserFollow, error)
	GetPendingRequests(userID uint) ([]*model.UserFollow, error)
	Update(follow *model.UserFollow) error
}

type followRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepository{db: db}
}

func (r *followRepository) Follow(follow *model.UserFollow) error {
	return r.db.Create(follow).Error
}

func (r *followRepository) Unfollow(followerID, followingID uint) error {
	return r.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).Delete(&model.UserFollow{}).Error
}

func (r *followRepository) GetFollow(followerID, followingID uint) (*model.UserFollow, error) {
	var follow model.UserFollow
	err := r.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&follow).Error
	return &follow, err
}

func (r *followRepository) GetFollowers(userID uint) ([]*model.UserFollow, error) {
	var follows []*model.UserFollow
	err := r.db.Where("following_id = ? AND status = ?", userID, "accepted").Find(&follows).Error
	return follows, err
}

func (r *followRepository) GetFollowing(userID uint) ([]*model.UserFollow, error) {
	var follows []*model.UserFollow
	err := r.db.Where("follower_id = ? AND status = ?", userID, "accepted").Find(&follows).Error
	return follows, err
}

func (r *followRepository) GetPendingRequests(userID uint) ([]*model.UserFollow, error) {
	var follows []*model.UserFollow
	err := r.db.Where("following_id = ? AND status = ?", userID, "pending").Find(&follows).Error
	return follows, err
}

func (r *followRepository) Update(follow *model.UserFollow) error {
	return r.db.Save(follow).Error
}
