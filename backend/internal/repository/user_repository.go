package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *model.User) error
	GetByID(id uint) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	Update(user *model.User) error
	List(sortSpec string, limit int) ([]*model.User, error)
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
