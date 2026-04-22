# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

### Fixed
- Release workflow no longer fails when changelog entries contain shell metacharacters such as apostrophes

## [0.3.0] - 2026-04-21

### Added
- AI prerequisites: Clone, LegalMoves, and Player interface (`games/hearts/`)
- Random AI player (`games/hearts/ai/`)
- Heuristic AI player (`games/hearts/ai/`)
- Trick history and pass history to Hearts game state (`games/hearts/`)

### Fixed
- Prevent uint8 underflow panic in heuristic AI's moon-shoot detection when all hearts have been played (`games/hearts/ai/`)

## [0.2.0] - 2026-04-14

### Added
- Hearts card game engine (`games/hearts/`)

## [0.1.0] - 2026-04-08

### Added
- Core engine primitives: Suit, Rank, Card, Deck, and Hand
