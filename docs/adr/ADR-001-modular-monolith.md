# ADR-001 — Modular Monolith as Initial Architecture

**Status:** Accepted  
**Date:** 2026-09-08

## Context

NEXUS Core Lab will contain multiple business domains related to the
simulation of telecommunications systems, including:

- subscribers
- devices
- sessions
- network
- location
- telemetry

Although these domains may eventually become independent services,
the project is still in its initial stage.

Starting immediately with microservices would introduce operational
complexity before the boundaries between domains are sufficiently
understood.

This complexity would include:

- service discovery
- distributed transactions
- network communication
- multiple deployments
- distributed tracing
- message consistency
- failure handling between services
- additional infrastructure

At this stage, these costs would not provide enough technical benefit.

## Decision

NEXUS Core Lab will initially use a **Modular Monolith** architecture.

The application will run as a single deployable backend while maintaining
clear boundaries between its internal domains.

Initial modules:

- Subscriber
- Device
- Session
- Network
- Location
- Telemetry

Each module should own its business logic and expose only the interfaces
required for communication with other parts of the application.

The architecture should avoid unnecessary coupling between modules.

## Initial Structure

```text
cmd/
└── api/
    └── main.go

internal/
├── subscriber/
├── device/
├── session/
├── network/
├── location/
└── telemetry/
```