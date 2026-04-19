# ADR-008: Configurable caps live as per-heuristic slots

**Date:** 2026-04-19
**Status:** Accepted
**Phase:** 5.2, 5.6, 7.2
**Commits:** 40f9872, 944525f, d213ea9

## Context

Several new heuristics need a numeric cap to bound cost:

- `H25`/`H26`: `pairCap` — max Cartesian pairs tested per firing (default 50)
- `H8`: `h8Cap` — max applics propagated from a generalization per firing (default 3)
- `H-Generate`: `generateCount` — how many instances to produce (default 10)

Three places the cap could live:

1. Hardcoded DSL literal inside the heuristic's CUE definition.
2. Engine-level Go knob (e.g. `Engine.PairCap int`).
3. Per-heuristic slot on the heuristic unit itself.

## Decision

Option 3. Each cap is a slot on the heuristic unit, read at firing time via `"HeuristicName" "capSlotName" get-slot`.

## Alternatives considered

- **Hardcoded DSL literal**: simplest. Rejected because mutations — both human and machine — should be able to tune caps without re-editing source. An H-AnalyzeApplics-like heuristic could plausibly learn to raise `pairCap` on ops whose discovery rate is bottlenecked.
- **Engine-level Go knob**: allows uniform tuning across all heuristics, but obscures per-heuristic semantics. Different caps serve different purposes; flattening them into one config knob loses information.
- **Task-Extra passed per-task**: heuristics could read from task context. Too specific — the cap is a property of the heuristic, not of any particular task.

## Consequences

- Caps live in CUE alongside the heuristic, discoverable via `get-slot`. A user (or a learning heuristic) can mutate `"H25" "pairCap" 100 set-slot` at runtime to try a different bound.
- Slot names (`pairCap`, `h8Cap`, `generateCount`) aren't registered in `domains/common/slots.cue`. The store happily accepts them anyway since slot enforcement is convention, not schema. Similar to existing flag slots like `specTaskAdded`, `predsExercised`, etc.
- If H-Specialize ever clones a heuristic (unlikely for these three, but possible for general mutation), the clone inherits the cap slot automatically since `set-slot` copies all slots. No special handling needed.
- Tests exploit this directly: `TestH25PairCap` calls `eng.Store.SetSlot("H25", "pairCap", 3)` to force a tiny cap and verify the cap is honored.
