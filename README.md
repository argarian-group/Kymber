<div align="center">
  <img src="./assets/banner.svg" width="100%" alt="Kymber" />

  <p>A project management platform that brings planning, tasks, deploys, collaboration, and progress tracking together in a single workspace.</p>

  <p>
    <a href="#why-kymber">Why Kymber</a> ·
    <a href="#features">Features</a> ·
    <a href="#getting-started">Getting started</a> ·
    <a href="#project-structure">Project structure</a> ·
    <a href="#roadmap">Roadmap</a>
  </p>

  <img src="https://img.shields.io/badge/status-early%20development-8B5CF6?style=flat-square" alt="Status: early development" />
  <img src="https://img.shields.io/badge/contributions-welcome-22C55E?style=flat-square" alt="Contributions welcome" />
</div>

> [!NOTE]
> Kymber is in early development. Accounts, workspaces, members and projects run on the API. Apps run as Docker containers on the servers you register. The workspace overview still shows sample data.

## Why Kymber

Knowing how a project is doing usually means checking one tool for tasks, another for deploys, and asking around for the rest. Kymber aims to show the whole project in one place: what is planned, what is in progress, and what has reached production.

## Features

**Available**

- **Accounts:** first-run root account, login, optional registration, and profile and password settings.
- **Workspaces:** create, rename and delete workspaces, and switch between them.
- **Team:** owner, admin and member roles, and invite links.
- **Projects:** create, edit and delete projects inside a workspace, each with an optional logo.
- **Apps:** create MySQL and Redis apps from templates, then deploy, start, stop and restart them in Docker.
- **Tasks:** boards with editable lists, draggable cards, a rich text editor for descriptions and comments, checklists, labels, multiple assignees and filters, across board, list and calendar views.
- **Appearance:** theme, accent color and density, saved in the browser.

**In preview, with sample data**

- **Overview:** workspace dashboard with projects, deadlines and deploys.

## Getting started

### Requirements

- [Go](https://go.dev/dl/) 1.27.3 or newer
- [Docker](https://docs.docker.com/get-docker/) with Docker Compose, for MariaDB
- Make
- [Node.js](https://nodejs.org/) and [pnpm](https://pnpm.io/) for the frontend

### Configuration

Copy the template and change the passwords before using it outside your machine:

```bash
cp .env.example .env
```

| Variable | Purpose |
| --- | --- |
| `SERVER_PORT` | API port, `8080` when not set |
| `SECURE_ONBOARD` | When `true`, the API listens only on localhost until the root account exists |
| `SERVICE_ENCRYPTION_KEY` | Encrypts app settings. Required to create apps. Generate it with `openssl rand -hex 32` |
| `KYMBER_API_URL` | API address used by a server |
| `KYMBER_SERVER_TOKEN` | Token shown when you register a server |
| `DB_HOST`, `DB_PORT`, `DB_NAME` | MariaDB address and database |
| `DB_USER`, `DB_PASSWORD` | MariaDB application user |
| `DB_ROOT_PASSWORD` | MariaDB root password |

### Run everything

```bash
make up
```

This starts MariaDB in Docker and waits until it is ready, then runs the API with live reload and the frontend. Vite prints the local URL and forwards `/api` to the API on `SERVER_PORT`. Migrations run when the API starts.

The first visit asks you to create the root account. Registration starts off, and root can turn it on under **Settings > System**.

Press `Ctrl+C` to stop the API and the frontend. MariaDB keeps running until you run `docker compose stop`.

> [!IMPORTANT]
> The session cookie is `Secure`, so logging in from anywhere other than `localhost` requires HTTPS.

### Run only one part

| Command | What it runs |
| --- | --- |
| `make run` | MariaDB and the API |
| `make dev` | MariaDB and the API with live reload |
| `make frontend` | The frontend |
| `make server` | A server that runs apps in Docker |
| `make build-server` | Compiles a server to `apps/backend/build/kymber-server` |
| `make build` | Compiles the API to `apps/backend/build/api` |

To check that the API is running:

```bash
curl http://localhost:8080/health
```

```json
{"status":"ok"}
```

### Run apps

Root registers servers under **Servers**, which shows the token once. An app stays on the server chosen at its first deploy.

To run a server on this machine:

```bash
KYMBER_SERVER_TOKEN=<token> make server
```

To run a server on another machine with Docker, compile it with `make build-server`, copy `apps/backend/build/kymber-server` there, then run:

```bash
KYMBER_API_URL=https://<kymber-address> KYMBER_SERVER_TOKEN=<token> ./kymber-server
```

The API must be reachable from that machine. Addresses other than this machine require HTTPS.

To delete all local data, stop the app and run `docker compose down -v`.

## Project structure

```text
apps/
├── backend/
│   ├── cmd/
│   │   ├── api/         # API entry point
│   │   ├── server/      # Server that runs apps in Docker
│   │   └── template-registry/  # Generates the template catalog
│   └── internal/
│       ├── agent/       # API used by servers
│       ├── auth/        # Accounts, sessions, onboarding and invitations
│       ├── database/    # Connection, migrations and IDs
│       ├── logger/      # Colored log output
│       ├── middleware/  # Rate limiting
│       ├── projects/    # Workspaces, members, projects, apps and servers
│       ├── routes/      # HTTP routes
│       ├── secrets/     # Encryption of app settings
│       ├── server/      # HTTP server
│       └── templates/   # MySQL and Redis templates
└── frontend/src/        # React app (Vite, Tailwind CSS)
    ├── components/      # Shared layout and UI components
    ├── features/        # One folder per product area, with pages and components
    └── lib/             # API client and shared hooks
assets/                  # Logo, banner and background
```

## Roadmap

- [x] Backend skeleton with a health check
- [x] Accounts, sessions and onboarding
- [x] Workspaces, members and invitations
- [x] Projects
- [x] Connect accounts, workspaces and projects to the API
- [x] Database apps deployed to local and remote servers
- [ ] Application apps and deploy history
- [x] Tasks in the API
- [ ] Collaboration and progress tracking
