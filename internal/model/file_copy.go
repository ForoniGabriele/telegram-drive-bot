package model

import "time"

// FileCopy records a single storage-channel copy of a File.
// One File may have multiple copies across redundant storage channels.
type FileCopy struct {
	ID            uint  `gorm:"primaryKey"`
	FileID        uint  `gorm:"not null;uniqueIndex:idx_file_copy_unique"`
	StorageChatID int64 `gorm:"not null;uniqueIndex:idx_file_copy_unique"`
	StorageMsgID  int   `gorm:"not null"`
	CreatedAt     time.Time
}
