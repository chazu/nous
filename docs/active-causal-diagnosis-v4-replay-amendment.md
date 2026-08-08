# Active causal diagnosis v4 replay-retry amendment

Status: accepted amendment, revision 6

## Scope and observed failure

The v3 training attempt succeeded from pretraining executable E3
`89a9221e97dd17d6ba220a22a12d4c0328417ffb`. It atomically published a
mechanically valid report and bundle, and the evidence-only direct child R3 is
`7482d1b9712f49cb988623a87ec8bb1c34667a26`. The report selected
`P=E;M=gain;S=C`, has training digest
`96b1cdf7579c0a186e5cd9aeb7aaa42f0c224ffe19989bf78b5b3aa320b17fa0`,
and records 2,724 oracle agreements, zero disagreements, all controls passing,
and all caps valid. Those published results are fixed evidence and may not be
rerun or tuned.

The first required dirty-R3 detached-E3 replay failed before the replay worker
could be built. The preserved v3 replay receipt is `failed`. The only error was
that E3's standalone `go.mod`/`go.sum` did not contain the checksum for its
direct `golang.org/x/sys/unix` import. Ordinary development builds had silently
obtained that module through the ignored local `go.work`, whose sibling PUDL
module requires `golang.org/x/sys`; the detached E3 worktree correctly had no
such ambient workspace authority.

No detached replay worker ran, no protected generator regenerated a fixture,
no replay output was created, and no validation or locked fixture was opened.
The v3 implementation minted its capability before the failed build, so it may
have contextually re-executed the already-public R3 bundle fixtures; this is not
claimed absent. The training evidence was already public before the failure,
so the generic module-build diagnostic exposed no new empirical value. The v3
replay receipt is consumed and must never be deleted, rewritten, retried, or
accepted as authority.

V4 is only a technical retry of the failed replay/build step. It does not
rerun training, change R3, change any empirical code or identity, or alter the
training, validation, or locked seed schedules.

## Root cause and fixed detached build

E3 imports `golang.org/x/sys/unix`, while its committed module graph does not
name `golang.org/x/sys`. A clean-room rehearsal from E3 with the pinned mise Go
toolchain and `GOWORK=off go build -mod=mod` succeeds and makes exactly two
temporary metadata edits:

- `go.mod` adds direct `golang.org/x/sys v0.41.0`;
- `go.sum` adds the v0.41.0 module and go.mod checksums.

The complete expected post-resolution file SHA-256 values are:

- `go.mod`: `e5875629b398cfccd32df7604196702818eb2fc5e9b605897a0207565c64866c`;
- `go.sum`: `930d1ecfb0438e23115d1365f24fbddcd27fa0a97d144436eb7bafc208bbb6d4`.

X4 commits exactly those two module metadata changes so subsequent clean
protected commands do not depend on the ignored workspace. Detached E3 still
starts from the original module files, resolves them temporarily with
`build -mod=mod`, and must end with bytes identical to X4's committed `go.mod`
and `go.sum`. No source, generated file, staged change, untracked path, or
other metadata change is allowed. The same audit repeats after the worker
exits. The worker still receives replay input only on inherited FD 3 and
writes only through inherited directory FD 4.

The accepted toolchain is the regular resolved Go executable
`/Users/chazu/.local/share/mise/installs/go/1.25.12/bin/go`, SHA-256
`8612de418d551a418517845c05cebdcfed49095cd08ef0a4d682bb2a5cf4896c`,
reporting exactly `go version go1.25.12 darwin/arm64`. Before preflight,
actual replay, synchronous C4 replay, validation, or locked execution, code
must resolve the mise Go path, follow symlinks, verify that digest and version,
and reject any other toolchain. Its exact GOROOT is
`/Users/chazu/.local/share/mise/installs/go/1.25.12`. A canonical manifest over
all 14,531 regular GOROOT files has SHA-256
`77a814b12481fa12b070a905d2d6fc1ab9671b0e2866d7ffb85c9b37861d9da9`.
The manifest visits slash-normalized relative paths in bytewise sorted order
and hashes, for each file, `path NUL octal-permission-bits NUL decimal-size
NUL contents NUL`. Traversal directories are permitted but not hashed;
symlinks and special non-directory entries, a different regular-file count,
different GOROOT, or any digest mismatch fail. This binds the driver, compiler,
assembler, linker, build tools, and standard-library source as one accepted
distribution tree.

Build subprocesses use the resolved Go binary directly from an explicit
environment allowlist—not inherited process environment—with the exact GOROOT,
`GOTOOLCHAIN=local`, `GOENV=off`, `GOWORK=off`, empty `GOFLAGS`, empty
`GOEXPERIMENT`, `CGO_ENABLED=0`, `GOOS=darwin`, `GOARCH=arm64`,
`GOARM64=v8.0`, empty `GODEBUG`, and `GOFIPS140=off`.
Each build receives fresh private `GOMODCACHE` and `GOCACHE` directories plus
fixed module-resolution policy: `GOPROXY=https://proxy.golang.org`,
`GOSUMDB=sum.golang.org`, and empty `GOPRIVATE`, `GONOPROXY`, and `GONOSUMDB`.
They also receive a fresh private `TMPDIR`; no `HOME`, user GOENV, PATH-based
tool lookup, or other ambient variable is passed. The actual replay worker
receives no arguments and no ambient environment variables, but frozen E3 code
invokes `git` by name. V4 therefore resolves `/opt/homebrew/bin/git`, requires
regular-file SHA-256
`00ad7d9b0732c80bd8971e443a7129cf09d0957ea4c1f6cf581bbffe6c2e0505`
and exact output `git version 2.47.1`, then creates a fresh private bin directory
containing only a verified `git` symlink. The worker environment is exactly
`PATH=<private-bin>`, `HOME=<fresh-empty-home>`,
`XDG_CONFIG_HOME=<fresh-empty-xdg>`, `GIT_CONFIG_NOSYSTEM=1`,
`GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_SYSTEM=/dev/null`,
`GIT_OPTIONAL_LOCKS=0`, `GIT_NO_REPLACE_OBJECTS=1`,
`GIT_ATTR_NOSYSTEM=1`, `LC_ALL=C`, and `TZ=UTC`. Variables including
`GIT_CONFIG_COUNT`, `GIT_CONFIG_PARAMETERS`, `GIT_EXTERNAL_DIFF`, and
`GIT_DIFF_OPTS` are absent because the environment is built from that exact
allowlist.

The repository-local Git state is also an input and is pinned before build
preflight, before receipt creation, before each worker start, and after each
worker exit. The common `.git/config` must have exact SHA-256
`58610c019ec3c32186d8dc2a9e5a18b900dafdb75047d7a56475baa84d268a15`,
contain no `include`, `includeIf`, `extensions.worktreeConfig`, external diff,
filter, fsmonitor, hooks, or status override, and remain unchanged. Common
`.git/info/exclude` must have exact SHA-256
`6671fe83b7a07c8932ee89164d1f2793b2318058eb8b98dc5c06ee0a5a3b0ec1`.
`.git/info/attributes`, grafts, shallow state, object alternates, worktree
config, and replacement refs must be absent. These floors prevent local config
from hiding dirt, changing object identity/diffs, or launching helpers. FD 3/4
remain the sole empirical payload/output channels.
The protected executable also checks its own runtime version, OS/architecture,
and build settings before validation or locked work. Operator verification and
protected commands use that same environment and `-mod=readonly` against X4's
committed module metadata. Hostile inherited values for every pinned variable
must be removed and replaced, not rejected or appended ambiguously.

Before a v4 receipt is created, a non-empirical build-only preflight creates a
fresh detached E3 worktree, builds but does not run the worker, checks the exact
two-file mutation and hashes, and removes the temporary worktree. It may not
mint replay capability, contextually verify training evidence, construct the
R3 fixture-fed replay input, invoke worker code, or open any fixture. Failure
at this preflight creates no v4 receipt and opens no fixture. Its API accepts
only repository root, E3 commit, and verified builder; it has no evidence
commit, report/bundle path, capability, fixture, replay input, or
worker-execution parameter. Tests make R3 evidence unavailable and observe
explicit evidence-read, capability-mint, replay-input-construction, and
worker-start counters, all of which must remain zero.

After build-only preflight, the implementation may structurally read and
strictly decode committed report/bundle provenance to populate the receipt,
but it must create and fsync the v4 `started` receipt before capability minting,
contextual fixture execution, replay-input construction, the second detached
build, or worker execution. All such work then occurs under that one-shot
receipt. Any error transitions it to `failed`.

Replay input and regenerated report/bundle identities remain their accepted
v3 and semantic-v2 values, including `causal-replay-input/v3` and experiment
plan commit `b4183595a9769882ad4673c606e9a35560cfb95c`. The detached worker is E3
code and must therefore consume exactly those bytes. Only the outer one-shot
receipt moves to v4:

- receipt version: `causal-replay-success/v4`;
- receipt filename: `active-causal-diagnosis-v4-replay.json`;
- candidate-diff digest domain: `causal-replay-candidate-diff/v4`;
- receipt plan commit: the accepted commit of this amendment.

The malformed or failed v3 receipt is opaque to v4 and remains byte-identical.
V4 requires its own receipt path to be absent before the build-only preflight.
It also requires the still-unused v3 validation and locked attempt, proof, and
result slots to be absent before build preflight. Any collision fails without
building, minting, reading evidence, creating a receipt, or opening a fixture.

## Chronology and candidate binding

The v4 replay-repair executable X4 is a descendant of R3. At clean X4 its
complete name-status diff from R3 must be exactly:

- `A docs/active-causal-diagnosis-v4-replay-amendment.md`;
- `M internal/causalexpv2/gates.go`;
- `M internal/causalexpv2/provenance.go`;
- `M internal/causalexpv2/provenance_test.go`;
- `M internal/causalexpv2/replay_gate.go`;
- `M go.mod`; and
- `M go.sum`.

All seven paths must be regular `100644` blobs, the amendment blob at X4 must
equal its accepted plan-commit blob, and every other path must be byte-identical
to R3. `go.mod` and `go.sum` must equal the exact resolved hashes above. In
particular `freeze.go`, every generator/executor/control/curriculum/statistics/
meter/evidence/report/verification file not named above, all causal vocab/
runtime files, and `internal/causalv2/control.go` remain unchanged. `freeze.go`
is the exact empty three-assignment file at X4.

Because R3 results are public before X4, path confinement is supplemented by
R3-anchored source floors. `beginLockedAttempt`, `mechanicallyValid`, panel
seed-range construction, report/evidence
verification, replay-input fixture construction and v3 identities, and all
unlisted existing functions must have byte-identical AST bodies. Allowed
existing-function changes are limited to:

- `mintReplayCapability`: replace ambient mise invocation with the verified
  pinned Go executable; its evidence reads, fixture selection, replay-input
  fields, versions, and verification remain unchanged;
- `verifyCandidateConstantsState`: recognize exact X4 and its constants-only
  child C4 while retaining the one-file freeze rule;
- `beginValidationAttempt`: replace only its obsolete direct R3-to-HEAD
  `freeze.go` diff check with delegation to `verifyCandidateConstantsState`
  before any evidence read; its seeds, evidence checks, replay sequence, and
  all remaining statements stay AST-identical;
- `Replay` and `buildReplayWorker`: use the pinned environment, `-mod=mod`,
  and exact temporary metadata audits;
- `ExecuteReplay`, receipt creation/verification, receipt path, and candidate
  digest: move only the outer receipt to v4, move receipt creation before
  mint/context/replay, and bind R3/X4/C4; and
- `ExecuteProtectedPanel`: add only the pinned toolchain/runtime precondition.

New helpers may implement only those enumerated duties. Tests compare protected
AST bodies to R3 and reject changes outside this function list. The X4 diff may
introduce no new branch or lookup on selected rule, actions, outcomes, scores,
aggregates, contrasts, or any other R3 empirical field. Existing frozen-rule
and digest equality checks remain unchanged.

The prescribed sequence is:

1. accept and commit this amendment;
2. implement and adversarially review only the bounded replay repair;
3. commit clean X4 and run all nonprotected checks;
4. insert only the three R3-derived frozen values into `freeze.go` while HEAD
   remains X4;
5. require the worktree diff from X4 to be exactly that one unstaged file;
6. run the build-only detached-E3 preflight before creating any receipt and
   without minting capability or touching fixture-fed paths;
7. structurally bind R3 provenance, create/fsync the v4 `started` receipt, then
   mint capability and perform one detached-E3 replay;
8. on exact report and bundle byte equality, mark the v4 receipt succeeded;
9. commit the constants-only direct child C4 of X4;
10. from clean C4, repeat detached byte replay synchronously before opening
    validation, then require the prior successful v4 receipt as corroborating
    provenance;
11. run validation once, and only a mechanically valid validation may
    authorize the single locked run.

The candidate-diff digest binds X4 plus the complete R3-to-dirty-candidate tree
diff; it cannot bind the not-yet-created C4 commit hash. The receipt's candidate
commit is X4. At clean C4 the gate independently requires its parent to be X4,
the X4-to-C4 diff to be exactly `freeze.go`, and the committed R3-to-C4 diff
digest to equal the receipt's dirty-tree digest. It no longer incorrectly
assumes that the constants commit is a direct child of R3. Evidence commit R3
must remain the exact two-file direct child of E3. The experiment `PlanCommit`
remains the v3 amendment commit, while a separate `ReplayRepairPlanCommit`
identifies this accepted v4 amendment; neither substitutes for the other.

Any v4 receipt creation or actual replay failure consumes v4 and terminates
this retry. A build-only preflight failure consumes nothing because it occurs
before receipt creation and before fixture replay. Validation and locked
attempt/result namespaces remain v3 because those one-shot slots were never
created or consumed; this amendment supplies the missing replay authority but
does not reinterpret their experiment identity.

## Required tests and review

Nonprotected tests must prove:

- the original detached E3 build fails under `GOWORK=off` without
  `-mod=mod`, with no worker produced;
- build-only preflight runs before v4 receipt creation and opens zero protected
  fixtures, never mints capability, never constructs replay input, and never
  invokes worker code;
- `-mod=mod` creates exactly the two expected metadata changes and hashes;
- any source, extra, staged, untracked, wrong-hash, ambient-workspace, or
  ambient `GOFLAGS`, `GOENV`, `GOTOOLCHAIN`, `GOEXPERIMENT`, `CGO_ENABLED`,
  `GOOS`, `GOARCH`, `GOARM64`, `GOROOT`, `GODEBUG`, or `GOFIPS140` condition
  fails closed or is unambiguously replaced;
- wrong Go path, driver digest, GOROOT path/tree manifest/file count,
  compiler/tool/stdlib byte, version, runtime version, architecture, GOARM64,
  GODEBUG, GOFIPS140, or protected-executable build setting fails before
  protected work;
- wrong Git path, digest, version, private-bin contents/symlink, config
  isolation, or worker environment fails before worker start;
- hostile local/worktree Git config, include, fsmonitor, diff/filter driver,
  status override, attributes, exclude change, graft, shallow state, object
  alternate, replacement ref, or config/diff environment fails before build
  preflight and before worker start;
- each build uses newly absent private module/build caches and the fixed
  proxy/sumdb policy, and hostile ambient cache/module policy is overwritten;
- the v3 failed receipt is ignored and byte-preserved, while any preexisting
  v4 receipt blocks the retry;
- any preexisting v3 validation/locked attempt, proof, or result blocks before
  build preflight and leaves every preflight/activity counter at zero;
- v3 and v4 receipt versions, filenames, plan commits, and candidate-diff
  domains reject each other;
- X4 confinement rejects missing, additional, renamed, mode-changed, or
  amendment-tampered paths and changes in every protected group;
- R3-anchored AST/source floors reject a change to every protected function,
  any non-enumerated change in the four protocol files, and any new branch or
  lookup on a published empirical field;
- dirty X4 plus the exact frozen constants is accepted, any other dirty state
  is rejected, and clean C4 must be the constants-only direct child of X4;
- synchronous replay and validation require the v4 receipt bound to R3, X4,
  the training digest, bundle digest, and exact dirty-tree diff, while the
  clean gate independently proves C4 is X4's constants-only child with that
  same committed tree diff;
- pre-receipt synthetic/development fixtures test only build, descriptor, and
  output transport and cannot assert equality to R3; the first actual R3 byte
  equality occurs under the v4 receipt, followed by clean-C4 synchronous
  equality;
- a non-empirical helper exercises E3's exact `git` lookup/state operations
  under the fixed worker environment; the first full real E3 worker launch is
  the one under the v4 receipt, which additionally proves that environment by
  succeeding, and clean-C4 repeats it; and
- normal, race, vet, full-suite, diff, and protected-byte-floor checks pass.

Architecture, theory, and experimental reviewers must unanimously accept the
plan and then the implementation. No v4 receipt or protected panel may be
opened before clean X4 and the post-commit confinement tests pass.
