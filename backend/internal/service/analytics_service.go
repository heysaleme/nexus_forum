package service

import (
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
	totalUsers, _ := s.repo.CountEvents("register", 0, 0)
	totalLogins, _ := s.repo.CountEvents("login", 0, 0)
	totalPostViews, _ := s.repo.CountEvents("page_view", 0, 0)
	topPosts, _ := s.repo.GetTopPosts(10)
	userGrowth, _ := s.repo.GetUserGrowth(30)

	// Count total users directly from user repo via List
	allUsers, _ := s.userRepo.List("", 10000)
	totalUserCount := int64(len(allUsers))

	return map[string]interface{}{
		"total_users":                 totalUserCount,
		"total_registrations_tracked": totalUsers,
		"total_logins_tracked":        totalLogins,
		"total_page_views":            totalPostViews,
		"top_posts":                   topPosts,
		"user_growth_30d":             userGrowth,
	}, nil
}
