package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type KeywordFilterRepository interface {
	Create(f *model.KeywordFilter) error
	Delete(id uint) error
	List() ([]*model.KeywordFilter, error)
}

type keywordFilterRepository struct {
	db *gorm.DB
}

func NewKeywordFilterRepository(db *gorm.DB) KeywordFilterRepository {
	return &keywordFilterRepository{db: db}
}

func (r *keywordFilterRepository) Create(f *model.KeywordFilter) error {
	return r.db.Create(f).Error
}

func (r *keywordFilterRepository) Delete(id uint) error {
	return r.db.Delete(&model.KeywordFilter{}, id).Error
}

func (r *keywordFilterRepository) List() ([]*model.KeywordFilter, error) {
	var filters []*model.KeywordFilter
	err := r.db.Order("created_at asc").Find(&filters).Error
	return filters, err
}
