# Ecosystem Report: defn, pudl, mu, nous

**Date:** 2026-04-14

---

## Part 1: defn -- The Reference Implementation

### What It Is

defn is a monorepo that serves as a reference implementation of agent-native platform engineering. It is simultaneously a workspace and a $HOME -- a configuration-as-code system where structural constraints replace instructions and the git repository acts as a self-validating fixed-point computation.

### Current State

defn is mature and actively maintained. Key metrics:

- 60 AI Decision Records (AIDRs) documenting every significant design choice
- 50+ Kubernetes applications managed via ArgoCD
- 14 AWS organizations, 117+ accounts
- 3 k3d local clusters
- 16 chat bots
- 490+ directories with BUILD.bazel files
- Full CUE type system for every directory and file classification

### Core Ideas

**1. Repository as Typed Namespace.** Directories are CUE struct types. Files are typed values. Path conventions are schema constraints. A file in the wrong place is a type error.

**2. Configuration as Lattice.** CUE defines sets of valid values, not individual files. Unification computes the greatest lower bound. Adding information never loosens what was already constrained -- a contradiction produces bottom immediately.

**3. Fixed-Point Computation Loop.**

```
ideas' = migrate(run(deploy(build(config(ideas * world)))))
```

Generation is idempotent: running twice produces identical output or the build fails. The repository is the fixed-point witness.

**4. BRICK Classification.** Every directory is one of: Relationship (manifest), Interface (schema/catalog), Component (concrete artifact), or Kit (composition of blocks).

**5. Catalog-Driven Generation.** One entry per resource instance in a central CUE inventory. Everything else derives -- BUILD.bazel files, Helm configs, kustomizations, mise tasks.

**6. Agent-Native Design.** Structure is precise enough that agents work with minimal context. Tests replace instructions. Agents receive only violated invariants, not prose guidance.

### Recent Development (April 2026)

- 4-tier AWS security hardening across 12/14 organizations (SCPs, RCPs, IAM Access Analyzer)
- Jianghu AWS account vending expansion (batch 3 of 6)
- CloudTrail organization-wide trails across all 14 orgs
- Gen pipeline performance: full validation now runs in 7 seconds

### Relevance to pudl/mu/nous

defn is where these ideas were discovered empirically. The three tools are the extraction and formalization of patterns that emerged from building and operating defn:

- **pudl** extracts the schema inference, catalog management, drift detection, and fact store patterns
- **mu** extracts the hermetic build coordination, plugin protocol, and content-addressed caching
- **nous** extracts the reasoning layer -- the part where you decide what to explore next

defn proves the patterns work at scale. The three tools make them reusable.

---

## Part 2: Current Functionality

### pudl -- The Knowledge Layer

**"What is true?"**

pudl is a CLI tool for building a local, schema-validated data lake with knowledge inference. One binary, a SQLite catalog, and CUE schemas in a git-tracked directory.

**Working functionality:**

| Capability | Status | Key commands |
|---|---|---|
| Multi-format import (JSON/YAML/CSV/NDJSON) | Complete | `pudl import` |
| CUE schema inference with heuristic scoring | Complete | `pudl schema new/add/reinfer` |
| Content-addressed dedup (SHA256 + proquint) | Complete | (automatic) |
| SQLite catalog with full provenance | Complete | `pudl list/show/delete` |
| Resource identity and versioning | Complete | (automatic) |
| CUE definition discovery and dependency graphs | Complete | `pudl definition list/show/validate/graph` |
| Bitemporal fact store | Complete | `pudl facts list/show/retract/invalidate` |
| Datalog evaluator (semi-naive, hash-indexed) | Complete | `pudl query`, `pudl rule add` |
| Drift detection (declared vs observed state) | Complete | `pudl drift check/report` |
| Observation recording | Complete | `pudl observe --source X --scope repo:path` |
| Per-repo workspace support | Complete | `pudl repo init`, `.pudl/workspace.cue` |
| ACUTE convergence feedback loop | Complete | `pudl export-actions`, `pudl ingest-observe/manifest`, `pudl status` |
| Agent onboarding | Complete | `pudl prime` |
| Public API for external consumers | Complete | `pkg/factstore`, `pkg/eval` |
| mu bridge (action export, manifest/observe ingestion) | Complete | `pudl export-actions`, `pudl ingest-*` |
| Fixed-point verification | Complete | `pudl verify` |

**Scale:** ~54 CLI commands, ~28 internal packages, comprehensive test suite, 21 documentation files, 62 implementation logs.

### mu -- The Execution Layer

**"How do we test hypotheses?"**

mu is a language-agnostic, plugin-driven build coordinator. No built-in semantics -- plugins fill it with meaning. ~7,500 lines of Go.

**Working functionality:**

| Capability | Status | Key commands |
|---|---|---|
| Config loading with preprocessor support | Complete | (automatic, CUE/TOML/YAML) |
| Plugin lifecycle (discover/plan/observe) | Complete | `mu plugin list/add` |
| NDJSON wire protocol | Complete | (internal) |
| DAG construction, topological sort, cycle detection | Complete | `mu build --plan` |
| Parallel execution with worker pool | Complete | `mu build --jobs N` |
| Content-addressed storage (OCI layout) | Complete | `mu cache inspect` |
| Sandbox execution (isolated PATH/TMPDIR) | Complete | (automatic) |
| Toolchain scratch builds | Complete | `mu scratch` |
| Build manifest output | Complete | `mu build --emit-manifest` |
| Drift observation | Complete | `mu observe` |
| Sealed inputs for secret injection | Complete | `resolve_secret` protocol |
| CAS integrity verification | Complete | `mu verify --fix` |
| BRICK metadata passthrough | Complete | kind/implements fields |

**Plugins (11 + shell built-in):** go, docker, k8s, terraform, file, zig, scratch, lint, pass, host, aws, cowsay, shell.

### nous -- The Reasoning Layer

**"What should we explore?"**

nous is a EURISKO-style discovery engine -- an agenda-driven heuristic interpreter. ~3,771 lines of Go.

**Working functionality:**

| Capability | Status | Key commands |
|---|---|---|
| Unit store with typed slots | Complete | (internal) |
| Agenda with duplicate merging | Complete | (internal) |
| Two-level control (task-driven + unit-focused) | Complete | `nous run` |
| Stack-based DSL (40+ builtins) | Complete | (internal) |
| Heuristic firing (if/then pattern matching) | Complete | (internal) |
| Mode 1: Math/set theory domain | Complete | `nous run -domain math` |
| Mode 2: Observation reasoning | Complete | `nous run -domain observations -pudl DIR` |
| pudl bridge (load observations, write facts) | Complete | (internal) |
| Token-level mutation (7 mutation types) | Complete | `-no-mutate` flag to disable |
| Credit assignment and applics tracking | Complete | (internal) |
| Claude Code hook installer | Complete | `nous init` |
| Agent guides | Complete | `nous guide TOPIC` |

---

## Part 3: Planned Functionality

### pudl -- Planned

1. **Observation promotion pipeline** -- Convert validated observations into Datalog rules or conventions
2. **Deeper nous integration** -- Bidirectional communication, candidate rules from nous, human review gate
3. **Cross-source correlation** -- Link AWS resources to Kubernetes resources
4. **Analytics** -- `pudl diff`, `pudl summary`, outlier detection, DuckDB/Parquet
5. **More type patterns** -- Azure, GCP, Terraform, Docker Compose, Helm, CI/CD configs
6. **UI** -- Dashboard/reporting, interactive TUI
7. **Deeper CUE integration** -- Catalog-driven code generation, richer fixed-point properties
8. **Richer mu plugin protocol** -- Action result feedback, richer action types

### mu -- Planned

1. **Tiered cache composition** -- Chain local + OCI backends (read-repair, write-through)
2. **GOCACHEPROG bridge** -- Fine-grained Go build cache integration
3. **OS-level sandboxing** -- Linux: user namespaces + overlayfs; macOS: sandbox-exec
4. **Remote execution** -- Distribute actions to worker pools
5. **Policy plugin** -- OPA/conftest for runtime enforcement
6. **Secret plugins** -- 1Password (`op`) plugin
7. **Midas pattern** -- pudl `stamp` command for component generation
8. **Protocol extensions** -- Streaming progress, async planning

### nous -- Planned

1. **Phase 3 completion** -- Wire `LoadDerived()` to bring Datalog-derived facts into units; incremental observation loading; human validation gate
2. **Phase 4: mu integration** -- Emit action specs from heuristics, wait for results, re-ingest via pudl drift detection
3. **Phase 5: Self-modification** -- Self-modifying heuristics via mutation; HindSight rule generation; worth-based heuristic selection
4. **Phase 6: Human/agent boundary** -- H-Delegate, H-TrustCalibration, H-EscalationLearning heuristics
5. **Phase 7: LLM-backed heuristics** -- LLM as mu plugin for evaluation
6. **Architecture domain** -- Concepts: Module, Service, Interface, DependencyEdge, Pattern; operations: Split, Merge, ExtractInterface
7. **CUE-as-RLL** -- Long-term vision to replace the stack DSL with CUE as the rule language (EURISKO's RLL mapped to CUE schemas)

---

## Part 4: Gaps and Contradictions

### Gap 1: nous-to-pudl Write-Back Path

**Status:** Partially implemented.

`pudlbridge.WriteFact()` exists and calls `factstore.AddFact()`, but the observation promotion pipeline (where nous discoveries become pudl rules or conventions) is not implemented. nous can write conjectures back as observations, but there is no mechanism for a nous-generated insight to become a first-class Datalog rule without manual intervention.

**Impact:** The three-loop architecture (fast Datalog, medium nous, slow human) is incomplete. The medium loop produces conjectures but cannot promote them to rules that feed the fast loop.

### Gap 2: nous-to-mu Integration Does Not Exist

**Status:** Designed, not implemented.

nous Phase 4 describes emitting mu action specs from heuristics, but no code exists for this. The bridge would need to:
- Serialize action specs from heuristic `thenCompute` slots
- Invoke mu (or produce a mu.json fragment)
- Wait for and parse manifest results
- Feed results back through pudl drift detection

**Impact:** nous currently cannot trigger any external execution. All Mode 1 evaluation is in-memory; Mode 2 can only read from and write to pudl. The simulation-based quality evaluation described in the architecture doc is aspirational.

### Gap 3: Incremental Observation Loading

**Status:** Not implemented.

`pudlbridge.LoadObservations()` reloads all facts from pudl on every run. For a small number of observations this is fine, but it does not scale and prevents nous from running as a persistent process or daemon.

**Impact:** Low-priority now, but will become a bottleneck when the observation count grows.

### Gap 4: CLI Inconsistency Between Projects

**Status:** Structural divergence.

- pudl uses cobra (spf13/cobra) with 54 command files
- mu uses a hand-rolled CLI dispatcher (~500 LOC)
- nous uses a minimal hand-rolled CLI

This is not necessarily a bug -- the user prefers minimal Go style and each tool's complexity justifies different choices. But it means there is no shared CLI pattern for future integration commands.

**Impact:** Cosmetic. Users interacting with all three tools will encounter different flag conventions and help output styles.

### Gap 5: Dependency Direction Between nous and pudl

**Status:** Potential concern.

nous imports `pkg/factstore` and `pkg/eval` from pudl. This creates a compile-time dependency: nous depends on pudl. The reverse direction (pudl depending on nous) does not exist. mu depends on neither.

This is the correct dependency direction (knowledge layer is foundational), but it means:
- pudl's public API is load-bearing for nous -- changes to `pkg/factstore` or `pkg/eval` can break nous
- There are no integration tests spanning both repos

**Impact:** Moderate. The public API packages were explicitly created for this purpose, but API stability is not formally guaranteed.

### Gap 6: No End-to-End Integration Test

**Status:** Missing.

There is no automated test that exercises the full loop:
1. `pudl observe` records observations
2. `nous run -domain observations -pudl DIR` reasons over them
3. nous writes conjectures back to pudl
4. `pudl query` retrieves the conjectures

Each tool has its own test suite, but the integration points are tested manually.

**Impact:** High. This is the most likely place for regressions as the tools evolve independently.

### Gap 7: HindSight and Self-Modification Are Designed But Not Wired

**Status:** Code exists, not connected.

The mutation system works (7 mutation types, token-level). Credit assignment tracks applics. But:
- Heuristics are not automatically mutated based on low applics success rates
- HindSight (learning from unit death) has engine support but the avoidance-rule creation is not implemented
- Worth adjustment of heuristics based on downstream impact is not connected

**Impact:** This is the core EURISKO differentiator. Without self-modification, nous is a fixed heuristic interpreter -- interesting but not revolutionary. This is the highest-value planned feature.

### Gap 8: defn Does Not Use pudl/mu/nous

**Status:** Intentional but notable.

defn is the reference implementation that inspired these tools, but it does not depend on or use them. defn has its own gen pipeline (Go binary), its own validation (CUE + Bazel), and its own convergence loop (ArgoCD + Terraform Operator). The three tools are an extraction, not a dependency.

**Impact:** There is no feedback loop where defn validates the tools. The tools could drift from the patterns that defn proved work.

### Contradiction 1: Worth Semantics Across Modes

In Mode 1, worth measures "interestingness" of mathematical concepts (structurally derived). In Mode 2, worth measures "importance" of observations (socially/practically derived). The same 0-1000 scale is used for both, but the heuristics that adjust worth are completely different.

This is not a bug -- EURISKO used worth the same way -- but it means worth values are not comparable across modes. A Mode 2 observation with worth 700 is not "more interesting" than a Mode 1 concept with worth 600; they are different metrics using the same name.

### Contradiction 2: Scope Format

nous uses `scope` as a unit slot (string). pudl uses `--scope repo:path` format for observations. The pudl bridge in nous parses scope from fact args. The format is consistent now (both use `repo:path`), but there is no shared type or validation between the two codebases.

---

## Part 5: Recommended Implementation Order

The following order maximizes value delivery while respecting dependencies. Each step produces a working, testable increment.

### Step 1: End-to-End Integration Test (1-2 days)

**Why first:** Without this, every subsequent change risks silent breakage at the seams.

- Write a test (in nous or a shared test directory) that:
  1. Creates a temp pudl directory
  2. Records 5-10 observations via pudl's API
  3. Runs nous in observation mode against that directory
  4. Verifies conjectures were written back
  5. Queries pudl for the conjectures
- This validates the current pudl bridge and establishes a regression gate for everything below.

### Step 2: Observation Promotion Pipeline (3-5 days)

**Why second:** This completes the medium loop of the three-loop architecture and makes nous immediately useful.

- In pudl: add `pudl promote` or extend `pudl rule add` to accept nous conjectures as candidate rules
- In nous: add a `thenPromote` action slot that writes a candidate rule (CUE) to a staging area
- Define the human review gate: promoted candidates land in a review directory; a human runs `pudl rule add` to accept
- The fast loop (Datalog) can then use rules that originated from nous reasoning

### Step 3: Incremental Observation Loading (1-2 days)

**Why third:** Prerequisite for running nous repeatedly or continuously.

- Track the last-seen transaction timestamp
- On subsequent runs, only load facts with `tx_start > last_seen`
- Merge new units into the existing store rather than rebuilding from scratch

### Step 4: HindSight and Worth-Based Mutation (3-5 days)

**Why fourth:** This is what makes nous more than a fixed rule engine. It is the core EURISKO value proposition.

- Wire the mutation trigger: after N cycles, identify heuristics with high failure rates and mutate them
- Implement avoidance-rule creation from HindSight: when a unit dies, trace its creditors and create an `HAvoid-X` rule
- Connect worth adjustment: downstream success/failure of units should propagate credit to their creating heuristics

### Step 5: pudl Public API Stabilization (1-2 days)

**Why fifth:** nous depends on `pkg/factstore` and `pkg/eval`. Before expanding this dependency surface, establish stability guarantees.

- Add version constraints or interface contracts to the public API
- Ensure the public API has its own test suite that runs in pudl's CI
- Document breaking-change policy

### Step 6: Architecture Domain Seed (5-7 days)

**Why sixth:** This is the first "real" domain beyond math and observations. It proves nous can reason about software systems.

- Define concepts: Module, Service, Interface, DependencyEdge, Pattern
- Define operations: Split, Merge, ExtractInterface, ApplyPattern
- Write seed heuristics: H-FindCoupledModules, H-ExtractSharedInterface, H-DetectCircularDeps
- Use pudl's catalog as input: treat catalog entries as architectural facts
- Validate against a real codebase (e.g., pudl itself)

### Step 7: mu Integration (5-7 days)

**Why seventh:** Requires steps 1-6 to be valuable. Without a real domain and working self-modification, mu integration is premature.

- Define the action spec format for nous heuristic emissions
- Build the bridge: nous `thenCompute` can emit a mu action spec (JSON)
- Implement the execution wait: nous pauses the current task, mu runs, manifest comes back
- Feed manifest results through pudl drift detection
- Drift triggers new nous tasks

### Step 8: Continuous Operation Mode (3-5 days)

**Why eighth:** Once all loops are connected, nous should be able to run as a daemon.

- Watch pudl for new facts (poll or inotify on the SQLite file)
- Maintain a persistent unit store across cycles
- Support graceful shutdown with state serialization
- Optionally: expose a simple API for querying current nous state

### Step 9: defn Integration (timeline varies)

**Why last:** This is the validation step. Deploy pudl/mu/nous against defn and verify the extracted tools work on the system that inspired them.

- Use pudl to ingest defn's catalog and schemas
- Use mu to coordinate defn's gen pipeline (or a subset)
- Use nous to reason about defn's architectural patterns and drift reports
- Compare results against defn's native tooling

---

## Part 6: Summary

The three tools form a clean, well-designed architecture:

```
pudl (knows) <---> nous (thinks) ---> mu (acts)
  ^                                      |
  |                                      |
  +--- results / manifests / drift ------+
```

**pudl** is the most mature: complete data lake with schema inference, Datalog, bitemporal facts, drift detection, and convergence feedback. It is production-grade.

**mu** is the most focused: a minimal build coordinator with a clean plugin protocol. It does exactly one thing well. The 11 plugins give it real-world utility.

**nous** is the most ambitious: a EURISKO-style discovery engine is a fundamentally harder problem. The core engine works, the DSL works, Mode 1 and Mode 2 work. But the self-modification loop (the thing that makes EURISKO EURISKO) is not yet connected.

**defn** is the existence proof: it demonstrates at scale that the patterns these tools extract actually work in production.

The critical path is: integration testing, observation promotion, self-modification, then mu integration. Everything else is valuable but secondary to closing the reasoning loop.
