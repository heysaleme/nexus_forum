package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type ModerationRepository interface {
	CreateLog(log *model.ModerationLog) error
	GetLogs(limit int) ([]*model.ModerationLog, error)
	GetLogsByCommunity(communityID uint, limit int) ([]*model.ModerationLog, error)
	GetLogsByTarget(targetType string, targetID uint) ([]*model.ModerationLog, error)

	CreateReport(report *model.Report) error
	GetReports() ([]*model.Report, error)
	GetReportByID(id uint) (*model.Report, error)
	UpdateReport(report *model.Report) error
}

type moderationRepository struct {
	db *gorm.DB
}

func NewModerationRepository(db *gorm.DB) ModerationRepository {
	return &moderationRepository{db: db}
}

func (r *moderationRepository) CreateLog(log *model.ModerationLog) error {
	return r.db.Create(log).Error
}

func (r *moderationRepository) GetLogs(limit int) ([]*model.ModerationLog, error) {
	var logs []*model.ModerationLog
	q := r.db.Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&logs).Error
	return logs, err
}

func (r *moderationRepository) GetLogsByCommunity(communityID uint, limit int) ([]*model.ModerationLog, error) {
	var logs []*model.ModerationLog
	q := r.db.Where("community_id = ?", communityID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&logs).Error
	return logs, err
}

func (r *moderationRepository) GetLogsByTarget(targetType string, targetID uint) ([]*model.ModerationLog, error) {
	var logs []*model.ModerationLog
	err := r.db.Where("target_type = ? AND target_id = ?", targetType, targetID).
		Order("created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *moderationRepository) CreateReport(report *model.Report) error {
	return r.db.Create(report).Error
}

func (r *moderationRepository) GetReports() ([]*model.Report, error) {
	var reports []*model.Report
	err := r.db.Order("created_at DESC").Find(&reports).Error
	return reports, err
}

func (r *moderationRepository) GetReportByID(id uint) (*model.Report, error) {
	var report model.Report
	err := r.db.First(&report, id).Error
	return &report, err
}

func (r *moderationRepository) UpdateReport(report *model.Report) error {
	return r.db.Save(report).Error
}
