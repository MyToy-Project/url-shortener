package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	health "url-shortener/internal/url/handler/health_check"
	"url-shortener/internal/url/handler/index"
	"url-shortener/internal/url/handler/short_url"
	"url-shortener/internal/url/middleware"
	"url-shortener/internal/url/repository"
	"url-shortener/internal/url/repository/model"
	"url-shortener/internal/url/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func CreateRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(
		cors.Options{
			AllowedOrigins: []string{"http://localhost:5500", "http://127.0.0.1:5500"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
			MaxAge:         300,
		},
	))
	r.HandleFunc("/health", health.HandleHealthCheck)
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/", index.HandleIndex)

	db := initDB()
	repo := repository.NewRepository(db)
	g := service.NewRandomCodeGenerator()
	svc := service.NewService(g, repo)

	// Rate limit only the shorten endpoint (per IP)
	// Example: 30 requests per minute with a burst of 1

	suh := short_url.NewHandler(svc)
	r.With(middleware.RateLimitByIP(30, 1)).Post("/api/shorten", suh.HandleShortURLCreation)
	r.With(middleware.RateLimitByIP(30, 5)).Get("/{code}", suh.HandleGettingOriginalURL)

	return r
}

func initDB() *gorm.DB {
	name := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")

	db, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", host, user, password, name, port)))
	if err != nil {
		slog.Error("can't connect database", "error", err)
		os.Exit(1)
	}
	err = db.Migrator().AutoMigrate(&model.ShortURL{})
	if err != nil {
		slog.Error("can't connect database", "error", err)
		os.Exit(1)
	}

	return db
}
