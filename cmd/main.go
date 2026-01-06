package main

import (
	"log"
	"url-shortener/internal/url"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("can't load env variables")
		return
	}
	app := url.NewApp()
	app.Run()
}
