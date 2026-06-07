package service

import (
	"errors"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type CommentService interface {
	Create(comment *model.Comment) error
	Update(userID uint, commentID uint, content string) (*model.Comment, error)
	Delete(userID, commentID uint) error
	GetByPostID(postID uint) ([]*model.Comment, error)
	Vote(userID, commentID uint, value int) error
}

type commentService struct {
	repo      repository.CommentRepository
	userRepo  repository.UserRepository
	postRepo  repository.PostRepository
	voteRepo  repository.VoteRepository
	notifRepo repository.NotificationRepository
}

func NewCommentService(
	repo repository.CommentRepository,
	userRepo repository.UserRepository,
	postRepo repository.PostRepository,
	voteRepo repository.VoteRepository,
	notifRepo repository.NotificationRepository,
) CommentService {
	return &commentService{
		repo:      repo,
		userRepo:  userRepo,
		postRepo:  postRepo,
		voteRepo:  voteRepo,
		notifRepo: notifRepo,
	}
}

func (s *commentService) Create(comment *model.Comment) error {
	err := s.repo.Create(comment)
	if err != nil {
		return err
	}

	// Increment comment count on post
	post, _ := s.postRepo.GetByID(comment.PostID)
	if post != nil {
		post.CommentCount++
		_ = s.postRepo.Update(post)

		// Notify post author
		if post.AuthorID != comment.AuthorID {
			author, _ := s.userRepo.GetByID(comment.AuthorID)
			authorName := "Кто-то"
			if author != nil {
				authorName = author.Username
			}
			notif := &model.Notification{
				UserID: post.AuthorID,
				Type:   "reply",
				Title:  "Новый ответ",
				Body:   authorName + " ответил на ваш пост.",
			}
			_ = s.notifRepo.Create(notif)
		}
	}

	// Add XP
	author, _ := s.userRepo.GetByID(comment.AuthorID)
	if author != nil {
		author.XP += 8
		recalculateLevel(author)
		_ = s.userRepo.Update(author)
	}

	return nil
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
	if comment.AuthorID != userID {
		return errors.New("unauthorized to delete this comment")
	}
	return s.repo.Delete(commentID)
}

func (s *commentService) GetByPostID(postID uint) ([]*model.Comment, error) {
	return s.repo.GetByPostID(postID)
}

func (s *commentService) Vote(userID, commentID uint, value int) error {
	if value != 1 && value != -1 {
		return errors.New("invalid vote value")
	}

	comment, err := s.repo.GetByID(commentID)
	if err != nil {
		return err
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
