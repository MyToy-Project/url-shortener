package main

import (
	"url-shortener/internal/url"
	"url-shortener/internal/url/router"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	r := router.CreateRouter()

	app := url.NewApp(r)
	app.Run()
}
