package storage

import (
	"log/slog"

	"nexus-forum-backend/internal/config"
)

// NewObjectStore uses MinIO when configured and reachable; otherwise local ./uploads.
func NewObjectStore(cfg *config.Config) (ObjectStore, error) {
	local, err := NewLocalStore(cfg.LocalUploadDir, cfg.PublicURL)
	if err != nil {
		return nil, err
	}

	if cfg.MinIOEndpoint == "" || cfg.MinIOAccessKey == "" {
		slog.Info("storage backend: local (MinIO not configured)")
		return local, nil
	}

	minioStore, err := NewMinIOStore(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOPublicURL,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		slog.Warn("minio unavailable, falling back to local uploads", "error", err)
		return local, nil
	}

	slog.Info("storage backend: minio", "endpoint", cfg.MinIOEndpoint, "bucket", cfg.MinIOBucket)
	return minioStore, nil
}
