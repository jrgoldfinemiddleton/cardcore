# System Architecture

## Package Structure
The engine uses a two-layer structure: the root package (`cardcore`) provides common primitives like cards and decks, while game-specific logic resides in independent subpackages under `games/`.

```
github.com/jrgoldfinemiddleton/cardcore   ← engine primitives
└── games/
    └── [game]/                           ← game-specific rules + state machine
```

## Game State Machines
Each game package implements its own state machine. As a theoretical example, a trick-taking game like Hearts would advance through phases such as Deal, Pass, Play, Score, and End, with game state updated via method calls that enforce rules and advance turns.

## API-First Vision
The engine is architected to expose an HTTP/WebSocket API in the future. All clients—whether CLI, web, desktop, or mobile—will be thin clients interacting with a central server running the engine. This separation ensures the engine contains zero I/O or presentation logic.

## Data Flow
Clients interact with the system by making requests to the server's API. The server translates these requests into engine method calls, which mutate the game state. After the state is updated, the server reads the new state and responds back to the client.

## Dependency Policy
cardcore maintains zero external runtime dependencies. It relies exclusively on the Go standard library, using `math/rand/v2` for shuffling and `sort` for hand management. This keeps the binary small and ensures reproducible builds. Go 1.24.1+ is required for the `tool` directive, which pins dev tool versions (e.g., golangci-lint) in `go.mod` for reproducible builds across contributors and CI.

## Future Phases
*   **CLI Client**: Direct command-line interaction for local play.
*   **HTTP/WebSocket Server**: Expose the engine as a network-accessible service.
*   **UI Clients**: Browser, desktop, and mobile interfaces for the card game engine.
*   **Second Game (Tiến Lên)**: Build another game to validate the engine's generality.
