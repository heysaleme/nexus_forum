package storage

import (
	"context"
	"io"
)

// Backend identifies which storage implementation saved an object.
type Backend string

const (
	BackendMinIO Backend = "minio"
	BackendLocal Backend = "local"
)

// ObjectStore persists binary objects and returns a public URL.
type ObjectStore interface {
	Backend() Backend
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (publicURL string, err error)
}
