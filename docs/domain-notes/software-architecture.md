# Software Architecture Domain — Design Notes

## Scope

Exploring software architecture as an abstract design space: how code is structured, how components relate, how boundaries are drawn. Not infrastructure operations (replicas, latency) or team/process concerns — those are too contextual for compositional exploration.

## What makes a domain EURISKO-compatible

The math domain works because:
1. **Operations compose** — you can chain SetIntersect with SetUnion
2. **Results are computable** — run an operation, get concrete data
3. **Quality is measurable** — compare results for equality, subset, size
4. **Generalization/specialization is natural** — SetIntersect-on-Primes is a specialization of SetIntersect

A good architecture domain needs the same properties.

## Proposed concept space

### Core units

- **Module** — a unit of code organization. Has measurable quality attributes as slot values.
- **Service** — a deployed module with a network boundary.
- **Interface** — a contract between modules (API surface, stability, coupling type).
- **DependencyEdge** — a directed relationship between modules (type: data, behavioral, temporal).
- **Pattern** — a reusable structural transform (Facade, Adapter, EventBus, etc.)

### Quality attributes (numeric slots on modules)

- `coupling` — how many other modules this depends on or is depended on by
- `cohesion` — how related the internal elements are (higher is better)
- `changeAmplification` — how many modules must change when this one changes
- `cognitiveLoad` — how hard the module is to understand in isolation
- `apiSurface` — number of public types/functions exposed
- `depthInGraph` — longest path from this module to a leaf
- `fanOut` — number of direct dependencies
- `fanIn` — number of modules that depend on this

### Operations (transforms on modules/structures)

- **Split** — divide a module into two. Reduces coupling, may reduce cohesion, increases coordination.
- **Merge** — combine two modules. Inverse of Split.
- **ExtractInterface** — create an explicit boundary between modules. Increases apiSurface, reduces coupling.
- **InvertDependency** — reverse the direction of a dependency edge. Changes coupling topology.
- **ApplyPattern** — apply a named Pattern to a module or set of modules.
- **IntroduceFacade** — wrap a complex subsystem behind a simpler interface.
- **InsertMediator** — add an intermediary between two tightly coupled modules (event bus, message queue).

Each operation has a `defn` slot with a DSL program that computes the effect on quality attributes.

### Seed patterns

- **Facade** — precondition: high apiSurface. Effect: reduces apiSurface, adds a module.
- **Adapter** — precondition: interface mismatch between two modules. Effect: decouples, adds a module.
- **EventBus** — precondition: many-to-many dependency edges. Effect: replaces direct coupling with indirect.
- **BulkheadIsolation** — precondition: shared failure domain. Effect: splits failure domains, adds resource cost.
- **StranglerFig** — precondition: legacy module with high cognitiveLoad. Effect: incremental replacement path.

### What the system would discover

- **Operation equivalences:** ExtractInterface + InvertDependency = DependencyInjection (a pattern emerges from composing primitives)
- **Specializations:** Split works well on modules with low cohesion but harms high-cohesion modules → specialized Split-LowCohesion
- **Anti-patterns:** Merge after Split is identity (or worse if it introduces coupling that didn't exist before)
- **Tradeoff curves:** Facade reduces apiSurface but increases depthInGraph — there's a sweet spot
- **Compositional patterns:** EventBus + BulkheadIsolation = a resilient decoupled architecture (discovered, not seeded)

## What the data slots look like

A Module's `data` is its metric vector:
```
{coupling: 7, cohesion: 3, apiSurface: 12, cognitiveLoad: 8, fanOut: 5, fanIn: 3}
```

An operation's `defn` computes a new metric vector from the old one:
```
# Split: coupling goes down, fanOut splits, add coordination cost
"mod" @ "data" get-slot "metrics" !
"metrics" @ "coupling" get-field 2 / "newCoupling" !
...
```

Evaluation: a DesignCandidate's worth is a function of its quality vector — lower coupling, higher cohesion, lower cognitiveLoad = higher worth.

## Open questions

1. **How to represent module graphs?** A single module's metrics are easy. But architecture is about the *graph* — the relationships between modules. The DSL would need to reason about graph properties (cycles, longest path, connected components). Do we add graph builtins, or represent graphs as sets of DependencyEdge units?

2. **Concrete vs abstract.** Do we seed a specific system ("a Go monolith with 5 packages") or work purely with abstract modules? Concrete is better for validation but limits generalization.

3. **Composition semantics.** In math, SetIntersect(A, B) takes two sets and returns a set. What does Split(Module) return? Two modules — but they need names, they need to be wired into the dependency graph, the old edges need to be redistributed. This is more complex than set operations.

## Subset analysis

**Good fit for EURISKO-style exploration:**
- Design patterns as composable transforms
- Dependency graph transforms and their metric effects
- API boundary design (interface width, stability, coupling type)

**Poor fit:**
- Team structure (too many human factors)
- SDLC pipelines (procedural, not compositional)
- Security practices (mostly checklists; auth boundaries *could* work as a subset)
