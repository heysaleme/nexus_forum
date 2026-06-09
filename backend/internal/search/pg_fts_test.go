package search

import (
	"os"
	"testing"

	"nexus-forum-backend/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresUserIDs_PartialMatch(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	if !PostgresFTSEnabled(db) {
		t.Skip("not postgres dialect")
	}

	u := model.User{Username: "amira_pg_test", Email: "amira_pg_test@example.com", PasswordHash: "x", ProfileTheme: "default"}
	if err := db.Where("username = ?", u.Username).Delete(&model.User{}).Error; err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() { _ = db.Delete(&u).Error })

	for _, q := range []string{"ami", "mir", "AMIRA", "Mir"} {
		ids, err := PostgresUserIDs(db, q, 10)
		if err != nil {
			t.Fatalf("PostgresUserIDs %q: %v", q, err)
		}
		found := false
		for _, id := range ids {
			if id == u.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("query %q: expected user id %d in %v", q, u.ID, ids)
		}
	}
}
