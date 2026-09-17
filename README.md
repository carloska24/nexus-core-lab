# NEXUS Core Lab

**Interactive telecom core network simulation and operations lab**

NEXUS Core Lab is a portfolio project for exploring telecom-oriented domain
modeling through a modular Go backend, a React operations dashboard,
PostgreSQL repositories, concurrent session handling, telemetry, and a real
cartographic view of Campinas. It demonstrates engineering decisions and
failure behavior; it does not claim to implement a commercial EPC or 5G Core.

## Live Demo

**Live Demo URL will be added after deployment validation.**

The public build runs an isolated, memory-only workspace for each anonymous
visitor. No account, personal information, or location permission is required.

## What is NEXUS Core Lab?

The system follows a simulated subscriber and device across a complete network
session. It makes domain transitions, persistence boundaries, IP allocation,
telemetry, and operational status visible through one interface.

The official Hero Flow is:

```text
Provision → Activate → Register → Attach → Handover → Detach
```

Use **Subscribers** to provision and activate an identity, **Devices** to
register an IMEI, and **Sessions** to attach, hand over, and detach. The
Overview and Network pages update from the same API state.

## Architecture

`cmd/api` is the composition root. It selects the storage mode, wires the
domain services, starts telemetry, and exposes one HTTP server.

```mermaid
flowchart LR
    Browser[React dashboard] -->|same-origin HTTP| API[Go HTTP server]
    Simulator[Concurrent CLI simulator] -->|HTTP| API
    API --> Domains[Subscriber / Device / Session]
    Domains --> Network[Cells and IP pool]
    Domains --> Telemetry[Bounded telemetry worker]
    Domains --> Repositories[Repository contracts]
    Repositories --> Memory[(MEMORY)]
    Repositories --> PostgreSQL[(PostgreSQL)]
```

See [Architecture](docs/ARCHITECTURE.md), [API reference](docs/API.md),
[ADR-001](docs/adr/ADR-001-modular-monolith.md), and
[ADR-002](docs/adr/ADR-002-radio-technology-as-semantic-metadata.md).

## Public Demo Architecture

The production container compiles the canonical frontend and serves it from
the Go process. API and SPA share one origin, so the public build needs no
internal reverse proxy or Node.js runtime.

```mermaid
flowchart TD
    Visitor[Anonymous visitor] -->|HTTPS| Host[Container host]
    Host --> Go[Go HTTP server]
    Go --> Static[React SPA and hashed assets]
    Go --> Context[Visitor-isolated demo context]
    Context --> Memory[(In-memory repositories)]
    Static --> Maps[OpenFreeMap vector tiles]
```

The demo cookie is an opaque random identifier. Each visitor receives separate
repositories, telemetry, IP allocation, limits, and reset behavior. Contexts
expire by idle and absolute TTL and disappear when the process restarts.

## Real vs Simulated

| Real software behavior | Simulated lab data |
|---|---|
| Go HTTP API and domain validation | IMSI, MSISDN, and IMEI values |
| Repository contracts and PostgreSQL adapters | Subscriber and Device identities |
| Subscriber, Device, and Session lifecycles | Telecom cell and device coordinates |
| Concurrency-safe IP allocation | LTE/5G infrastructure and coverage |
| Telemetry and recent events | Movement and RF behavior |
| Visitor isolation and reset | Radio access behavior |
| MapLibre rendering and Campinas cartography | Illustrative antenna placement |

## Campinas Map

The Network view uses MapLibre GL JS with OpenFreeMap vector tiles. Campinas
streets and place names are real cartography; telecom cells, connected-device
positions, coverage, and paths are illustrative. The application does not
request geolocation and does not contain real antenna coordinates. If the tile
service is unavailable, a local dark fallback preserves the network topology.

## Visitor Isolation and Demo Reset

Public Demo Mode stores each visitor's state only in process memory. The
**Configuration → Reset Demo** action clears only the current anonymous
visitor's records. Other visitors are unaffected. A container restart clears
all public demo contexts by design; the UI and documentation do not imply
durable public persistence.

## Security and Privacy

Public Demo Mode includes:

- an anonymous `HttpOnly`, `SameSite=Strict` cookie, marked `Secure` over HTTPS;
- origin validation for state-changing requests;
- request body, resource, context, and mutation-rate limits;
- a restrictive Content Security Policy and blocked geolocation, camera, and
  microphone permissions;
- no permissive CORS policy, authentication secret, PII, or GPS collection;
- visitor-local reset and bounded TTL cleanup.

`PUBLIC_DEMO_MODE=true` is intentionally incompatible with `DATABASE_URL`.

## Technology Stack

- Go 1.27, `net/http`, `database/sql`, and pgx;
- PostgreSQL 16 with versioned SQL migrations;
- React 18, TypeScript 5, Vite 6, Lucide, MapLibre GL JS 6;
- Docker multi-stage build with a non-root distroless runtime;
- Go tests, race detector, Node tests, and browser verification.

## Local Development

Requirements: Go 1.27, Node.js 22, and npm 10.

```sh
git clone https://github.com/carloska24/nexus-core-lab.git
cd nexus-core-lab/web/reference-version
npm ci
npm run dev
```

Open the URL printed by Vite. The development script starts the Go API, waits
for `/health`, and starts Vite with a same-path proxy. Without `DATABASE_URL`,
the local API uses shared process memory. Press `Ctrl+C` to stop both services.

To exercise visitor isolation locally, run the API with
`PUBLIC_DEMO_MODE=true`; do not set `DATABASE_URL` in that terminal.

## Docker

Docker is the shortest production-like path and requires no host Go or Node.js:

```sh
docker build -t nexus-core-lab:local .
docker run --rm -p 8080:8080 \
  -e PUBLIC_DEMO_MODE=true \
  nexus-core-lab:local
```

Open `http://localhost:8080`. The image defaults to `PORT=8080`,
`PUBLIC_DEMO_MODE=true`, and `NEXUS_WEB_DIR=/app/web`. A platform may override
`PORT`. Do not configure `DATABASE_URL` for a public demo deployment.

The final image contains the statically linked application and compiled web
assets only. Node.js, npm, the Go toolchain, source control metadata, local
environment files, and host dependencies are not copied into the runtime.

## PostgreSQL Local and Integration Mode

From the repository root:

```sh
cp .env.example .env
# Replace change_me with a local-only password.
docker compose up -d postgres
export DATABASE_URL='postgres://nexus:change_me@localhost:5433/nexus_core_lab?sslmode=disable'
go run ./cmd/migrate -up
cd web/reference-version
npm run dev
```

PowerShell uses `$env:DATABASE_URL = '...'` instead of `export`. Never commit
the resulting `.env` or connection URL. See
[Local PostgreSQL setup](docs/engineering/LOCAL_POSTGRES_SETUP.md).

## Simulator CLI

With the API running on port 8080:

```sh
go run ./cmd/simulator -api http://localhost:8080 -devices 5
```

The CLI calls only the public HTTP API and runs concurrent Hero Flows. The
dashboard's **Simulator / Sandbox** page is intentionally frontend-only.

## Testing

Backend validation:

```sh
gofmt -w .
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
npm run test:map
npm run build
```

Production image validation:

```sh
docker build -t nexus-core-lab:local .
```

## Continuous Integration

The GitHub Actions workflow checks Go formatting, vet, build, tests, the race
detector, PostgreSQL-backed tests, clean frontend installation, map presentation
tests, the Vite production build, and the Docker image build. It does not deploy
or require cloud credentials.

## Repository Structure

```text
cmd/                    API, migration CLI, and concurrent simulator
internal/               domain modules and platform adapters
migrations/             ordered PostgreSQL up/down migrations
docs/                   architecture, API, ADRs, and engineering notes
web/reference-version/  canonical operational dashboard
web/src/                retained legacy mock-oriented frontend source
Dockerfile              reproducible single-service production image
```

## Engineering Decisions

- A modular monolith keeps domain boundaries visible without distributed-system
  infrastructure that the lab does not need.
- Consumer-owned interfaces prevent domain packages from leaking types across
  module boundaries.
- MEMORY mode supports a zero-setup demo; PostgreSQL demonstrates durable
  repositories, migration validation, and startup state reconstruction.
- Static frontend files remain on the final container filesystem. This keeps the
  Go build independent from generated web output while preserving a single
  runtime process and image.
- The bounded telemetry worker avoids unbounded backpressure on domain writes.

## Limitations

- This is not an EPC, 5GC, radio network, or telecom protocol implementation.
- LTE/5G labels are semantic metadata; RF compatibility is not enforced.
- Public demo data is intentionally ephemeral and anonymous.
- Recent events are bounded process memory rather than durable audit history.
- The public build has no authentication or multi-tenant user accounts; its
  isolation boundary is the anonymous demo cookie and resource limits.
- OpenFreeMap availability affects live tiles; the local topology fallback
  remains available.

## AI-Assisted Engineering Disclosure

This project was developed with AI-assisted engineering workflows.
Architecture, scope, domain decisions, implementation acceptance, security
boundaries, and testing were subject to explicit human review. See the
[engineering guidelines](docs/engineering/AI_ENGINEERING_GUIDELINES.md).

## License

Licensed under the [MIT License](LICENSE). Copyright © 2026 Carlos Pereira.
