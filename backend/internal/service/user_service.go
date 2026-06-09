package service

import (
	"errors"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type UserService interface {
	GetByID(id uint) (*model.User, error)
	GetByIDs(ids []uint) (map[uint]*model.User, error)
	UpdateProfile(userID uint, req model.User) (*model.User, error)
	Save(user *model.User) error
	UpdateUser(actorID, userID uint, role string, isBanned *bool) (*model.User, error)
	IsFollowing(followerID, followingID uint) (bool, error)
	GetFollowers(userID uint) ([]*model.User, error)
	GetFollowing(userID uint) ([]*model.User, error)
	Follow(followerID, followingID uint) error
	Unfollow(followerID, followingID uint) error
	List(sortSpec string, limit int) ([]*model.User, error)
	AcceptFollowRequest(followerID, followingID uint) error
	RejectFollowRequest(followerID, followingID uint) error
	GetPendingFollowRequests(userID uint) ([]*model.User, error)
	GetFollowRecord(followerID, followingID uint) (*model.UserFollow, error)
	Search(query string, limit int) ([]*model.User, error)
	GetProfileStats(userID uint) (*repository.ProfileStats, error)
	GetAchievements(userID uint) ([]Achievement, error)
}

type userService struct {
	repo       repository.UserRepository
	followRepo repository.FollowRepository
	notifRepo  repository.NotificationRepository
	modRepo    repository.ModerationRepository
	karmaRepo  repository.KarmaRepository
}

func NewUserService(
	repo repository.UserRepository,
	followRepo repository.FollowRepository,
	notifRepo repository.NotificationRepository,
	modRepo repository.ModerationRepository,
	karmaRepo repository.KarmaRepository,
) UserService {
	return &userService{
		repo:       repo,
		followRepo: followRepo,
		notifRepo:  notifRepo,
		modRepo:    modRepo,
		karmaRepo:  karmaRepo,
	}
}

func (s *userService) GetByID(id uint) (*model.User, error) {
	_ = s.repo.SyncFollowCounts(id)
	user, err := s.repo.GetByID(id)
	if err == nil && s.karmaRepo != nil {
		s.karmaRepo.HydrateUser(user)
	}
	return user, err
}

func (s *userService) GetProfileStats(userID uint) (*repository.ProfileStats, error) {
	return s.repo.GetProfileStats(userID)
}

func (s *userService) GetAchievements(userID uint) ([]Achievement, error) {
	stats, err := s.repo.GetProfileStats(userID)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if s.karmaRepo != nil {
		s.karmaRepo.HydrateUser(user)
	}
	return computeAchievements(userID, stats, stats.CommunitiesOwned, user.TotalKarma), nil
}

func (s *userService) GetByIDs(ids []uint) (map[uint]*model.User, error) {
	return s.repo.GetByIDs(ids)
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

func (s *userService) Save(user *model.User) error {
	return s.repo.Update(user)
}

func (s *userService) Follow(followerID, followingID uint) error {
	if followerID == followingID {
		return errors.New("cannot follow yourself")
	}

	existing, err := s.followRepo.GetFollow(followerID, followingID)
	if err == nil {
		if existing.Status == "pending" {
			return errors.New("follow request already pending")
		}
		return errors.New("already following")
	}

	followingUser, err := s.repo.GetByID(followingID)
	if err != nil {
		return errors.New("user not found")
	}

	status := "accepted"
	if followingUser.IsPrivate {
		status = "pending"
	}

	follow := &model.UserFollow{
		FollowerID:  followerID,
		FollowingID: followingID,
		Status:      status,
	}

	err = s.followRepo.Follow(follow)
	if err != nil {
		return err
	}

	followerUser, _ := s.repo.GetByID(followerID)

	if status == "accepted" {
		if followerUser != nil {
			followerUser.FollowingCount++
			_ = s.repo.Update(followerUser)
			_ = s.notifRepo.Create(&model.Notification{
				UserID:      followingID,
				Type:        "follow",
				Title:       "Новый подписчик",
				Body:        followerUser.Username + " подписался на вас.",
				ActorAvatar: followerUser.AvatarURL,
			})
		}
		followingUser.FollowersCount++
		_ = s.repo.Update(followingUser)
	} else {
		// Send pending notification
		if followerUser != nil {
			notif := &model.Notification{
				UserID:      followingID,
				Type:        "follow_request",
				Title:       "Запрос на подписку",
				Body:        followerUser.Username + " хочет подписаться на ваши обновления.",
				ActorAvatar: followerUser.AvatarURL,
			}
			_ = s.notifRepo.Create(notif)
		}
	}

	return nil
}

func (s *userService) Unfollow(followerID, followingID uint) error {
	follow, err := s.followRepo.GetFollow(followerID, followingID)
	if err != nil {
		return errors.New("not following")
	}

	err = s.followRepo.Unfollow(followerID, followingID)
	if err != nil {
		return err
	}

	if follow.Status == "accepted" {
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
	}

	return nil
}

func (s *userService) List(sortSpec string, limit int) ([]*model.User, error) {
	return s.repo.List(sortSpec, limit)
}

func (s *userService) UpdateUser(actorID, userID uint, role string, isBanned *bool) (*model.User, error) {
	actor, err := s.repo.GetByID(actorID)
	if err != nil {
		return nil, errors.New("actor not found")
	}
	if actor.Role != "admin" {
		return nil, errors.New("admin access required")
	}

	user, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	if role != "" && role != user.Role {
		oldRole := user.Role
		user.Role = role
		_ = s.modRepo.CreateLog(&model.ModerationLog{
			ActorID:    actorID,
			TargetID:   userID,
			TargetType: "user",
			Action:     "role_changed",
			Details:    "Changed role from " + oldRole + " to " + role,
		})
	}
	if isBanned != nil && *isBanned != user.IsBanned {
		user.IsBanned = *isBanned
		action := "user_banned"
		if !*isBanned {
			action = "user_unbanned"
		}
		_ = s.modRepo.CreateLog(&model.ModerationLog{
			ActorID:    actorID,
			TargetID:   userID,
			TargetType: "user",
			Action:     action,
			Details:    "User ban status updated",
		})
	}

	err = s.repo.Update(user)
	return user, err
}

func (s *userService) IsFollowing(followerID, followingID uint) (bool, error) {
	follow, err := s.followRepo.GetFollow(followerID, followingID)
	if err != nil {
		return false, nil
	}
	return follow.Status == "accepted", nil
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

func (s *userService) AcceptFollowRequest(followerID, followingID uint) error {
	follow, err := s.followRepo.GetFollow(followerID, followingID)
	if err != nil {
		return errors.New("follow request not found")
	}
	if follow.Status != "pending" {
		return errors.New("request is not pending")
	}

	follow.Status = "accepted"
	err = s.followRepo.Update(follow)
	if err != nil {
		return err
	}

	follower, _ := s.repo.GetByID(followerID)
	if follower != nil {
		follower.FollowingCount++
		_ = s.repo.Update(follower)
	}

	following, _ := s.repo.GetByID(followingID)
	if following != nil {
		following.FollowersCount++
		_ = s.repo.Update(following)
	}

	// Send notification
	if following != nil {
		notif := &model.Notification{
			UserID:      followerID,
			Type:        "follow_accept",
			Title:       "Запрос принят",
			Body:        following.Username + " принял ваш запрос на подписку.",
			ActorAvatar: following.AvatarURL,
		}
		_ = s.notifRepo.Create(notif)
	}

	// Write audit log
	_ = s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:    followingID,
		TargetID:   followerID,
		TargetType: "user",
		Action:     "follow_request_accepted",
		Details:    "Accepted follow request",
	})

	return nil
}

func (s *userService) RejectFollowRequest(followerID, followingID uint) error {
	follow, err := s.followRepo.GetFollow(followerID, followingID)
	if err != nil {
		return errors.New("follow request not found")
	}
	if follow.Status != "pending" {
		return errors.New("request is not pending")
	}

	err = s.followRepo.Unfollow(followerID, followingID)
	if err != nil {
		return err
	}

	// Send notification
	following, _ := s.repo.GetByID(followingID)
	if following != nil {
		notif := &model.Notification{
			UserID:      followerID,
			Type:        "follow_reject",
			Title:       "Запрос отклонен",
			Body:        following.Username + " отклонил ваш запрос на подписку.",
			ActorAvatar: following.AvatarURL,
		}
		_ = s.notifRepo.Create(notif)
	}

	// Write audit log
	_ = s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:    followingID,
		TargetID:   followerID,
		TargetType: "user",
		Action:     "follow_request_rejected",
		Details:    "Rejected follow request",
	})

	return nil
}

func (s *userService) GetPendingFollowRequests(userID uint) ([]*model.User, error) {
	follows, err := s.followRepo.GetPendingRequests(userID)
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

func (s *userService) GetFollowRecord(followerID, followingID uint) (*model.UserFollow, error) {
	return s.followRepo.GetFollow(followerID, followingID)
}

func (s *userService) Search(query string, limit int) ([]*model.User, error) {
	return s.repo.Search(query, limit)
}
