package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"nexus-forum-backend/internal/config"
	"nexus-forum-backend/internal/demo"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/search"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	dbFile := cfg.SqliteDB
	if dbFile == "" {
		dbFile = "nexus_forum.db"
	}
	backup := fmt.Sprintf("%s.backup-%s", dbFile, time.Now().Format("20060102-150405"))
	if err := copyFile(dbFile, backup); err != nil {
		log.Fatalf("backup failed: %v", err)
	}
	log.Printf("backup created: %s", backup)

	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{}, &model.UserFollow{}, &model.Community{}, &model.CommunityMember{},
		&model.Post{}, &model.Comment{}, &model.Vote{}, &model.SavedPost{}, &model.Notification{},
		&model.ChatRoom{}, &model.Message{}, &model.ModerationLog{}, &model.AnalyticsEvent{},
		&model.Report{}, &model.PasswordResetToken{}, &model.RefreshToken{}, &model.EmailVerification{},
		&model.PostSearchToken{},
	); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	_ = search.Init(db)

	if err := demo.ResetMinimalDemo(db); err != nil {
		log.Fatalf("reset: %v", err)
	}
	log.Println("demo cleanup finished successfully")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil && !os.IsExist(err) {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
