# Active causal diagnosis v1 invalid training run

## Decision

The one-time v1 training run from pretraining commit
`ed5004cec4f20177b2d42e100716b46958938b9f` is `invalid`. Its internally
reported `status: valid` is superseded by this independent post-run audit. The
selected code `P=E;M=raw;S=C`, its training digest, and every apparent gate in
that report are prohibited from becoming frozen constants or evidence for an
empirical claim.

The original outputs are retained unchanged as diagnostic evidence:

- `docs/evidence/invalid/active-causal-diagnosis-training-v1-invalid.json`; and
- `docs/evidence/invalid/active-causal-diagnosis-training-episodes-v1-invalid.json`.

## What the run established

The bounded causal semantics were operational: the implementation enumerated
72 legal SCMs and 58 interventional-equivalence classes, generated the intended
12 fixtures, ran a seed-major 40-by-12 matrix, completed 480 CUE-mediated
episodes, retained the teacher, and observed zero partition/posterior
disagreements in the checks it actually performed. Successful episode scores
equal intervention cost. The run produced an 18-rule credit tie and selected
the semantic first member.

Those facts are implementation diagnostics only. They do not establish a valid
training result because the audit boundary itself was incomplete.

## Mechanical invalidators

Adversarial implementation review by Chandrasekhar, Lovelace, and Harvey found
these independent blockers:

1. The online driver held the hidden hypothesis and called oracle partition and
   filter functions before terminal instead of possessing only `Teacher.Respond`.
2. Profile, transcript, bundle, and training-digest preimages did not implement
   the accepted canonical typed schemas; HTML escaping and pretty output also
   made the committed bytes differ from reported byte counts.
3. Curriculum and episode evidence was self-certified. There was no fresh-store
   replay, strict certificate admission, complete matrix verifier, or
   post-selection replay.
4. Required response authorization, boundary verification, sealed artifact
   envelopes, collision rejection, semantic cache, and transcript artifacts
   were absent or incomplete.
5. The driver created most curriculum inputs, while CUE did not materialize the
   required tie and selection ledger or prove unique `(rule,seed)` coverage.
6. Episode and curriculum work counters were estimates or constants rather than
   measurements of `causal-work/v1`; the curriculum attributed-unit count of one
   was demonstrably false.
7. Training and evaluation controls and several validity flags were hardcoded
   true instead of derived from executed adversarial trials.
8. Clean-commit panel gates were not enforced, the cost-skewed RNG consumed
   extra draws, and generated aliases/presentation order never reached the
   production descriptor.

The pretty training report is 403,778 bytes and its bundle is 787,830 bytes,
while their internal compact-byte fields claim 341,004 and 646,700. These
disagreements alone invalidate the run.

## Required disposition

V1 is closed as invalid and must not be overwritten or rerun. Semantic fixes
require an `active-causal-diagnosis/v2` amendment, wholly disjoint seed panels,
a new clean pretraining commit, fresh adversarial implementation review, and a
single v2 training execution followed by independent replay before freezing.
