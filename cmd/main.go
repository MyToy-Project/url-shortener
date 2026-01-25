package main

import (
	"url-shortener/internal/url"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	app := url.NewApp()
	app.Run()
}
