package repository

import (
	"encoding/json"
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type ChatRepository interface {
	CreateRoom(room *model.ChatRoom) error
	GetRoom(id uint) (*model.ChatRoom, error)
	GetRoomsByUser(userID uint) ([]*model.ChatRoom, error)
	UpdateRoom(room *model.ChatRoom) error
	CreateMessage(msg *model.Message) error
	GetMessagesByRoom(roomID uint, limit int) ([]*model.Message, error)
	MarkRoomMessagesRead(roomID, userID uint) error
	DeleteMessage(id uint) error
	GetMessage(id uint) (*model.Message, error)
	UpdateMessage(msg *model.Message) error
	DeleteRoom(id uint) error
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) CreateRoom(room *model.ChatRoom) error {
	return r.db.Create(room).Error
}

func (r *chatRepository) GetRoom(id uint) (*model.ChatRoom, error) {
	var room model.ChatRoom
	err := r.db.First(&room, id).Error
	return &room, err
}

func (r *chatRepository) GetRoomsByUser(userID uint) ([]*model.ChatRoom, error) {
	rooms := []*model.ChatRoom{}
	var allRooms []*model.ChatRoom
	err := r.db.Order("last_message_at DESC").Find(&allRooms).Error
	if err != nil {
		return nil, err
	}

	// Filter dynamically by parsing JSON participants
	for _, room := range allRooms {
		var pids []uint
		if err := json.Unmarshal([]byte(room.Participants), &pids); err == nil {
			for _, pid := range pids {
				if pid == userID {
					var count int64
					r.db.Model(&model.Message{}).Where("chat_room_id = ? AND sender_id != ? AND is_read = ?", room.ID, userID, false).Count(&count)
					room.UnreadCount = int(count)
					rooms = append(rooms, room)
					break
				}
			}
		}
	}
	return rooms, nil
}

func (r *chatRepository) UpdateRoom(room *model.ChatRoom) error {
	return r.db.Save(room).Error
}

func (r *chatRepository) CreateMessage(msg *model.Message) error {
	return r.db.Create(msg).Error
}

func (r *chatRepository) GetMessagesByRoom(roomID uint, limit int) ([]*model.Message, error) {
	var msgs []*model.Message
	q := r.db.Where("chat_room_id = ?", roomID).Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&msgs).Error
	return msgs, err
}

func (r *chatRepository) MarkRoomMessagesRead(roomID, userID uint) error {
	return r.db.Model(&model.Message{}).Where("chat_room_id = ? AND sender_id != ?", roomID, userID).Update("is_read", true).Error
}

func (r *chatRepository) DeleteMessage(id uint) error {
	return r.db.Delete(&model.Message{}, id).Error
}

func (r *chatRepository) GetMessage(id uint) (*model.Message, error) {
	var msg model.Message
	err := r.db.First(&msg, id).Error
	return &msg, err
}

func (r *chatRepository) UpdateMessage(msg *model.Message) error {
	return r.db.Save(msg).Error
}

func (r *chatRepository) DeleteRoom(id uint) error {
	if err := r.db.Where("chat_room_id = ?", id).Delete(&model.Message{}).Error; err != nil {
		return err
	}
	return r.db.Delete(&model.ChatRoom{}, id).Error
}
