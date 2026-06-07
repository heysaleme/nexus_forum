package service

import (
	"encoding/json"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
	"time"
)

type ChatService interface {
	CreateRoom(userID uint, participantIDs []uint, name string) (*model.ChatRoom, error)
	GetRoomsByUser(userID uint) ([]*model.ChatRoom, error)
	GetMessages(roomID uint, limit int) ([]*model.Message, error)
	SendMessage(senderID uint, roomID uint, content string) (*model.Message, error)
}

type chatService struct {
	repo     repository.ChatRepository
	userRepo repository.UserRepository
}

func NewChatService(repo repository.ChatRepository, userRepo repository.UserRepository) ChatService {
	return &chatService{repo: repo, userRepo: userRepo}
}

func (s *chatService) CreateRoom(userID uint, participantIDs []uint, name string) (*model.ChatRoom, error) {
	// Ensure user is one of the participants
	found := false
	for _, pid := range participantIDs {
		if pid == userID {
			found = true
			break
		}
	}
	if !found {
		participantIDs = append(participantIDs, userID)
	}

	pBytes, _ := json.Marshal(participantIDs)
	room := &model.ChatRoom{
		Name:         name,
		Type:         "direct",
		Participants: string(pBytes),
	}

	err := s.repo.CreateRoom(room)
	return room, err
}

func (s *chatService) GetRoomsByUser(userID uint) ([]*model.ChatRoom, error) {
	return s.repo.GetRoomsByUser(userID)
}

func (s *chatService) GetMessages(roomID uint, limit int) ([]*model.Message, error) {
	return s.repo.GetMessagesByRoom(roomID, limit)
}

func (s *chatService) SendMessage(senderID uint, roomID uint, content string) (*model.Message, error) {
	sender, _ := s.userRepo.GetByID(senderID)
	username := "Пользователь"
	avatar := ""
	if sender != nil {
		username = sender.Username
		avatar = sender.AvatarURL
	}

	msg := &model.Message{
		ChatRoomID:     roomID,
		SenderID:       senderID,
		SenderUsername: username,
		SenderAvatar:   avatar,
		Content:        content,
		MessageType:    "text",
	}

	err := s.repo.CreateMessage(msg)
	if err != nil {
		return nil, err
	}

	// Update room last message info
	room, err := s.repo.GetRoom(roomID)
	if err == nil {
		room.LastMessage = content
		room.LastMessageAt = time.Now()
		_ = s.repo.UpdateRoom(room)
	}

	return msg, nil
}
