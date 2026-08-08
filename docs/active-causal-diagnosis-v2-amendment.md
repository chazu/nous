# Active causal diagnosis v2 amendment

## Status and inheritance

Accepted amendment, revision 5. Revision 4 passed adversarial architecture,
theory, and experimental review on 2026-08-07, but its implementation review
found contradictions in the retained-control schema and child-VM control plus
nonconforming cache, benchmark, replay, and core-runtime changes. Revision 5
repairs those proof boundaries before any protected execution. No v2 seed has
been generated or executed, and no v2 attempt, result, or evidence path exists,
so the corrected schemas retain the `/v2` domains and disjoint v2 seed panels.

Revision 5 passed adversarial architecture, theory, and experimental review on
2026-08-07 after five rounds. The commit containing this accepted text is the
new `plan_commit`; the earlier revision-4 commit cannot authorize a v2 attempt.

This amendment inherits every semantic definition, bound, report field,
acceptance threshold, non-claim, and test obligation from the accepted v1 plan
at commit `702617aacc0e89cc9f9db95d6b2d7478431247ec` unless explicitly replaced
below. V1 was executed once and independently classified invalid at commit
`dc36a728f64d1126698ba02f0f9109091124398a`; its rule and digests can never be
frozen. V2 exists only to repair mechanical proof boundaries, not to tune the
causal task or empirical threshold.

## Revision 5 corrective contract

This section replaces the revision-4 passages it names. It changes no task
generator, protected seed, grammar, policy, score, threshold, cohort, or
empirical hypothesis.

### Core-runtime boundary and executed mutation/child controls

The inherited roadmap boundary remains exact: the v2 implementation has no
diff in `internal/engine`, `internal/agenda`, or `internal/dsl/vm.go`, adds no VM
field or hook, and changes no mutation machinery. The dependency verifier
rejects either a forbidden dependency edge or a forbidden-path diff from the
accepted plan commit.

The causal builtin adapter may maintain a package-private, race-safe mapping
from an exact top-level `*dsl.VM` identity to one opaque runner task scope. It
rejects rebinding, is removed when the runner closes, and is absent for every
nested child VM. It contains no fixture, hidden hypothesis, teacher, or oracle
capability. A runtime unit name or copied store is insufficient authority.

`mutation-inert` uses only the existing public engine mutation configuration
and normal focus cycles. Its retained evidence contains the off/on
configuration, semantic projections, and the exact sorted mutant-unit records
observed in the two stores. Off must produce no mutant. On must create at least
one unit through the existing mutation path. Both semantic projections must be
equal and independently valid; a synthetic unit inserted by the control is not
evidence of a mutation attempt.

The mutation inputs are frozen. Both sides first complete the selected-rule
episode for training seed 122001, then set only
`H-Causal-V2-Propose.overallRecord` to `{successes:0,failures:1}` and construct a
fresh ordinary engine on that post-terminal store; the existing constructor
uses RNG seed 42. Both run exactly two normal focus cycles with interval 1,
maximum mutants 1, mutant worth 400, validation disabled, minimum applications
1, and threshold 2.0. Off differs only by `enabled:false`; on uses
`enabled:true`. No RNG read, agenda insertion, unit-focus override, store edit,
or direct mutator call occurs between engine construction and those two cycles.
The retained configuration must equal these values, not merely describe
producer-chosen values.

`child-vm` is a denial control, replacing the revision-4 sentence that required
an identical successful child projection. A real nested child VM sharing the
parent store but lacking an explicit adapter registration attempts one causal
task operation and must fail before adding an artifact, changing an attributed
counter, or consuming teacher evidence. Equality between separately authorized
top-level executions remains covered by deterministic JSON and replay controls.

### Exact cache and reported cache provenance

The exact cache payload remains the revision-4 object
`{action,posterior_artifact_digest,cells,E,W,H,C,R}`. The posterior field is the
outer digest of the current sealed posterior artifact, not a semantic-set
digest. `R` is the exact repeat feature. A reusable in-memory semantic cache may
be keyed by `(profile_digest,semantic_posterior_set_digest,action)`, but every
use materializes the exact step-local sealed payload above and rederives `R`.

Production records a cache status at each six-action cache lookup. Statuses are
ordered step-major, then by the six canonical action codes returned by
`causal.Actions()`. Every consumed step contributes exactly six statuses; the
selected status copied into the transcript is the status at that step's
selected-action position. Evaluation `cache_hits|cache_misses` are reconstructed
from the complete verified cache artifacts and satisfy
`cache_hits + cache_misses = 6*action_count`. They never use dynamic-program
memo counters. Static/recomputed retained rows contain the complete treatment
and control cache traces. Recomputed evidence has exactly `6*action_count`
statuses, all `miss`; its work is no lower than cached work.

### Complete dynamic benchmark meter

For each dynamic-optimal fixture, one production dynamic-policy instance owns
one DP meter. The meter includes table construction, every lookup along the
realized teacher trajectory, and every lookup while separately simulating each
possible hidden member of the initial posterior. The uniform expected cost is
computed by those same simulations, not by an unmetered second policy. The
retained fixture DP item is independently reconstructed and must remain within
1,873 states and 193,117 work. Omitting the realized trajectory, any hidden
member, or any table lookup makes the fixture mechanically invalid.

### Control bundle and separately retained control evidence

The control bundle stays the exact revision-4 three-field object and has no
fourth field:

```json
{"control_bundle_version":"causal-control-bundle/v2","certificates":["18 exact certificates"],"control_bundle_digest":"sha256-hex"}
```

Both training and evaluation reports insert `control_evidence` and
`control_evidence_digest` immediately after `control_bundle_digest`.
`control_evidence_digest` is present both inside the object and beside it and
the two values must be identical. Its digest uses
`causal-control-evidence/v2`, NUL, and the exact object with its inner digest
empty:

```json
{
  "control_evidence_version":"causal-control-evidence/v2",
  "selected_rule":"canonical-rule-code",
  "static_rule":"canonical-rule-code",
  "static_matrix":["exact-matrix-row"],
  "recomputed_matrix":["exact-matrix-row"],
  "mutation":"exact-mutation-proof",
  "child_vm":"exact-child-vm-proof",
  "corruption":"exact-corruption-proof",
  "no_credit":"exact-no-credit-proof",
  "dependency":"exact-dependency-proof",
  "control_evidence_digest":"sha256-hex"
}
```

The following are normative ordered object schemas. `counter-array` always
means exactly 15 nonnegative integers in the frozen counter-field order.
`base64url` means unpadded RFC 4648 base64url of canonical bytes. Empty arrays
are `[]`, never `null`.

```text
cache-trace = {statuses:["hit|miss"],hits:integer,misses:integer}

matrix-row = {
  seed:integer,
  fixture_digest:sha256-hex,
  treatment_episode_digest:sha256-hex,
  treatment_certificate_digest:sha256-hex,
  control_episode_digest:sha256-hex-or-empty,
  control_certificate_digest:sha256-hex-or-empty,
  treatment:exact-control-result,
  control:exact-control-result,
  treatment_meter_counts:counter-array,
  control_meter_counts:counter-array,
  treatment_cache:cache-trace,
  control_cache:cache-trace
}

mutation-config = {
  enabled:bool,interval:integer,max_mutants:integer,mutant_worth:integer,
  validate_only:bool,min_applics:integer,mutation_threshold:canonical-number
}
mutant-record = {
  name:string,mutant_of:string,source_slot:string,operation:string,
  program_digest:sha256-hex,worth:integer
}
mutation-proof = {
  fixture_digest:sha256-hex,off_config:mutation-config,on_config:mutation-config,
  off_result:exact-control-result,on_result:exact-control-result,
  off_mutants:[mutant-record],on_mutants:[mutant-record],
  off_meter_counts:counter-array,on_meter_counts:counter-array
}

child-vm-proof = {
  fixture_digest:sha256-hex,profile_digest:sha256-hex,
  operation:"causal-v2-task-valid?",artifacts_before:integer,
  artifacts_after:integer,meter_counts_before:counter-array,
  meter_counts_after:counter-array,teacher_calls_before:integer,
  teacher_calls_after:integer,failure_code:"child-vm-unauthorized"
}

corruption-case = {
  name:canonical-case-name,mutation_descriptor:canonical-case-name,
  mutated_bytes_digest:sha256-hex,rejection_code:canonical-failure,
  meter_counts:counter-array
}
corruption-proof = {
  enumerator_version:"causal-corruption-enumerator/v2",
  fixture_bytes:base64url,profile_bytes:base64url,
  baseline_artifacts:[base64url],case_count:integer,
  case_set_digest:sha256-hex,cases:[corruption-case]
}

no-credit-proof = {
  central_profile_bytes:base64url,
  certificate_digests:["exactly-480-sha256-hex-in-seed-major-rule-major-order"],
  artifact_bytes:["exactly-1041-base64url-sealed-artifacts"],
  aggregates:["exactly-40-rule-aggregate-objects"],
  central_transcript:[],task_meter_items:["exactly-525-task-meter-items"],
  counts:counter-array,resolution:"unresolved",winner_ties:[],selected_rule:"",
  terminal_transcript_digest:"64-ascii-zeroes"
}

dependency-parameter = {
  function:qualified-function,parameter_index:integer,type:canonical-go-type
}
dependency-file = {
  path:repository-relative-path,source_sha256:sha256-hex,
  imports:[sorted-import-path],exported_function_parameters:[dependency-parameter]
}
runner-field = {name:string,type:canonical-go-type,hidden_bearing:bool}
dependency-proof = {
  audited_commit:40-hex,audited_roots:[sorted-repository-relative-path],
  files:[dependency-file],runner_methods:[sorted-method-signature],
  runner_fields:[runner-field],teacher_methods:[sorted-method-signature],
  lookups:integer,forbidden:[sorted-canonical-violation]
}
```

Both matrices contain exactly 12 rows in training-seed order. Every static row
has all four episode/certificate digests nonempty and matched to committed
training evidence. Every recomputed row has nonempty treatment digests matched
to the committed cached selected-rule episode/certificate and empty control
digests because the cache-disabled execution has no committed training record.
No proof contains a literal `passed`; verification derives it from exact
evidence and fresh execution.

Static and recomputed certificates embed the exact seed-122001 row's actual
fixture digest and treatment/control results. They never put an aggregate or
secondary digest into a profile- or transcript-digest field. Certificate meter
counts are the exact componentwise sum of the 12 retained rows. A certificate
has no standalone authority: its result and `passed` value are accepted only
after its evidence section reconstructs.

The static rule is exactly `causal.Rules()[0]` in frozen grammar order, without
lexical sorting. Both sides run on all 12 already-opened private fixtures from
the verified training bundle, in seed order, with fresh stores, equivalent
independent teachers, identical hidden models, costs, ceilings, and budgets,
and separately signed acquisition profiles. For each side, the underlying fresh
training episode and application certificate must byte-equal the corresponding
committed seed/rule records named by the row digests; its `ControlResult` must
then byte-equal the canonical projection derived from that verified episode.
Learned/static semantic equality or learned superiority is not required.
Static passes only when the complete declared baseline matrix reproduces
exactly and both sides are independently valid. Recomputed uses the selected
rule on both sides and requires exact equality of actions, outcomes, posterior
transitions, costs, terminal, score, and accuracy across all 12 rows.

The no-credit proof retains the exact fields above. Its central transcript is
empty because no selection occurs, and its terminal digest is 64 zeroes. Fresh
verification reruns all 480 admissions from the exact training episodes and
certificates and byte-compares every retained result.

Dependency evidence is rooted at an exact 40-hex audited commit and contains
the sorted audited roots; for every source file, its relative path, SHA-256,
sorted imports, and sorted exported function/callback parameters; the sorted
runner fields and methods; the sorted teacher methods; lookup count; and the
complete forbidden list. Verification enumerates and rereads the Git tree at
that commit and byte-compares the reconstructed proof. Worktree counts alone
are insufficient.

The only `audited_roots` value is `["."]`. Files are every tracked regular
`*.go` file plus every tracked `domains/causal/**/*.cue` file in
`git ls-tree -r --full-tree <audited_commit>`; symlinks and submodules are
forbidden. Repository-relative paths use slash separators, no leading `./`, and
no empty, `.` or `..` segment. All ordered strings use ascending UTF-8 byte
order. Imports are unquoted Go import paths. Go types are
`go/types.TypeString(t,q)` where `q(pkg)` returns `pkg.Path()`. A qualified
function is `<package-import-path>.<function-name>` or
`<package-import-path>.(<receiver-type>).<method-name>`; method signatures use
the same type printer. Parameter indices are zero-based and retain their
function association. This same complete file set is compared with the
worktree and rejects any diff under `internal/engine`, `internal/agenda`, or
`internal/dsl/vm.go` from the accepted plan commit.

Corruption evidence retains one canonical baseline and the ordered case
records. A case name is also the closed enumerator's exact mutation descriptor;
the verifier deterministically reconstructs its mutated ledger, checks
`mutated_bytes_digest`, reruns rejection, and checks its meter counts. Mutated
ledgers are not duplicated per case. Changing, deleting, duplicating,
reordering, or re-signing evidence fails the evidence digest or fresh verifier.

The corruption baseline is the nonprotected development fixture at seed 112001,
generator index zero, executed with `P=H;M=gain;S=C` and its own signed
development profile and private teacher. Its preregistration witness terminates
`identified` after exactly two actions and has these exact kind cardinalities:
`descriptor-snapshot:5,observation:1,posterior:3,cache:12,partition:12,
proposal:12,score:12,tie:2,selection:2,authorization:2,result:2,elimination:7,
consumption:2,transcript:2,terminal:1`. Contextual verification regenerates this
development fixture and exact ledger; it never substitutes a protected
training, validation, or locked episode.

The corruption enumerator is frozen, not evidence-selected. That baseline must
contain every episode artifact kind, in this exact kind order:
`descriptor-snapshot,observation,posterior,cache,partition,proposal,score,tie,
selection,authorization,result,elimination,consumption,transcript,terminal`.
For each kind it selects the first ledger artifact of that kind. On those 15
representatives, in kind order, it mutates every outer field in ascending UTF-8
field-name order, then recursively mutates every payload object field in that
order, descending into objects and only the first object of a nonempty array,
then emits delete, duplicate, and forged-charge-index cases. Finally it scans
the complete ledger in index order and emits one adjacent-swap case for the
first occurrence of each distinct ordered pair of adjacent artifact kinds;
later occurrences of the same pair are skipped.

A string appends `-corrupt`, a JSON number adds one, a bool negates, an empty
array becomes `["corrupt"]`, a nonempty array mutates element zero, and an object
gains `"unexpected_corruption_field":true`. Case names are respectively
`kind-<kind>-field-<path>`, `kind-<kind>-<payload-path>`,
`delete-kind-<kind>`, `duplicate-kind-<kind>`, `forge-kind-<kind>`, and
`reorder-kind-<left>-<right>`. The 15 schemas yield exactly 135 outer-field and
81 payload-field cases, plus 45 delete/duplicate/forge cases and at most 225
distinct ordered kind-pair cases: `case_count <= 486`. Each canonical case is
capped at 2,048 bytes, so the full cases array is at most
`486*2048 + 485 + 2 = 995815` bytes including brackets, within its 1 MiB subcap. The case-set digest
uses `causal-corruption-enumerator/v2`, NUL, and the exact ordered case-name
array; `case_count` equals its length. Verification reconstructs the entire
array and rejects a subset, extra, duplicate, or reorder.

The exact manifest inserts
`"control_evidence_byte_cap":4194304` immediately after
`control_bundle_byte_cap`, followed by the five aggregate subcaps in the exact
manifest. The control bundle retains its 2 MiB cap. Control evidence has its own
4 MiB cap, the nonrecord shell retains its 1 MiB cap, and the complete report
retains its 16 MiB cap. Its maximum record allowance is exactly
`1572864+524288+1048576+262144+524288 = 3932160`, leaving 262,144 bytes for the
fixed evidence shell under 4 MiB. Actual canonical nested-array and complete
object lengths must independently satisfy every subcap and total cap. Golden
maximum-width tests prove the bounds.

Each evidence subcap measures canonical JSON of the named complete array,
including brackets, element commas, quotes, and base64url strings. A group with
multiple arrays is the sum of those complete encoded-array lengths; separators
between object fields remain in the fixed shell and are not added twice. The
exact groups are:

- `no_credit_artifacts`: `no_credit.artifact_bytes`;
- `corruption_baseline`: `corruption.baseline_artifacts`;
- `corruption_cases`: `corruption.cases`;
- `dependency_files`: `dependency.files`; and
- `other_records`: `static_matrix`, `recomputed_matrix`,
  `mutation.off_mutants`, `mutation.on_mutants`,
  `no_credit.certificate_digests`, `no_credit.aggregates`,
  `no_credit.central_transcript`, `no_credit.task_meter_items`,
  `dependency.audited_roots`, `dependency.runner_methods`,
  `dependency.runner_fields`, `dependency.teacher_methods`, and
  `dependency.forbidden`.

Every control-evidence record-array path belongs to exactly one group. The
fixed evidence shell is the canonical evidence object with all listed arrays
replaced by `[]` and its digest empty; it is capped at 262,144 bytes. Because
the five group measurements conservatively include brackets also present in
the shell, shell plus group caps is an upper bound on the final object.

The training-digest input inserts `control_evidence_digest` immediately after
`control_bundle_digest`. Evaluation self-digests cover the complete control
evidence and its digest.

### Contextual verification and authority

Strict canonical report decoding proves schema and self-consistency only and
returns no authority. Training freeze, replay, validation, locked execution,
and publication accept only package-private verified values returned by
contextual verifiers.

The contextual training verifier accepts repository root plus exact report and
bundle bytes. It reruns the training matrix from the bundle's already-opened 12
private fixtures, reconstructs the static and recomputed matrices, reruns the
full credit and no-credit curricula, and audits dependency evidence at the
recorded pretraining commit. The contextual evaluation verifier loads the
committed training report/bundle named by `training_report_commit`, invokes the
contextual training verifier, and uses only already-opened training fixtures or
the fixed development witness to freshly reconstruct static, recomputed,
no-credit, corruption, mutation, and child-VM evidence. Static and recomputed
use all 12 training fixtures; every other single-context control uses training
seed 122001 except corruption, which uses development seed 112001. It never
invokes a validation or locked generator for a control. Dependency evidence alone is
regenerated at `implementation_commit`. Only then may it reconstruct controls,
mechanics, status, and report digest. A structurally valid report never mints a
capability by itself.

### Detached replay worker

The single-use parent replay capability remains mintable only after committed R
evidence and the exact dirty-R or direct-child-C constants state verify. It
opens the exact 12 private fixtures from R and writes a canonical transient
input through inherited pipe file descriptor 3. Its exact field order is
`{replay_input_version,plan_commit,pretraining_commit,evidence_commit,
training_digest,bundle_digest,fixtures,corruption_fixture,replay_input_digest}`;
the version is `causal-replay-input/v2`, `fixtures` contains exactly 12 private
fixtures in training-seed order, and `corruption_fixture` is the exact private
development-112001 witness. The self-digest uses that domain, NUL, and the
object with its digest empty.

The fixed detached-E replay worker accepts no arguments, panel, seed, generator
option, environment authorization flag, attempt capability, or path. It reads
one canonical input to EOF from inherited pipe FD 3 and writes through inherited
FD 4, an already-opened empty temporary directory. It resolves its own worktree
as clean detached E. All output uses no-follow, exclusive `openat`; it cannot
call any generator. It can only regenerate training evidence from the supplied
12 protected fixtures and supplied development witness. Direct invocation may process caller-supplied fixtures if it
constructs both descriptors, but cannot discover or generate protected bytes
and confers no acceptance. The parent alone byte-compares both output files to
R, consumes its in-process capability, and may record the diagnostic receipt.

Old environment-variable or seed-bearing replay requests, a noncanonical input,
a symlink, wrong E/R/evidence digest, changed fixture, nonempty output directory,
attempted publication, or output outside the inherited directory fails before
regeneration or leaves byte comparison false. The pipe is consumed once; reuse
of the parent capability fails before a second worker is started.

### Revision 5 acceptance tests

In addition to every inherited test, revision 5 requires synthetic rejection
tests for every schema, evidence, cache, dependency, and replay attack named
above. It requires `git diff -- internal/engine internal/agenda
internal/dsl/vm.go` to be empty; scoped normal and race tests for parallel
top-level VM isolation, rebinding, close/reuse, and child denial; an exact cache
payload golden including `posterior_artifact_digest` and `R`; production cache
accounting independent of DP memoization; full DP-meter reconstruction; static
baseline binding to the committed training matrix; full no-credit and rooted
dependency reconstruction; direct replay-worker and transient-input attacks;
maximum-width byte-cap goldens; `mise exec -- go test ./...`; focused
`mise exec -- go vet`; and `git diff --check`. No test may call a protected
panel generator. Authorization tests stop before generation and assert a zero
generator-call counter. Tests use hand/development fixtures or exact fixture
bytes supplied as test data; only the real single-use runner opens a protected
panel.

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
  "control_evidence_byte_cap": 4194304,
  "control_evidence_no_credit_artifacts_byte_cap": 1572864,
  "control_evidence_corruption_baseline_byte_cap": 524288,
  "control_evidence_corruption_cases_byte_cap": 1048576,
  "control_evidence_dependency_files_byte_cap": 262144,
  "control_evidence_other_records_byte_cap": 524288,
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
`causal-control-evidence`,
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
"credit_enabled":"bool","profile_digest":"sha256-hex"}`. `training_key` is SHA-256 of
`causal-central-profile/v2`, NUL, the plan commit, NUL, and pretraining commit.
`profile_digest` is SHA-256 of `causal-central-profile/v2`, NUL, and the exact
object with `profile_digest:""`. Training uses `credit_enabled:true`; the
distinct no-credit profile uses `credit_enabled:false`. The central descriptor payload has
`expected_rules:40`, the ordered 12 training seeds, `expected_certificates:480`,
and the same signed `credit_enabled` value. It is immutable configuration. A narrow unsealed
central cursor holds phase
`initializing|admitting|aggregating|selecting|terminal`; barriers and the sealed
central transcript corroborate every phase transition, and neither selection
nor scoring may read the cursor as evidence.
The no-credit control uses that distinct signed control profile.

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
digests include the bundle digest and the separately retained control-evidence
digest. Report booleans reconstruct jointly from certificates and the complete
freshly verified control evidence; neither object has standalone authority.

An exact control result is
`{"profile_digest":"sha256-hex-or-empty","actions":["canonical-action"],
"outcomes":["three-bit-string"],"posterior_digests":["sha256-hex"],
"costs":["integer"],"terminal":"terminal-or-empty","score":"integer",
"failure_code":"canonical-failure-or-empty","transcript_digest":"sha256-hex-or-empty"}`.
Thus retained evidence is embedded rather than represented by an undefined
secondary digest.

Alias, presentation, proposal-order, and mutation controls require
identical normalized semantic projections—actions, outcomes, posterior
transitions, costs, terminal, and score—while each context-specific profile and
transcript digest must independently verify; raw digest equality is not
expected when public context changes. Child-VM instead has the revision-5
fail-before-evidence denial semantics. Hidden
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
`"control_evidence":<exact-control-evidence-object>`,
`"control_evidence_digest":"sha256-hex"`,
`"task_meter_items":["exact-task-meter-item"]`, and
`"task_meter_items_digest":"sha256-hex"`. The evaluation report makes the
same six insertions immediately after `status`. A task meter item is exactly
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
every evaluation policy's fixtures; and these exact paths inside control
evidence: `static_matrix`, `recomputed_matrix`, `mutation.off_mutants`,
`mutation.on_mutants`, `corruption.baseline_artifacts`, `corruption.cases`,
`no_credit.certificate_digests`, `no_credit.artifact_bytes`,
`no_credit.aggregates`, `no_credit.central_transcript`,
`no_credit.task_meter_items`, `dependency.audited_roots`, `dependency.files`,
`dependency.runner_methods`, `dependency.runner_fields`,
`dependency.teacher_methods`, and `dependency.forbidden`. Each listed array is
replaced once by `[]`; subcategories are not independently removed.
Fixed aggregate meter,
rule, cohort, contrast, and limitation arrays are part of the nonrecord shell.
To compute `nonrecord_bytes`, replace only those record arrays with `[]`, put
`00000000` in both byte fields, put 64 ASCII zeroes in both occurrences of the
control-evidence digest, and, for an evaluation report, put 64 ASCII zeroes in
`report_digest`; serialize and record the length. Reinsert the record
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
2 MiB control-bundle cap, the 4 MiB control-evidence cap, and the 1 MiB
nonrecord cap, both reports remain below 16 MiB. Training's 480 application
certificates add at most `480*1024 + 479 = 491999` bytes, so its five in-report
allowances total `491999+1522124+2097152+4194304+1048576 = 9354155` bytes. Locked
evaluation adds at most `448*8192 + 447 = 3670463` fixture-record bytes to
`983999+2097152+4194304+1048576`, totaling 11,994,494 bytes. The 8 MiB training
episode bundle is a separate file, not embedded in the report. Actual canonical
lengths and the final report cap are still mandatory.

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
  "control_evidence_digest": "sha256-hex",
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

A locked capability is minted only after contextual evaluation verification
loads and verifies committed training evidence, freshly regenerates the full
validation matrix and every control from already-opened training fixtures,
audits dependency evidence at the same clean C, and requires byte equality with
the mechanically valid validation report and its self-digest. A mechanical
validation failure or changed C prevents locked generation and forces v3;
threshold results never permit tuning.

V2 evidence paths are exactly
`docs/evidence/active-causal-diagnosis-v2/training.json` and
`docs/evidence/active-causal-diagnosis-v2/training-episodes.json`. The clean
detached pretraining worktree writes both files into one absent sibling staging
directory, fsyncs and strictly reopens both, then publishes the containing
directory with a single no-replace rename. This makes two-file publication
atomic. A surviving staging directory is failed evidence, never resumed or
overwritten.

Tests generate only hand fixtures and development seeds. No test invokes a
training, validation, or locked generator; authorization tests stop before the
generator call and prove its counter remains zero. The real protected entry
points require their respective provenance tokens. After v2 plan acceptance
the sequence is:

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
8. from clean direct-child C, synchronously repeat the detached-E fixture-fed
   byte replay and require both files to equal R; the dirty-R receipt is checked
   only for diagnostic consistency and grants no authority;
9. only after that fresh replay, run validation without changes from clean C,
   contextually reopen and freshly regenerate its matrix and controls, then run
   locked once from the same clean C. Panel reports are written outside the
   repository, so neither run changes the candidate commit.

Any v2 training failure is preserved as invalid and requires v3 with another
disjoint seed set.
