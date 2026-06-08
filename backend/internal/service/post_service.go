package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type PostService interface {
	Create(post *model.Post) error
	GetByID(id uint) (*model.Post, error)
	Update(post *model.Post) error
	Delete(userID, postID uint) error
	List(sortSpec string, limit int, viewerID uint) ([]*model.Post, error)
	ListFollowing(userID uint, sortSpec string, limit int) ([]*model.Post, error)
	Filter(filter map[string]interface{}, sortSpec string, limit int, viewerID uint) ([]*model.Post, error)
	Vote(userID, postID uint, value int) error
	VotePoll(userID, postID uint, optionIndex int) error
	SavePost(userID, postID uint) error
	UnsavePost(userID, postID uint) error
	GetSavedByUser(userID uint) ([]*model.SavedPost, error)
	GetVote(userID, postID uint) (*model.Vote, error)
	Search(query string, limit int, viewerID uint) ([]*model.Post, error)
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

func (s *postService) List(sortSpec string, limit int, viewerID uint) ([]*model.Post, error) {
	return s.repo.List(sortSpec, limit, viewerID)
}

func (s *postService) ListFollowing(userID uint, sortSpec string, limit int) ([]*model.Post, error) {
	return s.repo.ListByFollowing(userID, sortSpec, limit, userID)
}

func (s *postService) Filter(filter map[string]interface{}, sortSpec string, limit int, viewerID uint) ([]*model.Post, error) {
	return s.repo.Filter(filter, sortSpec, limit, viewerID)
}

func (s *postService) Search(query string, limit int, viewerID uint) ([]*model.Post, error) {
	return s.repo.Search(query, limit, viewerID)
}

func (s *postService) Vote(userID, postID uint, value int) error {

	fmt.Println("===== VOTE SERVICE =====")
	fmt.Println("USER:", userID)
	fmt.Println("POST:", postID)
	fmt.Println("VALUE:", value)

	if value != 1 && value != -1 && value != 0 {
		return errors.New("invalid vote value")
	}

	post, err := s.repo.GetByID(postID)
	if err != nil {
		return err
	}

	if value == 0 {
		existing, err := s.voteRepo.GetVote(userID, "post", postID)
		if err != nil {
			return nil
		}
		err = s.voteRepo.DeleteVote(userID, "post", postID)
		if err != nil {
			return err
		}
		if existing.Value == 1 {
			post.Upvotes = maxZero(post.Upvotes - 1)
			post.Score--
		} else {
			post.Downvotes = maxZero(post.Downvotes - 1)
			post.Score++
		}
		return s.repo.Update(post)
	}

	existing, err := s.voteRepo.GetVote(userID, "post", postID)

	if err == nil {
		if existing.Value == value {
			_ = s.voteRepo.DeleteVote(userID, "post", postID)
			if value == 1 {
				post.Upvotes = maxZero(post.Upvotes - 1)
				post.Score--
			} else {
				post.Downvotes = maxZero(post.Downvotes - 1)
				post.Score++
			}
		} else {
			// Меняем голос +1 <-> -1
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

		err = s.voteRepo.SaveVote(vote)

		fmt.Println("SAVE VOTE ERR:", err)
		fmt.Printf("VOTE: %+v\n", vote)

		if err != nil {
			return err
		}
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

func (s *postService) GetVote(userID, postID uint) (*model.Vote, error) {
	return s.voteRepo.GetVote(userID, "post", postID)
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

func (s *postService) VotePoll(userID, postID uint, optionIndex int) error {
	post, err := s.repo.GetByID(postID)
	if err != nil {
		return err
	}

	var options []map[string]interface{}
	if err := json.Unmarshal([]byte(post.PollOptions), &options); err != nil {
		return err
	}

	if optionIndex < 0 || optionIndex >= len(options) {
		return errors.New("invalid option")
	}

	votes := map[string]int{}

	if post.PollVotes != "" {
		_ = json.Unmarshal([]byte(post.PollVotes), &votes)
	}

	userKey := fmt.Sprintf("%d", userID)

	// Если уже голосовал
	if oldOption, exists := votes[userKey]; exists {

		// Нажал тот же вариант -> убрать голос
		if oldOption == optionIndex {

			if count, ok := options[oldOption]["votes"].(float64); ok {
				options[oldOption]["votes"] = int(count) - 1
			}

			delete(votes, userKey)

		} else {

			// Переголосование

			if count, ok := options[oldOption]["votes"].(float64); ok {
				options[oldOption]["votes"] = int(count) - 1
			}

			if count, ok := options[optionIndex]["votes"].(float64); ok {
				options[optionIndex]["votes"] = int(count) + 1
			}

			votes[userKey] = optionIndex
		}

	} else {

		// Новый голос

		if count, ok := options[optionIndex]["votes"].(float64); ok {
			options[optionIndex]["votes"] = int(count) + 1
		}

		votes[userKey] = optionIndex
	}

	optionsJSON, _ := json.Marshal(options)
	votesJSON, _ := json.Marshal(votes)

	post.PollOptions = string(optionsJSON)
	post.PollVotes = string(votesJSON)

	return s.repo.Update(post)
}
