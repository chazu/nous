# Architecture Decision Records

ADRs here record *why* non-obvious design choices were made, not *what* the
code does. Every ADR below corresponds to a conscious fork in the road where
a different answer was genuinely possible. Read the linked commit(s) for the
resulting implementation.

## Index

| # | Title | Phase | Date |
|---|---|---|---|
| [001](001-protoconjec-naming.md) | ProtoConjec readable names + supportCount dedupe | 7.5 | 2026-04-18 |
| [002](002-h1-eurisko-faithful.md) | H1 emits conjec AND enqueues spec task | 4.1 | 2026-04-18 |
| [003](003-pred-interestingness-gate.md) | Predicate interestingness gate: worth OR isAInt OR rare-rarity | 5.11 | 2026-04-18 |
| [004](004-h27-h28-one-shot-whyint-seed.md) | H27/H28 one-shot via unit-exists? + auto-seed whyInt on ≥4 members | 5.11 | 2026-04-18 |
| [005](005-protoconjec-h19-exemption.md) | Exempt ProtoConjec from H19Criterial duplicate-killer | 7.5 follow-up | 2026-04-19 |
| [006](006-predicate-wakeup.md) | Wake up H27/H28 via seed-worth bumps + H-ExercisePreds | 5.11 follow-up | 2026-04-19 |
| [007](007-opair-materialization.md) | H25/H26 materialize OPair units per tested pair | 5.2 | 2026-04-19 |
| [008](008-configurable-caps-per-heuristic.md) | Configurable caps live as per-heuristic slots | 5.2, 5.6, 7.2 | 2026-04-19 |
| [009](009-tests-match-production-seed.md) | Tests that exercise the engine end-to-end must call SeedInitialAgenda | cross-cutting | 2026-04-19 |
| [010](010-phase-5-6-slicing.md) | Slice Phase 5.6 into independent A/B/C/D pieces | 5.6 | 2026-04-19 |
| [011](011-type-predicates-kind-introspection.md) | Type predicates via is-int?/is-list? kind introspection | 5.6 C.1 | 2026-04-19 |
| [012](012-generator-format.md) | Generator format = `{initial: [values], step: "<dsl>"}` | 7.2 | 2026-04-19 |

## Template

```markdown
# ADR-NNN: <Title>

**Date:** YYYY-MM-DD
**Status:** Accepted | Superseded by ADR-MMM | Deprecated
**Phase:** <phase number or "cross-cutting">
**Commits:** <short SHA> [, <short SHA> ...]

## Context
What was the situation? What made this a choice rather than a no-op?

## Decision
The chosen path, stated concisely.

## Alternatives considered
Other options that were on the table, and why they lost.

## Consequences
What this commits us to, and what it forecloses. Include honest
tradeoffs and known risks.
```
