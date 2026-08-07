# AI Agent Guidance: Hearts AI Players

## OVERVIEW

The `games/hearts/ai` package provides computer-controlled Hearts players: Random, Heuristic, and PIMC.

## STRUCTURE

Production files:
- `doc.go` — Package documentation
- `random.go` — Random legal move player
- `heuristic.go` — Rule-based heuristic player
- `analysis.go` — Per-decision game-state analysis
- `pimc.go` — PIMC orchestration
- `pimc_sample.go` — Possible hand sampling
- `pimc_rollout.go` — Random rollout simulation
- `pimc_aggregate.go` — Rollout score aggregation

Test/bench files:
- `helpers_test.go` — Shared test helpers (`c`, `pointTrick`, `playRoundWithPlayer`, `setupShootActiveEarlyGame`, `setupShootCandidateSouth`)
- `bench_helpers_test.go` — Six canonical benchmark fixtures
- `random_test.go`, `heuristic_test.go`, `analysis_test.go`, `pimc_test.go` — Strategy tests
- `pimc_sample_test.go`, `pimc_rollout_test.go`, `pimc_aggregate_test.go` — PIMC component tests
- `stats_test.go` — AI statistical profiles
- `*_bench_test.go` — Benchmarks for each strategy

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Change random behavior | `random.go` | `ChoosePlay` and `ChoosePass` |
| Change heuristic rules | `heuristic.go` | Documented decision tree |
| Add analysis helpers | `analysis.go` | Card counting, void detection, Q♠ location, moon threat |
| Change PIMC orchestration | `pimc.go` | Sampling, rollout, aggregation pipeline |
| Change PIMC sampling | `pimc_sample.go` | Possible hand generation |
| Change PIMC simulation | `pimc_rollout.go` | Random rollout to end of round |
| Change PIMC scoring | `pimc_aggregate.go` | Rollout result aggregation |
| Add test fixtures | `helpers_test.go` / `bench_helpers_test.go` | Shared fixtures |
| Understand PIMC design | `doc/games/hearts/ai-pimc-design.md` | Design rationale |

## CONVENTIONS

- `rand.Rand` is injected via constructor; no global RNG state.
- `Player` interface methods return values, never errors.
- `analysis.go` is pure read-only computation; decisions live in `heuristic.go`.
- Heuristic decision tree is documented in `heuristic.go`.

## ANTI-PATTERNS

- Never mutate the game state from AI code.
- Never cast `[]Card` to modify the engine's trick; AI returns selections.
- Never rely on side effects from analysis.

## COMMANDS

```bash
go test ./games/hearts/ai/...
go test -bench=. ./games/hearts/ai/...
go test -run TestShootTheMoonFrequency ./games/hearts/...
```
