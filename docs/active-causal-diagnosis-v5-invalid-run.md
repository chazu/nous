# Active causal diagnosis v5 invalid replay

Status: invalid. The protected v5 replay was consumed on 2026-08-08 and did
not reach evidence comparison, validation, or locked evaluation.

## Fixed provenance

- accepted replay plan P5: `630a0ce8c36118957d008fbf7132af2424ee91f0`;
- implementation candidate X5: `3bce9836b60c9fc419696f0a82450af3a158ec19`;
- pretraining commit E3: `89a9221e97dd17d6ba220a22a12d4c0328417ffb`;
- evidence commit R3: `7482d1b9712f49cb988623a87ec8bb1c34667a26`;
- failed-v4 predecessor SHA-256:
  `7ade7b64b49ec797865766e5a0f99ff329a6619ef785f163ed553844277c0fcf`;
- failed-v5 receipt SHA-256:
  `2e3c67fe774b288d72ea8672f9b07a063533a377a1394ba5de082f9c316a6577`.

The v5 receipt is stored outside the worktree at
`.git/nous-attempts/active-causal-diagnosis-v5-replay.json`. It is canonical,
bound to P5, E3, R3, X5, the fixed training and bundle digests, and the exact
failed-v4 predecessor digest, and has state `failed`.

## Observed failure

The separately built pinned operator was invoked once under the accepted exact
environment. Its complete human-visible failure was:

```text
error: worker private cache audit failed
```

The error lacks the `regeneration executable failed:` prefix, so pipe writing
and worker waiting succeeded. The parent rejected post-exit private cache
metadata before reading or comparing replay outputs. Output equality is
therefore unknown. No cache path, filename, count, size, digest, content, or
replay output was exposed.

## Consequence

V5 is mechanically invalid and cannot support an empirical conclusion. The
frozen constants were restored to their blank X5 state. Validation and locked
attempt slots remain unused.

The v5 replay must not be retried. Any further technical retry requires a new
accepted amendment, a new versioned receipt bound to the immutable v5 receipt,
and a new direct implementation candidate. Until then, Phase 2 remains invalid
and blocks advancement under the accepted vocabulary research roadmap.
