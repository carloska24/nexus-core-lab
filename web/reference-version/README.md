# NEXUS Core Lab — Canonical Dashboard

This is the official operational frontend for NEXUS Core Lab. It is a
React/TypeScript/Vite application integrated with the Go HTTP API.

The older mock-oriented frontend under the parent `web/` directory is retained
as legacy source history. This application does not import its code, assets,
configuration, lockfile, or dependencies.

## Requirements

- Node.js 22+
- npm 10+
- Go 1.27

## Install and run

From this directory:

```sh
npm ci
npm run dev
```

The combined development command builds the Go API into a temporary directory,
starts it on port 8080 by default, waits for `/health`, and then starts Vite.
Without `DATABASE_URL`, the API uses temporary in-memory repositories. Press
`Ctrl+C` to stop both processes.

To run only the frontend against an API already available at
`http://127.0.0.1:8080`:

```sh
npm run dev:frontend
```

Set `NEXUS_API_TARGET` before `npm run dev:frontend` to use another development
API origin.

## Build

```sh
npm ci
npm run build
```

The lockfile in this directory makes installation independent of
`../node_modules` and the legacy frontend manifest.

## Product boundaries

- Operational pages use the Go API for Subscribers, Devices, Sessions,
  Telemetry, recent events, IP pool, and storage diagnostics.
- Network cells and the Campinas map are an illustrative local catalog.
- The Simulator/Sandbox page is frontend-only and never executes the Go CLI.
- Historical Gate reports remain versioned as engineering records; they are not
  current operating instructions.

Start with the repository [README](../../README.md). See the
[architecture](../../docs/ARCHITECTURE.md), [API reference](../../docs/API.md),
[development notes](DEV.md), and
[PostgreSQL setup](../../docs/engineering/LOCAL_POSTGRES_SETUP.md).
