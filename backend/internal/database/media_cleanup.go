package database

import (
	"encoding/json"
	"log/slog"
	"strings"

	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

// PurgeLegacyBase64Media removes inline data URLs from DB columns so API payloads stay small.
// New uploads must use /api/upload (MinIO or local storage).
func PurgeLegacyBase64Media(db *gorm.DB) {
	clearedUsers := db.Model(&model.User{}).Where("avatar_url LIKE ?", "data:%").Update("avatar_url", "").RowsAffected
	clearedUsers += db.Model(&model.User{}).Where("banner_url LIKE ?", "data:%").Update("banner_url", "").RowsAffected

	clearedCommunities := db.Model(&model.Community{}).Where("avatar_url LIKE ?", "data:%").Update("avatar_url", "").RowsAffected
	clearedCommunities += db.Model(&model.Community{}).Where("banner_url LIKE ?", "data:%").Update("banner_url", "").RowsAffected

	var posts []model.Post
	db.Select("id", "media_urls").Find(&posts)
	clearedPosts := int64(0)
	for _, post := range posts {
		if !strings.Contains(post.MediaUrls, "data:") {
			continue
		}
		cleaned := filterURLListJSON(post.MediaUrls)
		if cleaned != post.MediaUrls {
			db.Model(&model.Post{}).Where("id = ?", post.ID).Update("media_urls", cleaned)
			clearedPosts++
		}
	}

	if clearedUsers+clearedCommunities+clearedPosts > 0 {
		slog.Info("purged legacy base64 media from database",
			"users", clearedUsers,
			"communities", clearedCommunities,
			"posts", clearedPosts,
		)
	}
}

func filterURLListJSON(raw string) string {
	if raw == "" {
		return "[]"
	}
	var urls []string
	if err := json.Unmarshal([]byte(raw), &urls); err != nil {
		if strings.HasPrefix(raw, "data:") {
			return "[]"
		}
		return raw
	}
	filtered := make([]string, 0, len(urls))
	for _, u := range urls {
		if u != "" && !strings.HasPrefix(u, "data:") {
			filtered = append(filtered, u)
		}
	}
	out, err := json.Marshal(filtered)
	if err != nil {
		return "[]"
	}
	return string(out)
}
