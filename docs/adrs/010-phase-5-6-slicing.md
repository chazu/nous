# ADR-010: Slice Phase 5.6 into independent A/B/C/D pieces

**Date:** 2026-04-19
**Status:** Accepted
**Phase:** 5.6
**Commits:** d733c13, 944525f

## Context

Phase 5.6 as originally spec'd in `docs/eurisko-parity-phases.md` was "Meta-operations with algorithms: give Compose and Restrict executable defn slots. Implement InvertOp. Implement Transpose." Three meta-ops plus tangent support for type predicates and H8 (the Phase 4.4 blocker).

Bundled as one commit, the phase would touch: a new generic mechanism for meta-op defns, four heuristics, type-predicate defns on every structural type, H8 implementation. Big diff, hard to review, hard to roll back any one piece.

The four meta-ops also don't share much implementation. Transpose is a one-line arg-swap. Compose needs domain/range compatibility checks. InvertOp is mathematically complex (partial inverses). Restrict already has a partial implementation (the `restrictedTo` slot plus H6-Specialize). They're naturally independent.

## Decision

Slice Phase 5.6 into four pieces, shippable in any order:

- **A — Transpose + H-Transpose** (small; creates `Transpose-<op>` variants for non-commutative binary ops)
- **B — Compose + H-Compose** (medium; supersedes H-CheckDomain's SelfCompose hack)
- **C — Type predicates + H8** (medium; unblocks Phase 4.4)
- **D — Restrict + InvertOp** (varied; Restrict partial, InvertOp complex)

Ship C first because it unblocks a long-stuck phase-4 issue (H8) and directly serves discovery density. A/B/D deferred until there's a concrete pull.

## Alternatives considered

- **Ship the whole phase at once**: conventional reading of the phase spec. Rejected because the pieces genuinely don't depend on each other, and the review/rollback burden of one big commit outweighs the narrative neatness of "Phase 5.6 done".
- **Slice differently** (e.g. infrastructure-first, with all meta-ops landing together later): meta-op infrastructure turned out to be thin — each one synthesizes a new op unit independently. There's no shared mechanism worth extracting ahead of time.

## Consequences

- The phase doc now shows 5.6 as PARTIAL with per-slice status. Readers who search for "Phase 5.6 complete" will have to understand the slicing.
- A/B/D are at genuine risk of never being shipped if they don't accumulate pull. This is fine — the original EURISKO parity goal was to reach operational behavior, not to tick every box. Transpose specifically has low immediate leverage (math ops are mostly commutative; SetDifference is the rare non-commutative case we care about). InvertOp is complex enough that deferring it is probably the right long-term call.
- The slicing pattern generalizes: when a phase's sub-issues are structurally independent, treat them as independent phases. Applied retroactively to Phase 5.2 (PARTIAL: OPair done, Pair/ReverseOPair deferred) and Phase 5.11 (PARTIAL: unary H27/H28 done, n-ary H25/H26 done via 5.2).
