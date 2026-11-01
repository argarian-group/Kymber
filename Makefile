.PHONY: build run backend dev frontend db up

build:
	cd apps/backend && go build -o build/api ./cmd/api

run: backend

backend: db
	cd apps/backend && go run ./cmd/api

dev: db
	cd apps/backend && go tool air

frontend:
	cd apps/frontend && pnpm install && pnpm dev

db:
	docker compose up -d --wait mariadb

up: db
	$(MAKE) -j2 dev frontend
