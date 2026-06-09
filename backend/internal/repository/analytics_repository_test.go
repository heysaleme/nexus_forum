package repository

import (
	"fmt"
	"testing"
	"time"

	"nexus-forum-backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAnalyticsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Post{}, &model.Comment{}, &model.Report{}, &model.AnalyticsEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAnalyticsRepository_GetUserGrowth_SQLite(t *testing.T) {
	db := setupAnalyticsDB(t)
	repo := NewAnalyticsRepository(db)
	now := time.Now()
	for i := 0; i < 3; i++ {
		if err := db.Create(&model.User{
			Username:     fmt.Sprintf("user%d", i),
			Email:        fmt.Sprintf("user%d@t.com", i),
			PasswordHash: "x",
			ProfileTheme: "default",
			CreatedAt:    now.AddDate(0, 0, -i),
		}).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	rows, err := repo.GetUserGrowth(7)
	if err != nil {
		t.Fatalf("GetUserGrowth: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected growth rows")
	}
}

func TestAnalyticsRepository_GetActivitySeries_SQLite(t *testing.T) {
	db := setupAnalyticsDB(t)
	repo := NewAnalyticsRepository(db)
	rows, err := repo.GetActivitySeries(7)
	if err != nil {
		t.Fatalf("GetActivitySeries: %v", err)
	}
	if rows == nil {
		t.Fatal("expected non-nil series")
	}
}

func TestAnalyticsRepository_GetRetentionRates_SQLite(t *testing.T) {
	db := setupAnalyticsDB(t)
	repo := NewAnalyticsRepository(db)
	rates, err := repo.GetRetentionRates()
	if err != nil {
		t.Fatalf("GetRetentionRates: %v", err)
	}
	for _, key := range []string{"d1", "d7", "d30"} {
		if _, ok := rates[key]; !ok {
			t.Fatalf("missing retention key %s", key)
		}
	}
}

func TestAnalyticsRepository_GetEngagementStats_SQLite(t *testing.T) {
	db := setupAnalyticsDB(t)
	repo := NewAnalyticsRepository(db)
	stats, err := repo.GetEngagementStats()
	if err != nil {
		t.Fatalf("GetEngagementStats: %v", err)
	}
	if stats == nil {
		t.Fatal("expected engagement stats")
	}
}

func TestAnalyticsRepository_isPostgres(t *testing.T) {
	sqliteDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if NewAnalyticsRepository(sqliteDB).(*analyticsRepository).isPostgres() {
		t.Fatal("sqlite should not be postgres")
	}
}
