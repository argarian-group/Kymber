<div align="center">
  <img src="./assets/banner.svg" width="100%" alt="Kymber" />

  <p>A project management platform that brings planning, tasks, deploys, collaboration, and progress tracking together in a single workspace.</p>

  <p>
    <a href="#why-kymber">Why Kymber</a> ·
    <a href="#planned-features">Planned features</a> ·
    <a href="#getting-started">Getting started</a> ·
    <a href="#project-structure">Project structure</a> ·
    <a href="#roadmap">Roadmap</a>
  </p>

  <img src="https://img.shields.io/badge/status-early%20development-8B5CF6?style=flat-square" alt="Status: early development" />
  <img src="https://img.shields.io/badge/contributions-welcome-22C55E?style=flat-square" alt="Contributions welcome" />
</div>

> [!NOTE]
> Kymber is at an early stage. Today the repository contains a backend skeleton with a health check and a dashboard that runs on sample data. Everything under [Planned features](#planned-features) describes the vision, is not implemented yet, and may change.

## Why Kymber

Knowing how a project is doing usually means checking one tool for tasks, another for deploys, and asking around for the rest. Kymber aims to show the whole project in one place: what is planned, what is in progress, and what has reached production.

## Planned features

- **Project overview:** the status and progress of each project at a glance.
- **Planning and tasks:** organize work, assign it, and follow it through to completion.
- **Deploy visibility:** see what was deployed, where, and when, next to the work it delivers.
- **Collaboration:** keep the team and its decisions in the same workspace as the work.
- **Progress tracking:** follow how a project advances over time.

## Getting started

### Requirements

- [Go](https://go.dev/dl/) 1.27.3 or newer
- Make
- [Node.js](https://nodejs.org/) and [pnpm](https://pnpm.io/) for the frontend

### Run everything

```bash
cp .env.example .env
make up
```

This starts MariaDB in Docker, then the API with live reload and the dashboard. Press `Ctrl+C` to stop the API and the dashboard. MariaDB keeps running until you run `docker compose stop`.

### Run the API

```bash
cp .env.example .env
docker compose up -d mariadb
make run
```

The API reads `SERVER_PORT` from `.env` and listens on port `8080` when it is not set. To check that it is running:

```bash
curl http://localhost:8080/health
```

```json
{"status":"ok"}
```

To compile a binary to `apps/backend/build/api`, run `make build`.

### Run the dashboard

```bash
make frontend
```

Vite prints the local URL and forwards `/api` to the API on `SERVER_PORT`. Login and projects use the API; the overview still shows sample data.

## Project structure

```text
apps/
├── backend/
│   ├── cmd/api/      # API entry point
│   └── internal/
│       ├── logger/   # Colored log output
│       └── routes/   # HTTP routes
└── frontend/src/      # React app (Vite, Tailwind CSS)
    ├── components/   # Shared layout and UI components
    └── features/     # One folder per product area
assets/               # Logo, banner and background
```

## Roadmap

- [x] Backend skeleton with a health check
- [ ] Projects and tasks
- [ ] Deploy visibility
- [ ] Collaboration and progress tracking
- [x] Dashboard layout with sample data
- [ ] Connect the dashboard to the API
