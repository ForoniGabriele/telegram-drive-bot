package model

import "time"

// Message records the context of each file submission.
// The same file may be sent multiple times (e.g. forwarded); each occurrence is a separate Message.
type Message struct {
	ID           uint      `gorm:"primaryKey"`
	FileID       uint      `gorm:"not null;index"`
	MessageID    int       `gorm:"not null"`
	ChatID       int64     `gorm:"not null"`
	FromUserID   int64     `gorm:"not null"`
	FromUsername string    `gorm:"size:255"`
	Caption      string    `gorm:"type:text"`
	MediaGroupID string    `gorm:"size:255"`
	ForwardFrom  string    `gorm:"type:text"`
	ReceivedAt   time.Time `gorm:"not null"`
}
