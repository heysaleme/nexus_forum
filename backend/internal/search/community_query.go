package search

import (
	"strings"

	"gorm.io/gorm"
)

// CommunityIDsMatching finds communities by Unicode-aware name/description search.
func CommunityIDsMatching(db *gorm.DB, query string, limit int) ([]uint, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 30
	}
	if PostgresFTSEnabled(db) {
		return PostgresCommunityIDs(db, query, limit)
	}
	norm := NormalizeFold(query)
	like := "%" + escapeLike(norm) + "%"
	prefix := escapeLike(norm) + "%"

	var ids []uint
	_ = db.Table("communities").
		Select("id").
		Where("name LIKE ? OR description LIKE ? OR LOWER(name) LIKE ? OR LOWER(description) LIKE ?",
			like, like, like, like).
		Limit(limit).
		Pluck("id", &ids).Error

	if len(ids) < limit && len(norm) >= 2 {
		var extra []uint
		_ = db.Table("communities").
			Select("id").
			Where("name LIKE ? OR LOWER(name) LIKE ?", prefix, prefix).
			Limit(limit).
			Pluck("id", &extra).Error
		ids = appendUnique(ids, extra...)
	}

	if len(ids) < limit && len(norm) >= 2 {
		type row struct {
			ID          uint
			Name        string
			Description string
		}
		var rows []row
		_ = db.Table("communities").Select("id, name, description").Limit(200).Scan(&rows).Error
		for _, r := range rows {
			blob := NormalizeFold(r.Name + " " + r.Description)
			if strings.Contains(blob, norm) {
				ids = appendUnique(ids, r.ID)
				if len(ids) >= limit {
					break
				}
			}
		}
	}

	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}
