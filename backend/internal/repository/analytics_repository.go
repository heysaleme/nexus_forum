package repository

import (
	"fmt"
	"nexus-forum-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	Track(event *model.AnalyticsEvent) error
	CountEvents(eventType string, since, until int64) (int64, error)
	GetTopPosts(limit int) ([]*model.Post, error)
	GetUserGrowth(days int) ([]map[string]interface{}, error)
	GetActivitySeries(days int) ([]map[string]interface{}, error)
	GetReportReasonBreakdown() ([]map[string]interface{}, error)
	CountActiveUsers(since time.Time) (int64, error)
	CountTotalUsers() (int64, error)
	CountBannedUsers() (int64, error)
	CountAdminUsers() (int64, error)
	CountCommunities() (int64, error)
	CountPublishedPosts() (int64, error)
	CountPendingReports() (int64, error)
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
	interval := fmt.Sprintf("-%d days", days)
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

func (r *analyticsRepository) CountActiveUsers(since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.AnalyticsEvent{}).
		Select("COUNT(DISTINCT user_id)").
		Where("user_id IS NOT NULL AND created_at >= ?", since).
		Scan(&count).Error
	return count, err
}

func (r *analyticsRepository) CountTotalUsers() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

func (r *analyticsRepository) CountBannedUsers() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("is_banned = ?", true).Count(&count).Error
	return count, err
}

func (r *analyticsRepository) CountAdminUsers() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("role = ?", "admin").Count(&count).Error
	return count, err
}

func (r *analyticsRepository) CountCommunities() (int64, error) {
	var count int64
	err := r.db.Model(&model.Community{}).Count(&count).Error
	return count, err
}

func (r *analyticsRepository) CountPublishedPosts() (int64, error) {
	var count int64
	err := r.db.Model(&model.Post{}).Where("status = ?", "published").Count(&count).Error
	return count, err
}

func (r *analyticsRepository) CountPendingReports() (int64, error) {
	var count int64
	err := r.db.Model(&model.Report{}).Where("status = ?", "pending").Count(&count).Error
	return count, err
}

func (r *analyticsRepository) GetReportReasonBreakdown() ([]map[string]interface{}, error) {
	type reasonCount struct {
		Reason string `gorm:"column:reason"`
		Count  int64  `gorm:"column:count"`
	}
	var results []reasonCount
	err := r.db.Model(&model.Report{}).
		Select("reason, COUNT(*) as count").
		Group("reason").
		Order("count DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]interface{}, 0, len(results))
	for _, row := range results {
		rows = append(rows, map[string]interface{}{
			"reason": row.Reason,
			"count":  row.Count,
		})
	}
	return rows, nil
}

func (r *analyticsRepository) GetActivitySeries(days int) ([]map[string]interface{}, error) {
	if days <= 0 {
		days = 7
	}
	interval := fmt.Sprintf("-%d days", days)

	type dayCount struct {
		Day   string `gorm:"column:day"`
		Count int64  `gorm:"column:count"`
	}

	var userRows []dayCount
	userQuery := "SELECT DATE(created_at) as day, COUNT(*) as count FROM users " +
		"WHERE created_at >= DATE('now', '" + interval + "') " +
		"GROUP BY DATE(created_at) ORDER BY day ASC"
	if err := r.db.Raw(userQuery).Scan(&userRows).Error; err != nil {
		return nil, err
	}

	var postRows []dayCount
	postQuery := "SELECT DATE(created_at) as day, COUNT(*) as count FROM posts " +
		"WHERE status = 'published' AND created_at >= DATE('now', '" + interval + "') " +
		"GROUP BY DATE(created_at) ORDER BY day ASC"
	if err := r.db.Raw(postQuery).Scan(&postRows).Error; err != nil {
		return nil, err
	}

	byDay := make(map[string]map[string]interface{})
	for _, row := range userRows {
		byDay[row.Day] = map[string]interface{}{
			"day":   row.Day,
			"users": row.Count,
			"posts": int64(0),
		}
	}
	for _, row := range postRows {
		if entry, ok := byDay[row.Day]; ok {
			entry["posts"] = row.Count
		} else {
			byDay[row.Day] = map[string]interface{}{
				"day":   row.Day,
				"users": int64(0),
				"posts": row.Count,
			}
		}
	}

	daysList := make([]string, 0, len(byDay))
	for day := range byDay {
		daysList = append(daysList, day)
	}
	// simple sort by day string (ISO dates sort lexicographically)
	for i := 0; i < len(daysList); i++ {
		for j := i + 1; j < len(daysList); j++ {
			if daysList[j] < daysList[i] {
				daysList[i], daysList[j] = daysList[j], daysList[i]
			}
		}
	}

	series := make([]map[string]interface{}, 0, len(byDay))
	for _, day := range daysList {
		series = append(series, byDay[day])
	}
	return series, nil
}
