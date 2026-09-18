# MAPLIBRE PRODUCTION RENDERING FIX REPORT

- **Project:** NEXUS Core Lab
- **Gate:** 13C.4A — MapLibre production rendering correction + controlled Network Map Experience Upgrade
- **Date:** 18 September 2026
- **Branch:** feat/demo-ui
**Baseline:** 38834a52cf1b5717f015bfb0333ada2603864ec8

## Executive result

The blank production map was reproduced, isolated and corrected without changing the backend, API contracts, domain model, deployment configuration or map provider. The final production-equivalent Docker image renders real Campinas cartography from OpenFreeMap, loads a self-contained MapLibre worker with HTTP 200, and preserves the local fallback when the provider is unavailable.

After the rendering fix was proved, the Network map was upgraded into a controlled NOC-style topology view. All telecom positions and coverage remain explicitly simulated or illustrative. No real antenna, RF, GPS or geolocation data was introduced.

No staging, commit, push or external deployment was performed.

## 1. Symptom reproduced

The public production page loaded the Network frame, controls, legend and MapLibre canvas, but the cartography remained blank. The container dimensions were valid and WebGL was available. This excluded a zero-size container, generic CSS clipping and absence of WebGL as the primary cause.

The original browser test reported success because it considered style.load sufficient and did not verify that a worker and vector tiles had actually loaded.

## 2. Root cause

Gate 13C changed MapTopology to a lazy-loaded Vite chunk. MapLibre GL JS 6.10.0 derives its default worker URL relative to its own module with import.meta.url. In this lazy production bundle, Vite emitted the MapTopology chunk but did not emit the expected sibling worker asset.

The diagnosis was proved in two steps:

1. The original production-equivalent bundle requested /assets/maplibre-gl-worker.mjs and received HTTP 404.
2. A first diagnostic fix using plain ?url emitted the small worker entry, but that entry requested /assets/maplibre-gl-shared.mjs, which also received HTTP 404. The worker existed, but its dependency graph was incomplete and the canvas remained transparent.

The final cause was therefore an incomplete MapLibre worker dependency graph in the Vite production output, triggered by the lazy chunk boundary.

## 3. Surgical correction

MapTopology.tsx now imports the MapLibre worker through Vite worker bundling:

~~~ts
import maplibreWorkerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
maplibregl.setWorkerUrl(maplibreWorkerUrl);
~~~

The ?worker&url query makes Vite bundle the complete worker dependency graph into one self-contained production asset. The generated file is:

- maplibre-gl-worker-BBYVuMpd.js;
- 509.47 KB uncompressed;
- HTTP 200 in the production-equivalent runtime;
- no request for maplibre-gl-shared.mjs.

src/vite-env.d.ts was added so TypeScript understands Vite asset query modules.

## 4. Lazy loading and lifecycle audit

Lazy loading remains active. The MapLibre library and map-specific CSS are still loaded only when the topology component is requested. The main application bundle remains approximately 223 KB uncompressed, while the map chunk remains separate at approximately 1.055 MB uncompressed / 288 KB gzip.

The existing lifecycle safeguards were preserved:

- one MapLibre instance per initialization attempt;
- explicit abort controller for the style request;
- ResizeObserver disconnected during cleanup;
- map instance removed during cleanup;
- retry creates a new instance intentionally;
- no global map instance;
- no additional polling loop;
- no geolocation or GPS request.

## 5. Production-equivalent proof

The final clean Docker build used the repository Dockerfile with --no-cache and completed successfully. It rebuilt the Go backend, frontend through npm ci and Vite, and the distroless runtime image.

| Resource | Result |
|---|---:|
| MapLibre bundled worker | HTTP 200 |
| OpenFreeMap style | HTTP 200 |
| OpenFreeMap sprites | HTTP 200 |
| OpenFreeMap vector tiles | HTTP 200 |
| MapLibre worker/shared failures | 0 |
| Uncaught JavaScript exceptions | 0 |
| WebGL context lost | No |
| Fallback when provider is blocked | PASS |

## Network Map Experience Upgrade

### Cell visualization

The three configured cells now use an operational node language instead of generic map pins:

- concentric core, ring and halo;
- visible LTE/5G technology glyph inside the node;
- explicit Cell ID, region, technology and current logical-link count;
- active/idle intensity derived from the current session snapshot;
- cell popup and persistent inspector with configuration state and active-session count.

### Illustrative coverage

Coverage uses low-opacity technology-colored circles with restrained outlines. The interface labels positions and coverage as simulated. These circles do not represent RF reach, RSRP, RSRQ, SINR, bands, azimuth, frequency or physical antenna coverage.

### Connected Device visualization

Connected devices use a distinct green core, ring and halo, plus a compact Device/technology label. Their position is deterministic from Session and Device identifiers and anchored around the configured Cell. The larger illustrative offset improves readability without introducing random movement.

The inspector and popup expose existing domain data only: Device alias, IMEI, virtual IP, serving Cell, technology and Session ID.

### Session Link

Each connected Session produces a two-layer logical association line between Device and serving Cell. The line is explicitly a logical topology link, not a route, RF signal, packet animation or physical trajectory.

### Handover feedback

When the same Session changes Cell, the UI detects the observed association change and temporarily shows:

- orange dashed line from previous Cell to current Cell;
- HANDOVER OBSERVED banner;
- Device alias;
- previous and target Cell IDs;
- statement that the logical association changed while Session and IP were preserved.

The feedback is derived from successive authoritative Session snapshots and does not invent a new backend event or endpoint.

### HUD, legend and hierarchy

The map includes a compact Network Digital Twin HUD with configured-cell count, active-cell count, logical-link count, current topology read state and a permanent simulated-position/coverage disclosure.

The legend distinguishes LTE and 5G with both glyph and color, identifies connected devices, and includes handover line semantics.

### Empty state

When zero connected sessions are observed, configured cells remain visible and a deliberate TOPOLOGY READY state explains that there are no dynamic connections. The page no longer looks unfinished or broken.

### Fallback

Provider or WebGL failure still selects the local, editable fallback. It preserves all three configured cells, local Campinas context, topology status, truthfulness wording, retry control and attribution.

### Performance

The map remains route-level lazy loaded. The complete worker is a separate asset, fetched only for the map. No new runtime dependency or polling process was introduced. The existing Vite chunk-size warning remains informational for the map-only chunk; the initial application bundle is still separated.

### Responsive validation

The following states were captured at 1920×1080, 1440×900 and 1366×768:

- empty;
- attached to CELL-SP-001;
- handover to CELL-SP-002;
- detached;
- provider fallback.

For every state and viewport, the Network workspace reported scrollWidth == clientWidth and scrollHeight == clientHeight. Evidence is stored locally under evidence/map-experience/. PNG files remain ignored by repository policy, while results.json is versionable evidence.

## Truthfulness Audit

| Element | Source | Classification |
|---|---|---|
| Campinas streets, districts and road labels | OpenFreeMap / OpenMapTiles / OpenStreetMap data | REAL CARTOGRAPHY |
| API read status | Existing frontend transport and API responses | REAL DOMAIN STATE |
| Subscriber, Device, Session and IP values | Existing NEXUS API | REAL DOMAIN STATE |
| Attach, Handover and Detach state changes | Existing Session API | REAL DOMAIN STATE |
| Three configured Cell identities and technologies | Existing frontend mirror of backend Cell catalogue | REAL DOMAIN STATE / CONFIGURATION |
| Cell anchor coordinates | Local presentation catalogue | SIMULATED |
| Device coordinates | Deterministic projection from existing Session/Device IDs | SIMULATED |
| Coverage circles and halos | Map presentation layer | ILLUSTRATIVE |
| Device-to-Cell line | Current Session association | REAL ASSOCIATION rendered ILLUSTRATIVELY |
| Orange handover path | Successive observed Cell associations | REAL STATE CHANGE rendered ILLUSTRATIVELY |
| Network HUD counts | Current configured cells and topology snapshot | REAL DOMAIN STATE |
| Fallback street image | Local OpenStreetMap-derived static asset | REAL CARTOGRAPHIC CONTEXT |

No nonexistent telecom information was invented. The UI does not claim real Campinas antennas, RF telemetry, user location, GPS, EPC/5G Core production monitoring, physical coverage, radio measurements or live mobile-network operations.

## 6. Tests and validation

### Frontend

| Command / check | Result |
|---|---|
| npm ci | PASS |
| npm run test:map | PASS — 7/7 |
| npm run build | PASS |
| npm audit | PASS — 0 vulnerabilities |
| Gate 13C production browser flow | PASS — 9/9 |
| Network map production/visual flow | PASS — 6/6 checks |
| Responsive state screenshots | PASS — 15/15 |
| Uncaught JavaScript exceptions | 0 |

The new regression test directly asserts Vite worker bundling, explicit setWorkerUrl configuration, serving-cell counts, worker HTTP 200, real vector-tile HTTP 200, zero MapLibre worker/shared failures and fallback operation.

The two console 404 messages recorded by the focused map flow are the existing, expected ACTIVE_SESSION_NOT_FOUND read path when no active Session exists. They are not worker, provider, tile or JavaScript exceptions.

### Go regression

| Command | Result |
|---|---|
| go fmt ./... | PASS — no Go source change |
| go vet ./... | PASS |
| go build ./... | PASS |
| go test -count=1 ./... | PASS |
| go test -race -count=1 ./... | PASS in official Go 1.27 Linux container |
| Data races | 0 |

The native Windows race command could not start because CGO is disabled. The project’s documented isolated method was used: official golang:1.27-bookworm, CGO_ENABLED=1, GOTOOLCHAIN=local, GOPROXY=off, no network, read-only source and module-cache mounts.

## 7. Files in the local change set

Modified:

- web/reference-version/package.json
- web/reference-version/src/App.tsx
- web/reference-version/src/MapTopology.tsx
- web/reference-version/src/comparison.css
- web/reference-version/src/map-presentation.ts
- web/reference-version/src/map-topology.css
- web/reference-version/tests/map-presentation.test.mjs

New:

- web/reference-version/src/vite-env.d.ts
- web/reference-version/tests/map-experience.cjs
- web/reference-version/evidence/map-experience/results.json
- web/reference-version/MAPLIBRE_PRODUCTION_RENDERING_FIX_REPORT.md

Local PNG evidence exists under evidence/map-experience/ and remains ignored.

## 8. Scope confirmation

- backend changed: **no**;
- API endpoint added or changed: **no**;
- domain semantics changed: **no**;
- provider changed: **no** — MapLibre GL JS + OpenFreeMap retained;
- geolocation/GPS added: **no**;
- real antenna/RF data added: **no**;
- dependency added: **no**;
- deployment configuration changed: **no**;
- external deploy performed: **no**;
- git add/commit/push performed: **no**.

## 9. Human Review state

The implementation, tests, local production-equivalent image and evidence are ready for Human Review. No checkpoint or remote action was performed.
