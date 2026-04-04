# nous Operating Modes

## Mode 1: Tight Simulation Loop

The original EURISKO model. nous runs operations on in-memory data, checks results immediately, and adjusts worth in the same cycle.

**Characteristics:**
- Feedback is instant — run an operation, evaluate the result, update credit
- The "world" is entirely inside the unit store
- Operations are pure functions: input data → output data
- The agenda loop runs continuously until it exhausts its exploration budget
- Evaluation is algorithmic — equality, subset, size, structural properties
- A single run explores a concept space from seed to saturation

**What it requires from a domain:**
- Operations must be computable in-process (no external systems)
- Results must be evaluable without human judgment or real-world observation
- The concept space must be rich enough to sustain exploration (compositions, specializations)
- Data representations must be concrete enough to compute over

**Current example:** The math/set-theory domain. `SetIntersect(Primes, Evens)` returns `{2}` in microseconds. H-Conjecture checks equality immediately. Credit assignment fires in the same cycle.

**Strengths:** Fast iteration, deterministic, self-contained, easy to test and debug.

**Limitations:** Only works for domains where "run it and check" is possible. Can't handle domains where evaluation requires real-world observation, takes weeks, or depends on context that isn't in the model.

## Mode 2: Accumulate-Then-Reason

nous operates over a growing corpus of structured event data, ingested over time by pudl. Instead of running operations and checking results, it finds patterns, forms conjectures, and builds heuristic rules from historical observations.

**Characteristics:**
- Feedback is delayed — events arrive over days/weeks, nous reasons over the accumulated record
- The "world" is external; nous only sees what pudl has ingested
- Operations are queries and correlations, not transforms
- The agenda loop runs periodically (after new data arrives, or on a schedule), not continuously
- Evaluation is statistical — correlations, frequencies, trend detection
- Conjectures are tested by whether new data confirms or contradicts them

**What it requires from a domain:**
- A stream of structured events with typed slots (timestamp, category, outcome, metrics)
- pudl ingesting and cataloging these events with schema inference
- Enough event volume to detect patterns (not one-off occurrences)
- Quality attributes that are observable, not just computable

**Hypothetical example:** DevOps/infrastructure operations. pudl ingests deploy events, incident reports, cost snapshots, metric percentiles. nous runs periodically and discovers: "deploys to service X on Fridays have 3x rollback rate" or "services that added a cache layer saw p99 drop but incident rate increase." These conjectures get worth based on how well they predict future events.

**Strengths:** Operates on real-world data, can discover things no simulation would find, directly actionable, improves with more data.

**Limitations:** Slow feedback (weeks, not microseconds), requires real event streams, statistical rather than deterministic, harder to test in isolation.

## Architectural Implications

### What stays the same in both modes

- Units, slots, worth, isA hierarchy — the knowledge representation
- Agenda, priority queue, task merging — the control flow
- Heuristic firing — IfParts/ThenParts pattern matching
- Credit assignment — worth changes based on success/failure
- HindSight — learning avoidance rules from failures
- Mutation — token-level variation of heuristic programs
- The DSL and interpreter

### What changes between modes

| Aspect | Mode 1 | Mode 2 |
|--------|--------|--------|
| Data source | Seed data in unit store | pudl catalog (external ingest) |
| Operations | Transforms (compute new data) | Queries (correlate existing data) |
| Evaluation | Immediate (algorithmic) | Delayed (statistical, predictive) |
| Loop timing | Continuous (tight loop) | Periodic (after new data) |
| "Success" | Correct result, interesting structure | Prediction confirmed by new data |
| DSL builtins needed | Set ops, arithmetic, apply-op | Aggregation, filtering, counting, correlation |
| pudl integration | Optional (Phase 3) | Required (core to the model) |
| mu integration | Optional (Phase 4) | Optional (for triggering observations) |

### Hybrid possibility

A domain could use both modes. Mode 1 for fast in-process exploration of structural possibilities, Mode 2 for grounding those explorations against real-world observations. For example:

1. Mode 1 explores software architecture transforms in simulation
2. Mode 2 ingests real codebase metrics from pudl (test coverage, dependency counts, change frequency)
3. Conjectures from Mode 1 are tested against Mode 2 data
4. Mode 2 patterns inform Mode 1 heuristics (e.g., "the simulation says splitting this module helps, but historical data shows splits of modules with >20 dependents always increase defect rate")

This is essentially the ACUTE loop from the design doc, with nous sitting between the simulated and observed worlds.

## Relationship to the three-loop architecture

Mode 1 and Mode 2 are descriptions of how nous itself operates. The three-loop architecture (see `docs/architecture.md`) describes how nous fits into the larger system:

- **Fast loop (Datalog):** Provides the IDB that Mode 2 nous reasons over
- **Medium loop (nous):** Operates in Mode 1 or Mode 2 depending on the domain
- **Slow loop (human validation):** Gates what gets promoted regardless of mode

Mode 1 doesn't need the fast or slow loops — it's self-contained. Mode 2 requires the fast loop (Datalog deriving facts for nous to reason over) and benefits from the slow loop (human validation of discovered patterns).

The progression is: build and validate the engine in Mode 1, then wire it to external data sources for Mode 2. Mode 1 domains are test harnesses; Mode 2 is the production architecture.
