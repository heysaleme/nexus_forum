package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type KarmaStats struct {
	PostKarma    int
	CommentKarma int
	TotalKarma   int
}

type KarmaRepository interface {
	GetForUser(userID uint) (*KarmaStats, error)
	HydrateUser(user *model.User)
	HydrateUsers(users []*model.User)
}

type karmaRepository struct {
	db *gorm.DB
}

func NewKarmaRepository(db *gorm.DB) KarmaRepository {
	return &karmaRepository{db: db}
}

func (r *karmaRepository) GetForUser(userID uint) (*KarmaStats, error) {
	var postKarma int64
	_ = r.db.Table("posts").Where("author_id = ? AND status = ?", userID, "published").
		Select("COALESCE(SUM(score), 0)").Scan(&postKarma)

	var commentKarma int64
	_ = r.db.Table("comments").Where("author_id = ? AND is_deleted = ?", userID, false).
		Select("COALESCE(SUM(score), 0)").Scan(&commentKarma)

	return &KarmaStats{
		PostKarma:    int(postKarma),
		CommentKarma: int(commentKarma),
		TotalKarma:   int(postKarma + commentKarma),
	}, nil
}

func (r *karmaRepository) HydrateUser(user *model.User) {
	if user == nil {
		return
	}
	stats, err := r.GetForUser(user.ID)
	if err != nil {
		return
	}
	user.PostKarma = stats.PostKarma
	user.CommentKarma = stats.CommentKarma
	user.TotalKarma = stats.TotalKarma
}

func (r *karmaRepository) HydrateUsers(users []*model.User) {
	if len(users) == 0 {
		return
	}
	ids := make([]uint, len(users))
	byID := make(map[uint]*model.User, len(users))
	for i, u := range users {
		ids[i] = u.ID
		byID[u.ID] = u
	}

	type row struct {
		AuthorID uint
		Total    int64
	}
	var postRows []row
	_ = r.db.Table("posts").Select("author_id, COALESCE(SUM(score),0) as total").
		Where("author_id IN ? AND status = ?", ids, "published").Group("author_id").Scan(&postRows)

	var commentRows []row
	_ = r.db.Table("comments").Select("author_id, COALESCE(SUM(score),0) as total").
		Where("author_id IN ? AND is_deleted = ?", ids, false).Group("author_id").Scan(&commentRows)

	postMap := map[uint]int{}
	for _, row := range postRows {
		postMap[row.AuthorID] = int(row.Total)
	}
	commentMap := map[uint]int{}
	for _, row := range commentRows {
		commentMap[row.AuthorID] = int(row.Total)
	}

	for id, u := range byID {
		pk := postMap[id]
		ck := commentMap[id]
		u.PostKarma = pk
		u.CommentKarma = ck
		u.TotalKarma = pk + ck
	}
}
