package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucket, key), nil
}
