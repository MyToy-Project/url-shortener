package model

import (
	"time"

	"gorm.io/gorm"
)

// ShortURL represents a shortened URL in the database
type ShortURL struct {
	UrlID       uint `gorm:"primaryKey"`
	OriginalURL string
	ShortCode   string `gorm:"uniqueIndex"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
