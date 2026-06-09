package search

import (
	"encoding/json"
	"strings"

	"gorm.io/gorm"
)

// SyncPostIndex rebuilds search_blob and token rows for a published post.
func SyncPostIndex(db *gorm.DB, postID uint, title, content, tags string) error {
	if postID == 0 {
		return nil
	}
	tagText := tags
	var tagList []string
	if strings.TrimSpace(tags) != "" {
		_ = json.Unmarshal([]byte(tags), &tagList)
		if len(tagList) > 0 {
			tagText = strings.Join(tagList, " ")
		}
	}

	blob := BuildBlob(title, content, tagText)
	tokens, stems := IndexTerms(title, content, tagText)

	if err := db.Table("posts").Where("id = ?", postID).Update("search_blob", blob).Error; err != nil {
		return err
	}

	if err := db.Exec("DELETE FROM post_search_tokens WHERE post_id = ?", postID).Error; err != nil {
		return err
	}

	type row struct {
		PostID uint
		Token  string
		Stem   string
		Kind   string
	}
	rows := make([]row, 0, len(tokens)+len(stems))
	for _, t := range tokens {
		rows = append(rows, row{PostID: postID, Token: t, Stem: StemToken(t), Kind: "token"})
	}
	for _, s := range stems {
		rows = append(rows, row{PostID: postID, Token: s, Stem: s, Kind: "stem"})
	}
	if len(rows) == 0 {
		return nil
	}
	return db.Table("post_search_tokens").Create(&rows).Error
}

func DeletePostIndex(db *gorm.DB, postID uint) error {
	_ = db.Exec("DELETE FROM post_search_tokens WHERE post_id = ?", postID)
	return db.Table("posts").Where("id = ?", postID).Update("search_blob", "").Error
}

// ReindexPublishedPosts rebuilds search indexes for all published posts.
func ReindexPublishedPosts(db *gorm.DB) error {
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
		if err := SyncPostIndex(db, p.ID, p.Title, p.Content, p.Tags); err != nil {
			return err
		}
	}
	return nil
}
