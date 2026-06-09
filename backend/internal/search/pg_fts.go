package search

import (
	"strings"

	"gorm.io/gorm"
)

// PostgresFTSEnabled is true when the active dialect is postgres.
func PostgresFTSEnabled(db *gorm.DB) bool {
	return db != nil && db.Dialector.Name() == "postgres"
}

func pgConfig() string {
	// 'simple' works for mixed Latin + Cyrillic without language-specific stemming issues.
	return "simple"
}

// InitPostgresSearch creates GIN indexes for full-text search (idempotent).
func InitPostgresSearch(db *gorm.DB) error {
	if !PostgresFTSEnabled(db) {
		return nil
	}
	stmts := []string{
		`CREATE INDEX IF NOT EXISTS idx_posts_fts ON posts USING GIN (to_tsvector('simple', coalesce(title,'') || ' ' || coalesce(content,'') || ' ' || coalesce(tags,'')))`,
		`CREATE INDEX IF NOT EXISTS idx_comments_fts ON comments USING GIN (to_tsvector('simple', coalesce(content,'')))`,
		`CREATE INDEX IF NOT EXISTS idx_users_fts ON users USING GIN (to_tsvector('simple', coalesce(username,'') || ' ' || coalesce(bio,'') || ' ' || coalesce(email,'')))`,
		`CREATE INDEX IF NOT EXISTS idx_communities_fts ON communities USING GIN (to_tsvector('simple', coalesce(name,'') || ' ' || coalesce(description,'')))`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

func pgIDs(db *gorm.DB, table, extraWhere, vectorExpr, query string, limit int) ([]uint, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 30
	}
	cfg := pgConfig()
	type row struct{ ID uint }
	var rows []row
	sql := `SELECT id FROM ` + table + ` WHERE ` + vectorExpr + ` @@ plainto_tsquery(?, ?)`
	args := []interface{}{cfg, query}
	if extraWhere != "" {
		sql += ` AND ` + extraWhere
	}
	sql += ` ORDER BY ts_rank(` + vectorExpr + `, plainto_tsquery(?, ?)) DESC LIMIT ?`
	args = append(args, cfg, query, limit)
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids, nil
}

func PostgresPostIDs(db *gorm.DB, query string, limit int) ([]uint, error) {
	vec := `to_tsvector('simple', coalesce(title,'') || ' ' || coalesce(content,'') || ' ' || coalesce(tags,''))`
	return pgIDs(db, "posts", "status = 'published'", vec, query, limit)
}

func PostgresCommentIDs(db *gorm.DB, query string, limit int) ([]uint, error) {
	vec := `to_tsvector('simple', coalesce(content,''))`
	return pgIDs(db, "comments", "is_deleted = false", vec, query, limit)
}

func PostgresUserIDs(db *gorm.DB, query string, limit int) ([]uint, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 30
	}
	norm := NormalizeFold(query)
	like := "%" + escapeLike(norm) + "%"

	var ids []uint
	err := db.Table("users").
		Select("id").
		Where("username ILIKE ? OR email ILIKE ? OR bio ILIKE ?", like, like, like).
		Limit(limit).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}

	// FTS boost: merge ranked full-word matches when ILIKE results are sparse.
	if len(ids) < limit {
		vec := `to_tsvector('simple', coalesce(username,'') || ' ' || coalesce(bio,'') || ' ' || coalesce(email,''))`
		ftsIDs, ftsErr := pgIDs(db, "users", "", vec, query, limit)
		if ftsErr == nil {
			ids = appendUnique(ids, ftsIDs...)
		}
	}
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func PostgresCommunityIDs(db *gorm.DB, query string, limit int) ([]uint, error) {
	vec := `to_tsvector('simple', coalesce(name,'') || ' ' || coalesce(description,''))`
	return pgIDs(db, "communities", "", vec, query, limit)
}
