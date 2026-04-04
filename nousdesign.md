# nous: A EURISKO-Style Discovery System on pudl + mu

## Overview

This document describes a system for open-ended exploration of software architecture and human-agent hybrid design spaces, built on the existing pudl/mu infrastructure. The architecture is directly inspired by Doug Lenat's EURISKO (1978-1983), which explored mathematical concept spaces using self-modifying heuristic rules.

The key insight is that the pudl/mu split already embodies the separation EURISKO needed but never cleanly achieved: a **knowledge layer** (pudl: schema inference, drift detection, catalog) and an **execution layer** (mu: hermetic actions, content-addressed caching, plugin protocol). What's missing is the **reasoning layer** between them --- the agenda-driven heuristic interpreter that decides what to explore, evaluates results, and learns from failures.

We call this middle layer **nous** (Greek: mind, intellect).

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

## How EURISKO's Architecture Maps to pudl/mu

### The Unit/Slot System → pudl Catalog + CUE Schemas

In EURISKO, everything is a "unit" --- a named entity with typed property slots stored on Interlisp property lists. Units include mathematical concepts, heuristic rules, slot definitions, and meta-concepts. A unit is identified by having a `Worth` property.

In nous, **units are pudl catalog entries** with CUE-validated schemas. The ISA hierarchy becomes CUE's type lattice via embedding. Inverse slot maintenance becomes CUE constraint propagation. The `Worth` gate becomes a CUE constraint: `worth: int & >=0 & <=1000`.

```cue
// schema/nous/core.cue

#Unit: {
    _pudl: {
        schema_type: "nous/core.#Unit"
        identity_fields: ["name"]
        tracked_fields: ["worth", "isA", "generalizations", "specializations"]
    }
    name:              string
    worth:             int & >=0 & <=1000
    isA:               [...string]
    generalizations:   [...string]
    specializations:   [...string]
    english?:          string
    abbrev?:           string
    creditors:         [...string]  // which heuristics created this
    applics:           [...#ApplicRecord]
    ...  // open: domain-specific slots allowed
}

#ApplicRecord: {
    taskNum:    int
    results:    [...string]
    creditUsed: int
    timestamp:  string
    reason?:    string
}
```

### The Agenda → Priority Queue with pudl-backed Task Persistence

EURISKO's agenda is a priority-ordered list of tasks, each of the form `(priority unit-name slot-name reasons supplementary-info)`. Tasks are merged when duplicates are proposed, with priority boosting.

```cue
// schema/nous/task.cue

#Task: {
    _pudl: {
        schema_type: "nous/core.#Task"
        identity_fields: ["unitName", "slotName"]
        tracked_fields: ["priority", "reasons"]
    }
    priority:  int & >=0 & <=1000
    unitName:  string
    slotName:  string
    reasons:   [...string]
    supplementary: {
        slotToChange?: string
        creditTo?:     [...string]
        ...
    }
}
```

In Go, the runtime agenda is a `container/heap` priority queue with a secondary index `map[taskKey]*Task` for O(1) merge lookups. On each cycle, the agenda state is checkpointed to pudl for persistence and auditability.

### Heuristic Rules → CUE-Defined Units with Expression Trees

EURISKO stores heuristic conditions and actions as Lisp lambdas on property lists. In nous, we define a `#RuleExpr` discriminated union in CUE and store heuristic logic as structured expression trees that a Go interpreter evaluates.

```cue
// schema/nous/heuristic.cue

#Heuristic: #Unit & {
    isA: [...string] & list.Contains("Heuristic")

    // Condition slots (IfParts)
    ifPotentiallyRelevant?: #RuleExpr
    ifTrulyRelevant?:       #RuleExpr
    ifWorkingOnTask?:       #RuleExpr
    ifFinishedWorkingOnTask?: #RuleExpr

    // Action slots (ThenParts)
    thenCompute?:              #RuleExpr
    thenAddToAgenda?:          #RuleExpr
    thenDefineNewConcepts?:    #RuleExpr
    thenDeleteOldConcepts?:    #RuleExpr
    thenPrintToUser?:          #RuleExpr
    thenConjecture?:           #RuleExpr

    // Performance tracking (updated by the engine)
    overallRecord?: { successes: int, failures: int, totalTime: int }
    rarity?:        { ratio: number, successes: int, failures: int }
}

// The expression language for rule conditions and actions.
// This is the equivalent of EURISKO's Lisp lambdas stored on property lists.
#RuleExpr: #And | #Or | #Not | #CheckSlot | #IsAKindOf | #HasHighWorth |
           #RunAlg | #Compare | #ForEach | #SetVar | #GetVar |
           #CreateUnit | #KillUnit | #UnionProp | #AddTask |
           #Print | #Literal | #SlotRef | #UnitRef | #Call

#And:         { op: "and",  args: [...#RuleExpr] }
#Or:          { op: "or",   args: [...#RuleExpr] }
#Not:         { op: "not",  arg:  #RuleExpr }
#CheckSlot:   { op: "checkSlot",  unit: #RuleExpr, slot: string, pred: #RuleExpr }
#IsAKindOf:   { op: "isAKindOf",  unit: #RuleExpr, category: string }
#HasHighWorth:{ op: "hasHighWorth", unit: #RuleExpr, threshold: int | *800 }
#Compare:     { op: "compare", left: #RuleExpr, right: #RuleExpr, cmp: "eq"|"gt"|"lt"|"gte"|"lte" }
#RunAlg:      { op: "runAlg",  fn: string, args: [...#RuleExpr] }
#ForEach:     { op: "forEach", collection: #RuleExpr, var: string, body: #RuleExpr }
#SetVar:      { op: "setVar",  name: string, value: #RuleExpr }
#GetVar:      { op: "getVar",  name: string }
#CreateUnit:  { op: "createUnit", name: #RuleExpr, parent?: string, slots: { [string]: _ } }
#KillUnit:    { op: "killUnit",   name: #RuleExpr }
#UnionProp:   { op: "unionProp",  unit: #RuleExpr, slot: string, value: #RuleExpr }
#AddTask:     { op: "addTask",    priority: #RuleExpr, unit: #RuleExpr, slot: string, reasons: [...string] }
#Print:       { op: "print",      verbosity: int, parts: [...#RuleExpr] }
#Literal:     { op: "literal",    value: _ }
#SlotRef:     { op: "slotRef",    slot: string }
#UnitRef:     { op: "unitRef",    name: string }
#Call:        { op: "call",       fn: string, args: [...#RuleExpr] }
```

### The Rule Interpreter → Go Expression Evaluator

EURISKO's `Interp2` walks the IfParts sub-slots, evaluates each lambda, and if all pass, walks the ThenParts. In nous:

```go
// internal/nous/interpreter.go

type Interpreter struct {
    store   *UnitStore      // backed by pudl catalog
    agenda  *Agenda         // priority queue
    env     *Environment    // variable bindings for current rule execution
    verbosity int
}

// Eval dispatches on the op field of a RuleExpr (parsed from CUE).
func (interp *Interpreter) Eval(expr RuleExpr) (Value, error) {
    switch expr.Op {
    case "and":
        for _, arg := range expr.Args {
            v, err := interp.Eval(arg)
            if err != nil || !v.Truthy() { return Nil, err }
        }
        return True, nil
    case "checkSlot":
        unit, _ := interp.Eval(expr.Unit)
        val := interp.store.GetSlot(unit.AsString(), expr.Slot)
        return interp.Eval(expr.Pred.WithBinding("_", val))
    case "isAKindOf":
        unit, _ := interp.Eval(expr.Unit)
        return Bool(interp.store.IsAKindOf(unit.AsString(), expr.Category)), nil
    case "createUnit":
        name, _ := interp.Eval(expr.Name)
        u := interp.store.CreateUnit(name.AsString(), expr.Parent, expr.Slots)
        return UnitVal(u), nil
    case "addTask":
        pri, _ := interp.Eval(expr.Priority)
        unit, _ := interp.Eval(expr.Unit)
        interp.agenda.Add(pri.AsInt(), unit.AsString(), expr.Slot, expr.Reasons)
        return True, nil
    // ... ~20 more cases
    }
}

// FireRule is the equivalent of EURISKO's Interp2.
func (interp *Interpreter) FireRule(rule string, arg string) bool {
    h := interp.store.GetUnit(rule)

    // Check all IfParts
    for _, ifSlot := range interp.store.SubSlots("IfParts") {
        expr := h.GetSlotExpr(ifSlot)
        if expr == nil { continue } // vacuous truth
        interp.env.Set("ArgU", arg)
        v, _ := interp.Eval(expr)
        if !v.Truthy() { return false }
    }

    // Execute all ThenParts
    for _, thenSlot := range interp.store.SubSlots("ThenParts") {
        expr := h.GetSlotExpr(thenSlot)
        if expr == nil { continue }
        interp.env.Set("ArgU", arg)
        interp.Eval(expr)
    }
    return true
}
```

### The Main Loop

```go
// internal/nous/engine.go

func (e *Engine) Run(ctx context.Context) error {
    e.Initialize()

    for {
        select {
        case <-ctx.Done():
            return nil
        default:
        }

        // Level 1: Agenda-driven
        if task, ok := e.agenda.Pop(); ok {
            e.WorkOnTask(task)
            continue
        }

        // Level 2: Unit-focused (when agenda is empty)
        unit := e.HighestWorthUnfocused()
        if unit == "" {
            if !e.PromptContinue() { return nil }
            e.ResetFocusSet()
            continue
        }
        e.WorkOnUnit(unit)
        e.MarkFocused(unit)
    }
}

func (e *Engine) WorkOnTask(task Task) {
    e.taskNum++

    // Set context for rule evaluation
    e.interp.env.Set("CurUnit", task.UnitName)
    e.interp.env.Set("CurSlot", task.SlotName)
    e.interp.env.Set("CurPri", task.Priority)

    // Try every heuristic against this task
    for _, h := range e.store.Examples("Heuristic") {
        for _, ifSlot := range e.store.SubSlots("IfTaskParts") {
            expr := e.store.GetUnit(h).GetSlotExpr(ifSlot)
            if expr == nil { continue }

            e.interp.env.Set("ArgU", task.UnitName)
            v, _ := e.interp.Eval(expr)
            if !v.Truthy() { continue }

            // Condition matched --- execute ThenParts
            result := v.AsString()
            if result == "AbortTask" {
                return // HAvoid rule killed this task
            }
            e.interp.ExecuteThenParts(h, task.UnitName)
        }
    }
}
```

### The Credit Assignment System

```go
// internal/nous/credit.go

// When a unit is killed, trace back to its creators and penalize them.
func (e *Engine) PunishCreators(unit string) {
    for _, creditor := range e.store.GetSlot(unit, "creditors").AsStrings() {
        worth := e.store.GetSlot(creditor, "worth").AsInt()
        e.store.SetSlot(creditor, "worth", worth / 2)
        e.log("Halved Worth of %s to %d (created the failed unit %s)", creditor, worth/2, unit)
    }
}

// Track success/failure ratio for every operation.
func (e *Engine) TrackRarity(op string, succeeded bool) {
    r := e.store.GetSlot(op, "rarity")
    if succeeded {
        r.Successes++
    } else {
        r.Failures++
    }
    r.Ratio = float64(r.Successes) / float64(r.Successes + r.Failures)
    e.store.SetSlot(op, "rarity", r)
}
```

## Applying It to Software Architecture

### Domain Ontology

The seed knowledge base defines the concept space for software system design:

```cue
// schema/nous/architecture.cue

#ServiceComponent: #Unit & {
    isA: [...string] & list.Contains("ServiceComponent")
    latencyP99?:   int     // milliseconds
    throughput?:    int     // requests/second
    replicas?:     int & >=1
    dependencies:  [...string]
    failureModes:  [...#FailureMode]
    costPerMonth?: number
}

#ArchitecturalPattern: #Unit & {
    isA: [...string] & list.Contains("ArchitecturalPattern")
    domain:  [...string]  // what kinds of components it applies to
    range:   [...string]  // what it produces
    preconditions: #RuleExpr
    transform:     #RuleExpr  // how it modifies the architecture
}

#AgentRole: #Unit & {
    isA: [...string] & list.Contains("AgentRole")
    capabilities:     [...string]
    latencyMs?:       int
    errorRate?:       number & >=0 & <=1
    costPerAction?:   number
    requiresHuman?:   bool | *false
    escalationPath?:  string
}

#InteractionPattern: #Unit & {
    isA: [...string] & list.Contains("InteractionPattern")
    participants: [...string]  // AgentRole or HumanRole names
    protocol:     #RuleExpr
    failureMode:  string
}

#DesignCandidate: #Unit & {
    isA: [...string] & list.Contains("DesignCandidate")
    components:   [...string]
    interactions: [...string]
    qualityScores: {
        latency?:        int
        throughput?:      int
        availability?:    number
        cost?:            number
        explainability?:  number
        reversibility?:   number
    }
}

#FailureMode: {
    name:        string
    probability: number
    impact:      string  // "degraded" | "unavailable" | "data_loss"
    mitigation?: string
}
```

### Seed Heuristics for Architecture

Here are examples of heuristics expressed in the CUE rule language. These are the architectural equivalents of EURISKO's H1-H29:

```cue
// schema/nous/seed_heuristics/h_decompose.cue

// Analogous to EURISKO's H1: "If an operation has mostly bad results,
// specialize it." Here: "If a component has high latency but only some
// endpoints are slow, split it."

h_decompose: #Heuristic & {
    name: "H-Decompose"
    worth: 700
    isA: ["Heuristic", "ArchitectureHeuristic"]
    english: """
        IF a service component has latency exceeding the target,
        AND analysis shows that only a subset of its endpoints are slow,
        THEN propose splitting it into fast-path and slow-path sub-services.
        """

    ifWorkingOnTask: {
        op: "and"
        args: [
            { op: "isAKindOf", unit: { op: "getVar", name: "CurUnit" }, category: "ServiceComponent" },
            { op: "compare"
              left:  { op: "slotRef", slot: "latencyP99" }
              right: { op: "literal", value: 200 }
              cmp:   "gt"
            },
        ]
    }

    thenDefineNewConcepts: {
        op: "and"
        args: [
            { op: "createUnit"
              name: { op: "call", fn: "packName", args: [
                  { op: "literal", value: "FastPath" },
                  { op: "getVar", name: "CurUnit" },
              ]}
              parent: "ServiceComponent"
              slots: {
                  creditors: ["H-Decompose"]
                  generalizations: [{ op: "getVar", name: "CurUnit" }]
              }
            },
            { op: "createUnit"
              name: { op: "call", fn: "packName", args: [
                  { op: "literal", value: "SlowPath" },
                  { op: "getVar", name: "CurUnit" },
              ]}
              parent: "ServiceComponent"
              slots: {
                  creditors: ["H-Decompose"]
                  generalizations: [{ op: "getVar", name: "CurUnit" }]
              }
            },
        ]
    }

    thenAddToAgenda: {
        op: "addTask"
        priority: { op: "call", fn: "averageWorths", args: [
            { op: "getVar", name: "CurUnit" },
            { op: "literal", value: "H-Decompose" },
        ]}
        unit: { op: "getVar", name: "NewUnit" }
        slot: "latencyP99"
        reasons: ["Component had high latency; decomposition may isolate the slow path"]
    }
}
```

```cue
// schema/nous/seed_heuristics/h_delegate.cue

// Analogous to H11: "Find applications by running the algorithm on domain
// examples." Here: "If a human role is a bottleneck, try delegating a
// subset of tasks to an agent."

h_delegate: #Heuristic & {
    name: "H-Delegate"
    worth: 700
    isA: ["Heuristic", "AgentHeuristic"]
    english: """
        IF a human role has high latency or high cost,
        AND there exists an agent capability that covers a subset of their tasks,
        THEN propose a delegation with human review for the covered subset.
        """

    ifWorkingOnTask: {
        op: "and"
        args: [
            { op: "isAKindOf", unit: { op: "getVar", name: "CurUnit" }, category: "HumanRole" },
            { op: "or", args: [
                { op: "compare", left: { op: "slotRef", slot: "latencyMs" }, right: { op: "literal", value: 60000 }, cmp: "gt" },
                { op: "compare", left: { op: "slotRef", slot: "costPerAction" }, right: { op: "literal", value: 10.0 }, cmp: "gt" },
            ]},
        ]
    }

    thenCompute: {
        op: "forEach"
        collection: { op: "call", fn: "examples", args: [{ op: "literal", value: "AgentRole" }] }
        var: "candidateAgent"
        body: {
            op: "and"
            args: [
                // Check if the agent's capabilities overlap with the human's
                { op: "call", fn: "capabilityOverlap", args: [
                    { op: "getVar", name: "CurUnit" },
                    { op: "getVar", name: "candidateAgent" },
                ]},
                // Create a new interaction pattern: agent does it, human reviews
                { op: "createUnit"
                  name: { op: "call", fn: "packName", args: [
                      { op: "literal", value: "DelegateWith" },
                      { op: "getVar", name: "candidateAgent" },
                  ]}
                  parent: "InteractionPattern"
                  slots: {
                      participants: [
                          { op: "getVar", name: "candidateAgent" },
                          { op: "getVar", name: "CurUnit" },
                      ]
                      creditors: ["H-Delegate"]
                  }
                },
            ]
        }
    }
}
```

```cue
// schema/nous/seed_heuristics/h_hindsight_arch.cue

// Analogous to H12: "When a concept dies, learn what went wrong and
// create an avoidance rule." Here: "When a design candidate is rejected,
// trace back and create a rule preventing that class of mistake."

h_hindsight_arch: #Heuristic & {
    name: "H-HindSight-Arch"
    worth: 800
    isA: ["Heuristic", "HindSightRule"]
    english: """
        IF a design candidate has just been killed (worth dropped to zero),
        THEN examine its creditors and the transformation that produced it,
        AND create a new avoidance heuristic that prevents the same class
        of architectural mistake.
        """

    ifPotentiallyRelevant: {
        op: "call"
        fn: "memberOf"
        args: [
            { op: "getVar", name: "ArgU" },
            { op: "call", fn: "getDeletedUnits", args: [] },
        ]
    }

    thenDefineNewConcepts: {
        op: "and"
        args: [
            // Trace the provenance: who created this, what slot was changed, what was the change
            { op: "setVar", name: "failedTransform",
              value: { op: "call", fn: "traceProvenance", args: [{ op: "getVar", name: "ArgU" }] }
            },
            // Create a new HAvoid-style rule
            { op: "createUnit"
              name: { op: "call", fn: "packName", args: [
                  { op: "literal", value: "HAvoid-Arch" },
                  { op: "getVar", name: "ArgU" },
              ]}
              parent: "H-Avoid-Arch"  // template avoidance heuristic
              slots: {
                  creditors: ["H-HindSight-Arch"]
                  // The avoidance condition is derived from the failed transformation
                  // (analogous to EURISKO's SUBPAIR of CSlot/GSlot/CSlotSibs into the HAvoid template)
              }
            },
        ]
    }
}
```

### Concrete Example: Exploring a Chat Service Design

Here's a walkthrough of how nous would explore a design space, step by step:

**Seed state:** A monolithic chat service (`ChatMonolith`) with a human operator role (`HumanOps`), imported into pudl as catalog entries.

```cue
chatMonolith: #ServiceComponent & {
    name: "ChatMonolith"
    worth: 500
    isA: ["ServiceComponent", "Anything"]
    latencyP99: 450   // too high
    throughput: 1000
    replicas: 1
    dependencies: ["PostgresDB", "RedisCache"]
    failureModes: [{name: "DB connection exhaustion", probability: 0.02, impact: "unavailable"}]
}

humanOps: #HumanRole & {
    name: "HumanOps"
    worth: 500
    isA: ["HumanRole", "Anything"]
    capabilities: ["deploy", "rollback", "scaleUp", "triageAlert", "approveChange"]
    latencyMs: 300000  // 5 minutes average response
    errorRate: 0.05
    costPerAction: 25.0
}
```

**Cycle 1 --- Agenda empty, focus on highest-Worth unit (ChatMonolith):**

WorkOnUnit tries every heuristic. **H-Decompose** fires: latencyP99 (450ms) exceeds 200ms. It creates `FastPathChatMonolith` and `SlowPathChatMonolith` as new units, posts a task to find their latency characteristics.

**Cycle 2 --- Agenda has the latency task. WorkOnTask:**

**H11-equivalent** fires: runs a simulation (mu action) to estimate latency for the decomposed services. Results come back: FastPath 80ms, SlowPath 800ms. These are stored as slot values.

**H-FailureMode** fires: enumerates single-point failures of the decomposed design. Finds that SlowPath has no fallback. Posts a task to add resilience.

**Cycle 3 --- Focus shifts to HumanOps (also high Worth):**

**H-Delegate** fires: HumanOps has latencyMs 300000 and costPerAction 25.0. It finds `AlertTriageAgent` in the AgentRole examples, which covers `triageAlert`. Creates `DelegateWithAlertTriageAgent` interaction pattern. Posts task to evaluate the delegation.

**Cycle 4 --- Evaluate the delegation:**

mu runs a simulation of the agent handling alerts. 90% of alerts are handled correctly. The 10% that fail are escalated to HumanOps. The interaction pattern gets a quality score.

**H-TrustCalibration** fires: acceptance rate is >85%, proposes creating a variant with reduced review requirements. Creates `ReducedReviewAlertTriage` unit.

**Cycle N --- A bad design gets killed:**

Suppose an earlier cycle created `SynchronousCrossServiceCall` as a component, and it violated the latency constraint. Its Worth drops to zero, triggering deletion.

**H-HindSight-Arch** fires: traces provenance --- `SynchronousCrossServiceCall` was created by `H-PatternApply` changing the interaction slot from async to sync. Creates `HAvoid-Arch-SyncCrossService` with the rule: "Don't change interaction slots from async to sync when the latency budget is under 200ms."

This avoidance heuristic now participates in all future cycles. If H-PatternApply ever tries the same transformation again, the HAvoid rule's `ifWorkingOnTask` returns `AbortTask`, preventing the mistake.


## Integration with pudl and mu

### Reading State from pudl

```go
// internal/nous/pudl_bridge.go

// LoadUnits reads all nous-typed entries from the pudl catalog.
func (b *PudlBridge) LoadUnits() ([]*Unit, error) {
    // pudl query: all catalog entries whose schema matches nous/*
    entries, err := b.pudl.Query("schema_type LIKE 'nous/%'")
    if err != nil { return nil, err }

    var units []*Unit
    for _, entry := range entries {
        u, err := unitFromCatalogEntry(entry)
        if err != nil { continue } // skip malformed entries
        units = append(units, u)
    }
    return units, nil
}
```

### Emitting Actions to mu

When a heuristic's ThenCompute needs to run a simulation or execute an external tool, it emits a mu action spec:

```go
// internal/nous/mu_bridge.go

// EmitAction creates a mu action spec and writes it for mu to execute.
func (b *MuBridge) EmitAction(action Action) (string, error) {
    spec := mu.ActionSpec{
        Command:  action.Command,
        Inputs:   action.InputArtifacts,
        Outputs:  action.ExpectedOutputs,
        Env:      action.Environment,
        Network:  action.NeedsNetwork,
    }

    // Write to the mu actions directory; mu picks it up
    hash, err := b.mu.Submit(spec)
    if err != nil { return "", err }

    // Wait for completion, re-ingest results into pudl
    result, err := b.mu.WaitForResult(hash)
    if err != nil { return "", err }

    b.pudl.Ingest(result.OutputPath)
    return result.ArtifactHash, nil
}
```

### The Drift-Driven Feedback Loop

pudl's existing drift detection closes the EURISKO feedback loop:

1. nous proposes a design change (via mu action)
2. mu executes it
3. `pudl import` re-ingests the resulting system state
4. `pudl drift` compares the new state against the design candidate's declared expectations
5. If drift is detected (actual != expected), nous receives the drift report as a new task
6. Heuristics fire on the drift report --- possibly HindSight rules if the drift represents a failure

```
nous: "Deploy FastPathChat with target latency <100ms"
  → mu: executes deployment action
  → pudl: re-imports actual metrics after deployment
  → pudl drift: "latencyP99 declared 80, actual 150 --- DRIFTED"
  → nous: drift task enters agenda, heuristics evaluate
  → H-HindSight: "The decomposition was too coarse; learn to check
                   dependency latency before splitting"
```

### Heuristics as mu Plugins (Optional Extension)

For maximum extensibility, heuristics could also be mu plugins:

```json
{
    "name": "llm-heuristic",
    "command": "nous-llm-plugin",
    "protocol": "ndjson",
    "capabilities": ["ifRelevant", "thenPropose"]
}
```

An LLM-backed heuristic plugin would receive the current task context as JSON, use Claude or another model to reason about the architecture, and return proposed actions. The credit assignment system would track its performance just like any other heuristic --- if it produces garbage, its Worth drops and it fires less often.


## The Generalization/Specialization Engine

The mutation operators that allow nous to create new heuristics and modify existing ones. These operate on `#RuleExpr` trees.

```go
// internal/nous/mutate.go

// GeneralizeExpr walks a RuleExpr tree and randomly replaces unit
// references with their generalizations from the ISA hierarchy.
func (m *Mutator) GeneralizeExpr(expr RuleExpr) RuleExpr {
    switch expr.Op {
    case "unitRef":
        genls := m.store.Generalizations(expr.Name)
        if len(genls) > 0 && m.rng.Float64() < 0.5 {
            choice := genls[m.rng.Intn(len(genls))]
            m.diff = append(m.diff, Diff{From: expr.Name, To: choice})
            return RuleExpr{Op: "unitRef", Name: choice}
        }
        return expr
    case "literal":
        if n, ok := expr.Value.(int); ok && m.rng.Float64() < 0.3 {
            // Generalize: widen numeric threshold
            newN := n + m.rng.Intn(n/2+1)
            m.diff = append(m.diff, Diff{From: n, To: newN})
            return RuleExpr{Op: "literal", Value: newN}
        }
        return expr
    default:
        // Recursively walk children
        return m.walkChildren(expr, m.GeneralizeExpr)
    }
}

// SpecializeExpr does the inverse: replaces with specializations,
// tightens numeric thresholds.
func (m *Mutator) SpecializeExpr(expr RuleExpr) RuleExpr {
    // ... symmetric to GeneralizeExpr but uses Specializations
    // and narrows numeric values
}
```

CUE validates every mutated expression tree against `#RuleExpr` before it's accepted. Invalid mutations are silently rejected --- a safety mechanism EURISKO lacked.


## Implementation Roadmap

### Phase 1: Core Engine (estimated: 2-3 weeks)

**Goal:** A working agenda loop that can load units from CUE files, fire hand-written heuristics, and print results.

- [ ] Define CUE schemas: `#Unit`, `#Heuristic`, `#RuleExpr`, `#Task`
- [ ] Implement `UnitStore` in Go (backed by `map[string]*Unit`, loaded from CUE)
- [ ] Implement the `#RuleExpr` interpreter (the ~20-case Eval switch)
- [ ] Implement the agenda (priority queue + merge logic)
- [ ] Implement the main loop (`Run`, `WorkOnTask`, `WorkOnUnit`)
- [ ] Write 3-5 seed heuristics in CUE (H-FindExamples, H-Specialize, H-KillDuplicates)
- [ ] Test with a small mathematical concept space (Sets, Lists, operations) to validate the engine works independently of the architecture domain

### Phase 2: Architecture Domain (estimated: 2-3 weeks)

**Goal:** Seed the system with software architecture concepts and domain-specific heuristics.

- [ ] Define architecture ontology in CUE (`#ServiceComponent`, `#ArchitecturalPattern`, `#AgentRole`, `#InteractionPattern`, `#DesignCandidate`)
- [ ] Write architecture-specific heuristics (H-Decompose, H-Delegate, H-PatternApply, H-FailureMode)
- [ ] Implement quality evaluation functions (latency estimation, cost calculation, availability modeling)
- [ ] Write HindSight rules for the architecture domain
- [ ] Test with a seed architecture and verify that the system explores reasonable designs

### Phase 3: pudl Integration (estimated: 1-2 weeks)

**Goal:** Units live in the pudl catalog; nous reads/writes through pudl.

- [ ] Register nous CUE schemas in pudl's schema directory
- [ ] Implement `PudlBridge`: load units from catalog, write back new/modified units
- [ ] Checkpoint agenda state to pudl after each cycle
- [ ] Hook into pudl's drift detection for the feedback loop

### Phase 4: mu Integration (estimated: 1-2 weeks)

**Goal:** Heuristic actions that require external execution go through mu.

- [ ] Implement `MuBridge`: emit action specs, wait for results, re-ingest
- [ ] Create mu plugins for architecture simulations (latency estimation, failure injection, cost modeling)
- [ ] Wire `mu observe` convergence checks into the evaluation function

### Phase 5: Self-Modification (estimated: 2-3 weeks)

**Goal:** The system can generalize, specialize, and create new heuristics.

- [ ] Implement the `#RuleExpr` mutation engine (GeneralizeExpr, SpecializeExpr)
- [ ] Implement credit assignment (track Creditors, Applics, Rarity)
- [ ] Implement HindSight rules that generate HAvoid-style avoidance heuristics from failures
- [ ] Implement the H1 analog: "If a heuristic mostly produces bad results, specialize it"
- [ ] CUE validation of all mutated expressions before acceptance
- [ ] Test self-modification with the math domain first, then architecture

### Phase 6: Human-Agent Boundary (estimated: 2-3 weeks)

**Goal:** Heuristics that reason about where to draw the line between human and agent work.

- [ ] Define `#HumanRole` and `#AgentRole` with capability/cost/reliability slots
- [ ] Write H-Delegate, H-TrustCalibration, H-EscalationLearning heuristics
- [ ] Implement performance tracking for agent roles (analogous to Rarity)
- [ ] Test with a hybrid operations scenario (alert triage, deployment approval, incident response)

### Phase 7: LLM-Backed Heuristics (optional, estimated: 1-2 weeks)

**Goal:** Use language models as heuristic plugins for semantically-guided mutation.

- [ ] Implement a mu plugin that wraps an LLM API
- [ ] Define the context format: current task, relevant units, ISA neighborhood
- [ ] LLM proposes mutations; credit assignment tracks quality
- [ ] Compare LLM-guided exploration vs. random mutation


## Open Questions

1. **Expression language richness vs. interpretability.** The `#RuleExpr` language defined above is minimal. EURISKO's lambdas had the full power of Lisp. How rich does the expression language need to be before interesting heuristics can be expressed? Is there a useful sweet spot short of Turing-completeness?

2. **Evaluation functions.** EURISKO had human judgment as the primary evaluator for many domains. For software architecture, we need computable quality metrics. How good do the simulation/estimation functions need to be? Can we bootstrap from simple heuristic estimates and refine as the system learns?

3. **Scale of the concept space.** EURISKO started with ~250 units. A realistic software architecture ontology could have thousands. Does the "try every heuristic against every unit" loop scale, or do we need indexing/filtering beyond `IfPotentiallyRelevant`?

4. **Persistence model.** Should nous maintain its own in-memory state with periodic pudl checkpoints, or should every slot write go through pudl immediately? The former is faster; the latter gives better auditability and crash recovery.

5. **Interactive vs. autonomous operation.** EURISKO required an attentive human operator. Should nous default to autonomous operation with human review of synthesized heuristics, or should it pause and ask for confirmation before major mutations?
