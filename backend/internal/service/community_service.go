package service

import (
	"errors"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type CommunityService interface {
	Create(community *model.Community) error
	GetByID(id uint) (*model.Community, error)
	GetBySlug(slug string) (*model.Community, error)
	List(sortSpec string, limit int) ([]*model.Community, error)
	Join(userID, communityID uint) error
	Leave(userID, communityID uint) error
	GetMemberships(userID uint) ([]*model.CommunityMember, error)
	GetMembers(communityID uint) ([]*model.CommunityMember, error)
	Delete(userID, communityID uint) error
	Search(query string, limit int) ([]*model.Community, error)
	PromoteModerator(actorID, communityID, targetUserID uint) error
	DemoteModerator(actorID, communityID, targetUserID uint) error
	ListModerators(communityID uint) ([]*model.CommunityMember, error)
}

type communityService struct {
	repo     repository.CommunityRepository
	userRepo repository.UserRepository
}

func NewCommunityService(repo repository.CommunityRepository, userRepo repository.UserRepository) CommunityService {
	return &communityService{repo: repo, userRepo: userRepo}
}

func (s *communityService) Create(community *model.Community) error {
	err := s.repo.Create(community)
	if err != nil {
		return err
	}

	// Auto-join community as Owner
	member := &model.CommunityMember{
		UserID:      community.OwnerID,
		CommunityID: community.ID,
		Role:        "owner",
	}
	_ = s.repo.AddMember(member)

	community.MemberCount = 1
	_ = s.repo.Update(community)

	return nil
}

func (s *communityService) GetByID(id uint) (*model.Community, error) {
	return s.repo.GetByID(id)
}

func (s *communityService) GetBySlug(slug string) (*model.Community, error) {
	return s.repo.GetBySlug(slug)
}

func (s *communityService) List(sortSpec string, limit int) ([]*model.Community, error) {
	return s.repo.List(sortSpec, limit)
}

func (s *communityService) Join(userID, communityID uint) error {
	_, err := s.repo.GetMember(userID, communityID)
	if err == nil {
		return errors.New("already a member")
	}

	member := &model.CommunityMember{
		UserID:      userID,
		CommunityID: communityID,
		Role:        "member",
	}

	err = s.repo.AddMember(member)
	if err != nil {
		return err
	}

	comm, _ := s.repo.GetByID(communityID)
	if comm != nil {
		comm.MemberCount++
		_ = s.repo.Update(comm)
	}

	return nil
}

func (s *communityService) Leave(userID, communityID uint) error {
	member, err := s.repo.GetMember(userID, communityID)
	if err != nil {
		return errors.New("not a member")
	}
	if member.Role == "owner" {
		return errors.New("owner cannot leave the community")
	}

	err = s.repo.RemoveMember(userID, communityID)
	if err != nil {
		return err
	}

	comm, _ := s.repo.GetByID(communityID)
	if comm != nil && comm.MemberCount > 0 {
		comm.MemberCount--
		_ = s.repo.Update(comm)
	}

	return nil
}

func (s *communityService) GetMemberships(userID uint) ([]*model.CommunityMember, error) {
	return s.repo.GetMemberships(userID)
}

func (s *communityService) GetMembers(communityID uint) ([]*model.CommunityMember, error) {
	return s.repo.GetMembers(communityID)
}

func (s *communityService) Delete(userID, communityID uint) error {
	comm, err := s.repo.GetByID(communityID)
	if err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	if comm.OwnerID != userID && user.Role != "admin" && user.Role != "moderator" {
		return errors.New("unauthorized to delete this community")
	}

	return s.repo.Delete(communityID)
}

func (s *communityService) Search(query string, limit int) ([]*model.Community, error) {
	return s.repo.Search(query, limit)
}

func (s *communityService) canManageMods(actorID, communityID uint) error {
	comm, err := s.repo.GetByID(communityID)
	if err != nil {
		return errors.New("community not found")
	}
	if comm.OwnerID == actorID {
		return nil
	}
	user, err := s.userRepo.GetByID(actorID)
	if err == nil && (user.Role == "admin" || user.Role == "moderator") {
		return nil
	}
	member, err := s.repo.GetMember(actorID, communityID)
	if err == nil && (member.Role == "owner" || member.Role == "moderator") {
		return nil
	}
	return errors.New("insufficient permissions")
}

func (s *communityService) PromoteModerator(actorID, communityID, targetUserID uint) error {
	if err := s.canManageMods(actorID, communityID); err != nil {
		return err
	}
	member, err := s.repo.GetMember(targetUserID, communityID)
	if err != nil {
		return errors.New("user is not a community member")
	}
	if member.Role == "owner" {
		return errors.New("cannot change owner role")
	}
	return s.repo.UpdateMemberRole(targetUserID, communityID, "moderator")
}

func (s *communityService) DemoteModerator(actorID, communityID, targetUserID uint) error {
	if err := s.canManageMods(actorID, communityID); err != nil {
		return err
	}
	member, err := s.repo.GetMember(targetUserID, communityID)
	if err != nil {
		return errors.New("user is not a community member")
	}
	if member.Role == "owner" {
		return errors.New("cannot demote owner")
	}
	return s.repo.UpdateMemberRole(targetUserID, communityID, "member")
}

func (s *communityService) ListModerators(communityID uint) ([]*model.CommunityMember, error) {
	return s.repo.ListModerators(communityID)
}
