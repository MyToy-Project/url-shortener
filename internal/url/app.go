package url

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// App represents the application for URL-Shortener
type App struct {
	lgr  *slog.Logger
	h    http.Handler
	stop chan os.Signal
}

// NewApp creates a new instance of the application
func NewApp(router http.Handler) *App {
	app := &App{
		lgr: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		})),
		h:    router,
		stop: make(chan os.Signal, 1),
	}
	slog.SetDefault(app.lgr)
	signal.Notify(app.stop, syscall.SIGINT, syscall.SIGTERM)

	return app
}

func (a *App) Run() {
	server := http.Server{
		Addr:    ":8080",
		Handler: http.TimeoutHandler(a.h, 1*time.Second, "request timeout"),
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
