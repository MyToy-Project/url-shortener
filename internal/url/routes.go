package url

import (
	"github.com/go-chi/cors"
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

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
