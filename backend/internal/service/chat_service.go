package service

import (
	"encoding/json"
	"errors"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
	"time"
)

type ChatService interface {
	CreateRoom(userID uint, participantIDs []uint, name string) (*model.ChatRoom, error)
	GetRoomsByUser(userID uint) ([]*model.ChatRoom, error)
	GetMessages(roomID uint, userID uint, limit int) ([]*model.Message, error)
	SendMessage(senderID uint, roomID uint, content string) (*model.Message, error)
	SendMessageWithAttachment(senderID, roomID uint, content, attachmentURL, attachmentType string) (*model.Message, error)
	MarkRoomAsRead(roomID, userID uint) error
	GetRoom(roomID, userID uint) (*model.ChatRoom, error)
	UpdateRoom(room *model.ChatRoom) error
	DeleteMessage(userID, msgID uint, deleteType string) (*model.Message, error)
	UpdateMessage(userID, msgID uint, content string) (*model.Message, error)
	DeleteRoom(userID, roomID uint) error
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

func (s *chatService) GetMessages(roomID uint, userID uint, limit int) ([]*model.Message, error) {
	msgs, err := s.repo.GetMessagesByRoom(roomID, limit)
	if err != nil {
		return nil, err
	}
	filtered := []*model.Message{}
	for _, msg := range msgs {
		var deletedUsers []uint
		if msg.DeletedForUsers != "" {
			_ = json.Unmarshal([]byte(msg.DeletedForUsers), &deletedUsers)
		}
		isDeletedForMe := false
		for _, du := range deletedUsers {
			if du == userID {
				isDeletedForMe = true
				break
			}
		}
		if isDeletedForMe {
			continue
		}
		if sender, err := s.userRepo.GetByID(msg.SenderID); err == nil && sender.IsShadowBanned && msg.SenderID != userID {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered, nil
}

func (s *chatService) SendMessage(senderID uint, roomID uint, content string) (*model.Message, error) {
	return s.SendMessageWithAttachment(senderID, roomID, content, "", "")
}

func (s *chatService) SendMessageWithAttachment(senderID, roomID uint, content, attachmentURL, attachmentType string) (*model.Message, error) {
	sender, _ := s.userRepo.GetByID(senderID)
	username := "Пользователь"
	avatar := ""
	if sender != nil {
		username = sender.Username
		avatar = sender.AvatarURL
	}

	msgType := "text"
	if attachmentURL != "" {
		if attachmentType == "image" {
			msgType = "image"
		} else {
			msgType = "file"
		}
	}

	msg := &model.Message{
		ChatRoomID:     roomID,
		SenderID:       senderID,
		SenderUsername: username,
		SenderAvatar:   avatar,
		Content:        content,
		MessageType:    msgType,
		AttachmentURL:  attachmentURL,
		AttachmentType: attachmentType,
	}

	err := s.repo.CreateMessage(msg)
	if err != nil {
		return nil, err
	}

	// Update room last message info
	room, err := s.repo.GetRoom(roomID)
	if err == nil {
		if attachmentURL != "" {
			if attachmentType == "image" {
				room.LastMessage = "[Изображение]"
			} else {
				room.LastMessage = "[Файл]"
			}
		} else {
			room.LastMessage = content
		}
		room.LastMessageAt = time.Now()
		_ = s.repo.UpdateRoom(room)
	}

	return msg, nil
}

func (s *chatService) MarkRoomAsRead(roomID, userID uint) error {
	return s.repo.MarkRoomMessagesRead(roomID, userID)
}

func (s *chatService) GetRoom(roomID, userID uint) (*model.ChatRoom, error) {
	return s.repo.GetRoom(roomID)
}

func (s *chatService) UpdateRoom(room *model.ChatRoom) error {
	return s.repo.UpdateRoom(room)
}

func (s *chatService) DeleteMessage(userID, msgID uint, deleteType string) (*model.Message, error) {
	msg, err := s.repo.GetMessage(msgID)
	if err != nil {
		return nil, err
	}

	if deleteType == "everyone" {
		if msg.SenderID != userID {
			user, err := s.userRepo.GetByID(userID)
			if err != nil || (user.Role != "admin" && user.Role != "moderator") {
				return nil, errors.New("unauthorized to delete this message for everyone")
			}
		}

		msg.Content = "Сообщение удалено"
		msg.AttachmentURL = ""
		msg.AttachmentType = ""

		err = s.repo.UpdateMessage(msg)
		return msg, err
	}

	// Default: "me"
	var deletedUsers []uint
	if msg.DeletedForUsers != "" {
		_ = json.Unmarshal([]byte(msg.DeletedForUsers), &deletedUsers)
	}

	found := false
	for _, du := range deletedUsers {
		if du == userID {
			found = true
			break
		}
	}
	if !found {
		deletedUsers = append(deletedUsers, userID)
	}

	bytes, _ := json.Marshal(deletedUsers)
	msg.DeletedForUsers = string(bytes)

	err = s.repo.UpdateMessage(msg)
	return msg, err
}

func (s *chatService) UpdateMessage(userID, msgID uint, content string) (*model.Message, error) {
	msg, err := s.repo.GetMessage(msgID)
	if err != nil {
		return nil, err
	}

	if msg.SenderID != userID {
		return nil, errors.New("unauthorized to edit this message")
	}

	msg.Content = content
	msg.IsEdited = true

	err = s.repo.UpdateMessage(msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *chatService) DeleteRoom(userID, roomID uint) error {
	room, err := s.repo.GetRoom(roomID)
	if err != nil {
		return err
	}
	var pids []uint
	if err := json.Unmarshal([]byte(room.Participants), &pids); err != nil {
		return err
	}
	isParticipant := false
	for _, pid := range pids {
		if pid == userID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		user, err := s.userRepo.GetByID(userID)
		if err != nil || (user.Role != "admin" && user.Role != "moderator") {
			return errors.New("unauthorized to delete this chat room")
		}
	}
	return s.repo.DeleteRoom(roomID)
}
