# NEXUS Core Lab

**Telecom Core Network Simulation Lab**

NEXUS Core Lab is an educational system for exploring telecom-oriented domain
modeling through a modular Go application, an HTTP API, a concurrent simulator,
PostgreSQL persistence, asynchronous telemetry, and an operational dashboard.
It models useful lifecycle and concurrency problems without claiming to
implement a commercial EPC or 5G Core.

## Overview

The project follows a subscriber and device from provisioning to network
detachment. It is intended to make engineering decisions visible: explicit
state transitions, module boundaries, persistence contracts, safe concurrency,
migrations, observable events, and end-to-end HTTP behavior.

## Why this project exists

Telecom systems provide a concrete setting for practicing backend engineering.
NEXUS uses that setting to demonstrate:

- domain lifecycles and invariants;
- a modular monolith with consumer-owned interfaces;
- concurrent session handling and IP allocation;
- interchangeable MEMORY and PostgreSQL repositories;
- transactional, versioned database migrations;
- bounded asynchronous telemetry;
- an external simulator that only uses the public HTTP API;
- a React dashboard that distinguishes live backend data from local sandbox UI.

## Architecture

`cmd/api` is the composition root. It selects the persistence mode, wires the
domain services, starts telemetry, and exposes one HTTP server.

```mermaid
flowchart TD
    Dashboard[React dashboard] --> API[cmd/api<br/>HTTP composition root]
    Simulator[cmd/simulator<br/>HTTP client] --> API
    API --> Domains[Subscriber / Device / Session]
    Domains --> Network[Network and IP pool]
    Domains --> Telemetry[Bounded telemetry worker]
    Domains --> Repositories[Repository contracts]
    Repositories --> Memory[In-memory repositories]
    Repositories --> PostgreSQL[(PostgreSQL)]
```

See [Architecture](docs/ARCHITECTURE.md), [API reference](docs/API.md),
[ADR-001](docs/adr/ADR-001-modular-monolith.md), and
[ADR-002](docs/adr/ADR-002-radio-technology-as-semantic-metadata.md).

## Hero Flow

```text
Provision Subscriber
        ↓
Activate
        ↓
Register Device
        ↓
Attach
        ↓
Handover
        ↓
Detach
```

The flow can be performed through the operational Subscriber, Device, and
Session pages or through `cmd/simulator`. The CLI simulator calls only the
public HTTP API and can run multiple virtual devices concurrently.

The dashboard's **Simulator / Sandbox** is frontend-only. A browser does not
execute `cmd/simulator` or shell commands.

## Technology stack

- Go 1.27, `net/http`, `database/sql`;
- PostgreSQL 16 with pgx and SQL migrations;
- React 18, TypeScript 5, Vite 6, and Lucide icons;
- Docker Compose for optional local PostgreSQL;
- native Go tests, race detector, and focused browser verification scripts.

## Quick Start — MEMORY

Requirements: Go 1.27, Node.js 22, and npm 10.

```sh
git clone https://github.com/carloska24/nexus-core-lab.git
cd nexus-core-lab/web/reference-version
npm ci
npm run dev
```

Open the URL printed by Vite. `npm run dev` builds and starts the Go API, waits
for `/health`, and starts the dashboard. Without `DATABASE_URL`, data is kept in
memory and is discarded when the process stops. Press `Ctrl+C` to stop both.

## PostgreSQL mode

Docker is required for this local option. From the repository root:

```sh
cp .env.example .env
# Replace change_me in .env with a local password.
docker compose up -d postgres
```

Export a matching connection URL in the terminal that runs Go, apply the
migrations, and start the dashboard:

```sh
export DATABASE_URL='postgres://nexus:change_me@localhost:5433/nexus_core_lab?sslmode=disable'
go run ./cmd/migrate -up
cd web/reference-version
npm run dev
```

PowerShell uses `$env:DATABASE_URL = '...'` instead of `export`. Replace the
placeholder locally and never commit the resulting `.env` or connection URL.
Detailed instructions are in
[Local PostgreSQL setup](docs/engineering/LOCAL_POSTGRES_SETUP.md).

## Dashboard

The canonical frontend is [`web/reference-version`](web/reference-version/).
It provides Overview, Subscribers, Devices, Sessions, Network, Events,
Telemetry, storage diagnostics, Simulator/Sandbox, and Configuration views.
The older mock-oriented files under `web/` are retained only as legacy source
history and are not a runtime dependency of the canonical frontend.

No promotional dashboard screenshot is currently versioned. The map asset is a
local, attributed visual layer rather than an external map service.

## Simulator CLI

With the API running on port 8080, open another terminal at the repository root:

```sh
go run ./cmd/simulator -api http://localhost:8080 -devices 5
```

`-devices` accepts 1–100 virtual devices. Each goroutine executes the complete
Hero Flow and the final report includes server telemetry.

## Testing

Backend validation from the repository root:

```sh
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

PostgreSQL integration tests run when `TEST_DATABASE_URL` points to a migrated,
disposable database. Frontend validation:

```sh
cd web/reference-version
npm ci
npm run build
```

Historical browser scripts under `web/reference-version/tests` were used for
engineering verification. They depend on an external browser runtime and are
not part of the clean-clone Quick Start.

## Project structure

```text
cmd/                    API, migration CLI, and concurrent simulator
internal/               domain modules and platform adapters
migrations/             ordered PostgreSQL up/down migrations
docs/                   architecture, API, ADRs, and engineering notes
web/reference-version/  canonical operational dashboard
web/src/                legacy mock-oriented frontend source
```

## Architectural decisions

The project deliberately uses a modular monolith. MEMORY mode keeps the first
run simple; PostgreSQL demonstrates durable repositories and migrations.
`database/sql` with pgx keeps SQL and transaction behavior explicit. The
telemetry worker is bounded so observability cannot apply unbounded backpressure
to domain operations.

## Educational scope and limitations

- This is not an EPC, 5GC, radio network, or protocol implementation.
- LTE/5G labels are semantic metadata; compatibility is not enforced.
- Cells and the Campinas topology are an illustrative static catalog.
- Recent events are bounded process memory, not durable audit history.
- The IP pool is in memory and is reconstructed from connected PostgreSQL
  sessions during startup.
- Authentication, authorization, multi-user operation, and production deployment
  are intentionally outside the current scope.

## AI-assisted engineering disclosure

This project was developed with AI-assisted engineering workflows. Architecture,
scope, domain decisions, implementation acceptance, and testing were subject to
explicit human review. See the
[engineering guidelines](docs/engineering/AI_ENGINEERING_GUIDELINES.md).

## License

Licensed under the [MIT License](LICENSE). Copyright © 2026 Carlos Pereira.
