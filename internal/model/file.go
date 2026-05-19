package model

import "time"

// File stores the core metadata of an indexed file, isolated per user.
// Storage copy locations (channel_id, message_id) live in the separate FileCopy table —
// a single File may have multiple copies across redundant storage channels.
type File struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       uint   `gorm:"not null;index;uniqueIndex:idx_user_file_unique"`
	FileUniqueID string `gorm:"size:255;not null;uniqueIndex:idx_user_file_unique"`
	// BotFileID 是 Telegram 自己签发的 file_id(那个长的 base64-like 字符串),
	// 与 File.ID(数据库主键)是两回事
	BotFileID string `gorm:"column:bot_file_id;type:text;not null"`
	FileType  string `gorm:"size:20;not null;index"` // See constants.FileType*
	FileName  string `gorm:"size:500"`
	MimeType  string `gorm:"size:100"`
	FileSize  int64
	Duration  int
	Width     int
	Height    int
	Performer string    `gorm:"size:255"`
	Title     string    `gorm:"size:255"`
	Caption   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
	Messages  []Message `gorm:"foreignKey:FileID"`
}
