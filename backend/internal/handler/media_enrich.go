package handler

import (
	"context"

	"nexus-forum-backend/internal/model"
)

func (h *Handlers) enrichPost(ctx context.Context, post *model.Post) {
	if post == nil || h.UploadService == nil {
		return
	}
	post.MediaUrls = h.UploadService.ResolveMediaJSON(ctx, post.MediaUrls)
	if url, err := h.UploadService.AccessibleURL(ctx, post.AuthorAvatar); err == nil && url != "" {
		post.AuthorAvatar = url
	}
	if url, err := h.UploadService.AccessibleURL(ctx, post.CommunityAvatar); err == nil && url != "" {
		post.CommunityAvatar = url
	}
}

func (h *Handlers) enrichPosts(ctx context.Context, posts []*model.Post) {
	for _, post := range posts {
		h.enrichPost(ctx, post)
	}
}

func (h *Handlers) enrichUser(ctx context.Context, user *model.User) {
	if user == nil || h.UploadService == nil {
		return
	}
	if url, err := h.UploadService.AccessibleURL(ctx, user.AvatarURL); err == nil && url != "" {
		user.AvatarURL = url
	}
	if url, err := h.UploadService.AccessibleURL(ctx, user.BannerURL); err == nil && url != "" {
		user.BannerURL = url
	}
}
