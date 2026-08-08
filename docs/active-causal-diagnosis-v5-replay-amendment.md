# Active causal diagnosis v5 replay amendment

Status: proposed technical retry after the consumed v4 replay.

## Fixed facts

Training and its public result do not change. E3 is
`89a9221e97dd17d6ba220a22a12d4c0328417ffb`; R3 is
`7482d1b9712f49cb988623a87ec8bb1c34667a26`; the selected rule is
`P=E;M=gain;S=C`; the training digest is
`96b1cdf7579c0a186e5cd9aeb7aaa42f0c224ffe19989bf78b5b3aa320b17fa0`;
and the bundle digest is
`117a0322464cdf26022b7c21b2d5401c67cbad974640f042e2591c920d982503`.
No training, fixture, scoring, selection, or evaluation behavior may change.

The v4 replay receipt is failed and immutable. Its SHA-256 at acceptance is
`7ade7b64b49ec797865766e5a0f99ff329a6619ef785f163ed553844277c0fcf`.
The sole returned v4 error was the parent's requirement that the fresh private
worker HOME remain empty after execution. Pipe writing and `command.Wait`
succeeded, but output equality remains unknown because comparison was not
reached. The v4 contract
required HOME and XDG_CONFIG_HOME to be fresh and empty before launch, but did
not require runtime libraries to leave those disposable directories empty.
The output comparison was not reached, so v5 remains a replay-only retry.
The complete human-visible v4 observation was
`error: worker private HOME is not empty` followed by outer process exit status
1. The absence of the `regeneration executable failed:` prefix establishes
that pipe writing and `command.Wait` succeeded. No replay output bytes, output
names, cache names, cache sizes, cache contents, or cache digests were exposed.

Before build preflight or any activity counter can advance, v5 requires the v4
receipt to exist with exactly that SHA-256 and to strictly decode as the fixed
failed v4 receipt bound to E3, R3, X4, the fixed training/bundle digests, and
the recorded candidate-diff digest. A missing or mismatched predecessor fails
with every preflight, evidence-read, capability-mint, replay-input, and worker
counter at zero. The v5 receipt carries the predecessor receipt SHA-256, and
clean-C5 replay, validation, and locked gates recheck both it and the immutable
v4 file.

## Bounded repair

V5 preserves every v4 toolchain, module, Git, source-floor, build-worktree,
clean-execution-worktree, descriptor, byte-comparison, and cleanup rule. The
only worker-environment change is phase-aware verification:

- before worker start, the exact eleven-variable environment, private Git
  symlink, and empty private HOME and XDG_CONFIG_HOME are required;
- after worker exit, the exact environment values and private Git symlink are
  reverified, while HOME and XDG_CONFIG_HOME may contain runtime-created cache
  state;
- post-exit content is never read as evidence, never copied or published, and
  is removed with the owned replay temporary root;
- symlinks, hardlinks (regular-file link count other than one), and special
  files anywhere below HOME or XDG_CONFIG_HOME are rejected. Traversal may not
  escape those roots and is capped at 4,096 nondirectory entries, 1 MiB of
  aggregate slash-normalized relative-path bytes, depth 32, and 64 MiB of
  aggregate logical regular-file sizes. This audit runs after successful and
  failed worker exits.

Before launch, the parent records the device and inode of each HOME/XDG root.
Each root must begin as an ordinary nonsymlink directory beneath the owned
replay temporary root and must remain that same device/inode and object type
after exit. Moving/replacing either root, including replacement by a symlink or
another directory, is rejected.

Cleanup permission normalization may chmod only directories inside the owned
temporary root. It must never chmod a regular file, because an untrusted
hardlink could otherwise change the mode of an external inode. Directory write
permission is sufficient to unlink read-only module-cache files.

The previously added private GOPATH remains part of the fixed Go build
environment. The protected operator continues to build a standalone pinned
binary and then execute it with exact PATH; `go run` is not an accepted
operator because it rewrites the child PATH. Both pre-receipt operator failures
are non-attempt diagnostics and created no receipt.

## Versioning and topology

X4 is fixed as `3f400d22a02812da41ed1ffb158db434cfb41ef9`.
This document is committed as P5 while `freeze.go` is empty, with exact X4 as
its parent and this document as its only tree change. X5 is P5's direct
child and changes only:

- `internal/causalexpv2/gates.go`;
- `internal/causalexpv2/provenance_test.go`; and
- `internal/causalexpv2/replay_gate.go`.

The complete R3-to-X5 name-status allowlist is exactly the v4 seven paths plus
`A docs/active-causal-diagnosis-v5-replay-amendment.md`; every path is a
regular 100644 blob and the P5/X5 amendment blob equals the accepted P5 blob.
Relative to P5, X5 modifies exactly the three implementation paths above. All
other X4 implementation bytes remain unchanged. X5 updates candidate
confinement to this topology and keeps `freeze.go` empty. C5 is X5's direct
regular-100644 constants-only child.

X4-anchored AST floors supplement path confinement. Existing signatures,
top-level declarations, and function bodies remain identical except for:

- splitting or phase-parameterizing `verifyPinnedWorkerEnvironment`;
- adding a private root-identity snapshot type/helper for the device/inode and
  bounded post-exit cache audit;
- changing `Replay` so any nonnil pinned worker environment takes the same
  launch/post-exit environment-audit path, while repository and resolved-build
  worktree audits remain conditional on an actually built protected worker;
- changing `makeTreeOwnerWritable` only to stop chmodding nondirectories;
- adding the predecessor field to `replaySuccessRecord`, the fixed predecessor
  digest/receipt constants, and the v5 receipt constants;
- updating `ExecuteReplay`, `requireReplayRetrySlotsAbsent`, structural receipt
  construction, receipt path/digest/create/verification, and X5/C5 topology
  functions only for predecessor sequencing and v5 identities.

New cache-audit helpers may implement only the bounded metadata checks above,
must not select an empirical field, and must not call semantic evidence
verification or regeneration. Tests reject any other X4-relative source delta.

The outer receipt alone moves to:

- version `causal-replay-success/v5`;
- filename `active-causal-diagnosis-v5-replay.json`;
- candidate-diff domain `causal-replay-candidate-diff/v5`;
- plan commit P5;
- predecessor receipt digest
  `7ade7b64b49ec797865766e5a0f99ff329a6619ef785f163ed553844277c0fcf`.

The v3 failed receipt is opaque and byte-preserved. The v4 failed receipt is
read only as the exact pinned predecessor and remains byte-preserved. Validation and
locked names and all replay-input/report/bundle identities remain v3/semantic
v2. V5 requires its own receipt and every still-unused validation/locked slot
to be absent before preflight. Receipt creation remains after structural R3
binding and before capability mint, contextual verification, replay input,
the actual worker build, or worker execution. Any failure after creation
consumes v5.

## Required gates

Before the v5 receipt can be created:

1. all normal, race, vet, diff, source-floor, and X5 confinement tests pass;
2. tests prove launch-time empty HOME/XDG rejection, bounded post-exit regular
   cache acceptance, and post-exit rejection of symlinks, hardlinks, special
   files, root move/replacement/device/inode change, depth/path/entry
   exhaustion, or over-cap content;
   these are synthetic-helper tests only and may not read R3 evidence, mint a
   replay capability, construct replay input, run E3 with an R3 fixture, inspect
   a prior replay output, or assert R3 byte equality; failures expose no cache
   path, filename, count, size, digest, or content;
3. a failing synthetic worker proves the post-exit audit still runs;
   an end-to-end protected `Replay` with a synthetic worker writes permissible
   HOME/XDG cache plus the exact two synthetic outputs, exits zero, passes the
   post-exit phase audit, and reaches byte comparison;
   both synthetic workers use the same environment-audited `Replay` path, not
   a direct cache-audit-helper call, while omitting protected build-worktree
   authority and all R3 inputs;
4. v3 and v4 receipt bytes are unchanged, the exact failed-v4 prerequisite is
   enforced at zero activity, and v5 identities reject older ones;
5. all three adversarial reviewers accept this plan and implementation;
6. X5 is committed clean, then `freeze.go` alone receives the three fixed R3
   constants;
7. the v5 replay is invoked once with a separately built pinned executable.

After replay success, C5 commits only `freeze.go`; synchronous replay,
validation, and locked evaluation proceed under the existing one-shot rules.
