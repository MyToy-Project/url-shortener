package url

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// App represents the application for URL-Shortener
type App struct {
	lgr  *slog.Logger
	r    *chi.Mux
	db   *gorm.DB
	stop chan os.Signal
}

// NewApp creates a new instance of the application
func NewApp() *App {
	app := &App{
		lgr: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		})),
		r:    chi.NewRouter(),
		stop: make(chan os.Signal, 1),
	}
	slog.SetDefault(app.lgr)

	// TODO Make it testable
	name := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")

	db, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", host, user, password, name, port)))
	if err != nil {
		app.lgr.Error("can't connect database", "error", err)
		return nil
	}
	err = db.Migrator().AutoMigrate(&ShortURL{})
	if err != nil {
		return nil
	}
	app.db = db

	signal.Notify(app.stop, syscall.SIGINT, syscall.SIGTERM)
	app.registerRoutes()

	return app
}

func (a *App) Run() {
	server := http.Server{
		Addr:    ":8080",
		Handler: http.TimeoutHandler(a.r, 1*time.Second, "request timeout"),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.lgr.Error("can't start server", "error", err)
		}
	}()

	a.lgr.Info("server started")

	<-a.stop

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	if err := server.Shutdown(ctx); err != nil {
		a.lgr.Error("can't shutdown server gracefully", "error", err)
		os.Exit(1)
	}
	a.lgr.Info("server shutdown complete")
}
