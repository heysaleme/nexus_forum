package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	JWTSecret   string
	DBType      string // "postgres" or "sqlite"
	DatabaseURL string // PostgreSQL URL (e.g. postgres://user:pass@host:port/dbname)
	SqliteDB    string // SQLite filename (e.g. nexus_forum.db)

	// Google OAuth (optional — features disabled when empty)
	GoogleClientID     string
	GoogleClientSecret string

	// GitHub OAuth (optional — features disabled when empty)
	GithubClientID     string
	GithubClientSecret string

	FrontendURL string // e.g. http://localhost:5173 — used to build OAuth redirect URIs

	// Cloudflare Turnstile (optional — CAPTCHA skipped when empty)
	TurnstileSecret string

	// Object storage (MinIO S3-compatible; falls back to local ./uploads)
	MinIOEndpoint   string
	MinIOAccessKey  string
	MinIOSecretKey  string
	MinIOBucket     string
	MinIOUseSSL     bool
	MinIOPublicURL  string // e.g. http://localhost:9000
	PublicURL       string // API public origin for local file URLs, e.g. http://localhost:8080
	LocalUploadDir  string
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

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	return &Config{
		Port:               port,
		JWTSecret:          jwtSecret,
		DBType:             dbType,
		DatabaseURL:        databaseURL,
		SqliteDB:           sqliteDB,
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		FrontendURL:        frontendURL,
		TurnstileSecret:    os.Getenv("CLOUDFLARE_TURNSTILE_SECRET"),
		MinIOEndpoint:      os.Getenv("MINIO_ENDPOINT"),
		MinIOAccessKey:     os.Getenv("MINIO_ACCESS_KEY"),
		MinIOSecretKey:     os.Getenv("MINIO_SECRET_KEY"),
		MinIOBucket:        envOr("MINIO_BUCKET", "nexus-forum"),
		MinIOUseSSL:        strings.EqualFold(os.Getenv("MINIO_USE_SSL"), "true"),
		MinIOPublicURL:     envOr("MINIO_PUBLIC_URL", "http://localhost:9000"),
		PublicURL:          envOr("PUBLIC_URL", "http://localhost:"+port),
		LocalUploadDir:     envOr("LOCAL_UPLOAD_DIR", "./uploads"),
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}


