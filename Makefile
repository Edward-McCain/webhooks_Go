.PHONY: tidy test test-race vet fmt build api worker seed compose-up compose-down

tidy:
	cd backend && go mod tidy

fmt:
	cd backend && gofmt -w .

vet:
	cd backend && go vet ./...

test:
	cd backend && go test ./...

test-race:
	cd backend && go test -race ./...

build:
	cd backend && go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker && go build -o bin/seed ./cmd/seed

api:
	cd backend && go run ./cmd/api

worker:
	cd backend && go run ./cmd/worker

seed:
	cd backend && go run ./cmd/seed

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down -v
