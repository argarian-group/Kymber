.PHONY: build build-server run backend dev server frontend db up template-registry

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

frontend:
	cd apps/frontend && pnpm install && pnpm dev

db:
	docker compose up -d --wait mariadb

up: db
	$(MAKE) -j2 dev frontend
