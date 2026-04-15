# Design Principles and Philosophy

## Overview
cardcore is a minimal, composable card game engine written in Go. It is designed as a library, not a standalone application. The engine provides the core logic and state management for card games, intended to be consumed by future HTTP or WebSocket APIs that will facilitate client-server interaction.

## Suckless Philosophy
We adhere to the [suckless philosophy](https://suckless.org/philosophy/): maintaining a small, readable codebase with zero external runtime dependencies. We avoid premature abstraction; generics and shared interfaces for entities like `Player` or `GameState` are deferred until they become necessary. Following the "rule of two games," we will only extract common abstractions after building at least two distinct games.

## Core Primitives
The engine is built on fundamental "atoms": `Suit`, `Rank`, `Card`, `Deck`, and `Hand`. These primitives represent the basic physical components of most card games. By keeping these simple and focused, the engine remains flexible and easy to extend for various game types.

## Game Subpackages
Game-specific logic is isolated in subpackages under `games/`. For example, a hypothetical `games/hearts` package would contain the complete rules, state machine, and scoring logic for Hearts. This modularity ensures that the core engine remains clean and that new games can be added without modifying existing game logic.

## Flexibility Notes
The engine's core primitives impose minimal constraints, allowing game subpackages to adapt them freely. For example, `Deck` is represented as a `[]Card` slice, so games can construct custom decks beyond the standard 52 cards. Each game defines its own constraints, such as player counts and rule variants.

## Testing
Reliability is enforced through comprehensive testing. All packages include unit tests, and the `make check` command serves as the mandatory gatekeeper. No changes are merged into the codebase without passing the full test suite.

## Error Handling
Functions return errors for conditions the caller cannot prevent or that depend on runtime input (invalid cards, wrong phase transitions, malformed requests). Precondition violations — conditions the caller is responsible for checking before the call — trigger panics. A panic signals a bug in the calling code, not a recoverable situation. This distinction keeps error returns meaningful: every `error` a caller handles represents a genuine failure mode, not a misuse of the API.
