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
	GetRetentionRates() (map[string]float64, error)
	GetEngagementStats() (map[string]interface{}, error)
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

	var commentRows []dayCount
	commentQuery := "SELECT DATE(created_at) as day, COUNT(*) as count FROM comments " +
		"WHERE created_at >= DATE('now', '" + interval + "') " +
		"GROUP BY DATE(created_at) ORDER BY day ASC"
	if err := r.db.Raw(commentQuery).Scan(&commentRows).Error; err != nil {
		return nil, err
	}

	byDay := make(map[string]map[string]interface{})
	ensureDay := func(day string) map[string]interface{} {
		if entry, ok := byDay[day]; ok {
			return entry
		}
		entry := map[string]interface{}{
			"day":      day,
			"users":    int64(0),
			"posts":    int64(0),
			"comments": int64(0),
		}
		byDay[day] = entry
		return entry
	}
	for _, row := range userRows {
		ensureDay(row.Day)["users"] = row.Count
	}
	for _, row := range postRows {
		ensureDay(row.Day)["posts"] = row.Count
	}
	for _, row := range commentRows {
		ensureDay(row.Day)["comments"] = row.Count
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

func (r *analyticsRepository) GetRetentionRates() (map[string]float64, error) {
	cohorts := []struct {
		key  string
		days int
	}{
		{"d1", 1},
		{"d7", 7},
		{"d30", 30},
	}
	out := map[string]float64{"d1": 0, "d7": 0, "d30": 0}
	for _, c := range cohorts {
		rate, err := r.retentionForDay(c.days)
		if err != nil {
			return nil, err
		}
		out[c.key] = rate
	}
	return out, nil
}

func (r *analyticsRepository) retentionForDay(dayOffset int) (float64, error) {
	type row struct {
		Cohort int64 `gorm:"column:cohort"`
		Back   int64 `gorm:"column:returned"`
	}
	var result row
	query := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT u.id) AS cohort,
			COUNT(DISTINCT CASE
				WHEN EXISTS (
					SELECT 1 FROM analytics_events e
					WHERE e.user_id = u.id
					AND DATE(e.created_at) = DATE(u.created_at, '+%d days')
				) THEN u.id END) AS returned
		FROM users u
		WHERE u.created_at <= DATE('now', '-%d days')
	`, dayOffset, dayOffset)
	if err := r.db.Raw(query).Scan(&result).Error; err != nil {
		return 0, err
	}
	if result.Cohort == 0 {
		return 0, nil
	}
	return float64(result.Back) / float64(result.Cohort) * 100, nil
}

func (r *analyticsRepository) GetEngagementStats() (map[string]interface{}, error) {
	type countRow struct {
		UserID uint  `gorm:"column:user_id"`
		Count  int64 `gorm:"column:count"`
	}

	var postCounts []countRow
	_ = r.db.Table("posts").Select("author_id as user_id, COUNT(*) as count").
		Where("status = ?", "published").Group("author_id").Order("count DESC").Limit(10).Scan(&postCounts)

	var commentCounts []countRow
	_ = r.db.Table("comments").Select("author_id as user_id, COUNT(*) as count").
		Group("author_id").Order("count DESC").Limit(10).Scan(&commentCounts)

	var voteCounts []countRow
	_ = r.db.Table("votes").Select("user_id, COUNT(*) as count").
		Group("user_id").Order("count DESC").Limit(10).Scan(&voteCounts)

	activeUsers, _ := r.CountActiveUsers(time.Now().Add(-7 * 24 * time.Hour))

	var totalPosts int64
	_ = r.db.Model(&model.Post{}).Where("status = ?", "published").Count(&totalPosts)
	var totalComments int64
	_ = r.db.Model(&model.Comment{}).Count(&totalComments)
	var totalVotes int64
	_ = r.db.Model(&model.Vote{}).Count(&totalVotes)
	totalUsers, _ := r.CountTotalUsers()

	engagementScore := float64(0)
	if totalUsers > 0 {
		engagementScore = float64(totalPosts+totalComments+totalVotes) / float64(totalUsers)
	}

	return map[string]interface{}{
		"posts_per_user_top":    postCounts,
		"comments_per_user_top": commentCounts,
		"votes_per_user_top":    voteCounts,
		"active_users_7d":       activeUsers,
		"engagement_score":      engagementScore,
		"total_posts":           totalPosts,
		"total_comments":        totalComments,
		"total_votes":           totalVotes,
	}, nil
}
