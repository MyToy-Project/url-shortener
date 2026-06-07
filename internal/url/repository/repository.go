package repository

import "gorm.io/gorm"

type Repository struct {
	ShortURLRepository ShortURLRepository
}

func NewRepository(db *gorm.DB) Repository {
	return Repository{
		ShortURLRepository: NewShortURLRepository(db),
	}
}
