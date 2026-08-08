# Active causal diagnosis v6 cache diagnostic plan

Status: accepted on 2026-08-08 after three adversarial review rounds.

Review record: the initial revision was rejected because it compared different
tree states, did not directly exercise the candidate correction, and omitted
receipt bindings. Revision 1 closed the causal and binding gaps but was
rejected for single candidate sampling and incomplete boundary/transplant
proof. Revision 2 was explicitly accepted without blockers or major concerns
by the architecture/integration, domain-theory, and experimental-validity
reviewers.

This plan authorizes exactly one non-empirical diagnostic execution after the
failed v5 replay. It implements Gate 0 of the accepted
[v2 vocabulary research program](vocabulary-research-program-v2.md). It does
not authorize another replay, validation access, locked access, or any read of
R3 evidence bytes.

## Fixed provenance

- governing program commit: `e6dd2fef97a80e24ceae5b2bb531d09ae310c0aa`;
- v5 plan P5: `630a0ce8c36118957d008fbf7132af2424ee91f0`;
- v5 implementation X5: `3bce9836b60c9fc419696f0a82450af3a158ec19`;
- E3: `89a9221e97dd17d6ba220a22a12d4c0328417ffb`;
- R3 identity only: `7482d1b9712f49cb988623a87ec8bb1c34667a26`;
- failed-v5 receipt SHA-256:
  `2e3c67fe774b288d72ea8672f9b07a063533a377a1394ba5de082f9c316a6577`;
- selected rule, training digest, bundle digest, fixtures, reports, and all
  empirical scoring identities remain immutable and unread by this diagnostic.

The v5 operator is never rerun. The diagnostic uses the actual replay worker
source at E3 but a fixed development-derived input. It creates no replay,
validation, or locked attempt receipt.

## Question and hypotheses

The only answerable question is whether the v5 descriptor-relative metadata
walker has a reproducible implementation discrepancy on the fixed diagnostic
tree, and whether the already specified `unix.ReadDirent` candidate both fixes
that discrepancy and preserves every accepted v5 predicate. The deleted v5
tree remains unknown. No diagnostic result classifies its historical metadata
as admissible or forbidden.

The frozen hypotheses are:

- H0, diagnostic-tree contract rejection: stable reference, current, and
  candidate walker passes all reject the post-worker metadata;
- H1, reproducible descriptor implementation defect: stable reference passes
  and the candidate walker accept, while two current walker passes reject;
- H2, non-reproduction: stable reference, current, and candidate passes all
  accept;
- H3, unsafe auditor disagreement: the stable verdict tuple is any other
  combination; and
- H4, diagnostic failure: the worker, receipt, environment, stability, or
  harness contract does not reach one of the four comparison outcomes.

Only H1 authorizes the single recovery amendment described below, because it
demonstrates a current-walker defect and directly exercises its predetermined
replacement. H0 is a valid rejection of only the diagnostic tree. H2 through
H4 are inconclusive. H0 and H2 through H4 leave Phase 2 `invalid` and terminate
the vocabulary program without another diagnostic or recovery attempt.

## Exact diagnostic input

The canonical input is `causal-replay-input/v3` with Phase 2 plan commit
`b4183595a9769882ad4673c606e9a35560cfb95c`, E3 as pretraining commit, R3 as
evidence-commit identity, 64 lowercase `a` digits as the deliberately
non-matching training digest, and 64 lowercase `b` digits as the deliberately
non-matching bundle digest.

The corruption fixture is diagnostic development fixture seed 112001,
generator attempt 0. The twelve workload fixtures are generated with
`NewDiagnosticDevelopmentCapability` from development seeds 112001 through
112012 and generator attempts 0 through 11. In index order, each public seed is
then replaced with training seed 122001 through 122012, its opaque token is
recomputed for panel `training`, and its public and private fixture digests are
re-signed. No generated choice depends on v5 or R3 output.

Relabeling changes only seed, panel-derived token, and integrity digests. Every
hidden and public workload payload remains development-generated. The operator
may call only `NewDiagnosticDevelopmentCapability`; protected training
generation and replay capabilities are source-forbidden.

This recipe has:

- replay-input digest:
  `f579b1ac920f57f8062dc0e030e7b708135d711161048d1ea44573d5eff21d7f`;
- canonical byte SHA-256:
  `08ca98db6a5975707124e579bad34acdc9361a74f15e9e08142116b10db71649`;
- canonical byte length: 91,054.

The implementation must regenerate those three identities before creating the
diagnostic receipt. Their mismatch is a zero-worker-start terminal failure.

The E3 worker is expected to execute the complete development-derived training
workload and reject the deliberately non-matching training digest before
publishing outputs. The harness records only the normalized terminal
`expected-digest-rejection`; it neither exposes worker output nor reads replay
output files.

This workload does not recreate v5. Source/AST tests establish only that the
digest rejection occurs after `regenerateTrainingEvidence`, and that the later
bundle verification and output-writing path has no CUE runner, Git command,
HOME/XDG, or external-process operation. That control makes the diagnostic a
reasonable exercise of the cache-producing workload; it does not reveal or
infer the deleted v5 metadata.

## Exact environment and executable

The diagnostic builds the standalone replay worker from a clean detached E3
worktree with the same pinned Go driver, resolved module metadata, private
module/build caches, and source/worktree verification used by v5. `go run` is
forbidden. The worker receives the same exact eleven-variable v5 environment,
including private `PATH`, `HOME`, and `XDG_CONFIG_HOME`; their absolute base is
normalized to `$DIAGNOSTIC_ROOT` before the environment digest is recorded.

The diagnostic operator itself is a separately built X6 executable invoked
with no arguments from the clean repository root. Its receipt binds the
operator commit and executable SHA-256, the E3 worker SHA-256, the normalized
environment digest, and the exact input identities. `debug.ReadBuildInfo` must
report `vcs.revision` equal to X6 and `vcs.modified=false`, and the operator must
pass the existing pinned runtime and protected Git checks. Both executable
digests are recomputed immediately before `Command.Start`.

The resolved E3 build worktree contains only the accepted resolved `go.mod` and
`go.sum` changes and is verified with `verifyResolvedReplayWorktree`. The
separate detached E3 execution worktree remains clean and is verified with
`verifyCleanReplayWorktree` before and after execution. Building and
preflighting either executable occurs before receipt creation; worker execution
occurs only after the exclusive `started` receipt is durable.

## Independent walker bracket

After worker exit, the operator first revalidates the exact environment,
private Git link, base identity, and HOME/XDG root identities outside any tree
walker. An outer-envelope failure is H4 and can never produce H1.

The operator then runs this exact bracket, using fresh descriptors and a fresh
HOME/XDG-shared budget for every complete pass:

1. streaming reference pass A;
2. current `auditWorkerCacheRoot` over HOME then XDG;
3. diagnostic-only candidate `unix.ReadDirent` pass;
4. a second candidate `unix.ReadDirent` pass;
5. a second current `auditWorkerCacheRoot` pass; and
6. streaming reference pass B.

It finally revalidates the outer envelope. H1 requires reference A and B to
have identical complete private metadata digests and verdicts, both current
passes to reject, both candidate passes to accept, and both outer-envelope
checks to accept. Candidate or current verdict disagreement is H4. Thus every
compared operation is localized to `auditWorkerCacheRoot` and bracketed by one
demonstrated stable metadata state.

All three walkers independently enforce the same v5 contract:

- unchanged ordinary root identities;
- no symlink traversal;
- only ordinary directories and regular single-link files;
- at most 4,096 nondirectory entries;
- at most 1 MiB of aggregate slash-normalized relative-path bytes;
- maximum depth 32; and
- at most 64 MiB of aggregate logical regular-file sizes.

The reference pass is independently implemented with path-based `Lstat`,
`unix.Open` with `O_NOFOLLOW`, and streaming `Readdirnames(128)`. It rechecks an
opened directory's identity before streaming it. It does not use
`filepath.WalkDir`, because that may read and sort an entire directory before a
callback can enforce bounds. It has a separate hard diagnostic ceiling of
8,192 entries, 2 MiB of path bytes, depth 64, and 128 MiB logical bytes; hitting
that outer ceiling is H4 rather than a partial digest.

The reference metadata-digest domain is exactly an ordered sequence of root
label; length-prefixed raw slash-normalized relative-path bytes; object-type
bits; device; inode; link count; and logical size for each fully traversed
entry, plus the two root identities. Invalid UTF-8 bytes are preserved rather
than JSON-normalized. It excludes permissions, owner, access/change/modify
times, and directory enumeration order. Records are canonicalized and sorted
only after the streaming traversal has enforced the hard memory/entry ceiling.
A stable reference rejection within that ceiling remains representable as H0:
the pass records accepted-contract violations but continues to its hard
diagnostic ceiling so its digest still covers the complete diagnostic tree.

The candidate pass independently uses `unix.ReadDirent`/`unix.ParseDirent`,
`Fstatat`, `Openat`, and `O_NOFOLLOW` with its own traversal and budget types.
The reference and candidate implementations may not call one another, the
current walker, or shared traversal, parsing, canonicalization, budget, or
verdict helpers. Source/AST tests enforce these prohibitions. Both share only
literal accepted contract constants and OS primitives. Candidate tests must
reject every v5 forbidden metadata case before a receipt can be created. Both
implementations are source-forbidden from filesystem mutation, including
write-capable opens, create, remove, rename, link, symlink, chmod, and chown.

Pre-receipt differential tests require current, reference, and candidate
acceptance for ordinary empty and nested trees; the exact 4,096-entry limit;
the exact 1 MiB path-byte limit; exact depth 32 including an empty deepest
directory; the exact 64 MiB logical-size limit; exact aggregate limits split
across HOME/XDG; and multiple enumeration/name-order permutations. One-unit
over-limit and every forbidden-type case must be rejected. Every pass uses a
fresh descriptor and budget.

The reference's streaming `Readdirnames` and the current `ReadDir` ultimately
share some Go `os.File` readdir machinery. This may conservatively cause H0 or
H4 rather than H1; the candidate's direct `unix.ReadDirent` parser is the
independent correction test.

Raw paths, names, counts, sizes, modes, link counts, device/inode values, and
private metadata digests never leave the helpers and are never written to the
receipt. Reference digest/verdict disagreement, current verdict disagreement,
candidate verdict disagreement, or an outer-envelope change is H4.

The reference audit is diagnostic evidence only. It is not callable from the
protected replay path and is not automatically eligible to replace the v5
descriptor audit.

## Frozen output schema and decision table

The sole human-visible and receipt result is one of:

- `contract-rejection` when both stable reference passes, both current passes,
  and both candidate passes reject;
- `descriptor-implementation-defect` when both stable reference passes and the
  two candidate passes accept while both current passes reject;
- `non-reproduction` when both stable reference passes, both current passes,
  and both candidate passes accept;
- `unsafe-auditor-disagreement` for every other stable verdict tuple; or
- `diagnostic-failure` for every precondition, worker, stability, or persistence
  failure.

The mapping is immutable. No predicate subtype, path, type, count, size, cap,
or digest is exposed. The operator prints only `diagnostic: <result>` and exits
zero only after a terminal receipt is durable. Any other error text is generic.

The exact one-line canonical JSON bindings are:

```json
{"result_version":"causal-cache-diagnostic-result/v1","results":["contract-rejection","descriptor-implementation-defect","non-reproduction","unsafe-auditor-disagreement","diagnostic-failure"]}
{"hypotheses_version":"causal-cache-diagnostic-hypotheses/v1","hypotheses":["diagnostic-tree-contract-rejection","reproducible-descriptor-implementation-defect","non-reproduction","unsafe-auditor-disagreement","diagnostic-failure"]}
{"decision_version":"causal-cache-diagnostic-decision/v1","h0":"terminate-invalid","h1":"authorize-read-dirent-recovery","h2":"terminate-invalid","h3":"terminate-invalid","h4":"terminate-invalid"}
```

Their respective SHA-256 digests are
`af39dcfb8bc6b40287b2e1702bf15a5f157720ef714defac76f2a02564b1e611`,
`9a69dc0cfb4665bc05a91a400d5469d00253abdc6693e0cc45ca98440ada07df`,
and `1fcb46171d7037ace6549bdcb0858dbeae433634514a2114d756c0124d4f7912`.

## Exclusive diagnostic receipt

The receipt path is
`.git/nous-attempts/active-causal-diagnosis-v6-cache-diagnostic.json`. Its
version is `causal-cache-diagnostic/v1`. Before any worker start, the operator
creates it exclusively, syncs the file and parent directory, and records:

- state `started` and creation time;
- accepted diagnostic-plan commit and X6 implementation commit;
- governing program, P5, X5, E3, and R3 identities;
- exact failed-v5 receipt digest;
- diagnostic input digest, byte digest, and length;
- diagnostic operator and E3 worker executable digests;
- normalized candidate-walker function-body AST digest;
- normalized environment, result-schema, hypotheses, and decision-table
  digests; and
- zero worker starts.

Collision, malformed predecessor, dirty topology, or identity mismatch before
exclusive creation is a zero-start non-attempt. After receipt creation, the
counter increments from zero to one exactly after successful `Command.Start`.
It immediately persists and syncs the `started` receipt with count one before
writing worker input. The operator atomically persists that exact final count
and state `completed` plus the normalized result, or state `failed` plus
`diagnostic-failure`, before returning. No deletion or retry path exists. A
test-kill hook proves a `started` receipt also consumes the diagnostic. A crash
or sync failure in the unavoidable interval after OS start but before the
count-one sync may leave count zero, but exclusivity still consumes it as H4.

If terminal replacement or directory sync fails, the durable `started` receipt
is the recoverable consumed residue. The operator emits only generic
`diagnostic-failure`; the residue is classified H4 and never authorizes repair,
receipt deletion, or retry.

## Implementation topology and restrictions

This plan is committed alone as P6, directly after governing commit
`e6dd2fef97a80e24ceae5b2bb531d09ae310c0aa`. X6 is P6's direct child and adds
exactly:

- `internal/causalexpv2/cache_diagnostic.go`;
- `internal/causalexpv2/cache_diagnostic_test.go`; and
- `internal/causalexpv2/cachediagexec/main.go`.

X6 may call existing build, worktree, environment, cleanup, and v5 audit
helpers. It may not modify them. The new source may not call replay capability
minting, `ExecuteReplay`, validation/locked gates, evidence publication,
`gitFile` for R3 evidence paths, or any result comparison. AST and source scans
enforce those prohibitions and the three-file topology.

Tests must prove input identity, predecessor binding, zero-start failures,
exclusive receipt behavior, durable terminal persistence, environment and
executable binding, generic output, reference stability, all four comparison
classes plus diagnostic failure, candidate/current repeatability,
outer-envelope localization, every
v5 forbidden metadata case for both independent walkers, cleanup confinement,
worker-start count, build self-binding, source/AST independence, and absence of
empirical attempt records.

The bound candidate digest covers exactly the normalized AST of the function
body, excluding the diagnostic-only function name, signature, and budget type.
The body is self-contained: it may use its parameters, local declarations,
standard-library/unix calls, and literal contract constants, but no candidate-
specific helper. Before receipt creation, a source-generation test transplants
that exact body into an otherwise fixed recovery scaffold with
`auditWorkerCacheRoot`'s signature and `workerCacheBudget`, then parses and type-
checks it without adapters or extra declarations. The generated body's digest
must equal the receipt-bound digest.

## Predetermined H1 correction

If and only if the consumed diagnostic returns
`descriptor-implementation-defect`, one recovery amendment may replace the
inside of `auditWorkerCacheRoot` with the exact accepted diagnostic candidate
`unix.ReadDirent` walker body, whose normalized AST digest is bound into the
diagnostic receipt. The replacement must preserve every v5 root-identity,
`Fstatat`, `Openat`, `O_NOFOLLOW`, type, hardlink, depth, path-byte, entry, and
logical-byte predicate exactly; it adds no accepted type, path exception,
retry, or larger cap. Names are parsed in bounded chunks and every entry is
revalidated through its parent directory descriptor.

That recovery remains a separately committed, three-reviewer plan and
implementation and receives exactly one new protected replay receipt bound to
this diagnostic receipt and failed v5. If the new walker does not pass
non-attempt tests, no replay occurs. If its one replay fails, Gate 0 terminates
`invalid` with no later version.

H1 proves only that the current walker is defective on the fixed diagnostic
tree and that the candidate preserves the frozen contract on that tree, the
exact accepted-boundary differential suite, and the complete forbidden-case
suite. It does not retroactively classify v5 cache metadata. The one protected
recovery remains capable of rejecting genuinely forbidden R3 cache metadata;
such a rejection is a final invalid result, not a reason to weaken the
contract.

## Acceptance gate before execution

The diagnostic plan and implementation must each receive explicit acceptance
from the architecture/integration, domain-theory, and experimental-validity
reviewers. All package tests, scoped race tests, vet, source/topology tests, and
`git diff --check` must pass on clean X6. Only then may the separately built
diagnostic operator be invoked once. Its receipt and exact output are committed
to the result record before any recovery design begins.
