# HTTP API

The API is served by `cmd/api` on port `8080` by default. It uses JSON and does
not currently implement authentication. Paths below are relative to the API
origin.

Errors use a stable envelope:

```json
{
  "error": "conflict",
  "message": "human-readable explanation",
  "code": "MACHINE_READABLE_CODE"
}
```

## Operations and telemetry

| Method | Path | Result |
|---|---|---|
| `GET` | `/health` | Liveness: `status` and service name |
| `GET` | `/telemetry` | Request, session, event, and dropped-event counters |
| `GET` | `/api/v1/system/storage` | `MEMORY` or `POSTGRESQL` diagnostics; PostgreSQL availability uses the application's existing pool |
| `GET` | `/api/v1/network/ip-pool` | CIDR, capacity, allocated, available, and utilization percentage |
| `GET` | `/api/v1/events/recent` | Up to 100 events from the current process, newest consumed first |

`/health` is liveness only. Storage availability is intentionally reported by
the separate storage endpoint. Recent events are not persisted.

## Subscribers

| Method | Path | Request / behavior | Success |
|---|---|---|---|
| `POST` | `/api/v1/subscribers` | `{"imsi":"724...","msisdn":"+55..."}` | `201`, provisioned Subscriber |
| `GET` | `/api/v1/subscribers` | Optional `status` filter; optional `imsi` exact lookup | `200`, collection or Subscriber |
| `GET` | `/api/v1/subscribers/{id}` | Lookup by ID | `200`, Subscriber |
| `POST` | `/api/v1/subscribers/{id}/activate` | No body required | `200`, updated Subscriber |
| `POST` | `/api/v1/subscribers/{id}/suspend` | Optional `{"reason":"..."}` | `200`, updated Subscriber |
| `POST` | `/api/v1/subscribers/{id}/deactivate` | Optional `{"reason":"..."}` | `200`, updated Subscriber |

Lifecycle: `PROVISIONED → ACTIVE → SUSPENDED/DEACTIVATED`, subject to the
domain transition rules. A suspended Subscriber cannot start a new attach; an
existing connected session is not automatically detached.

Main failures include `400 INVALID_TELECOM_IDENTITY`, `404
SUBSCRIBER_NOT_FOUND`, `409 SUBSCRIBER_ALREADY_EXISTS`, and `409
INVALID_STATE_TRANSITION`.

## Devices

| Method | Path | Request / behavior | Success |
|---|---|---|---|
| `POST` | `/api/v1/devices` | `{"subscriber_id":"...","imei":"...","technology":"LTE"}` | `201`, registered Device |
| `GET` | `/api/v1/devices` | Global collection; optional `subscriber_id` filter | `200`, collection |
| `GET` | `/api/v1/devices/{id}` | Lookup by ID | `200`, Device |

`technology` accepts the modeled LTE/5G metadata. Registration requires an
ACTIVE Subscriber. Main failures include `400 INVALID_DEVICE_DATA`, `404
SUBSCRIBER_NOT_FOUND`, `409 DEVICE_ALREADY_EXISTS`, and `422
SUBSCRIBER_NOT_ACTIVE`.

## Sessions

| Method | Path | Request / behavior | Success |
|---|---|---|---|
| `POST` | `/api/v1/sessions/attach` | `{"device_id":"...","cell_id":"CELL-SP-001"}` | `201`, connected Session with allocated IP |
| `POST` | `/api/v1/sessions/{id}/handover` | `{"target_cell_id":"CELL-SP-002"}` | `200`, updated Session |
| `POST` | `/api/v1/sessions/{id}/detach` | No body required | `200`, disconnected Session |
| `GET` | `/api/v1/sessions/{id}` | Lookup by Session ID | `200`, Session |
| `GET` | `/api/v1/sessions?device_id={id}` | Active Session for one Device | `200`, Session |

Important failures include `400 CELL_NOT_FOUND`, `404 DEVICE_NOT_FOUND`, `404
SESSION_NOT_FOUND`, `404 ACTIVE_SESSION_NOT_FOUND`, `422
DEVICE_NOT_ELIGIBLE`, `422 SUBSCRIBER_NOT_ACTIVE`, and `422
SESSION_NOT_CONNECTED`.

## Minimal Hero Flow

The dashboard exposes the same domain actions. For an automated concurrent run,
start the API and execute:

```sh
go run ./cmd/simulator -api http://localhost:8080 -devices 5
```

The simulator provisions and activates unique Subscribers, registers Devices,
attaches them to `CELL-SP-001`, hands them over to `CELL-SP-002`, detaches them,
and reads `/telemetry`. It does not access repositories or PostgreSQL directly.
