package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"nexus-forum-backend/internal/storage"
)

// Upload categories — single namespace for all media types.
const (
	UploadProfileAvatar    = "profile/avatars"
	UploadProfileBanner    = "profile/banners"
	UploadCommunityAvatar  = "community/avatars"
	UploadCommunityBanner  = "community/banners"
	UploadPostImage        = "posts/images"
	UploadPostVideo        = "posts/videos"
	UploadChatAttachment   = "chat/attachments"
)

var allowedMIMETypes = map[string]bool{
	"image/png":                true,
	"image/jpeg":               true,
	"image/gif":                true,
	"image/webp":               true,
	"video/mp4":                true,
	"video/webm":               true,
	"video/quicktime":          true,
	"application/pdf":          true,
	"text/plain":               true,
	"application/zip":          true,
	"application/x-zip-compressed": true,
}

var categoryMaxBytes = map[string]int64{
	UploadProfileAvatar:   5 * 1024 * 1024,
	UploadProfileBanner:   8 * 1024 * 1024,
	UploadCommunityAvatar: 5 * 1024 * 1024,
	UploadCommunityBanner: 8 * 1024 * 1024,
	UploadPostImage:       10 * 1024 * 1024,
	UploadPostVideo:       50 * 1024 * 1024,
	UploadChatAttachment:  10 * 1024 * 1024,
}

type UploadService interface {
	Upload(ctx context.Context, category, originalFilename string, reader io.Reader, size int64, sniff []byte) (url string, mimeType string, err error)
	Backend() storage.Backend
}

type uploadService struct {
	store storage.ObjectStore
}

func NewUploadService(store storage.ObjectStore) UploadService {
	return &uploadService{store: store}
}

func (s *uploadService) Backend() storage.Backend {
	return s.store.Backend()
}

func (s *uploadService) Upload(ctx context.Context, category, originalFilename string, reader io.Reader, size int64, sniff []byte) (string, string, error) {
	category = normalizeCategory(category)
	maxSize, ok := categoryMaxBytes[category]
	if !ok {
		return "", "", fmt.Errorf("unknown upload category: %s", category)
	}
	if size > maxSize {
		return "", "", fmt.Errorf("file size exceeds %dMB limit for %s", maxSize/(1024*1024), category)
	}

	mimeType := http.DetectContentType(sniff)
	cleanMime := strings.Split(mimeType, ";")[0]
	if !allowedMIMETypes[cleanMime] {
		return "", "", fmt.Errorf("invalid file type: %s", cleanMime)
	}

	if category == UploadPostVideo && !strings.HasPrefix(cleanMime, "video/") {
		return "", "", fmt.Errorf("category %s requires a video file", category)
	}
	if category == UploadPostImage && !strings.HasPrefix(cleanMime, "image/") {
		return "", "", fmt.Errorf("category %s requires an image file", category)
	}
	if (category == UploadProfileAvatar || category == UploadCommunityAvatar) && !strings.HasPrefix(cleanMime, "image/") {
		return "", "", fmt.Errorf("avatars must be images")
	}
	if (category == UploadProfileBanner || category == UploadCommunityBanner) && !strings.HasPrefix(cleanMime, "image/") {
		return "", "", fmt.Errorf("banners must be images")
	}

	key := storage.SafeObjectKey(category, originalFilename)
	url, err := s.store.Put(ctx, key, reader, size, cleanMime)
	if err != nil {
		return "", "", err
	}
	return url, cleanMime, nil
}

func normalizeCategory(category string) string {
	category = strings.TrimSpace(category)
	switch category {
	case UploadProfileAvatar, UploadProfileBanner,
		UploadCommunityAvatar, UploadCommunityBanner,
		UploadPostImage, UploadPostVideo, UploadChatAttachment:
		return category
	case "avatar", "profile_avatar":
		return UploadProfileAvatar
	case "banner", "profile_banner":
		return UploadProfileBanner
	case "community_avatar":
		return UploadCommunityAvatar
	case "community_banner":
		return UploadCommunityBanner
	case "post_image", "image":
		return UploadPostImage
	case "post_video", "video":
		return UploadPostVideo
	case "chat", "attachment", "":
		return UploadChatAttachment
	default:
		if strings.HasPrefix(category, "posts/") || strings.HasPrefix(category, "profile/") || strings.HasPrefix(category, "community/") || strings.HasPrefix(category, "chat/") {
			return category
		}
		return UploadChatAttachment
	}
}

func ExtensionForMIME(mime string) string {
	switch strings.Split(mime, ";")[0] {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	default:
		return filepath.Ext(mime)
	}
}
