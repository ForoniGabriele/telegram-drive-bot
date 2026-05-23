package model

import "time"

// Role values are defined in package constants (constants.RoleOwner/RoleAdmin/RoleUser).
type User struct {
	ID         uint   `gorm:"primaryKey"`
	TelegramID int64  `gorm:"uniqueIndex;not null"`
	Username   string `gorm:"size:255"`
	FirstName  string `gorm:"size:255"`
	Role       string `gorm:"size:20;not null;default:'user'"`
	Language   string `gorm:"size:10;not null;default:''"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Files      []File `gorm:"foreignKey:UserID"`
}
