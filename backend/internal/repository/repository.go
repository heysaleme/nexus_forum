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

func IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
