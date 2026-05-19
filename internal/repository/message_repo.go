package repository

import (
	"tg-drive-bot/internal/model"

	"gorm.io/gorm"
)

// MessageRepo is the GORM-backed implementation of MessageRepository.
type MessageRepo struct {
	db *gorm.DB
}

// NewMessageRepo creates a MessageRepo bound to the given DB or transaction handle.
func NewMessageRepo(db *gorm.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

// Create inserts a new message record.
func (r *MessageRepo) Create(msg *model.Message) error {
	return r.db.Create(msg).Error
}

// DeleteByFileID removes every message row referencing the given file.
func (r *MessageRepo) DeleteByFileID(fileID uint) error {
	return r.db.Where("file_id = ?", fileID).Delete(&model.Message{}).Error
}
