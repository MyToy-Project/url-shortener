package url

import (
	"time"

	"github.com/go-chi/cors"
	"gorm.io/gorm"
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// ShortURL represents a shortened URL in the database
type ShortURL struct {
	UrlID       uint `gorm:"primaryKey"`
	OriginalURL string
	ShortCode   string `gorm:"uniqueIndex"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// ShortURLRequest represents a request to create a shortened URL
type ShortURLRequest struct {
	OriginalURL string `json:"original_url"`
}

// ShortCodeResponse represents a response containing the shortened URL
type ShortCodeResponse struct {
	ShortCode string `json:"short_code"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message"`
}

func (a *App) registerRoutes() {
	a.r.Use(cors.Handler(
		cors.Options{
			AllowedOrigins: []string{"http://localhost:5500", "http://127.0.0.1:5500"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
			MaxAge:         300,
		},
	))
	a.r.Get("/", a.buildIndexServeHandler())

	// Rate limit only the shorten endpoint (per IP)
	// Example: 30 requests per minute with a burst of 1
	a.r.With(rateLimitByIP(30, 1)).Post("/api/shorten", a.buildShortURLCreationHandler())
	a.r.With(rateLimitByIP(30, 5)).Get("/{code}", a.buildGettingShortURLHandler())
}
