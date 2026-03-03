APP_NAME=donrat-bot
ECR_REPO?=donrat-bot
IMAGE_TAG?=latest

.PHONY: tidy build run test docker-up docker-down docker-build docker-tag

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

docker-build:
	docker build -t $(ECR_REPO):$(IMAGE_TAG) .

docker-tag:
	@test -n "$(IMAGE_URI)" || (echo "IMAGE_URI is required, e.g. make docker-tag IMAGE_URI=123456789012.dkr.ecr.us-east-1.amazonaws.com/donrat-bot:latest" && exit 1)
	docker tag $(ECR_REPO):$(IMAGE_TAG) $(IMAGE_URI)
