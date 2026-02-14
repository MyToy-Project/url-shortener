package main

import (
	"fmt"
	"log/slog"
	"os"
	"url-shortener/internal/url"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

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
	err = db.Migrator().AutoMigrate(&url.ShortURL{})
	if err != nil {
		slog.Error("can't connect database", "error", err)
		os.Exit(1)
	}

	app := url.NewApp(db)
	app.Run()
}
