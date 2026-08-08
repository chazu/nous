# Active causal diagnosis v2 amendment

## Status and inheritance

Accepted amendment, revision 4. Adversarial architecture, theory, and
experimental reviews all passed on 2026-08-07. No v2 seed has been generated or
executed.

This amendment inherits every semantic definition, bound, report field,
acceptance threshold, non-claim, and test obligation from the accepted v1 plan
at commit `702617aacc0e89cc9f9db95d6b2d7478431247ec` unless explicitly replaced
below. V1 was executed once and independently classified invalid at commit
`dc36a728f64d1126698ba02f0f9109091124398a`; its rule and digests can never be
frozen. V2 exists only to repair mechanical proof boundaries, not to tune the
causal task or empirical threshold.

## Exact v2 manifest, domains, and seed reset

The complete v2 manifest, in field order, is:

```json
{
  "experiment_version": "active-causal-diagnosis/v2",
  "generator_version": "three-binary-scm/v2",
  "hypothesis_version": "ordered-dag-mechanisms/v1",
  "acquisition_version": "lexicographic-pairs/v1",
  "oracle_version": "independent-scm-enumerator/v2",
  "teacher_version": "opaque-single-response/v2",
  "training_version": "credit-curriculum/v2",
  "baseline_version": "exact-partition-policies/v2",
  "cost_version": "intervention-cost/v1",
  "report_version": "causal-diagnosis-report/v2",
  "statistics_version": "paired-resampling/v1",
  "profile_version": "causal-profile/v2",
  "development_seeds": {"start": 112001, "count": 16, "step": 1},
  "training_seeds": {"start": 122001, "count": 12, "step": 1},
  "validation_seeds": {"start": 132001, "count": 32, "step": 1},
  "locked_seeds": {"start": 142001, "count": 64, "step": 1},
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
  "certificate_replay_semantic_work_cap": 3932160,
  "post_selection_replay_semantic_work_cap": 3932160,
  "oracle_audit_work_cap": 4000000,
  "control_work_cap": 262144,
  "control_attributed_unit_cap": 18000,
  "dynamic_state_cap": 531441,
  "dynamic_work_cap": 4000000,
  "episode_semantic_work_cap": 8192,
  "teacher_evaluation_cap": 10,
  "report_byte_cap": 16777216,
  "fixture_record_byte_cap": 8192,
  "evaluation_fixture_base_byte_cap": 6144,
  "evaluation_fixture_meter_items_byte_cap": 2048,
  "training_fixture_byte_cap": 16384,
  "application_certificate_byte_cap": 1024,
  "task_meter_item_byte_cap": 1024,
  "control_certificate_byte_cap": 4096,
  "control_bundle_byte_cap": 2097152,
  "training_episode_base_byte_cap": 6144,
  "episode_meter_items_byte_cap": 2048,
  "training_episode_report_byte_cap": 8192,
  "training_episode_bundle_byte_cap": 8388608,
  "nonrecord_report_byte_cap": 1048576,
  "maximum_limitations": 32,
  "limitation_byte_cap": 512,
  "locked_accuracy_gate": 1.0,
  "minimum_primary_reduction": 0.10,
  "integrity_contract": "budgeted-transcript/v2",
  "duplicate_policy": "canonical-code-deduplicate-before-profile",
  "cache_policy": "episode-policy-local-semantic-partition-cache/v2",
  "alpha": 0.05,
  "bootstrap_replicates": 10000,
  "randomization_replicates": 10000,
  "bootstrap_indices_zero_based": [249, 9749],
  "contrast_seed_rule": "active-causal-diagnosis/v2|locked|<information-gain|cost-skewed>|<randomization|bootstrap>",
  "tie_policy": "all-ties-reported-first-semantic-code-executed",
  "mutation_enabled": false
}
```

Digest domains `causal-profile`, `causal-central-profile`, `causal-public-token`, `causal-public-fixture`, `causal-private-fixture`,
`causal-artifact`, `causal-authorization`, `causal-hypothesis-set`,
`causal-partition`, `causal-transcript-entry`, `causal-empty-transcript`,
`causal-application-certificate`, `causal-rule-applications`,
`causal-training-episode`, `causal-training-episode-bundle`,
`causal-control-certificate`, `causal-control-bundle`,
`causal-training-digest-input`, `causal-semantic-key`, `causal-meter-array`,
`causal-task-meter-items`, `causal-central-transcript-event`, and
`causal-diagnosis-report` all use `/v2`.
The hypothesis/action canonical
encodings, acquisition rule codes, intervention cost meaning, and resampling
algorithm retain `/v1` exactly as shown in the manifest because they did not
change.

The cohort assignment, 40-rule grammar, 480-episode curriculum, seven policies,
cost ranges, score, 10% effects, resampling algorithms, and semantic work caps
are unchanged. V2 adds only encoding subcaps: 16,384 bytes for a private
training fixture, 6,144 bytes for an inherited episode body, and 2,048 bytes for
its compact meter items. Cost-skewed generation draws exactly three values from its low,
medium, and high ranges; it must not consume unused balanced-cost draws.

## Capability split

Online execution moves to `internal/causalrun`. That package may import the
production vocabulary, engine, agenda, seed loader, and store. It must not
import `internal/causaloracle`, `internal/causalexp`, generator code, or a type
containing a hidden hypothesis. Its constructor accepts:

```text
NewEpisode(publicFixtureBytes, profileBytes, teacher Teacher) Runner
Teacher.Respond(opaqueToken, canonicalActionCode) (threeBitOutcome, error)
```

`PublicFixture` has this exact field order:

```json
{
  "seed": "integer",
  "generator_attempt": "integer",
  "cohort": "cost-skewed|balanced|equivalence|irrelevant",
  "aliases": ["three-position-ordered-strings"],
  "costs": ["three-position-ordered-integers"],
  "passive_outcome": "three-bit-string",
  "pool": ["32-sorted-canonical-hypothesis-codes"],
  "presentation": ["32-integers-forming-a-permutation-of-0-through-31"],
  "initial_posterior": ["8-to-32-sorted-canonical-hypothesis-codes"],
  "uniform_random_actions": ["exactly-ten-canonical-action-codes"],
  "opaque_token": "sha256-hex",
  "fixture_digest": "sha256-hex"
}
```

The public token is hidden-independent: its exact preimage object is
`{"token_version":"causal-public-token/v2","panel":"panel-enum",
"seed":"integer","generator_attempt":"integer"}` and its value is
SHA-256 of `causal-public-token/v2`, NUL, and that object. A private teacher registry binds that token to a hidden
model outside public bytes. Counterfactual-hidden twins reuse identical public
fixture/profile/token bytes and substitute only that private binding.
Presentation code `i` is `pool[presentation[i]]`; strict decoding proves every
integer from zero through 31 occurs exactly once. The
public fixture digest uses `causal-public-fixture/v2`, NUL, and the exact object
with `fixture_digest:""`, thereby binding aliases, the sorted pool and its
presentation permutation, random
actions, and token. It contains no hidden code or index. The private bundle
fixture is `{"public_fixture":<exact-object>,"hidden_hypothesis":"canonical-code",
"private_fixture_digest":"sha256-hex"}` and uses the corresponding `/v2`
domain with its self-digest empty.

Each canonical private training fixture is capped at 16,384 bytes. The exact
bundle allowance is `12*16384 + 480*8192 + 490 + 1048576 = 5177834`, below the
unchanged 8 MiB bundle cap; 490 is the exact inter-record comma allowance for the
two record arrays after their brackets are counted in the nonrecord shell. The
8,192-byte `fixture_record_byte_cap` remains the cap for
the compact evaluation-report fixture record, not the private training fixture.

The private teacher factory lives outside `causalrun`, closes over the hidden model,
and returns only the narrow interface. The online runner may invoke only
`Respond`, exactly once for each authorized selected action. The prohibition on
partition/filter/prediction applies to oracle and generator capabilities;
production CUE and pure production vocabulary must compute public predicted
partitions. Learned, information-gain, worst, and lexical policies execute in
production CUE. Uniform-random consumes the digest-bound public action prefix.
Dynamic-optimal is a hidden-free production `ActionPolicy` over public
posterior, costs, and consumed actions under `internal/causalrun`; the independent
oracle reimplements and audits its chosen actions post-terminal.

Only after a production terminal does `internal/causalexp` pass the public
fixture, action/response transcript, and private audit record to the independent
oracle. Dependency tests walk imports and method sets. A trap teacher records
every call and fails any duplicate, wrong-token, or unselected action.

## Canonical bytes and strict decoding

Production and oracle each implement the inherited exact typed schemas
independently. Encoding is always compact JSON, `SetEscapeHTML(false)`, with no
trailing newline. Every decoder uses `DisallowUnknownFields`, proves EOF,
re-encodes, and requires byte equality. Golden tests cover every domain prefix,
field order, empty self-digest, `<...>` manifest string, fixed-width byte field,
and representative Unicode alias.

`nonrecord_bytes` and `report_bytes` are eight-character decimal strings. Both
are `00000000` in the nonrecord preimage; their replacements preserve width.
The final report and bundle paths contain exactly canonical bytes, and file
length must equal the recorded value. The verifier reopens files from disk and
recomputes every digest and byte count. Report generation is atomic via a
temporary sibling file and rename only after full verification.

Profile validation decodes the exact inherited profile object, checks the full
v2 manifest and public fixture digest, clears and recomputes its digest, and
compares every descriptor field at every proposal, response, update, and
terminal boundary. A nonempty digest is never sufficient.

## Sealed episode protocol

Every attributed unit implements the inherited artifact envelope: exact profile
digest, episode key, step, kind, semantic key, full SHA-256 authoritative
digest, and unique monotonically increasing charge index. Allocation checks an
occupied name before reuse: identical sealed content is idempotent; any other
content or unsealed occupant is a mechanical collision failure. Full digests,
not truncated hashes, are used in names.

The exact envelope is
`{"artifact_version":"causal-artifact/v2","profile_digest":"sha256-hex",
"scope":"episode-key-or-training-key","step":"integer","kind":"enum-below",
"semantic_key":"canonical-string","payload":"kind-specific-object",
"charge_index":"integer","artifact_digest":"sha256-hex"}`. Its digest is
SHA-256 of `causal-artifact/v2`, NUL, and the exact envelope with
`artifact_digest:""`; its name is `Causal.<kind>.<full-artifact-digest>`.
Charge indices start at zero and are gap-free within one profile/episode or one
central training profile. A narrow unsealed runtime cursor contains only the
current state and latest descriptor-snapshot digest; it is outside attributed
evidence and cannot be read for scoring. Sealed snapshots exist only at verified
boundaries: the initial `ready`, `awaiting-teacher` after selection and
authorization, and the next `ready|terminal` after update. The
`proposing`, `response-present`, and `updating` cursor states are transient and
are corroborated by the selection/authorization, result, and consumption
artifacts. Thus every verified boundary transition, not every cursor
transition, materializes a snapshot: exactly one initial snapshot and exactly
two snapshots per consumed action.

Episode allocation order is observation, initial posterior, initial `ready`
descriptor snapshot; then, step-major, `cache,proposal,partition,score,tie,
selection,authorization,awaiting-teacher-descriptor-snapshot,result,
elimination,posterior,consumption,transcript,ready-or-terminal-descriptor-snapshot,
terminal`. The two snapshot positions are mandatory even if intermediate CUE
tasks are drained in batches. Central allocation order is `central-descriptor`,
then all 480 `certificate` artifacts in seed-major/rule-major order, then all 40
`central-rule` artifacts in semantic-code order, then for each certificate its
`application` immediately followed by `credit`, then all `aggregate`, all
winner `central-tie`, one `central-selection`, and 521 `transcript` artifacts.

For every kind, `semantic_key` is the lowercase SHA-256 hex of
`causal-semantic-key/v2`, NUL, and compact canonical JSON
`{"kind":"exact-envelope-kind","payload":<the exact payload below>}`. This is
the normative per-kind derivation: episode descriptor snapshots use the
`descriptor-snapshot` kind; episode transcript events use `transcript`;
curriculum descriptor/rules/ties/selection use `central-descriptor`,
`central-rule`, `central-tie`, and `central-selection`; and every other row uses
its literal row name. A verifier rejects a semantic key not rederived from the
typed payload, so no producer-chosen canonical string remains.

Kind payloads have these exact field orders; bracketed values are fixed-order
arrays of canonical strings or integers:

- descriptor-snapshot: `{previous_snapshot_artifact_digest,state,aliases,costs,presentation,initial_posterior_artifact_digest,
  posterior_digest,acquisition_code,total_cost,action_count,remaining_evaluations,
  remaining_work,remaining_cycles,remaining_units}`;
- observation: `{outcome}`, posterior: `{hypotheses,semantic_set_digest}`;
- cache: `{action,posterior_artifact_digest,cells,E,W,H,C,R}`;
- partition: `{action,posterior_artifact_digest,cells}`;
- proposal: `{action,cache_artifact_digest}`, score: `{action,rule_code,cache_artifact_digest}`;
- tie: `{action,score_artifact_digest}`, selection: `{action,tie_artifact_digests}`;
- authorization: `{profile_digest,episode,step,action,selection_artifact_digest,opaque_token,
  authorization_digest}`;
- result: `{authorization_artifact_digest,action,outcome}`;
- elimination: `{hypothesis,result_artifact_digest}`;
- consumption: `{result_artifact_digest,posterior_artifact_digest}`;
- episode-scope transcript and terminal: the exact inherited transcript object and
  `{terminal,posterior_digest,total_cost,action_count,transcript_digest}`;
- central-descriptor:
  `{central_profile_digest,expected_rules,expected_seeds,expected_certificates,credit_enabled}`;
- central rule: `{rule_code}`, certificate: `{certificate_bytes,certificate_digest}`;
- application: `{seed,rule_code,certificate_digest,score,terminal,cost}`;
- credit: `{application_artifact_digest,delta}`, aggregate: the exact inherited rule object;
  central tie: `{rule_code,aggregate_artifact_digest}`; and central selection:
  `{selected_rule,tie_artifact_digests}`;
- central-scope transcript: the exact central event object defined below.

The initial ready snapshot uses 64 ASCII zeroes for
`previous_snapshot_artifact_digest`; every later snapshot uses the immediately
prior snapshot artifact's outer digest. Snapshot `state` is exactly
`ready|awaiting-teacher|terminal`.

`certificate_bytes` is a JSON string containing unpadded RFC 4648 base64url of
the exact canonical application-certificate bytes. `certificate_digest` is
the application certificate's verified digest, not a digest of the base64
spelling. Strict verification decodes the string, rejects padding or a
noncanonical re-encoding, and then verifies the decoded certificate bytes.

Every field ending `_artifact_digest` is exactly the referenced envelope's
outer `artifact_digest`; it has no second preimage. `semantic_set_digest` uses
`causal-hypothesis-set/v2`. The authorization payload's
`authorization_digest` is its independently defined self-digest below; all
other payload-level self-digests are removed. Selection identity is its outer
artifact digest, and cache identity is its outer artifact digest.

The authorization object is both the authorization payload and an independently
canonical typed object. Its self-digest uses `causal-authorization/v2`. The
proposal verifier is read-only and returns canonical authorization bytes; the
runner enqueues one descriptor-declared CUE task that materializes those exact
bytes and then the `awaiting-teacher` descriptor snapshot, in that order. The
verifier never writes the store.

The authorization-to-response sequence is exact: the proposal verifier returns
bytes; the runner enqueues exactly one authorization task; the engine drains; a
fresh read-only authorization verifier requires one matching sealed artifact,
cursor state `awaiting-teacher`, and empty agenda; only then may the runner call
`Teacher.Respond`. The runner inserts one immutable result candidate, enqueues
one pre-update validation task, and drains. A fresh read-only response verifier
requires its authorization/action/profile/step tuple, requires no existing
consumption for that result, and returns a decision without writing the store;
one CUE update task then materializes eliminations, posterior, consumption,
transcript, and next descriptor snapshot. The final ready/terminal verifier is
read-only and must pass before another task or terminal report is allowed.

The implementation must materialize descriptor, passive observation, initial
posterior, six proposals, six partitions, six scores, every exact tie, one
selection, response authorization, teacher result, eliminations, posterior,
cache entries, transcript entry, and terminal as applicable. It implements the
inherited state machine including the explicit `updating` state.

Fresh verifiers are separate Go functions that read only public descriptor and
store artifacts. Before a teacher call, the proposal verifier requires six
complete exact proposal/partition/score triples, the complete tie set, one
selection, a selection digest, no result, no eligible proposal/update task, and
an empty agenda. It then creates an authorization bound to profile, episode,
step, action, and selection digest. The response verifier requires the exact
authorization tuple and action, requires no prior consumption, and rejects any
second or stale response. The subsequent CUE task alone materializes the one
immutable consumption artifact. The ready/terminal verifier recomputes the posterior,
eliminations, cost, transcript prefix, caps, terminal completeness, empty
agenda, and absence of pending artifacts before the runner may enqueue the sole
next task.

Transcript entries, posterior sets, partitions, and all other digests use the
exact inherited schemas. Corruption tests alter, delete, duplicate, reorder, or
forge each field and artifact kind. Standalone `nous run -domain causal` starts
a real seed descriptor using development seed 112001 and stops at verified
`awaiting-external-teacher`.

Relative to v1's 664/6,866 ledger, v2 adds ten authorizations, ten consumptions,
and twenty transition snapshots beyond the initial descriptor: the conservative
artifact bound is 704. The validation accounting is fixed: 64 full
profile/manifest fields are checked and cached once, then each of ten actions
checks 14 fields at the `awaiting-teacher` snapshot, seven authorization fields,
four result-tuple fields, four consumption-link fields, and 14 fields at the
next `ready|terminal` snapshot. That is `64 + 10*(14+7+4+4+14) = 494`
field validations. Replacing v1's 100-field allowance with 494 and adding 40
materializations gives `6,866 + (494-100) + 40 = 7,300`; the registered
conservative bound is 7,500.
Both remain below the unchanged 1,000-unit and 8,192-work episode caps.
Teacher work is separately capped at ten evaluations. Fresh certificate replay
uses the ordinary per-episode caps for each isolated replay; certificate replay
and post-selection replay each have an independent 3,932,160 aggregate cap.
Oracle audit has a 4,000,000-work cap. All controls together have 262,144 work
and 18,000 attributed-unit caps. Oracle audit, teacher, DP, and control work are
reported separately and never charged to the primary score. Reports include
per-category totals and maxima for production, teacher, certificate replay,
post-selection replay, oracle audit, DP, and controls, plus the governing cap
for each.

## Measured work and semantic cache

An episode-local `WorkMeter` owns named counters for every `causal-work/v2`
tariff event. Pure builtins receive the meter through the descriptor context and
increment at the operation site: each SCM evaluation, partition assignment,
cell accumulation, exact comparison, posterior membership check, materialized
artifact, transcript field, and profile field. No report code estimates or
assigns these totals.

Each action partition is computed once per `(profile,posteriorDigest,action)` and
stored in a sealed cache artifact containing the partition and features. Every
proposal, comparison, and tie scan consumes that cache entry. Recomputed-rule
control disables reuse while charging every recomputation. Production, teacher,
post-terminal audit, and DP meters are distinct.

The dynamic oracle memoizes terminal, finite, and nonfinite states and counts a
state on first memo insertion, including the root. It charges one lookup for
every memo access, one Q evaluation per considered action, one cell term per
cell, and every realized/uniform-simulation table lookup. The analytical
1,873-reachable-state and 193,117-work proof applies to every pool of at most 32.
A combinatorial enumerator visits every legal canonical size-8-or-larger subset
of each passive-outcome class without running the expensive DP, and mechanically
checks the analytical proof premises: size at most 32, at most eight nonempty
cells/action, and the `min(8^k,32)` state bound. A second tiny DP enumerator
exhaustively compares exact state values, selected actions, state counts,
state-expansion charges, and work counts on every posterior induced by all
action subsets of a frozen corpus containing all 72 models and all 58
observational-equivalence classes.
Full DP execution runs on each preregistered fixture only after that panel's
token is authorized; no test touches a future panel.

## Verified CUE curriculum

The central profile is the exact canonical object
`{"central_profile_version":"causal-central-profile/v2","manifest":<exact-v2-manifest>,
"plan_commit":"40-hex","pretraining_commit":"40-hex","training_key":"sha256-hex",
"profile_digest":"sha256-hex"}`. `training_key` is SHA-256 of
`causal-central-profile/v2`, NUL, the plan commit, NUL, and pretraining commit.
`profile_digest` is SHA-256 of `causal-central-profile/v2`, NUL, and the exact
object with `profile_digest:""`. The central descriptor payload has
`expected_rules:40`, the ordered 12 training seeds, `expected_certificates:480`,
and `credit_enabled:true`. It is immutable configuration. A narrow unsealed
central cursor holds phase
`initializing|admitting|aggregating|selecting|terminal`; barriers and the sealed
central transcript corroborate every phase transition, and neither selection
nor scoring may read the cursor as evidence.
The no-credit control uses a distinct control profile and `credit_enabled:false`.

The post-episode verifier regenerates each training fixture and reruns the
episode in a fresh store before issuing a certificate. Certificate bytes enter
the curriculum store as sealed `CausalApplicationCertificate` units. The driver
seeds only the central descriptor at charge index zero and these 480 certificate
units at indices 1 through 480, in seed-major/rule-major order. A CUE
initialization heuristic enumerates the frozen grammar through one-rule-at-a-time
pure refinements and materializes exactly 40 `CausalAcquisitionRule` units at
indices 481 through 520 in semantic-code order. The driver cannot preassign or
reserve future rule indices. A CUE
admission heuristic calls only a strict one-certificate verifier, enforces the
unique `(rule,seed)` key, and materializes the application and credit-delta
units. A complete-matrix barrier proves exactly 40 rules by 12 seeds before any
aggregate task is eligible.

CUE heuristics materialize all 40 aggregates, all exact winner-tie units, one
selection unit, and the central transcript. The driver cannot set a selected
rule slot. Selection reads verified credit-delta units, then applies worth,
exhaustion, and semantic-code ordering. All central artifacts have sealed
envelopes and charge indices in the canonical order above: each application is
immediately followed by its credit delta, followed by aggregates, ties,
selection, and transcript artifacts. The work meter supplies actual curriculum work,
cycle, and attributed-unit totals; the preregistered bounds remain ceilings,
not values to copy into a report.

The conservative v2 central ledger is 2,083 units: one descriptor, 40 rules,
480 certificates, 480 applications, 480 credit deltas, 40 aggregates, 40 winner
ties, one selection, and 521 transcript events. Its work bound is 18,979 (the
v1 revision-6 bound plus 480 certificate materializations), and at most 525
tasks execute: one rule initialization, 480 admissions, 40 aggregates, and four
barriers. These remain within 4,096 units, 32,768 work, and 2,048 cycles.

A post-selection verifier strictly decodes all 480 certificates and the bundle,
replays all episodes again in fresh stores, requires every field and canonical
byte to equal its original episode before certificate issuance, reconstructs
every application, delta, aggregate, tie, and selection, and requires exact
byte/digest equality. Admission clears and recomputes each certificate digest,
rejects unknown fields, and requires equality with the bundle episode—not mere
self-consistency. The no-credit control admits the identical verified
480-certificate matrix and materializes the same rule/application evidence, but
suppresses only credit deltas; it must leave no selection or a declared
unresolved tie. It cannot pass by disabling admission or rule initialization.

## Executed controls and validity

No validity or control boolean may be assigned a literal success value. Every
boolean is the result of a named verifier or executed trial whose evidence is
retained. V2 executes all inherited controls, including hidden twins, wrong
context, static and recomputed rules, opaque aliases, pool/action permutations,
cost perturbation, occupied names, alternate descriptors, mutation on/off,
child VM, corruption, deterministic JSON, no-credit, and dependency boundary.

Controls occur in this fixed order: `hidden-twin,wrong-context,static-rule,
recomputed-rule,opaque-alias,presentation-order,proposal-order,cost-perturbation,
occupied-name,alternate-descriptor,mutation-inert,child-vm,stale-response,
duplicate-response,corruption-suite,deterministic-json,no-credit,dependency`.
Each emits
`{"control_version":"causal-control-certificate/v2","name":"control-enum",
"fixture_digest":"sha256-hex-or-empty","treatment_evidence":"exact-control-result",
"control_evidence":"exact-control-result","observed":"canonical-result-string",
"passed":"bool","meter_counts":["15-integers-in-counter-field-order"],
"work":"integer","certificate_digest":"sha256-hex"}`. `work` must equal the
`total_work` position in `meter_counts`.
Its self-digest uses the matching `/v2` domain. The ordered array plus
`control_bundle_digest` forms `{"control_bundle_version":"causal-control-bundle/v2",
"certificates":[...],"control_bundle_digest":"sha256-hex"}` and is capped at
2 MiB; each certificate is capped at 4,096 bytes. Training and evaluation
digests include the bundle digest, and report booleans are reconstructed only
from these certificates.

An exact control result is
`{"profile_digest":"sha256-hex-or-empty","actions":["canonical-action"],
"outcomes":["three-bit-string"],"posterior_digests":["sha256-hex"],
"costs":["integer"],"terminal":"terminal-or-empty","score":"integer",
"failure_code":"canonical-failure-or-empty","transcript_digest":"sha256-hex-or-empty"}`.
Thus retained evidence is embedded rather than represented by an undefined
secondary digest.

Alias, presentation, proposal-order, mutation, and child-VM controls require
identical normalized semantic projections—actions, outcomes, posterior
transitions, costs, terminal, and score—while each context-specific profile and
transcript digest must independently verify; raw digest equality is not
expected when public context changes. Hidden
twins require identical production bytes through the first authorization.
Wrong-context, alternate-descriptor, occupied-name, stale/duplicate response,
and every corruption case must fail closed before consuming new evidence. Cost
perturbation must invalidate the stale profile/cache and, after a newly signed
profile, recompute cost-derived features and score. Recomputed-rule requires
identical actions and accuracy with separately metered work no lower than the
cached rule. Static rule must reproduce the semantic-first rule's declared
episode matrix. No-credit has the behavior specified above. Dependency must
prove the forbidden import/method graph is empty.

`all_valid` is the conjunction of strict schema/provenance, exact panel and
matrix cardinality, all profile/boundary/transcript/artifact/certificate/bundle
verifiers, zero oracle disagreement, every cap, and every required control.
There is no separate path that can emit `valid` while a reported control or
mechanical field is false.

Meters are verifier-owned Go capabilities, never mutable unit slots. Sealed
artifacts contain only remaining-budget snapshots. Episode and replay evidence
contains every named category count and total; an independent verifier
reproduces them. Recomputed-rule keeps the ordinary 8,192 production-work cap;
an overrun is an executed failed control and makes the panel invalid rather than
receiving a larger budget.

## Exact v2 evidence and report schema deltas

V2 starts from the complete inherited v1 schemas and applies only these ordered
typed replacements; fields not named retain their inherited position and type.
This insertion list is itself normative.

The training episode bundle changes `bundle_version` to
`causal-training-episode-bundle/v2`, uses the exact private-fixture object above
for `fixtures`, and inserts `meter_items` immediately before each episode's
`all_caps_valid`. `meter_items` is the compact fixed-order episode array defined
below; aggregate meter objects exist only in report mechanics.

The training report changes `report_version` to `causal-training-report/v2` and
inserts, immediately after `episode_bundle_bytes`, `"control_bundle":<exact
control-bundle-object>` then `"control_bundle_digest":"sha256-hex"`,
`"task_meter_items":["exact-task-meter-item"]`, and
`"task_meter_items_digest":"sha256-hex"`. The evaluation report makes the
same four insertions immediately after `status`. A task meter item is exactly
`{"name":"certificate-replay|post-selection-replay|curriculum",
"subject":"canonical-unique-task-key","counts":["15-integers-in-counter-field-order"]}`.
Training emits 480 items for each replay name and one curriculum item for each
of its at-most-525 executed tasks; evaluation emits the fresh verification
replay items it actually executes. Subject keys are certificate digests for
replays and exactly `%06d:<task-kind>` for curriculum. Items are ordered first
by the two replay names in meter order and their seed-major/rule-major
certificate order, then by curriculum task index. The digest is SHA-256 of
`causal-task-meter-items/v2`, NUL, and the exact ordered array. Its
`mechanical` object is replaced, in order, by:

```json
{
  "all_valid": "bool",
  "credit_recomputed": "bool",
  "selection_verified": "bool",
  "oracle_agreements": "integer",
  "oracle_disagreements": "integer",
  "meters": ["exact-meter-object"],
  "max_descriptor_bytes": "integer",
  "max_training_episode_report_bytes": "integer",
  "max_application_certificate_bytes": "integer",
  "nonrecord_bytes": "eight-digit-zero-padded-decimal",
  "report_bytes": "eight-digit-zero-padded-decimal",
  "all_caps_valid": "bool"
}
```

The evaluation report changes `report_version` to
`causal-diagnosis-report/v2` and replaces `mechanical` with:

```json
{
  "all_valid": "bool",
  "dependency_boundary": "bool",
  "profile_valid": "bool",
  "transcript_valid": "bool",
  "training_freeze_valid": "bool",
  "oracle_agreements": "integer",
  "oracle_disagreements": "integer",
  "meters": ["exact-meter-object"],
  "max_descriptor_bytes": "integer",
  "max_fixture_record_bytes": "integer",
  "nonrecord_bytes": "eight-digit-zero-padded-decimal",
  "report_bytes": "eight-digit-zero-padded-decimal",
  "all_caps_valid": "bool"
}
```

Each evaluation policy fixture appends the same compact `meter_items` field
immediately after inherited `oracle_disagreements`, followed by a new
`all_caps_valid` boolean reconstructed from those item checks. Its shell with
`meter_items:[]` is capped at 6,144 bytes and its between-brackets meter content
at 2,048 bytes; both are checked independently and compose under the existing
8,192-byte `fixture_record_byte_cap`.

The report inserts `"report_digest":"sha256-hex"` immediately before
`limitations`. Byte reconstruction follows one algorithm. Record arrays are the
training report's applications and the one complete `task_meter_items` array;
the training bundle's fixtures and episodes; the control bundle's certificates;
and every evaluation policy's fixtures. The task array is replaced once by
`[]`; its replay and curriculum categories are not removed independently.
Fixed aggregate meter,
rule, cohort, contrast, and limitation arrays are part of the nonrecord shell.
To compute `nonrecord_bytes`, replace only those record arrays with `[]`, put
`00000000` in both byte fields, and, for an evaluation report, put 64 ASCII
zeroes in `report_digest`; serialize and record the length. Reinsert the record
arrays and final nonrecord field, leave `report_bytes` zero and the evaluation
digest as 64 zeroes, serialize, and record that length as `report_bytes`.
Both substitutions preserve final field width. Finally, the evaluation digest
is SHA-256 of `causal-diagnosis-report/v2`, NUL, and the complete canonical
report with `report_digest:""` and both byte fields already final. Insert the
digest, serialize once, and require the actual length to equal `report_bytes`.
Control certificates are record arrays for this calculation even though the
control bundle is embedded. Validation and locked reports are durably written without newline
to `<git-common-dir>/nous-results/active-causal-diagnosis-v2-<panel>.json` using
exclusive create; the validation report digest is the locked-capability input.

Each task meter item is capped at 1,024 bytes. Training has at most 1,485 such
items, so reinsertion costs at most `1485*1024 + 1484 = 1522124` bytes.
Evaluation has exactly 960 replay items, costing at most
`960*1024 + 959 = 983999` bytes. Added to the inherited record maxima, the
2 MiB control-bundle cap, and the 1 MiB nonrecord cap, both reports remain below
16 MiB; actual canonical lengths and the final report cap are still mandatory.

An episode `meter_items` array has all eight names in the fixed order below.
Each item is
`{"name":"meter-name","active":"bool","counts":["15-integers-in-counter-field-order"]}`;
the integer positions are, exactly, `scm_evaluations,partition_assignments,
cell_accumulations,rule_comparisons,posterior_checks,artifact_materializations,
transcript_fields,profile_fields,memo_states,memo_lookups,q_evaluations,
table_lookups,engine_cycles,
attributed_units,total_work`. This compact evidence records actual counts only;
it does not duplicate aggregate maxima or caps in every episode. Strict episode
verification reproduces the 15 values and computes `all_caps_valid` against the
manifest. The canonical episode shell with `meter_items:[]` is capped at 6,144
bytes and the serialized meter-item contents between those brackets at 2,048
bytes, so the combined record remains under the 8,192-byte training-episode
cap. Golden tests exercise
the maximum-width decimal representation permitted by every counter cap.

Every aggregate meter object in training or evaluation report mechanics has
this exact field order. `totals` and `maxima` are exact
counter objects with the field order shown; maxima are the largest count in one
episode (or one control/curriculum task when that meter is not episodic):

```json
{
  "name": "production|teacher|certificate-replay|post-selection-replay|oracle-audit|dp|controls|curriculum",
  "episodes": "integer",
  "totals": {
    "scm_evaluations": "integer",
    "partition_assignments": "integer",
    "cell_accumulations": "integer",
    "rule_comparisons": "integer",
    "posterior_checks": "integer",
    "artifact_materializations": "integer",
    "transcript_fields": "integer",
    "profile_fields": "integer",
    "memo_states": "integer",
    "memo_lookups": "integer",
    "q_evaluations": "integer",
    "table_lookups": "integer",
    "engine_cycles": "integer",
    "attributed_units": "integer",
    "total_work": "integer"
  },
  "maxima": "the-same-exact-counter-object",
  "caps": {
    "per_episode_evaluation_cap": "integer",
    "aggregate_evaluation_cap": "integer",
    "per_episode_state_cap": "integer",
    "aggregate_state_cap": "integer",
    "per_episode_work_cap": "integer",
    "aggregate_work_cap": "integer",
    "per_episode_unit_cap": "integer",
    "aggregate_unit_cap": "integer",
    "per_episode_cycle_cap": "integer",
    "aggregate_cycle_cap": "integer"
  },
  "valid": "bool"
}
```

Meter arrays always use the eight-name order shown. An inapplicable cap is zero;
a nonzero per-episode cap is checked against the corresponding maximum and a
nonzero aggregate cap against the corresponding total. Evaluations mean
`scm_evaluations`; state caps govern `memo_states`; work, unit, and cycle caps
govern their namesake counters. `valid` is computed from every applicable
comparison and from exact `totals == sum(items)` and `maxima == max(items)`;
it is never assigned directly. When `N=0`, `episodes` is zero and both totals
and maxima are the all-zero 15-counter object; aggregate caps are checked
against zero. Teacher aggregate evaluation cap is
`episodes*10`; replay aggregate caps are 3,932,160; production, DP, oracle,
controls, and curriculum use the manifest caps at their declared per-episode or
aggregate scope. `meter_digest` is SHA-256 of `causal-meter-array/v2`, NUL, and
this exact array. During nonrecord reconstruction the fixed eight-meter array
remains present.

Every charged work event increments exactly one of the first twelve counter
positions. The equation is normative:

```text
total_work = scm_evaluations + partition_assignments + cell_accumulations
           + rule_comparisons + posterior_checks + artifact_materializations
           + transcript_fields + profile_fields + memo_states + memo_lookups
           + q_evaluations + table_lookups
```

`engine_cycles` and `attributed_units` are independent resource counters and
are not added again. A DP state is charged once at first memo insertion, every
memo access charges `memo_lookups`, every considered action charges
`q_evaluations`, and every realized or uniform-simulation table access charges
`table_lookups`. No named tariff exists outside these twelve positions.

An inactive item must contain fifteen zeroes and is excluded from `episodes`,
totals, maxima, and `N`; an active item is included even when all its measured
counts happen to be zero.

The source-item rule is exact. Training production, teacher, oracle-audit, and
DP items are the bundle episodes' compact `meter_items`; evaluation items for
those names are each policy fixture's compact `meter_items`. Certificate and
post-selection replay plus curriculum items come from `task_meter_items`.
Control items come only from the ordered control certificates' `meter_counts`.
Each active source item is consumed once. The aggregate verifier groups by name,
requires unique subjects where applicable, sums and maximizes every counter,
and rejects a total or maximum not equal to that reconstruction.

This table is the complete cap assignment. `N` is the number of source items
for that meter in that report; `N*x` is checked as exact integer multiplication,
not copied from report data. A dash encodes zero/inapplicable.

| meter | evaluation per / aggregate | state per / aggregate | work per / aggregate | units per / aggregate | cycles per / aggregate |
| --- | --- | --- | --- | --- | --- |
| production, training | 4,096 / 2,000,000 | - | 8,192 / `N*8192` | 1,000 / `N*1000` | 5,000 / `N*5000` |
| production, evaluation | 4,096 / `N*4096` | - | 8,192 / `N*8192` | 1,000 / `N*1000` | 5,000 / `N*5000` |
| teacher | 10 / `N*10` | - | - | - | - |
| certificate-replay | 4,096 / `N*4096` | - | 8,192 / 3,932,160 | 1,000 / `N*1000` | 5,000 / `N*5000` |
| post-selection-replay | 4,096 / `N*4096` | - | 8,192 / 3,932,160 | 1,000 / `N*1000` | 5,000 / `N*5000` |
| oracle-audit | - | - | - / 4,000,000 | - | - |
| dp | - | 531,441 / `N*531441` | 4,000,000 / `N*4000000` | - | - |
| controls | - | - | - / 262,144 | - / 18,000 | - |
| curriculum | - | - | - / 32,768 | - / 4,096 | - / 2,048 |

Both replay rows require `N=480`; otherwise the report is invalid. Production,
teacher, and oracle-audit training each require `N=480`, with all three
corresponding items active in every training episode. Evaluation production,
teacher, and oracle-audit each require `N=7*panelSeedCount`, with those three
items active in every evaluation fixture. Evaluation DP contains one active
item per distinct public fixture, owned by that fixture's `dynamic-optimal`
policy record; the other six policy records carry inactive zero DP items.
Training DP requires `N=12`, one active item per distinct training fixture,
owned by that seed's episode for the lexically first semantic rule code; its
other 39 episodes carry inactive zero DP items. Controls require `N=18`, one item from each
certificate in the fixed control order. Any cardinality mismatch is invalid,
including an audit item made inactive to suppress work or disagreements.

The v2 training-digest input replaces the inherited object with:

```json
{
  "digest_input_version": "causal-training-digest-input/v2",
  "manifest": "exact-v2-manifest",
  "plan_commit": "40-hex",
  "pretraining_commit": "40-hex",
  "central_profile_digest": "sha256-hex",
  "episode_bundle_digest": "sha256-hex",
  "control_bundle_digest": "sha256-hex",
  "task_meter_items_digest": "sha256-hex",
  "meter_digest": "sha256-hex",
  "fixture_digests": ["sha256-hex"],
  "application_certificates": ["exact-v2-application-object"],
  "rule_aggregates": ["exact-v2-rule-object"],
  "winner_ties": ["canonical-rule-code"],
  "selected_rule": "canonical-rule-code"
}
```

The central transcript contains exactly 521 events: 480 certificate admissions,
40 aggregate completions, then selection. An event is
`{"event_version":"causal-central-transcript/v2","index":"integer",
"previous_digest":"sha256-hex-or-64-zeroes","kind":"admission|aggregate|selection",
"subject_artifact_digest":"sha256-hex","work_before":"integer",
"work_after":"integer","event_digest":"sha256-hex"}`. The self-digest uses
`causal-central-transcript-event/v2`; index zero uses 64 ASCII zeroes and later
events use the prior digest. Its terminal digest is the final event digest.
Each event is also the exact payload of one central-scope `transcript` artifact
allocated after the selection artifact, in event-index order. An admission
event's subject is its credit artifact, an aggregate event's subject is its
aggregate artifact, and the selection event's subject is the central selection
artifact. Episode-scope transcript artifacts continue to use the inherited
episode transcript payload and can never decode as central events.

## Provenance gates

The v2 training runner itself resolves `HEAD`, requires a clean worktree, proves
the accepted v2-plan commit is an ancestor, and records that exact `HEAD` as the
pretraining commit. It accepts no caller-supplied substitute. Validation and
locked runners reopen and verify the committed training report/bundle, frozen
rule/report-commit/digest constants, clean candidate `HEAD`, and ancestry.

Before any panel generator is called, the runner creates with exclusive
create/no-replace semantics an attempt record under
`<git-common-dir>/nous-attempts/active-causal-diagnosis-v2-<panel>.json`. Its
exact object is `{attempt_version,plan_commit,pretraining_commit,panel,
seed_range,executable_commit,created_utc,state}` with `created_utc` encoded as
RFC3339 UTC and state
`started|published|failed`. The record is fsynced before generation. An existing
record or evidence directory refuses execution. Interruption or failure updates
the same record to `failed`, preserves staging bytes, and forces v3 for
training; validation/locked are also single-use and cannot be retried in place.
No generic exported generator accepts future-panel names without the matching
unconsumed attempt capability.

A distinct read-only replay capability is mintable only after R and its two
evidence digests verify and the current diff from R is the allowlisted constants
edit. It binds E, R, seed range, and both committed file digests; creates no
attempt, cannot publish evidence or issue a report, cannot update the attempt
record, writes only to a fresh temporary directory, and exposes only a
byte-equality result. Thus the empirical attempt remains single-use. Hand
fixtures and repeatable development diagnostics use a separate
`diagnostic-development` capability that cannot produce an acceptance status,
certificate, or evidence file.

A locked capability is minted only from a mechanically valid validation report
self-digest bound to the same clean C and v2 manifest. A mechanical validation
failure or changed C prevents locked generation and forces v3; threshold
results never permit tuning.

V2 evidence paths are exactly
`docs/evidence/active-causal-diagnosis-v2/training.json` and
`docs/evidence/active-causal-diagnosis-v2/training-episodes.json`. The clean
detached pretraining worktree writes both files into one absent sibling staging
directory, fsyncs and strictly reopens both, then publishes the containing
directory with a single no-replace rename. This makes two-file publication
atomic. A surviving staging directory is failed evidence, never resumed or
overwritten.

Tests may generate only hand fixtures and development seeds before training.
Training, validation, and locked generator entry points require their respective
provenance tokens; tests prove unauthorized access fails without exposing a
fixture. After v2 plan acceptance the sequence is:

1. commit the accepted amendment;
2. implement and adversarially review the corrected pretraining executable;
3. commit clean pretraining executable E with empty frozen constants and absent
   v2 evidence paths;
4. in a clean detached worktree at E, execute training once, atomically publish
   and independently reopen/replay both canonical files, then commit those
   unchanged as evidence commit R, a child of E;
5. on R insert only selected rule, R's commit, and training digest;
6. the replay verifier reads E from the committed report, creates a fresh
   detached worktree at E, regenerates under that recorded identity, and
   requires both generated files to equal R byte-for-byte. It also proves the
   candidate diff from R contains only the three allowlisted constants;
7. run all tests, commit that constants-only diff as candidate C, and require a
   clean C `HEAD`;
8. run validation without changes from clean C, then run locked once from the
   same clean C. Panel reports are written outside the repository, so neither
   run changes the candidate commit.

Any v2 training failure is preserved as invalid and requires v3 with another
disjoint seed set.
