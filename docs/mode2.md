# Mode 2: Reasoning Over Observations

Mode 2 connects nous to pudl's bitemporal fact store. Instead of running operations on in-memory data (Mode 1), nous loads accumulated observations from pudl, reasons over them with heuristics designed for correlation and pattern detection, and writes discoveries back to pudl as new facts.

## The Loop

```
agents/humans                      nous                        pudl
      |                              |                           |
      |--- pudl observe ------------>|                           |
      |                              |                           |
      |                              |<-- load observations -----|
      |                              |                           |
      |                          [fire heuristics]               |
      |                          [find hotspots]                 |
      |                          [form conjectures]              |
      |                              |                           |
      |                              |--- write discoveries ---->|
      |                              |                           |
      |<-- pudl facts list ----------|                           |
      |<-- pudl query ---------------|                           |
```

Agents and humans record observations via `pudl observe`. nous loads those observations, reasons over them, and writes discoveries (conjectures, hotspots) back to pudl as facts. Those facts are then queryable by anyone — including future nous runs.

## Running Mode 2

```bash
# Prerequisite: observations exist in pudl
pudl observe "auth has circular dependency with user" --kind obstacle --scope myapp:pkg/auth
pudl observe "config struct is too large" --kind suggestion --scope myapp:internal/config
# ... more observations from agents, humans, CI hooks, etc.

# Run nous in Mode 2
nous run -domain observations -pudl ~/.pudl -cycles 50 -no-mutate
```

| Flag | Description |
|------|-------------|
| `-domain observations` | Load Mode 2 types and heuristics |
| `-pudl DIR` | pudl config directory (e.g. `~/.pudl`). Enables Mode 2. |
| `-cycles N` | Maximum engine cycles (default 100) |
| `-v N` | Verbosity (0=quiet, 1=normal, 2=detailed, 3=debug) |
| `-no-mutate` | Disable heuristic mutation (recommended for Mode 2 until mutation vocabulary is adapted) |

On startup, nous:
1. Loads the observation domain (type hierarchy + heuristics)
2. Opens the pudl catalog via the bridge
3. Reads all current observations as units with `isA: ["Observation"]`
4. Reports how many observations were loaded

On exit, nous writes discoveries back to pudl:
- **Conjectures** become observations with `source: "nous"` and `kind` from the conjecture
- **Scope hotspots** become `scope_hotspot` facts with scope and observation count

## How Observations Become Units

Each observation from pudl becomes a nous unit:

```
pudl fact:
  relation: "observation"
  args: {"kind": "obstacle", "description": "circular dep in auth", "scope": "myapp:pkg/auth", "source": "agent-1", ...}

nous unit:
  name: "Obs-circular-dep-in-auth"
  isA: ["Observation", "Anything"]
  kind: "obstacle"
  description: "circular dep in auth"
  scope: "myapp:pkg/auth"
  source: "agent-1"
  worth: 500
```

The fact's JSON args are mapped directly to unit slots. Worth defaults to 500 (mid-range on the 0-1000 scale).

## Mode 2 Heuristics

Five seed heuristics designed for observation reasoning. They operate on structural properties of observations — kind, scope, source, worth — not on data content.

### H-FindScopeHotspots (worth 700)

Fires on each Observation. Counts how many observations share the same `scope`. If 2 or more, creates a `ScopeHotspot` unit.

**What it detects:** Scopes that attract disproportionate attention — multiple agents or humans independently flagging issues in the same place.

### H-CorroborateObstacles (worth 650)

Fires on Observation units where `kind=obstacle`. Checks if other sources reported obstacles in the same scope. If corroborated, boosts the observation's worth.

**What it detects:** Obstacles confirmed by independent observers. Corroboration is a strong signal that the obstacle is real and important.

### H-ConjectureFromPatterns (worth 600)

Fires on each Observation. Counts how many distinct scopes have the same `kind`. If 3 or more scopes share the same kind, creates a `Conjecture` unit describing a systemic issue.

**What it detects:** Cross-cutting patterns — when the same kind of observation (e.g., "suggestion" or "antipattern") appears across many scopes, it may indicate a systemic problem rather than a local one.

### H-BoostCorroborated (worth 500)

Fires on each Observation. Checks if any other source recorded an observation with the exact same description. If so, boosts worth.

**What it detects:** Exact corroboration — two agents independently saying the same thing.

### H-PenalizeStaleObservations (worth 400)

Fires on each Observation. If the observation's status is still "raw" (never reviewed or promoted), slightly decreases its worth.

**What it does:** Implements decay. Observations that go unreviewed gradually lose influence, preventing the system from being dominated by stale, unvalidated assertions.

## What Nous Writes Back

After the engine runs, discoveries are written to pudl:

### Conjectures → Observations

Conjecture units become new observations in pudl with `source: "nous"`. These are distinguishable from human/agent observations and represent nous's own conclusions.

```
nous → pudl:
  relation: "observation"
  args: {"kind": "pattern", "description": "Systemic issue: pattern observed across 3 scopes", ...}
  source: "nous"
```

### Scope Hotspots → Facts

ScopeHotspot units become `scope_hotspot` facts:

```
nous → pudl:
  relation: "scope_hotspot"
  args: {"scope": "pudl:internal/database", "observation_count": 3}
  source: "nous"
```

These are queryable via `pudl facts list --relation scope_hotspot` or through Datalog rules.

## Differences from Mode 1

| | Mode 1 | Mode 2 |
|---|---|---|
| **Data source** | In-memory seed data | pudl fact store |
| **Operations** | Run functions on data | Query/correlate observations |
| **Feedback** | Immediate (same cycle) | Accumulated over time |
| **Heuristics** | H-RunOnExamples, H-Specialize, etc. | H-FindScopeHotspots, H-CorroborateObstacles, etc. |
| **Evaluation** | Algorithmic (set equality, structural) | Statistical (counts, clustering, corroboration) |
| **Write-back** | None (in-memory only) | Conjectures and hotspots to pudl |
| **Mutation** | Token-level DSL mutation | Not yet adapted (use `-no-mutate`) |

## The Bridge (`internal/pudlbridge`)

The bridge is the Go interface between nous and pudl. It imports pudl's public API packages (`pkg/factstore`, `pkg/eval`) and provides:

- `New(pudlDir)` — open connection to pudl's catalog
- `LoadObservations(store)` — read observations, create units
- `LoadDerived(store, rules)` — evaluate Datalog rules, load IDB as units
- `WriteFact(relation, args, source)` — write a fact back to pudl
- `QueryFacts(rules, relation, constraints)` — evaluate and query
- `ScanFacts(relation)` — read raw EDB

The bridge uses pudl's public packages rather than internal packages. pudl exposes `pkg/factstore` (fact CRUD) and `pkg/eval` (Datalog evaluator, EDB constructors, rule loading) for this purpose.

## Current Limitations

- **No mutation in Mode 2.** The mutation vocabulary (swap tokens, widen numerics, etc.) is designed for Mode 1's stack DSL programs. Mode 2 heuristics would need a different mutation strategy. Use `-no-mutate` for now.
- **No incremental loading.** Each run loads all current observations from scratch. For large observation volumes, incremental loading (only new observations since last run) would be needed.
- **No Datalog rule evaluation on load.** `LoadDerived` exists but isn't called from main.go yet. The current flow loads raw observations only. Wiring in Datalog-derived facts as units is a natural next step.
- **Heuristic vocabulary is small.** Five seed heuristics cover basic patterns (hotspots, corroboration, systemic conjectures, decay). More sophisticated reasoning (temporal correlation, causal inference, prediction) requires expanding the vocabulary.
- **Human validation gate not built.** The slow loop (human review of nous's conjectures before promotion) is described in the architecture doc but not implemented. Currently, nous writes directly to pudl without human review.
