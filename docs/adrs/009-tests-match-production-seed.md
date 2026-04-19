# ADR-009: Tests that exercise the engine end-to-end must call SeedInitialAgenda

**Date:** 2026-04-19
**Status:** Accepted
**Phase:** cross-cutting
**Commits:** 40f9872

## Context

While landing Phase 5.2, bumping predicate seed worths to 700 (to wake up H25–H28; see ADR-006) broke two long-standing tests: `TestSpecializationPipeline` and `TestH19CriterialSparesSpecializations`. Both were originally written before Phase 5.11's predicate additions and set `MaxCycles` to tight values (50, 60) for speed.

Neither test called `SeedInitialAgenda`. Without that seed, `engine.Run` starts in unit-focus mode (agenda empty → pick highest-worth unfocused unit). Before the predicate worth bump, unit-focus cycled through heuristics first, then hit ops like `SetIntersect` at worth 500, giving H-Specialize a chance to fire. After the worth bump, `IsEmpty`/`IsSingleton`/`SetEqual`/`SubsetOf` all jumped to worth 700, pushing into the focus queue ahead of ops. Within 50 cycles the engine never got to an op; H-Specialize never fired; the tests failed.

The production binary has *always* called `SeedInitialAgenda` in `cmd/nous/main.go` before `Run`. The tests were simulating a different execution environment.

## Decision

Tests that purport to verify end-to-end engine behavior must call `eng.SeedInitialAgenda()` before `eng.Run(...)`, matching production. The two failing tests were updated accordingly.

## Alternatives considered

- **Revert the worth bumps**: would leave H25–H28 inert in production runs. Rejected; the discovery-layer wake-up was the whole point of the bumps.
- **Raise `MaxCycles` further**: tried 100, 150, 300 — all still failed because the agenda-starving shift is structural, not temporal. The predicate focuses cause tasks to enter the agenda, switching the engine to task-focus mode, and ops at worth 500 are never reached.
- **Make `SeedInitialAgenda` run automatically inside `Run`**: considered, rejected because tests that exercise focused sub-behaviors (e.g. `fireUnitRule("H27", "IsEmpty")` directly) legitimately don't want it. Keeping the call explicit preserves the escape hatch.

## Consequences

- Future tests that invoke `eng.Run` as a black box should follow this pattern. The existing tests that drive the engine indirectly (via `fireUnitRule` / `WorkOnTask`) don't need it.
- This surfaced that the unit-focus focus queue is *worth-sensitive* in ways the engine doesn't advertise. A unit being worth 700 vs 500 changes whether a test sees it before the cycle budget runs out. This is expected behavior but worth flagging.
- The honest reading of the earlier test pass is that it was narrowly timing-dependent. The re-grounded version is more robust but still cycle-bounded.
