.PHONY: build build-server run backend dev dev-run server server-run frontend db init check-user up local-secrets docker-build docker-up docker-agent docker-down

PNPM ?= $(shell command -v pnpm 2>/dev/null || { test -x "$$HOME/.bun/bin/pnpm" && printf '%s/.bun/bin/pnpm' "$$HOME"; })

DEV_ENV = SERVICE_ENCRYPTION_KEY_FILE=$(CURDIR)/.local/service-encryption-key \
	KYMBER_LOCAL_SSH_PRIVATE_KEY_FILE=$(CURDIR)/.local/kymber-local-ssh \
	KYMBER_LOCAL_SSH_HOST_KEY_FILE=/etc/ssh/ssh_host_ed25519_key.pub \
	KYMBER_LOCAL_SSH_HOST=127.0.0.1 \
	TRAEFIK_PANEL_API_TARGET=http://host.docker.internal:8080 \
	TRAEFIK_PANEL_WEB_TARGET=http://host.docker.internal:5173 \
	KYMBER_METRICS_TOKEN_FILE=$(CURDIR)/.local/metrics-token \
	KYMBER_PROMETHEUS_URL=http://127.0.0.1:9090

build:
	cd apps/backend && go build -o build/api ./cmd/api

build-server:
	cd apps/backend && go build -o build/kymber-server ./cmd/server

run: backend

backend: init
	cd apps/backend && env $(DEV_ENV) go run ./cmd/api

dev: init
	$(MAKE) dev-run

dev-run:
	@test -w apps/backend/build/air || test ! -e apps/backend/build/air || { echo "apps/backend/build/air is not writable; repair its ownership before continuing" >&2; exit 1; }
	cd apps/backend && env $(DEV_ENV) go tool air

server:
	cd apps/backend && go run ./cmd/server

server-run: server

frontend:
	@test -n "$(PNPM)" || { echo "pnpm not found; install pnpm or add ~/.bun/bin to PATH" >&2; exit 127; }
	cd apps/frontend && $(PNPM) install && $(PNPM) dev

db:
	docker compose up -d --wait mariadb

init: check-user local-secrets db
	./scripts/setup-local-ssh.sh

check-user:
	@test "$$(id -u)" -ne 0 || { echo "Run make without sudo" >&2; exit 1; }

local-secrets:
	@test "$$(id -u)" -ne 0 || { echo "Run make without sudo" >&2; exit 1; }
	@mkdir -p .local
	@test -w .local || { echo "$(CURDIR)/.local is not writable; fix its ownership before continuing" >&2; exit 1; }
	@chmod 700 .local
	@umask 077; test -s .local/service-encryption-key || openssl rand -hex 32 > .local/service-encryption-key
	@umask 077; test -s .local/metrics-token || openssl rand -hex 32 > .local/metrics-token

docker-build:
	docker compose build

docker-up: init
	docker compose up --build -d

docker-agent:
	docker compose up --build -d server

docker-down:
	docker compose down

up:
	$(MAKE) check-user
	$(MAKE) init
	TRAEFIK_CONFIG_URL=http://host.docker.internal:8080/internal/traefik/panel-domain docker compose up -d traefik
	KYMBER_API_HOST=host.docker.internal:8080 KYMBER_METRICS_TOKEN=$$(cat .local/metrics-token) docker compose up -d prometheus
	SERVICE_ENCRYPTION_KEY_FILE=$(CURDIR)/.local/service-encryption-key \
	KYMBER_SERVER_TOKEN_FILE=$(CURDIR)/.local/server-token \
	KYMBER_LOCAL_SSH_PRIVATE_KEY_FILE=$(CURDIR)/.local/kymber-local-ssh \
	KYMBER_LOCAL_SSH_HOST_KEY_FILE=/etc/ssh/ssh_host_ed25519_key.pub \
	KYMBER_LOCAL_SSH_HOST=127.0.0.1 \
	$(MAKE) -j2 dev-run frontend
