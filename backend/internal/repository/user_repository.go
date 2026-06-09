package repository

import (
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/search"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *model.User) error
	GetByID(id uint) (*model.User, error)
	GetByIDs(ids []uint) (map[uint]*model.User, error)
	GetByEmail(email string) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	GetByOAuth(provider, subject string) (*model.User, error)
	Update(user *model.User) error
	List(sortSpec string, limit int) ([]*model.User, error)
	Search(query string, limit int) ([]*model.User, error)
	GetProfileStats(userID uint) (*ProfileStats, error)
	SyncFollowCounts(userID uint) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	return &user, err
}

func (r *userRepository) GetByIDs(ids []uint) (map[uint]*model.User, error) {
	result := make(map[uint]*model.User)
	if len(ids) == 0 {
		return result, nil
	}
	var users []*model.User
	if err := r.db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		result[user.ID] = user
	}
	return result, nil
}

func (r *userRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("LOWER(email) = LOWER(?)", email).First(&user).Error
	return &user, err
}

func (r *userRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("LOWER(username) = LOWER(?)", username).First(&user).Error
	return &user, err
}

func (r *userRepository) GetByOAuth(provider, subject string) (*model.User, error) {
	var user model.User
	err := r.db.Where("oauth_provider = ? AND oauth_subject = ?", provider, subject).First(&user).Error
	return &user, err
}

func (r *userRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) List(sortSpec string, limit int) ([]*model.User, error) {
	var users []*model.User
	q := r.db.Order(parseSort(sortSpec))
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&users).Error
	return users, err
}

func (r *userRepository) Search(query string, limit int) ([]*model.User, error) {
	ids, err := search.UserIDsMatching(r.db, query, limit)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []*model.User{}, nil
	}
	var users []*model.User
	err = r.db.Where("id IN ?", ids).Find(&users).Error
	return users, err
}
