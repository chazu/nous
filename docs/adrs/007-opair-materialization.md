# ADR-007: H25/H26 materialize OPair units per tested pair

**Date:** 2026-04-19
**Status:** Accepted
**Phase:** 5.2
**Commits:** 40f9872

## Context

H25/H26 produce `SatisfyingSetFor<pred>` / `FailingSetFor<pred>` categories for binary predicates. The category's *examples* need to reference the tested pairs somehow. Two representations:

1. **Materialize an OPair unit per pair**, with deterministic names (`OPair-<a>-<b>`) and dedupe via `unit-exists?`. Category examples are unit names.
2. **Store raw tuples in a `pairs` slot**: `[[a, b], [c, d], ...]`. Breaks the "examples are unit names" invariant but produces no new units.

## Decision

Option 1. Materialize OPair units.

## Alternatives considered

- **Raw-tuples slot**: simpler, no unit-creation cascade. Rejected because downstream heuristics (H24, H-Conjecture, H8, …) all assume `examples` is a list of unit names with `data` slots. A parallel "raw pairs" representation would require teaching every downstream consumer two codepaths. The invariant is load-bearing.
- **Materialize only the satisfying (or only failing) pairs** and store the other side as tuples: asymmetric, rejected for the same downstream-consistency reason.

## Consequences

- A 300-cycle math run creates 115 OPair units (on SetEqual + SubsetOf × Set × Set examples). That's one firing per pair produced. Dedupe means re-derivation across the two heuristics (H25 and H26 can produce `OPair-A-B` on distinct runs) is a cheap pointer-bump, not a duplicate unit.
- OPair units are themselves first-class: H24 can look for rare predicates on them, H-Conjecture can find equalities among them, future heuristics can treat them as any other MathObj. This is the upside — we traded unit-count for analytical uniformity.
- The `OPair-<a>-<b>` name encodes the pair inline. For deeply-nested arg names (e.g. `OPair-SetIntersect-on-EmptySet-on-SetOfOdds-...`) the names grow long. Fine; same tradeoff as ADR-001.
- No `Pair` (unordered) type or `ReverseOPair` operation yet. Deferred — binary-pred analysis only needs ordered pairs; unordered pairs are a separate abstraction whose consumers don't exist yet.
