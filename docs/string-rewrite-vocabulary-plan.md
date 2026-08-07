# String-rewrite vocabulary plan

## Research question

Can Nous construct a new executable operation by composing independently
represented rewrite operations, select the composition by behavioral evidence,
and propagate the resulting worth growth back to the components that created
it? The protocol experiment selected relations over supplied transforms. This
experiment must instead materialize and execute operations that did not exist
in the seed vocabulary.

## Review status

Stabilized on 2026-08-07 after three adversarial architecture and experimental
review rounds. Both reviewers accepted this revision before implementation.

## Implementation outcome

Implemented and verified on 2026-08-07. The seed trial constructs all 12
ordered compositions and their 48 linked results and observations. Exactly one
program satisfies the four-example corpus: the operation containing `ab -> x`
followed by the operation containing `xc -> y`. Its worth rises from 500 to
800; the synthesis heuristic and both components each receive one `+150`
credit reward, while both decoys receive none.

The alternate runtime trial removes every seed rule and example, installs a
different shuffled problem, and promotes its different uniquely correct pair.
Opaque/adversarial identities, occupied artifact names, generated probes,
primitive deletion, malformed corpora, overflow, and fixed literal held-out
outputs all pass. No mutation is eligible in the mutation-enabled control.

The bounded target is a deterministic two-pass string transformation. Nous is
given four primitive rewrite operations and four input/output examples. It must
construct every ordered pair of distinct primitives, execute each synthesized
program, retain evidence for every candidate, and promote the unique program
that satisfies the complete training corpus. The expected program is never a
seed unit or named by a heuristic.

## Claims and non-claims

Successful execution demonstrates:

- construction of new operation units with executable definitions;
- behavioral selection among order-sensitive compositions;
- complete positive and negative evidence rather than success-only reporting;
- provenance from synthesized operation to heuristic and component operations;
- kernel worth-growth credit propagating to successful components;
- rejection of decoys and resistance to opaque operation aliases; and
- execution of the promoted program on examples absent during discovery.

It does not demonstrate open-ended program synthesis, learned search control,
credit-guided pruning, useful HindSight rules, heuristic mutation, grammar
induction, or superiority to exhaustive enumeration. Search is deliberately
exhaustive and the baseline performs the same finite enumeration. Credit is
observed after selection; this slice does not claim that credit improved the
search that produced it.

## Rewrite semantics

Text and rule fragments contain only lowercase ASCII letters. Training and
held-out inputs are at most 64 bytes. A primitive rule has a non-empty left
side of at most eight bytes and a right side of at most eight bytes, which may
be empty. A primitive operation replaces all non-overlapping occurrences of
its left side in one left-to-right pass. Go's `strings.ReplaceAll` behavior is
the normative implementation model.

A program is an ordered list of primitive operation identities. Program
execution feeds the input through each operation exactly once. It therefore
always terminates. Each intermediate and final value is capped at 256 bytes;
an invalid input, invalid rule, or overflowing result returns `nil` through the
DSL adapter. A semantic `nil` cannot become a successful result, application,
behavioral support, or rarity observation. During corpus evaluation it instead
produces an explicit diagnostic observation included in the candidate's
evidence.

Pure semantics live in `internal/vocab/rewrite`, independent of units, the DSL,
and the engine. They validate text/rules and apply one rule or an ordered rule
sequence. Tests cover empty strings, deletion rules, overlapping matches,
order sensitivity, output limits, composition associativity as function
application, and differential agreement with a small independent reference
implementation.

## Vocabulary-scoped words

`RewriteVocabulary` selects `dslExtension: "rewrite"`. The extension adds:

- `rewrite-valid?` — total representation predicate for a string;
- `rewrite-rule-valid?` — total classifier for `(left, right)`;
- `rewrite-replace-all` — semantic operation returning `nil` on invalid input;
- `rewrite-rule-applies?` — semantic predicate returning `nil` on invalid
  input; and
- `rewrite-output-length` — semantic measurement returning `nil` on invalid
  input;
- `rewrite-compose-name` — an order-preserving, reversible encoding of two
  component identities; and
- `rewrite-artifact-name` — a collision-safe encoding of an artifact kind,
  program identity, and example identity, with a deterministic fresh suffix
  when the encoded base is already occupied.

The words remain unavailable to math, build-graph, and protocol VMs. Child VMs
inherit the selected registry. As with protocols, only total validity
classifiers record false rarity for malformed representations; semantic nils
do not alter predicate rarity.

## Ontology and seed corpus

The pack defines `RewriteVocabulary`, `RewriteString`, `RewriteStringResult`,
`RewriteTrainingExample`, `PrimitiveRewriteOp`, `CompositeRewriteOp`,
`RewriteProgramEvidence`, `RewriteObservation`, and `RewriteProgramSchema`.
The shared CUE unit schema is widened to permit scalar `data: string` in
addition to the existing list forms. Rewrite examples store scalar `input` and
`expected` slots; every `RewriteStringResult` stores its output as scalar
`data`. Synthesized unary operations therefore consume the same representation
whether their argument came from a seed example or an earlier result unit.

Four primitive operation units encode these rule bodies under neutral unit
names:

```text
ab -> x
xc -> y
bc -> z
x  -> q
```

Each primitive stores authoritative `rewriteLeft` and `rewriteRight` slots as
well as an ordinary executable DSL definition. A parity test applies every
primitive definition to generated probes and compares it with an independent
test-only scanner driven by those structured slots. The first two rules
compose, in that order, to implement the target behavior. The latter two are
plausible decoys; reversing the target pair must also fail.

Training examples are:

```text
abc    -> y
zabc   -> zy
abcc   -> yc
abcabc -> yy
```

Names and prose do not identify which rules solve the examples. Tests include
an opaque-alias run that replaces all primitive identities before VM and
engine construction, followed by a stronger runtime trial described under
blinding below.

Held-out cases are created only after the run and never inserted into the unit
store. They add new prefixes/suffixes and repeated target substrings. Expected
outputs are fixed independently rather than generated by the discovered
program during the assertion.

## Synthesis lane

`H-ComposeRewritePrograms` is generic over `PrimitiveRewriteOp`. Once per
primitive it enumerates every other primitive, constructing one operation for
each ordered distinct pair. For four primitives the exact search space is 12
programs. A synthesized unit contains:

- `isA: ["CompositeRewriteOp", "UnaryOp", "Op", "Anything"]`;
- ordered `components`;
- a `defn` formed by concatenating the component definitions in execution
  order;
- domain, range, arity, and synthesis-method slots;
- creditors containing the synthesis heuristic and both components; and
- one `rewriteEvaluation` task.

`rewrite-compose-name` preserves order and encodes component boundaries, so
`A then B` and `B then A` cannot collide even for adversarial component names.
A persistent `compositionsGenerated` guard prevents duplicates under repeated
focus. An occupied generated base name is never overwritten or silently
reused: deterministic fresh-name allocation selects a suffix, and the
candidate's ordered `components` slots remain authoritative. The heuristic
contains no primitive names, rule fragments, expected outputs, or
target-program identity.

## Evaluation lane

`H-EvaluateRewritePrograms` handles only `rewriteEvaluation` tasks for
unevaluated composite operations. For every training example it:

1. validates the input and expected output;
2. applies the synthesized operation to the input;
3. on a valid result, creates a data-bearing `RewriteStringResult` unit and
   records its identity as the direct application's output;
4. creates one immutable observation with input, expected output, actual
   output or diagnostic status, outcome, result-unit identity when present,
   program, and example identities; and
5. increments support, failure, or invalid count.

It preflights and records the complete corpus cardinality, then creates one
evidence unit containing every example and observation, the exact support,
failure, invalid, and evaluated counts, and `exhaustive-training-corpus/v1` as
the method. Invalid inputs, expected outputs, rules, and overflowing results
produce explicit diagnostic observations; they never produce a successful
result/application or affect predicate rarity. Every candidate is retained. A
candidate rises from creation worth 500 to worth 800 only when the corpus has
at least three examples, evaluated count equals corpus size, support equals
corpus size, and failure and invalid counts are zero. Every other candidate is
set to worth 300.

Only an exact candidate creates a `RewriteProgramSchema` and an attributed
`ProtoConjec` whose evidence is the evidence-unit identity. Schema creation
sets its creation and last-rewarded baselines equal to its final worth so it
cannot contaminate the component-credit measurement.

The evidence and evaluation guards prevent repeated tasks from changing
counts or creating duplicate artifacts. Every artifact identity uses the
scoped collision-safe allocator. Tests pre-seed occupied base identities and
prove they are neither overwritten nor mistaken for experiment output.

## Credit observation

Composite units name the synthesis heuristic and both components as creditors.
The engine's existing periodic worth-growth mechanism sees the exact program's
increase from 500 to 800 and rewards each creditor by 150. Because every
non-exact candidate ends below its creation worth, it creates no positive
worth-growth reward.

Acceptance tests run enough focus cycles for the periodic reward pass and
assert that the synthesis heuristic and both target components each receive
exactly `+150`, while decoys receive zero. They check that the composite's
`lastRewardedWorth` becomes 800 and that a later reward interval changes none
of those worths. This proves one-shot credit propagation, not that credit
influenced this exhaustive search.

## Independent baseline and blinding

Tests independently enumerate all ordered distinct primitive pairs with a
test-only left-to-right replacement scanner that does not call the production
rewrite package or DSL adapter. For every candidate they compare each literal
expected and actual output, diagnostic status, support, failure, invalid count,
worth, and promotion with the units produced by Nous. The oracle reads only
structured `rewriteLeft`/`rewriteRight` slots, never executable definitions or
canonical operation names.

The first opaque-alias trial deletes all four canonical primitive units and
recreates them under shuffled opaque identities while preserving their
structured rules and definitions. It must still produce all 12 candidates,
identify the semantically correct ordered pair, promote exactly one schema,
and pass the held-out corpus. After discovery, tests delete the primitive units
and execute the promoted composite again, proving that its definition inlines
executable behavior rather than delegating to seed identities.

A second runtime-created trial removes every seed primitive and training
example before VM construction. It installs different rule fragments, shuffled
identities in which the target pair is not lexicographically first, a different
uniquely correct ordered pair, and independently fixed training and held-out
outputs. The same unmodified heuristics must materialize the full matrix and
promote only that pair. This defeats hardcoding of the seed rule bodies,
corpus, component names, and enumeration position.

This is a blinding test of the generic heuristics, not a claim that the finite
search is difficult.

## Verification contract

Pure and adapter tests cover:

- exact validation bounds and malformed input classes;
- one-pass, non-overlapping, left-to-right replacement semantics;
- deletion, no-match, overlap, order-sensitive, and output-overflow cases;
- sequence execution and independent-reference differential tests;
- parity between each structured primitive rule and its executable `defn`;
- every synthesized definition over generated probes versus sequential
  reference execution of its ordered components;
- applying a synthesized operation to a prior `RewriteStringResult.data`
  scalar and obtaining another valid scalar result;
- vocabulary-scoped exposure and child-VM inheritance;
- total validity-predicate rarity versus semantic-nil rarity; and
- malformed values returning explicit nil/false according to the adapter
  contract without affecting semantic-predicate rarity.

Engine acceptance tests cover:

- independent loading without math-, protocol-, or build-graph-only units;
- exactly 12 composite candidates, 12 evidence units, 48 result units, and 48
  observations for the valid seed corpus;
- exactly four application records per composite;
- exact one-to-one links among candidates, examples, results, observations,
  evidence, applications, and their independently computed values;
- complete agreement with the independent exhaustive baseline;
- exactly one promoted schema and conjecture;
- rejection of reversed and decoy compositions;
- immutable evidence and counts under repeated focus/tasks;
- component and heuristic provenance, with all three receiving exactly one
  `+150` reward and decoys receiving none;
- a later reward interval producing no second reward;
- opaque aliases, adversarial names, and pre-existing artifact-name collisions
  preserving the result without overwrites;
- the alternate runtime corpus discovering a different semantic pair;
- the promoted program remaining executable after primitive deletion;
- malformed inputs/outputs, invalid rules, and overflow producing diagnostic
  evidence but no promotion or false complete-corpus claim;
- held-out success without held-out units entering the store;
- canonical final-store equality both in-process and from separate CLI
  processes using a generic final-store JSON snapshot mode; and
- byte-identical stdout across three no-mutation and three mutation-enabled
  CLI processes.

The mutation-enabled replay is explicitly an enabled-but-inactive control: it
asserts that no heuristic meets the mutation threshold and that no mutant is
created. It does not count as a test of mutation determinism.

Repository verification remains:

```sh
mise exec -- go test ./...
mise exec -- go test -race ./...
mise exec -- go vet ./...
git diff --check
```

## Scope exclusions

- iterative rewriting to a fixed point;
- nondeterministic rule choice, backtracking, or critical-pair analysis;
- arbitrary-length synthesis or mutation of rule text;
- regex, context-free grammar, parsing, or grammar induction;
- deleting failed candidates or generating HindSight rules;
- claiming search guidance from post-selection credit;
- PUDL/Mu integration or external side effects; and
- changing the frozen protocol experiment except for reusable kernel fixes.

The only planned reusable kernel addition is deterministic canonical store
JSON exposed by a CLI snapshot flag. It supports process-level reproducibility
for every vocabulary and does not alter discovery semantics.

## Completed implementation sequence

1. Implemented and tested the pure bounded rewrite semantics.
2. Registered the scoped rewrite adapter and its rarity behavior.
3. Added the rewrite ontology, primitive rules, examples, and both generic
   heuristics.
4. Added exhaustive, alias, held-out, repeat-focus, malformed-input, credit, and
   snapshot acceptance tests.
5. Documented the pack and ran the complete verification contract.
