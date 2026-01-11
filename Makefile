APP_NAME := url-shortener

build:
	go build -o bin/$(APP_NAME) ./cmd

run: build
	./bin/$(APP_NAME)

test:
	go test -v ./... -count=1