# NEXUS Core Lab — Project Vision

## 1. Vision

NEXUS Core Lab is an educational backend platform designed to simulate
concepts found in 4G/5G mobile networks.

The project aims to explore software engineering applied to
telecommunications, distributed systems, concurrent processing,
service communication, observability, and high-performance systems.

The system does NOT intend to implement a commercial 4G/5G Core Network
or replace real components used by telecommunications operators.

Instead, it works as an engineering laboratory where devices,
subscribers, sessions, cells, events, and network services are
represented by simulated entities.

---

## 2. Technical Objectives

The project is designed to demonstrate practical knowledge in:

- Go
- Backend development
- REST APIs
- gRPC
- PostgreSQL
- Redis
- Asynchronous messaging
- Docker
- Modular architecture
- Concurrency in Go
- Distributed systems
- Observability
- Metrics
- Structured logging
- Automated testing
- Integration testing
- CI/CD
- Load simulation
- 4G/5G telecommunications concepts

At a later stage, a small module written in C will be developed for
experiments involving message and protocol processing.

---

## 3. Domain

The initial domain will be divided into five main areas.

### Subscriber

Represents a fictional network subscriber.

Example information:

- Internal ID
- Simulated IMSI
- Simulated MSISDN
- Status
- Creation timestamp

No real telecommunications identifiers will be used.

### Device

Represents a virtual device associated with a subscriber.

Example information:

- Device ID
- Subscriber ID
- Device type
- Supported network technology
- Status

Initially supported simulated technologies:

- LTE
- 5G

### Session

Represents the logical connection of a device to the simulated network.

Initial states:

- DISCONNECTED
- ATTACHING
- CONNECTED
- DETACHING

A session may contain:

- Device
- Subscriber
- Network technology
- Cell
- Session start time
- Last event
- Session termination time

### Location

Represents the logical location of a device within the simulated network.

Initially, location will be based on simulated network cells.

Example:

`CELL-CPS-001`

`Campinas / SP`

Each movement may generate a location event.

### Network

Represents simulated infrastructure elements.

Initially:

- Cells
- Network nodes
- Network status

Concepts inspired by 4G/5G network components may be introduced later
as the laboratory evolves.

---

## 4. Basic Flow

An initial connection flow may work as follows:

```text
Device Simulator
        |
        v
Subscriber Validation
        |
        v
Session Creation
        |
        v
Network Assignment
        |
        v
Location Registration
        |
        v
Event Publication
        |
        v
Telemetry / Persistence