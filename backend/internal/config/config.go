package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	JWTSecret   string
	DBType      string // "postgres" or "sqlite"
	DatabaseURL string // PostgreSQL URL (e.g. postgres://user:pass@host:port/dbname)
	SqliteDB    string // SQLite filename (e.g. nexus_forum.db)
}

func LoadConfig() (*Config, error) {
	// Try loading .env file if it exists, ignore error if it does not
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "nexus-super-secret-key-1234567890"
	}

	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "sqlite" // Default to sqlite for quick local start, but we can set DB_TYPE=postgres
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "host=localhost user=postgres password=postgres dbname=nexus_forum port=5432 sslmode=disable"
	}

	sqliteDB := os.Getenv("SQLITE_DB")
	if sqliteDB == "" {
		sqliteDB = "nexus_forum.db"
	}

	return &Config{
		Port:        port,
		JWTSecret:   jwtSecret,
		DBType:      dbType,
		DatabaseURL: databaseURL,
		SqliteDB:    sqliteDB,
	}, nil
}
