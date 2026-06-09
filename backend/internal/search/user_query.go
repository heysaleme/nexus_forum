package search

import (
	"strings"

	"gorm.io/gorm"
)

// UserIDsMatching finds users by Unicode-aware username/bio/email search.
func UserIDsMatching(db *gorm.DB, query string, limit int) ([]uint, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 30
	}
	if PostgresFTSEnabled(db) {
		return PostgresUserIDs(db, query, limit)
	}
	norm := NormalizeFold(query)
	like := "%" + escapeLike(norm) + "%"
	prefix := escapeLike(norm) + "%"
	stem := StemToken(norm)

	var ids []uint
	_ = db.Table("users").
		Select("id").
		Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ? OR LOWER(bio) LIKE ? OR username LIKE ? OR email LIKE ? OR bio LIKE ?",
			like, like, like, like, like, like).
		Limit(limit).
		Pluck("id", &ids).Error

	if len(ids) < limit && len(norm) >= 2 {
		var extra []uint
		_ = db.Table("users").
			Select("id").
			Where("LOWER(username) LIKE ? OR LOWER(username) = ?", prefix, stem).
			Limit(limit).
			Pluck("id", &extra).Error
		ids = appendUnique(ids, extra...)
	}
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}
