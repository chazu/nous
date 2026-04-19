# ADR-011: Type predicates via is-int? / is-list? kind introspection

**Date:** 2026-04-19
**Status:** Accepted
**Phase:** 5.6 C.1
**Commits:** d733c13

## Context

Phase 5.6 C.1 needs `Number`, `Set`, `List`, `Bag` to have executable `defn` slots that act as type tests — necessary input for H8's positional-domain-match check (Phase 4.4's blocker).

What does "is this value a Number?" mean in our DSL? Our numeric values are `VInt`; set data is stored as `VList` of elements. The distinction is at the Value-kind level, not at any semantic level.

Options for implementing type predicates:

1. **Kind introspection**: new DSL builtins (`is-int?`, `is-list?`, `is-string?`) that check the top-of-stack Value's internal kind. Type defns become one-liner references: `Number.defn = "is-int?"`, `Set.defn = "is-list?"`.
2. **Structural semantic tests**: `Set.defn` checks "is a list AND has no duplicates"; `List.defn` just checks is-a-list; `Bag.defn` likewise. Semantically richer, but requires each type to have a bespoke test.
3. **Meta-op route** (the more EURISKO-faithful option hinted at by the original Phase 5.6 framing): type predicates are *produced* by a meta-op that introspects type structure. Much more infrastructure.

## Decision

Option 1. Kind introspection with three small builtins; all structural types get `is-list?` as their defn; Number gets `is-int?`.

Set, List, and Bag all share `defn = "is-list?"` — we don't try to distinguish them at the kind level.

## Alternatives considered

- **Distinguish Set vs List vs Bag**: we'd need "has no duplicates" for Set, "anything goes" for List, "allows repeats" for Bag. But we already store Set data as ordinary `VList`; deduplication is a convention enforced by `make-set`, not by the kind system. Pretending otherwise at the type-predicate level would lie.
- **Meta-op route**: more faithful long-term but pure infrastructure work with no downstream consumer ready. Explicitly rejected — "ship fast, refactor later if needed" was the user's call.

## Consequences

- `apply-pred` on Set, List, or Bag returns `true` for any list value. H8 uses this as a *necessary* type check: if arg is a list, it's at least plausibly a Set. H8 then relies on the op's `defn` to handle the actual computation, so a non-Set list passed in would either produce a garbage result or fail cleanly.
- Number.defn correctly rejects list values, which is H8's tighter constraint for numeric ops.
- Downstream discovery heuristics (H25–H28) don't use type defns currently, so the coarseness is fine. If a future heuristic needs "is this specifically a Set", we'd add a `set-invariants?` that checks no-duplicates — at that point, Option 2's semantic richness becomes worth the cost.
- The three-builtin addition is cheap: each is a one-liner examining `Value.Kind()`. No risk of breaking existing programs.
