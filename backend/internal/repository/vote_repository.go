package repository

import (
	"fmt"
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type VoteRepository interface {
	GetVote(userID uint, entityType string, entityID uint) (*model.Vote, error)
	GetVotesForEntities(userID uint, entityType string, entityIDs []uint) map[uint]int
	SaveVote(vote *model.Vote) error
	DeleteVote(userID uint, entityType string, entityID uint) error
}

type voteRepository struct {
	db *gorm.DB
}

func NewVoteRepository(db *gorm.DB) VoteRepository {
	return &voteRepository{db: db}
}

func (r *voteRepository) GetVotesForEntities(userID uint, entityType string, entityIDs []uint) map[uint]int {
	result := make(map[uint]int)
	if len(entityIDs) == 0 {
		return result
	}
	var votes []model.Vote
	if err := r.db.Where(
		"user_id = ? AND entity_type = ? AND entity_id IN ?",
		userID, entityType, entityIDs,
	).Find(&votes).Error; err != nil {
		return result
	}
	for _, vote := range votes {
		result[vote.EntityID] = vote.Value
	}
	return result
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

	fmt.Println("DELETE QUERY")
	fmt.Println("USER:", userID)
	fmt.Println("TYPE:", entityType)
	fmt.Println("ENTITY:", entityID)

	result := r.db.
		Where("user_id = ? AND entity_type = ? AND entity_id = ?",
			userID, entityType, entityID).
		Delete(&model.Vote{})

	fmt.Println("ROWS AFFECTED:", result.RowsAffected)
	fmt.Println("DELETE ERROR:", result.Error)

	return result.Error
}
