.PHONY: build build-server run backend dev server frontend db up template-registry local-secrets dev-run server-run docker-build docker-up docker-agent docker-down

PNPM ?= $(shell command -v pnpm 2>/dev/null || { test -x "$$HOME/.bun/bin/pnpm" && printf '%s/.bun/bin/pnpm' "$$HOME"; })

build: template-registry
	cd apps/backend && go build -o build/api ./cmd/api

build-server: template-registry
	cd apps/backend && go build -o build/kymber-server ./cmd/server

run: backend

backend: export SERVICE_ENCRYPTION_KEY_FILE := $(CURDIR)/.local/service-encryption-key
backend: export KYMBER_LOCAL_SSH_PRIVATE_KEY_FILE := $(CURDIR)/.local/kymber-local-ssh
backend: export KYMBER_LOCAL_SSH_HOST_KEY_FILE := /etc/ssh/ssh_host_ed25519_key.pub
backend: export KYMBER_LOCAL_SSH_HOST := 127.0.0.1
backend: db template-registry local-secrets
	./scripts/setup-local-ssh.sh
	cd apps/backend && go run ./cmd/api

server: template-registry
	cd apps/backend && go run ./cmd/server

dev: export SERVICE_ENCRYPTION_KEY_FILE := $(CURDIR)/.local/service-encryption-key
dev: export KYMBER_LOCAL_SSH_PRIVATE_KEY_FILE := $(CURDIR)/.local/kymber-local-ssh
dev: export KYMBER_LOCAL_SSH_HOST_KEY_FILE := /etc/ssh/ssh_host_ed25519_key.pub
dev: export KYMBER_LOCAL_SSH_HOST := 127.0.0.1
dev: db template-registry local-secrets
	./scripts/setup-local-ssh.sh
	cd apps/backend && go tool air

template-registry:
	cd apps/backend && go run ./cmd/template-registry

local-secrets:
	@test "$$(id -u)" -ne 0 || { echo "Run make without sudo; local development tools must run as your user" >&2; exit 1; }
	mkdir -p .local
	@test -w .local || { echo "$(CURDIR)/.local is not writable; repair its ownership with: sudo chown -R $(USER):$(shell id -gn) $(CURDIR)/.local" >&2; exit 1; }
	chmod 700 .local
	umask 077; test -s .local/service-encryption-key || openssl rand -hex 32 > .local/service-encryption-key

docker-build:
	docker compose build

docker-up:
	./scripts/setup-local-ssh.sh
	docker compose up --build -d

docker-agent:
	docker compose up --build -d server

docker-down:
	docker compose down

frontend:
	@test -n "$(PNPM)" || { echo "pnpm not found; install pnpm or add ~/.bun/bin to PATH" >&2; exit 127; }
	cd apps/frontend && $(PNPM) install && $(PNPM) dev

db:
	docker compose up -d --wait mariadb

dev-run: export SERVICE_ENCRYPTION_KEY_FILE := $(CURDIR)/.local/service-encryption-key
dev-run: export KYMBER_LOCAL_SSH_PRIVATE_KEY_FILE := $(CURDIR)/.local/kymber-local-ssh
dev-run: export KYMBER_LOCAL_SSH_HOST_KEY_FILE := /etc/ssh/ssh_host_ed25519_key.pub
dev-run: export KYMBER_LOCAL_SSH_HOST := 127.0.0.1
dev-run:
	@test "$$(id -u)" -ne 0 || { echo "Run make without sudo" >&2; exit 1; }
	@test ! -e apps/backend/build/air -o -w apps/backend/build/air || { echo "apps/backend/build/air is not writable; repair its ownership with: sudo chown -R $(USER):$(shell id -gn) $(CURDIR)/apps/backend/build" >&2; exit 1; }
	cd apps/backend && go tool air

server-run:
	cd apps/backend && go run ./cmd/server

up: export SERVICE_ENCRYPTION_KEY_FILE := $(CURDIR)/.local/service-encryption-key
up: export KYMBER_SERVER_TOKEN_FILE := $(CURDIR)/.local/server-token
up: export KYMBER_LOCAL_SSH_PRIVATE_KEY_FILE := $(CURDIR)/.local/kymber-local-ssh
up: export KYMBER_LOCAL_SSH_HOST_KEY_FILE := /etc/ssh/ssh_host_ed25519_key.pub
up: export KYMBER_LOCAL_SSH_HOST := 127.0.0.1
up:
	@test "$$(id -u)" -ne 0 || { echo "Run make up without sudo" >&2; exit 1; }
	$(MAKE) db template-registry local-secrets
	./scripts/setup-local-ssh.sh
	$(MAKE) -j2 dev-run frontend
