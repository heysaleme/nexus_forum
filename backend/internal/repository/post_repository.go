package repository

import (
	"strings"
	"time"

	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/search"

	"gorm.io/gorm"
)

type PostRepository interface {
	Create(post *model.Post) error
	GetByID(id uint) (*model.Post, error)
	Update(post *model.Post) error
	Delete(id uint) error
	List(sortSpec string, limit int, viewerID uint) ([]*model.Post, error)
	ListByFollowing(followerID uint, sortSpec string, limit int, viewerID uint) ([]*model.Post, error)
	ListByCommunityMembership(userID uint, sortSpec string, limit int, viewerID uint) ([]*model.Post, error)
	Filter(filter map[string]interface{}, sortSpec string, limit int, viewerID uint) ([]*model.Post, error)
	Search(query string, limit int, viewerID uint) ([]*model.Post, error)
	PublishDueScheduled() ([]*model.Post, error)
	IncrementViews(id uint) error
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(post *model.Post) error {
	if err := r.db.Create(post).Error; err != nil {
		return err
	}
	if post.Status == "published" {
		_ = search.SyncPostIndex(r.db, post.ID, post.Title, post.Content, post.Tags)
		_ = search.IndexPost(r.db, post.ID, post.Title, post.Content, post.Tags)
	}
	return nil
}

func (r *postRepository) GetByID(id uint) (*model.Post, error) {
	var post model.Post
	err := r.db.First(&post, id).Error
	if err != nil {
		return nil, err
	}
	r.hydratePostFields([]*model.Post{&post})
	return &post, nil
}

func (r *postRepository) Update(post *model.Post) error {
	if err := r.db.Save(post).Error; err != nil {
		return err
	}
	if post.Status == "published" {
		_ = search.SyncPostIndex(r.db, post.ID, post.Title, post.Content, post.Tags)
		_ = search.IndexPost(r.db, post.ID, post.Title, post.Content, post.Tags)
	} else {
		_ = search.DeletePostIndex(r.db, post.ID)
		_ = search.DeletePost(r.db, post.ID)
	}
	return nil
}

func (r *postRepository) Delete(id uint) error {
	if err := r.db.Delete(&model.Post{}, id).Error; err != nil {
		return err
	}
	_ = search.DeletePostIndex(r.db, id)
	_ = search.DeletePost(r.db, id)
	return nil
}

func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func (r *postRepository) List(sortSpec string, limit int, viewerID uint) ([]*model.Post, error) {
	var posts []*model.Post
	q := r.db.Where("status = ?", "published").Order(parsePostSort(r.db.Dialector.Name(), sortSpec))
	q = r.applyShadowFilter(q, viewerID)
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
}

func (r *postRepository) ListByCommunityMembership(userID uint, sortSpec string, limit int, viewerID uint) ([]*model.Post, error) {
	var posts []*model.Post
	communitySubquery := r.db.Model(&model.CommunityMember{}).
		Select("community_id").
		Where("user_id = ?", userID)

	q := r.db.Where("status = ?", "published").
		Where("community_id IN (?)", communitySubquery).
		Order(parsePostSort(r.db.Dialector.Name(), sortSpec))
	q = r.applyShadowFilter(q, viewerID)
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
}

func (r *postRepository) ListByFollowing(followerID uint, sortSpec string, limit int, viewerID uint) ([]*model.Post, error) {
	var posts []*model.Post
	followingSubquery := r.db.Model(&model.UserFollow{}).
		Select("following_id").
		Where("follower_id = ? AND status = ?", followerID, "accepted")

	q := r.db.Where("status = ?", "published").
		Where("author_id IN (?)", followingSubquery).
		Order(parsePostSort(r.db.Dialector.Name(), sortSpec))
	q = r.applyShadowFilter(q, viewerID)
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
}

func (r *postRepository) Filter(filter map[string]interface{}, sortSpec string, limit int, viewerID uint) ([]*model.Post, error) {
	var posts []*model.Post
	q := r.db.Where(filter).Order(parsePostSort(r.db.Dialector.Name(), sortSpec))
	q = r.applyShadowFilter(q, viewerID)
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
}

func (r *postRepository) Search(query string, limit int, viewerID uint) ([]*model.Post, error) {
	var posts []*model.Post
	query = strings.TrimSpace(query)
	if query == "" {
		return posts, nil
	}
	if limit <= 0 {
		limit = 30
	}

	ids, err := search.PostIDsMatching(r.db, query, limit*2)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return posts, nil
	}

	q := r.db.Where("status = ? AND id IN ?", "published", ids)
	q = r.applyShadowFilter(q, viewerID)
	if limit > 0 {
		q = q.Limit(limit)
	}
	err = q.Find(&posts).Error
	if err == nil {
		r.hydratePostFields(posts)
	}
	return posts, err
}

func (r *postRepository) PublishDueScheduled() ([]*model.Post, error) {
	now := time.Now()
	var due []*model.Post
	if err := r.db.Where("status = ? AND publish_at IS NOT NULL AND publish_at <= ?", "scheduled", now).Find(&due).Error; err != nil {
		return nil, err
	}
	published := make([]*model.Post, 0, len(due))
	for _, p := range due {
		p.Status = "published"
		p.PublishAt = nil
		if err := r.db.Save(p).Error; err != nil {
			continue
		}
		_ = search.SyncPostIndex(r.db, p.ID, p.Title, p.Content, p.Tags)
		_ = search.IndexPost(r.db, p.ID, p.Title, p.Content, p.Tags)
		published = append(published, p)
	}
	return published, nil
}

func (r *postRepository) IncrementViews(id uint) error {
	return r.db.Model(&model.Post{}).Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + ?", 1)).Error
}

// applyShadowFilter filters out shadow-banned author posts and shadow content
// unless the viewer is the author themselves.
func (r *postRepository) applyShadowFilter(q *gorm.DB, viewerID uint) *gorm.DB {
	// Subquery: get IDs of shadow-banned users
	shadowBannedSubquery := r.db.Model(&model.User{}).Select("id").Where("is_shadow_banned = ?", true)
	if viewerID > 0 {
		// Hide shadow-banned author posts unless viewer IS the author
		q = q.Where("author_id NOT IN (?) OR author_id = ?", shadowBannedSubquery, viewerID)
		// Hide shadow content unless viewer IS the author
		q = q.Where("is_shadow_content = ? OR author_id = ?", false, viewerID)
	} else {
		// Anonymous viewer — hide all shadow content and shadow-banned author posts
		q = q.Where("author_id NOT IN (?)", shadowBannedSubquery)
		q = q.Where("is_shadow_content = ?", false)
	}
	return q
}

func (r *postRepository) hydratePostFields(posts []*model.Post) {
	if len(posts) == 0 {
		return
	}

	authorIDs := make(map[uint]struct{})
	communityIDs := make(map[uint]struct{})
	for _, post := range posts {
		authorIDs[post.AuthorID] = struct{}{}
		communityIDs[post.CommunityID] = struct{}{}
	}

	authorsByID := make(map[uint]model.User)
	if len(authorIDs) > 0 {
		ids := make([]uint, 0, len(authorIDs))
		for id := range authorIDs {
			ids = append(ids, id)
		}
		var authors []model.User
		if err := r.db.Where("id IN ?", ids).Find(&authors).Error; err == nil {
			for _, author := range authors {
				authorsByID[author.ID] = author
			}
		}
	}

	communitiesByID := make(map[uint]model.Community)
	if len(communityIDs) > 0 {
		ids := make([]uint, 0, len(communityIDs))
		for id := range communityIDs {
			ids = append(ids, id)
		}
		var communities []model.Community
		if err := r.db.Where("id IN ?", ids).Find(&communities).Error; err == nil {
			for _, community := range communities {
				communitiesByID[community.ID] = community
			}
		}
	}

	for _, post := range posts {
		if author, ok := authorsByID[post.AuthorID]; ok {
			post.AuthorUsername = author.Username
			post.AuthorAvatar = author.AvatarURL
		}
		if community, ok := communitiesByID[post.CommunityID]; ok {
			post.CommunityName = community.Name
			post.CommunityAvatar = community.AvatarURL
		}
	}

	r.hydrateCommentCounts(posts)
	r.hydrateCrosspostFields(posts)
}

func (r *postRepository) hydrateCrosspostFields(posts []*model.Post) {
	origIDs := make([]uint, 0)
	for _, post := range posts {
		if post.OriginalPostID != nil && *post.OriginalPostID > 0 {
			origIDs = append(origIDs, *post.OriginalPostID)
		}
	}
	if len(origIDs) == 0 {
		return
	}
	var originals []model.Post
	if err := r.db.Where("id IN ?", origIDs).Find(&originals).Error; err != nil {
		return
	}
	byID := make(map[uint]model.Post, len(originals))
	commIDs := make(map[uint]struct{})
	for _, o := range originals {
		byID[o.ID] = o
		commIDs[o.CommunityID] = struct{}{}
	}
	comms := make(map[uint]model.Community)
	if len(commIDs) > 0 {
		ids := make([]uint, 0, len(commIDs))
		for id := range commIDs {
			ids = append(ids, id)
		}
		var rows []model.Community
		if err := r.db.Where("id IN ?", ids).Find(&rows).Error; err == nil {
			for _, c := range rows {
				comms[c.ID] = c
			}
		}
	}
	for _, post := range posts {
		if post.OriginalPostID == nil {
			continue
		}
		if orig, ok := byID[*post.OriginalPostID]; ok {
			post.OriginalPostTitle = orig.Title
			if c, ok := comms[orig.CommunityID]; ok {
				post.OriginalCommunity = c.Name
			}
		}
	}
}

func (r *postRepository) hydrateCommentCounts(posts []*model.Post) {
	if len(posts) == 0 {
		return
	}

	postIDs := make([]uint, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
	}

	type commentCountRow struct {
		PostID uint
		Count  int64
	}
	var rows []commentCountRow
	_ = r.db.Model(&model.Comment{}).
		Select("post_id, COUNT(*) as count").
		Where("post_id IN ?", postIDs).
		Group("post_id").
		Scan(&rows).Error

	countsByPostID := make(map[uint]int, len(rows))
	for _, row := range rows {
		countsByPostID[row.PostID] = int(row.Count)
	}

	for _, post := range posts {
		post.CommentCount = countsByPostID[post.ID]
	}
}
