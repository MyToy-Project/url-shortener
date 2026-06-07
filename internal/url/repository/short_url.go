package repository

import (
	"url-shortener/internal/url/repository/model"

	"gorm.io/gorm"
)

type ShortURLRepository interface {
	Create(model.ShortURL) error
	FindShortURLByCode(model.ShortURL) (model.ShortURL, error)
}

type ShortURL struct {
	db *gorm.DB
}

func NewShortURLRepository(gormDB *gorm.DB) *ShortURL {
	return &ShortURL{db: gormDB}
}

func (s *ShortURL) Create(shortURL model.ShortURL) error {
	return s.db.Create(&shortURL).Error
}

func (s *ShortURL) FindShortURLByCode(shortCode model.ShortURL) (model.ShortURL, error) {
	var shortURL model.ShortURL
	err := s.db.Where(&shortCode).First(&shortURL).Error

	if err != nil {
		return model.ShortURL{}, err
	}

	return shortURL, nil
}
