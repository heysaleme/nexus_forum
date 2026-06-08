package service

import (
	"time"

	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type AnalyticsService interface {
	// Track records an analytics event asynchronously (fire-and-forget).
	Track(userID *uint, eventType, entityType string, entityID *uint, meta string) error
	// GetDashboard returns aggregated stats for the admin panel.
	GetDashboard() (map[string]interface{}, error)
}

type analyticsService struct {
	repo     repository.AnalyticsRepository
	userRepo repository.UserRepository
	postRepo repository.PostRepository
}

func NewAnalyticsService(
	repo repository.AnalyticsRepository,
	userRepo repository.UserRepository,
	postRepo repository.PostRepository,
) AnalyticsService {
	return &analyticsService{repo: repo, userRepo: userRepo, postRepo: postRepo}
}

func (s *analyticsService) Track(userID *uint, eventType, entityType string, entityID *uint, meta string) error {
	event := &model.AnalyticsEvent{
		UserID:     userID,
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		Metadata:   meta,
	}
	// Fire-and-forget: errors are non-critical
	_ = s.repo.Track(event)
	return nil
}

func (s *analyticsService) GetDashboard() (map[string]interface{}, error) {
	totalRegistrations, _ := s.repo.CountEvents("register", 0, 0)
	totalLogins, _ := s.repo.CountEvents("login", 0, 0)
	totalPostViews, _ := s.repo.CountEvents("page_view", 0, 0)
	topPosts, _ := s.repo.GetTopPosts(10)
	userGrowth, _ := s.repo.GetUserGrowth(30)
	activity7d, _ := s.repo.GetActivitySeries(7)
	reportReasons, _ := s.repo.GetReportReasonBreakdown()
	dau, _ := s.repo.CountActiveUsers(time.Now().Add(-24 * time.Hour))
	mau, _ := s.repo.CountActiveUsers(time.Now().Add(-30 * 24 * time.Hour))
	totalUsers, _ := s.repo.CountTotalUsers()
	bannedUsers, _ := s.repo.CountBannedUsers()
	adminUsers, _ := s.repo.CountAdminUsers()
	totalCommunities, _ := s.repo.CountCommunities()
	totalPosts, _ := s.repo.CountPublishedPosts()
	pendingReports, _ := s.repo.CountPendingReports()

	return map[string]interface{}{
		"total_users":                 totalUsers,
		"banned_users":                bannedUsers,
		"total_admins":                adminUsers,
		"total_communities":           totalCommunities,
		"total_posts":                 totalPosts,
		"pending_reports":             pendingReports,
		"total_registrations_tracked": totalRegistrations,
		"total_logins_tracked":        totalLogins,
		"total_page_views":            totalPostViews,
		"dau":                         dau,
		"mau":                         mau,
		"top_posts":                   topPosts,
		"user_growth_30d":             userGrowth,
		"activity_7d":                 activity7d,
		"report_reasons":            reportReasons,
	}, nil
}
