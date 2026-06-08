package repository

import (
	"strings"
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type CommunityRepository interface {
	Create(community *model.Community) error
	GetByID(id uint) (*model.Community, error)
	GetBySlug(slug string) (*model.Community, error)
	Update(community *model.Community) error
	List(sortSpec string, limit int) ([]*model.Community, error)
	Filter(filter map[string]interface{}) ([]*model.Community, error)
	Search(query string, limit int) ([]*model.Community, error)

	AddMember(member *model.CommunityMember) error
	RemoveMember(userID, communityID uint) error
	GetMember(userID, communityID uint) (*model.CommunityMember, error)
	GetMemberships(userID uint) ([]*model.CommunityMember, error)
	GetMembers(communityID uint) ([]*model.CommunityMember, error)
	Delete(id uint) error
}

type communityRepository struct {
	db *gorm.DB
}

func NewCommunityRepository(db *gorm.DB) CommunityRepository {
	return &communityRepository{db: db}
}

func (r *communityRepository) Create(community *model.Community) error {
	return r.db.Create(community).Error
}

func (r *communityRepository) GetByID(id uint) (*model.Community, error) {
	var community model.Community
	err := r.db.First(&community, id).Error
	return &community, err
}

func (r *communityRepository) GetBySlug(slug string) (*model.Community, error) {
	var community model.Community
	err := r.db.Where("slug = ?", slug).First(&community).Error
	return &community, err
}

func (r *communityRepository) Update(community *model.Community) error {
	return r.db.Save(community).Error
}

func (r *communityRepository) List(sortSpec string, limit int) ([]*model.Community, error) {
	var communities []*model.Community
	q := r.db.Order(parseSort(sortSpec))
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&communities).Error
	return communities, err
}

func (r *communityRepository) Filter(filter map[string]interface{}) ([]*model.Community, error) {
	var communities []*model.Community
	err := r.db.Where(filter).Find(&communities).Error
	return communities, err
}

func (r *communityRepository) Search(query string, limit int) ([]*model.Community, error) {
	var communities []*model.Community
	dialect := r.db.Dialector.Name()
	var q *gorm.DB
	if dialect == "postgres" {
		q = r.db.Where("to_tsvector('simple', name || ' ' || description) @@ plainto_tsquery('simple', ?)", query)
	} else {
		likePattern := "%" + strings.ToLower(query) + "%"
		q = r.db.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", likePattern, likePattern)
	}

	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&communities).Error
	return communities, err
}

func (r *communityRepository) AddMember(member *model.CommunityMember) error {
	return r.db.Create(member).Error
}

func (r *communityRepository) RemoveMember(userID, communityID uint) error {
	return r.db.Where("user_id = ? AND community_id = ?", userID, communityID).Delete(&model.CommunityMember{}).Error
}

func (r *communityRepository) GetMember(userID, communityID uint) (*model.CommunityMember, error) {
	var member model.CommunityMember
	err := r.db.Where("user_id = ? AND community_id = ?", userID, communityID).First(&member).Error
	return &member, err
}

func (r *communityRepository) GetMemberships(userID uint) ([]*model.CommunityMember, error) {
	var members []*model.CommunityMember
	err := r.db.Where("user_id = ?", userID).Find(&members).Error
	return members, err
}

func (r *communityRepository) GetMembers(communityID uint) ([]*model.CommunityMember, error) {
	var members []*model.CommunityMember
	err := r.db.Where("community_id = ?", communityID).Find(&members).Error
	return members, err
}

func (r *communityRepository) Delete(id uint) error {
	return r.db.Delete(&model.Community{}, id).Error
}
