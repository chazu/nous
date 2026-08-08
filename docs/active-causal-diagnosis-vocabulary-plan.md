# Active causal-diagnosis vocabulary plan

## Status and decision

Accepted Phase 2 plan, revision 6. No implementation or panel seed has been
executed.

Adversarial review record, 2026-08-07:

- Chandrasekhar, architecture, capability, and provenance boundary: `ACCEPT`;
- Lovelace, causal semantics, acquisition mathematics, and finite bounds:
  `ACCEPT`; and
- Harvey, experimental validity, replayability, and leakage resistance:
  `ACCEPT`.

Earlier review rounds blocked overlapping terminal categories, implicit action
bytes, underdetermined seed streams and digest preimages, inaccessible training
evidence, incomplete resource ledgers, loose DP feasibility, and
self-referential report sizes. Revision 5 incorporated every required
correction; accepted revision 6 reduces training from 640 to 480 episodes to
obey the accepted roadmap's 512-episode ceiling and recomputes all dependent
ledgers.

Revision 1 was blocked by all three reviewers. Revisions 2--4 correct the
gain-per-cost mathematics and witness, freeze the 72-model universe, generator
streams, and exact dynamic recurrence, define the multi-resume teacher state
machine, make the training curriculum and provenance executable, add exact
ledgers, byte accounting, certificate preimages, and wire schemas, and close
hidden-teacher and statistical admissibility gaps.

This phase asks whether Nous can retain an explicit version space of causal
hypotheses, choose interventions from predicted outcome partitions, and learn a
reusable acquisition rule that reduces the cost of correct identification. It
does not ask Nous to infer causality from uncontrolled telemetry and grants no
authority over live systems.

The implementation is confined to `domains/causal`, `internal/vocab/causal`,
`internal/dsl/builtins_causal.go`, `internal/causaloracle`,
`internal/causalexp`, the `causal-trials` command, tests, and documentation.
It changes no engine, VM, agenda, mutation, global credit, or existing pack.

## Preregistered manifest

```json
{
  "experiment_version": "active-causal-diagnosis/v1",
  "generator_version": "three-binary-scm/v1",
  "hypothesis_version": "ordered-dag-mechanisms/v1",
  "acquisition_version": "lexicographic-pairs/v1",
  "oracle_version": "independent-scm-enumerator/v1",
  "teacher_version": "opaque-single-response/v1",
  "training_version": "credit-curriculum/v1",
  "baseline_version": "exact-partition-policies/v1",
  "cost_version": "intervention-cost/v1",
  "report_version": "causal-diagnosis-report/v1",
  "statistics_version": "paired-resampling/v1",
  "profile_version": "causal-profile/v1",
  "development_seeds": {"start": 12001, "count": 16, "step": 1},
  "training_seeds": {"start": 22001, "count": 12, "step": 1},
  "validation_seeds": {"start": 32001, "count": 32, "step": 1},
  "locked_seeds": {"start": 42001, "count": 64, "step": 1},
  "cohort_assignment": "index-mod-8:0-3-cost-skewed,4-5-balanced,6-equivalence,7-irrelevant",
  "locked_cohorts": {"cost-skewed": 32, "balanced": 16, "equivalence": 8, "irrelevant": 8},
  "variables": 3,
  "maximum_indegree": 2,
  "maximum_pool": 32,
  "minimum_initial_posterior": 8,
  "legal_interventions": 6,
  "maximum_consumed_interventions": 10,
  "candidate_acquisition_rules": 40,
  "intervention_cost_minimum": 1,
  "intervention_cost_maximum": 100,
  "episode_cost_ceiling": 1000,
  "invalid_or_exhausted_score": 1001,
  "episode_engine_cycle_cap": 5000,
  "episode_attributed_unit_cap": 1000,
  "descriptor_byte_cap": 8192,
  "curriculum_attributed_unit_cap": 4096,
  "curriculum_semantic_work_cap": 32768,
  "curriculum_engine_cycle_cap": 2048,
  "episode_hypothesis_evaluation_cap": 4096,
  "training_hypothesis_evaluation_cap": 2000000,
  "dynamic_state_cap": 531441,
  "dynamic_work_cap": 4000000,
  "episode_semantic_work_cap": 8192,
  "report_byte_cap": 16777216,
  "fixture_record_byte_cap": 8192,
  "application_certificate_byte_cap": 1024,
  "training_episode_report_byte_cap": 8192,
  "training_episode_bundle_byte_cap": 8388608,
  "nonrecord_report_byte_cap": 1048576,
  "maximum_limitations": 32,
  "limitation_byte_cap": 512,
  "locked_accuracy_gate": 1.0,
  "minimum_primary_reduction": 0.10,
  "integrity_contract": "budgeted-transcript",
  "duplicate_policy": "canonical-code-deduplicate-before-profile",
  "cache_policy": "episode-policy-local-semantic-partition-cache",
  "alpha": 0.05,
  "bootstrap_replicates": 10000,
  "randomization_replicates": 10000,
  "bootstrap_indices_zero_based": [249, 9749],
  "contrast_seed_rule": "active-causal-diagnosis/v1|locked|<information-gain|cost-skewed>|<randomization|bootstrap>",
  "tie_policy": "all-ties-reported-first-semantic-code-executed",
  "mutation_enabled": false
}
```

The exact manifest is emitted in every report. Development may change code and
the plan. Training is executed once only after the plan is accepted; its
selected acquisition-rule code and report digest are then frozen in source.
Validation may reject the implementation but may not change the rule,
thresholds, generator, or costs. The locked runner requires an existing clean
candidate commit equal to `HEAD`; no locked seed is exposed earlier. Any change
after training creates v2 and requires new disjoint seed ranges.

Every episode uses this exact compact-canonical profile object in the shown
field order:

```json
{
  "profile_version": "causal-profile/v1",
  "manifest": "exact-object-above",
  "panel": "development|training|validation|locked",
  "seed": "integer",
  "acquisition_code": "canonical-rule-or-policy-code",
  "fixture_digest": "sha256-hex",
  "profile_digest": "sha256-hex"
}
```

The digest is SHA-256 of `causal-profile/v1`, one NUL byte, and this exact
object with `profile_digest:""`. The profile and its descriptor encoding are
each capped at 8,192 bytes. Decoding rejects unknown fields, mismatched frozen
constants, or noncanonical codes.

## Structural causal models

Variables occupy descriptor positions `0,1,2`, independently aliased in every
episode. Parents must have smaller positions, so topological order is unique and
cycles are unrepresentable in canonical hypotheses. Each variable has zero to
two distinct parents and exactly one deterministic mechanism:

- arity zero: constant 0 or constant 1;
- arity one: copy or negate its parent; and
- arity two: conjunction, disjunction, or exclusive-or of ordered parents.

A hypothesis semantic code is compact canonical JSON with HTML escaping
disabled and this exact field order:

```json
{"nodes":[{"parents":[],"mechanism":"constant-0"},{"parents":[0],"mechanism":"copy"},{"parents":[0,1],"mechanism":"and"}]}
```

There are exactly three node objects in position order. Parent positions are
ascending integers. Parent-list enum order is `[]`, `[0]`, `[1]`, `[0,1]`
after discarding lists illegal for the node position. Within arity, mechanism
enum order is `constant-0,constant-1`; `copy,negate`; and `and,or,xor`.
Unknown or extra JSON fields and noncanonical encodings are rejected. The
semantic code is not a unit name. Safety rejects wrong
arity, duplicate parents, forward/self edges, unknown mechanisms, non-binary
values, duplicate hypothesis codes, and pools outside `[8,32]` after the initial
observation.

Every instruction to sort by semantic code means ascending bytewise UTF-8 order
of these canonical JSON strings, not a Go struct comparator or map order.

Evaluation proceeds in position order. An intervention `do(V=b)` replaces only
V's mechanism for that trial; descendants use the intervened value. The result
is the complete three-bit outcome. An observation is evaluation without a
replacement and is represented separately from an intervention result.

The six semantic action codes, in their only valid enum and `IntN(6)` index
order, are `do:0=0`, `do:0=1`, `do:1=0`, `do:1=1`, `do:2=0`, and `do:2=1`.
Aliases are display-only and never alter these codes. Every sorted action array,
lexical tie break, transcript, and digest uses this order.

The independent generator enumerates the complete legal three-variable SCM
universe, samples 32 distinct hypotheses without replacement, samples a hidden
member using a separate PCG stream, obtains its passive outcome, and filters the
pool by that outcome. It rejects and retries unless 8–32 hypotheses remain and
the hidden model remains. Alias, list-order, cost, pool, and hidden-member
streams use independently labelled seeds. The hidden model and its
index never occur in a fixture name, alias, descriptor, unit, queue, or
production callback. The universe has exactly
`2 * (2 + 2) * (2 + 4 + 3) = 72` hypotheses in lexicographic variable,
parent-list, then mechanism order. Exhaustive tests prove 58 six-action
equivalence classes: 14 of size two and 44 singletons.

Sampling uses Go `math/rand/v2` PCG with the first 128 SHA-256 bits, decoded as
two big-endian uint64 words, of
`active-causal-diagnosis/v1|<panel>|<seed>|<label>|<attempt>` as its two words.
The only labels are literal `pool`, `hidden-member`, `alias`, `list-order`,
`cost-values`, `cost-assignment`, and `uniform-random`; unknown labels are
invalid. Hidden selection is `sortedPool[IntN(len(sortedPool))]`. Alias starts
from `node-u,node-v,node-w` and Fisher-Yates shuffles those three strings.
List-order independently Fisher-Yates shuffles the sorted pool for store
presentation; semantic profiles and digests retain sorted code order.
Pool sampling Fisher-Yates shuffles the canonical 72-code array from the last
index down with `j=IntN(i+1)`, takes the first 32, then sorts by semantic code.
Inclusive costs use `lo+IntN(hi-lo+1)`; three drawn costs are assigned to
variable positions by an independently labelled Fisher-Yates permutation.
Uniform-random uses its label with the fixture's accepted generator attempt and
draws `IntN(6)` exactly ten times.
Attempts are exactly
`0..4095`; failure to satisfy all cohort predicates is a generator error and
invalidates the panel rather than changing a bound. Exhaustive source tests
construct at least one witness for every cohort.

Costs attach to variables and apply equally to `do(V=0)` and `do(V=1)`. The
cost-skewed cohort has one cost in `[1,10]`, one in `[30,50]`, and one in
`[80,100]`; balanced, equivalence, and irrelevant costs are independently
sampled in `[20,40]`; equivalence fixtures require the hidden model's complete
class within the initial posterior to have size two; irrelevant fixtures contain
a variable V for which, for both b, every hypothesis in the initial posterior
predicts the same outcome under `do(V=b)`, so both partitions have one cell.
Rejection
checks those semantic properties with the independent generator, not names.

## Posterior and terminal semantics

For posterior P and action a, the predicted partition maps each three-bit
outcome to the sorted hypotheses in P that predict it. After the teacher returns
outcome o, production retains exactly partition cell `P[a=o]`. Empty cells,
teacher outcomes absent from the predicted partition, posterior growth, or a
posterior that loses the teacher are mechanically invalid.

Two hypotheses are interventionally equivalent when all six legal action
signatures match. A terminal is mutually exclusive:

- `identified`: the posterior contains exactly one hypothesis, which the
  post-terminal oracle verifies is the retained hidden hypothesis;
- `equivalence`: the posterior contains more than one hypothesis and equals
  exactly one complete six-action-signature class within the original initial
  posterior; the post-terminal oracle verifies it is the hidden SCM's class;
  or
- `budget-exhausted`: no correct terminal was reached before the next action
  would exceed cost 1,000 or after ten consumed interventions.

Production detects equivalence without hidden access by choosing any posterior
member, comparing its six-action signature with every original-posterior
member, and requiring the resulting complete class to equal the posterior.
Incorrect identification and all mechanical faults score 1,001. A valid
budget-exhausted weak policy also scores 1,001. Passive-only immediately emits
budget exhaustion unless the initial posterior is already terminal. Repeated
actions are legal but deterministic and normally uninformative; they consume
cost and count toward the ten-action cap.

A correct singleton or equivalence terminal has `score = intervention_cost`
exactly. No work, cache, action-count, or credit term changes the score.

Executing all six distinct actions always reaches the exact equivalence class
and costs at most 600. Every production scoring round admits and proposes all
six actions, including consumed actions, so R is a real feature under learned
rules. Information-gain and worst-case choose an unconsumed refining action
because its gain is positive while a repeat's is zero; lexical minimizes
`(R,semantic-code)`; dynamic proves repeats dominated. Uniform-random
precomputes ten independent with-replacement draws, consumes only their prefix
until terminal or budget exhaustion, and may repeat. Learned rules may repeat and
must earn the accuracy gate; passive-only is expected to exhaust. Accuracy is one exactly when a singleton is the
hidden hypothesis or an equivalence posterior equals the class defined above.

## Acquisition features and the 40-rule language

For a posterior of size N and partition cell sizes `n_i`, production records:

- expected-remaining numerator `E = sum(n_i^2)`; minimizing E/N minimizes the
  expected posterior under the uniform prior;
- worst case `W = max(n_i)`;
- entropy product `H = product(n_i^n_i)`; for fixed N, minimizing this exact
  unbounded integer maximizes Shannon outcome entropy without floating point;
- action cost `C`; and
- repeat flag `R`, zero before an action has been consumed and one afterwards.

The semantic cache records the sorted outcome/cell-size partition and all five
features. It never stores or queries the teacher outcome.

The rule language contains exactly 40 codes. A code selects one primary from
`E,W,H,C,R`, selects `raw` or `gain-per-cost`, and selects one different
secondary feature: `5 * 2 * 4 = 40`. Codes are
`P=<E|W|H|C|R>;M=<raw|gain>;S=<other>` in that literal enum order.
Raw rules minimize `(primary, secondary, semantic action code)`. Gain rules
maximize their primary benefit per cost, then minimize `(secondary, semantic
action code)`:

- E benefit compares `(N^2-E_a)*C_b` with `(N^2-E_b)*C_a`;
- W benefit compares `(N-W_a)*C_b` with `(N-W_b)*C_a`;
- entropy information gain compares
  `N^(N*C_b)*H_b^C_a` with `N^(N*C_a)*H_a^C_b`, choosing a when the left side
  is greater; and
- C and R gain modes are defined to use their raw minimization, making them
  deliberate grammar aliases whose complete ties are reported.

All integers are exact. Each exponentiated entropy factor is at most 16,001 bits
and the direct product comparator is at most 32,001 bits because `N<=32` and
`C<=100`; arbitrary-precision exponentiation and comparison are
charged by the semantic tariff below.

Every rule reports all primary-and-secondary ties before semantic-code
selection. No map order or unit name participates. The learned rule is selected
by Nous in a dedicated training curriculum store. The driver runs each of the
40 rules on each of 12 isolated episode stores. A fresh post-episode verifier
creates a content-addressed certificate containing profile/fixture digest, rule
code, score, terminal, cost, posterior digest, transcript digest, oracle
agreements/disagreements, all-caps validity, full episode-report digest, and
certificate digest; there is no secret or signature. The certificate is issued
only after fresh profile, transcript, cap, and zero-disagreement verification.
Its digest is SHA-256 of `causal-application-certificate/v1`, one NUL byte, and
compact canonical JSON of the exact typed application schema below with
`certificate_digest:""`. Decoding rejects unknown/extra fields; the curriculum
independently clears and recomputes the digest before accepting it. The
curriculum accepts only the exact 40-by-12 `(rule,seed)` matrix from committed
episode reports, rejects duplicate/missing/extra/digest-mismatched certificates,
and post-selection audit independently replays every report. Ordinary CUE
heuristics materialize 40 rule units, 480 application units, per-rule aggregate
evidence, credit deltas, all exact winner ties, and the selection. Credit is the
negative episode score plus a fixed 1,001 success offset; the selection barrier
requires highest recomputed worth, then fewest exhaustions, then lowest semantic
code. The driver cannot create the selection unit. A no-credit training control
must prevent that same selection or leave a declared unresolved tie; otherwise
the learning claim fails mechanically.

Freeze provenance is:

1. commit the accepted plan;
2. commit a clean pretraining executable containing no frozen rule;
3. run training once from that exact commit and atomically commit its canonical
   report plus canonical episode-evidence bundle, whose
   `training_report_commit` field remains `""` permanently;
4. define the training digest by the canonical preimage below;
5. make the sole permitted post-training semantic edit: insert selected code,
   training-report commit, and digest constants;
6. replay training byte-for-byte and commit the implementation candidate;
7. run validation only from that exact clean candidate `HEAD`, making no
   changes; then permit locked execution from the same exact clean `HEAD`.

Panel runners reject training before the accepted-plan/pretraining commits,
validation before frozen constants resolve, and locked before the candidate
barrier. Any other semantic edit after training creates v2. Training replay
must reproduce the committed report byte-for-byte. Later source and evaluation
reports name the commit containing the training report; the report never names
its own commit.

The flat application array is seed-major in manifest training-seed order, then
rule-major in the 40-code enum order.

Each rule aggregate's `application_digest` is SHA-256 of
`causal-rule-applications/v1`, one NUL byte, and compact canonical JSON of the
array of that rule's 12 exact application-certificate objects in manifest
training-seed order. Verification rejects unknown fields or wrong order and
recomputes the digest from the accepted certificates.

The training digest is SHA-256 of
`causal-training-digest-input/v1`, one NUL byte, and compact canonical JSON of
this exact typed object in the shown field order:

```json
{
  "digest_input_version": "causal-training-digest-input/v1",
  "manifest": "exact-object-above",
  "plan_commit": "40-hex",
  "pretraining_commit": "40-hex",
  "episode_bundle_digest": "sha256-hex",
  "fixture_digests": ["sha256-hex"],
  "application_certificates": ["exact-typed-application-object"],
  "rule_aggregates": ["exact-typed-rule-object"],
  "winner_ties": ["canonical-rule-code"],
  "selected_rule": "canonical-rule-code"
}
```

The episode-bundle digest is independently verified first. Fixture digests are
in manifest training-seed order; application certificates
are seed-major then rule-enum order; rule aggregates are rule-enum order; and
winner ties are ascending semantic rule code. The application and rule object
types are exactly the corresponding training-report schemas below, without
projection or omitted fields. Encoding uses the same compact canonical JSON
rules as reports. Digest decoding rejects unknown or extra fields, wrong array
lengths or order, and noncanonical rule codes. Independent verification rebuilds
this object from the accepted application certificates and recomputed
aggregates rather than trusting the report's stored bytes.

Training evidence is committed at
`docs/evidence/active-causal-diagnosis-training-episodes-v1.json`; the canonical
training report is committed beside it as
`docs/evidence/active-causal-diagnosis-training-v1.json`. The curriculum can
receive verified application certificates but has no handle to the evidence
bundle or its hidden fixture fields. The bundle becomes available only to the
post-selection verifier after the central agenda drains.

The exact fixture object is:

```json
{
  "seed": "integer",
  "generator_attempt": "integer",
  "cohort": "cost-skewed|balanced|equivalence|irrelevant",
  "aliases": ["three-position-ordered-strings"],
  "costs": ["three-position-ordered-integers"],
  "passive_outcome": "three-bit-string",
  "pool": ["32-sorted-canonical-hypothesis-codes"],
  "initial_posterior": ["8-to-32-sorted-canonical-hypothesis-codes"],
  "hidden_hypothesis": "canonical-hypothesis-code",
  "fixture_digest": "sha256-hex"
}
```

Its public fixture digest excludes the hidden hypothesis: it is SHA-256 of
`causal-public-fixture/v1`, one NUL byte, and compact canonical JSON of the
object above with the `hidden_hypothesis` field omitted and
`fixture_digest:""`. This is exactly the production-visible fixture projection,
so counterfactual-hidden twins share it. The bundle digest below binds the full
post-terminal fixture, including the hidden hypothesis. The exact episode report
object is:

```json
{
  "episode_report_version": "causal-training-episode/v1",
  "seed": "integer",
  "profile_digest": "sha256-hex",
  "fixture_digest": "sha256-hex",
  "rule_code": "canonical-rule-code",
  "actions": ["canonical-action-code"],
  "teacher_outcomes": ["three-bit-string"],
  "terminal": "identified|equivalence|budget-exhausted",
  "score": "integer",
  "cost": "integer",
  "final_posterior": ["sorted-canonical-hypothesis-codes"],
  "posterior_digest": "sha256-hex",
  "transcript_digest": "sha256-hex",
  "hypothesis_evaluations": "integer",
  "semantic_work": "integer",
  "attributed_units": "integer",
  "engine_cycles": "integer",
  "oracle_agreements": "integer",
  "oracle_disagreements": "integer",
  "all_caps_valid": "bool",
  "episode_report_digest": "sha256-hex"
}
```

Action and outcome arrays have equal length at most ten. The episode digest is
SHA-256 of `causal-training-episode/v1`, one NUL byte, and compact canonical JSON
of that object with `episode_report_digest:""`. A fresh verifier regenerates the
fixture from `(training,seed)`, replays the actions through a newly constructed
production store and independent teacher, and requires every regenerated
terminal, digest, counter, cap, oracle result, and application-certificate field
to match. Thus the compact episode report is replay evidence, not a trusted
summary.

The exact bundle is:

```json
{
  "bundle_version": "causal-training-episode-bundle/v1",
  "manifest": "exact-object-above",
  "plan_commit": "40-hex",
  "pretraining_commit": "40-hex",
  "fixtures": ["exact-fixture-object"],
  "episodes": ["exact-episode-report-object"],
  "bundle_digest": "sha256-hex"
}
```

Fixtures are in training-seed order and episodes are seed-major then rule-enum
order. The bundle digest is SHA-256 of
`causal-training-episode-bundle/v1`, one NUL byte, and compact canonical JSON of
the exact bundle with `bundle_digest:""`. All four typed decoders reject unknown
fields, wrong order, duplicate semantic keys, noncanonical nested encodings, and
wrong lengths. Each encoded fixture and episode is capped at 8,192 bytes. With
12 fixtures, 480 episodes, 490 intervening commas, and a 1,048,576-byte
nonrecord shell, the bundle is bounded by 5,079,530 bytes and must also pass its
independent 8 MiB final-file cap.

## Policies and baselines

Locked evaluation runs these independent policies:

1. `learned`: the training-frozen acquisition rule;
2. `information-gain-per-cost`: maximize the exact entropy-gain comparator
   defined above;
3. `worst-split-per-cost`: maximize `(N-W)/C` by cross multiplication;
4. `lexical-fixed`: minimize `(repeat flag, semantic action code)`;
5. `uniform-random`: ten PCG `IntN(6)` draws with replacement, independent of
   hidden/model streams;
6. `passive-only`; and
7. `dynamic-optimal`: exact uniform-prior dynamic programming over
   `(posterior, remaining actions)` minimizing expected additional cost, with
   semantic-code tie handling.

For posterior P and unused actions A, dynamic optimal uses exact `big.Rat`:

```text
V(P,A) = 0                                      when P is terminal
V(P,{}) = explicit non-finite sentinel         otherwise
V(P,A) = min_a [ C(a) + sum_o |P_a,o|/|P| * V(P_a,o,A-{a}) ]
```

Nonrefining actions are dominated unless all actions are nonrefining; semantic
action code breaks exact Q-value ties. Memoization keys the sorted posterior
codes and remaining-action bitset. Values are `{finite bool; value big.Rat}`;
the sentinel is never passed to `big.Rat`. A state is determined by a subset of
the six actions and one outcome cell for each, so the coarse finite bound is
`sum(k=0..6, C(6,k)*8^k) = 9^6 = 531,441` states per initial posterior. A pool
contains at most 32 hypotheses, so at most `min(8^k,32)` distinct nonempty
outcome cells are reachable after any k-action subset. The tighter bound is
therefore `sum(k=0..6, C(6,k)*min(8^k,32)) = 1,873` reachable states. At each
state the tariff charges at most one expansion plus six action-Q evaluations,
48 cell rational terms, and 48 memo lookups, or 103 work units; the conservative
table-construction bound is `1,873*103 = 192,919`. The realized trajectory and
up to 32 hidden-member simulations each perform at most six separately charged
policy-table lookups, adding at most 198; the conservative per-fixture DP-work
bound is therefore 193,117. Exhaustive tests over every
legal passive-filtered initial posterior must remain below both caps. DP work is
reported as oracle/benchmark work and excluded from production acquisition
work. Profiles fix the 531,441-state cap and a
4,000,000-work cap independently for each fixture, where one state expansion,
action Q evaluation, cell rational term, or memo lookup is one work unit.
Crossing either cap is mechanically invalid. The policy is
evaluated both on the realized teacher trajectory and, separately, by simulating
every possible hidden member of the initial posterior to report its exact
uniform expected cost. It is an empirical benchmark, not a pointwise lower
bound or a positive-result gate.

Policies share fixture bytes and ceilings
but never stores, transcripts, caches, random streams, or credit.

No-credit, wrong-context, static-rule, and recomputed-rule controls are separate
development runs. The static rule is the semantic first rule code. Recomputed
replays the frozen rule's feature calculations without its cached partitions;
it must choose the same actions and accuracy, isolating cache execution cost
from acquisition quality.

A counterfactual-hidden twin control holds the visible initial posterior,
observation, costs, aliases, order, policy, and profile byte-identical while the
private teacher token selects two different posterior members. All production
bytes through the first `awaiting-teacher` boundary and the selected first
action must be byte-identical. Only the authorized response and its descendants
may diverge.

## Production protocol and artifacts

The driver creates an opaque descriptor and initial-posterior unit, then seeds
ordinary CUE tasks. Go builtins only validate/compute pure SCM semantics,
partitions, exact feature comparisons, and transcript replay. CUE heuristics:

1. materialize one action proposal and predicted partition for every legal
   action;
2. score proposals with the descriptor-selected acquisition rule;
3. materialize all exact ties and the chosen action;
4. stop at the pre-intervention barrier;
5. allow the external teacher to insert one result unit only for that chosen
   action;
6. resume ordinary engine execution to materialize eliminated hypotheses and
   the new posterior; and
7. record a verified terminal or repeat.

The descriptor state machine is exact:

```text
ready -> proposing -> awaiting-teacher -> response-present -> updating -> ready
                                                              \-> terminal
```

One `Engine.Run` advances `ready` through `proposing` to `awaiting-teacher`.
At that boundary a fresh verifier requires six proposals, partitions, and
scores; the complete tie set and one selection; no response; no eligible
proposal/update task; and an empty agenda. Standalone `nous run -domain causal`
stops there with terminal text `awaiting-external-teacher` and never fabricates
a response. The driver then authorizes exactly one collision-safe response name
bound to `(profile, episode, step, selected action, selection digest)`, calls
only `Teacher.Respond`, inserts that response, changes state to
`response-present`, and adds exactly one descriptor-declared resume task.

The resumed Run validates authorization before entering `updating`, consumes
the response once, reaches `ready` or `terminal`, and drains the agenda. At a
nonterminal `ready` boundary a fresh verifier requires the exact posterior and
cost update, consumed response, no pending artifact, and empty agenda; the
driver then pushes exactly one descriptor-declared next-proposal task. A
response in any other state, a second response, a response/action mismatch, or
nonempty agenda at a boundary fails closed. Every Run receives
`MaxCycles = 5000 - cumulativePriorCycles`; the driver sums `Engine.Cycle()`
across resumes and rejects zero remaining budget, so resets cannot multiply the
cap. At most 21 Runs occur: one proposal boundary and one update boundary for
each of ten actions, plus finalization.

The teacher API receives only `(fixture token, chosen action)` and returns a
three-bit outcome; it cannot read or mutate the Nous store. Production never
imports the oracle package. The opaque token resolves only inside a private
teacher object that implements `Respond`; the online driver is prohibited from
calling enumeration, hidden-model inspection, prediction, or audit methods.
All other oracle/generator/audit access begins only after terminal. Dependency
and capability tests enforce that `Respond` is the sole online information
right.

Every attributed artifact has profile key, episode, step, kind, collision-safe
semantic key, sealed authoritative digest, and one charge index. Required kinds
are descriptor, initial observation, posterior, proposal, partition, score,
tie, selection, teacher result, elimination, cache, terminal, and transcript.
Collision or conflicting reuse fails closed.

The transcript is gap-free and commits to previous digest, chosen rule/action,
posterior-before digest, complete predicted partition digest, teacher outcome,
posterior-after digest, eliminated-set digest, cost before/after, action count,
cache status, attributed-unit prefix, and remaining evaluation/cycle/unit
budgets. Replay reconstructs every posterior and cost from the initial pool and
teacher results. Selection cannot occur before every legal action has a
verified partition and score. Terminal verification independently recomputes
singleton/equivalence completeness and budget state.

Posterior and eliminated-set digests are SHA-256 of
`causal-hypothesis-set/v1`, one NUL byte, and compact canonical JSON of the
sorted hypothesis-code array. A partition digest is SHA-256 of
`causal-partition/v1`, one NUL byte, and compact canonical JSON of nonempty cell
objects `{"outcome":"000","hypotheses":[...]}` in binary outcome order with
sorted codes. Each transcript entry uses this exact field order:

```json
{
  "transcript_version": "causal-transcript/v1",
  "episode": "collision-safe-semantic-key",
  "step": "integer",
  "previous_digest": "sha256-hex-or-64-zeroes",
  "rule_code": "canonical-rule-or-policy-code",
  "action": "canonical-action-code",
  "posterior_before_digest": "sha256-hex",
  "partition_digest": "sha256-hex",
  "teacher_outcome": "three-bit-string",
  "posterior_after_digest": "sha256-hex",
  "eliminated_digest": "sha256-hex",
  "cost_before": "integer",
  "cost_after": "integer",
  "action_count": "integer",
  "cache_status": "hit|miss",
  "attributed_unit_prefix": "integer",
  "remaining_hypothesis_evaluations": "integer",
  "remaining_semantic_work": "integer",
  "remaining_engine_cycles": "integer",
  "remaining_attributed_units": "integer",
  "transcript_digest": "sha256-hex"
}
```

An entry digest is SHA-256 of `causal-transcript-entry/v1`, one NUL byte, and
compact canonical JSON of the entry with `transcript_digest:""`. Step zero uses
64 ASCII zeroes as its previous digest; later steps use the prior entry digest.
The episode transcript digest is the last entry digest, or SHA-256 of
`causal-empty-transcript/v1` for a zero-action terminal. Typed decoders reject
unknown fields and noncanonical enum values.

## Independent oracle and work accounting

`internal/causaloracle` independently parses and enumerates SCMs, evaluates
observations/interventions, builds partitions, filters posteriors, computes
equivalence classes, implements all baselines, and audits every production
proposal, result, update, terminal, and held-out prediction. It imports neither
the production vocabulary nor DSL.

The primary endpoint counts external intervention cost only. `causal-work/v1`
separately charges one unit for each complete SCM evaluation, partition
assignment, cell feature accumulation, exact rule comparison (including one
bounded big-integer comparison), posterior membership check, artifact
materialization, transcript field incorporation, and descriptor/profile field
validation. Teacher evaluation and independent audit work are separate fields.

One episode's conservative upper bounds are:

| Work | Bound |
| --- | ---: |
| production SCM evaluations (`32 + 10*6*32`) | 1,952 |
| partition assignments (`10*6*32`) | 1,920 |
| cell feature operations (`10*6*8*3`) | 1,440 |
| pairwise rule comparisons (`10*15`) | 150 |
| posterior membership checks (`10*32`) | 320 |
| artifact materializations | 664 |
| transcript fields (`20*16`) | 320 |
| descriptor/profile validation | 100 |
| total semantic work | 6,866 |

The per-episode profile fixes ceilings of 4,096 SCM evaluations, 8,192 semantic
work, 1,000 attributed units, and 5,000 cumulative engine cycles. The
conservative artifact-count bound is three initial units plus ten steps of 66 units (six each of
proposal/partition/score/tie/cache, one selection, one response, up to 31
eliminations, one posterior, and two transcripts) plus one terminal: 664.
Training runs 480 isolated episodes and therefore charges at most 936,960 SCM
evaluations, below its frozen 2,000,000 panel ceiling; its central curriculum
has conservative bounds of 1,603 units: one descriptor, 40 rules, 480
applications, 40 aggregates, 480 credit deltas, 40 winner ties, one selection,
and 521 transcript actions. Its semantic-work bound is 18,499: 6,720 certificate
fields, 480 credit operations, 480 aggregate operations, 780 pair comparisons,
8,336 transcript fields, 1,603 materializations, and 100 profile operations.
At most 524 curriculum tasks run. These fit exact central caps of 4,096 units,
32,768 work, and 2,048 cycles. Episode units remain governed per isolated
episode store and their summed count is reported separately, not compared with
the central-store cap. Panel reports embed summaries rather than artifact
stores.

Report encoding is compact canonical JSON with indentation disabled and HTML
escaping disabled. Each fixture or application object is first encoded alone
and checked against its component cap. The report is then encoded with every
record array replaced by `[]` and both `nonrecord_bytes` and `report_bytes` set
to the literal eight-character string `00000000`; that encoded byte count,
including every other key, value, bracket, quote, escape, and aggregate, is the
nonrecord count and is checked against 1,048,576 bytes. The two fields are then
replaced by their zero-padded eight-decimal-digit values, which preserves the
preimage width and eliminates self-reference. Reports of 100,000,000 bytes or
more are unrepresentable and invalid. Reinserting records adds their measured object
bytes and exactly one comma between adjacent objects. Thus 448 locked fixture
records use at most 3,670,016 bytes and the seven 64-record arrays add exactly
441 commas, for a proved component maximum of 4,719,033 bytes. The training
report's 480 application certificates use at most 491,520 bytes and add 479
commas, for a maximum of 1,540,575 bytes. Every string field is either a bounded
enum, fixed-length digest, or separately length-checked limitation; actual JSON
encoding, rather than source character counts, supplies the escaping evidence;
there are at most 32 limitation strings and each is at most 512 encoded bytes.
Both component proofs remain below the independent final 16 MiB encoded-report
cap. Component caps, nonrecord reconstruction equality, and the final encoded
cap are mechanical gates.

All resource ceilings are exact v1 profile constants. Lowered or raised values
are invalid profiles. Any overrun is mechanically invalid; only the semantic
episode-cost/action-count exhaustion described above is a valid scored outcome.

## Statistics and acceptance

The primary paired endpoint is the ratio-of-means reduction
`1 - mean(learned)/mean(information-gain)`. A `valid-positive` result requires:

- every mechanical gate and adversarial control passes;
- learned correct singleton/equivalence identification accuracy is 1.0 overall
  and in every cohort;
- information-gain correct singleton/equivalence identification accuracy is
  also 1.0 overall and in every cohort, making the primary contrast a cost
  comparison among correct terminals rather than a failure-penalty comparison;
- primary reduction is at least 10%;
- paired randomization has `p < 0.05` and the 95% paired bootstrap interval for
  reduction excludes zero;
- on cost-skewed fixtures, learned reduction against information gain is at
  least 10% with `p < 0.05` and a positive bootstrap interval.

The primary contrast resamples the 64 seed-aligned learned/information-gain
score pairs; the cost-skewed contrast resamples exactly its 32 such pairs. The
randomization and bootstrap algorithms, replicate counts, indices, seed
derivation, strict inequality, invalid non-finite handling, and ratio-of-means
procedure are otherwise identical to the accepted Phase 1 paired-resampling contract. A
zero information-gain mean, non-finite statistic, invalid baseline terminal,
wrong cohort count, or missing pair mechanically invalidates the report.
Training never uses locked thresholds. A mechanically intact threshold failure
is `valid-null`; a mechanical failure is `invalid`.

The report includes exact manifest and provenance; frozen rule code and training
digest; per-policy/per-cohort scores, costs, actions, terminal counts, accuracy,
cache and work counts; per-fixture posterior sizes and transcript digests;
all contrasts and gates; oracle agreements/disagreements; and every control.
No nullable or omitted field is allowed, arrays have fixed order, JSON is
deterministic, and the encoded report must remain below 16 MiB.

The canonical training report order and types are:

```json
{
  "report_version": "causal-training-report/v1",
  "manifest": "exact-object-above",
  "plan_commit": "40-hex",
  "pretraining_commit": "40-hex",
  "training_report_commit": "",
  "episode_bundle_digest": "sha256-hex",
  "episode_bundle_bytes": "integer",
  "panel": "training",
  "status": "valid|invalid",
  "fixture_digests": ["sha256-hex"],
  "applications": [{
    "seed": "integer",
    "profile_digest": "sha256-hex",
    "fixture_digest": "sha256-hex",
    "rule_code": "canonical-rule-code",
    "score": "integer",
    "terminal": "identified|equivalence|budget-exhausted",
    "cost": "integer",
    "posterior_digest": "sha256-hex",
    "transcript_digest": "sha256-hex",
    "oracle_agreements": "integer",
    "oracle_disagreements": "integer",
    "all_caps_valid": "bool",
    "episode_report_digest": "sha256-hex",
    "certificate_digest": "sha256-hex"
  }],
  "rules": [{
    "code": "canonical-rule-code",
    "applications": 12,
    "total_score": "integer",
    "total_cost": "integer",
    "identified": "integer",
    "equivalence": "integer",
    "budget_exhausted": "integer",
    "worth": "integer",
    "application_digest": "sha256-hex"
  }],
  "winner_ties": ["canonical-rule-code"],
  "selected_rule": "canonical-rule-code",
  "training_digest": "sha256-hex",
  "mechanical": {
    "all_valid": "bool",
    "credit_recomputed": "bool",
    "selection_verified": "bool",
    "oracle_agreements": "integer",
    "oracle_disagreements": "integer",
    "episode_hypothesis_evaluations_total": "integer",
    "episode_hypothesis_evaluations_max": "integer",
    "episode_semantic_work_total": "integer",
    "episode_semantic_work_max": "integer",
    "episode_attributed_units_total": "integer",
    "episode_attributed_units_max": "integer",
    "episode_engine_cycles_total": "integer",
    "episode_engine_cycles_max": "integer",
    "max_descriptor_bytes": "integer",
    "max_training_episode_report_bytes": "integer",
    "curriculum_semantic_work": "integer",
    "curriculum_attributed_units": "integer",
    "curriculum_engine_cycles": "integer",
    "max_application_certificate_bytes": "integer",
    "nonrecord_bytes": "eight-digit-zero-padded-decimal",
    "report_bytes": "eight-digit-zero-padded-decimal",
    "all_caps_valid": "bool"
  },
  "controls": {
    "no_credit_changes_selection": "bool",
    "hidden_twin": "bool",
    "wrong_context": "bool",
    "static_rule": "bool",
    "deterministic_json": "bool"
  },
  "limitations": ["string"]
}
```

Rules are in the 40-code enum order. Applications are seed-major in manifest
training-seed order, then rule-major in the 40-code enum order. The certificate
preimage uses the application-object field order shown above and includes every
field through `certificate_digest`, whose preimage value is the empty string.
Training is `valid` exactly when the complete matrix, credit, selection,
provenance, oracle, all controls, all caps, deterministic encoding, and
non-null schema pass. `all_caps_valid` requires every episode maximum,
descriptor/evidence byte maximum, and curriculum total to be at or below its
matching manifest cap, the panel hypothesis-evaluation total to be at or below
its panel cap, the certificate and report byte fields to equal independent
re-encoding, and all report, component, and episode-bundle byte caps to pass.
Any failure is `invalid` and forbids freezing.
The canonical evaluation report is:

```json
{
  "report_version": "causal-diagnosis-report/v1",
  "manifest": "exact-object-above",
  "plan_commit": "40-hex",
  "pretraining_commit": "40-hex",
  "training_report_commit": "40-hex",
  "training_digest": "sha256-hex",
  "frozen_rule": "canonical-rule-code",
  "implementation_commit": "40-hex-or-empty-nonlocked",
  "panel": "development|validation|locked",
  "status": "valid-positive|valid-null|invalid",
  "mechanical": {
    "all_valid": "bool",
    "dependency_boundary": "bool",
    "profile_valid": "bool",
    "transcript_valid": "bool",
    "training_freeze_valid": "bool",
    "oracle_agreements": "integer",
    "oracle_disagreements": "integer",
    "audit_work": "integer",
    "max_hypothesis_evaluations": "integer",
    "max_semantic_work": "integer",
    "max_engine_cycles": "integer",
    "max_attributed_units": "integer",
    "max_descriptor_bytes": "integer",
    "max_fixture_record_bytes": "integer",
    "nonrecord_bytes": "eight-digit-zero-padded-decimal",
    "report_bytes": "eight-digit-zero-padded-decimal",
    "all_caps_valid": "bool"
  },
  "policies": [{
    "name": "learned|information-gain-per-cost|worst-split-per-cost|lexical-fixed|uniform-random|passive-only|dynamic-optimal",
    "fixtures": [{
      "seed": "integer",
      "cohort": "cost-skewed|balanced|equivalence|irrelevant",
      "terminal": "identified|equivalence|budget-exhausted",
      "score": "integer",
      "intervention_cost": "integer",
      "actions": ["canonical-action-code"],
      "action_count": "integer",
      "initial_posterior": "integer",
      "final_posterior": "integer",
      "correct": "bool",
      "teacher_retained": "bool",
      "equivalence_complete": "bool",
      "hypothesis_evaluations": "integer",
      "semantic_work": "integer",
      "engine_cycles": "integer",
      "attributed_units": "integer",
      "cache_hits": "integer",
      "cache_misses": "integer",
      "transcript_digest": "sha256-hex",
      "oracle_agreements": "integer",
      "oracle_disagreements": "integer"
    }],
    "overall": {
      "name": "",
      "fixtures": "integer",
      "identified": "integer",
      "equivalence": "integer",
      "budget_exhausted": "integer",
      "correct": "integer",
      "total_score": "integer",
      "mean_score": "number",
      "total_cost": "integer",
      "mean_cost": "number",
      "mean_actions": "number",
      "accuracy": "number"
    },
    "cohorts": [{
      "name": "cost-skewed|balanced|equivalence|irrelevant",
      "fixtures": "integer",
      "identified": "integer",
      "equivalence": "integer",
      "budget_exhausted": "integer",
      "correct": "integer",
      "total_score": "integer",
      "mean_score": "number",
      "total_cost": "integer",
      "mean_cost": "number",
      "mean_actions": "number",
      "accuracy": "number"
    }]
  }],
  "contrasts": [{
    "name": "information-gain|cost-skewed",
    "treatment": "learned",
    "control": "information-gain-per-cost",
    "statistic": "score-ratio-of-means",
    "relative_reduction": "number",
    "mean_difference": "number",
    "p_value": "number",
    "ci95": ["number", "number"],
    "randomization_replicates": 10000,
    "bootstrap_replicates": 10000,
    "minimum_effect": 0.10,
    "passed": "bool"
  }],
  "gates": {
    "learned_accuracy": "bool",
    "information_gain_accuracy": "bool",
    "primary_reduction": "bool",
    "primary_p_value": "bool",
    "primary_ci": "bool",
    "cost_skewed_reduction": "bool",
    "cost_skewed_p_value": "bool",
    "cost_skewed_ci": "bool"
  },
  "controls": {
    "hidden_twin": "bool",
    "no_credit": "bool",
    "wrong_context": "bool",
    "static_rule": "bool",
    "recomputed_rule": "bool",
    "opaque_alias": "bool",
    "pool_order": "bool",
    "action_order": "bool",
    "cost_perturbation": "bool",
    "occupied_name": "bool",
    "alternate_descriptor": "bool",
    "mutation_inert": "bool",
    "corruption_suite": "bool",
    "deterministic_json": "bool"
  },
  "dynamic_benchmark": {
    "realized_mean_cost": "number",
    "uniform_expected_mean_cost": "number",
    "total_dp_states": "integer",
    "max_dp_states": "integer",
    "total_dp_work": "integer",
    "max_dp_work": "integer"
  },
  "limitations": ["string"]
}
```

Policy order is the seven-item list above; fixture order is seed order; cohort
order is cost-skewed, balanced, equivalence, irrelevant; contrast order is
information-gain then cost-skewed. Empty arrays use `[]`, empty strings use
`""`, and zero counts use `0`; no `null` or omitted field is valid. Status is
computed only after encoding, cap checks, provenance resolution, mechanics,
controls, and gates. Evaluation `all_caps_valid` requires every fixture field
and the reported maxima to satisfy the four per-episode caps, the dynamic state
and work fields to satisfy their caps, every isolated fixture object and the
nonrecord encoding to satisfy their component caps, and `report_bytes` to equal
independent final encoding below its cap. `all_valid=false` or
`all_caps_valid=false` always means `invalid`; otherwise locked is positive iff
every gate is true and null otherwise. Nonlocked mechanically valid reports are
always `valid-null` sensitivity evidence.

## Required controls and tests

- complete legal-SCM enumeration, malformed/cyclic/arity rejection, and
  hand-checked copy/negate/and/or/xor intervention witnesses;
- production/oracle differential evaluation for every legal three-variable
  hypothesis and all six actions;
- exact posterior partitions, equivalence classes, entropy-product ordering,
  rational dynamic-programming values, and 40-rule grammar cardinality;
- hidden-model absence from store, descriptor, names, aliases, queues, and
  production dependencies;
- alias permutation, pool order, action order, cost perturbation, occupied-name,
  alternate descriptor, mutation-on/off, child-VM, and deterministic JSON;
- transcript replay and deletion/duplication/digest/profile/cost/result/
  elimination/posterior corruption;
- missing/extra proposal, partition, score, tie, teacher result, and terminal
  artifacts; false singleton, incomplete equivalence, and teacher-loss attacks;
- no-credit, wrong-context, static, recomputed, passive, random, lexical,
  information-gain, worst-case, and dynamic-optimal matched-budget runs;
- training replay produces the frozen rule and digest exactly; changing any
  training fixture changes the digest and invalidates the freeze; and
- `mise exec -- go test ./...`, `mise exec -- go vet ./...`, scoped race tests,
  and `git diff --check` before the candidate commit.

## Hand-checked witness

For variables `(A,B,C)`, compare:

```text
h1: A=0; B=copy(A); C=copy(B)
h2: A=0; B=0;       C=copy(A)
h3: A=0; B=copy(A); C=OR(A,B)
```

All three passively produce `000`. Under `do(A=1)`, h1 produces `111` and h2
produces `101`; under `do(B=1)`, h1 produces `011` and h2 produces `010`.
Thus passive evidence is insufficient, either declared intervention separates
`h2` from `h1`. Hypotheses h1 and h3 have identical outcomes under all six
single-variable interventions, so `{h1,h3}` is their complete universe
equivalence class and must never be falsely reported as a singleton. Tests list
and independently verify all six outcomes for all three hypotheses.

## Claims and non-claims

A positive result would establish only that Nous's ordinary training-credit
curriculum can select a frozen finite acquisition rule that reduces
intervention cost on a disjoint synthetic distribution while identifying the
hidden SCM up to the declared intervention equivalence. A valid
null still establishes the mechanics of active version-space diagnosis. Neither
result validates a causal graph for Kubernetes, Terraform, or production
telemetry; handles noise, hidden confounding, intervention failure, nonstationary
mechanisms, continuous values, or safety approval; or authorizes a real action.
