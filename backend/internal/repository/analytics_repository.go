package repository

import (
	"nexus-forum-backend/internal/model"
	"strings"

	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	Track(event *model.AnalyticsEvent) error
	CountEvents(eventType string, since, until int64) (int64, error)
	GetTopPosts(limit int) ([]*model.Post, error)
	GetUserGrowth(days int) ([]map[string]interface{}, error)
}

type analyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) Track(event *model.AnalyticsEvent) error {
	return r.db.Create(event).Error
}

func (r *analyticsRepository) CountEvents(eventType string, since, until int64) (int64, error) {
	var count int64
	q := r.db.Model(&model.AnalyticsEvent{})
	if eventType != "" {
		q = q.Where("event_type = ?", eventType)
	}
	if since > 0 {
		q = q.Where("created_at >= ?", since)
	}
	if until > 0 {
		q = q.Where("created_at <= ?", until)
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *analyticsRepository) GetTopPosts(limit int) ([]*model.Post, error) {
	var posts []*model.Post
	q := r.db.Where("status = ?", "published").Order("score DESC, views DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	return posts, err
}

func (r *analyticsRepository) GetUserGrowth(days int) ([]map[string]interface{}, error) {
	type dayCount struct {
		Day   string `gorm:"column:day"`
		Count int64  `gorm:"column:count"`
	}
	var results []dayCount
	interval := strings.TrimSpace(strings.Repeat("-1 ", days))
	query := "SELECT DATE(created_at) as day, COUNT(*) as count FROM users " +
		"WHERE created_at >= DATE('now', '" + interval + "') " +
		"GROUP BY DATE(created_at) ORDER BY day ASC"
	err := r.db.Raw(query).Scan(&results).Error
	rows := make([]map[string]interface{}, 0, len(results))
	for _, row := range results {
		rows = append(rows, map[string]interface{}{
			"day":   row.Day,
			"count": row.Count,
		})
	}
	return rows, err
}
