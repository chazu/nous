# Vocabulary research program v2

Status: accepted on 2026-08-08 after three adversarial review rounds.

Review record:

- revision 0 was rejected for an unenforceable Go/CUE boundary, Phase 5/6
  endpoint drift, adaptive Gate 0 diagnostics, invalid/negative conflation,
  and unbounded later phases;
- revisions 1 and 2 were rejected for remaining dependency, one-shot receipt,
  feasibility-ledger, hidden-SUT, and terminal-extraction gaps; and
- revision 3 was explicitly accepted without blockers or major concerns by the
  architecture/integration, experimental-validity, and domain-theory reviewers.

This document extends the accepted
[vocabulary research roadmap](vocabulary-research-roadmap.md). It preserves the
roadmap's completed phases and its accepted phases 3 through 6, makes recovery
from the invalid active-causal replay an explicit prerequisite, and adds four
later vocabularies selected for distinct discovery capabilities and eventual
software-delivery relevance.

The program is a sequence of bounded research experiments. It is not a promise
that Nous may mutate production systems. Every represented deployment, trace,
service, workflow, failure, concurrent object, or optimizer remains a fixture
until a separate integration contract grants external authority.

## Program objective

The program asks whether EURISKO-style inspectable concepts and heuristics can
do progressively more than exhaustive construction in small closed domains.
Across the sequence, Nous must demonstrate—or honestly fail to demonstrate—the
ability to:

1. learn which candidates to explore when exhaustive search is unavailable;
2. invent and reuse defeasible concepts supported by evidence;
3. choose informative actions against a hidden system;
4. transfer learned artifacts across structurally related tasks;
5. discover partial-order, causal, and equivalence structure; and
6. improve a bounded search process without hiding the search in Go.

Correct semantics and attractive examples are insufficient. Each empirical
claim requires an independent oracle, conventional and random baselines,
matched semantic budgets, frozen held-out evaluation, explicit ablations, and
inspectable evidence linking discoveries to their consequences.

## Authoritative baseline

The original roadmap remains authoritative for its program-wide invariants,
experimental status model, information rights, evidence contracts,
preregistration rules, encoded-evaluator controls, and per-phase delivery
protocol. This document narrows or adds requirements; it does not relax them.

Current phase state is:

- Phase 1, relational rule induction: complete, locked `valid-null`;
- Phase 2, active causal diagnosis: `invalid` after the consumed v5 replay;
- phases 3 through 10: not started.

The exact v5 failure is recorded in
[active-causal-diagnosis-v5-invalid-run.md](active-causal-diagnosis-v5-invalid-run.md).
The failed v3, v4, and v5 receipts are immutable. Phase 2 blocks Phase 3 until a
new accepted, versioned Phase 2 amendment produces a mechanically valid replay
and the remaining validation and locked gates are completed. A replay-only
recovery may use a narrowly scoped technical amendment that preserves all
semantic identities. If recovery would change training, fixtures, scoring,
selected rule, evidence, or empirical claims, the amendment must instead
revise the affected experimental contract and explain why any further locked
attempt remains scientifically admissible.

## Fixed sequence

The execution order is:

0. active-causal replay recovery and completion of Phase 2;
1. Phase 3: resource-constrained deployment scheduling;
2. Phase 4: execution-trace invariant discovery;
3. Phase 5: black-box service model learning;
4. Phase 6: spatial transformation puzzles;
5. Phase 7: workflow and process-model discovery;
6. Phase 8: failure-inducing input and change minimization;
7. Phase 9: concurrent-history anomaly discovery; and
8. Phase 10: equality-saturation strategy learning.

Phases run one at a time. Before implementation, each phase receives its own
repository design and preregistration document and three independent reviews:
architecture/integration, domain algorithms, and experimental validity. All
three must explicitly accept the current revision. Implementation then
receives the same three reviews before locked evaluation.

A `valid-positive` or `valid-null` phase advances. An `invalid` phase blocks the
next phase. A blocked phase may be repaired only by a versioned amendment that
preserves the failed artifacts and states exactly which information was
observed. Locked results may never be tuned in place.

## Shared implementation contract

### Repository boundary

Each vocabulary is a normal pack under `domains/<pack>` loaded with
`domains/common` and no other domain. A v1 implementation may add:

- pure bounded semantics under `internal/vocab/<pack>`;
- at most one store-aware scoped DSL adapter under `internal/dsl`;
- separate independent oracle, baseline, and experiment-driver packages with
  mechanically enforced dependency boundaries;
- CLI wiring, deterministic reports, tests, and documentation.

The engine, agenda, mutation machinery, VM state, common and math packs, and
existing vocabulary semantics are fixed. If a phase requires changing one of
those surfaces, implementation stops and a separately reviewed architecture
amendment is required.

Every phase has four dependency layers:

1. production vocabulary and DSL packages implement only bounded one-object or
   one-step operations and may not import an oracle, baseline, fixture
   generator, experiment driver, or report package;
2. isolated baseline packages may implement complete conventional algorithms
   but may not import production, oracle, generator, driver, or private-fixture
   packages;
3. isolated oracle packages may implement exhaustive tiny-instance search and
   hidden semantics, but may not import production or baseline packages, or
   shared semantic/canonicalization implementations; and
4. the experiment driver may import baselines, oracles, and neutral seed,
   engine, store, and orchestration packages, but may not import the production
   vocabulary or its DSL adapter. It communicates with a loaded production run
   only through versioned store artifacts and declared CLI inputs.

Dependency tests mechanically enforce this graph. Complete branch-and-bound,
beam search, process mining, delta debugging, partial-order reduction, and
equality-saturation baselines belong only in layer 2. They are never callable
from production search.

### What Go may do

Go may parse and canonicalize one explicit semantic object, validate it,
execute one explicit transition or transformation, return bounded legal
one-step refinements for one explicit hole, and check artifact integrity.

Production Go may not inspect a training corpus while generating candidates,
enumerate a complete answer-bearing universe in one call, rank a population,
select a winner, choose an active query, perform global pruning, or combine
generation and evaluation. Ordinary CUE heuristics and agenda tasks must
materialize candidates, evidence, refinements, selections, and learned
artifacts. These restrictions do not prohibit isolated oracle/baseline
packages from implementing their declared complete reference algorithms.
A fixed, charged decoder may select a reported artifact only after the search
policy has irreversibly terminated, provided the phase contract makes its
inputs inaccessible to policy selection and charges all decoder work.

### Candidate and evidence protocol

Every phase chooses one accepted integrity contract:

- `exhaustive-matrix` for a fully declared and mechanically complete Cartesian
  evaluation; or
- `budgeted-transcript` for a deterministic, charged subset search or active
  interaction.

Budgeted search records eligibility, proposal, semantic identity, work charge,
result, evidence, and terminal reason for every consumed action. Unevaluated
candidates remain explicitly unevaluated. A subset result may not imply
universe coverage.

Each policy and ablation starts from an isolated clean store, transcript, and
cache. Every competitor receives the same information rights and one
wall-clock-independent total semantic-work ceiling. A phase-specific,
policy-neutral ledger charges generation, eligibility, ranking, refinement,
normalization, semantic evaluation, teacher work, duplicates, and cache work;
matching only evaluation counts is insufficient. The phase plan must show that
every baseline can be expressed and charged under that ledger.

Every selected artifact has a stable semantic key independent of aliases,
allocation order, unit names, and prose. Ties remain complete. Collision-safe
allocation never overwrites an occupied unit. Training-store bytes are frozen
before validation and locked evaluation.

### Credit and reuse

Contextual credit may change only the distribution over the same legal
one-step refinements available to the no-credit control. It may not change the
grammar or budget. Scalar and contextual updates, exploration reserve, cache
hits, duplicate handling, and semantic work are all reported.

Reuse claims require:

- a frozen learned artifact used on later tasks;
- a no-artifact ablation;
- an equal-expressivity inlined or independently recomputed ablation;
- discovery and definition cost charged before amortization; and
- alias-preserving behavior.

Merely retaining a human-written descriptor or removing necessary
expressivity does not count as learned transfer.

### Independent evaluation

The oracle must not import the production vocabulary, its DSL adapter, or any
semantic, canonicalization, or execution implementation shared with
production. It reimplements the relevant semantics and verifies tiny instances
exhaustively. Dependency and source-audit tests enforce that isolation.
Main-instance search results are compared with conventional baselines under
matched budgets; the oracle is not presented as a feasible large-instance
competitor when it is not one.

Each plan freezes development, training, validation, and locked panels,
generator versions, rejection criteria, primary endpoint, effect threshold,
paired test, interval, limits, tie policy, and failure scoring. Corruption,
alias, alternate-descriptor, candidate insertion/deletion, no-solution, and
irrelevant-feature cohorts are mandatory where meaningful.

`invalid` is reserved for a mechanical, integrity, leakage, oracle,
budget-accounting, or preregistration failure. A correctly reported
`budget-exhausted`, `no-solution`, `no-violation`, `ambiguous`, `abstained`, or
otherwise unsuccessful policy terminal is a mechanically valid observation
and receives its frozen sentinel score. It can make the phase `valid-null`; it
cannot by itself make the phase `invalid`. If a later phase assumes a
capability that received `valid-null`, its design must remove that assumption
or independently retest it before review.

### Hidden-system interaction

Phases 2, 4, 5, 8, and 9 use the roadmap's active `awaiting-teacher` protocol.
The isolated experiment driver owns the hidden teacher or system under test,
its state, and its private cache. Production and active baseline policies each
materialize exactly one semantic action and pause; the driver reads that
artifact, performs one hidden action, appends exactly one collision-safe
response, and resumes the policy. Baselines and oracles are sibling packages
that cannot import one another; the driver is the only mediator. Standalone
domain runs terminate safely in `awaiting-teacher` and never fabricate
responses.

Each policy owns a gap-free, prefix-digested transcript recording action,
response, cost, remaining budget, cache status, and terminal reason. Production
never imports, hashes, or inspects the hidden implementation, private state, or
unreturned alternatives. An independent verifier replays every transition.

### Delivery loop for every phase

1. Audit the preceding phase and confirm that advancement is allowed.
2. Write the phase-specific design and preregistration document.
3. Obtain the three adversarial design reviews and revise to unanimous
   acceptance.
4. Commit the accepted plan before implementation.
5. Implement the pure semantics and independent oracle first.
6. Implement scoped adapters, descriptors, ordinary heuristics, evidence,
   reporting, and CLI orchestration.
7. Run deterministic, corruption, alternate-descriptor, oracle, baseline, and
   development-only gates without accessing protected panels.
8. Obtain three adversarial implementation reviews and close every blocker.
9. Commit a clean implementation candidate and bind its digest into exclusive
   protected-attempt receipts.
10. Run validation once as one atomic attempt containing production, every
    baseline, and every ablation. The `started` receipt is durable before any
    fixture access; partial failure consumes the attempt. Any code change after
    access requires a newly reviewed version and cannot reuse that panel.
11. Only after mechanically valid validation, run locked evaluation once under
    a distinct exclusive locked receipt with the same binding, atomicity, and
    policy-cohort requirements. Preserve and classify the result; an
    unfavorable valid result is never retried.
12. Run repository tests, vet, scoped race tests, and `git diff --check`, then
    commit the result before starting the next phase.

## Gate 0: active-causal replay recovery

### Goal

Recover the already-trained Phase 2 result without changing any semantic or
empirical decision and complete its unused validation and locked panels.

### Fixed facts

E3, R3, the selected acquisition rule, training digest, bundle digest, v3/v4/v5
receipts, and every report/fixture/scoring identity are immutable. V5 failed
after its worker exited, during a deliberately generic metadata-only cache
audit. The audit internally inspected cache metadata and exposed exactly one
bit: its predicate failed. Output equality is unknown. No cache path, name,
count, size, digest, content, metadata class, or replay-output fact was exposed
to the operator.

### Diagnostic guidelines

- Never rerun the v5 protected operator.
- Before any diagnostic execution of the E3 worker, commit a diagnostic
  amendment freezing one synthetic or development-only input, exactly one run,
  the disclosed metadata schema, candidate hypotheses, an immutable decision
  table, and terminal rules. Do not read R3 output or protected fixtures as a
  diagnostic oracle.
- The diagnostic amendment receives the same three independent acceptances as
  a phase plan. It binds v5, X5, E3, the exact input, executable, environment,
  schema, hypotheses, and decision-table digest.
- That one diagnostic may exercise the actual E3 worker binary and post-exit
  audit mechanics in a new disposable root without minting protected replay
  authority. A separate exclusive diagnostic `started` receipt is durably
  created before worker start and terminally persisted before return; collision
  makes a second invocation impossible. The canonical receipt records and
  verifies the accepted diagnostic-plan and implementation commits plus the
  complete frozen binding tuple from the preceding bullet. It is not an
  empirical panel receipt.
- The diagnostic may disclose only the preregistered normalized class needed
  to distinguish the fixed hypotheses. Its implementation may internally read
  only the raw metadata required to compute that class, but may not expose file
  contents, paths, names, counts, sizes, or digests. Its result may not be used
  to add a new cache type, raise a cap, or choose another diagnostic.
- The decision table must terminate as either a proved implementation defect
  with one predetermined correction, a valid audit rejection, or
  `diagnosis-inconclusive`. The latter two leave Phase 2 `invalid` and end the
  program; no further diagnostic or recovery version is authorized.
- Preserve exact commands, inputs, output, the diagnostic receipt, and proof
  that no protected empirical-attempt receipt was created.

### Recovery implementation guidelines

Only the decision-table branch for a proved implementation defect authorizes
one recovery version. Its phase-specific amendment must bind the new plan,
implementation candidate, E3, R3, immutable training and bundle identities,
exact environment, and immutable v5 receipt digest into a new receipt version,
and authorize the predetermined smallest correction. It must specify source
topology, AST/body floors, pre-receipt tests, failure persistence, and one-shot
behavior. Three reviewers must accept both plan and implementation.

Only after a clean new candidate passes non-attempt preflight may the three R3
constants be inserted and the new replay run once. Its `started` receipt is
durably persisted before protected evidence access, and terminal success or
failure is durably persisted before any error returns. Success is followed by
a constants-only commit, synchronous replay-record verification, one
mechanically valid validation attempt, and only then locked evaluation under
the already accepted Phase 2 rules. An unfavorable but mechanically valid
validation result is preserved and classified under those rules and does not
authorize a recovery retry. Failure is preserved as another invalid version;
that operator is never rerun, and Gate 0 authorizes no later recovery version.

### Completion gate

Gate 0 completes only if Phase 2 has a canonical successful replay receipt,
mechanically valid validation and locked reports, immutable predecessor
receipts, a documented result classification, and all normal verification
green. A technical replay success alone is insufficient.

## Phase 3: resource-constrained deployment scheduling

Research anchor: Kolisch and Sprecher's
[PSPLIB](https://doi.org/10.1016/S0377-2217(96)00170-1).

### Research question and objects

Can Nous learn transferable priority-rule programs that construct feasible,
low-makespan schedules under a budget too small to enumerate the rule or
schedule space?

Objects are bounded tasks, durations, precedence edges, renewable resource
demands and capacities, maintenance windows, incompatibilities, schedules,
resource profiles, priority-rule ASTs, and evaluation transcripts. Learned
objects are priority rules and refinement heuristics, never memorized held-out
schedules.

### Implementation guidelines

- Preserve the grammar, bounds, cohorts, baselines, endpoint, and non-claims in
  Phase 3 of the accepted roadmap unless its specific plan narrows them.
- Production Go owns canonical instances, feature evaluation for one task, one
  explicit priority-rule evaluation, deterministic serial schedule generation,
  schedule validation, and loss. Tiny exact branch-and-bound belongs only to
  the isolated oracle/baseline layer.
- CUE heuristics construct rule ASTs one hole at a time, request evaluations,
  retain feasible and infeasible results, assign contextual credit, and freeze
  selected tied rules before held-out scheduling.
- Use `budgeted-transcript`; materialize every refinement and rule/schedule
  evaluation charge. Random, beam, no-credit, and credit-guided policies use
  the same grammar, information rights, and total semantic-work ceiling under
  the preregistered policy-neutral ledger.
- Include shortest/longest duration, critical path, successor count, random AST
  search, and validation-guided beam search baselines.
- Tiny instances receive exact optima; main instances receive independent
  feasibility validation and optimality lower bounds, not an infeasible exact
  oracle claim.

### Acceptance

Mechanical success requires valid schedules, exact tiny results, honest subset
accounting, frozen-rule transfer, and oracle agreement. A positive claim is
authorized only if the frozen credit-guided learner improves paired locked mean
loss over equal-budget beam search by the preregistered effect, test, and
interval while meeting the frozen feasibility noninferiority margin. Otherwise
the mechanically valid phase is `valid-null`. Production rollout authority and
large-instance optimality remain out of scope.

## Phase 4: execution-trace invariant discovery

Research anchor: Ernst et al.'s
[Daikon system](https://homes.cs.washington.edu/~mernst/pubs/daikon-tool-scp2007-abstract.html).

### Research question and objects

Can Nous invent useful relational and temporal invariants, distinguish stable
properties from accidental trace correlations, and learn probes that falsify
weak conjectures?

Objects are bounded typed events, traces, expression ASTs, guards, derived
variables, invariant candidates, probes, counterexamples, implication links,
and status transitions. `observed-supported`, `falsified`, and
`proved-on-bounded-model` are distinct states.

### Implementation guidelines

- Preserve the canonical-class universe, bounds, cohorts, baselines, macro-F1
  endpoint, and truth-status rules in Phase 4 of the accepted roadmap.
- Production Go evaluates one explicit expression, invariant, or probe,
  canonicalizes one AST, and checks one implication on one explicit bounded
  witness. Only the isolated oracle enumerates small-machine executions or
  proves bounded implication universally.
- CUE heuristics compose invariants, derive guarded variants after failures,
  propose one probe at a time, retain the exact falsifying location, and learn
  bounded probe-scoring programs from training machines.
- Use `budgeted-transcript` for candidate/probe selection. Initial evidence,
  legal probes, cost, and hidden machine are shared, but every policy owns its
  transcript and cache.
- Compare fixed templates, passive enumeration, random probes, and a
  boundary/coverage-directed conventional policy.
- Require a reusable derived variable or conditional invariant and compare it
  with no-artifact and equal-expressivity expanded forms.

### Acceptance

Mechanical success requires exact semantic-class truth, calibrated statuses,
valid counterexamples, and conserved probe budgets. A positive claim is
authorized only if locked macro F1 improves over the frozen boundary/coverage
policy by the preregistered effect, test, and interval at the same probe cost.
Otherwise the mechanically valid phase is `valid-null`. No claim of static
proof or arbitrary-program correctness is allowed.

## Phase 5: black-box service model learning

Research anchor: Isberner and Steffen's
[counterexample-analysis framework](https://proceedings.mlr.press/v34/isberner14a.html).

### Research question and objects

Can Nous actively identify a minimal deterministic service model by choosing
queries and counterexample-processing strategies, then reuse those strategies
across hidden services?

Objects are bounded Mealy machines, membership/equivalence queries,
observation-table rows and suffixes, access sequences, counterexamples,
hypothesis states/transitions, and typed decision programs.

### Implementation guidelines

- Preserve Phase 5's machine, query-length, product-bound, transcript, cohort,
  baseline, and symbol-budget contracts.
- The hidden service and shortest-counterexample teacher live only in the
  independent oracle. Production sees returned query results and
  counterexamples, never target states.
- Pure Go parses and executes one explicit machine/query and validates one
  table or hypothesis operation. CUE heuristics choose one legal membership or
  equivalence action and one typed repair/processor action at a time.
- Use `budgeted-transcript`. Count membership queries, equivalence queries,
  total symbols, table growth, counterexample processing, and duplicate cache
  behavior separately.
- Preserve the roadmap's scalar primary cost exactly: membership-query symbols
  plus returned counterexample symbols plus 16 per equivalence query;
  budget-exhausted episodes score ceiling plus one.
- Compare conventional linear, binary, and discrimination-tree processors plus
  random and static strategies under the same teacher and symbol budget.
- Freeze reusable query/processor decision programs before held-out services;
  include no-artifact, inlined, and independently recomputed controls.

### Acceptance

Mechanical success requires an oracle-equivalent minimal promoted machine or a
correct budget-exhausted terminal, exact transcript replay, and query-bound
proofs. A positive claim is authorized only if the frozen decision program
reduces the roadmap's scalar primary semantic cost relative to binary-split L*
by the preregistered effect, test, and interval while meeting the frozen exact-
recovery-rate gate. Otherwise the mechanically valid phase is `valid-null`.
Real service conformance and noisy protocols are deferred.

## Phase 6: spatial transformation puzzles

Research anchor: Chollet's
[Abstraction and Reasoning Corpus](https://arxiv.org/abs/1911.01547).

### Research question and objects

Can Nous invent reusable spatial concepts and transformation programs from a
few examples and transfer them to structurally related held-out grids?

Objects are bounded colored grids, components, masks, bounding boxes,
symmetries, relative positions, transformation ASTs, invented subprograms,
applications, and exact-output evidence.

### Implementation guidelines

- Preserve Phase 6's finite grammar, task-family separation, library-cost,
  baseline, and exact-grid endpoint rules.
- Pure Go owns one explicit grid operation, component extraction under a
  declared connectivity rule, AST canonicalization, and bounded execution.
  It may not infer a whole program or inspect all examples while refining.
- CUE heuristics construct transformation ASTs, invent named semantic
  subprograms, retain failed applications, and freeze a library before target
  tasks.
- Use `budgeted-transcript`; charge refinement, execution, normalization,
  invention, call sites, and library amortization.
- Compare random enumeration, beam/MDL program synthesis, no-library,
  task-local invention, inlined-library, and frozen shared-library policies.
- Split task families structurally, not by recoloring aliases, and require
  beneficial, neutral, harmful, and no-solution cohorts.

### Acceptance

Mechanical success requires exact execution/oracle agreement, no target
leakage, honest library use, and complete budget transcripts. The primary
endpoint remains the roadmap's lexicographic exact held-out accuracy, then
charged semantic work among equally accurate policies. A positive claim is
authorized only if the learned object-library policy preserves locked exact
accuracy and reduces primary semantic work relative to the equal-expressivity
inlined-fragment baseline by the preregistered effect, test, and interval.
Otherwise the mechanically valid phase is `valid-null`. Beam/MDL results are
secondary. General ARC performance or visual intelligence is not claimed.

## Phase 7: workflow and process-model discovery

Research anchor: van der Aalst, Weijters, and Maruster's
[workflow-mining work](https://doi.org/10.1109/TKDE.2004.47).

### Research question and objects

Can Nous infer explicit sequence, choice, and concurrency concepts from event
logs and construct a compact process model that both replays observed behavior
and rejects unsupported behavior?

Objects are bounded case traces, activities, directly-follows relations,
causal/parallel/exclusive relation conjectures, process-tree ASTs, bounded
workflow nets, replay evidence, and counterexample traces. V1 admits sequence,
exclusive choice, and parallel composition. Loops, timestamps, noise, and
cross-case resources are deferred.

Roadmap ceilings are six activities, 32 observed traces of length at most
eight, 15 process-tree nodes, 20,000 normalized candidates, 10,000 evaluated
models, 512 combined frozen positive and negative conformance traces per
fixture, 5,000 engine cycles, 250,000 attributed units, and a 24 MiB report.
The worst-case main-instance conformance ledger is therefore 5,120,000 bounded
trace/model executions per policy; exhaustive language comparison is
restricted to tiny oracle fixtures and its separately frozen ceiling.

### Implementation guidelines

- The phase-specific plan must freeze a finite process-tree grammar, model
  normalization, soundness rules, trace language bound, incomplete-log
  generator, negative-trace construction independent of production, repeated-
  label policy, and structural AST normalization. Production canonical identity
  is syntactic; behavioral-equivalence classes belong only to the oracle and
  report layer.
- Production Go validates one log/model, executes one trace against one model,
  evaluates one declared statistic or conjecture for one explicit activity
  pair, and checks one model-local soundness obligation. A full footprint,
  complete miner, or model choice belongs only to isolated baselines/oracles.
- CUE heuristics propose relation concepts, compose process trees one node at a
  time, retain fitting and rejecting traces, derive counterexamples, and
  promote complete tied minimal models within the evaluated subset.
- Use `budgeted-transcript` unless the phase plan proves the entire normalized
  model grammar is small enough for a meaningful `exhaustive-matrix` trial.
- Compare the alpha-family footprint construction, a bounded inductive
  process-tree baseline, random grammar search, and a direct-follows-only
  control under matched semantic work.
- Cohorts must separate pure sequence, exclusive choice, true concurrency,
  ambiguous incomplete logs, and out-of-language/no-solution cases. Aliases and
  case order may not affect identity.
- Terminal states are `promoted`, `ambiguous`, `no-model-within-grammar`, and
  `budget-exhausted`; every state receives frozen precision/recall/F1 sentinel
  handling and is mechanically valid when its transcript is complete. A
  promoted complete tied set takes precedence; exhausted work is
  `budget-exhausted`; otherwise the frozen ambiguity and grammar predicates
  distinguish the two remaining states.
- Every promoted model must replay every observed trace. A non-fitting model is
  an evaluated candidate, never a promotable result.

### Acceptance

Mechanical success requires sound promoted models, exact bounded-language
replay, independent conformance scoring, and honest terminal reporting. A
positive claim is authorized only if held-out positive/negative trace F1
improves by the preregistered effect, test, and interval over the bounded
inductive process-tree baseline frozen before validation, at matched work.
Otherwise the mechanically valid phase is `valid-null`. Production process
mining, noisy logs, and BPMN generation remain out of scope.

## Phase 8: failure-inducing input and change minimization

Research anchor: Zeller and Hildebrandt's
[Delta Debugging](https://www.st.cs.uni-saarland.de/papers/tse2002/).

### Research question and objects

Can Nous learn reusable decomposition and probe-selection heuristics that find
smaller failure-inducing configurations, traces, or change sets with fewer
costly executions than conventional delta debugging?

Objects are bounded structured artifacts, semantic atoms and dependency edges,
subsets or subtrees, tri-state test outcomes (`pass`, `fail`, `unresolved`),
probes, cached results, one-minimal witnesses, and decision programs. Fixtures
represent synthetic Kubernetes/Terraform-like records, command traces, and
patch sets without invoking real tools.

Roadmap ceilings are 24 atoms, 48 directed dependency edges, 512 hidden-test
probes, 64 cost units per hidden test, and 32,768 hidden-test cost units per
policy, seven nodes per decision program, 20,000 normalized programs, 64
evaluated programs over at most 16
training fixtures, 5,000 engine cycles, 200,000 attributed units, and a 16 MiB
report. The grammar freezes eight state/evidence features, partition and
complement actions, lexicographic/weighted composition, and integer constants
in `{-2,-1,0,1,2}`. Tiny oracle fixtures use at most 12 atoms and may exhaust
all 4,096 subsets.

Per probe, the ledger caps and charges 24 atom visits, 48 edge visits, 24
partition visits, 24 closure/canonicalization visits, one duplicate/cache
operation, and the declared hidden-test cost. Main-policy structural work is
therefore at most 61,952 charged operations plus 32,768 hidden-test cost units;
program training has a separate ceiling of 1,024 episodes, 128 probes per
episode, 131,072 total probes, 15,859,712 structural operations, and 8,388,608
hidden-test cost units.

### Implementation guidelines

- Freeze dependency-aware sets as the sole v1 artifact algebra. The plan must
  define dependency direction, canonical post-closure identity, whether the
  failure predicate is monotone, the exact tri-state `ddmin` variant and
  unresolved/progress rules, and whether `one-minimal` means no single legal
  atom deletion preserves failure. On tiny exhaustive fixtures, the
  `co-minimal set` is the complete set of inclusion-minimal failing dependency-
  closed subsets; it is not the set of minimum-cardinality causes.
- A hidden deterministic test function lives only in the oracle. Production
  receives a tri-state response to one explicit dependency-closed probe.
- Production Go validates one artifact/probe, computes dependency closure for
  one explicit candidate probe, and applies one explicit deletion. CUE chooses
  a partition, but each member and complement becomes a separate eligible,
  materialized, charged probe. One-minimality and complete `ddmin` live only in
  the independent checker/baseline.
- CUE heuristics choose granularity, partitions, complements, and dependency-
  aware probes one action at a time; failures credit retained atoms and useful
  decompositions while unresolved results remain explicit evidence.
- Use `budgeted-transcript`. Count test executions and weighted test cost,
  cache hits, duplicate probes, closure additions, and terminal reason.
- Compare canonical `ddmin`, greedy single deletion, random deletion, and a
  static dependency-aware policy. Include irrelevant, interacting, multiple-
  minimal, unresolved, and no-reducible-witness cohorts.
- In the no-reducible-witness cohort, the full original failing artifact is a
  verified `one-minimal` terminal when it is inclusion-minimal.
- Terminal precedence is mutually exclusive: verified `one-minimal` wins
  first; otherwise exhausted probe or test cost is `budget-exhausted` with the
  best reproducer; otherwise an unresolved response that prevents the frozen
  progress rule is `unresolved`; otherwise the terminal is
  `reproducing-not-minimal`. Every state is mechanically valid with a complete
  transcript and receives frozen witness-size and test-cost sentinels.

### Acceptance

Mechanical success requires exact tri-state transcript replay, dependency
validity, no hidden-test leakage, and correct conditional verification of any
promoted one-minimal witness. A positive claim is authorized only if locked
test cost improves over frozen canonical `ddmin` by the preregistered effect,
test, and interval while meeting the frozen witness-size noninferiority margin.
Otherwise the mechanically valid phase is `valid-null`. All co-minimal
witnesses are reported only on tiny exhaustive fixtures. Real command
execution and claims of globally minimum causes are deferred.

## Phase 9: concurrent-history anomaly discovery

Research anchor: Herlihy and Wing's
[linearizability condition](https://www.cs.cmu.edu/~wing/publications/HerlihyWing90.pdf).

### Research question and objects

Can Nous learn commutativity, independence, and schedule-selection concepts
that expose and minimize non-linearizable concurrent histories under a bounded
execution budget?

Objects are bounded sequential specifications, invocation/response events,
operation intervals, schedules, histories, happens-before edges,
linearization witnesses, counterexamples, commutativity conjectures, and
schedule-priority programs. V1 covers deterministic registers, counters, and
queues with bounded values and operations. Liveness and weak-memory semantics
are deferred.

Roadmap ceilings are two threads, eight completed operations, 16 scheduled
invocation/response steps, 32 abstract SUT states, 4,096 schedules per policy,
40,320 candidate linearization orders per complete tiny history, seven nodes
per schedule-priority program, 20,000 normalized programs, 64 evaluated
programs over at most 16 training objects, 5,000 engine cycles, 250,000
attributed units, and a 32 MiB report. The grammar freezes eight schedule-
prefix features, lexicographic/weighted composition, and integer constants in
`{-2,-1,0,1,2}`.

The ledger caps production at 65,536 executed steps, 2,000,000 checker nodes,
and 16,000,000 checker operation visits per policy. Each history check consumes
that shared ceiling and either returns `linearizable`, `violation`, or
`checker-exhausted`; the last terminates the policy with its frozen sentinel.
Tiny exhaustive checks still obey the same aggregate ceiling. Program training
is capped at 1,024 episodes, each with 64 schedules, 1,024 executed steps,
8,192 checker nodes, and 65,536 checker operation visits: at most 65,536
schedules, 1,048,576 steps, 8,388,608 checker nodes, and 67,108,864 checker
operation visits across training.

### Implementation guidelines

- Freeze operation/state/history caps, a deterministic system-under-test
  transition model, schedule grammar, and a bounded complete checker for tiny
  histories.
- The system under test and independent linearizability checker must not import
  production scheduling or commutativity helpers. The checker is also a sibling
  of, and may not import, the SUT implementation; it reimplements the sequential
  specification used for linearization.
- Production Go proposes and validates one explicit schedule extension and
  evaluates one explicit commutativity witness consisting of one descriptor-
  derived sequential-spec state and an ordered operation pair; this is never
  hidden SUT state. Only the isolated driver executes the hidden transition and
  returns its observable response. Universal bounded
  commutativity and linearization search belong only to the independent
  checker; production may not globally search schedules or see hidden state.
- CUE heuristics propose one schedule extension, infer independence from
  evidence, retain failed commutations and linearization counterexamples, and
  learn schedule-priority programs across training objects.
- Use `budgeted-transcript`; count schedules, transitions, history-check work,
  partial-order reductions, duplicate prefixes, and minimization probes.
- Compare random scheduling, depth-first exhaustive prefixes, a conventional
  dynamic partial-order reduction baseline, and static conflict heuristics.
- Include linearizable, buggy, rare-interleaving, misleading-commutativity,
  alias, and no-violation-within-bound cohorts.
- Evidence-derived independence may prioritize schedules but never prune them
  in v1. No commutativity-proof oracle is exposed to production. The plan
  freezes the complete bounded schedule/prefix coverage universe and metric.
- Terminal states are `violation-found`, `no-violation-within-bound`, and
  `budget-exhausted`, plus `checker-exhausted`; all are valid with a complete
  transcript. Counterexample minimality is required only after a violation is
  found. `No-violation-within-bound` requires complete coverage of the frozen
  bounded universe; an incomplete run is exhausted, not negative proof.

### Acceptance

Mechanical success requires independently verified histories, correct terminal
scoring, and conditionally minimal found counterexamples under the declared
condition. A positive claim is authorized only if the learned policy uses
fewer charged executions to find a violation than the frozen conventional
dynamic partial-order reduction baseline, by the preregistered effect, test,
and interval while meeting the frozen coverage noninferiority margin on non-
buggy cases. Otherwise the mechanically valid phase is `valid-null`. Absence
of a found violation is not a proof of concurrency correctness.

## Phase 10: equality-saturation strategy learning

Research anchor: Willsey et al.'s
[egg](https://doi.org/10.1145/3434304).

### Research question and objects

Can Nous learn transferable rewrite-selection and e-class prioritization
heuristics that extract lower-cost equivalent expressions under a saturation
budget?

Objects are bounded typed expression ASTs, rewrite schemas, substitutions,
e-nodes, e-classes, congruence links, analyses, rewrite applications, extracted
expressions, and strategy programs. The learned object is a scheduling
strategy, not the e-graph implementation or a fixture-specific optimized term.

Roadmap ceilings are 15 nodes per input AST, 24 independently validated rewrite
schemas, 20,000 e-nodes, 10,000 e-classes, 100,000 explicit eligible matches,
25,000 individually charged applications, 25,000 rebuild steps, seven nodes per
strategy program, 20,000 normalized programs, 64 evaluated programs over at
most 16 training expressions, 5,000 engine cycles, 300,000 attributed units,
and a 32 MiB report. E-nodes have arity at most two, so a graph has at most
40,000 child edges. The strategy grammar freezes eight graph/frontier
features, lexicographic/weighted composition, and integer constants in
`{-2,-1,0,1,2}`. Program training is capped at 1,024 episodes, each with 2,000
e-nodes, 1,000 e-classes, 5,000 cursor inspections and match attempts, 1,000
applications, rebuilds, analyses, and rollbacks, and 3,000 extraction visits.
Across training that is at most 2,048,000 e-nodes, 1,024,000 e-classes,
5,120,000 cursor inspections, 5,120,000 match attempts, 1,024,000 steps in
each application/rebuild/analysis/rollback field, and 3,072,000 extraction
visits.

A run terminates when any ceiling is reached, but only at a congruent committed
graph. Frontier cursor inspections, successful and failed matches,
applications, rebuild cascades, analyses, rollbacks, and terminal extraction
have separate monotonically conserved ledger fields. Their roadmap maxima are
100,000 cursor inspections, 100,000 match attempts, 25,000 applications,
25,000 rebuild steps, 25,000 analysis steps, 25,000 rollback steps, and 30,000
terminal extraction visits per policy.

### Implementation guidelines

- Freeze a small typed expression language, sound rewrite set, exact semantic
  interpreter, e-graph/work bounds, extraction cost, and structurally separated
  training/locked expression families.
- Production Go may validate one rewrite, enumerate one explicit schema/e-class
  match into the eligible frontier, apply one explicit substitution at one
  e-class, restore congruence for one charged step, and run one declared local
  analysis. It may not rank the frontier, saturate to completion, or choose the
  next rewrite/e-class. A deterministic cursor orders only by canonical schema,
  e-class, e-node, and substitution keys; every inspected failed or successful
  match is charged, and every successful substitution is a distinct versioned
  frontier artifact.
- CUE heuristics propose exactly one eligible rewrite application at a time,
  materialize its effect, learn strategy ASTs from credit, and freeze them
  before held-out expressions. Apply-all and opaque batch actions are forbidden.
- A fixed least-cost extractor runs only after the policy terminates. It is a
  charged terminal decoder used for scoring, is inaccessible to strategy
  selection, and has no effect on the explored graph. It uses a frozen
  deterministic relaxation/worklist algorithm over canonical e-class/e-node
  order. It either proves and returns the least-cost reachable finite term
  within the extraction ledger or returns `extractor-exhausted`; a partial
  incumbent is never reported as least-cost and receives the frozen sentinel.
- Each application is transactional from the last congruent graph. Tentative
  changes use a copy-on-write journal. Before starting, the operation reserves
  rollback work equal to its maximum journal footprint; insufficient reserve
  terminates before mutation. It commits atomically only if its rebuild cascade
  reaches congruence within the remaining rebuild budget. Otherwise every
  journaled node/edge discard visit consumes the reserve before termination.
  Extraction therefore never sees a partially rebuilt graph.
- Use `budgeted-transcript`; charge matching, rebuilding, analysis, nodes,
  classes, applications, extraction, and duplicate suppression.
- Compare round-robin equality saturation, random frontier selection, greedy
  immediate cost reduction, and a conventional analysis-guided scheduler.
- Include helpful, neutral, explosive, cyclic, misleading-local-improvement,
  and no-improvement cohorts. All extracted terms receive independent bounded
  semantic equivalence checks.
- Rewrite soundness is established independently over the declared finite
  typed domain or by a separately checked proof rule; finite sample agreement
  is insufficient. Operational terminals are `frontier-empty` and
  `budget-exhausted`; terminal extraction separately classifies the outcome as
  `improved`, `no-improvement`, or `extractor-exhausted`. Every combination is
  mechanically valid when its graph and ledger verify.

### Acceptance

Mechanical success requires sound rewrites, congruence integrity, exact work
accounting, equivalent extracted terms, and frozen-strategy transfer. A
positive claim is authorized only if locked extracted cost improves over the
frozen conventional analysis-guided scheduler by the preregistered effect,
test, and interval at matched work. Otherwise the mechanically valid phase is
`valid-null`. General compiler optimization, unbounded saturation, and proof
of global optimality are not claimed.

## Cross-phase uniqueness contract

Each added phase must demonstrate a capability not supplied by renaming an
earlier object. Its phase plan freezes this distinguishing artifact, algorithm,
metric, and causal ablation:

- Phase 7 learns normalized branching/concurrent process trees from incomplete
  multi-case logs. Language conformance over sequence, choice, and parallel
  structure distinguishes it from Phase 4 formula truth and Phase 5 Mealy
  equivalence. Its ablation removes or inlines learned relation/process
  fragments while preserving the model grammar.
- Phase 8 learns a decomposition and probe-selection program over dependency-
  closed subsets. Witness reduction under tri-state costly tests distinguishes
  it from Phase 2 hidden-cause identification. Its ablation removes the
  decision program while preserving the legal probes and `ddmin` expressivity.
- Phase 9 learns a schedule-priority program plus defeasible commutativity
  concepts. Real-time partial-order history exploration distinguishes it from
  Phase 3 precedence scheduling. Its ablation removes learned priority and
  commutativity artifacts without reducing the schedule universe.
- Phase 10 learns a rewrite/e-class scheduling strategy. Navigation of
  equivalence classes under graph-growth bounds distinguishes it from earlier
  construction over independent ASTs. Its ablation removes the scheduler while
  preserving rewrites, frontier, work ceiling, and terminal extractor.

If a phase-specific design cannot maintain its row, it must be amended or
removed before implementation.

## Program completion

The program is complete only when every gate and phase has a preserved accepted
plan, implementation candidate, locked result, status classification, review
record, and verification evidence. A mixture of valid-positive and valid-null
outcomes is acceptable. Any remaining invalid phase means the program is not
complete.

At program end, a cross-phase audit must distinguish four kinds of evidence:

- domain hosting: Nous can represent and execute the vocabulary;
- bounded discovery: ordinary heuristics construct an evidence-supported
  artifact;
- search advantage: a frozen learned policy beats matched baselines;
- reusable abstraction: a learned artifact causally improves later tasks over
  equal-expressivity controls.

No aggregate claim may upgrade a weaker kind of evidence into a stronger one.
