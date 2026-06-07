package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type VoteRepository interface {
	GetVote(userID uint, entityType string, entityID uint) (*model.Vote, error)
	SaveVote(vote *model.Vote) error
	DeleteVote(userID uint, entityType string, entityID uint) error
}

type voteRepository struct {
	db *gorm.DB
}

func NewVoteRepository(db *gorm.DB) VoteRepository {
	return &voteRepository{db: db}
}

func (r *voteRepository) GetVote(userID uint, entityType string, entityID uint) (*model.Vote, error) {
	var vote model.Vote
	err := r.db.Where("user_id = ? AND entity_type = ? AND entity_id = ?", userID, entityType, entityID).First(&vote).Error
	return &vote, err
}

func (r *voteRepository) SaveVote(vote *model.Vote) error {
	return r.db.Save(vote).Error
}

func (r *voteRepository) DeleteVote(userID uint, entityType string, entityID uint) error {
	return r.db.Where("user_id = ? AND entity_type = ? AND entity_id = ?", userID, entityType, entityID).Delete(&model.Vote{}).Error
}
