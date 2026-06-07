package service

import (
	"errors"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type UserService interface {
	GetByID(id uint) (*model.User, error)
	UpdateProfile(userID uint, req model.User) (*model.User, error)
	Follow(followerID, followingID uint) error
	Unfollow(followerID, followingID uint) error
	List(sortSpec string, limit int) ([]*model.User, error)
	UpdateUser(userID uint, role string, isBanned *bool) (*model.User, error)
	IsFollowing(followerID, followingID uint) (bool, error)
	GetFollowers(userID uint) ([]*model.User, error)
	GetFollowing(userID uint) ([]*model.User, error)
}

type userService struct {
	repo       repository.UserRepository
	followRepo repository.FollowRepository
}

func NewUserService(repo repository.UserRepository, followRepo repository.FollowRepository) UserService {
	return &userService{repo: repo, followRepo: followRepo}
}

func (s *userService) GetByID(id uint) (*model.User, error) {
	return s.repo.GetByID(id)
}

func (s *userService) UpdateProfile(userID uint, req model.User) (*model.User, error) {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Bio != "" {
		user.Bio = req.Bio
	}
	if req.Title != "" {
		user.Title = req.Title
	}
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}
	if req.BannerURL != "" {
		user.BannerURL = req.BannerURL
	}
	if req.ProfileTheme != "" {
		user.ProfileTheme = req.ProfileTheme
	}
	user.AllowDMs = req.AllowDMs
	user.IsPrivate = req.IsPrivate

	err = s.repo.Update(user)
	return user, err
}

func (s *userService) Follow(followerID, followingID uint) error {
	if followerID == followingID {
		return errors.New("cannot follow yourself")
	}

	_, err := s.followRepo.GetFollow(followerID, followingID)
	if err == nil {
		return errors.New("already following")
	}

	follow := &model.UserFollow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}

	err = s.followRepo.Follow(follow)
	if err != nil {
		return err
	}

	// Update stats & add XP
	follower, _ := s.repo.GetByID(followerID)
	if follower != nil {
		follower.FollowingCount++
		follower.XP += 4
		recalculateLevel(follower)
		_ = s.repo.Update(follower)
	}

	following, _ := s.repo.GetByID(followingID)
	if following != nil {
		following.FollowersCount++
		recalculateLevel(following)
		_ = s.repo.Update(following)
	}

	return nil
}

func (s *userService) Unfollow(followerID, followingID uint) error {
	err := s.followRepo.Unfollow(followerID, followingID)
	if err != nil {
		return err
	}

	follower, _ := s.repo.GetByID(followerID)
	if follower != nil && follower.FollowingCount > 0 {
		follower.FollowingCount--
		_ = s.repo.Update(follower)
	}

	following, _ := s.repo.GetByID(followingID)
	if following != nil && following.FollowersCount > 0 {
		following.FollowersCount--
		_ = s.repo.Update(following)
	}

	return nil
}

func (s *userService) List(sortSpec string, limit int) ([]*model.User, error) {
	return s.repo.List(sortSpec, limit)
}

func (s *userService) UpdateUser(userID uint, role string, isBanned *bool) (*model.User, error) {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	if role != "" {
		user.Role = role
	}
	if isBanned != nil {
		user.IsBanned = *isBanned
	}

	err = s.repo.Update(user)
	return user, err
}

func (s *userService) IsFollowing(followerID, followingID uint) (bool, error) {
	_, err := s.followRepo.GetFollow(followerID, followingID)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *userService) GetFollowers(userID uint) ([]*model.User, error) {
	follows, err := s.followRepo.GetFollowers(userID)
	if err != nil {
		return nil, err
	}
	var users []*model.User
	for _, f := range follows {
		if u, err := s.repo.GetByID(f.FollowerID); err == nil {
			users = append(users, u)
		}
	}
	return users, nil
}

func (s *userService) GetFollowing(userID uint) ([]*model.User, error) {
	follows, err := s.followRepo.GetFollowing(userID)
	if err != nil {
		return nil, err
	}
	var users []*model.User
	for _, f := range follows {
		if u, err := s.repo.GetByID(f.FollowingID); err == nil {
			users = append(users, u)
		}
	}
	return users, nil
}
