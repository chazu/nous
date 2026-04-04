# Workflow / State Machine Design Domain

## Motivation

Agents orchestrating multi-step processes — deploy pipelines, data processing, incident response, CI/CD workflows — are implicitly designing state machines. States, transitions, guards, error handlers, and rollback paths compose. An EURISKO-style system could explore the space of workflow structures and discover optimal designs: minimal state machines that handle all cases, error recovery patterns that generalize across domains, equivalent workflows that differ in complexity.

This is the automata/formal-grammars domain pointed at something directly useful for agentic coding.

## Why this fits Mode 1

- State machines are finite, compositional, and have computable properties
- Operations on state machines are well-defined: merge states, split states, add/remove transitions, compose machines
- Properties are checkable: reachability, deadlock freedom, liveness, determinism
- Equivalence is decidable: two machines accept the same sequences of events
- The output is directly usable: "this workflow structure handles these cases with fewer states"

## Concept space

### Core types

- **State** — a named state with metadata (is it terminal? is it an error state? does it require human input?).
- **Transition** — an edge between states, labeled with an event/action and optional guard condition.
- **Workflow** — a complete state machine: states, transitions, initial state, terminal states, error states.
- **WorkflowOp** — a transform on workflows.
- **EventType** — a type of event that triggers transitions.
- **Guard** — a boolean condition on a transition (DSL program).

### Data representation

A Workflow as a unit:
```
name: "DeployPipeline"
data: {
    states: ["idle", "building", "testing", "staging", "canary", "production", "failed", "rolledBack"],
    initial: "idle",
    terminal: ["production", "rolledBack"],
    error: ["failed"],
    transitions: [
        {from: "idle",       to: "building",    event: "deploy"},
        {from: "building",   to: "testing",     event: "buildSuccess"},
        {from: "building",   to: "failed",      event: "buildFail"},
        {from: "testing",    to: "staging",     event: "testsPass"},
        {from: "testing",    to: "failed",      event: "testsFail"},
        {from: "staging",    to: "canary",      event: "stagingOK"},
        {from: "staging",    to: "failed",      event: "stagingFail"},
        {from: "canary",     to: "production",  event: "canaryOK"},
        {from: "canary",     to: "failed",      event: "canaryFail"},
        {from: "failed",     to: "rolledBack",  event: "rollback"},
        {from: "failed",     to: "idle",        event: "retry"},
    ],
}
invariants: {
    stateCount: 8,
    transitionCount: 11,
    maxDepth: 6,
    hasDeadlock: false,
    allTerminalReachable: true,
    errorRecoveryPaths: 2,
    deterministic: true,
    linearPathLength: 6,
    branchingFactor: 1.375,
}
```

### Operations

**Structural:**
- **MergeStates** — combine two states into one (if they have identical outgoing transitions). Simplifies the workflow.
- **SplitState** — divide a state into two with a guard condition choosing between them. Adds granularity.
- **AddErrorHandler** — add an error transition from a state to an error state.
- **AddRecoveryPath** — add a transition from an error state to a recovery/retry state.
- **AddRollback** — add rollback transitions from each state back to a safe checkpoint.
- **RemoveState** — remove an unreachable or redundant state and rewire transitions.
- **InsertState** — add a new state between two existing states (e.g., add an approval gate).

**Compositional:**
- **SequentialCompose** — run workflow A, then workflow B (terminal states of A connect to initial of B).
- **ParallelCompose** — run A and B concurrently, join at a sync point.
- **ConditionalCompose** — run A or B based on a guard, join at a merge point.
- **LoopCompose** — repeat a workflow until a condition is met.
- **SubworkflowEmbed** — replace a single state with an entire sub-workflow.

**Optimization:**
- **Minimize** — merge equivalent states (analogous to DFA minimization).
- **RemoveUnreachable** — delete states that can't be reached from the initial state.
- **RemoveDeadEnds** — delete states from which no terminal state is reachable.
- **Determinize** — if nondeterministic (multiple transitions on same event), resolve with priorities or guards.

### Quality attributes

- `stateCount` — fewer is simpler
- `transitionCount` — fewer is simpler
- `maxDepth` — longest path from initial to terminal (pipeline length)
- `branchingFactor` — average outgoing transitions per state
- `deterministic` — is every (state, event) pair unambiguous?
- `deadlockFree` — can we always make progress from any non-terminal state?
- `liveTerminals` — are all terminal states reachable?
- `errorCoverage` — fraction of non-terminal states that have an error transition
- `recoveryCoverage` — fraction of error states that have a recovery path
- `rollbackCoverage` — fraction of states that can roll back to a safe point
- `avgPathLength` — average path length from initial to terminal
- `pathCount` — number of distinct paths through the workflow

### What the system could discover

**Workflow equivalences:**
- A linear pipeline (build → test → stage → deploy) with retry-from-start on failure is equivalent to one with per-stage retry if each stage is idempotent. The system discovers this by comparing reachable terminal states.

**Optimal error handling patterns:**
- Adding error handlers to every state is redundant if errors propagate — a single error handler at the top level catches everything that individual handlers would. The system discovers when per-state error handling is equivalent to global error handling.

**Gate ordering:**
- In a pipeline with both automated tests and human approval, the system discovers that tests-before-approval dominates approval-before-tests: same terminal states, fewer wasted human reviews on code that would fail tests anyway.

**Composition patterns:**
- `Sequential(A, B)` followed by `AddRollback` to the composition is equivalent to `Sequential(A-with-rollback, B-with-rollback)` only if A and B are independent. If B depends on A's output, the rollback semantics differ. The system discovers this distinction.

**Minimal workflow for a capability:**
- Given a set of required properties (must handle build failure, must have staging, must support rollback), the system explores all workflows meeting those constraints and finds the minimal one.

**Anti-patterns:**
- A state that can transition to itself on the same event with no guard (infinite loop). The system discovers this as a deadlock pattern and creates an avoidance rule.

### DSL builtins needed

- `workflow-new` — create from states, transitions, initial, terminal
- `workflow-merge-states`, `workflow-split-state`, `workflow-insert-state`
- `workflow-add-error`, `workflow-add-recovery`, `workflow-add-rollback`
- `workflow-sequential`, `workflow-parallel`, `workflow-conditional`
- `workflow-minimize`, `workflow-remove-unreachable`, `workflow-remove-dead-ends`
- `workflow-reachable?` — can state B be reached from state A?
- `workflow-deadlock-free?` — are there any non-terminal states with no outgoing transitions?
- `workflow-deterministic?`
- `workflow-equivalent?` — do two workflows accept the same event sequences?
- `workflow-error-coverage`, `workflow-recovery-coverage`
- `workflow-paths` — enumerate paths from initial to terminal
- `workflow-state-count`, `workflow-transition-count`

### Implementation complexity

**Low to moderate.** State machines are simpler than general automata — no alphabet abstraction needed, transitions are event-labeled edges. Reachability is BFS/DFS. Minimization is partition refinement (same as DFA minimization). Equivalence checking is product construction + emptiness check.

The compositional operations (sequential, parallel, conditional) are slightly more involved but well-defined in the automata theory literature.

### Connection to agentic coding

This domain is directly useful for agents:

1. **Workflow generation:** An agent asked to "build a deploy pipeline" could use nous-discovered patterns as templates.
2. **Workflow validation:** Given a proposed workflow, check it against discovered properties (deadlock-free? error-covered?).
3. **Workflow optimization:** Given a working but complex workflow, apply discovered simplifications.
4. **Error handling:** Discovered error recovery patterns become reusable components.

### Seed content

- 3-4 example workflows: simple linear pipeline (3 states), deploy pipeline (8 states), incident response (5 states), data processing batch job (6 states)
- Basic operations: MergeStates, InsertState, AddErrorHandler, AddRollback
- Compositional operations: SequentialCompose, ParallelCompose
- Optimization: Minimize, RemoveUnreachable
- Known properties as seed conjectures: "every workflow needs at least one error state," "rollback from any state to idle is always safe if all transitions are idempotent"
