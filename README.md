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
> Kymber is in early development. Accounts, workspaces, members and projects run on the API. Apps run as Docker containers on the servers you register.

## Why Kymber

Knowing how a project is doing usually means checking one tool for tasks, another for deploys, and asking around for the rest. Kymber aims to show the whole project in one place: what is planned, what is in progress, and what has reached production.

## Features

**Available**

- **Overview:** workspace dashboard with project health, late tasks, deadlines and app status.
- **Accounts:** first-run root account, login, optional registration, and profile and password settings.
- **Workspaces:** create, rename and delete workspaces, and switch between them.
- **Team:** owner, admin and member roles, and invite links.
- **Projects:** create, edit and delete projects inside a workspace, each with an optional logo.
- **Apps:** create MySQL and Redis apps from templates, or any public Docker Hub image, then deploy, start, stop and restart them in Docker.
- **Environments:** add environments such as production and dev to a project. Apps in an environment share a private network, and apps can also stay outside any environment.
- **Tasks:** boards with editable lists, draggable cards, a rich text editor for descriptions and comments, checklists, labels, multiple assignees and filters, across board, list and calendar views.
- **Appearance:** theme, accent color and density, saved in the browser.

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
| `KYMBER_TRUST_PROXY` | When `true`, rate limits use the client address sent by the reverse proxy. Set it only when the API is reachable through that proxy alone |
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

Root registers servers under **Servers**. Local setup follows Coolify's host-install approach: run `make up` as your normal user; `scripts/setup-local-ssh.sh` creates a persistent Ed25519 key and uses `sudo` to add its public key to root's `authorized_keys` on first setup only. Do not run `sudo make up`, since the frontend tools should run as your user. For a direct Docker Compose install, run the setup script before `docker compose up --build -d`. The host needs `sshd`, Docker, and its Ed25519 host public key at `/etc/ssh/ssh_host_ed25519_key.pub`. The backend stores the private key encrypted and executes Docker actions over SSH.

For a remote server, provide its SSH host key during registration and install the displayed public key in the remote user's `~/.ssh/authorized_keys`. Deployments and service actions then run directly over SSH; the agent is not used for deployments.

An app remains assigned to the server selected at its first deploy. The old heartbeat agent is optional and only reports presence; it has no task-claim or Docker-socket access.

To publish a web app or the panel over HTTPS, set `TRAEFIK_ACME_EMAIL` and a random `TRAEFIK_CONFIG_TOKEN` in `.env`, point the public DNS record to this machine, and allow inbound ports 80 and 443. In the app's **Connection** section, set a domain and save it, then deploy the app. Traefik obtains and renews HTTPS certificates automatically. Public app domains currently support HTTP application services on the local server; remote servers and database services are not exposed.

The root-account onboarding also offers an optional domain for the Kymber panel. It can later be edited or removed in **Settings → System**. DNS checking only reports whether the hostname resolves; create or update its A/AAAA record at your DNS provider and verify that it points to this server. Cloudflare changes are not automated.

Apps can alternatively publish an HTTP host port from their **Connection** settings. Choose an unused port, save, then deploy; access it at `http://<server-ipv4>:<port>`. This bypasses the Docker network, exposes the app to anyone who can reach that host port, and does not enable TLS. Domain and host-port publishing are mutually exclusive.

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
│       └── templates/   # MySQL, Redis and Docker Hub image templates
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

## License

Kymber is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0-only). If you modify Kymber and offer it to users over a network, you must make your modified source code available to them.
