# Build / Dependency Graph Optimization Domain

## Motivation

An agent managing a build system or codebase faces a compositional optimization problem: modules depend on modules, builds form DAGs, caching strategies have computable tradeoffs. This ties directly into mu, which is already a content-addressed DAG executor. nous could explore the space of build graph structures and discover which organizations minimize rebuild scope, maximize cache hit rates, and reduce build times.

## Why this fits Mode 1

- Build graphs are DAGs with computable properties (critical path, parallelism, cache key stability)
- Operations on build graphs are well-defined: split a target, merge targets, add/remove edges, introduce interface boundaries
- Effects are measurable: rebuild scope when a node changes, parallelism factor, total vs incremental build time
- mu already has the machinery — action DAGs, content-addressed caching, topological execution

## Concept space

### Core types

- **Target** — a build target (module, package, library, binary). Has source files, produces artifacts.
- **DepEdge** — a directed dependency between targets.
- **BuildGraph** — a DAG of targets and edges. This is the primary data object.
- **CacheConfig** — caching strategy for a target (content-addressed, timestamp, none).
- **GraphOp** — a transform on build graphs.

### Data representation

A BuildGraph as a unit:
```
name: "MonolithBuild"
data: {
    targets: ["core", "auth", "api", "db", "cli", "tests"],
    edges: [
        ["api", "core"], ["api", "auth"], ["api", "db"],
        ["auth", "core"], ["auth", "db"],
        ["cli", "api"],
        ["tests", "api"], ["tests", "auth"], ["tests", "core"],
    ],
    sizes: {"core": 50, "auth": 30, "api": 80, "db": 40, "cli": 20, "tests": 60},
}
invariants: {
    targetCount: 6,
    edgeCount: 9,
    depth: 3,
    criticalPath: ["cli", "api", "core"],
    criticalPathLength: 150,
    maxParallelism: 3,
    avgFanOut: 1.5,
    rebuildScopeOnChange: {"core": 5, "auth": 3, "api": 2, "db": 3, "cli": 1, "tests": 1},
}
```

### Operations

- **SplitTarget** — divide a target into two, distributing its sources and dependencies. Reduces per-target build time, may increase edge count.
- **MergeTargets** — combine two targets into one. Reduces edge count, increases per-target build time.
- **IntroduceInterface** — insert a thin interface target between two targets. Increases target count, reduces rebuild scope (changes behind the interface don't propagate).
- **RemoveTransitive** — remove edges that are transitively implied (A→B→C means A→C is redundant). Simplifies the graph without changing semantics.
- **Parallelize** — restructure to increase maximum parallelism (split sequential chains into independent branches).
- **AddCacheLayer** — mark a target as a cache boundary (its output is content-addressed, downstream only rebuilds if the output hash changes).
- **InlineTarget** — remove a target by folding its sources into its dependents. Opposite of IntroduceInterface.

### Quality attributes

- `totalBuildTime` — sum of all target sizes on the critical path (sequential worst case)
- `parallelBuildTime` — critical path length (with unlimited parallelism)
- `incrementalBuildTime(changed)` — how much rebuilds when a specific target changes
- `avgRebuildScope` — average number of targets affected when any single target changes
- `maxRebuildScope` — worst case rebuild scope
- `cacheEfficiency` — fraction of targets that can be cache-hit on a typical change
- `edgeDensity` — edges / (targets × (targets-1)/2) — how interconnected
- `depth` — longest dependency chain
- `parallelismFactor` — maxParallelism / targetCount

### What the system could discover

- **Interface insertion points:** The system discovers that inserting an interface target between `core` and its dependents reduces avgRebuildScope by 40% — changes to `core`'s implementation don't propagate past the interface.

- **Optimal granularity:** Too few targets = no parallelism, too many = edge overhead. The system finds the sweet spot for a given graph shape.

- **Merge equivalences:** Merging `auth` and `db` produces the same build time as the original if they're always built together anyway (no parallelism between them). The system discovers when merging is free.

- **Cache boundary placement:** Adding cache layers at high-fan-in nodes (many dependents) is always better than at leaf nodes. The system discovers this general rule from specific examples.

- **Transitive reduction:** Removing redundant edges never changes build semantics but simplifies the graph. The system discovers that some edge-removal operations are always safe.

### DSL builtins needed

- `build-graph-new` — create from targets and edges
- `build-graph-split`, `build-graph-merge`, `build-graph-add-interface`
- `build-graph-remove-transitive` — transitive reduction
- `build-graph-critical-path`, `build-graph-depth`
- `build-graph-rebuild-scope` — given a changed target, what rebuilds?
- `build-graph-parallelism` — maximum parallel width
- `build-graph-total-time`, `build-graph-parallel-time`
- `build-graph-add-cache-boundary`

### Connection to mu

This domain could eventually operate on real mu build graphs:
- Import a mu.json project as a BuildGraph unit
- Explore transforms
- Export optimized configurations back to mu
- pudl drift detection catches when the real build graph diverges from the optimized version

### Implementation complexity

**Low to moderate.** DAG operations are straightforward. Critical path is a topological sort with weights. Rebuild scope is a reverse reachability query. The main data structure is an adjacency list with node weights — simpler than the graph-theory domain since we don't need chromatic numbers or planarity testing.

### Seed content

- 2-3 example build graphs: a small monolith (6 targets), a microservice set (4 independent services), a deep chain (5 targets in sequence)
- Basic operations: Split, Merge, IntroduceInterface, RemoveTransitive
- Known optimizations as seed conjectures: "transitive edges are always removable," "cache boundaries at high-fan-in nodes reduce rebuild scope"
