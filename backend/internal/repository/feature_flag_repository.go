package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type FeatureFlagRepository interface {
	List() ([]*model.FeatureFlag, error)
	GetByKey(key string) (*model.FeatureFlag, error)
	Upsert(flag *model.FeatureFlag) error
	IsEnabled(key string) bool
}

type featureFlagRepository struct {
	db *gorm.DB
}

func NewFeatureFlagRepository(db *gorm.DB) FeatureFlagRepository {
	return &featureFlagRepository{db: db}
}

func (r *featureFlagRepository) List() ([]*model.FeatureFlag, error) {
	var flags []*model.FeatureFlag
	err := r.db.Order("key ASC").Find(&flags).Error
	return flags, err
}

func (r *featureFlagRepository) GetByKey(key string) (*model.FeatureFlag, error) {
	var flag model.FeatureFlag
	err := r.db.Where("key = ?", key).First(&flag).Error
	if err != nil {
		return nil, err
	}
	return &flag, nil
}

func (r *featureFlagRepository) Upsert(flag *model.FeatureFlag) error {
	var existing model.FeatureFlag
	if err := r.db.Where("key = ?", flag.Key).First(&existing).Error; err == nil {
		existing.Enabled = flag.Enabled
		existing.Description = flag.Description
		return r.db.Save(&existing).Error
	}
	return r.db.Create(flag).Error
}

func (r *featureFlagRepository) IsEnabled(key string) bool {
	flag, err := r.GetByKey(key)
	return err == nil && flag.Enabled
}
