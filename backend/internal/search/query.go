package search

import (
	"strings"

	"gorm.io/gorm"
)

// PostIDsMatching returns published post IDs matching a Unicode-aware query.
func PostIDsMatching(db *gorm.DB, query string, limit int) ([]uint, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 30
	}
	if PostgresFTSEnabled(db) {
		return PostgresPostIDs(db, query, limit)
	}

	terms := Tokenize(query)
	if len(terms) == 0 {
		return nil, nil
	}

	var ids []uint
	for _, term := range terms {
		if len(term) < 2 {
			continue
		}
		part, err := postIDsMatchingTerm(db, term, limit)
		if err != nil {
			return nil, err
		}
		ids = appendUnique(ids, part...)
		if len(ids) >= limit {
			break
		}
	}

	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func postIDsMatchingTerm(db *gorm.DB, term string, limit int) ([]uint, error) {
	norm := NormalizeFold(term)
	stem := StemToken(norm)
	like := "%" + escapeLike(norm) + "%"
	prefix := escapeLike(norm) + "%"

	var ids []uint

	var blobIDs []uint
	if err := db.Table("posts").
		Select("id").
		Where("status = ? AND search_blob LIKE ? ESCAPE '\\'", "published", like).
		Limit(limit).
		Pluck("id", &blobIDs).Error; err != nil {
		return nil, err
	}
	ids = appendUnique(ids, blobIDs...)

	if len(ids) < limit && (len(norm) >= 2 || len(stem) >= 2) {
		var tokenIDs []uint
		q := db.Table("post_search_tokens").Select("DISTINCT post_id")
		switch {
		case len(stem) >= 2 && len(norm) >= 2:
			q = q.Where("(token LIKE ? ESCAPE '\\' OR token = ? OR stem = ?)", prefix, stem, stem)
		case len(norm) >= 2:
			q = q.Where("token LIKE ? ESCAPE '\\'", prefix)
		default:
			q = q.Where("stem = ?", stem)
		}
		if err := q.Limit(limit).Pluck("post_id", &tokenIDs).Error; err != nil {
			return nil, err
		}
		ids = appendUnique(ids, tokenIDs...)
	}
	return ids, nil
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func appendUnique(dst []uint, src ...uint) []uint {
	seen := make(map[uint]struct{}, len(dst))
	for _, id := range dst {
		seen[id] = struct{}{}
	}
	for _, id := range src {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		dst = append(dst, id)
	}
	return dst
}
