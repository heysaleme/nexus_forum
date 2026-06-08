package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// Helper for parsing Sort Specifications
func parseSort(sortSpec string) string {
	if sortSpec == "" {
		return "created_at DESC"
	}
	desc := false
	if strings.HasPrefix(sortSpec, "-") {
		desc = true
		sortSpec = sortSpec[1:]
	}

	var dbField string
	switch sortSpec {
	case "created_date":
		dbField = "created_at"
	case "member_count":
		dbField = "member_count"
	case "views":
		dbField = "views"
	case "score":
		dbField = "score"
	case "last_message_at":
		dbField = "last_message_at"
	case "name":
		dbField = "name"
	default:
		dbField = sortSpec
	}

	if desc {
		return dbField + " DESC"
	}
	return dbField + " ASC"
}

// parsePostSort maps feed sort keys (hot, new, top) to SQL ORDER BY clauses.
func parsePostSort(dialect, sortSpec string) string {
	if sortSpec == "" {
		return "created_at DESC"
	}

	desc := strings.HasPrefix(sortSpec, "-")
	key := sortSpec
	if desc {
		key = sortSpec[1:]
	}

	switch key {
	case "hot":
		return hotOrderClause(dialect, !desc)
	case "new", "created_date", "created_at":
		if desc {
			return "created_at ASC"
		}
		return "created_at DESC"
	case "top", "score":
		if desc {
			return "score ASC"
		}
		return "score DESC"
	default:
		return parseSort(sortSpec)
	}
}

// hotOrderClause implements Reddit-like ranking: score decays as post ages.
func hotOrderClause(dialect string, desc bool) string {
	var expr string
	if dialect == "postgres" {
		expr = "(score::float / POWER(GREATEST(EXTRACT(EPOCH FROM (NOW() - created_at)) / 3600.0 + 2, 2), 1.5))"
	} else {
		age := "MAX((julianday('now') - julianday(created_at)) * 24.0 + 2, 2)"
		expr = "(score * 1.0 / (" + age + " * " + age + "))"
	}
	if desc {
		return expr + " DESC"
	}
	return expr + " ASC"
}

func IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
