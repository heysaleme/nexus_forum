package service

import (
	"errors"
	"fmt"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type CommentService interface {
	Create(comment *model.Comment) error
	Update(userID uint, commentID uint, content string) (*model.Comment, error)
	Delete(userID, commentID uint) error
	GetByPostID(postID uint, viewerID uint) ([]*model.Comment, error)
	Vote(userID, commentID uint, value int) error
	GetVote(userID, commentID uint) (*model.Vote, error)
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

	// Deduct XP from original comment author
	if comment.AuthorID != 0 && comment.AuthorID != userID {
		// Only deduct if someone else deleted it (mod/admin action)
		author, err := s.userRepo.GetByID(comment.AuthorID)
		if err == nil && author != nil {
			author.XP = maxZero(author.XP - 8)
			recalculateLevel(author)
			_ = s.userRepo.Update(author)
		}
	} else if comment.AuthorID == userID {
		// Self-deletion also loses XP
		userObj, err := s.userRepo.GetByID(userID)
		if err == nil && userObj != nil {
			userObj.XP = maxZero(userObj.XP - 8)
			recalculateLevel(userObj)
			_ = s.userRepo.Update(userObj)
		}
	}

	return s.repo.Update(comment)
}

func (s *commentService) GetByPostID(postID uint, viewerID uint) ([]*model.Comment, error) {
	return s.repo.GetByPostID(postID, viewerID)
}

func (s *commentService) Vote(userID, commentID uint, value int) error {

	fmt.Println("===== COMMENT VOTE SERVICE =====")

	fmt.Println("USER:", userID)

	fmt.Println("COMMENT:", commentID)

	fmt.Println("VALUE:", value)

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
