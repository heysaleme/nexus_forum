package service

import (
	"errors"
	"regexp"
	"strings"

	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

var mentionPattern = regexp.MustCompile(`@([a-zA-Z0-9_\p{L}]+)`)

type CommentService interface {
	Create(comment *model.Comment) error
	Update(userID uint, commentID uint, content string) (*model.Comment, error)
	Delete(userID, commentID uint) error
	GetByID(commentID uint) (*model.Comment, error)
	GetByPostID(postID uint, viewerID uint) ([]*model.Comment, error)
	Vote(userID, commentID uint, value int) error
	GetVote(userID, commentID uint) (*model.Vote, error)
	GetVotesForComments(userID uint, commentIDs []uint) map[uint]int
}

type commentService struct {
	repo      repository.CommentRepository
	userRepo  repository.UserRepository
	postRepo  repository.PostRepository
	voteRepo  repository.VoteRepository
	notifRepo repository.NotificationRepository
	commRepo  repository.CommunityRepository
}

func NewCommentService(
	repo repository.CommentRepository,
	userRepo repository.UserRepository,
	postRepo repository.PostRepository,
	voteRepo repository.VoteRepository,
	notifRepo repository.NotificationRepository,
	commRepo repository.CommunityRepository,
) CommentService {
	return &commentService{
		repo:      repo,
		userRepo:  userRepo,
		postRepo:  postRepo,
		voteRepo:  voteRepo,
		notifRepo: notifRepo,
		commRepo:  commRepo,
	}
}

func (s *commentService) Create(comment *model.Comment) error {
	if comment.ParentID != nil && *comment.ParentID > 0 {
		parent, err := s.repo.GetByID(*comment.ParentID)
		if err != nil {
			return errors.New("parent comment not found")
		}
		if parent.PostID != comment.PostID {
			return errors.New("parent comment belongs to a different post")
		}
		replyToID := parent.AuthorID
		comment.ReplyToUserID = &replyToID
	}

	err := s.repo.Create(comment)
	if err != nil {
		return err
	}

	if hydrated, hydrateErr := s.repo.GetByID(comment.ID); hydrateErr == nil {
		*comment = *hydrated
	}

	author, _ := s.userRepo.GetByID(comment.AuthorID)
	authorName := "Кто-то"
	authorAvatar := ""
	if author != nil {
		authorName = author.Username
		authorAvatar = author.AvatarURL
	}

	// Increment comment count on post
	post, _ := s.postRepo.GetByID(comment.PostID)
	if post != nil {
		post.CommentCount++
		_ = s.postRepo.Update(post)

		isReply := comment.ParentID != nil && *comment.ParentID > 0
		if post.AuthorID != comment.AuthorID {
			title := "Новый комментарий"
			body := authorName + " прокомментировал ваш пост."
			notifType := "comment"
			if isReply {
				title = "Новый ответ"
				body = authorName + " ответил в обсуждении вашего поста."
				notifType = "reply"
			}
			_ = s.notifRepo.Create(&model.Notification{
				UserID:      post.AuthorID,
				Type:        notifType,
				Title:       title,
				Body:        body,
				ActorAvatar: authorAvatar,
			})
		}

		if isReply {
			parent, err := s.repo.GetByID(*comment.ParentID)
			if err == nil && parent.AuthorID != comment.AuthorID && parent.AuthorID != post.AuthorID {
				_ = s.notifRepo.Create(&model.Notification{
					UserID:      parent.AuthorID,
					Type:        "reply",
					Title:       "Ответ на комментарий",
					Body:        authorName + " ответил на ваш комментарий.",
					ActorAvatar: authorAvatar,
				})
			}
		}
	}

	for _, username := range extractMentions(comment.Content) {
		mentioned, err := s.userRepo.GetByUsername(username)
		if err != nil || mentioned.ID == comment.AuthorID {
			continue
		}
		_ = s.notifRepo.Create(&model.Notification{
			UserID:      mentioned.ID,
			Type:        "mention",
			Title:       "Упоминание",
			Body:        authorName + " упомянул вас в комментарии.",
			ActorAvatar: authorAvatar,
		})
	}

	return nil
}

func extractMentions(content string) []string {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	matches := mentionPattern.FindAllStringSubmatch(content, -1)
	seen := map[string]struct{}{}
	var out []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		name := strings.ToLower(m[1])
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, m[1])
	}
	return out
}

func (s *commentService) Update(userID uint, commentID uint, content string) (*model.Comment, error) {
	comment, err := s.repo.GetByID(commentID)
	if err != nil {
		return nil, err
	}
	if comment.AuthorID != userID {
		return nil, errors.New("unauthorized to edit this comment")
	}
	comment.Content = content
	err = s.repo.Update(comment)
	return comment, err
}

func (s *commentService) Delete(userID, commentID uint) error {
	comment, err := s.repo.GetByID(commentID)
	if err != nil {
		return err
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	isAuthorized := false
	if comment.AuthorID == userID {
		isAuthorized = true
	} else if user.Role == "admin" || user.Role == "moderator" {
		isAuthorized = true
	} else {
		// Verify if community owner where the comment belongs
		post, err := s.postRepo.GetByID(comment.PostID)
		if err == nil {
			comm, err := s.commRepo.GetByID(post.CommunityID)
			if err == nil && comm.OwnerID == userID {
				isAuthorized = true
			}
		}
	}

	if !isAuthorized {
		return errors.New("unauthorized to delete this comment")
	}

	comment.IsDeleted = true
	if user.Role == "admin" || user.Role == "moderator" {
		comment.Content = "[удалено модератором]"
	} else {
		comment.Content = "[удалено]"
	}

	return s.repo.Update(comment)
}

func (s *commentService) GetByID(commentID uint) (*model.Comment, error) {
	return s.repo.GetByID(commentID)
}

func (s *commentService) GetByPostID(postID uint, viewerID uint) ([]*model.Comment, error) {
	return s.repo.GetByPostID(postID, viewerID)
}

func (s *commentService) Vote(userID, commentID uint, value int) error {
	if value != 1 && value != -1 && value != 0 {
		return errors.New("invalid vote value")
	}

	comment, err := s.repo.GetByID(commentID)
	if err != nil {
		return err
	}

	if value == 0 {
		existing, err := s.voteRepo.GetVote(userID, "comment", commentID)
		if err != nil {
			return nil
		}

		err = s.voteRepo.DeleteVote(userID, "comment", commentID)
		if err != nil {
			return err
		}

		if existing.Value == 1 {
			comment.Score--
		} else {
			comment.Score++
		}

		return s.repo.Update(comment)
	}

	existing, err := s.voteRepo.GetVote(userID, "comment", commentID)
	if err == nil {
		if existing.Value == value {
			_ = s.voteRepo.DeleteVote(userID, "comment", commentID)
			comment.Score -= value
		} else {
			existing.Value = value
			_ = s.voteRepo.SaveVote(existing)
			comment.Score += 2 * value
		}
	} else {
		vote := &model.Vote{
			UserID:     userID,
			EntityType: "comment",
			EntityID:   commentID,
			Value:      value,
		}
		_ = s.voteRepo.SaveVote(vote)
		comment.Score += value
	}

	return s.repo.Update(comment)
}

func (s *commentService) GetVote(userID, commentID uint) (*model.Vote, error) {
	return s.voteRepo.GetVote(userID, "comment", commentID)
}

func (s *commentService) GetVotesForComments(userID uint, commentIDs []uint) map[uint]int {
	return s.voteRepo.GetVotesForEntities(userID, "comment", commentIDs)
}
