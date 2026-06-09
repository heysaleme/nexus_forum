package repository

import (
	"strings"
	"time"

	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type SearchQueryRepository interface {
	Record(query string) error
	Trending(limit int) ([]*model.SearchQuery, error)
}

type searchQueryRepository struct {
	db *gorm.DB
}

func NewSearchQueryRepository(db *gorm.DB) SearchQueryRepository {
	return &searchQueryRepository{db: db}
}

func normalizeQuery(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}

func (r *searchQueryRepository) Record(query string) error {
	q := normalizeQuery(query)
	if len(q) < 2 {
		return nil
	}
	var existing model.SearchQuery
	if err := r.db.Where("query = ?", q).First(&existing).Error; err == nil {
		return r.db.Model(&existing).Updates(map[string]interface{}{
			"count":      gorm.Expr("count + 1"),
			"updated_at": time.Now(),
		}).Error
	}
	return r.db.Create(&model.SearchQuery{Query: q, Count: 1, UpdatedAt: time.Now()}).Error
}

func (r *searchQueryRepository) Trending(limit int) ([]*model.SearchQuery, error) {
	if limit <= 0 {
		limit = 10
	}
	var rows []*model.SearchQuery
	err := r.db.Order("count DESC, updated_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
