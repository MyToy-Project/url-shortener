package service

import (
	"errors"
	"log/slog"
	"strings"
	"url-shortener/internal/url"
	"url-shortener/internal/url/repository"
	"url-shortener/internal/url/repository/model"

	"gorm.io/gorm"
)

type codeGenerator interface {
	newCode(code int) (string, error)
}

type ShortURLService struct {
	generator  codeGenerator
	repository repository.Repository
}

func NewService(generator codeGenerator, repository repository.Repository) *ShortURLService {
	return &ShortURLService{generator: generator, repository: repository}
}

func (s *ShortURLService) GenerateShortURL(originalURL string) (string, error) {
	originalURL = strings.TrimSpace(originalURL)
	err := verifyURL(originalURL)

	if err != nil {
		url.CountUpShortURLCreationCounter("failed")
		return "", err
	}

	if !strings.HasPrefix(originalURL, "https://") {
		originalURL = "https://" + originalURL
	}

	code, err := s.generator.newCode(8)
	if err != nil {
		url.CountUpShortURLCreationCounter("failed")
		slog.Error("can't generate short code", "url", originalURL, "error", err)
		return "", err
	}

	shortURL := model.ShortURL{
		OriginalURL: originalURL,
		ShortCode:   code,
	}

	if err = s.repository.ShortURLRepository.Create(shortURL); err != nil {
		// TODO abstract error layer.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			var regeneratedCode string
			regeneratedCode, err = s.generator.newCode(8)
			if err != nil {
				url.CountUpShortURLCreationCounter("failed")
				return "", err
			}

			//shortURL.ShortCode = regeneratedCode
			if err = s.repository.ShortURLRepository.Create(shortURL); err != nil {
				url.CountUpShortURLCreationCounter("failed")
				slog.Error("can't create short url to DB", "url", originalURL, "error", err)
				return "", err
			}
			url.CountUpShortURLCreationCounter("success")
			return regeneratedCode, nil
		}
		url.CountUpShortURLCreationCounter("failed")
		slog.Error("can't create short url", "url", originalURL, "error", err)
		return "", err
	}

	url.CountUpShortURLCreationCounter("success")
	return code, nil
}

func verifyURL(url string) error {
	if url == "" {
		return errors.New("url is empty")
	}
	if len(url) > 2048 {
		return errors.New("url is too long")
	}
	if strings.Contains(url, " ") {
		return errors.New("url must not contain spaces")
	}

	return nil
}

func (s *ShortURLService) GetOriginalURL(code string) (string, error) {
	// Reject codes that contain a file extension (e.g. ".html", ".png")
	if strings.Contains(code, ".") {
		return "", errors.New("invalid short code")
	}
	if code == "" {
		return "", errors.New("code is required")
	}

	shortURL, err := s.repository.ShortURLRepository.FindShortURLByCode(model.ShortURL{ShortCode: code})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("short code not found")
		}
		slog.Error("can't find short url", "code", code, "error", err)
		return "", errors.New("something went wrong")
	}

	return shortURL.OriginalURL, nil
}
