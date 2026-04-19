# ADR-012: Generator format = `{initial: [values], step: "<dsl>"}`

**Date:** 2026-04-19
**Status:** Accepted
**Phase:** 7.2
**Commits:** d213ea9

## Context

Phase 7.2 introduces the `Generator` slot (its definition existed but was unused). EURISKO's format is a Lisp triplet like `((0) (ADD1) (old))`: initial value list, step operation, a hint about reusing prior results. We needed a nous-native representation that fits our DSL and slot system.

Options:

1. **Lisp-style literal port**: store the triplet as a nested list, interpret in Go.
2. **Map with named fields**: `{initial: [...], step: "<program>"}` — step is a DSL program string.
3. **Single DSL program**: `step(prevList) -> nextElement`, no structural separation of initial state vs step. Caller supplies the initial list out-of-band.

## Decision

Option 2. A map with `initial` (list of starting values) and `step` (DSL program string).

The step program receives the previous *single* value on the stack and leaves the next value on top. EURISKO's `(old)` reuse hint — which would pass the entire accumulated list to the step — is intentionally *not* implemented.

## Alternatives considered

- **Lisp-style triplet**: more faithful but adds parsing complexity and has no real benefit over a named-field map. The triplet's positional encoding (index 0 = initial, index 1 = step, index 2 = flags) is strictly less readable than field names.
- **Single DSL program taking the whole prev list**: more general — Fibonacci needs two prior values, so you'd want at least the last two. But the math domain's one seeded use (counting integers on `Number`) doesn't need it. Deferred; if a future generator does need history, we either extend `step`'s stack contract or add an `{initial, step, history: 2}` variant.

## Consequences

- `Number`'s generator `{initial: [0], step: "1 +"}` produces `[0, 1, 2, 3, ...]` via the one-value step contract.
- Fibonacci-style generators (which need two prior values) can't be expressed with current `step` semantics. Acceptable — no consumer needs that yet.
- The generator value is a `map[string]any` in the store. `run-generator` reads `initial` as either `[]any` or `[]int` (CUE encoding varies). Minor annoyance; hidden inside the builtin.
- The step program runs on a fresh sub-VM each iteration (shared store, fresh stack). Same pattern as `apply-op` / `apply-pred`. Side effects in the step program would affect the store globally — deliberate choice, lets generators that create units work.
- No way currently to *reset* a generator's `generated` flag so it produces more. One-shot by design (ADR-004 logic). If a unit gains more examples via other means and we want the generator to top it up, that's future work.
