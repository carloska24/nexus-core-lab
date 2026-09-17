# Local dashboard development

Run these commands from `web/reference-version`:

```sh
npm ci
npm run dev
```

`npm run dev` performs the following steps:

1. compiles `cmd/api` to an operating-system temporary directory;
2. starts the API on `PORT` or `8080`;
3. waits for a valid `/health` response;
4. starts the Vite development server;
5. shuts down both processes and removes the temporary API executable on
   `Ctrl+C`.

No database or migration is started automatically. Without `DATABASE_URL`, the
API uses MEMORY mode. To use PostgreSQL, configure the environment and apply
migrations from the repository root before starting this command; see
[Local PostgreSQL setup](../../docs/engineering/LOCAL_POSTGRES_SETUP.md).

Useful options:

```sh
# Choose the API port used by the combined runner.
PORT=18080 npm run dev

# Start only Vite against the default API at 127.0.0.1:8080.
npm run dev:frontend

# Start only Vite against another API.
NEXUS_API_TARGET=http://127.0.0.1:18080 npm run dev:frontend

# Forward Vite options through the combined runner.
npm run dev -- --port 5175 --host 127.0.0.1 --strictPort
```

PowerShell sets environment variables with `$env:PORT = '18080'` or
`$env:NEXUS_API_TARGET = 'http://127.0.0.1:18080'` before the npm command.

Build validation:

```sh
npm ci
npm run build
```

The scripts under `tests/` are historical engineering browser checks. They use
an external browser automation runtime that is not declared as a project
dependency, so they are not exposed as a clean-clone npm test command.
