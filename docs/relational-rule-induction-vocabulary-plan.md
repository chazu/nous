# Relational rule-induction vocabulary plan

## Status and decision

Accepted Phase 1 implementation plan, revision 5.

Adversarial review record, 2026-08-07:

- Chandrasekhar, architecture and current-engine integration: `ACCEPT`;
- Lovelace, Horn semantics, search, and feasibility: `ACCEPT`; and
- Harvey, experimental validity and leakage resistance: `ACCEPT`.

Five revisions resolved temporal causality, the factored conventional baseline,
stage/profile identity, queue rights, generator degrees of freedom, exact work
and artifact bounds, transcript accounting, and inference gates. The accepted
revision is the version contained in the plan commit.

This phase tests one narrow capability: whether Nous can construct a normalized
invented Horn predicate from examples, execute it as a first-class shared
library definition in two target theories, and reduce charged semantic work
relative to an equal-expressivity task-local learner. It does not test general
inductive logic programming.

The implementation is confined to `domains/ruleinduction`,
`internal/vocab/ruleinduction`, `internal/dsl/builtins_ruleinduction.go`,
`internal/ruleinductionoracle`, `internal/ruleinductionexp`, a small trial
command, tests, and documentation. It changes no engine, agenda, mutation,
common-domain, math-domain, VM, or global credit behavior.

## Preregistered manifest

The mechanically parsed manifest is the JSON object below. Source constants and
the emitted report must reproduce it byte for byte.

```json
{
  "experiment_version": "rule-induction/v1",
  "generator_version": "closure-pairs/v1",
  "grammar_version": "binary-horn-three-relations/v1",
  "oracle_version": "independent-fixed-point/v1",
  "cost_version": "semantic-work/v1",
  "report_version": "rule-induction-report/v1",
  "baseline_version": "factored-direct-lff/v1",
  "statistics_version": "paired-resampling/v1",
  "queue_version": "policy-queues/v1",
  "cache_version": "semantic-definition-stage-cache/v1",
  "integrity_contract": "budgeted-transcript",
  "development_seeds": {"start": 11001, "count": 16, "step": 1},
  "training_seeds": {"start": 21001, "count": 64, "step": 1},
  "validation_seeds": {"start": 31001, "count": 32, "step": 1},
  "locked_seeds": {"start": 41001, "count": 64, "step": 1},
  "cohort_assignment": "index-mod-16:0-9-beneficial,10-12-neutral,13-15-harmful",
  "locked_cohorts": {"beneficial": 40, "neutral": 12, "harmful": 12},
  "constants": 8,
  "background_predicates": 3,
  "target_predicates": 2,
  "invented_predicates": 1,
  "variables": ["X", "Y", "Z"],
  "metarules": ["identity", "tailrec", "invented-projection"],
  "clauses_per_theory": 4,
  "body_literals_per_clause": 2,
  "normalized_definitions_per_task": 15,
  "normalized_joint_theories": 240,
  "complete_candidate_evaluation_cap": 31,
  "fixed_point_step_cap": 1000000,
  "total_semantic_work_cap": 500000,
  "engine_cycle_cap": 1000,
  "attributed_unit_cap": 100000,
  "report_byte_cap": 8388608,
  "locked_accuracy_gate": 1.0,
  "minimum_primary_reduction": 0.15,
  "minimum_recomputed_reduction": 0.15,
  "minimum_inlined_reduction": 0.05,
  "maximum_harmful_ratio": 2.0,
  "minimum_beneficial_stage2_candidate_reduction": 0.25,
  "policy_queues": {
    "naive-direct": "canonical-code",
    "uniform-random": "uniform-order-stage-<1|2>",
    "all-causal-policies": "causal-order-stage-<1|2>"
  },
  "cache_policy": "stage-local-semantic-key;only-shared-library-carries-frozen-I",
  "alpha": 0.05,
  "confidence_interval": "paired-bootstrap-two-sided-95",
  "paired_test": "paired-randomization-two-sided",
  "bootstrap_replicates": 10000,
  "randomization_replicates": 10000,
  "bootstrap_indices_zero_based": [249, 9749],
  "contrast_seed_rule": "rule-induction/v1|locked|<direct|recomputed|inlined|beneficial-candidates>|<randomization|bootstrap>",
  "tie_policy": "first-exact-frozen-queue-report-evaluated-exact-ties",
  "budget_exhaustion_score": 500001,
  "audit_work_in_primary": false,
  "mutation_enabled": false
}
```

The seed notation denotes the exact finite arithmetic sequence
`start + i*step` for `0 <= i < count`; panels never overlap. Locked seeds are
not executed until the implementation-candidate commit exists. Any subsequent
change to code, fixture filtering, grammar, costs, metrics, or thresholds creates
`v2` and preserves the v1 report.

## Logical language and exact grammar

Every fixture declares eight opaque constants, three opaque binary background
predicates, two opaque binary targets, and one allocated invented predicate.
Facts and examples are sorted duplicate-free ground atoms. Positive and negative
examples are disjoint. The negative set is explicit; absence is not negation.

Clauses are safe, function-free, and use only `X`, `Y`, and `Z`. V1 admits these
normalized forms:

```text
identity:              P(X,Y) :- Q(X,Y)
tailrec:               P(X,Y) :- Q(X,Z), P(Z,Y)
invented-projection:   T(X,Y) :- I(X,Y)
```

`Q` ranges over exactly the three descriptor-declared background predicates.
For any head `P`, identity contributes three clauses and tail recursion
contributes three clauses. A two-clause definition is an unordered pair of
distinct clauses, giving `C(6,2) = 15` normalized definitions.

A shared joint theory contains one two-clause definition of `I` and one
projection clause for each target: 15 possible joint theories. A task-local
joint theory contains an independent two-clause definition for each target:
`15 * 15 = 225` possible combinations. Thus the declarative joint-theory
universe contains 240 theories, but no policy searches the 225 combinations as
a Cartesian product.

The experiment has two temporal stages. Stage 1 exposes only target 1. Every
policy searches at most 15 definitions and freezes its selected definition.
Only then does the driver add target 2 examples and schedule the ordinary
stage-2 task. The shared policy first evaluates the single projection through
the frozen `I`; it may not change or replace `I`. If projection fails, it
searches at most 15 task-local definitions. Conventional local policies search
15 definitions independently in each stage. The maximum is therefore 31
complete candidate evaluations, not 240. Shared and local languages can both
express every locked target.

Alpha-renaming, body ordering, clause ordering, and descriptor aliases do not
change identity. Canonical codes contain only descriptor positions and
metarule positions, never unit names. Duplicate clauses, unsafe variables,
undeclared predicates, recursive projections, mutual recursion, negation,
functions, constants in clauses, and more than one invented predicate are
invalid.

## Semantics

Evaluation uses a frozen deterministic work-list algorithm. Identity clauses
are scanned once in canonical order. Every newly inserted pair `(z,y)` is
dequeued once; for each recursive clause, constants `x` are scanned in
descriptor order and `Q(x,z)` may insert `(x,y)`. Duplicate insertions are
charged but not queued. This computes the least fixed point. The maximum ground
relation has 64 atoms, so every valid theory terminates.

Stage 1 candidates contain a two-clause definition and a projection to target
1. The first training-exact candidate in the frozen queue is selected. An exact
candidate entails every positive and no negative example. The selection barrier
proves that every earlier queue element was explicitly executed, structurally
pruned by a stored sound constraint, or ineligible for a recorded semantic reason. Exact
ties among the evaluated subset are retained; the deployment identity is the
first exact queue element and cannot change after the barrier.

For `shared-library`, stage 1's selected definition is normalized under the
fresh allocated predicate `I`, materialized, and frozen. Stage 2 is not present
in the store before this freeze. The driver records the canonical store digest,
adds only target-2 training examples and one descriptor-declared continuation
task, and resumes. No stage-2 action may alter `I`, its definition, or its
stage-1 evidence. All policies cross the same stage boundary and receive the
same target-2 examples at that point.

Held-out facts and labels are never inserted into the training store. After a
theory is frozen, the experiment driver evaluates it on an independently
generated held-out graph and the oracle scores all 128 target/constant-pair
predictions. Incorrect predictions are empirical failures, not integrity
failures.

## Fixture generator and cohorts

Every random stream is derived independently as the first 128 bits of
`SHA-256("closure-pairs/v1|<panel>|<seed>|<stream>|<attempt>")`, interpreted
as two big-endian `uint64` values and passed to Go's `math/rand/v2.NewPCG`.
Stream
names are `training-graph-0..2`, `heldout-graph-0..2`, `constant-aliases`,
`predicate-aliases`, `causal-order-stage-1`, `causal-order-stage-2`,
`uniform-order-stage-1`, and `uniform-order-stage-2`. Graph streams use the
accepted attempt index; alias and order streams always use attempt zero.

One graph attempt lists the 56 non-self ordered pairs lexicographically,
Fisher-Yates shuffles them, draws edge count uniformly from `[8,18]`, and takes
that prefix. The generator computes reachability, branch count, longest simple
path by exhaustive depth-first search, and closure cardinality. It accepts only
graphs with a branch, non-universal closure, and cohort-specific depth. Attempts
run from 0 through 99 in order. Rejection and exhaustion depend only on these
frozen semantic predicates; exhaustion is invalid. Held-out graphs use the same
algorithm and independent streams.

For each stage, the oracle builds examples from its 15 definitions. Atom order
is tuple order over canonical constant positions `(subjectIndex, objectIndex)`,
never opaque alias strings. It starts with the first two positive atoms and
first two negative atoms in that order. Then it visits wrong candidates in
canonical code order; if the current set does not distinguish one, it appends
the first atom in canonical tuple order on which that
candidate differs from ground truth. It stops when all wrong candidates are
distinguished. A fixture is rejected if either stage needs more than 24 examples
or lacks two labels of either polarity. This exact separator algorithm is frozen
and independent of production queue order.

The cohorts are:

- `beneficial`: both targets are the transitive closure of the same background
  relation at descriptor position 0, whose training and held-out graphs have
  longest simple-path depth in `[3,6]`. Positions 1 and 2 are independently
  generated decoys with depth in `[1,6]`.
- `neutral`: all three background relations are the same graph with longest
  simple-path depth one. Only `training-graph-0` and `heldout-graph-0` are
  sampled; their accepted edges are copied byte-for-byte to positions 1 and 2.
  Many definitions are extensionally tied, so the artifact supplies little
  identifying information to stage 2.
- `harmful`: target 1 is the closure of position 0 and target 2 the closure of
  position 1. Both graphs have depth in `[3,6]`, their closures differ, and
  position 2 is an independent depth-`[1,6]` decoy. The frozen stage-1 artifact
  must fail stage 2 before the local search.

For every beneficial and harmful attempt, the generator enumerates all 15
stage-local definitions on training facts and accepts only when the intended
identity-plus-tailrec definition is the sole training-exact semantic class for
each target. A candidate is `wrong` exactly when its 64-bit training extension
differs from the target extension. If no differing atom exists it is a semantic
tie: beneficial/harmful reject the attempt, while neutral accepts it only under
the source-to-sink family theorem that all such ties remain equivalent for every
allowed held-out graph. Mixed and pure-decoy definitions are treated identically
by this check. The separator records safe ties without adding an example and
terminates after all 15 candidates.

For every panel, cohort assignment is `index mod 16`: values 0-9 beneficial,
10-12 neutral, and 13-15 harmful. This yields locked counts 40/12/12 and freezes
development, training, and validation assignments too. Held-out data for seed
`s` uses the held-out stream names above, not an arithmetic transformation of
the training RNG. Reports are mandatory overall and by cohort.

Eight development-only no-solution controls use seeds 51001-51008 and targets
equal to the symmetric closure of a relation on graphs independently verified
to have no exact grammar theory. Every policy must terminate `no-solution`
without promotion. They do not enter the locked advantage statistic.

## Search policies and causal controls

Every policy receives identical staged facts/examples, grammar, caps, and
information rights. The five causal policies share `causal-order`;
`naive-direct` uses canonical order and `uniform-random` uses `uniform-order`.
Each owns its store, transcript, and cache.
Stage-local caches may be reused across candidates only by semantic definition
key. Only `shared-library` may carry the frozen stage-1 invented-relation cache
into stage 2; that right is the treatment under test and the avoided work is
reported. Baselines otherwise have identical cache algorithms.

The preregistered policies are:

1. `naive-direct`: two independent canonical-code 15-definition searches with
   clauses headed directly by `T` and no failure-derived pruning. Ordering is a
   controlled baseline difference.
2. `lff-direct`: the strongest conventional baseline; two independent
   15-definition searches with clauses headed directly by `T`, structural LFF,
   and the `causal-order` permutation.
3. `lff-task-local-invention`: the same `causal-order` searches, but each stage
   allocates a fresh task-local invented predicate plus projection and charges
   both; no artifact crosses the stage boundary.
4. `uniform-random`: direct clauses, no LFF, and the independent
   `uniform-order` permutation.
5. `shared-inlined`: freezes the same stage-1 definition but expands and
   recomputes it during the single stage-2 projection evaluation.
6. `shared-recomputed`: discards the stage-1 artifact after the boundary and
   performs the ordinary independent 15-definition stage-2 search.
7. `shared-library`: learns `I` from target 1 alone, freezes it, evaluates its
   projection once on target 2, and falls back to an independent local search
   only if projection fails.

The shared-first structural bias is fixed before examples are read. It can find
any of the 15 invented definitions and does not encode which background
predicate is correct. The harmful cohort measures its cost. The phase makes no
claim that Nous learned this meta-policy.

The driver constructs the candidate permutation before engine creation and
stores it in the descriptor. No heuristic calls `random-choice`, `random-int`,
or `random-subset`; consequently the engine's private default RNG state cannot
affect a trial. Production and all baselines set `MutConfig.Enabled = false`.
The mutation-enabled control uses the default seed 42, proves that no mutation
is eligible before terminal, and must emit the identical semantic report. A
mutation-enabled run is never part of the locked statistic.

`lff-direct`, `lff-task-local-invention`, `shared-library`, `shared-inlined`, and
`shared-recomputed` use the identical `causal-order` permutation in each stage.
`naive-direct` and `uniform-random` use the distinct queues stated above. Every
queue is a permutation of the same 15 canonical codes. Policies stop at the first
exact candidate or after all 15 candidate dispositions prove no solution.
Shared stage 2 evaluates the projection first and falls back immediately after
its recorded failure. A structurally pruned candidate counts as a consumed
candidate but not a semantic execution; reports distinguish both counts.
Projection failures never create structural LFF constraints because a frozen
projection and a local two-clause definition inhabit incomparable grammars;
they create ordinary failure evidence only. Thus each 15-definition search has
at most 210 constraint comparisons and the two-stage maximum remains 420.

Learning from failures retains the failed executed theory and an explicit
too-general or too-specific structural constraint. Before execution, CUE visits
constraints in creation order and compares each with the explicit candidate by
standard theory theta-subsumption over the finite `X/Y/Z` substitution set.
Too-general failures may prune only proven structural generalizations;
too-specific failures may prune only proven structural specializations. The
comparison word accepts exactly two explicit normalized theories and returns a
certificate; it never scans a candidate set. Every comparison and positive
prune is materialized and independently checked extensionally. This grammar may
offer little structural pruning; the experiment does not claim LFF itself wins.

## Ordinary-heuristic implementation

Pure package `internal/vocab/ruleinduction` owns immutable terms, atoms,
clauses, theories, normalization, safety validation, substitution, fixed-point
execution of one explicit theory, entailment, and per-operation work counters.
It imports no store, DSL, engine, experiment, or oracle package.

The partial-definition AST has two ordered clause slots. Each clause has exactly
two fields: `kind` then `backgroundPredicatePosition`; variables and literal
links are fixed by the selected metarule. The sole hole order is
`clause0.kind`, `clause0.background`, `clause1.kind`,
`clause1.background`. A refinement call binds exactly the next field. From one
root the raw tree has `1 + 2 + 6 + 12 + 36 = 57` nodes and 56 edges; normalization
rejects duplicate clauses and noncanonical clause order, leaving 15 complete
definitions. A call can never turn a root, theory slot, or empty clause into a
complete clause or multi-clause theory.

The `ruleinduction` DSL extension exposes only:

- total parsers/validators for an explicit atom, clause, partial theory, or
  complete theory;
- `ri-one-step-refinements`, which fills exactly one declared hole of one
  explicit partial theory using descriptor-declared predicates and metarules;
- execution of one explicit complete theory on explicit facts and examples,
  returning outcomes and semantic work;
- theta-subsumption and extensional comparison of two explicit objects;
- normalized semantic keys and store-aware collision-safe artifact allocation;
- `ri-ready-to-select?`, a pure read-only pre-selection verifier; and
- `ri-experiment-complete?`, a separate pure read-only post-selection verifier.

No word enumerates a clause/theory universe, sees examples while refining,
chooses a candidate, returns a frontier, prunes globally, or reads oracle data.
Every returned one-hole refinement becomes an attributed unit and incurs one
refinement charge. Source and behavioral tests call the adapter at every depth
and prove that only one field changes, every intermediate child is scheduled,
and neither a root nor a one-field node can yield a complete clause or theory.

Ordinary CUE heuristics implement these lanes:

1. create shared and local root partial theories from descriptor slots;
2. refine one hole and materialize every edge and child;
3. schedule eligible complete theories according to the selected policy;
4. execute one candidate and materialize per-example results, observations,
   aggregate evidence, and semantic cost;
5. materialize sound failure constraints and explicit prune decisions;
6. freeze an exact library definition or transition from shared to local
   fallback;
7. close the search ledger at a proposed terminal and call
   `ri-ready-to-select?` from the CUE finalization guard;
8. select the first exact queue candidate or record `no-solution`/exhaustion;
   and
9. call `ri-experiment-complete?` to validate selection and promotion.

The descriptor owns every category name, task slot, priority, cap, predicate
binding, metarule binding, and policy mode. Tests replace all of them and reorder
every case. Heuristics contain no fixture identity, predicate name, fact,
example, target clause, theory code, or expected count beyond descriptor caps.
The descriptor validator requires every declared category to exist, be pairwise
inheritance-disjoint, and be addressed by exact direct membership scans.

The trial uses two ordinary `Engine.Run` invocations and no new engine hook.
Stage 1 sets `MaxCycles=500`, seeds the descriptor's one `initialTasks` entry,
and runs with mutation disabled. Its agenda work is bounded below 154 total
phase tasks; after the `awaiting-stage-2` terminal, any remaining cycles are
inert unit focus. The driver requires an empty agenda, a valid stage-1 barrier,
and the frozen store digest. It then inserts the stage-2 corpus unit and one
continuation task and calls `Run` again with `MaxCycles=500`. `Run` resetting its
private cycle counter is expected; the driver reports and caps the sum at 1,000.
Credit, deletion, and mutation are unused, so no lifecycle is bypassed. A plain
`nous run -domain ruleinduction` never has the external corpus; it terminates
safely at `awaiting-stage-2` without fabricating target-2 evidence.

Every artifact has one immutable `experimentProfileKey` plus a
stage-specific `stageProfileKey` and `stage` integer. The experiment key commits
to manifest versions, panel, seed, policy, ordered descriptor category/task
bindings, predicate positions, metarules, caps, and candidate queues. Stage 1's
key additionally commits to its fact/example semantic digest. Stage 2 commits
to its fact/example digest, the immutable experiment key, and the accepted
stage-1 terminal/freeze digest. It does not replace or stale stage-1 identity.
Final verification requires the exact stage-1-to-stage-2 key chain and rejects
cross-stage evidence use except the declared frozen library relation.

Keys are lowercase `sha256:<hex(canonical-json(profile-v1 fields))>`. Map keys
are sorted; arrays retain semantic order; unit identities and prose are
excluded. Allocation/idempotence matches on experiment key, artifact stage, and
that stage's profile key.

## Artifact and transcript contract

Every artifact has `experimentProfileKey`, `stage`, `stageProfileKey`,
`artifactKind`, and a
collision-safe semantic key. Allocation reuses exactly one existing artifact
only when category, experiment, stage profile, semantic key, and every authoritative
slot match. Unrelated occupancy receives a deterministic `-collision-N` suffix.
Partial/conflicting or multiple attributed matches fail closed. Repeated tasks
cannot append transcript actions, consume budget, change worth, or duplicate
evidence.

The gap-free transcript records refinement,
evaluation, prune, fallback, promotion, and termination actions with sequence
number, prefix digest, charged work, remaining budgets, and artifact identities.
Transcript actions are individual attributed units. The search ledger freezes
before the pre-selection audit. Both verifiers are idempotent, read-only, and
charge their recomputation only to a separate `audit_work` report field excluded
from policy cost and selection.

`ri-ready-to-select?` validates descriptor/profile/category boundaries, exact
direct membership, the gap-free digest chain, every refinement/evaluation/
constraint/prune/fallback charge, budget conservation, candidate dispositions,
stage-boundary immutability, absence of missing/extra/stale artifacts, and the
eligibility of the proposed `identified`, `no-solution`, or `budget-exhausted`
terminal. Selection is forbidden until it returns true.
`ri-experiment-complete?` independently recomputes the selected first exact
queue candidate, frozen library definition, fallback status, promotions,
terminal reason, and final ledger digest. It returns false on a selection or
promotion created before the barrier.

The maximum single run materializes:

| Kind | Bound |
| --- | ---: |
| partial/complete candidates (`57 * 2 + 1`) | 115 |
| refinement edges (`56 * 2`) | 112 |
| complete execution signatures | 31 |
| example results (`31 * 24`) | 744 |
| observations (`31 * 24`) | 744 |
| aggregate evidence | 31 |
| failure constraints (`31 * 2`) | 62 |
| constraint comparisons (`2 * 2 * sum(0..14)`) | 420 |
| transcript actions | 633 |
| descriptor, stage-1 corpus, boundary, stage-2 corpus | 4 |
| library/schema/selection/terminal records | 8 |
| total attributed units | 2,904 |

Facts and examples are bounded encoded lists on the two corpus units, not
individual units: stage 1 stores at most 54 background facts and 24 examples;
stage 2 stores only its at-most-24 examples and references the immutable fact
digest. The boundary unit stores the prior terminal, frozen library, corpus
digest, and canonical store digest. The descriptor and all four input/boundary
units participate in exact-member and profile scans.

`semantic-work/v1` has one authoritative counter table:

| Counter | Charge |
| --- | ---: |
| partial field bind, validation, or order comparison | 1 each |
| fixed-point ground substitution attempted | 1 each |
| body-relation lookup | 1 each |
| insertion attempted after a true body, including duplicates | 1 each |
| novel atom inserted | 1 additional |
| complete-theory parse/normalization field operation | 1 each, exactly 16 per definition |
| example entailment lookup | 1 each |
| theta substitution, clause match, literal match, or term match attempted | 1 each |
| cache lookup | 1 each |
| artifact allocation or collision-name probe | 1 each |
| artifact envelope write or idempotence comparison | 32 each |
| transcript field incorporated into the prefix digest | 1 each |
| selection/terminal comparison | 1 each |

No other operation changes the primary counter. An artifact may contain at most
32 authoritative slots; the fixed envelope tariff deliberately overcharges
smaller artifacts. Audit, oracle, report encoding, fixture generation, inert
focus, and held-out prediction are excluded and separately counted.

For one two-clause definition, the exact worst-case rows are 128 identity plus
1,024 recursive substitutions, 1,152 body lookups, 1,152 insertion attempts,
64 novel inserts, 16 normalization fields, and 24 example checks: 3,560. Thirty
one executions cost at most 110,360, below the fixed-point ceiling.

Theta-subsumption first unifies the two binary heads. This fixes `X` and `Y`;
only the image of `Z` remains, so at most three substitutions are attempted for
each of four clause pairs. At most eight clause/literal/term match operations
follow, for 96 operations per constraint comparison and `420 * 96 = 40,320`
total. The other maxima are 1,000 partial-AST operations, 100 cache lookups,
10,128 transcript digest fields, 185,856 artifact envelope work
(`2,904 * 32 * 2`, allowing one idempotence comparison), 2,968 allocation probes
(one per artifact plus at most 64 collisions), and 64 selection operations.
The total is 350,796, below 500,000.

At most 64 unrelated base names may be preoccupied, never more than one for the
same semantic base. Allocation probes the base then `-collision-1`; a second
occupied suffix is an invalid profile rather than an unbounded loop. Source
tests verify that every adapter-internal theta and allocation step increments
the corresponding counter.

At most 114 refinement tasks, 31 evaluation tasks, two start tasks, two
pre-selection tasks, two finalization tasks, one fallback task, and two initial
seed/transition tasks are required: 154 agenda cycles. The remaining cycles up
to 1,000 may inertly focus units after the terminal record; no focused unit is
eligible to mutate or append experiment artifacts, and the agenda is empty.
The cap remains below the roadmap ceiling.
The deterministic report contains aggregates, selected
theories, complete transcript digests, all constraint/prune counts, and oracle
disagreements rather than embedding every unit, and must remain below 8 MiB.

Budget exhaustion terminates at semantic score 500,001 and is a valid scored
outcome if all integrity checks pass. A missing result for a consumed evaluation,
an uncharged refinement/prune, an unsound constraint, an incorrect exact claim,
or any artifact beyond a cap is mechanically invalid.

## Independent oracle and leakage boundary

`internal/ruleinductionoracle` reimplements safety checks, ground evaluation,
normalization equivalence, candidate-set enumeration, exactness, held-out
predictions, and LFF constraint soundness without importing production vocab or
DSL code. `internal/ruleinductionexp` may import the oracle and production
orchestration but interacts with production only through `seed.LoadDomain`, the
engine, VM execution, and store artifacts. Dependency tests enforce both
boundaries.

The oracle enumerates all 240 theories only after the production run to verify
the declared universe, separating training set, every evaluation, every prune,
all exact ties, and terminal eligibility. Oracle enumeration is reported as
oracle-only work and never affects production search or selection.

Generator ground truth, held-out targets, and cohort labels do not enter the
training store. The canonical training-store bytes are captured before held-out
evaluation and must be identical afterward.

## Semantic cost and endpoint

Only the authoritative `semantic-work/v1` table above changes primary work.
Cache hits charge their lookup and report avoided execution. Pre/post audit,
oracle, fixture generation, reporting, inert focus, and held-out scoring are
separate. Definition and library-freeze work is charged in stage 1 before
exactly one downstream task.

The primary per-fixture endpoint is total charged production semantic work from
the stage-1 root through the frozen stage-2 selection, including candidate
construction, hypothesis execution, materialization, and allowed cache reuse.
Candidate/refinement counts and hypothesis-execution work are separately
reported so cached execution is never called search. Held-out prediction work
and accuracy are also separate. Exhaustion scores 500,001. The locked
primary statistic is the ratio of paired arithmetic means,
`1 - mean(shared-library)/mean(lff-direct)`, over all 64 fixtures.

A `valid-positive` result requires:

- all mechanical gates pass;
- shared-library held-out accuracy is exactly 1.0 overall and in every cohort;
- the primary relative reduction is at least 15% against both
  `lff-direct` and `shared-recomputed`;
- relative reduction against `shared-inlined` is at least 5%, with identical
  accuracy and candidate schedule, and its paired bootstrap interval excludes
  zero and paired randomization has `p < 0.05`, isolating execution of the
  materialized relation from mere equal expressivity;
- direct and recomputed work contrasts each have paired-randomization
  `p < 0.05` and a 95% paired bootstrap interval excluding zero;
- on beneficial fixtures, mean stage-2 candidate dispositions are at least 25%
  lower than `lff-direct`, with paired-randomization `p < 0.05` and a 95%
  paired bootstrap interval excluding zero, the separate search-control gate;
- harmful-cohort ratio of means
  `mean(shared-library)/mean(lff-direct)` is at most 2.0.

Failure of an empirical gate with intact mechanics is `valid-null`. No threshold
is retuned after locked evaluation. Secondary metrics are candidate evaluations,
fixed-point steps, constraints, prune precision, cache benefit, definition size,
cohort accuracy, and wall time. Wall time supports operations only and cannot
support the capability claim.

Each randomization statistic is the absolute paired mean difference in work or
candidate dispositions. Ten thousand sign vectors are drawn from a PCG seeded
by the first 128 bits of the manifest's contrast-specific seed material; the Monte Carlo p-value is
`(1 + count(permuted >= observed)) / 10001`. Bootstrap resamples are 64 fixture
pairs sampled with replacement using that contrast's `|bootstrap` seed. Direct,
recomputed, and inlined use all 64 pairs; beneficial-candidates uses the 40
beneficial pairs. The
interval is the percentile interval at zero-based sorted indices 249 and 9749
of 10,000 finite ratio-of-means relative reductions. A zero control mean or any
non-finite statistic mechanically invalidates the experiment; NaNs are never
sorted or dropped. The same frozen procedure is used for the inlined causal
contrast. Strict `p < 0.05` applies; equality fails.

The deterministic report schema is:

```json
{
  "report_version": "string",
  "manifest": "object-exactly-as-preregistered",
  "implementation_commit": "string",
  "plan_commit": "string",
  "panel": "development|training|validation|locked",
  "status": "valid-positive|valid-null|invalid",
  "mechanical": {
    "all_valid": "bool",
    "dependency_boundary": "bool",
    "oracle_agreements": "integer",
    "oracle_disagreements": "integer",
    "transcript_valid": "bool",
    "stage_boundary_immutable": "bool",
    "training_store_unchanged_by_holdout": "bool",
    "audit_work": "integer"
  },
  "policies": [{
    "name": "string",
    "fixtures": [{
      "seed": "integer",
      "cohort": "beneficial|neutral|harmful|no-solution",
      "terminal": "identified|no-solution|budget-exhausted",
      "stage1_definition": "canonical-code-or-empty",
      "stage2_definition": "canonical-code-or-empty",
      "used_frozen_library": "bool",
      "fell_back": "bool",
      "candidates_consumed": "integer",
      "candidates_executed": "integer",
      "candidates_pruned": "integer",
      "fixed_point_steps": "integer",
      "work": {
        "partial_ast": "integer",
        "fixed_point": "integer",
        "theta": "integer",
        "cache": "integer",
        "allocation_probes": "integer",
        "artifact_envelopes": "integer",
        "transcript_digest": "integer",
        "selection": "integer",
        "total": "integer"
      },
      "heldout_correct": "integer",
      "heldout_total": "integer",
      "accuracy": "number",
      "terminal_digest": "string"
    }],
    "overall": {
      "fixtures": "integer",
      "identified": "integer",
      "no_solution": "integer",
      "budget_exhausted": "integer",
      "candidates_consumed": "integer",
      "candidates_executed": "integer",
      "candidates_pruned": "integer",
      "stage2_candidates_mean": "number",
      "fixed_point_steps": "integer",
      "total_work": "integer",
      "mean_work": "number",
      "heldout_correct": "integer",
      "heldout_total": "integer",
      "accuracy": "number"
    },
    "cohorts": [{
      "name": "beneficial|neutral|harmful|no-solution",
      "fixtures": "integer",
      "identified": "integer",
      "no_solution": "integer",
      "budget_exhausted": "integer",
      "candidates_consumed": "integer",
      "candidates_executed": "integer",
      "candidates_pruned": "integer",
      "stage2_candidates_mean": "number",
      "fixed_point_steps": "integer",
      "total_work": "integer",
      "mean_work": "number",
      "heldout_correct": "integer",
      "heldout_total": "integer",
      "accuracy": "number"
    }]
  }],
  "contrasts": [{
    "treatment": "string",
    "control": "string",
    "statistic": "work-ratio-of-means|candidate-ratio-of-means",
    "relative_reduction": "number",
    "mean_difference": "number",
    "p_value": "number",
    "ci95": ["number", "number"],
    "randomization_replicates": "integer",
    "bootstrap_replicates": "integer",
    "minimum_effect": "number",
    "passed": "bool"
  }],
  "gates": {
    "accuracy": "bool",
    "direct_reduction": "bool",
    "direct_p_value": "bool",
    "direct_ci": "bool",
    "recomputed_reduction": "bool",
    "recomputed_p_value": "bool",
    "recomputed_ci": "bool",
    "inlined_reduction": "bool",
    "inlined_p_value": "bool",
    "inlined_ci": "bool",
    "beneficial_search": "bool",
    "beneficial_search_p_value": "bool",
    "beneficial_search_ci": "bool",
    "harmful_ratio": "bool"
  },
  "controls": {
    "opaque_alias": "bool",
    "alternate_descriptor": "bool",
    "case_order": "bool",
    "occupied_name": "bool",
    "candidate_insert_corruption": "bool",
    "candidate_delete_corruption": "bool",
    "candidate_duplicate_corruption": "bool",
    "alternate_queue_omit": "bool",
    "evidence_positive_flip": "bool",
    "evidence_negative_flip": "bool",
    "mutation_inert": "bool",
    "heldout_store_immutable": "bool",
    "deterministic_json": "bool"
  },
  "limitations": ["string"]
}
```

No nullable or omitted fields are permitted; empty values use their declared
zero representation. Object keys are emitted in struct order and fixture/policy
arrays in preregistered order. Cohort arrays always use beneficial, neutral,
harmful, then no-solution order, including zero-count entries. Contrast order is
direct, recomputed, inlined, then beneficial-candidates.

## Required tests before locked evaluation

- unit tests for parsing, safety, alpha normalization, least fixed points,
  recursion termination, work counters, and malformed inputs;
- differential tests against the independent evaluator over generated tiny
  theories and facts;
- exact proof of the 15/225/240 grammar cardinalities;
- hand-checked witnesses for the shared and no-sharing theories and all their
  entailments;
- CUE seed, opaque-alias, alternate-descriptor, case-order, occupied-name,
  child-VM, idempotence, mutation-on/off, and category-injection tests;
- transcript replay plus deletion, duplication, forgery, stale-profile,
  unsound-prune, and budget corruption tests;
- controls showing alias permutations, fact insertion order, held-out streams,
  and oracle-only truth do not change an already frozen queue or work ledger;
  and a separate control showing that changing the declared queue seed changes
  its permutation and experiment-profile digest deterministically;
- post-run candidate insertion, deletion, and duplication corruption controls,
  each of which must make the unchanged v1 profile fail closed; alternate-profile
  controls that reorder or omit one of the existing 15 legal codes before the
  run, change the profile digest, and make production/oracle agree on the next
  selection or `no-solution`; and evidence perturbations that flip one positive
  and one negative label and force the independently predicted selection change;
- no-artifact, inlined, recomputed, task-local, wrong-context, random, naive,
  and LFF baseline runs under matched budgets;
- held-out store immutability and byte-identical deterministic JSON; and
- `mise exec -- go test ./...`, `mise exec -- go vet ./...`, scoped race tests,
  and `git diff --check`.

## Hand-checked witnesses

For a beneficial fixture whose causal background relation is `q`, the shared
witness is:

```text
i(X,Y)  :- q(X,Y)
i(X,Y)  :- q(X,Z), i(Z,Y)
t1(X,Y) :- i(X,Y)
t2(X,Y) :- i(X,Y)
```

The equal-expressivity no-sharing witness is:

```text
t1(X,Y) :- q(X,Y)
t1(X,Y) :- q(X,Z), t1(Z,Y)
t2(X,Y) :- q(X,Y)
t2(X,Y) :- q(X,Z), t2(Z,Y)
```

Both contain four clauses and entail exactly the transitive closure of `q` for
both targets. The shared mechanism can reduce work only because the materialized
fixed point of `i` is computed once and consumed twice; deleting `i`, expanding
it twice, or independently rediscovering it preserves expressivity but removes
that reuse. The phase plan's implementation tests instantiate a concrete
eight-node branched graph and list every entailed and rejected pair for both
witnesses before implementation is accepted.

That concrete pure-semantics unit witness, which is not a generator fixture, has
`q = {(a,b),(b,c),(b,d),(d,e)}`. Its exact closure is
`{(a,b),(a,c),(a,d),(a,e),(b,c),(b,d),(b,e),(d,e)}`; all other ordered pairs,
including every self pair, are rejected. The two decoy relations use disjoint
edge sets whose closures and every mixed two-clause definition differ on at
least one of those 64 pairs. The stage-1 separator therefore makes the intended
definition uniquely exact; stage 2 reuses it without modification.

For neutral fixtures, all three relations are identical source-to-sink graphs.
Every training-exact definition contains an identity seed and all recursive
clauses are extensionally inert. The held-out generator preserves those two
semantic properties, so all training-exact ties—not merely the selected
canonical code—make identical held-out predictions. The oracle checks this
family theorem and reports the complete tie set.

The frozen harmful sensitivity fixture is manually specified rather than drawn
from any panel seed. It satisfies every harmful generator shape predicate and
uses these eight-edge legal relations, but no claim is made that its literal
edges or queues are PCG outputs:

```text
q0 = {a-b, b-c, b-d, d-e, f-g, g-h, a-f, c-e}
q1 = {h-g, g-f, g-e, e-d, c-b, b-a, h-c, f-d}
q2 = {a-e, a-f, a-g, a-h, b-e, b-f, b-g, b-h}
```

`q0` and `q1` have depth three, branches, non-universal distinct closures, and
`q2` is the legal depth-one decoy. Exhaustive evaluation makes canonical
definition `03` (identity-0 plus tailrec-0) uniquely exact for target 1 and
`14` uniquely exact for target 2. The frozen separator produces exactly:

```text
target 1: +(a,b), +(a,c), -(a,a), -(b,a)
target 2: +(b,a), +(c,a), -(a,a), -(a,b)
```

Its manually frozen stage-1 causal queue is
`[03,14,04,05,12,13,15,23,24,25,01,02,34,35,45]`; stage 2 is
`[14,03,04,05,12,13,15,23,24,25,01,02,34,35,45]`. Both are the literal
manual sensitivity queues. The intended definition is first in each, the worst
case for relative treatment overhead.
With the fixed artifact envelope and four examples, the preregistered ledger is:

| Component | Direct LFF | Shared library |
| --- | ---: | ---: |
| stage-1 refinement, first execution, evidence, selection | 7,642 | 7,642 |
| invented-library freeze and provenance `L` | 0 | 115 |
| failed stage-2 frozen projection `E` | 0 | 484 |
| stage-2 local refinement, first execution, evidence, selection | 7,642 | 7,642 |
| total | 15,284 | 15,883 |

Each 7,642 stage expands as partial AST 326, fixed-point/example 419,
allocation probes 181, artifact envelopes 5,792, transcript fields 920, and
selection 4. `L` has three individual artifacts—library definition,
provenance/schema, and freeze transcript action—so `L = 3 probes + 3*32
envelopes + 16 transcript fields = 115`. `E` has ten evidence artifacts
(signature, four results, four observations, aggregate), three transcript
actions (evaluation, failure, fallback), one cache lookup, four example lookups,
and two comparisons: `E = 13 probes + 13*32 envelopes + 3*16 transcript fields
+ 1 + 4 + 2 = 484`. The implementation test must reproduce every component,
not merely the total.
The ratio is `1.03919`, below 2.0. More generally, `L+E` is bounded by 2,305:
one library/provenance pair plus one no-fixed-point projection with 24 examples.
Each two-stage direct run materializes two full refinement trees and is at least
7,232 before hypothesis execution, so the preregistered 2.0 gate is feasible.
Fallback combines the independently learned stage definitions without Cartesian
enumeration.

## Claims and non-claims

A successful phase would demonstrate bounded relational induction, a real
first-class invented predicate, temporal reuse within one pack/run, sound
failure-derived constraint handling, and a measured total-work and candidate
search benefit under the frozen fixture distribution. Target 2 is an
alias-distinct task over the same background facts; held-out prediction on a new
graph does not establish downstream search transfer on new facts. The phase does
not demonstrate unrestricted ILP, useful invention in an unknown grammar,
natural-language concept learning, cross-pack transfer, production diagnosis,
or open-ended EURISKO behavior.
