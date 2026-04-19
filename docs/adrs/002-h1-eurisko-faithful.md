# ADR-002: H1 emits conjec AND enqueues spec task

**Date:** 2026-04-18
**Status:** Accepted
**Phase:** 4.1
**Commits:** 492a593, dc760b9

## Context

EURISKO's H1 does two things when it notices an op has >4/5 bad results: it creates a `ProtoConjec` about the op's failure pattern, and it enqueues a specialization task on the op. We had to decide whether to replicate both or just the first.

Option A (emit-only): H1 creates the ProtoConjec and stops. A later, more sophisticated heuristic would pick up the conjecture and decide whether/how to specialize.

Option B (EURISKO-faithful): H1 creates the ProtoConjec *and* pushes a bare `<op>.specializations` task. The existing H3-RandomSlot / H5-Criterial proposers then pick up the bare task and turn it into a populated specialize task for H6-Specialize.

## Decision

Option B. Stay close to EURISKO.

## Alternatives considered

- Emit-only (A). Cleaner separation of concerns, easier to reason about. Rejected because it defers the loop-closing to a heuristic that doesn't exist yet.
- Have H1 itself pick the specialization target (analyze which args failed, choose a narrowing type). Rejected as too much scope for H1 — the proposer heuristics already do that job.

## Consequences

- The loop closes without additional glue: H1 → bare spec task → H3/H5 → populated spec task → H6-Specialize → new specialized op. Verified end-to-end in `TestH1SpecTaskReachesProposers`.
- H3/H5 pick the specialization *randomly* within the criterial-slot space, whereas EURISKO's H1 analyzed which specific targets failed and picked a narrowing that would exclude them. Our version is coarser. A future enhancement would be a failure-pattern-aware proposer. Documented as future work in the commit message, not in this ADR's scope.
- Investigation tripwire: during review we noticed "H1 enqueues a task with no Extra; is that a dead letter?" — the answer (no, H3/H5 gate on empty Extra and are the designed proposers for bare tasks) was non-obvious enough to warrant its own test.
