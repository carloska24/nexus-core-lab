# NEXUS Core Lab — AI Engineering Guidelines

## 1. Purpose

This document defines mandatory engineering rules for AI-assisted development
within NEXUS Core Lab.

AI agents are implementation assistants, not autonomous product designers
or software architects.

They must respect the existing architecture, domain model, design system,
coding conventions, and documented decisions before introducing changes.

The objective is to prevent AI-generated patterns that create unnecessary
complexity, inconsistent interfaces, generic visual design, speculative
architecture, or code that cannot be clearly justified.

---

## 2. Core Principle

> Do not invent complexity.

Every architectural decision, abstraction, dependency, component, visual
pattern, or infrastructure element must solve a concrete problem.

When the simplest implementation satisfies the current requirement,
prefer it.

Do not implement future requirements unless explicitly requested.

---

## 3. Before Writing Code

Before implementing any task, the AI agent must:

1. inspect the existing project structure;
2. read the relevant project documentation;
3. inspect applicable ADRs;
4. identify existing patterns and conventions;
5. identify the module responsible for the requested behavior;
6. reuse existing components when appropriate;
7. understand the scope of the requested change.

The agent must not assume that a missing feature should be implemented
using a new framework, abstraction, service, or dependency.

---

## 4. Scope Discipline

Implement only what is required for the current task.

Do not:

- create speculative features;
- implement future roadmap items;
- refactor unrelated modules;
- redesign unrelated interfaces;
- introduce infrastructure that is not currently required;
- create placeholder functionality that appears complete but is not;
- silently change architectural decisions.

If a task exposes a larger architectural problem, document or report the
problem before expanding the implementation scope.

---

## 5. Architecture

NEXUS Core Lab initially follows a Modular Monolith architecture.

Respect module boundaries.

Do not introduce microservices unless an approved Architecture Decision
Record explicitly authorizes the extraction.

Do not create distributed-system complexity merely to make the project
appear more sophisticated.

Architecture must evolve from measurable requirements.

---

## 6. Abstractions

Avoid premature abstraction.

Do not automatically create patterns such as:

- BaseRepository
- BaseService
- GenericRepository
- GenericService
- AbstractHandler
- Manager
- Helper
- Utils
- Common
- Factory

unless a concrete and documented problem requires them.

An abstraction should normally exist because multiple real implementations
or behaviors justify it.

Do not create interfaces only to mirror every concrete type.

In Go, interfaces should generally be introduced where they are consumed
and when they provide actual value.

Prefer explicit, readable code over artificial enterprise complexity.

---

## 7. Dependencies

Every new external dependency must have a technical reason.

Before introducing a dependency, consider whether the Go standard library
or an existing project dependency already solves the problem adequately.

Do not add libraries solely because they are popular or commonly generated
by templates.

Avoid dependency duplication.

---

## 8. Naming

Names must communicate domain meaning.

Prefer:

- subscriber
- device
- session
- cell
- location
- network
- event

Avoid vague names such as:

- data
- object
- item
- thing
- helper
- manager
- misc
- common

Functions should describe behavior.

Types should describe concepts.

Packages should represent clear responsibilities.

---

## 9. Comments

Do not generate comments that merely repeat the code.

Avoid:

```go
// CreateSubscriber creates a subscriber.
func CreateSubscriber() {}
```

AI agents must not create, modify, delete, rename, or reorganize governance
documents, ADRs, project vision files, or engineering guidelines unless
explicitly instructed to do so.
