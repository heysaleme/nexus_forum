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
}

type communityService struct {
	repo     repository.CommunityRepository
	userRepo repository.UserRepository
}

func NewCommunityService(repo repository.CommunityRepository, userRepo repository.UserRepository) CommunityService {
	return &communityService{repo: repo, userRepo: userRepo}
}

func (s *communityService) Create(community *model.Community) error {
	// Add XP to owner
	owner, _ := s.userRepo.GetByID(community.OwnerID)
	if owner != nil {
		owner.XP += 30
		recalculateLevel(owner)
		_ = s.userRepo.Update(owner)
	}

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
