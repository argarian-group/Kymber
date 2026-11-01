.PHONY: build run backend dev

build:
	cd apps/backend && go build -o build/api ./cmd/api

run: backend

backend:
	cd apps/backend && go run ./cmd/api

dev:
	cd apps/backend && go tool air
