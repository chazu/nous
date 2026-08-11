# Guarded action-relations vocabulary: implementation and trial record

Status: **development mechanically invalid; validation and locked panels were not run**

This document records Vocabulary 3 of the
[Part 3 vocabulary research program](vocabulary-research-program-v3.md). The
semantic implementation and its evidence verifiers passed review and ordinary
tests, but the one public development attempt failed after its persistent start
transition. It therefore supplies no empirical guarded-commutativity claim.

## Frozen authority

- accepted plan: `95860673664799ea1e18c7cb1f7e433238830216`;
- reviewed implementation: `26c57cd57e6da886a597699ff5daecc3aa55a06d`;
- reviewed implementation archive SHA-256:
  `bfb28a82e4e98189bb364e57ca0a1a48adafe0cb3fc35670c2f066683efdca06`;
- unanimous implementation-review authority commit:
  `0e465b473510bc2b11000520e86573873149d2d8`;
- build and competence authority commit:
  `d7274357a3d6befc0bd7f8397aa8ecf5cb718cc2`;
- source root:
  `a5de9b5743665e27d6c0a4453fd092018e6896672d8633926781acb04c5b5014`;
- reviewed binary SHA-256:
  `67ae43dae5527156f7d3f44a9a1a2ce4f742305210a036b02c15baebf5b9d59c`;
  and
- competence: 55,805 cases, terminal `passed`.

Architecture, action-semantics, and experimental-validity reviewers accepted
the same implementation commit and archive. The canonical verdicts and
attestations are retained in
`docs/actionrelations-implementation-reviews.json` and
`docs/actionrelations-reviews/implementation/round-1/`.

## Development outcome

The development attempt created the persistent marker for identity
`a5de9b5743665e27d6c0a4453fd092018e6896672d8633926781acb04c5b5014`,
then failed when the primary isolated worker returned:

```text
error: lstat /Users: operation not permitted
```

The canonical terminal receipt is
`.nous/actionrelations-v1-development-terminal-receipt.json`, SHA-256
`08ac1a0fa5573e4cc1cc7f27701f641436e1f6e50c94cbab921b56b8875b862c`.
It classifies the attempt `invalid` and binds the same printable-ASCII reason
to the three required invalid-authority documents:

| Authority | SHA-256 |
| --- | --- |
| development report | `1677ffac8ebfb6a8d003ec32ab78428b3fe5459fddba16b8a8e63ffd4882c07e` |
| fixture root | `74d7c526ee8c76da792b7927d78e04680f801d7fbdf03fcd59e909ef9fcc370c` |
| evidence payload | `82c347f1a8556a275abcc74b39304df1bb78c112c28517e4d87a014dc5c4a324` |

No publication exists. Repeating the exact execute invocation entered recovery,
returned the same terminal-invalid reason, and left all four hashes unchanged.
The start marker remains mode `0600`; it is not repository authority and is not
deleted or replaced.

## Root cause and protocol consequence

The supervisor passes canonical absolute paths to the isolated worker. The
worker calls `filepath.EvalSymlinks` on those paths, which performs metadata
lookups on their ancestors. The macOS sandbox profile grants input metadata
below the temporary input root but not on the already disclosed literal parent
directories. Canonicalization therefore fails at `/Users` before policy work
can begin.

This was a post-start infrastructure failure. Under the frozen append-only
protocol it cannot be retried, overwritten, or converted into a successful V1
attempt. Validation and locked execution are not authorized. A future attempt
would require a separately planned and reviewed identity or protocol revision;
it must not delete this marker or reclassify this result. Such a revision should
either grant metadata-only access to each literal path ancestor or eliminate
worker-side ancestor traversal while retaining no-follow canonical-path checks.

For the Part 3 capability matrix, **Compress order** is `invalid`. Guarded
relation acquisition and replay competence passed, but no development evidence
exists from which to classify preservation or marginal search utility.

## Exact verification and execution commands

The reviewed implementation passed:

```sh
mise exec -- go test ./internal/actionrelation... -count=1
mise exec -- go vet ./internal/actionrelation...
git diff --check
```

Prerequisites were constructed with:

```sh
mise exec -- go run ./cmd/nous actionrelation-trials \
  -stage prepare -panel development \
  -repo-root /Users/chazu/dev/go/nous
```

The development attempt and its idempotent recovery check both used:

```sh
env -i GOMAXPROCS=1 LC_ALL=C \
  PATH=/opt/homebrew/bin:/usr/bin:/bin TZ=UTC \
  /Users/chazu/dev/go/nous/.nous/bin/actionrelation-nous-v1 \
  actionrelation-trials -stage execute -panel development \
  -repo-root /Users/chazu/dev/go/nous
```

These commands document the recorded attempt. The idempotent recovery check was
performed at prerequisite commit `d7274357a3d6befc0bd7f8397aa8ecf5cb718cc2`
before this result record changed the umbrella document and therefore the
working source checkout. The persistent marker and committed invalid receipt
still forbid a second V1 attempt; the command is not an instruction to rewrite
the current checkout or recreate the recorded namespace.
