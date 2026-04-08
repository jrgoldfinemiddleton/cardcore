# ADR-004: API-First Architecture

**Date:** 2026-04-03
**Status:** Accepted

## Context
Card game clients come in many forms: CLI, web browser, mobile app, desktop GUI. A previous design (rejected) proposed a JavaScript engine with an HTML5 Canvas frontend and a Go server. This creates a language split: engine logic duplicated or split across JS and Go.

## Decision
The Go engine is the single source of truth. It will expose an HTTP/WebSocket API. All clients — CLI, web, desktop, and mobile — are thin clients that call the same server. No game logic lives in clients.

## Consequences
(+) Single engine implementation, no logic duplication. (+) Any client in any language can play any game. (+) Engine is testable independently of all UI. (-) Requires a network layer (future work). (-) Adds latency for local single-player play (acceptable tradeoff).
