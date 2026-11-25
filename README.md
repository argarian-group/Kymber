<div align="center">
  <img src="./assets/banner.svg" width="100%" alt="Kymber" />

  <p>A project management platform that brings planning, tasks, deploys, collaboration, and progress tracking together in a single workspace.</p>

  <img src="https://img.shields.io/badge/status-early%20development-8B5CF6?style=flat-square" alt="Status: early development" />
  <img src="https://img.shields.io/badge/contributions-welcome-22C55E?style=flat-square" alt="Contributions welcome" />
</div>

> [!NOTE]
> Kymber is in early development.

Kymber shows the whole project in one place: what is planned, what is in progress, and what has reached production.

## Features

- **Projects and tasks:** boards, lists and calendar views, with checklists, labels, comments and assignees.
- **Team:** workspaces, invite links, and owner, admin, member or custom roles.
- **Apps:** MySQL, Redis, any Docker Hub image or a Compose stack, deployed to your servers over SSH, with environments, domains and deploy history.
- **Git deploys:** GitHub, GitLab, Bitbucket or any Git server over SSH, built with a Dockerfile, Nixpacks or as a static site. Push deploys, commit statuses, pull request previews and rollback.
- **Monitoring:** server and app metrics with Prometheus, and alerts on the overview.

## Install

On Debian, Ubuntu, Arch, Fedora, RHEL and their derivatives:

```bash
curl -fsSL https://raw.githubusercontent.com/argarian-group/Kymber/main/install.sh | sudo bash
```

It installs Docker, Buildx and Nixpacks, generates the secrets in `/opt/kymber/.env` and starts Kymber with its build controller. Set `KYMBER_INSTALL_DIR` to install elsewhere.

> [!IMPORTANT]
> Use HTTPS outside a trusted network. Set a panel domain during onboarding, or later under **Settings › System**, then set `TRAEFIK_ACME_EMAIL` in `.env` and open ports 80 and 443.

## Development

Requires Go 1.27.3+, Docker with Compose and Buildx, Make, Node.js and pnpm.

```bash
cp .env.example .env   # then set the passwords and secrets
make up                # MariaDB, API, frontend and build controller
```

Open the URL Vite prints and create the root account. Then register the local server under **Servers**: `make up` sets up its SSH key on first run, asking for `sudo` once. Run it as your normal user.

| Command | What it does |
| --- | --- |
| `Ctrl+C` | Stops the API, frontend and build controller |
| `docker compose stop` | Stops MariaDB, Traefik and Prometheus |
| `docker compose down -v` | Deletes all local data |
| `make dev` / `make frontend` / `make build-worker` | Runs one part on its own |
| `make build` / `make build-server` / `make build-cli` | Compiles the API, the server agent or the `kymber` CLI |

Main `.env` settings:

| Variable | Purpose |
| --- | --- |
| `SERVER_PORT` | API port, `8080` by default |
| `SERVICE_ENCRYPTION_KEY` | Encrypts app settings: `openssl rand -hex 32` |
| `KYMBER_BUILD_WORKER_TOKEN` | Secret shared with the build controller: `openssl rand -hex 32` |
| `KYMBER_GIT_BUILDS_ENABLED` | `true` turns Git builds on |
| `KYMBER_TRUST_PROXY` | `true` only when the API is reachable through a reverse proxy alone |
| `DB_*` | MariaDB connection and passwords |

## Deploy from Git

1. Connect a provider under **Settings › Git**: a GitHub App, an access token, or an SSH deploy key. Public GitHub repositories need none.
2. In the app's **Source** tab, choose the repository, branch and build type, then save and build.
3. For push deploys, add the webhook shown in the Source tab to the repository. GitHub Apps add it themselves.

The GitHub App, webhooks and pull request previews need the panel on a public HTTPS domain. Previews also need a wildcard DNS record, and pull requests from forks are never built.

## CLI

Create a token under **Settings › Account › API tokens**, then:

```bash
make build-cli
apps/backend/build/kymber login https://panel.example.com kym_...
apps/backend/build/kymber apps
apps/backend/build/kymber deploy -wait <app-id>
```

In CI, set `KYMBER_URL` and `KYMBER_TOKEN` instead of running `login`.

## Project structure

```text
apps/backend/    Go API (cmd/api) and server agent (cmd/server)
apps/frontend/   React app (Vite, Tailwind CSS)
scripts/         Build controller, build runner and local SSH setup
assets/          Logo, banner and background
```

## Roadmap

- [x] Accounts, workspaces, projects and tasks
- [x] Apps on local and remote servers, with deploy history
- [x] Git builds, push deploys, pull request previews and rollback
- [ ] Isolated build worker on a separate host
- [ ] Collaboration and progress tracking

## Contributing

Open an issue before anything larger than a small fix. Keep each change focused, follow the code around it, and check before opening a pull request:

```bash
(cd apps/backend && go vet ./... && go test ./...)
(cd apps/frontend && pnpm build)
```

Use [Conventional Commits](https://www.conventionalcommits.org/). Write code and UI text in English, wrap UI text in `t()` and add its translation to `apps/frontend/src/locales/pt-BR.ts`. Never commit secrets, `.env` or local tool folders.

## License

[AGPL-3.0-only](LICENSE). If you offer a modified Kymber over a network, you must share its source code with its users.
