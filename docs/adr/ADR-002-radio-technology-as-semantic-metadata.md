# ADR-002 — Radio technology remains semantic metadata

**Status:** Accepted
**Date:** 2026-09-15

## Context

NEXUS Core Lab labels Devices and Cells as `LTE` or `5G` for educational
visualization. The current model does not implement radio access compatibility,
NAS, RRC, inter-RAT mobility, EPC, 5GC, or GTP behavior.

## Decision

Device and Cell technology remain semantic metadata. Attach and Handover do not
enforce matching labels, so the existing teaching flow may attach a 5G Device to
an LTE Cell or an LTE Device to a 5G Cell.

These labels must not be interpreted as evidence of real LTE/5G interoperability
or protocol behavior. Compatibility enforcement may be introduced only if a
future domain model explicitly represents the required radio and core-network
rules.
