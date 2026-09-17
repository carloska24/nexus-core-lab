# NEXUS Core Lab — Gate 12 Portfolio Release Readiness

## 1. Baseline

- Branch: `feat/demo-ui`
- Initial HEAD: `2648b2e37f7a2689f7308efc0c41071649b6cee4`
- Initial working tree: clean
- Scope: documentation, reproducibility, CI, licensing, and validation only
- Commit/push: not performed

## 2. Files changed

Created:

- `README.md`
- `LICENSE`
- `.github/workflows/ci.yml`
- `docs/API.md`
- `docs/ARCHITECTURE.md`
- `web/reference-version/package-lock.json`
- this Gate report

Updated:

- `docs/PROJECT_VISION.md`
- `docs/pt-BR/PROJECT_VISION.md`
- `docs/adr/ADR-001-modular-monolith.md`
- `docs/engineering/HUMAN_ENGINEERING_REVIEW.md`
- `docs/engineering/LOCAL_POSTGRES_SETUP.md`
- `web/reference-version/README.md`
- `web/reference-version/DEV.md`
- `web/reference-version/package.json`

No product endpoint, domain behavior, migration, or frontend feature was added.

## 3. Canonical README

The new root `README.md` is the public entry point. It explains the educational telecom simulation scope, real engineering capabilities, stack, execution modes, testing, project structure, decisions, limitations, AI-assisted workflow, and license without overstating the implementation.

## 4. Architecture documentation

`docs/ARCHITECTURE.md` documents the modular monolith, `cmd/api` composition root, dependency direction, domain modules, MEMORY/PostgreSQL persistence, telemetry worker, recent-event semantics, IP pool, CLI simulator, dashboard, and intentional boundaries.

## 5. API documentation

`docs/API.md` centrally documents health, telemetry, storage, IP pool, recent events, subscribers and lifecycle operations, global/subscriber device collections, sessions, attach, handover, detach, common errors, and the Hero Flow.

## 6. Mermaid

Two small Mermaid diagrams are present: one in the public README and one in the architecture document. Their syntax and relative links were checked. The diagrams represent the implemented modular monolith rather than microservices.

## 7. Canonical frontend

`web/reference-version/` is explicitly identified as the canonical operational frontend. Its source, assets, Vite configuration, scripts, package manifest, and lockfile are self-contained.

## 8. Legacy frontend decision

The older `web/src`, root `web/package.json`, and related files were classified as legacy and are not imported by the canonical frontend. Their removal would improve repository clarity, but deletion was not completed because the automated approval review rejected the destructive operation. The public documentation now clearly distinguishes the canonical frontend and prevents ambiguity. This is a P2 cleanup, not a publication blocker.

## 9. Lockfile

The canonical frontend now has its own npm lockfile (`lockfileVersion: 3`) generated normally from its package manifest. Build-only packages are declared as development dependencies, while React and runtime libraries remain regular dependencies.

## 10. Independent npm installation

The canonical frontend files were copied to a temporary directory outside the repository without any parent `node_modules`. In that isolated copy:

```text
npm ci          PASS — 72 packages installed, 0 vulnerabilities
npm run build   PASS — 1,616 modules transformed
```

The temporary copy was removed after validation. This proves the canonical frontend does not rely on `../node_modules` or the legacy package manifest.

## 11. Continuous integration

`.github/workflows/ci.yml` defines one Linux job with:

- PostgreSQL 16 disposable service container;
- migrations before backend validation;
- `go vet`, normal tests, race tests, and build;
- Node.js 22;
- `npm ci` and `npm run build` in `web/reference-version`.

Only test-local PostgreSQL credentials are present. No deployment, registry, cloud, or release automation was introduced. A hosted CI run can occur only after a future approved commit and push.

## 12. License

The repository now contains the standard MIT License with `Copyright (c) 2026 Carlos Pereira`. The public README links to it.

## 13. Obsolete documentation corrected

- ADR-001 no longer lists an unimplemented `location` module.
- The current engineering review no longer claims the frontend is outside scope.
- The canonical frontend README and development guide no longer contain a user-specific absolute path or pre-Gate 10/11 status.
- Project vision documents are marked as historical/aspirational and link to current architecture.
- PostgreSQL setup no longer contains stale checkpoint wording.

Historical Gate reports were left intact as historical records.

## 14. Disposable PostgreSQL

A unique `postgres:16-alpine` container named `nexus-gate12-postgres-20260915` was started on `127.0.0.1:55432` with test-only credentials and no persistent volume. PostgreSQL reported version `16.15`. No personal/external database was used, and existing user containers were not modified. The disposable container was removed after validation.

## 15. Migration 000004

Migrations `000001` through `000004_expand_msisdn_length` were applied successfully to the disposable database.

## 16. Schema version

The migrated database reported schema version `4`.

## 17. Sixteen-character MSISDN

The PostgreSQL integration suite persisted and read the maximum supported `+` plus 15-digit MSISDN representation and validated its uniqueness behavior.

## 18. PostgreSQL integration tests

With `TEST_DATABASE_URL` pointing only to the disposable PostgreSQL instance, `go test -count=1 ./...` passed. This exercised subscriber, device, and session PostgreSQL repositories, including the MSISDN boundary and relevant concurrency/session behavior.

## 19. Hero Flow

The README and API documentation describe the implemented flow:

```text
Provision Subscriber → Activate → Register Device → Attach → Handover → Detach
```

It is available through the operational UI or through the HTTP-only CLI simulator.

## 20. Simulator

Against the disposable PostgreSQL-backed API, the simulator completed a two-device run:

```text
completed 2/2; failed 0
attach 2; handover 2; detach 2; dropped 0
```

The database contained the expected two subscribers, two devices, and two sessions. A separate restart check attached an existing device, restarted the API, confirmed IP pool warm-up with one allocated address, then detached the session and confirmed allocation returned to zero. The documentation explicitly states that the browser does not execute `cmd/simulator`; the dashboard Sandbox is frontend-only.

## 21. Go validations

```text
go fmt ./...              PASS
go vet ./...              PASS
go build ./...            PASS
go test -count=1 ./...    PASS (with disposable PostgreSQL)
```

Formatting produced no Go content changes. File hashes matched the index after the command.

## 22. Race detector

Windows did not provide the required native CGO race environment, so the already-established isolated Docker method was used with network disabled and cached modules mounted read-only:

```text
go test -race -count=1 ./...    PASS
data races                       0
```

## 23. Frontend validations

```text
isolated npm ci                         PASS
isolated npm run build                  PASS
workspace canonical npm run build       PASS
```

The final bundle completed with 1,616 transformed modules. Existing browser verification remains engineering evidence because its external runtime is not a declared Quick Start requirement; no E2E framework was added.

## 24. Repository hygiene

The ignore rules and working tree were checked for `node_modules`, `dist`, `.env`, logs, caches, temporary database data, generated binaries, and evidence images. No new generated artifact or temporary infrastructure was retained. No history rewrite was performed. No artificial promotional screenshot was created or versioned.

## 25. Targeted secrets scan

The tracked/current workspace was scanned for credential URLs, sensitive assignments, common token formats, private-key headers, authorization headers, and cookies while excluding dependency/build directories. No private key, external credential, API token, or secret was found. Matches were limited to documented placeholders, test fixtures, and disposable CI/local PostgreSQL credentials.

## 26. AI-assisted engineering disclosure

The root README contains a short disclosure that AI-assisted workflows were used while architecture, scope, domain decisions, acceptance, and testing remained subject to explicit human review.

## 27. Final readiness matrix

| Area | Status | Evidence |
|---|---|---|
| README | READY | Canonical root entry point |
| Quick Start | READY | MEMORY and PostgreSQL paths documented |
| Architecture | READY | Current architecture document and Mermaid diagrams |
| API Documentation | READY | Central endpoint and error reference |
| Hero Flow | READY | UI and HTTP-only simulator paths explained |
| Testing | READY | Go, race, frontend, and PostgreSQL checks pass |
| PostgreSQL | READY | PostgreSQL 16.15, migrations 001–004, schema v4 |
| Frontend | READY | Canonical app independently installs and builds |
| Security/Hygiene | READY | Targeted scan and artifact review completed |
| CI | READY | Minimal Linux workflow with disposable PostgreSQL |
| Reproducibility | READY | MEMORY quick start plus isolated npm/DB validation |
| Technical Debt | MINOR_WORK | Clearly isolated legacy frontend remains |
| Portfolio Narrative | READY | Scope and limitations are factual and concise |

No BLOCKER or IMPORTANT_WORK item remains for publication.

## 28. P2/FUTURE items

P2:

- Remove the clearly identified legacy frontend after explicit authorization for that deletion.
- Observe the first hosted GitHub Actions run after a future approved push.
- Select a reviewed promotional dashboard screenshot later if desired.

Future scope remains intentionally outside this release: authentication, durable event history, cloud deployment, realtime transports, production observability, and additional telecom protocol modeling.

## 29. Git diff stat

Before this report was added, the tracked-file diff was:

```text
8 files changed, 117 insertions(+), 55 deletions(-)
```

New untracked release-readiness files are listed in the final `git status --short`; Git does not include untracked files in `git diff --stat` until staging. Nothing was staged for commit.

## 30. Git status

Expected final changes consist only of the documentation, canonical frontend manifest/lockfile, license, and CI files listed in section 2. The exact final status is reported with the Human Review handoff. No Go implementation file is modified.

## 31. Recommendation

**PORTFOLIO READY: YES**

The repository is discoverable, reproducible, documented, validated against real disposable PostgreSQL, and CI-verifiable. Remaining items are optional P2/FUTURE work and do not block a public portfolio release.

Gate 13 was not started.
