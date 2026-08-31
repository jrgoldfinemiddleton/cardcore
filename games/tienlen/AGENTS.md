# AI Agent Guidance: Tiến Lên Engine

## OVERVIEW

The `games/tienlen/` package implements the Tiến Lên climbing (shedding)
card game. In place: combination classification, comparison, and the Killer
chop table. Later steps add the game state machine (tienlen.go), the Player
interface (player.go), and the settlement ledger (ledger.go).

## STRUCTURE

```
games/tienlen/
├── doc.go           # Package documentation
├── sentinels.go     # Sentinel errors, "tienlen: "-prefixed
├── combo.go         # Combo classification, comparison, chop tables
├── helpers_test.go  # Rank/suit aliases and the c() card constructor
└── combo_test.go    # Classification, comparison, and chop-matrix tests
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Change Tiến Lên rules | `combo.go` | Update `doc/games/tienlen/rules.md` first (ADR-006) |
| Change classification | `combo.go` | Classify is shape-pure; legality policy lives in the state machine |
| Change comparison or chops | `combo.go` | Chops are variant-gated; Killer is wired, Standard arrives with its enum value |
| Add a variant | `combo.go` | Variants are fixed and added wholesale (ADR-010) |

## CONVENTIONS

- The rules doc is the spec: change `doc/games/tienlen/rules.md` before
  changing code (ADR-006).
- Game logic never compares raw cardcore.Rank or cardcore.Suit numeric
  values. Tiến Lên ordering (ranks 3 low through 2 high; suits
  ♠ < ♣ < ♦ < ♥) matches neither engine order, so every comparison goes
  through rankKey, suitKey, and compareCards — never Hand.Sort.
- Terminology map: the neutral code names Quad and PairSequence are
  Killer's "Killer" and "Bomb" and Standard's "tứ quý" and "đôi thông".
  Per-ruleset display names are display-only and have no code consumer.
- Variants are fixed (Killer first, then Standard) and added wholesale
  (ADR-010). Switches over Variant omit the default case so a new variant
  fails the build until it is wired.
- Classify copies its input before sorting; hands must never be reordered
  as a side effect.
- State-machine test fixtures use `// Round N:` comments (1-indexed,
  spelled out) once tienlen.go lands.
- The ledger (a later step) tallies chop payments and penalties as
  transfers that net to zero.

## ANTI-PATTERNS

- Never use raw rank or suit ordering for game logic; use the ordering
  keys in `combo.go`.
- Never add a default case to a Variant switch.
- Never put legality policy (what may be played, and when) into Classify;
  it classifies shapes only.
- Never detect instant wins in the combination layer; that is
  state-machine business.

## COMMANDS

```bash
# Run the Tiến Lên tests
go test ./games/tienlen/

# Run the full project gate
make check
```
