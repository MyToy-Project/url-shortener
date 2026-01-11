APP_NAME := url-shortener

build:
	go build -o bin/$(APP_NAME) ./cmd

run:
	build
	./bin/app

test:
	go test -v ./... -count=1