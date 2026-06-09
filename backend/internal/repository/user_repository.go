package repository

import (
	"strings"

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
	var users []*model.User
	dialect := r.db.Dialector.Name()
	query = strings.TrimSpace(query)
	var q *gorm.DB
	if dialect == "postgres" {
		q = r.db.Where("to_tsvector('simple', coalesce(username,'') || ' ' || coalesce(bio,'') || ' ' || coalesce(email,'')) @@ plainto_tsquery('simple', ?)", query)
	} else if search.FTSEnabled() {
		ftsQuery := search.BuildFTSQuery(query)
		if ftsQuery != "" {
			sub := r.db.Table("users_fts").Select("rowid").Where("users_fts MATCH ?", ftsQuery)
			q = r.db.Where("id IN (?)", sub)
		} else {
			likePattern := "%" + strings.ToLower(query) + "%"
			q = r.db.Where("LOWER(username) LIKE ? OR LOWER(bio) LIKE ? OR LOWER(email) LIKE ?", likePattern, likePattern, likePattern)
		}
	} else {
		likePattern := "%" + query + "%"
		likeLower := "%" + strings.ToLower(query) + "%"
		q = r.db.Where(
			"username LIKE ? OR bio LIKE ? OR email LIKE ? OR LOWER(username) LIKE ? OR LOWER(email) LIKE ?",
			likePattern, likePattern, likePattern, likeLower, likeLower,
		)
	}

	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&users).Error
	return users, err
}
