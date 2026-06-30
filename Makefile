.PHONY: run test docker-build setup

setup:
	go mod tidy

run:
	go run ./cmd/main.go

test:
	go test ./... -race -count=1

docker-build:
	docker build -t snaply/api-gateway:latest .
