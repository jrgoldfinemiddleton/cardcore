# AI Agent Guidance: Architecture Decisions

## Overview

This directory contains Architecture Decision Records (ADRs) — durable design policies that govern the cardcore project. ADRs state laws, not status reports.

## How to Use

1. **Before making any architectural change**, read the relevant ADR(s)
2. **If no ADR covers your situation**, propose one before implementing
3. **If an ADR is wrong**, write a new ADR that supersedes it — never edit substantive content of an existing ADR after initial commit

## Conventions

- Sequential numbering (`001-`, `002-`, ...)
- `Status` field is the only mutable part after initial commit
- Status values: `Proposed`, `Accepted`, `Superseded`, `Deprecated`
- Self-contained: a reader should never need to consult a superseded ADR

## Key ADRs

| ADR | Topic | Critical Rule |
|-----|-------|---------------|
| 001 | ADR process | How we write and maintain ADRs |
| 002 | Documentation structure | Three-tier docs: README, design docs, ADRs |
| 003 | Language choice | Go — stdlib-first, no external deps |
| 004 | API-first architecture | Design the API before implementation |
| 005 | No premature abstractions | Extract generics only when second game demands it |
| 006 | Rules-Driven Development | Write rules doc before game implementation |
| 007 | Automated convention enforcement | `convention_test.go` enforces doc comments, function ordering |
| 008 | AI design principles | Superseded by ADR-009 |
| 009 | AI difficulty and personality | AI is read-only, stdlib-only, one type per difficulty |

## Anti-Patterns

- **Never cite AGENTS.md in an ADR** — ADRs are the authority, not derived from guidance
- **Never write "X is deferred"** — Frame as "X is introduced when Y demands it"
- **Never put mutable lists (approved deps, etc.) in ADRs** — Use living documents referenced by ADRs
