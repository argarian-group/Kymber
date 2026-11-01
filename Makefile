.PHONY: build build-server run backend dev server frontend db up template-registry docker-build docker-up docker-agent docker-down

build: template-registry
	cd apps/backend && go build -o build/api ./cmd/api

build-server: template-registry
	cd apps/backend && go build -o build/kymber-server ./cmd/server

run: backend

backend: db template-registry
	cd apps/backend && go run ./cmd/api

server: template-registry
	cd apps/backend && go run ./cmd/server

dev: db template-registry
	cd apps/backend && go tool air

template-registry:
	cd apps/backend && go run ./cmd/template-registry

docker-build:
	docker compose build

docker-up:
	docker compose up --build -d

docker-agent:
	docker compose up --build -d server

docker-down:
	docker compose down

frontend:
	cd apps/frontend && pnpm install && pnpm dev

db:
	docker compose up -d --wait mariadb

up: db
	$(MAKE) -j2 dev frontend
