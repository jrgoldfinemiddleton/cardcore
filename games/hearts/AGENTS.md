# AI Agent Guidance: Hearts Engine

## OVERVIEW

The `games/hearts` package implements the Hearts card game state machine, rules enforcement, and player interface.

## STRUCTURE

```
games/hearts/
├── doc.go           # Package documentation
├── hearts.go        # Game state machine (Game, Trick, phases, legal play validation)
├── player.go        # Player interface: ChoosePass, ChoosePlay
├── hearts_test.go   # Integration tests for full game lifecycle
└── ai/              # Computer-controlled players (see ai/AGENTS.md)
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Change Hearts rules | `hearts.go` | Update `doc/games/hearts/rules.md` first |
| Change passing logic | `hearts.go` | `SetPass` / pass phase validation |
| Change trick resolution | `hearts.go` | `PlayCard`, `Winner`, `EndRound` |
| Add legal-play constraints | `hearts.go` | Hearts broken, lead suit, point-card first trick |
| Change player contract | `player.go` | `ChoosePass` and `ChoosePlay` signatures |
| Add game lifecycle tests | `hearts_test.go` | Point conservation, hand depletion, phase transitions |
| Add AI behavior | `ai/` | See `ai/AGENTS.md` |

## CONVENTIONS

- Use constants `NumPlayers`, `HandSize`, `PassCount`, `MaxScore`, `MoonPoints` for rule parameters.
- Name predefined cards `queenOfSpades` and `twoOfClubs`.
- Return phase transitions explicitly via `Game` state fields; avoid hidden side effects.
- Integration tests must verify point conservation, hand depletion, and no state leaks between rounds.

## ANTI-PATTERNS

- Never expose raw game internals to AI players beyond the `Player` interface.
- Never allow `PlayCard` to mutate the game without validating the move first.
- Never store per-round state in the `Game` struct after `EndRound` resets it.
- Never put random or heuristic decision logic in this package; that belongs in `ai/`.

## COMMANDS

```bash
# Run the Hearts engine tests
go test ./games/hearts

# Run all Hearts tests including AI
go test ./games/hearts/...

# Run the full project gate
make check
```
