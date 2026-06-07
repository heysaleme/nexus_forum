package service

import (
	"errors"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type PostService interface {
	Create(post *model.Post) error
	GetByID(id uint) (*model.Post, error)
	Update(post *model.Post) error
	Delete(userID, postID uint) error
	List(sortSpec string, limit int) ([]*model.Post, error)
	Filter(filter map[string]interface{}, sortSpec string, limit int) ([]*model.Post, error)
	Vote(userID, postID uint, value int) error
	SavePost(userID, postID uint) error
	UnsavePost(userID, postID uint) error
	GetSavedByUser(userID uint) ([]*model.SavedPost, error)
}

type postService struct {
	repo      repository.PostRepository
	userRepo  repository.UserRepository
	commRepo  repository.CommunityRepository
	voteRepo  repository.VoteRepository
	savedRepo repository.SavedPostRepository
	notifRepo repository.NotificationRepository
}

func NewPostService(
	repo repository.PostRepository,
	userRepo repository.UserRepository,
	commRepo repository.CommunityRepository,
	voteRepo repository.VoteRepository,
	savedRepo repository.SavedPostRepository,
	notifRepo repository.NotificationRepository,
) PostService {
	return &postService{
		repo:      repo,
		userRepo:  userRepo,
		commRepo:  commRepo,
		voteRepo:  voteRepo,
		savedRepo: savedRepo,
		notifRepo: notifRepo,
	}
}

func (s *postService) Create(post *model.Post) error {
	err := s.repo.Create(post)
	if err != nil {
		return err
	}

	// Update community post count
	comm, _ := s.commRepo.GetByID(post.CommunityID)
	if comm != nil {
		comm.PostCount++
		_ = s.commRepo.Update(comm)
	}

	// Add XP
	author, _ := s.userRepo.GetByID(post.AuthorID)
	if author != nil {
		author.XP += 20
		recalculateLevel(author)
		_ = s.userRepo.Update(author)
	}

	return nil
}

func (s *postService) GetByID(id uint) (*model.Post, error) {
	post, err := s.repo.GetByID(id)
	if err == nil {
		post.Views++
		_ = s.repo.Update(post)
	}
	return post, err
}

func (s *postService) Update(post *model.Post) error {
	return s.repo.Update(post)
}

func (s *postService) Delete(userID, postID uint) error {
	post, err := s.repo.GetByID(postID)
	if err != nil {
		return err
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	if post.AuthorID != userID && user.Role != "admin" && user.Role != "moderator" {
		return errors.New("unauthorized to delete this post")
	}

	// Deduct XP from original author
	if post.AuthorID != 0 {
		author, err := s.userRepo.GetByID(post.AuthorID)
		if err == nil && author != nil {
			author.XP = maxZero(author.XP - 20)
			recalculateLevel(author)
			_ = s.userRepo.Update(author)
		}
	}

	// Decrement community post count
	comm, _ := s.commRepo.GetByID(post.CommunityID)
	if comm != nil && comm.PostCount > 0 {
		comm.PostCount--
		_ = s.commRepo.Update(comm)
	}

	return s.repo.Delete(postID)
}


func (s *postService) List(sortSpec string, limit int) ([]*model.Post, error) {
	return s.repo.List(sortSpec, limit)
}

func (s *postService) Filter(filter map[string]interface{}, sortSpec string, limit int) ([]*model.Post, error) {
	return s.repo.Filter(filter, sortSpec, limit)
}

func (s *postService) Vote(userID, postID uint, value int) error {
	if value != 1 && value != -1 {
		return errors.New("invalid vote value")
	}

	post, err := s.repo.GetByID(postID)
	if err != nil {
		return err
	}

	existing, err := s.voteRepo.GetVote(userID, "post", postID)
	if err == nil {
		// Existing vote
		if existing.Value == value {
			// Cancel vote
			_ = s.voteRepo.DeleteVote(userID, "post", postID)
			if value == 1 {
				post.Upvotes = maxZero(post.Upvotes - 1)
				post.Score--
			} else {
				post.Downvotes = maxZero(post.Downvotes - 1)
				post.Score++
			}
		} else {
			// Flip vote
			existing.Value = value
			_ = s.voteRepo.SaveVote(existing)
			if value == 1 {
				post.Upvotes++
				post.Downvotes = maxZero(post.Downvotes - 1)
				post.Score += 2
			} else {
				post.Downvotes++
				post.Upvotes = maxZero(post.Upvotes - 1)
				post.Score -= 2
			}
		}
	} else {
		// New vote
		vote := &model.Vote{
			UserID:     userID,
			EntityType: "post",
			EntityID:   postID,
			Value:      value,
		}
		_ = s.voteRepo.SaveVote(vote)
		if value == 1 {
			post.Upvotes++
			post.Score++
		} else {
			post.Downvotes++
			post.Score--
		}
	}

	return s.repo.Update(post)
}

func (s *postService) SavePost(userID, postID uint) error {
	ok, _ := s.savedRepo.IsSaved(userID, postID)
	if ok {
		return errors.New("already saved")
	}
	saved := &model.SavedPost{UserID: userID, PostID: postID}
	return s.savedRepo.Save(saved)
}

func (s *postService) UnsavePost(userID, postID uint) error {
	return s.savedRepo.Unsave(userID, postID)
}

func (s *postService) GetSavedByUser(userID uint) ([]*model.SavedPost, error) {
	return s.savedRepo.GetByUser(userID)
}
