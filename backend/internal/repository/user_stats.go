package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

// ProfileStats holds live counts for a user profile.
type ProfileStats struct {
	FollowersCount    int64 `json:"followers_count"`
	FollowingCount    int64 `json:"following_count"`
	CommunitiesCount  int64 `json:"communities_count"`
	CommunitiesOwned  int64 `json:"communities_owned"`
	PostsCount        int64 `json:"posts_count"`
	CommentsCount     int64 `json:"comments_count"`
	SavedCount        int64 `json:"saved_count"`
	AchievementsCount int64 `json:"achievements_count"`
}

func (r *userRepository) GetProfileStats(userID uint) (*ProfileStats, error) {
	var exists int64
	if err := r.db.Model(&model.User{}).Where("id = ?", userID).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	stats := &ProfileStats{}
	db := r.db

	if err := db.Model(&model.UserFollow{}).
		Where("following_id = ? AND status = ?", userID, "accepted").
		Count(&stats.FollowersCount).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.UserFollow{}).
		Where("follower_id = ? AND status = ?", userID, "accepted").
		Count(&stats.FollowingCount).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.CommunityMember{}).
		Where("user_id = ?", userID).
		Count(&stats.CommunitiesCount).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.Community{}).
		Where("owner_id = ?", userID).
		Count(&stats.CommunitiesOwned).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.Post{}).
		Where("author_id = ? AND status = ?", userID, "published").
		Count(&stats.PostsCount).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.Comment{}).
		Where("author_id = ?", userID).
		Count(&stats.CommentsCount).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.SavedPost{}).
		Where("user_id = ?", userID).
		Count(&stats.SavedCount).Error; err != nil {
		return nil, err
	}

	var user model.User
	if err := db.Select("level", "xp").First(&user, userID).Error; err != nil {
		return nil, err
	}
	stats.AchievementsCount = countAchievementRules(stats, user.Level, user.XP)
	return stats, nil
}

func (r *userRepository) SyncFollowCounts(userID uint) error {
	var followers, following int64
	if err := r.db.Model(&model.UserFollow{}).
		Where("following_id = ? AND status = ?", userID, "accepted").
		Count(&followers).Error; err != nil {
		return err
	}
	if err := r.db.Model(&model.UserFollow{}).
		Where("follower_id = ? AND status = ?", userID, "accepted").
		Count(&following).Error; err != nil {
		return err
	}
	return r.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"followers_count": followers,
		"following_count": following,
	}).Error
}

func countAchievementRules(stats *ProfileStats, level, xp int) int64 {
	var n int64
	if stats.PostsCount >= 1 {
		n++
	}
	if stats.CommentsCount >= 1 {
		n++
	}
	if stats.CommunitiesOwned >= 1 {
		n++
	}
	if stats.FollowersCount >= 5 {
		n++
	}
	if level >= 3 {
		n++
	}
	if xp >= 100 {
		n++
	}
	return n
}
