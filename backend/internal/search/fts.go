package search

import (
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"
)

var ftsEnabled bool
var ftsMu sync.RWMutex

func FTSEnabled() bool {
	ftsMu.RLock()
	defer ftsMu.RUnlock()
	return ftsEnabled
}

// Init creates SQLite FTS5 virtual tables used for full-text search.
func ftsUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "fts5") || strings.Contains(msg, "no such module")
}

func Init(db *gorm.DB) error {
	if db.Dialector.Name() != "sqlite" {
		return nil
	}
	if err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS posts_fts USING fts5(title, content, tags, tokenize='unicode61')`).Error; err != nil {
		if ftsUnavailable(err) {
			return nil
		}
		return err
	}
	if err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS users_fts USING fts5(username, bio, email, tokenize='unicode61')`).Error; err != nil {
		if ftsUnavailable(err) {
			return nil
		}
		return err
	}
	if err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS communities_fts USING fts5(name, description, tokenize='unicode61')`).Error; err != nil {
		if ftsUnavailable(err) {
			return nil
		}
		return err
	}
	ftsMu.Lock()
	ftsEnabled = true
	ftsMu.Unlock()
	return nil
}

// ReindexAll rebuilds FTS tables from primary tables (SQLite only).
func ReindexAll(db *gorm.DB) error {
	if db.Dialector.Name() != "sqlite" || !FTSEnabled() {
		return nil
	}
	_ = db.Exec("DELETE FROM posts_fts")
	_ = db.Exec("DELETE FROM users_fts")
	_ = db.Exec("DELETE FROM communities_fts")

	type postRow struct {
		ID      uint
		Title   string
		Content string
		Tags    string
	}
	var posts []postRow
	if err := db.Table("posts").Select("id, title, content, tags").Where("status = ?", "published").Find(&posts).Error; err != nil {
		return err
	}
	for _, p := range posts {
		if err := IndexPost(db, p.ID, p.Title, p.Content, p.Tags); err != nil {
			return err
		}
	}

	type userRow struct {
		ID       uint
		Username string
		Bio      string
		Email    string
	}
	var users []userRow
	if err := db.Table("users").Select("id, username, bio, email").Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		if err := IndexUser(db, u.ID, u.Username, u.Bio, u.Email); err != nil {
			return err
		}
	}

	type commRow struct {
		ID          uint
		Name        string
		Description string
	}
	var comms []commRow
	if err := db.Table("communities").Select("id, name, description").Find(&comms).Error; err != nil {
		return err
	}
	for _, c := range comms {
		if err := IndexCommunity(db, c.ID, c.Name, c.Description); err != nil {
			return err
		}
	}
	return nil
}

func IndexPost(db *gorm.DB, id uint, title, content, tags string) error {
	if db.Dialector.Name() != "sqlite" {
		return nil
	}
	_ = db.Exec("DELETE FROM posts_fts WHERE rowid = ?", id)
	err := db.Exec(
		"INSERT INTO posts_fts(rowid, title, content, tags) VALUES (?, ?, ?, ?)",
		id, title, content, tags,
	).Error
	if ftsUnavailable(err) || strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return nil
	}
	return err
}

func DeletePost(db *gorm.DB, id uint) error {
	if db.Dialector.Name() != "sqlite" {
		return nil
	}
	return db.Exec("DELETE FROM posts_fts WHERE rowid = ?", id).Error
}

func IndexUser(db *gorm.DB, id uint, username, bio, email string) error {
	if db.Dialector.Name() != "sqlite" {
		return nil
	}
	_ = db.Exec("DELETE FROM users_fts WHERE rowid = ?", id)
	return db.Exec(
		"INSERT INTO users_fts(rowid, username, bio, email) VALUES (?, ?, ?, ?)",
		id, username, bio, email,
	).Error
}

func IndexCommunity(db *gorm.DB, id uint, name, description string) error {
	if db.Dialector.Name() != "sqlite" {
		return nil
	}
	_ = db.Exec("DELETE FROM communities_fts WHERE rowid = ?", id)
	return db.Exec(
		"INSERT INTO communities_fts(rowid, name, description) VALUES (?, ?, ?)",
		id, name, description,
	).Error
}

// BuildFTSQuery turns user input into an FTS5 MATCH expression with prefix support.
func BuildFTSQuery(query string) string {
	terms := strings.Fields(strings.TrimSpace(query))
	if len(terms) == 0 {
		return ""
	}
	parts := make([]string, 0, len(terms))
	for _, term := range terms {
		clean := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '_' || r > 127 {
				return r
			}
			return -1
		}, term)
		if clean == "" {
			continue
		}
		escaped := strings.ReplaceAll(clean, `"`, `""`)
		parts = append(parts, fmt.Sprintf(`"%s"*`, escaped))
	}
	return strings.Join(parts, " OR ")
}
