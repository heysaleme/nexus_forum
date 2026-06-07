package service

import (
	"errors"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type ModerationService interface {
	// BanUser bans a user globally. Only admins/mods can do this.
	BanUser(moderatorID, targetUserID uint, reason string) error
	// UnbanUser removes a ban. Only admins/mods.
	UnbanUser(moderatorID, targetUserID uint, reason string) error
	// RemovePost soft-deletes a post (sets status="removed"). Mods or admins.
	RemovePost(moderatorID, postID uint, reason string) error
	// RemoveComment soft-deletes a comment. Mods or admins.
	RemoveComment(moderatorID, commentID uint, reason string) error
	// GetLogs returns global moderation log entries.
	GetLogs(limit int) ([]*model.ModerationLog, error)
	// GetLogsByCommunity returns mod logs for a specific community.
	GetLogsByCommunity(communityID uint, limit int) ([]*model.ModerationLog, error)
	// CreateModerationLog persists any log entry (used for reports).
	CreateModerationLog(modID uint, modUsername, action, targetType string, targetID uint, reason string) error
}

type moderationService struct {
	modRepo     repository.ModerationRepository
	userRepo    repository.UserRepository
	postRepo    repository.PostRepository
	commentRepo repository.CommentRepository
	commRepo    repository.CommunityRepository
}

func NewModerationService(
	modRepo repository.ModerationRepository,
	userRepo repository.UserRepository,
	postRepo repository.PostRepository,
	commentRepo repository.CommentRepository,
	commRepo repository.CommunityRepository,
) ModerationService {
	return &moderationService{
		modRepo:     modRepo,
		userRepo:    userRepo,
		postRepo:    postRepo,
		commentRepo: commentRepo,
		commRepo:    commRepo,
	}
}

func (s *moderationService) requireAdminOrMod(moderatorID uint) (*model.User, error) {
	mod, err := s.userRepo.GetByID(moderatorID)
	if err != nil {
		return nil, errors.New("moderator not found")
	}
	if mod.Role != "admin" && mod.Role != "moderator" {
		return nil, errors.New("insufficient permissions: admin or moderator role required")
	}
	return mod, nil
}

func (s *moderationService) BanUser(moderatorID, targetUserID uint, reason string) error {
	mod, err := s.requireAdminOrMod(moderatorID)
	if err != nil {
		return err
	}

	target, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return errors.New("target user not found")
	}
	if target.IsBanned {
		return errors.New("user is already banned")
	}

	target.IsBanned = true
	if err := s.userRepo.Update(target); err != nil {
		return err
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ModeratorID:       moderatorID,
		TargetID:          targetUserID,
		TargetType:        "user",
		Action:            "ban",
		Reason:            reason,
		ModeratorUsername: mod.Username,
	})
}

func (s *moderationService) UnbanUser(moderatorID, targetUserID uint, reason string) error {
	mod, err := s.requireAdminOrMod(moderatorID)
	if err != nil {
		return err
	}

	target, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return errors.New("target user not found")
	}

	target.IsBanned = false
	if err := s.userRepo.Update(target); err != nil {
		return err
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ModeratorID:       moderatorID,
		TargetID:          targetUserID,
		TargetType:        "user",
		Action:            "unban",
		Reason:            reason,
		ModeratorUsername: mod.Username,
	})
}

func (s *moderationService) RemovePost(moderatorID, postID uint, reason string) error {
	post, err := s.postRepo.GetByID(postID)
	if err != nil {
		return errors.New("post not found")
	}

	isAuthorized := false
	user, err := s.userRepo.GetByID(moderatorID)
	if err == nil && (user.Role == "admin" || user.Role == "moderator") {
		isAuthorized = true
	} else {
		comm, err := s.commRepo.GetByID(post.CommunityID)
		if err == nil && comm.OwnerID == moderatorID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return errors.New("insufficient permissions: admin, moderator or community owner role required")
	}

	if post.Status == "removed" {
		return errors.New("post is already removed")
	}

	post.Status = "removed"
	if err := s.postRepo.Update(post); err != nil {
		return err
	}

	var modUsername string
	if user != nil {
		modUsername = user.Username
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ModeratorID:       moderatorID,
		TargetID:          postID,
		TargetType:        "post",
		Action:            "remove_post",
		Reason:            reason,
		ModeratorUsername: modUsername,
	})
}

func (s *moderationService) RemoveComment(moderatorID, commentID uint, reason string) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		return errors.New("comment not found")
	}

	post, err := s.postRepo.GetByID(comment.PostID)
	if err != nil {
		return errors.New("associated post not found")
	}

	isAuthorized := false
	user, err := s.userRepo.GetByID(moderatorID)
	if err == nil && (user.Role == "admin" || user.Role == "moderator") {
		isAuthorized = true
	} else {
		comm, err := s.commRepo.GetByID(post.CommunityID)
		if err == nil && comm.OwnerID == moderatorID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return errors.New("insufficient permissions: admin, moderator or community owner role required")
	}

	comment.IsDeleted = true
	comment.Content = "[удалено модератором]"
	if err := s.commentRepo.Update(comment); err != nil {
		return err
	}

	var modUsername string
	if user != nil {
		modUsername = user.Username
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ModeratorID:       moderatorID,
		TargetID:          commentID,
		TargetType:        "comment",
		Action:            "remove_comment",
		Reason:            reason,
		ModeratorUsername: modUsername,
	})
}

func (s *moderationService) GetLogs(limit int) ([]*model.ModerationLog, error) {
	return s.modRepo.GetLogs(limit)
}

func (s *moderationService) GetLogsByCommunity(communityID uint, limit int) ([]*model.ModerationLog, error) {
	return s.modRepo.GetLogsByCommunity(communityID, limit)
}

func (s *moderationService) CreateModerationLog(modID uint, modUsername, action, targetType string, targetID uint, reason string) error {
	return s.modRepo.CreateLog(&model.ModerationLog{
		ModeratorID:       modID,
		ModeratorUsername: modUsername,
		Action:            action,
		TargetType:        targetType,
		TargetID:          targetID,
		Reason:            reason,
	})
}
