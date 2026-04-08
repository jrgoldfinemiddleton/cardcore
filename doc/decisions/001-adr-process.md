# ADR-001: Use Architecture Decision Records

**Date:** 2026-04-03
**Status:** Accepted

## Context
We want a lightweight, durable way to record significant architectural choices. ADRs are text files that live with the code, are immutable once committed, and are easy for contributors (human and AI) to read.

## Decision
We will use ADRs stored in `doc/decisions/` as Markdown files. Each ADR is numbered sequentially. Each ADR carries a Status field with one of: **Proposed**, **Accepted**, **Deprecated**, **Superseded**, or **Rejected**. ADRs are never edited after their initial commit; a status change is recorded by writing a new ADR that references the original.

## Consequences
(+) Decisions are traceable and self-documenting. (+) AI co-developers can read ADRs for context. (-) Requires discipline to write ADRs at decision time, not retroactively.
