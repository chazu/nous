# Active causal diagnosis v3 technical-retry amendment

Status: accepted amendment, revision 3

## Why v3 exists

The single permitted v2 training attempt ran from clean pretraining commit
`44f19d2823307906a8a0393e13a995d00d33f639` and failed closed with
`dependency arrays are not canonical`. The attempt record is preserved under
the Git common directory with state `failed`; its proof records all 12 opened
fixture digests and no published digest. No training report, episode bundle,
score matrix, selected rule, validation result, or locked result was
published or inspected.

V2 is consumed and must never be retried, resumed, overwritten, or treated as
evidence. V3 is a technical retry of the unchanged preregistered experiment,
not an opportunity to tune it.

## Frozen empirical contract

V3 inherits the complete accepted revision-5 v2 contract except for the
explicit protocol and defect-repair deltas below. In particular it preserves
the exact generators, seeds, cohorts, grammar, policies, meters, caps,
controls, statistics, thresholds, limitations, fixture counts, and E-R-C
chronology. Repair and review never read or condition on v2 residue; only the
frozen protected v3 runner may regenerate the same fixtures.

The causal report, bundle, artifact, control, curriculum, and meter schemas
remain their accepted `/v2` semantic schemas. Changing those schemas would be
a new experiment rather than this technical retry.

## Root cause and bounded repair

The exact v2 failure is the empty `forbidden` array. `AuditDependencyBoundary`
constructs a non-nil empty slice, but `rootedDependencyProofAt` copied it with
`append([]string(nil), summary.Forbidden...)`. Appending zero values to a nil
slice yields nil, which encodes as `null`. The dependency validator requires
every ordered array to be non-nil and emits the exact observed error,
`dependency arrays are not canonical`. The recursive Git file list at failed
E was already globally UTF-8 sorted, and an out-of-order file would instead
emit `invalid dependency file`; file ordering was not the failure.

V3 copies `forbidden` with a non-nil destination and explicitly sorts it. It
also makes dependency canonicality errors diagnostic by naming the failing
array, index, nil state, and adjacent values. This changes no acceptance
predicate. A synthetic regression must reproduce the v2 nil-array failure
before the fix and pass after it.

Before a v3 protected attempt is possible, a nonprotected regression test must
construct the dependency proof from the complete tracked tree at the current
commit, invoke the actual `causalv2` dependency validator through a narrow
exported verification wrapper, require byte equality across two independent
reconstructions, assert complete tracked Go plus causal-CUE coverage and
globally sorted unique paths, and prove every ordered array is non-nil. It also
passes a deliberately nil `forbidden` array to that actual validator and
requires the precise rejection.

The same reconstruction is a runtime authorization preflight at exact clean
E3. It runs before attempt/proof creation and before any protected generator
call. Failure consumes nothing and leaves the protected-generator counter at
zero. A test alone is not a substitute for this E3 preflight.

No other production, curriculum, control, credit, generator, oracle, or
statistical behavior may change in this repair.

## Namespace and provenance isolation

The implementation package may remain `internal/causalexpv2`, because the
semantic machinery is unchanged, but all one-shot protocol identities move to
v3:

- plan commit: the accepted commit of this amendment;
- attempt records: `causal-attempt/v3` and `causal-attempt-proof/v3`;
- attempt filenames: `active-causal-diagnosis-v3-<panel>.json` and the matching
  proof filename;
- evidence directory: `docs/evidence/active-causal-diagnosis-v3`;
- staging directory: `.active-causal-diagnosis-v3.staging`;
- result filenames: `active-causal-diagnosis-v3-<panel>.json`;
- replay receipt filename: `active-causal-diagnosis-v3-replay.json`;
- replay input domain: `causal-replay-input/v2` to
  `causal-replay-input/v3`;
- replay-only worker capability: `causal-replay/v2` to `causal-replay/v3`;
- replay success record: `causal-replay-success/v2` to
  `causal-replay-success/v3`;
- candidate-diff digest: `causal-replay-candidate-diff/v2` to
  `causal-replay-candidate-diff/v3`.

The statistical RNG domain in `statistics.go` and every report, bundle,
artifact, curriculum, control, meter, profile, fixture, and certificate `/v2`
domain remain byte-identical semantic identities. Namespace replacement is
never a blind `/v2` to `/v3` rewrite. Tests prove every v2 replay input,
worker record, success record, candidate-diff digest, attempt record, and proof
record is rejected by v3, and the corresponding v3 identity is rejected by v2
test decoders.

The v3 pretraining executable must have empty frozen constants and the v3
evidence path absent. Before the v3 training attempt record is created, the
runner requires all six training/validation/locked v3 attempt and proof record
paths, the evidence directory, staging directory, replay receipt, and every v3
validation/locked result path to be absent. Any collision fails with zero
protected generator calls.

V3 production never opens, parses, decodes, removes, rewrites, or consumes a
v2 attempt, proof, staging directory, result, replay receipt, or evidence file.
Malformed sentinel bytes at v2 common-directory attempt, proof, replay, and
result locations are opaque: tests prove they remain byte-for-byte unchanged,
neither block nor authorize v3, and an independently failing v3 authorization
still makes zero protected generator calls. V2 worktree evidence, staging, and
result paths must be absent at clean E3; authorization checks their absence
without opening them. Tests prove a sentinel at any such worktree path fails
before generation and remains unchanged. The preserved real v2 failed attempt
and opaque fixture digests are metadata only.

The v3 attempt record's plan and pretraining commits must identify the accepted
v3 amendment and clean v3 executable. Reports and bundles retain their semantic
v2 schema versions but bind the v3 plan commit, so v2 bytes cannot verify as v3
evidence.

## Mechanical E to E3 confinement

The consumed empirical baseline is exactly
`44f19d2823307906a8a0393e13a995d00d33f639`. Before creating a v3 training
attempt, an E3 gate compares that commit to clean HEAD and requires the complete
name-status diff to equal this status-sensitive allowlist exactly. `A` means a
new regular file and `M` an existing regular file modified in place:

- `A docs/active-causal-diagnosis-v3-amendment.md`;
- `A internal/causalv2/dependency_verify.go`;
- `A internal/causalv2/dependency_verify_test.go`;
- `M cmd/nous/main.go`;
- `M internal/causalexpv2/dependency_evidence.go`;
- `M internal/causalexpv2/dependency_evidence_test.go`;
- `M internal/causalexpv2/provenance.go`;
- `M internal/causalexpv2/provenance_test.go`;
- `M internal/causalexpv2/publication.go`;
- `M internal/causalexpv2/result.go`;
- `M internal/causalexpv2/replay_hook.go`;
- `M internal/causalexpv2/replay_gate.go`; and
- `M internal/causalexpv2/gates.go`.

The amendment bytes at E3 must equal the accepted plan-commit blob. Existing
allowlisted files must be `M`, the three named new files must be `A`, and every
expected record must be present. `D`, `R`, `C`, `T`, mode/type changes,
unexpected additions, missing changes, and any amendment modification after
acceptance fail before generation.

Every other path is byte-identical to the baseline. The gate separately
requires an empty diff for `domains/causal`, `internal/causalrun`,
`internal/causalcurriculum`, `internal/causaldpproof`, and the causal experiment
generator, executor, training/evaluation executor, control, curriculum,
statistics, meter, evidence, report, and verification files. `freeze.go` is
also byte-identical and empty at E3. Tests inject one change in each protected
group and one non-allowlisted path and require preflight rejection before
generation.

Within allowlisted protocol files, review and tests constrain changes to the
enumerated v3 identities, absence/preflight gates, v2 isolation, and their
tests. Within dependency files, changes are limited to non-nil canonical array
construction, diagnostics, actual-validator exposure through the new narrow
`dependency_verify.go`, whole-tree preflight, and regression tests. The
existing `internal/causalv2/control.go` and its acceptance predicate remain
byte-identical to E. No generator, policy, action, outcome, score, credit,
control result, meter tariff, statistic, or threshold may change.

## Exposure and terminal states

The v2 audit found no evidence directory, result, replay receipt, or staging
directory. The only human-visible values were the generic error, attempt
metadata, and opaque fixture digests. No action, outcome, score, aggregate,
selected rule, report, bundle, or decoded private fixture was observed. V3
regenerates the same fixtures only inside the frozen protected runner; it does
not read them from v2 residue.

Before contextual verification and atomic publication, v3 returns or logs no
report, bundle, score, rule, action, outcome, aggregate, or private fixture
bytes. A mechanically valid or invalid completed training report is atomically
published, the attempt becomes `published`, and the CLI may then print the
published report. Invalid published evidence is preserved but never committed
as R3 and authorizes no replay, validation, locked run, or retry.

A technical failure marks the attempt `failed`, publishes no evidence, and
preserves any staging directory without opening or logging its contents. If a
technical failure leaves staging bytes, or if any empirical item named above
is otherwise exposed, the seed schedule is burned: no later amendment may use
these seeds as a confirmatory technical retry. A same-seed v3 retry is justified
only by the audited v2 absence and non-exposure stated above.

A nonprotected forced-failure harness captures command stdout, stderr, returned
errors, and filesystem effects before publication. It injects sentinel
empirical values and proves no report, bundle, selected-rule, action, outcome,
score, aggregate, or private-fixture bytes appear in any captured channel.
Public panel names, seed-range metadata, commit identities, opaque digests, and
the generic failure class are explicitly non-empirical. The test also proves
no evidence is published and any allowed failure record contains only its
fixed provenance schema.

## V3 execution sequence

1. Accept and commit this amendment.
2. Implement only the nil-array/diagnostic repair, the whole-tree
   regression, and v3 namespace isolation.
3. Run nonprotected executable checks, scoped normal/race tests, focused vet,
   the full repository suite, diff checks, and the protected-core floor check.
4. Obtain unanimous read-only architecture, theory, and experimental review.
5. Commit the clean v3 pretraining executable E3 with empty freeze constants
   and absent v3 evidence paths.
6. Execute v3 training once. Preserve any null, invalid, or failed result.
7. If training is mechanically invalid or fails, create no R3 and terminate
   the protocol. If mechanically valid, commit exactly the two newly added canonical v3
   evidence files as direct child R3 of E3.
8. Insert only the three frozen constants, perform detached-E3 byte replay,
   test, and commit the constants-only direct child C3 of R3.
9. From clean C3 repeat detached replay, then run validation once without
   tuning or source changes. Mechanically invalid validation creates no locked
   authority and terminates the protocol. Only mechanically valid validation
   permits the single locked run.

Any v3 protected failure consumes v3 and requires a new amendment. A passing
technical repair does not make a failed v2 attempt valid.

## Acceptance

V3 is accepted for implementation only when adversarial review agrees that:

- the retry cannot read or tune against unpublished v2 empirical results;
- v2 and v3 attempts, evidence, results, replay receipts, and plan identities
  cannot collide;
- the complete commit-rooted dependency proof is globally canonical before a
  protected attempt;
- all inherited revision-5 evidence and authority guarantees remain intact;
- the implementation delta is mechanically confined to the stated repair and
  namespace reset.
