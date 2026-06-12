package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"nexus-forum-backend/internal/resilience"
)

type MinIOStore struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func NewMinIOStore(endpoint, accessKey, secretKey, bucket, publicURL string, useSSL bool) (*MinIOStore, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
		slog.Info("minio bucket created", "bucket", bucket)
	}

	publicURL = strings.TrimRight(publicURL, "/")
	return &MinIOStore{client: client, bucket: bucket, publicURL: publicURL}, nil
}

func (s *MinIOStore) Backend() Backend { return BackendMinIO }

func (s *MinIOStore) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	key = strings.TrimPrefix(key, "/")
	_, err := resilience.Execute(resilience.BreakerMinIO, func() (interface{}, error) {
		return s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
			ContentType: contentType,
		})
	})
	if err != nil {
		return "", err
	}
	return ReferenceURL(s.publicURL, s.bucket, key), nil
}

// ReferenceURL builds the canonical stored reference for an object (not presigned).
func ReferenceURL(publicURL, bucket, key string) string {
	publicURL = strings.TrimRight(publicURL, "/")
	key = strings.TrimPrefix(key, "/")
	return fmt.Sprintf("%s/%s/%s", publicURL, bucket, key)
}

// ObjectKeyFromReference extracts the MinIO object key from a stored reference URL.
func (s *MinIOStore) ObjectKeyFromReference(reference string) (string, bool) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", false
	}

	prefix := s.publicURL + "/" + s.bucket + "/"
	if strings.HasPrefix(reference, prefix) {
		return strings.TrimPrefix(reference, prefix), true
	}

	// Alternate host (e.g. minio:9000 internally vs localhost:9000 in browser).
	if u, err := url.Parse(reference); err == nil && u.Path != "" {
		path := strings.TrimPrefix(u.Path, "/")
		if strings.HasPrefix(path, s.bucket+"/") {
			return strings.TrimPrefix(path, s.bucket+"/"), true
		}
	}

	return "", false
}

// AccessibleURL returns a time-limited presigned GET URL for private bucket objects.
func (s *MinIOStore) AccessibleURL(ctx context.Context, reference string) (string, error) {
	key, ok := s.ObjectKeyFromReference(reference)
	if !ok {
		// External URL (picsum, CDN, etc.) — return unchanged.
		return reference, nil
	}
	presigned, err := s.client.PresignedGetObject(ctx, s.bucket, key, DefaultPresignExpiry, nil)
	if err != nil {
		return "", fmt.Errorf("presign object %q: %w", key, err)
	}
	return presigned.String(), nil
}
