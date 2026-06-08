package handler

import (
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/service"
)

func collectAuthorIDs(posts []*model.Post) []uint {
	seen := make(map[uint]struct{})
	ids := make([]uint, 0, len(posts))
	for _, post := range posts {
		if _, ok := seen[post.AuthorID]; ok {
			continue
		}
		seen[post.AuthorID] = struct{}{}
		ids = append(ids, post.AuthorID)
	}
	return ids
}

func filterPostsByPrivacy(
	posts []*model.Post,
	authors map[uint]*model.User,
	viewerID uint,
	authenticated bool,
	isGeneralFeed bool,
	userService service.UserService,
) []*model.Post {
	visible := make([]*model.Post, 0, len(posts))
	followingCache := make(map[uint]bool)

	for _, post := range posts {
		author := authors[post.AuthorID]
		if author == nil || !author.IsPrivate {
			visible = append(visible, post)
			continue
		}
		if isGeneralFeed {
			continue
		}
		if !authenticated {
			continue
		}
		if viewerID == post.AuthorID {
			visible = append(visible, post)
			continue
		}
		following, cached := followingCache[post.AuthorID]
		if !cached {
			following, _ = userService.IsFollowing(viewerID, post.AuthorID)
			followingCache[post.AuthorID] = following
		}
		if following {
			visible = append(visible, post)
		}
	}
	return visible
}

func applyPostVotes(posts []*model.Post, userID uint, postService service.PostService) {
	if userID == 0 || len(posts) == 0 {
		return
	}
	ids := make([]uint, len(posts))
	for i, post := range posts {
		ids[i] = post.ID
	}
	votes := postService.GetVotesForPosts(userID, ids)
	for _, post := range posts {
		if value, ok := votes[post.ID]; ok {
			post.UserVote = value
		}
	}
}
