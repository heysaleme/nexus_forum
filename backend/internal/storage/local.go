package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalStore struct {
	baseDir   string
	publicURL string
}

func NewLocalStore(baseDir, publicURL string) (*LocalStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	publicURL = strings.TrimRight(publicURL, "/")
	return &LocalStore{baseDir: baseDir, publicURL: publicURL}, nil
}

func (s *LocalStore) Backend() Backend { return BackendLocal }

func (s *LocalStore) Put(ctx context.Context, key string, reader io.Reader, _ int64, _ string) (string, error) {
	_ = ctx
	key = strings.TrimPrefix(key, "/")
	dstPath := filepath.Join(s.baseDir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return "", err
	}

	tmpPath := dstPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, reader); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, dstPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	return fmt.Sprintf("%s/uploads/%s", s.publicURL, key), nil
}

// SafeObjectKey builds a unique object key under a category prefix.
func SafeObjectKey(category, originalName string) string {
	safeName := filepath.Base(originalName)
	safeName = strings.ReplaceAll(safeName, " ", "_")
	return fmt.Sprintf("%s/%d_%s", strings.Trim(category, "/"), time.Now().UnixNano(), safeName)
}
