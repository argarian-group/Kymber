.PHONY: build run backend

build:
	cd apps/backend && go build -o build/api ./cmd/api

run: backend

backend:
	cd apps/backend && go run ./cmd/api
