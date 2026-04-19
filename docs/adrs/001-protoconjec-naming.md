# ADR-001: ProtoConjec readable names + supportCount dedupe

**Date:** 2026-04-18
**Status:** Accepted
**Phase:** 7.5
**Commits:** 9518b30

## Context

Phase 7.5 introduces ProtoConjec as a real unit category in the store. Every firing of H-Conjecture (and later H1, H16, …) creates a conjecture unit. In a 300-cycle math-domain run we expected thousands of these — rapid re-derivation of the same equalities/subsets is normal as the op space grows.

Two name-scheme options were on the table:
1. Readable: `Conjec-<kind>-<sorted-about-joined-by-dash>` e.g. `Conjec-SetEqual-SetOfNumbers-SetOfPrimes`.
2. Uniform opaque: `Conjec-00042`, auto-numbered.

## Decision

Readable names, deduplicated on `(kind, sorted-about)`. When `make-protoconjec` is called with a combination that already exists, it returns the existing unit name and increments its `supportCount` slot rather than creating a twin.

## Alternatives considered

- **Opaque auto-numbering**: easier to generate, avoids collision anxiety, but loses at-a-glance readability when debugging 4000+ conjectures in a log. Also sheds useful structural information — the name itself encodes what the conjecture is about.
- **No dedupe**: let each firing produce a distinct unit. Rejected because it doubles as evidence-strength tracking — `supportCount` cheaply records "this was re-derived N times", which is real information about robustness.

## Consequences

- Debugging is drastically easier: log lines like "killed Conjec-SetEqual-X-Y" tell you immediately what was killed and why.
- `supportCount` becomes a natural evidence measure. Future heuristics can weight conjectures by it.
- Name length grows with `about`-list length, which for binary-pred conjectures over specialized-op names can be very long (e.g. `Conjec-SubsetOf-SetIntersect-on-EmptySet-on-SetOfPrimes-SetOfEvens-SetUnion-on-SetOfNumbers-SetOfOdds`). Lived with — it's information, not noise.
- The sorted-about canonicalization means `Conjec-SetEqual-A-B` and `Conjec-SetEqual-B-A` dedupe to the same unit. Correct for commutative kinds (SetEqual), questionable for ordered kinds (SubsetOf). For SubsetOf we currently accept the collision; if it becomes a problem we'd teach make-protoconjec to sort only for symmetric kinds.
