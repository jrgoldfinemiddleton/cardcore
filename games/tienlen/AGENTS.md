# AI Agent Guidance: Tiến Lên Engine

## OVERVIEW

The `games/tienlen/` package implements the Tiến Lên climbing (shedding)
card game. In place: combination classification, comparison, and chop
tables; the Killer game state machine (deal, declaration window,
climb/pass/lockout, self-beat, finisher rule, places); the ordered event
log; legal-move enumeration; the Player interface; and the settlement
ledger that folds each hand's events into transfers and balances. Later
steps add the AI players and the Standard variant.

## STRUCTURE

```
games/tienlen/
├── doc.go             # Package documentation
├── sentinels.go       # Sentinel errors, "tienlen: "-prefixed
├── combo.go           # Combo classification, comparison, chop tables
├── tienlen.go         # Game state machine (Config, phases, pile, turn)
├── events.go          # Sealed Event types and the Events() log accessor
├── moves.go           # LegalMoves, CanPass, combination enumeration
├── player.go          # Player interface: ChoosePlay, ChooseDeclareAutoWin
├── ledger.go          # Settlement: fold of events into transfers, Balances/Transfers/Winner
├── helpers_test.go    # Rank/suit aliases and the c() card constructor
├── combo_test.go      # Classification, comparison, and chop-matrix tests
├── tienlen_test.go    # State-machine tests, integration tests, fixtures
├── moves_test.go      # Enumeration and LegalMoves/CanPass tests
├── events_test.go     # Chop-chain bookkeeping and event-order tests
├── ledger_test.go     # Fold unit tests, settlement integration, ledger invariants
└── sentinels_test.go  # Sentinel error coverage
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Change Tiến Lên rules | `combo.go`, `tienlen.go` | Update `doc/games/tienlen/rules.md` first (ADR-006) |
| Change classification | `combo.go` | Classify is shape-pure; legality policy lives in the state machine |
| Change comparison or chops | `combo.go` | Chops are variant-gated; Killer is wired, Standard arrives with its enum value |
| Add a variant | `combo.go`, `tienlen.go` | Variants are fixed and added wholesale (ADR-010). Wire `rulesFor` and the chop table, and check the divergence points that are code-shaped rather than parameterized: self-beat arming in `advanceTurn`, the declaration window in `StartPlay`/`DeclareAutoWin`, and Killer's every-hand lowest-card opening lead in `StartPlay` |
| Change the turn/pile model | `tienlen.go` | Self-beat, lockout, finisher rule, and the resolution pause live here |
| Change settlement inputs | `events.go` | The event log feeds the ledger; keep it append-only and complete |
| Change settlement | `ledger.go` | The fold consumes the event log; the state machine's play paths never compute payments; the zero-sum invariant is test-locked |
| Change legal-move queries | `moves.go` | Enumeration must stay exhaustive and in the documented stable order |
| Change the player contract | `player.go` | `ChoosePlay` returns cards (empty = pass); `ChooseDeclareAutoWin` answers the window |
| Add game lifecycle tests | `tienlen_test.go` | Fixtures build mid-game states directly; keep the invariants asserted |

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
  as a side effect. The same goes for LegalMoves results: fresh slices.
- Drive a game by phase: Deal, optional DeclareAutoWin calls while the
  declaration window is open, StartPlay, then Play and ResolvePile until
  PhaseScore; EndHand starts the next hand or ends the match. The opening
  play of every hand must include the lowest card still in play.
- A pass is Play(seat, nil); CanPass reports whether passing is legal
  (never when leading or self-beating).
- Queries never enforce turn: LegalMoves returns an empty slice (no
  error) for valid seats that cannot act, including non-turn seats.
  ErrOutOfTurn belongs to Play only.
- Events are the settlement record: append-only inside the engine,
  exposed only as a deep copy via Events(), and rejected actions append
  nothing. The ledger (ledger.go) folds each hand's events into
  transfers exactly once, at the transition into PhaseScore
  (completeHand); the state machine's play paths never compute payments.
- Balances and the transfer record are match-long state, private like
  the event log and exposed only as copies via Balances() and
  Transfers(); transfers are append-only and never netted. Killer
  settlement: a chop chain settles at its last inter-player chop (a
  self-chop freezes the chain without cancelling it), and chopping a
  finisher's final play is always payment-free.
- State-machine test fixtures use `// Round N:` comments (1-indexed,
  spelled out) at pile boundaries.
- Fixture hands list cards in ascending Tiến Lên order — rank 3 through
  A, then 2, with ♠♣♦♥ within a rank — grouping same-rank cards on one
  line when they fit.
- Test names carry the Killer prefix when the mechanic under test is
  Killer-specific (self-beat, absolute lockout, the chop table, the
  automatic-win set and priority); tests of shared mechanics stay
  unprefixed.

## ANTI-PATTERNS

- Never use raw rank or suit ordering for game logic; use the ordering
  keys in `combo.go`.
- Never add a default case to a Variant switch.
- Never put legality policy (what may be played, and when) into Classify;
  it classifies shapes only.
- Never detect instant wins in the combination layer; that is
  state-machine business.
- Never let the turn land on a finished, declared, or locked-out seat;
  if no responder remains, either the holder self-beats or the pile
  pauses for ResolvePile.
- Never mutate the event log or expose its internals; clients get a deep
  copy.

## COMMANDS

```bash
# Run the Tiến Lên tests
go test ./games/tienlen/

# Run the full project gate
make check
```
