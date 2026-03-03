APP_NAME=donrat-bot

.PHONY: tidy build run test docker-up docker-down

tidy:
	go mod tidy

build:
	go build -o bin/$(APP_NAME) ./cmd/bot

run:
	go run ./cmd/bot

test:
	go test ./...

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v
