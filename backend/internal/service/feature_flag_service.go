package service

import (
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type FeatureFlagService interface {
	List() ([]*model.FeatureFlag, error)
	Set(key string, enabled bool, description string) (*model.FeatureFlag, error)
	IsEnabled(key string) bool
}

type featureFlagService struct {
	repo repository.FeatureFlagRepository
}

func NewFeatureFlagService(repo repository.FeatureFlagRepository) FeatureFlagService {
	return &featureFlagService{repo: repo}
}

func (s *featureFlagService) List() ([]*model.FeatureFlag, error) {
	return s.repo.List()
}

func (s *featureFlagService) Set(key string, enabled bool, description string) (*model.FeatureFlag, error) {
	flag := &model.FeatureFlag{Key: key, Enabled: enabled, Description: description}
	if err := s.repo.Upsert(flag); err != nil {
		return nil, err
	}
	return s.repo.GetByKey(key)
}

func (s *featureFlagService) IsEnabled(key string) bool {
	return s.repo.IsEnabled(key)
}
