# NEXUS Core Lab — Gate 11 Domain Invariants

**Date:** 2026-09-15
**Branch:** `feat/demo-ui`
**Baseline:** `78f1fbd8335f7179c8af72beddf0d707d4f3a295`
**Status:** implementation complete; waiting for Human Review
**Source control:** no commit and no push

## 1. Baseline and audited paths

The branch, baseline commit, and clean working tree were confirmed before the
Gate started. The audit covered `internal/subscriber`, `internal/device`,
`internal/session`, `internal/network`, the MEMORY and PostgreSQL repositories,
`cmd/api/main.go`, migrations, related tests, and the Sessions frontend error
flow.

The current dependency path is:

```text
Subscriber Service
        ↓ adapter in cmd/api
Device Service
        ↓ adapter in cmd/api
Session Service → Session Repository / IPPool / Telemetry emitter
```

Session continues to consume a small `DeviceChecker` interface and does not
import Subscriber models.

## 2. Previous SUSPENDED/Attach behavior

Device registration already required an ACTIVE Subscriber, but the Session
adapter later checked only whether the Device still existed and was REGISTERED.
Consequently, a Device registered while its Subscriber was ACTIVE could start a
new Attach after that Subscriber became SUSPENDED or DEACTIVATED.

## 3. New Attach invariant

Every new Attach now rechecks the owning Subscriber through the existing
composition adapter. Only `ACTIVE` is accepted. `PENDING_ACTIVATION`,
`SUSPENDED`, and `DEACTIVATED` are rejected with the Session domain error
`ErrSubscriberNotActive`.

HTTP maps that error to:

```text
422 Unprocessable Entity
code: SUBSCRIBER_NOT_ACTIVE
message: subscriber must be ACTIVE to start a new session
```

Device-not-found and Device-not-eligible errors remain distinct.

## 4. Existing connected Session and re-attach ordering

Suspension does not mutate an already CONNECTED Session. There is no automatic
Detach, no cascade from Subscriber to Session, no IP release, and no event.

The Device/Subscriber eligibility check now runs inside the per-Device lock and
before IP allocation, Session construction, persistence, stale replacement, or
event emission. A rejected re-attach therefore preserves the original Session
ID, IP address, and CONNECTED state. It emits neither `ATTACH` nor
`STALE_DISCONNECT`, creates no Session, and leaves the IPPool allocation count
unchanged.

Gate 11 changes only new Attach eligibility. Handover and Detach of the existing
connected Session remain allowed after suspension.

## 5. ACTIVE regression and lifecycle coverage

Tests confirm:

- ACTIVE Subscriber + REGISTERED Device can Attach and re-attach as before.
- PENDING, SUSPENDED, and DEACTIVATED reject a new Attach.
- A connected Session remains connected when its Subscriber is suspended.
- A rejected re-attach preserves the original Session and IP.
- Handover and Detach still work for that existing Session.
- Rejected requests do not change ATTACH, STALE_DISCONNECT, or IPPool state.

PENDING cannot normally own a Device through the public registration flow,
because registration itself requires ACTIVE. Its Session contract is still
covered with a valid pre-existing Device fixture to protect the invariant from
persisted or reconstructed states.

## 6. LTE/5G decision

No radio-technology compatibility enforcement was added. Device and Cell
technology remain semantic educational metadata. Tests explicitly preserve:

- 5G Device → `CELL-SP-001` LTE;
- LTE Device → `CELL-SP-002` 5G.

No `TECHNOLOGY_MISMATCH` error exists. The concise architectural decision is in
`docs/adr/ADR-002-radio-technology-as-semantic-metadata.md`. The ADR states that
the model does not represent RAT compatibility, NAS, RRC, inter-RAT mobility,
EPC, 5GC, or GTP, and that no real LTE/5G behavior may be inferred.

## 7. MSISDN contract and persistence consistency

The audited application rule remains `^\+?[1-9]\d{9,14}$`: 10–15 digits with
an optional leading `+`. Therefore `+123456789012345` is valid and occupies 16
characters. Migration `000001` used `VARCHAR(15)`, which could not persist the
complete domain contract.

The domain validation was preserved. Tests cover MSISDN with and without `+`,
minimum and maximum valid sizes, values above the digit limit, round-trip in the
MEMORY repository, and global MSISDN uniqueness.

## 8. Migration and schema version

Incremental migration `000004_expand_msisdn_length` changes only
`subscribers.msisdn`:

```text
UP:   VARCHAR(15) → VARCHAR(16)
DOWN: VARCHAR(16) → VARCHAR(15) only when no stored value exceeds 15 characters
```

The DOWN migration explicitly raises an exception when incompatible data
exists. It never truncates or normalizes an MSISDN. Historical migrations were
not edited.

The API now requires schema version `4` through the existing startup validation
mechanism. Contract tests confirm version 3 is rejected, version 4 is accepted,
and a newer schema remains accepted. Operationally, PostgreSQL deployments must
run `go run ./cmd/migrate -up` before starting the updated API.

## 9. MEMORY/PostgreSQL parity

MEMORY accepts, reads, and enforces uniqueness for the 16-character maximum
representation. A PostgreSQL integration test was added for persistence,
reading, and uniqueness of the same value after migration 000004.

`TEST_DATABASE_URL` was absent. PostgreSQL integration tests were not executed.
No database was created, no personal database was used, and no migration was
applied to an external environment. SQL was checked by a migration contract
test for the required UP expansion and safe, non-truncating DOWN behavior.

## 10. Frontend impact

No frontend production code changed. `SessionsPage` already renders the HTTP
status, semantic error code, and backend message through `sessionError`, while
requiring a refresh after a failed write. The backend remains the authority for
Subscriber eligibility.

## 11. Hero Flow and domain scenario

The existing automated Hero Flow passed:

```text
Provision → Activate → Register → Attach CELL-SP-001
→ Handover CELL-SP-002 → Detach
```

The Gate 11 composition test also passed:

```text
Provision → Activate → Register → Attach → Suspend
→ attempted re-attach rejected → original Session preserved
```

## 12. Validation results

| Validation | Result |
|---|---|
| `go fmt ./...` | PASS; incidental Windows line-ending rewrites outside scope were restored |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `go test -count=1 ./...` | PASS |
| `go test -race -count=1 ./...` | PASS via existing Docker Desktop and `golang:latest` |
| Data races | **ZERO** |
| Hero Flow named test | PASS |
| Suspend/re-attach named test | PASS |
| `npm run build` in `web/reference-version` | PASS |
| PostgreSQL integration tests | Not executed: `TEST_DATABASE_URL` absent |

The native Windows race command could not start because CGO was disabled and no
local C compiler was available. The successful required run used the already
installed Docker Desktop, `CGO_ENABLED=1`, `GOTOOLCHAIN=local`, `GOPROXY=off`,
network disabled, and read-only mounts for source and the Go module cache.

## 13. Files changed

- `cmd/api/main.go`
- `cmd/api/main_test.go`
- `docs/adr/ADR-002-radio-technology-as-semantic-metadata.md`
- `internal/platform/postgres/postgres.go`
- `internal/platform/postgres/postgres_test.go`
- `internal/session/device.go`
- `internal/session/handler.go`
- `internal/session/handler_test.go`
- `internal/session/service.go`
- `internal/session/session_test.go`
- `internal/subscriber/postgres_repository_test.go`
- `internal/subscriber/subscriber_test.go`
- `migrations/000004_expand_msisdn_length.up.sql`
- `migrations/000004_expand_msisdn_length.down.sql`
- `migrations/migration_contract_test.go`
- `web/reference-version/GATE11_DOMAIN_INVARIANTS.md`

No unrelated production behavior, frontend UI, historical migration, new
dependency, event type, event bus, database, or architecture was introduced.
Gate 12 was not started.

## 14. Final Git state

`git diff --check` passed. The tracked-file statistic is:

```text
 cmd/api/main.go                                 | 20 +++++-
 internal/platform/postgres/postgres.go          | 19 ++++--
 internal/session/device.go                      |  5 +-
 internal/session/handler.go                     |  2 +
 internal/session/handler_test.go                | 21 ++++++
 internal/session/service.go                     |  8 ++-
 internal/session/session_test.go                | 85 +++++++++++++++++++++++++
 internal/subscriber/postgres_repository_test.go | 40 ++++++++++++
 internal/subscriber/subscriber_test.go          | 52 +++++++++++++--
 9 files changed, 233 insertions(+), 19 deletions(-)
```

New files do not appear in the standard unstaged `git diff --stat`; they are
listed in section 13 and in this final `git status --short`:

```text
 M cmd/api/main.go
 M internal/platform/postgres/postgres.go
 M internal/session/device.go
 M internal/session/handler.go
 M internal/session/handler_test.go
 M internal/session/service.go
 M internal/session/session_test.go
 M internal/subscriber/postgres_repository_test.go
 M internal/subscriber/subscriber_test.go
?? cmd/api/main_test.go
?? docs/adr/ADR-002-radio-technology-as-semantic-metadata.md
?? internal/platform/postgres/postgres_test.go
?? migrations/000004_expand_msisdn_length.down.sql
?? migrations/000004_expand_msisdn_length.up.sql
?? migrations/migration_contract_test.go
?? web/reference-version/GATE11_DOMAIN_INVARIANTS.md
```

The working tree is intentionally modified for Human Review. A targeted secrets
scan found no credential-like assignments. No commit and no push were performed.
