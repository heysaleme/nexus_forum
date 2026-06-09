package service

import (
	"nexus-forum-backend/internal/repository"
)

type Achievement struct {
	ID                      string `json:"id"`
	UserID                  uint   `json:"user_id"`
	AchievementName         string `json:"achievement_name"`
	AchievementDescription  string `json:"achievement_description"`
	Tier                    string `json:"tier"`
}

func computeAchievements(userID uint, stats *repository.ProfileStats, ownedCommunities int64, level int, xp int) []Achievement {
	var out []Achievement
	if stats.PostsCount >= 1 {
		out = append(out, Achievement{ID: "first_post", UserID: userID, AchievementName: "first_post", AchievementDescription: "Первый пост", Tier: "silver"})
	}
	if stats.CommentsCount >= 1 {
		out = append(out, Achievement{ID: "first_comment", UserID: userID, AchievementName: "first_comment", AchievementDescription: "Первый комментарий", Tier: "bronze"})
	}
	if ownedCommunities >= 1 {
		out = append(out, Achievement{ID: "community_builder", UserID: userID, AchievementName: "community_builder", AchievementDescription: "Создатель сообщества", Tier: "gold"})
	}
	if stats.FollowersCount >= 5 {
		out = append(out, Achievement{ID: "rising_star", UserID: userID, AchievementName: "rising_star", AchievementDescription: "5+ подписчиков", Tier: "gold"})
	}
	if level >= 3 {
		out = append(out, Achievement{ID: "veteran", UserID: userID, AchievementName: "veteran", AchievementDescription: "Уровень 3+", Tier: "silver"})
	}
	if xp >= 100 {
		out = append(out, Achievement{ID: "top_contributor", UserID: userID, AchievementName: "top_contributor", AchievementDescription: "100+ XP", Tier: "gold"})
	}
	return out
}
