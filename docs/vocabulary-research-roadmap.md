# Vocabulary research roadmap

## Status

Accepted roadmap, revision 3. This document freezes the intended six-phase
sequence and the shared experimental standard. Each phase still requires its
own stabilized implementation plan before code is written.

The accepted [v2 research program](vocabulary-research-program-v2.md) extends
this roadmap with the required Phase 2 recovery gate and Phases 7 through 10.
This document remains authoritative for the original program invariants and
Phases 1 through 6; the v2 program is the accepted amendment for execution
order and later phases.

Adversarial review record, 2026-08-07:

- Chandrasekhar, architecture and integration boundary: `ACCEPT`;
- Lovelace, domain algorithms and bounded feasibility: `ACCEPT`; and
- Harvey, experimental validity and leakage resistance: `ACCEPT`.

The accepted revision is the version contained in the roadmap commit. Earlier
review rounds rejected ambiguous engine boundaries, invalid-null conflation,
unbounded semantic universes, shared active-policy transcripts, and incomplete
meta-search bounds; revision 3 incorporates their required corrections.

The sequence is:

1. relational rule induction and predicate invention;
2. active causal diagnosis;
3. resource-constrained deployment scheduling;
4. execution-trace invariant discovery;
5. black-box service model learning; and
6. spatial transformation puzzles.

Phases are delivered one at a time. A phase is not started until its specific
plan has passed adversarial review. Each phase ends in one of three states:

- `valid-positive`: all mechanical gates pass and the preregistered empirical
  hypothesis passes;
- `valid-null`: all mechanical gates pass but the empirical hypothesis does not
  pass; or
- `invalid`: a mechanical, leakage, budget, oracle, or preregistration gate
  fails, so no empirical conclusion is allowed.

Both valid outcomes are committed and may advance the sequence. A null result
cannot be retuned in place: changes require a new experiment version that
preserves the old report. An invalid phase remains open and blocks the next
phase. If a later phase assumed a capability that produced a valid null result,
its design must remove that assumption or explicitly retest it before review.

## Why this sequence

The existing protocol, rewrite, configuration-repair, tiny-stack, and game
packs demonstrate that Nous can host bounded representations, exhaustive
candidate construction, complete evidence barriers, independent oracles,
contextual credit, held-out evaluation, and vector-valued selection. They do
not yet demonstrate that Nous can:

- invent a reusable concept or representation;
- choose which evidence to obtain;
- improve search under a budget too small for enumeration;
- infer and actively falsify a property from traces;
- identify an unknown system through queries; or
- learn reusable abstractions across structurally related tasks.

The roadmap is a deliberate research-risk progression, not a software
dependency graph. It tackles the hardest representational uncertainty first,
then active evidence choice, budgeted search control, defeasible conjectures,
counterexample-guided identification, and finally representation/library reuse
in a less language-like domain. Later packs do not consume earlier learned
state. The order may be amended only through another accepted roadmap review.

This follows the central lesson of EURISKO's representation: syntactic changes
must usually denote meaningful semantic variants, and learned concepts and
heuristics must remain first-class inspectable objects. It also reflects later
work on predicate invention, active causal discovery, reusable program
libraries, invariant inference, and active automata learning.

Research anchors:

- Douglas Lenat, [EURISKO: A program that learns new heuristics and domain
  concepts](https://doi.org/10.1016/S0004-3702(83)80005-8);
- Andrew Cropper and Rolf Morel, [Predicate Invention by Learning From
  Failures](https://arxiv.org/abs/2104.14426);
- Nino Scherrer et al., [Learning Neural Causal Models with Active
  Interventions](https://arxiv.org/abs/2109.02429);
- Kevin Ellis et al., [DreamCoder: Growing generalizable, interpretable
  knowledge with wake-sleep Bayesian program
  learning](https://arxiv.org/abs/2006.08381);
- Michael Ernst et al., [The Daikon system for dynamic detection of likely
  invariants](https://homes.cs.washington.edu/~mernst/pubs/daikon-tool-scp2007-abstract.html);
- Malte Isberner and Bernhard Steffen, [An Abstract Framework for
  Counterexample Analysis in Active Automata
  Learning](https://proceedings.mlr.press/v34/isberner14a.html); and
- François Chollet, [On the Measure of Intelligence](https://arxiv.org/abs/1911.01547).

These sources motivate experimental shapes; they are not dependencies and
their results are not claims about Nous.

## Program-wide invariants

### Vocabulary boundary

Every phase is a normal domain pack loaded with `domains/common` and no other
domain. It may register one scoped DSL extension. V1 phase implementations may
add only `domains/<pack>`, pure `internal/vocab/<name>` code, store-aware scoped
`internal/dsl/builtins_<name>.go` adapters, experiment/oracle code, CLI wiring,
tests, and documentation. They may use only the existing VM, Store, Agenda, and
RNG surfaces.

This roadmap forbids edits to the engine, agenda, mutation machinery, common or
math domains, VM fields or hooks, global teacher registries, and imports between
domain packs. If a phase appears to require one, implementation stops. The
roadmap must be amended and accepted again, accompanied by a separate
architecture decision, before such code is written.

Packs remain isolated. Sequence dependencies are evidence gates only. No phase
imports prior domain units, learned store state, scoped DSL words, experiment
fixtures, or reports. Reuse claims mean that an invented rule, heuristic, or
abstraction is reused across multiple tasks in the same pack and run. A shared
pure-code extraction requires a separately reviewed refactor, cannot include
domain fixtures or learned state, and does not count as learned transfer.

### Ordinary heuristics own discovery

Go may provide bounded parsing, canonicalization, pure execution of one
explicit semantic object, per-object comparison, and artifact-integrity checks.
A builtin may validate or execute one explicit object, or return
descriptor-legal one-step refinements for one explicit candidate hole. Every
refinement edge and its work charge is materialized.

A builtin may not inspect the training corpus while generating, assemble a
complete multi-step candidate, enumerate an entire hypothesis/program universe
for one call, perform global pruning or ranking, return an argmax/frontier or
answer-bearing subset, choose the next active action, combine generation with
evaluation, or expose oracle truth. Ordinary CUE heuristics must materialize
candidates, request evaluations, apply evidence-backed pruning, and promote
supported discoveries through agenda tasks.

Descriptor-selected categories, slots, examples, task slots, priorities,
bounds, and comparison methods prevent heuristics from naming the answer.
Opaque aliases and a runtime-built alternate descriptor must work without
heuristic edits.

### Active trial protocol

Phases 2, 4, and 5 interact with a hidden deterministic teacher without adding
an engine hook. Teacher and transcript-replay semantics live in an isolated
`internal/<name>oracle` package with no dependency path to engine, DSL, or the
production vocabulary. The experiment driver may import both that isolated
package and production orchestration. An ordinary heuristic may only materialize one proposed
intervention, probe, or query and put the descriptor in a versioned
`awaiting-teacher` state. The `internal/<name>exp` driver then:

1. pauses after bounded engine work;
2. reads the proposed semantic action from store artifacts;
3. invokes the isolated independent teacher;
4. appends exactly one collision-safe semantic response unit; and
5. resumes the run by pushing an ordinary descriptor-declared agenda task.

The production vocabulary and DSL packages never import, own, query, hash, or
inspect the hidden teacher/model. `nous run -domain <pack>` terminates safely
with `awaiting-teacher`; it never fabricates an answer. The transcript records
gap-free sequence numbers, prefix digests, semantic actions and responses,
costs, remaining budgets, cache status, and terminal reason. A replay verifier
independently checks every transition.

### Evidence and integrity

Every evaluated candidate retains inspectable candidate, result, observation,
application, aggregate-evidence, selection, schema, and conjecture records as
appropriate. All records carry a versioned semantic experiment key and a
canonical profile digest. Allocation is semantic, idempotent, and
collision-safe.

Each phase plan selects exactly one integrity contract:

- `exhaustive-matrix`: the verifier proves the entire declared Cartesian
  candidate/evidence matrix complete and exact; or
- `budgeted-transcript`: the verifier proves a gap-free action/evaluation
  transcript, deterministic eligibility, budget conservation, no missing result
  for a consumed action, selection only over the declared evaluated subset, and
  one terminal reason from `identified`, `no-solution`, `budget-exhausted`, or
  `awaiting-teacher`.

The budgeted contract never implies universe coverage. A post-selection
verifier recomputes the declared selection and validates every artifact, link,
category, worth, count, transcript prefix, and terminal state. Missing,
duplicate, forged, extra, miscategorized, uncharged, or stale-profile evidence
fails closed.

### Independent evaluation

Each phase has an independent oracle. The entire `internal/<name>exp` package
must have no dependency path to `internal/vocab/<name>` or its DSL adapter, as
enforced by import/dependency tests, or the oracle must live in a separate
`internal/<name>oracle` package with the same mechanical proof. Trial
orchestration exercises production only through `seed.LoadDomain`, the engine,
and store artifacts. Tiny instances are exhaustively checked. For deliberately
non-exhaustive experiments the oracle verifies semantics and small-instance
optima, not the main search result.

Held-out fixtures, labels, schedules, targets, and oracle-only ground truth are
absent from training units and attributed artifacts. Reporting held-out results
must leave the canonical training store byte-identical.

### Preregistration and locked evaluation

Every accepted phase plan contains a machine-readable or mechanically parsed
preregistration manifest freezing:

- generator, grammar, semantic-cost, oracle, baseline, and experiment versions;
- exact development, training, validation, and locked-test seed panels;
- cohort sizes, generator distributions, semantic rejection criteria, and
  stopping rules;
- the primary endpoint, direction, minimum effect size, paired test, confidence
  interval, alpha, tie policy, and all secondary metrics;
- maximum descriptor size, exact finite candidate-grammar bound, evaluated
  candidates, queries/interventions, engine cycles, attributed units,
  trace/query/program length, deterministic report bytes, and oracle-only bound;
- every baseline and hyperparameter, cache and duplicate policy, information
  rights, shared-randomness rule, and matched semantic budget;
- ambiguity, timeout, failure, invalid, no-solution, abstention, and
  budget-exhaustion scoring; and
- the complete report schema and selected integrity contract.

Before implementation, the plan gives a worst-case semantic-work and artifact
calculation under these bounds. Generator rejection depends only on frozen
semantic properties, never on production or baseline success. Development and
locked panels are distinct. Locked evaluation happens only after an
implementation-candidate commit. Inspecting locked output and then changing
code, grammar, features, filtering, thresholds, or heuristics creates a new
experiment version and preserves the earlier result.

Unless a phase plan justifies an exact alternative, empirical cohorts contain
at least 64 locked paired instances and use a paired randomization test at
`alpha=0.05` with a two-sided 95% bootstrap confidence interval. A positive
claim also requires its frozen minimum effect size; statistical significance
alone is insufficient.

### Search and transfer claims

From phase 1 onward, correctness alone is insufficient. Every report contains:

- candidate or hypothesis evaluations consumed;
- evidence queries or interventions consumed;
- a complete wall-clock-independent semantic cost ledger, including candidate
  refinement, normalization, fixed-point steps, adapter-internal work,
  validation, teacher work, and cache/duplicate handling;
- results for a random baseline with fixed seeds;
- results for at least one credible conventional algorithm and one static or
  random baseline;
- results for the relevant ablation, such as no invention, passive evidence,
  no credit, no learned rule, or cell-only representation;
- complete tied selections rather than name-based tie breaking; and
- negative and no-solution outcomes.

A phase may claim search or evidence advantage only if the preregistered primary
endpoint, effect size, paired test, and interval pass on the locked panel. A
single seed instance is a sensitivity case, not an advantage claim. All
multi-objective reports freeze one primary scalar, lexicographic, or Pareto
deployment rule before holdout; a larger set does not win merely because it
contains one favorable held-out member.

Reuse must be causal in the experiment. Every claim includes both a no-artifact
ablation and an equal-expressivity inline or independently recomputed-artifact
ablation. Discovery and definition costs are charged and amortized over a
frozen number of downstream tasks. Aliasing preserves the benefit. Merely
observing the same pattern twice, or removing the only expressive primitive,
is not transfer.

### Encoded-evaluator controls

Every phase includes neutral, adversarial, beneficial, and harmful/irrelevant
generated cohorts where applicable; target and alias permutation; a
runtime-built alternate descriptor; candidate insertion and deletion; evidence
perturbation that changes the correct answer; and source/dependency enforcement
for the oracle. Production code cannot use fixture names, lexical order, target
labels, hidden seeds, or oracle annotations. Material work inside Go adapters is
charged; oracle work is excluded and clearly separated.

### Reproducibility and safety

Every trial exposes its random seed and emits deterministic JSON for that seed.
No vocabulary can mutate an external system. Kubernetes, Terraform, service,
or deployment names are representational fixtures only. PUDL and Mu adapters
remain outside these bounded discovery experiments until a later integration
contract grants read or execution authority.

Tests cover vocabulary isolation, child-VM extension inheritance, task
idempotence, mutation-on and mutation-off determinism, malformed descriptors,
category injection, occupied names, opaque aliases, alternate descriptors,
case-order identity, evidence corruption, held-out leakage, and byte-identical
reports.

### Per-phase delivery protocol

For each phase:

1. write `docs/<phase>-vocabulary-plan.md` with the preregistration manifest,
   finite budget table, feasibility calculation, fixtures, baselines, expected
   sensitivity observations, non-claims, correctness gates, and empirical
   hypothesis;
2. obtain independent architecture, domain/algorithm, and experimental-validity
   reviews;
3. revise and re-review until all three explicitly accept the contract;
4. record all three verdicts and the accepted revision, then commit the plan;
5. implement the pure semantic layer and independent oracle first;
6. implement the descriptor, scoped adapter, ordinary CUE heuristics, report,
   and CLI command;
7. run development/validation, corruption, alternate-descriptor, baseline, and
   deterministic controls, then commit the implementation candidate;
8. run the locked panel once, preserve its report, and classify the phase as
   `valid-positive`, `valid-null`, or `invalid`;
9. document what the result does and does not establish and perform a
   requirement-by-requirement phase audit;
10. run repository tests, vet, scoped race checks, and `git diff --check`; and
11. commit the results, status, accepted-plan commit, implementation commit, and
    reviewer verdict identities before the next design.

Reviewers reject a phase if the expected answer is encoded in fixture names,
heuristic bodies, enumeration order, the Go adapter, or a post-hoc metric.
Mechanical validity and empirical success are separate. A valid null advances;
an invalid result blocks. Proceeding past an invalid phase requires an accepted
roadmap amendment.

## Phase 1: relational rule induction and predicate invention

Status: complete, locked `valid-null`. The [accepted plan](relational-rule-induction-vocabulary-plan.md)
and [locked trial report](relational-rule-induction-trials.md) show mechanically
valid bounded Horn induction, cross-task reuse, and large search/work reductions,
but the isolated materialized-execution effect was 3.284%, below the frozen 5%
positive gate. Per the roadmap, this valid null advances to Phase 2.

### Research question

Can Nous learn bounded relational programs from positive and negative examples,
invent a semantically useful intermediate predicate, and reuse it to solve new
tasks with less search than an otherwise identical no-invention learner?

### Representation

The pack is `domains/ruleinduction`, the scoped extension is `ruleinduction`,
the pure package is `internal/vocab/ruleinduction`, the independent trial
package is `internal/ruleinductionexp`, and the CLI command is
`ruleinduction-trials`.

Facts are canonical ground unary or binary relations over bounded symbols.
Rules are safe, function-free Horn clauses using canonical variables and
descriptor-declared predicate signatures. Hypotheses contain a bounded ordered
set of clauses. V1 permits descriptor-selected metarules for identity, chain,
projection, and bounded tail recursion; at most one invented binary predicate;
at most four clauses; at most three body literals per clause; and finite-domain
least-fixed-point evaluation.

Invented predicate names are allocation identities. Their semantic keys derive
from normalized defining clauses. Alpha-equivalent clauses and hypotheses have
one semantic identity.

Roadmap ceilings are 12 constants, six background predicates, three target
predicates, the variables `X/Y/Z`, five metarules, 64 ground facts, four tasks,
four clauses per joint theory, three body literals per clause, 50,000 legal
normalized theories, 20,000 evaluated theories, 1,000,000 charged fixed-point
derivation steps, 2,000 engine cycles, 100,000 attributed units, and an 8 MiB
report. The phase plan must choose values no larger than these, derive the exact
metarule-instantiation and normalized-theory counts, and show that every
baseline fits its equal budget.

### Trial shape

The seed family uses renamed directed service-topology facts. Multiple target
tasks require transitive dependency information, such as affected-service and
maintenance-impact relations. Neither an invented predicate's name nor its
clauses appear in the descriptor.

The locked panel separately reports invention-beneficial, invention-neutral,
and invention-harmful cohorts. On the search-efficiency cohort, both shared and
no-sharing languages can express and solve every target; the experiment cannot
win by reserving necessary expressivity for invention. Training includes
positive and negative examples for at least two tasks.
Held-out graphs change node names, topology depth, branching, and target facts.
An alternate descriptor changes every category, relation name, metarule name,
task order, and example identity.

Baselines are (1) naive bounded no-invention enumeration without pruning,
(2) conventional learning-from-failures without invention, and (3)
learning-from-failures with equal-expressivity task-local invention but no
shared library. The production policy adds shared invention. All use identical
metarules, ordering rights, entailment cache, and semantic-work budget. The
primary metric is charged semantic work to solve all tasks, including
refinement, normalization, failed hypotheses, pruning-constraint construction,
and fixed-point derivations. Description length charges shared definitions plus
every call site and is secondary.

### Implementation guidelines

- Implement canonical terms, substitutions, unification, safe-clause
  validation, stratified bounded evaluation, entailment, and alpha
  normalization as pure functions.
- Keep candidate generation in ordinary heuristics. Go may return legal
  one-hole metarule fillers for one explicit partial clause without seeing the
  corpus; it may not enumerate complete clauses, theories, or winners.
- Represent failure-derived constraints explicitly: too-general failures prune
  generalizations and too-specific failures prune specializations. Retain the
  failed hypothesis and the learned pruning constraint as evidence.
- Separate per-task target clauses from shared library clauses. Promotion of an
  invented predicate freezes after the training curriculum and requires use by
  a preregistered number of downstream tasks. Compare no-artifact,
  equal-expressivity inline, task-local reinvention, and shared-library runs;
  charge invention cost before amortization.
- Compare complete programs extensionally on the bounded domain. Do not treat a
  textual clause match as correctness.
- The independent oracle implements its own fixed-point evaluator and searches
  the tiny fixture space without importing production normalization or
  entailment helpers.
- Before implementation the phase plan supplies a hand-checked invented joint
  theory and a hand-checked no-sharing theory, their normalized sizes, exact
  entailments, and the non-expressivity-based mechanism expected to reduce
  search. Both must fit the same budget.

### Acceptance boundary

Mechanical validity requires exact semantics, complete tied minimal-theory
reporting within the declared evaluated space, oracle-correct scoring of every
frozen held-out prediction, honest cost accounting, and real frozen-library
execution. Held-out accuracy is an empirical outcome, not a validity gate. The
positive-result hypothesis requires the preregistered work reduction against
the equal-expressivity task-local baseline while meeting the frozen locked
accuracy threshold.
A valid null is retained. The phase does not establish unrestricted ILP,
natural-language concept formation, or cross-pack knowledge transfer.

## Phase 2: active causal diagnosis

### Research question

Can Nous maintain competing causal hypotheses and choose interventions that
identify a hidden fault mechanism using fewer costly observations than passive,
random, and fixed-order policies?

### Representation

The pack is `domains/causal`, extension `causal`, pure package
`internal/vocab/causal`, trial package `internal/causalexp`, and CLI command
`causal-trials`.

V1 uses acyclic structural causal models over three to five binary variables,
maximum indegree two, and mechanisms of arity at most two.
Each variable has a bounded deterministic mechanism selected from constant,
copy, negation, conjunction, disjunction, and exclusive-or over declared
parents. A hypothesis contains a DAG plus mechanisms. Observations and
interventions are distinct records; interventions replace one mechanism for
one trial and have explicit cost.

The hidden model exists only in the independent teacher. Training units contain
the variable vocabulary, allowed mechanisms, initial observations, intervention
costs, and candidate hypotheses—not oracle ground truth.

Each descriptor declares and deduplicates an explicit pool of at most 4,096
hypotheses; the hidden model is a member. Legal actions are `do(V=0)` and
`do(V=1)` for one variable, at most ten actions. One action returns all declared
variables. V1 is noiseless and deterministic; noisy and failed interventions
are deferred to a separately reviewed version. The prior is uniform over the
current pool. Identification means a singleton pool or a declared
interventional-equivalence class when no legal action distinguishes its
members.

Roadmap ceilings are five variables, 4,096 hypotheses, ten legal interventions,
ten consumed interventions per episode, 40 candidate acquisition rules, 512
training episodes, 128 locked episodes, 5,000 engine cycles, 150,000 attributed
units, 2,000,000 charged hypothesis evaluations, and a 16 MiB report.

### Trial shape

Small exact fixtures model cascading service symptoms with observationally
ambiguous causes. Candidate interventions resemble bounded safe diagnostic
actions such as forcing one synthetic component healthy or unhealthy.

Nous learns and freezes an acquisition-rule program across training episodes.
A rule lexicographically combines a bounded subset of expected remaining
hypotheses, worst-case remaining hypotheses, outcome entropy, intervention
cost, and repeat/cache indicators. Ordinary heuristics construct and credit
these rules; they choose actions only from stored predicted partitions.

Baselines are passive-only, lexical fixed order, uniformly random,
cost-normalized information gain, worst-case split per cost, and an exact
dynamic optimal policy on oracle-bounded three-variable fixtures. All share the
same hidden model, deterministic response function, initial observations,
hypothesis pool, intervention set, uniform prior, total-cost ceiling, and seed.
Each policy has its own transcript and semantic cache; neither is shared across
policies. Cohorts vary aliases, graph shape, mechanism mix, costs, irrelevant
variables, and identifiable versus equivalence-only outcomes. Generator aliases
and order are independent of the hidden model.

The primary endpoint is total intervention cost to correct identification,
with the intervention-cost ceiling frozen at 1,000 and budget exhaustion scored
at the sentinel 1,001. Every legal intervention has integer cost in `[1,100]`.
One deterministic
semantic-code-minimal action is executed from a score tie; alternatives are
reported but do not consume trials. The empirical hypothesis compares the
frozen learned acquisition rule with cost-normalized information gain.

### Implementation guidelines

- Implement DAG validation, topological evaluation, intervention semantics,
  observational equivalence, and exact hypothesis filtering in the pure layer.
- Materialize every proposed intervention, predicted outcome partition,
  teacher result, eliminated hypothesis, and posterior candidate set.
- Candidate intervention scoring is an ordinary heuristic computation over
  declared costs and predicted partitions under the uniform current pool. The
  exact expected-remaining, worst-case, entropy, and cost feature formulas are
  frozen in the phase plan. The hidden outcome cannot be queried during scoring.
- Freeze deterministic tie handling and report all equal-scoring interventions;
  a seeded policy chooses among ties only when the trial explicitly measures a
  sequential policy.
- Freeze acquisition-rule training order and credit before locked episodes.
  Include no-credit, wrong-context, static-rule, and equal-expressivity
  recomputed-rule controls.
- The oracle independently enumerates and evaluates SCMs and verifies every
  hypothesis-set update and intervention partition.

### Acceptance boundary

Mechanical validity requires exact hypothesis filtering, transcript replay,
budget conservation, and oracle-correct scoring of every terminal outcome.
Correct model or equivalence-class identification, declared indistinguishability,
and budget exhaustion are all valid scored outcomes. The replay verifier proves
that every `identified` posterior is singleton or equivalence-complete and
contains the teacher model; a false `identified` terminal is mechanically
invalid. A weak policy must terminate `budget-exhausted` and score 1,001 rather
than assert identification.
The positive-result hypothesis requires the frozen learned acquisition rule to
beat cost-normalized information gain by the preregistered paired effect size.
A valid null is retained. The phase does not establish causal validity for real
telemetry or authority to intervene on live systems.

## Phase 3: resource-constrained deployment scheduling

### Research question

Can Nous construct and improve priority heuristics that find feasible,
high-quality deployment schedules under a search budget that cannot enumerate
the schedule space, and do those heuristics transfer to held-out instances?

### Representation

The pack is `domains/scheduling`, extension `scheduling`, pure package
`internal/vocab/scheduling`, trial package `internal/schedulingexp`, and CLI
command `scheduling-trials`.

An instance declares tasks, integer durations, precedence edges, renewable
resource demands and capacities, maintenance windows, and incompatibilities. A
schedule assigns start times and is valid only when all temporal and resource
constraints hold. Rollback milestones and stochastic durations are deferred
from V1.

The learned objects are priority-rule programs, not memorized schedules. A
bounded rule combines critical-path length, successor count, slack, duration,
resource scarcity, and unlock value. V1 has one algorithm: the rule orders
eligible tasks and one deterministic serial schedule-generation scheme places
them. Local search and neighborhood moves are deferred.

Rule ASTs contain at most five nodes, use at most three features, and draw
integer weights only from `{-2,-1,0,1,2}`; lexical comparators use at most three
keys. Instances contain at most 12 tasks, three renewable resources, duration
and capacity at most 16, horizon 64, 32 precedence edges, four windows, and four
incompatibilities. Tiny exact instances contain at most eight tasks. The exact
finite grammar is computed in the phase plan and capped at 50,000 normalized
rules; at most 512 rules are evaluated per training instance, 5,000 engine
cycles and 200,000 attributed units are allowed, and reports are at most 16 MiB.

### Trial shape

Tiny instances have exact branch-and-bound optima. Main instances are large
enough that the phase evaluation budget covers only a small fraction of rule
programs. Training, validation, and locked cohorts are structurally disjoint by
preregistered graph/resource generator family, not merely aliases. They include
feasible, infeasible, bottleneck-neutral, and misleading-feature controls.

Baselines are shortest and longest duration, critical path, most successors,
uniform random AST search, and validation-guided beam search over the identical
rule grammar. All receive the same initial rules, serial generator, cache, and
rule/schedule-evaluation budgets. The primary per-instance loss is `makespan`
for a feasible schedule and `horizon+1` for infeasibility; the paired locked
endpoint is mean loss. Feasibility, tiny-instance optimality gap, and work are
secondary and cannot replace it post hoc.

### Implementation guidelines

- Implement canonical instances, resource profiles, deterministic serial
  schedule generation, validation, primary loss, and tiny exact search in the
  pure layer.
- Generate priority-rule ASTs compositionally as ordinary candidate units with
  component and contextual credit. Do not add a builtin that searches all ASTs.
- Use a strict evaluation budget and report unevaluated candidates. The
  selection barrier proves budget accounting and evidence completeness for the
  evaluated subset, not exhaustive coverage of the universe.
- Separate rule learning on training instances from held-out scheduling. Freeze
  learned rules before held-out evaluation.
- Include credit, no-credit, exploration-reserve, and static-rule ablations.
  Contextual credit changes only the distribution of one-step AST refinements;
  the no-credit policy samples those same legal refinements uniformly with the
  same evaluation count. Freeze learned rules and credit before locked runs.
  Compare no rule, equal-expressivity inlined rule, independently regenerated
  rule, and frozen shared rule.
- The oracle independently validates schedules and solves only bounded tiny
  instances exactly; it must not be presented as the main search baseline.

### Acceptance boundary

Mechanical validity requires schedule correctness, exact tiny optima, complete
subset/transcript integrity, and budget accounting. The positive-result
hypothesis requires the frozen credit-guided rule learner to improve the
preregistered paired primary loss over equal-budget beam search without lower
locked feasibility. A valid null is retained. The phase does not establish
production-safe rollout planning or optimality on large instances.

## Phase 4: execution-trace invariant discovery

### Research question

Can Nous infer useful relational and temporal invariants from execution traces,
distinguish stable properties from accidental correlations, and choose
counterexample probes that falsify weak conjectures?

### Representation

The pack is `domains/invariants`, extension `invariants`, pure package
`internal/vocab/invariants`, trial package `internal/invariantsexp`, and CLI
command `invariant-trials`.

Traces contain ordered events and bounded typed variables at declared program
points. V1 has at most eight Boolean or bounded integer variables, with every
integer domain a declared subrange of `[-4,4]`, five descriptor constants, 32
events per trace, and temporal horizon 32. Candidate
expressions contain at most seven AST nodes and combine variables, constants,
differences, counts, and one-level aggregates. Candidate invariants include
equality, inequality, bounds, monotonicity, implication, conservation, and
bounded precedes; membership and until are deferred.

Invariant forms are compositional ASTs rather than one hard-coded template per
answer. Derived variables and conditional guards are first-class candidate
concepts with provenance.

The finite candidate universe is capped at 10,000 normalized invariants and 256
legal probes. Each machine has at most 64 reachable bounded states; its exact
oracle ground truth is every reachable execution up to the frozen horizon,
subject to a preregistered ceiling of 1,000,000 execution prefixes and
16,000,000 transition-formula evaluations per machine. A generator rejects a
machine on semantic size alone if either ceiling would be exceeded.

Class identity is the invariant's canonical logical normal form, independent
of the machine and observed executions. Normalization covers typed alias
renaming, constant folding, commutative ordering, comparison orientation, and
the phase plan's finite sound rewrite table; it does not merge two formulas
merely because they agree on reachable traces. A class is labelled true only
when every reachable execution satisfies its representative. The complete
denominator for precision and recall is the same finite set of normalized
candidate classes for every policy on that fixture. Implication and redundancy
are separate bounded-oracle relations between classes. Production evaluates at
most 5,000 candidates and 64 probes, uses at most 5,000 engine cycles and
200,000 attributed units, and emits at most 16 MiB.

### Trial shape

Generated queue, replica, and resource-lifecycle machines produce training
traces. Some correlations are true invariants; others survive the small
training sample but fail on adversarial probes. A separate teacher can execute
a bounded input probe and return a trace. Held-out machines preserve selected
semantic invariants while changing names, ranges, and irrelevant variables.

The locked panel freezes proportions for true-invariant, accidental-correlation,
probe-neutral, and no-nontrivial-invariant machines. Held-out schema transfer
means freezing a normalized invariant/probe heuristic after source machines,
mapping only descriptor-declared typed aliases, and allowing only the frozen
initial evidence on target machines.

Baselines are a fixed canonical-template detector, passive enumeration, random
probes, and a conventional boundary/coverage-directed probe policy. All share
initial traces, legal probes, probe cost, hidden machine, response function,
and seed, but each policy owns its transcript and semantic cache. The primary endpoint is
macro F1 over the complete canonical-class universe after the fixed probe
budget. If truth and prediction are both empty F1 is one; if exactly one is
empty F1 is zero; otherwise it is the harmonic mean of precision and recall.
Precision, recall, false discoveries, probes to falsification, AST
size, and work are secondary.

### Implementation guidelines

- Implement typed trace parsing, expression evaluation, temporal evaluation,
  canonical AST identity, implication checks over bounded traces, and exhaustive
  small-machine truth in the pure layer.
- Distinguish `observed-supported`, `falsified`, and `proved-on-bounded-model`.
  Never label trace survival alone as proof.
- Materialize counterexamples and the exact trace position that falsified an
  invariant. HindSight-style heuristics should derive narrower guards or better
  probes from failures.
- Freeze the probe budget. Candidate probes are scored without oracle outcomes
  using stored candidate disagreement, uncovered transitions, boundary values,
  and cost. Ordinary heuristics construct and freeze bounded probe-scoring
  programs before locked machines. A scoring program contains at most five AST
  nodes, uses at most three of those four features, combines them with
  lexicographic comparison and weighted sum, and uses weights from
  `{-2,-1,0,1,2}`. The phase plan derives a maximum of 2,000 normalized
  programs, evaluates at most 512 of them over at most 256 training machines,
  permits at most four one-hole refinements from one artifact, and stores at
  most 4,096 scoring-program artifacts.
- Require at least one useful derived variable or conditional invariant whose
  no-artifact and equal-expressivity expanded/inlined ablations distinguish
  reuse from lost expressivity and charge definition cost.
- The independent oracle enumerates bounded machine executions and evaluates
  invariants using a separate AST interpreter.

### Acceptance boundary

Mechanical validity requires exact candidate semantics, semantic-class ground
truth, calibrated `observed-supported`/`falsified`/`proved-on-bounded-model`
statuses, exact counterexamples, and transcript/budget integrity. The
positive-result hypothesis requires the frozen active policy to improve locked
macro F1 over boundary/coverage probing by the preregistered effect size at the
same probe cost. A valid null is retained. The phase does not establish static
program verification or correctness of arbitrary production traces.

## Phase 5: black-box service model learning

### Research question

Can Nous actively infer a minimal behavioral model of an unknown service by
choosing queries, refining hypotheses from counterexamples, and reusing query
heuristics across systems?

### Representation

The pack is `domains/modellearning`, extension `modellearning`, pure package
`internal/vocab/modellearning`, trial package `internal/modellearningexp`, and
CLI command `modellearning-trials`.

V1 learns deterministic Mealy machines with two or three input symbols, two or
three output symbols, and at most six reachable states. The service-under-learning is
an oracle-only machine. Training units contain the alphabet, query budget,
observation table, access sequences, distinguishing suffixes, hypothesis
machine, and returned counterexamples.

Membership queries have length at most 72. Because both target and promoted
hypothesis contain at most six reachable states, their product has at most 36
state pairs; the phase plan must prove that a shortest distinguishing Mealy
trace has length at most 36 under its frozen output convention. Counterexamples
therefore have length at most 36 and the teacher never truncates one. Access
sequences are canonical shortest hypothesis paths of length at most five and
stored distinguishing suffixes have length at most 36. Linear, binary, and
discrimination-tree processing may concatenate at most one counterexample
prefix of length 36 with one stored suffix of length 36; closure and consistency
queries contain at most a five-symbol access sequence, one input symbol, and a
36-symbol suffix. Thus every grammar-legal processing query is at most 72
symbols; the phase plan must prove these bounds for each typed action. An
episode permits 512 membership queries, 32 equivalence queries, 8,192 total
membership/counterexample symbols, 256 observation-table rows, 64 suffix
columns, 5,000 engine cycles, 200,000 attributed units, and a 16 MiB report.
Teacher equivalence work is reported separately. The teacher always returns the
shortest counterexample and breaks equal-length ties lexicographically, so
counterexample choice cannot favor a policy.

The learned reusable object is a bounded counterexample/query decision program
typed by decision context. Counterexample processors may select linear prefix
scan, binary split, or discrimination-tree suffix; closure repairs may select
only access-sequence additions; consistency repairs may select only legal
distinguishing-suffix additions. They use counterexample length, current table
dimensions, defect type, prior outcome, and semantic cost. Every action must
strictly increase the number of filled cells or resolve the selected
lexicographically first defect; normalized learner states cannot repeat within
an equivalence round. Ordinary heuristics construct, credit, and freeze these
programs after a declared training-machine curriculum.

A decision program has at most six AST nodes, tests at most four features with
`<`, `=`, and typed conjunction, uses thresholds from
`{0,1,2,4,8,16,32,36,72}`, and ends in a context-legal action. The phase plan
derives a maximum of 4,096 normalized programs, evaluates at most 512 over at
most 256 training machines, permits at most four one-hole refinements from one
artifact, and stores at most 8,192 decision-program artifacts.

### Trial shape

The learner issues membership queries and equivalence proposals.
Counterexample-processing heuristics choose prefixes and suffixes that refine
the observation partition. Machine fixtures vary aliases, state count,
distinguishing depth, redundant access sequences, and long counterexamples.

Baselines are conventional L* with linear counterexample scan, L* with binary
split, random legal traces, and exhaustive sampling through a declared finite
depth that is explicitly not an equivalence proof unless it reaches the
phase-plan sufficient bound. All use identical observation-table semantics,
caching rules, hidden machine, deterministic response function, initial
evidence, budgets, and seed, but each policy owns its transcript and cache. The
primary semantic cost is total membership-query symbols plus returned
counterexample symbols plus a frozen per-equivalence-query charge of 16;
budget-exhausted episodes receive the ceiling plus one. Raw query counts,
teacher work, redundant queries, and learned states are secondary.

### Implementation guidelines

- Reuse semantic lessons from `protocols` but do not import its domain units.
  Extracting pure code is allowed only through an explicit refactor that keeps
  both accepted packs green.
- Implement Mealy execution, reachability, minimization, equivalence, and
  shortest counterexamples in the pure layer. The independent teacher and
  oracle reimplement execution/equivalence separately.
- Make the evolving observation table and every closure/consistency defect
  inspectable. Ordinary heuristics schedule the next query or hypothesis repair.
- Cache queries semantically and count cache hits separately; duplicate queries
  cannot masquerade as useful work.
- A learned model is promoted only after an equivalence response and complete
  post-selection verification. No hypothesis may exceed six reachable states.
  Query-budget exhaustion is a normal negative result and the teacher never
  truncates a semantic response to make it fit the remaining budget.
- Compare counterexample heuristics across a deterministic generated cohort and
  freeze the selected program before held-out machines. Include no-credit,
  wrong-context, static conventional, no-artifact, alias-preserving, and
  equal-expressivity recomputed-program controls. Deleting the promoted program
  must change query cost without changing the learner's expressivity.
- Deduplicate generated machines by minimal behavioral identity and prevent
  state numbering or alphabet order from correlating with fixture difficulty.

### Acceptance boundary

Mechanical validity requires table/transcript integrity, deterministic teacher
behavior, correct cost accounting, and oracle-correct scoring of every promoted,
budget-exhausted, or otherwise terminal episode. Every promoted model must be
minimally behaviorally equivalent to the target; budget exhaustion is a valid
scored outcome rather than an invalid experiment. The positive-result
hypothesis requires the frozen decision program to reduce locked primary
semantic cost relative to binary-split L* while meeting the preregistered exact
recovery-rate gate. A valid null is retained. This unique contribution
is observation-table closure/consistency plus counterexample decomposition over
interaction sequences, not generic finite hypothesis elimination. It does not
establish learning of nondeterministic, timed, stochastic, or live services.

## Phase 6: spatial transformation puzzles

### Research question

Can Nous choose an object-centric representation, synthesize transformations
from a few demonstrations, invent reusable spatial abstractions, and transfer
them to structurally novel held-out grids?

### Representation

The pack is `domains/spatial`, extension `spatial`, pure package
`internal/vocab/spatial`, trial package `internal/spatialexp`, and CLI command
`spatial-trials`.

Grids are rectangular color arrays of at most 8-by-8 cells, five colors, six
connected components, and three demonstrations per task. Candidate perceptions
identify connected components, bounding boxes, unique markers, adjacency,
reflection axes, and color roles. Holes, repeated-shape induction, containment,
and unrestricted symmetry are deferred. Candidate programs combine selection,
translation, reflection, recoloring, copying, composition, and rendering.

V1 searches one explicit meta-choice between cell-relative and object-relative
candidate grammars under equal semantic budgets; object annotations are never
provided by the oracle. Perception candidates are ordinary units produced by
one-step refinements, not a Go-returned answer-bearing object set.

V1 uses generated micro-puzzles inspired by general spatial abstraction work,
not copied hidden ARC evaluation items. Every generator and split is versioned
and deterministic.

### Trial shape

V1 contains exactly three preregistered families sharing a marker-relative
object-selection fragment: translate the selected object beside a unique
marker, reflect it across the marker-defined axis, and copy/recolor it into a
marker-defined target region. Pattern completion, extraction, and arbitrary
two-operation composition are deferred. Each task supplies three
demonstrations and one held-out grid.

The curriculum freezes library entries after the first training family. The
locked panel contains within-family withheld grids and family-disjoint generator
variants, reported separately; new colors or sizes alone do not count as
structural novelty. Compositions are outside V1. Generator filtering uses
semantic well-formedness only.

Roadmap ceilings are 256 perception candidates, 20,000 normalized programs,
six AST nodes, two library entries, 10,000 evaluated programs per task, 5,000
engine cycles, 250,000 attributed units, and a 24 MiB report. Invented fragments
are normalized executable ASTs with definition cost charged before their
frozen downstream task count.

Baselines are matched-expressivity cell-relative search, fixed object primitives
without library invention, task-local fragment reinvention, an inlined shared
fragment, uniform random search, and a conventional enumerative/CSP search over
the same grammar. All receive identical demonstrations, grammar expressivity,
semantic program-evaluation budgets, and pre-holdout selector rights. The
primary endpoint is lexicographic: exact held-out grid accuracy first, then
charged semantic work among equally accurate policies. Family-wise results are
mandatory; aggregate success cannot hide a failed family.

### Implementation guidelines

- Implement grid validation, connected components, canonical object identity,
  spatial relations, transformations, and rendering as pure bounded semantics.
- Separate perception candidates from transformation candidates. A program's
  evidence records which perception produced each referenced object.
- Invented library operations are normalized executable program fragments, not
  labels attached after success. Deleting the fragment must break or slow the
  later solutions that claim reuse.
- Freeze a grammar and cost model before trial execution. Do not add primitives
  after inspecting held-out failures in the same experiment version.
- Require exact reconstruction on every demonstration before held-out
  execution. Report all co-minimal exact programs and behavioral equivalence
  classes, but choose one semantic-cost-minimal, then semantic-code-minimal
  program before revealing holdout; success cannot be awarded because any
  member of a post-hoc set matches.
- The independent oracle has a separate grid executor and generator checks but
  does not search with production primitives or rank programs.

### Acceptance boundary

Mechanical validity requires exact grid/perception semantics, demonstration
reconstruction, frozen pre-holdout selection, no oracle object leakage, and
complete budget/integrity evidence. The positive-result hypothesis requires
the learned object-library policy to preserve locked exact accuracy and reduce
primary semantic work relative to the equal-expressivity inlined-fragment
baseline by the preregistered effect size. A valid null is retained. It does not
establish general ARC performance, visual reasoning over natural images, or
unrestricted representation learning.

## Cross-phase completion audit

The roadmap is complete only when all six phase plans and implementations are
committed. Each row records `demonstrated`, `not-demonstrated`, or `invalid`;
only the first supports the capability claim, while either valid state counts
as an honestly completed experiment:

| Phase | Unique artifact/algorithm | Unique evidence and ablation |
| --- | --- | --- |
| 1 | normalized invented Horn predicate shared across target tasks | equal-expressivity inline/task-local controls isolate shared concept reuse |
| 2 | cost-sensitive acquisition-rule program over causal-hypothesis partitions | optimal tiny policy and information-gain controls isolate active interventions |
| 3 | credited priority-rule AST searched under incomplete candidate coverage | equal-budget beam/no-credit controls isolate credit-directed search |
| 4 | calibrated defeasible invariant plus targeted falsification probe policy | bounded truth classes and boundary-probe controls isolate conjecture/falsification |
| 5 | observation-table counterexample/query decision program | conventional L* controls isolate closure, consistency, and counterexample refinement |
| 6 | executable perception/library fragment chosen across representations | cell/object/inlined controls isolate representation and library reuse |

A phase-specific plan is rejected or merged into an earlier phase if it cannot
identify an artifact, algorithm, primary metric, and causal ablation unique to
its row.

The final audit also verifies every phase's non-claims, report determinism,
independent oracle boundary, held-out immutability, negative controls, git
history, locked-panel preservation, and repository-wide regression results. Six
valid experiments do not by themselves prove open-ended discovery or production
utility. The final report states precisely which capabilities were
demonstrated, which produced valid null results, and whether any experiment was
invalid.
