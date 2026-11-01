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
> Kymber is in early development. Accounts, workspaces, members and projects run on the API. The overview and each project's apps, deploys and tasks still show sample data.

## Why Kymber

Knowing how a project is doing usually means checking one tool for tasks, another for deploys, and asking around for the rest. Kymber aims to show the whole project in one place: what is planned, what is in progress, and what has reached production.

## Features

**Available**

- **Accounts:** first-run root account, login, optional registration, and profile and password settings.
- **Workspaces:** create, rename and delete workspaces, and switch between them.
- **Team:** owner, admin and member roles, and invite links.
- **Projects:** create, edit and delete projects inside a workspace.
- **Appearance:** theme, accent color and density, saved in the browser.

**In preview, with sample data**

- **Apps and nodes:** apps grouped by node, such as Production and Staging, with their status.
- **Deploys:** recent deploys in each project's overview.
- **Tasks:** board, list and calendar views.

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
| `make build` | Compiles the API to `apps/backend/build/api` |

To check that the API is running:

```bash
curl http://localhost:8080/health
```

```json
{"status":"ok"}
```

To delete all local data, stop the app and run `docker compose down -v`.

## Project structure

```text
apps/
├── backend/
│   ├── cmd/api/         # API entry point
│   └── internal/
│       ├── auth/        # Accounts, sessions, onboarding and invitations
│       ├── database/    # Connection, migrations and IDs
│       ├── logger/      # Colored log output
│       ├── middleware/  # Rate limiting
│       ├── projects/    # Workspaces, members and projects
│       ├── routes/      # HTTP routes
│       └── server/      # HTTP server
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
- [x] Project pages with sample apps, deploys and tasks
- [ ] Apps, nodes and deploys in the API
- [ ] Tasks in the API
- [ ] Collaboration and progress tracking
