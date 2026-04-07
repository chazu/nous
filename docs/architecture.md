# nous — System Architecture

## What nous is

nous is a EURISKO-style discovery engine — an agenda-driven heuristic interpreter that explores concept spaces, evaluates results, learns from failures, and modifies its own heuristics. It sits between a knowledge layer (pudl) and an execution layer (mu) as the reasoning middle.

```
                    ┌─────────────────────┐
                    │      nous           │
                    │  agenda + heuristics │
                    │  credit assignment   │
                    │  meta-learning       │
                    └──────┬──────┬───────┘
                           │      │
              read state   │      │  emit actions
              + schemas    │      │
                    ┌──────▼──┐ ┌─▼──────────┐
                    │  pudl   │ │    mu       │
                    │ (knows) │ │   (acts)    │
                    └────▲────┘ └──────┬──────┘
                         │             │
                         └─── results ─┘
                       re-ingest outputs,
                       detect drift
```

## What exists today

~4,700 lines of Go. Zero external dependencies.

### Core engine (Phase 1)
- **Stack-based DSL** — Forth-style interpreter (`if ... then`, `each ... end`) with ~60 builtin words
- **Unit store** — in-memory map of named units with typed slots (the EURISKO property-list model)
- **Agenda** — priority queue with duplicate merging and priority boosting
- **Main loop** — two-level control: Level 1 pops tasks from the agenda, Level 2 focuses on highest-worth unfocused units
- **Heuristic firing** — IfParts filtering → ThenParts execution (the Interp2 equivalent)

### Rich math domain
- Concrete data: sets of numbers (primes, evens, odds), executable operation definitions
- 11 seed heuristics: H-FindExamples, H-RunOnExamples, H-CheckExtremes, H-Specialize, H-CheckDomain, H-Conjecture, H-ExploreSlots, H-KillWorthless, H-PenalizeTrivial, H-BoostInteresting
- Domain registry: `nous run -domain math` (extensible to other domains)

### Credit assignment
- Worth-based evaluation: units gain/lose worth based on utility
- Creditor tracking: every machine-created unit records who made it
- Punishment: when a unit is killed, its creditors' worth is halved
- Reward: interesting results (singletons, novel structures) boost creditor worth
- Graveyard: dead units' metadata preserved for HindSight analysis

### HindSight (Phase 2-lite)
- When a unit is killed, HindSight creates an HAvoid-* avoidance heuristic
- The avoidance rule blocks the creditor from repeating the same class of mistake
- Avoidance heuristics participate in the normal heuristic pool

### Self-modification (Phase 5)
- Token-level mutation of heuristic DSL programs
- 7 mutation operators: swap, delete, insert, widen numeric, narrow numeric, replace unit ref with generalization/specialization, duplicate subsequence
- Validation by trial execution before acceptance
- Mutant heuristics enter the pool with their own worth, subject to the same credit assignment
- Mutants of mutants possible (second-generation self-modification observed)

### Demonstrated results
- System discovers mathematical facts: Intersect(Evens, Primes) = {2}, Evens ⊂ Numbers, etc.
- Creates ~50 new units from 51 seed units in 500 cycles
- Credit assignment stratifies the heuristic pool: useful heuristics rise, junk producers sink
- A mutant heuristic (M-H-RunOnExamples, narrowed result cap) outperforms its parent
- Multiple units killed, avoidance rules created, the system learns from its mistakes

## The three-loop architecture (target state)

The full vision has three loops operating at different timescales:

### Fast loop — Datalog inference
- **Built in pudl** (`pudl query`, `internal/datalog/`).
- Runs continuously as pudl ingests new facts from mu
- Derives IDB from EDB: dependency closures, invariant violations, susceptibility pattern matches
- Reactive and deterministic
- Timescale: seconds to minutes

### Medium loop — nous agenda
- **This is what exists today.**
- Runs periodically over the current state (currently in-memory units; eventually Datalog IDB)
- Fires heuristics against high-worth agenda items
- Produces candidate concepts, rules, type refinements
- Generative and exploratory
- Timescale: minutes to hours

### Slow loop — deliberative validation
- **Not yet built.** nous writes conjectures back to pudl, but there is no review/promotion gate.
- Candidates from nous enter human review
- Validated candidates get promoted: new pudl types, new Datalog rules, new conventions
- Human judgment gates what accretes into the stable knowledge base
- Timescale: hours to days

### Why three loops matter

Lenat's EURISKO suffered from heuristic quality degradation — self-modification had no governor, so meta-heuristics couldn't distinguish useful mutations from lucky ones. The three-loop architecture addresses this:

- The fast loop provides **ground truth** (what's actually happening in the system)
- The medium loop provides **generative exploration** (what patterns exist, what might work)
- The slow loop provides **quality control** (human judgment gates promotion)

nous proposes. Humans validate. pudl stabilizes. Each layer does exactly what it's suited for.

## Rules vs Heuristics

Rules and heuristics are both stored as CUE values in the bitemporal store, both enter through the slow validation loop, but they are categorically different:

### Rules

Rules are **deterministic inference machinery**. They derive facts from other facts with no uncertainty. Given the EDB, a rule either fires or it doesn't. Evaluated by the Datalog engine.

```cue
transitiveDep: #Rule & {
    head: { rel: "depends_transitive", args: { from: "$X", to: "$Z" } }
    body: [
        { rel: "depends",            args: { from: "$X", to: "$Y" } },
        { rel: "depends_transitive", args: { from: "$Y", to: "$Z" } },
    ]
}
```

Output: IDB facts. No weight, no threshold, no context-sensitivity.

### Heuristics

Heuristics are **context-sensitive, worth-weighted suggestions**. They don't derive facts — they propose agenda candidates. They have a worth score that determines whether they're tried. They can fail to produce anything useful and that's fine — it affects their worth. Evaluated by the nous engine.

Output: agenda candidates (new units, tasks, rule candidates, type refinements).

### The boundary

| | Rules | Heuristics |
|---|---|---|
| Output | Derived facts (IDB) | Agenda candidates |
| Firing | Deterministic | Worth-weighted, context-sensitive |
| Representation | `#Rule` CUE value | `#Heuristic` CUE value with stack program |
| Evaluated by | Datalog engine | nous interpreter |
| Live in | pudl catalog / bitemporal store | pudl catalog / bitemporal store |

The nous engine reads IDB (produced by rules) and fires heuristics against it. Heuristic output, when validated, can include new rules — but a heuristic is never itself a rule.

## Heuristic representation in CUE

Heuristics have two parts with different relationships to CUE:

**Structure** (metadata, preconditions, output shape, worth) — fully CUE-native, validated at definition time:

```cue
#Heuristic: {
    name:  string
    worth: number & >=0 & <=1
    produces: "agenda-item" | "rule-candidate" | "type-candidate" | "fact"

    // Preconditions are IDB patterns — same #Literal as rule body atoms
    preconditions: [...#Literal]

    // Stack program — typed but semantically opaque to CUE
    program: [...#StackOp]
}
```

Preconditions are the same `#Literal` type as Datalog rule body atoms — CUE validates them. This means a heuristic's "when should I fire" condition is checked at the schema level.

**Execution logic** (the stack program) — CUE-contained but Go-interpreted:

```cue
#StackOp: {
    op:  "push" | "pop" | "match" | "score" | "emit" | "branch" | ...
    arg?: _
}
```

CUE validates the shape of each operation. The Go interpreter executes them. This is the existing nous DSL interpreter with a CUE wrapper.

### DSL vocabulary as CUE-defined words

New domain words are CUE values with stack effect declarations:

```cue
#StackWord: {
    name:   string
    domain: string
    worth:  number & >=0 & <=1
    effect: #StackEffect
    impl:   #Primitive | #Composite
}

#StackEffect: {
    consumes: [...string]
    produces: [...string]
}

#Primitive: { kind: "primitive", handler: string }
#Composite: { kind: "composite", program: [...#StackOp] }
```

- **Kernel words** — primitive, Go-implemented, domain-agnostic, stable
- **Domain words** — composite, CUE-defined, introduced as the system encounters new domains, worth-tracked
- **Meta-heuristics** — watch for recurring compositions in high-worth programs, propose new composite words

With CUE custom functions registered via the Go API, composite words can eventually become CUE expressions calling Go builtins — making stack effect validation native and enabling meta-heuristics to generate CUE rather than bytecode. This is the long-term evolution, not an immediate requirement.

### Migration path from current implementation

1. Keep the Go interpreter as-is for execution (it's the runtime dispatch layer)
2. Move heuristic definitions from Go strings to CUE files with `#Heuristic` schema
3. Preconditions become `#Literal` patterns (validated by CUE, matched by Datalog engine)
4. Stack programs become `[...#StackOp]` (validated structurally by CUE, executed by Go)
5. Word definitions move to CUE with `#StackWord` schema and `#StackEffect` declarations
6. Eventually: composite words as CUE expressions calling Go-registered builtins

Steps 1-4 are straightforward refactoring. Step 5 is a real extension. Step 6 changes the mutation story — the mutator would operate on CUE values rather than token streams.

## Operating modes

### Mode 1: Tight simulation loop

The current mode. nous runs operations on in-memory data, checks results immediately, adjusts worth in the same cycle.

**Properties:**
- Feedback is instant — run operation, evaluate, update credit
- The world is entirely inside the unit store
- Operations are pure functions: input → output
- The agenda runs continuously until budget exhausted
- Evaluation is algorithmic (equality, subset, structural properties)

**Requirements from a domain:**
- Operations computable in-process
- Results evaluable without external observation
- Concept space rich enough for composition and specialization

**Current domain:** Math/set-theory. Future candidates: graph structures, formal grammars, workflow state machines, configuration spaces, code transforms. See `docs/domain-notes/` for detailed analysis.

### Mode 2: Accumulate-then-reason

The target mode for integration with pudl and mu. nous reasons over a growing corpus of structured event data ingested over time.

**Properties:**
- Feedback is delayed — events arrive over days/weeks
- The world is external; nous sees what pudl has ingested
- Operations are queries and correlations, not transforms
- The agenda runs periodically after new data arrives
- Evaluation is statistical (correlations, frequencies, prediction accuracy)

**Requirements from a domain:**
- A stream of structured events with typed slots
- pudl ingesting and cataloging with schema inference
- Enough event volume for pattern detection
- Observable quality attributes

**The integration:** pudl ingests → Datalog derives → nous reasons → nous proposes → human validates → pudl stabilizes as types.

### Hybrid possibility

A domain could use both modes:
1. Mode 1 explores structural possibilities in simulation
2. Mode 2 grounds those explorations against real-world observations
3. Conjectures from Mode 1 are tested against Mode 2 data
4. Mode 2 patterns inform Mode 1 heuristics

This is the ACUTE loop from the original design doc (nousdesign.md).

## The Datalog layer

The fast loop is a Datalog evaluator that uses CUE as its surface syntax. CUE represents facts, rules, and queries — the Go engine evaluates them. CUE never executes anything; it structures the data the evaluator reads.

### CUE as Datalog syntax

See `chat2.md` for the full design discussion. Key decisions:

**Facts are CUE values validated by pudl schemas.** A relation is a `#Definition`, instances conform to it. pudl already does this — the EDB is the pudl catalog.

**Rules are `#Rule` CUE values** with head/body structure:

```cue
#Rule & {
    head: { rel: "depends_transitive", args: { from: "$X", to: "$Z" } }
    body: [
        { rel: "depends",            args: { from: "$X", to: "$Y" } },
        { rel: "depends_transitive", args: { from: "$Y", to: "$Z" } },
    ]
}
```

**Variables use $-prefix convention** (`$X`, `$Y`). The evaluator identifies variables by prefix. CUE validates the structure; the Go engine does unification.

**Queries are relation names with optional field constraints.** `_` (CUE top) is a wildcard.

**Results are CUE values** — one per solution, with variables bound to ground terms. Results are themselves validatable by pudl schemas.

### Rule storage and scoping

Rules follow pudl's existing workspace pattern — global and repo-scoped, with repo shadowing global:

**Global rules** (`~/.pudl/rules/`) — organizational knowledge that applies everywhere. Dependency analysis, risk detection, general conventions.

**Repo-scoped rules** (`.pudl/rules/`) — project-specific knowledge. "In this repo, packages under `internal/` should not import from `cmd/`." Repo rules shadow global rules with the same name.

The evaluator loads both layers, repo-scoped first (first-found-wins, same as pudl's schema and definition shadowing).

### Query interface

Three ways to query, all feeding the same evaluator:

**By name (common case).** Rules are already in the catalog. Just ask for a derived relation:

```
pudl query depends_transitive --from api
pudl query at_risk
```

The evaluator loads all stored rules (global + repo), evaluates to fixed point, filters IDB for the named relation.

**From file (ad-hoc analysis).** Load ephemeral rules in addition to stored rules:

```
pudl query -f my-analysis.cue
```

File rules are additive and session-local — they don't get ingested.

**REPL (interactive exploration).**

```
pudl query -i

> at_risk
[{service: "api", dependency: "db"}, ...]

> :load my-analysis.cue
loaded 3 rules

> my_new_relation --service auth
[...]
```

The REPL maintains a session with stored rules loaded plus any `:load`ed session rules.

### Workspace scoping for queries

Follows pudl's existing patterns:

```
pudl query at_risk                  # repo rules + repo data (default)
pudl query at_risk --all-workspaces # global rules + all data
```

### Connection to nous

nous generates candidate `#Rule` CUE values through heuristic firing. These enter the slow validation loop:

1. nous discovers a pattern (e.g., "services importing untested dependencies tend to have incidents")
2. nous emits a `#Rule` value expressing this pattern
3. The rule lands in the repo-scoped rules directory as a candidate
4. Human review validates or rejects
5. If validated, the rule is active — the fast loop evaluates it on every query
6. If the rule proves generally useful, a human promotes it to global

Repo-scoped = candidate/local knowledge. Global = validated/universal knowledge. pudl's workspace mechanism implements the slow loop's promotion pathway without new infrastructure.

### Temporal queries (future)

The bitemporal extension adds `validAt` and `asOf` fields to queries:

```cue
#TemporalQuery & {
    rel:     "depends_transitive"
    args:    { from: "api", to: _ }
    validAt: "2026-01-01T00:00:00Z"
    asOf:    "2026-02-01T00:00:00Z"
}
```

The evaluator constructs a temporal EDB snapshot from pudl's catalog, then evaluates rules over that snapshot. This enables "what did we believe then about what was true then" queries — essential for failure post-mortems and drift analysis.

## Agent observations — the EDB ingestion path

Agents encounter real patterns, problems, and opportunities as they work. Without a way to record these, that knowledge evaporates when the conversation ends. `pudl observe` is the low-friction interface for agents (and humans) to register structured observations that become EDB for the Datalog layer and raw material for nous.

### The interface

```
pudl observe "auth package has circular dependency with user package" \
    --kind obstacle \
    --scope pkg/auth,pkg/user

pudl observe "all database calls go through a single connection pool" \
    --kind pattern

pudl observe "error handling in API layer is inconsistent" \
    --kind antipattern \
    --scope cmd/api

pudl observe "the Config struct has 47 fields, should be split" \
    --kind suggestion \
    --scope internal/config
```

Repo-scoped by default (writes to the workspace catalog). `--global` flag for cross-repo observations. Each observation gets a timestamp, source identity (agent name or "human"), and optional scope (files, packages, modules it pertains to).

### The schema

```cue
#Observation: {
    _pudl: {
        schema_type:    "nous/core.#Observation"
        identity_fields: ["hash"]
        tracked_fields:  ["status", "worth"]
    }
    hash:        string   // content hash for dedup
    kind:        "fact" | "obstacle" | "pattern" | "antipattern" |
                 "suggestion" | "bug" | "opportunity"
    description: string
    scope:       [...string]  // file paths, package names, module names
    source:      string       // agent name or "human"
    timestamp:   string
    status:      "raw" | "reviewed" | "promoted" | "rejected" | *"raw"
    worth:       number | *0.5
    promotedTo?: string       // if promoted, what rule/convention it became
}
```

The `kind` taxonomy is deliberately small — seven categories. Agents shouldn't choose from 30 options; that's where vocabulary inflation kills signal. One kind, one sentence description, optional scope. That's the whole interface.

### Deduplication

If three agents independently observe "circular dependency in auth," that's three observations, not one. The count is signal — corroboration. But the *same* agent doesn't double-register the *same* observation — pudl's existing content-hash dedup handles this per-source.

### Worth on observations

Observations start at worth 0.5. Worth changes over time:
- **Corroboration**: multiple agents independently flag the same thing → worth goes up
- **Contradiction**: evidence or human review disagrees → worth goes down
- **Promotion**: human confirms and promotes to a rule/convention → worth goes to 1.0
- **Rejection**: human explicitly rejects → worth goes to 0
- **Decay**: observations that are never corroborated, promoted, or acted on slowly lose worth

Observations below a threshold get auto-archived, preventing unbounded growth.

### Workspace scoping

Follows pudl's existing workspace pattern:

- **Repo-scoped** (default): observations about *this* repo. Stored in the workspace catalog. Agents working in a repo see only that repo's observations.
- **Global** (`--global`): observations about cross-cutting concerns. Stored in the global catalog. Visible from all workspaces.

```
pudl observe "this repo has no integration tests" --kind obstacle
pudl observe "all Go repos should use golangci-lint" --kind suggestion --global
```

### The promotion pipeline

1. **Agents write** raw observations via `pudl observe` (or MCP tool, or Claude Code hook)
2. **Observations accumulate** in the repo-scoped (or global) catalog as EDB
3. **Datalog rules** surface patterns across observations: "3 agents independently flagged circular dependencies in pkg/auth"
4. **nous** (medium loop) reasons over accumulated observations — finds clusters, proposes generalizations, generates candidate rules
5. **Human reviews** and promotes: `pudl promote <observation-id>` converts an observation (or cluster) to a Datalog rule or convention
6. **Promoted rules** become active in the fast loop — the Datalog evaluator derives facts from them on every query

### Agent integration

The CLI is the right interface — not a library, not an API. Agents already shell out to tools.

**Claude Code / agentic tools**: `pudl observe` is a tool-use call. An MCP tool definition would be:

```json
{
    "name": "pudl_observe",
    "description": "Record an observation about the codebase",
    "parameters": {
        "description": { "type": "string" },
        "kind": { "enum": ["fact", "obstacle", "pattern", "antipattern", "suggestion", "bug", "opportunity"] },
        "scope": { "type": "array", "items": { "type": "string" } }
    }
}
```

**Claude Code hooks**: a post-task hook could prompt the agent to register observations before the conversation ends.

**Batch ingestion**: `pudl observe -f observations.json` for importing structured observation data from other tools or logs.

### What this enables for nous

Observations are the bridge between Mode 1 and Mode 2. In Mode 2, nous doesn't run operations and check results — it reads accumulated observations and finds patterns:

- "Agents working on pkg/auth consistently report obstacles related to circular dependencies" → propose a rule: `circular_dep_risk(pkg/auth)`
- "Every repo where agents flagged 'no integration tests' also had agents flagging 'inconsistent error handling'" → propose a conjecture: lack of integration tests correlates with inconsistent error handling
- "Agents in 5 repos independently suggested 'split the Config struct'" → propose promoting this to a global convention

The observations are grist for the nous mill. The Datalog layer derives facts from them. nous finds patterns in those derived facts. Humans validate. pudl stabilizes.

## The accretion mechanism

How the symbolic knowledge layer grows:

1. **Instance accumulation** — mu produces facts, agents register observations, pudl ingests and catalogs them
2. **Inference** — Datalog derives higher-order facts (IDB) from base facts (EDB)
3. **Pattern discovery** — nous heuristics fire against IDB, find patterns, form conjectures
4. **Candidate generation** — nous produces candidate concepts, susceptibility patterns, type refinements
5. **Validation** — human review confirms or refutes candidates
6. **Promotion** — validated patterns become pudl types, Datalog rules, architectural invariants
7. **Continuous checking** — promoted invariants are enforced by the fast loop going forward

Agent observations (via `pudl observe`) feed step 1. The graveyard + HindSight mechanism already implements steps 3-4 for failure patterns within the nous engine. The jump to the full architecture is wiring this to the observation stream and adding the Datalog and validation layers.

## Connection to failure analysis

The chat.md conversation identifies failure analysis as a key use case. The ontological structure of failures maps to what nous already does:

| Failure concept | nous equivalent |
|----------------|-----------------|
| Failure event | Unit killed (enters graveyard) |
| Failure mode | GraveRecord type (isA, slots at death) |
| Failure mechanism | Creditor chain (which heuristic made what) |
| Latent condition | Pattern that led to creation of the doomed unit |
| Susceptibility pattern | HAvoid rule (prevents recurrence) |

FMEA-style analysis is essentially what HindSight does: trace the provenance of a failed unit, identify the mechanism, create an avoidance rule. The full vision extends this to real system failures ingested by pudl.

## Bitemporal storage — scaling and implementation

The bitemporal fact store holds agent observations, mu action results, Datalog rules, and heuristic metadata — all with full temporal provenance (valid time + transaction time). This section addresses how pudl scales to handle this.

### Expected load

| Source | Volume | Notes |
|--------|--------|-------|
| Agent observations | 500-2K/day per repo | 10 agents × 5-20 obs × 10 sessions |
| mu action results | ~10K/day | Active build system |
| Cross-repo (50 repos) | ~100K/day peak | High end estimate |
| Datalog rules | Dozens to low hundreds | Grows slowly via promotion |

This is not big data. SQLite handles millions of rows comfortably.

### Storage architecture

**Keep SQLite.** Embedded, zero-ops, handles the expected scale. pudl already depends on it. No need for Postgres or a separate service.

**One database per workspace, one global database.** Matches pudl's existing model:
- Repo-scoped facts in `.pudl/facts.db`
- Global facts in `~/.pudl/facts.db`
- Cross-workspace queries use SQLite's `ATTACH DATABASE` to query multiple databases natively

**Schema:**

```sql
CREATE TABLE facts (
    id          TEXT PRIMARY KEY,
    relation    TEXT NOT NULL,
    args        TEXT NOT NULL,      -- JSON
    valid_start INTEGER NOT NULL,   -- unix timestamp
    valid_end   INTEGER,            -- null = still valid
    tx_start    INTEGER NOT NULL,   -- when asserted
    tx_end      INTEGER,            -- null = not retracted
    source      TEXT,               -- agent name, "human", "nous", "mu"
    provenance  TEXT                -- JSON: agent, activity, context
);

CREATE INDEX idx_facts_relation ON facts(relation);
CREATE INDEX idx_facts_valid ON facts(relation, valid_start, valid_end);
CREATE INDEX idx_facts_tx ON facts(tx_start, tx_end);
```

Four canonical temporal query modes:
- `AsOfNow()` — current valid time, current transaction time
- `AsOfValid(t)` — what was true at time t (current knowledge)
- `AsOfTransaction(t)` — what we believed at time t (regardless of valid time)
- `AsOf(validT, txT)` — what we believed at txT about what was true at validT

### Datalog evaluation performance

Semi-naive bottom-up evaluation iterates the EDB until fixed point. For N rules and M facts, worst case is O(N × M²) for joins. With 100K facts and 50 rules, evaluation runs in seconds.

**Lazy IDB — don't materialize by default.** Evaluate rules on query. Cache hot IDB tables in memory with invalidation on EDB writes. For commonly-queried derived relations (transitive deps, at-risk services), consider materialized views that rebuild incrementally.

**Why not materialize everything:** Transitive closure of a dependency graph with 10K nodes produces O(N²) edges — potentially 100M rows. Selective materialization with invalidation is the right approach.

### Scaling concerns and mitigations

**Temporal accumulation.** Bitemporal stores never delete — retracted facts get a `tx_end` timestamp but stay in the table. Over years, the table grows monotonically. This is by design (full audit trail).

*Mitigation:* Compaction for old history. Facts older than N months that have been superseded (tx_end is set) can be moved to a cold archive table. The main table stays hot. Temporal queries older than N months hit the archive. Optional — can be added later when needed.

**Cross-workspace queries.** `--all-workspaces` loads facts from every repo's catalog. 50 repos × 100K facts = 5M facts in one evaluation.

*Mitigation:* SQLite `ATTACH DATABASE` handles this natively. Per-relation indexes keep scans tight. For very large cross-workspace queries, the evaluator can filter by relation before loading — most queries only touch 2-3 relations.

**Index bloat.** Four-column temporal indexes on a table with millions of rows.

*Mitigation:* Partition by relation name if needed — each relation gets its own table. Keeps index scans tight and makes per-relation temporal queries fast. This is a later optimization, not needed initially.

### The honest assessment

At the scale we're talking about, SQLite is fine for years. The scaling concern is premature unless the system is ingesting millions of observations per day or materializing huge transitive closures. The thing to get right now is the schema and temporal query patterns. The storage engine can be swapped later — the Datalog evaluator doesn't care whether facts come from SQLite, Postgres, or parquet. It just needs a relation iterator.

## Package structure

```
cmd/nous/main.go              — CLI entry point (Mode 1 + Mode 2)
internal/
  unit/                        — Unit, Store (property-list model)
  agenda/                      — Priority queue with merge
  dsl/                         — Stack-based interpreter
    vm.go                      — Core interpreter
    builtins.go                — ~30 general builtins
    builtins_math.go           — ~30 math/set builtins
    token.go                   — Tokenizer
    value.go                   — Value type (nil, bool, int, float, string, list)
  engine/                      — Main loop, heuristic firing
    engine.go                  — Run, WorkOnTask, WorkOnUnit
    fire.go                    — FireRule (Interp2 equivalent)
    credit.go                  — Credit assignment, HindSight, graveyard
    mutation.go                — Heuristic mutation integration
  mutate/                      — Token-level mutation engine
    mutate.go                  — 7 mutation operators + validation
  pudlbridge/                  — Mode 2: connection to pudl
    bridge.go                  — Read observations, write discoveries, query Datalog
  seed/                        — Domain loaders
    registry.go                — Domain registry (-domain flag)
    math.go                    — Math/set-theory domain (Mode 1)
    heuristics.go              — 11 seed heuristics (Mode 1)
    observations.go            — Observation domain + 5 heuristics (Mode 2)
```

## What's built

### pudl integration (built 2026-04-05/06)

The Datalog evaluator and bitemporal fact store now exist in pudl. See pudl's `docs/facts.md` and `docs/datalog.md`.

Mode 2 is operational: nous reads observations from pudl, reasons with Mode 2 heuristics, and writes discoveries back. See `docs/mode2.md` for usage and details.

**pudl side:**
- Bitemporal fact store with four temporal query modes
- Datalog evaluator (semi-naive, hash-indexed) with CUE rule loader
- `pudl observe`, `pudl facts`, `pudl query`, `pudl rule add` commands
- Public API packages (`pkg/factstore`, `pkg/eval`) for external consumers

**nous side:**
- `internal/pudlbridge` — reads observations as units, writes discoveries as facts
- `internal/seed/observations.go` — Mode 2 domain (4 types, 5 heuristics)
- `-pudl DIR` flag on `nous run` to enable Mode 2

## Open design issues

### Rule domains / namespacing

Rules currently have no domain concept. All rules in `~/.pudl/schema/pudl/rules/` and `.pudl/schema/pudl/rules/` are loaded and evaluated together on every `pudl query` call. As rules accumulate across different analysis domains (dependency analysis, code quality, failure analysis), this becomes a problem — irrelevant rules fire against unrelated data, and there's no way for a nous domain to pull only its own rule set.

Options under consideration:

1. **Domain field on `#Rule`** — add `domain?: string` to the schema. The evaluator filters by domain before evaluation. Requires tagging every rule.
2. **Directory-based namespacing** — `rules/deps/`, `rules/quality/`, `rules/failures/`. The evaluator loads from a specific subdirectory. Convention-based, no schema change. Mirrors how pudl already organizes schemas by package directory.
3. **Query-time filtering** — `pudl query --domain deps depends_transitive`. The domain maps to a directory or a filter.

Leaning toward option 2 (directory structure) with option 3 as the CLI surface. No decision made yet.

## What's next

See `docs/domain-notes/` for detailed analysis of candidate Mode 1 domains.

**Near-term:**
- Human validation gate (slow loop): review nous conjectures before promotion to Datalog rules
- Load Datalog-derived facts as units (bridge.LoadDerived is built but not wired into main)
- More Mode 2 heuristics: temporal correlation, causal inference, prediction accuracy
- Adapt mutation vocabulary for Mode 2 heuristic programs

**Medium-term:**
- mu integration (actions go through mu, results re-ingested by pudl)
- Mode 1 units backed by pudl catalog (hybrid Mode 1+2)
- Another Mode 1 domain (graphs, grammars, workflows)

**Long-term:**
- CUE-native RLL (see below)
- Failure analysis as the first real Mode 2 domain
- Human-agent boundary heuristics (Phase 6 from nousdesign.md)
- LLM-backed heuristics as mu plugins (Phase 7)

## CUE as RLL: the long-term architecture

The current implementation (v1) uses CUE for schema validation and rule syntax, with Go doing all reasoning. Heuristics are Go strings containing stack DSL programs — opaque to CUE, not inspectable by meta-heuristics, mutated only at the token level. This works but limits meta-reasoning to token manipulation rather than structural analysis.

The long-term goal (v2) is to make CUE the native representation language — an analog of Lenat's RLL (Representation Language Language) — where knowledge, heuristics, and meta-knowledge are all CUE values that the system can inspect, unify, and reason about structurally.

### Why CUE is viable as RLL

RLL was a meta-level substrate: a language for describing knowledge representations, represented in itself. CUE has structural parallels:

- **RLL's slot inheritance ↔ CUE's type lattice.** Units inherit from generalizations in RLL; definitions inherit constraints via embedding in CUE. Both form a lattice with computable meet and join.
- **RLL's self-describing units ↔ CUE's self-describing values.** A CUE value carries its constraints. Two values can be unified and the system reports compatibility. Analogous to RLL's inspectable slot facets.
- **RLL's monotonic accretion ↔ CUE's monotonic constraints.** CUE only adds constraints, never removes them. Knowledge grows, matching the accretion model.

Where CUE falls short of RLL on its own: no Turing-completeness (no loops, no recursion, no side effects), no triggers (if-added, if-removed), and all-or-nothing unification (no partial matches or soft constraints).

### Three mechanisms that close the gap

**1. Custom op functions (the `op` package pattern).**

Go functions registered as CUE operations give CUE values computational capability. A heuristic precondition can call `op.#Match` for pattern matching against the knowledge base. A then-part can call `op.#Emit` to produce agenda items. The heuristic remains a CUE value — inspectable, unifiable, meta-reasonable — but has access to computational primitives.

This mirrors RLL's `if-needed` and `if-added` facets, which were Lisp functions attached to slots. They were inspectable as data (Lisp code is data) but executable as computation. CUE values with op calls have the same dual nature.

**2. The Go API as the meta-level.**

CUE can't generate new CUE at runtime the way Lisp generates new code. But Go can — it can construct CUE source strings and compile them. The meta-level (heuristics that modify heuristics) goes through Go generating CUE. Less elegant than Lisp's homoiconicity but functionally equivalent. The Go wrapper also handles agenda management, credit assignment, and the parts of evaluation that require state and side effects.

**3. CUE's function pattern for heuristics.**

CUE definitions framed as input → output transforms:

```cue
#FindHotspots: {
    // Metadata — inspectable by meta-heuristics
    name:     "find-hotspots"
    worth:    0.7
    produces: "scope-hotspot"
    creditors: [...string]

    // Input schema — what the engine provides
    input: {
        observations: [...#Observation]
        threshold: int | *2
    }

    // Output schema — what the heuristic produces
    output: {
        hotspots: [...{
            scope: string
            count: int
        }]
    }

    // Computation — CUE for declarative parts, op calls for the rest
    _grouped: op.#GroupBy & { data: input.observations, key: "scope" }
    output: hotspots: op.#Filter & {
        data: _grouped.result
        pred: { count: >= input.threshold }
    }
}
```

This maps directly to RLL's heuristic structure:

| RLL | CUE-native RLL |
|-----|----------------|
| Relevance conditions ("when should I fire?") | Input schema + preconditions (CUE constraints) |
| Action body ("what do I do?") | Function body with op calls |
| Output spec ("what do I produce?") | Output schema (CUE definition) |
| Meta-slots (worth, creditors, english) | Regular CUE fields alongside the function |

### What meta-reasoning this enables

With heuristics as CUE values, meta-heuristics can reason structurally:

- **Subsumption checking:** "This heuristic's input schema subsumes that one's — it's strictly more general." (CUE computes this natively.)
- **Redundancy detection:** "These two heuristics have identical preconditions but different outputs — one may be redundant."
- **Precondition analysis:** "This heuristic requires observations with `kind=obstacle` — it's irrelevant if there are no obstacles in the knowledge base." (Checkable by unifying the input schema against the current EDB.)
- **Structural mutation:** Instead of token-level DSL mutation, mutate the CUE structure itself — swap op calls, widen input constraints, narrow output schemas. The mutation operates on meaningful structure, not opaque tokens.
- **Composition:** "These two heuristics are often fired in sequence — propose a composite heuristic whose input is the first's input and whose output is the second's output."

None of this is possible when heuristics are opaque Go string programs. The CUE representation makes the internal structure of heuristics available to the system's own reasoning.

### Relationship to the current implementation

The current system (v1) is not thrown away. It's the working prototype that validates the engine mechanics — agenda, credit assignment, HindSight, mutation, the two-level control loop. The CUE-native RLL (v2) replaces *how heuristics are represented and inspected*, not the engine that fires them.

The migration path from v1 to v2:

1. **Current state:** Heuristics are Go strings with stack DSL programs. CUE validates `#Rule` and `#Observation` schemas. Go does everything.
2. **Intermediate:** Heuristic definitions move to CUE files. Preconditions become CUE patterns matched by the Datalog engine. Stack programs remain as opaque `[...#StackOp]` lists. Meta-reasoning limited to precondition analysis.
3. **Target:** Heuristics are CUE input→output functions with op calls. The stack DSL is retired. Meta-heuristics reason over CUE structure. Mutation operates on CUE values. Go is the evaluator and meta-level, CUE is the representation.

Step 2 is the current migration path described in "Heuristic representation in CUE" above. Step 3 is the full RLL vision. The op package and Go API are what make step 3 viable rather than aspirational.
