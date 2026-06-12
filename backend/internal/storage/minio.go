package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	"nexus-forum-backend/internal/resilience"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStore struct {
	client        *minio.Client // internal endpoint (e.g. minio:9000 inside Docker)
	presignClient *minio.Client // browser-reachable host from MINIO_PUBLIC_URL
	bucket        string
	publicURL     string
}

func NewMinIOStore(endpoint, accessKey, secretKey, bucket, publicURL string, useSSL bool) (*MinIOStore, error) {
	creds := credentials.NewStaticV4(accessKey, secretKey, "")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  creds,
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
	presignClient := client
	if pubHost, pubSSL, ok := publicEndpointFromURL(publicURL); ok && pubHost != endpoint {
		// Region must be set so presigning does not dial the public host from inside Docker
		// (bucket location lookup). MinIO defaults to us-east-1.
		pc, perr := minio.New(pubHost, &minio.Options{
			Creds:  creds,
			Secure: pubSSL,
			Region: "us-east-1",
		})
		if perr != nil {
			return nil, fmt.Errorf("minio presign client (%s): %w", pubHost, perr)
		}
		presignClient = pc
		slog.Info("minio presign host configured", "internal", endpoint, "public", pubHost)
	}

	return &MinIOStore{
		client:        client,
		presignClient: presignClient,
		bucket:        bucket,
		publicURL:     publicURL,
	}, nil
}

// publicEndpointFromURL extracts host:port and TLS flag from MINIO_PUBLIC_URL.
func publicEndpointFromURL(publicURL string) (host string, secure bool, ok bool) {
	u, err := url.Parse(publicURL)
	if err != nil || u.Host == "" {
		return "", false, false
	}
	return u.Host, u.Scheme == "https", true
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
	reference = CanonicalMediaReference(reference)
	if reference == "" {
		return "", false
	}

	if u, err := url.Parse(reference); err == nil && u.Path != "" {
		path := strings.TrimPrefix(u.Path, "/")
		if strings.HasPrefix(path, s.bucket+"/") {
			key := strings.TrimPrefix(path, s.bucket+"/")
			return key, key != ""
		}
	}

	prefix := s.publicURL + "/" + s.bucket + "/"
	if strings.HasPrefix(reference, prefix) {
		key := strings.TrimPrefix(reference, prefix)
		if i := strings.IndexAny(key, "?#"); i >= 0 {
			key = key[:i]
		}
		return key, key != ""
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
	presigned, err := s.presignClient.PresignedGetObject(ctx, s.bucket, key, DefaultPresignExpiry, nil)
	if err != nil {
		return "", fmt.Errorf("presign object %q: %w", key, err)
	}
	return presigned.String(), nil
}
