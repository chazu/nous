# Tiny-stack program-synthesis vocabulary plan

## Research question

Can a domain-neutral, declaratively configured heuristic protocol construct
bounded executable programs, retain complete failure evidence, and select the
shortest program satisfying an input/output corpus when the represented values
are integer stacks rather than strings or configurations?

A second lane asks whether the same protocol can identify finite-probe
simplifications between synthesized two-instruction programs and supplied
primitive instructions.

This follows the [bounded-program synthesis analysis](bounded-program-synthesis-analysis.md).
The descriptor is a candidate reusable interface. Because the current loader
cannot share CUE units between domain packs without copying or putting them in
`common`, this experiment does not yet claim cross-pack heuristic reuse.

## Review status

Accepted after adversarial architecture, experiment-validity, and credit
review. Each review required revisions, and all reported blockers are closed.

## Bounded claim

The seed descriptor supplies seven unary integer-stack instructions and asks
one generic synthesis heuristic to construct every ordered sequence with
repetition of length one through three:

```text
7 + 7^2 + 7^3 = 399 candidates
```

Every candidate is evaluated against four examples. Exactly two candidates
match the complete corpus: `over add` and the behaviorally redundant
`over swap add`. A barriered finalizer sees the complete 399-candidate matrix
and uniquely selects the shorter two-instruction program. Names, prose,
enumeration order, and allocated identities do not participate in selection.

Success demonstrates bounded ordered program construction, complete
underflow/mismatch evidence, global shortest-exact selection, executable
reuse, finite-probe simplification discovery, and position-sensitive credit.
It does not demonstrate unbounded synthesis, inferred examples, global program
equivalence, learned heuristic abstraction, safe execution of arbitrary code,
or transfer of the CUE heuristics to another domain pack.

## Stack semantics

An integer stack is a DSL list containing zero through seven strict integers.
Admitted inputs have depth zero through four and values in `[-100, 100]`.
Input, intermediate, and output magnitudes are bounded to
`[-100_000_000, 100_000_000]`.
Execution returns semantic nil for a malformed stack, unknown opcode,
underflow, depth overflow, or arithmetic overflow. No partial output escapes a
failed program.

The semantic opcode set supported by the scoped stack layer is:

```text
dup     ( ... a -- ... a a )
swap    ( ... a b -- ... b a )
drop    ( ... a -- ... )
over    ( ... a b -- ... a b a )
add     ( ... a b -- ... a+b )
mul     ( ... a b -- ... a*b )
double  ( ... a -- ... 2a )
neg     ( ... a -- ... -a )
```

The seed vocabulary supplies the first seven. `neg` is used only by the
alternate runtime trial. Stack order is bottom to top. Equality requires exact
Value kind, list length, order, and integer values.

Pure semantics live in `internal/vocab/tinystack`. They validate stacks and
opcodes, execute one instruction or a bounded program, and never depend on
units, the DSL, or the engine.

## Scoped extensions

The pack selects two independent vocabulary extensions:

- `StackVocabulary` / `stack`, which supplies `stack-valid?`,
  `stack-input-valid?`, `stack-exec-op`, and `stack-equal?`; and
- `ProgramSynthesisVocabulary` / `programsynth`, which supplies strict,
  domain-neutral descriptor validation, ordered sequence identity,
  executable-definition composition, collision-safe allocation, structural
  finalization checks, and semantic decision identity.

The program-synthesis words may inspect units and execute descriptor-declared
validators/comparators. They must not mention stack categories, stack bounds,
instruction names, example outputs, or target sequences. Both extensions are
absent from unselected VMs and inherited by operation child VMs.

## Descriptor contract

`StackSynthesisExperiment` is an instance of
`BoundedProgramSynthesisExperiment`. It declares:

```text
experimentKey                 stack/ordered-programs/v1
primitiveCategory             StackInstruction
candidateCategory             StackProgramCandidate
exampleCategory               StackProgramExample
valueCategory                 IntegerStack
resultCategory                StackProgramResult
observationCategory           StackProgramObservation
evidenceCategory              StackProgramEvidence
selectionEvidenceCategory     StackSelectionEvidence
promotedSchemaCategory        StackProgramSchema
inputSlot                     input
expectedSlot                  expected
resultValueSlot               data
inputValidator                ValidIntegerStackInput
outputValidator               ValidIntegerStack
comparator                    EqualIntegerStacks
primitiveSemanticSlot         semanticOpcode
maxLength                     3
minCorpus                     4
primitiveCap                  8
exampleCap                    16
probeCap                      16
candidateCap                  600
simplificationComparisonCap   4096
synthesisMethod               ordered-sequences-up-to-3/v1
creditContext                 stack/corpus-a/ordered-sequences-up-to-3/v1
synthesis/evaluation/finalization task slots and priorities
probeCategory                 StackSimplificationProbe
probeInputSlot                data
simplificationProgramLength   2
simplificationPairCategory    StackSimplificationPair
simplificationExecutionObservationCategory StackSimplificationExecutionObservation
simplificationExecutionResultCategory StackSimplificationExecutionResult
simplificationComparisonObservationCategory StackSimplificationComparisonObservation
simplificationEvidenceCategory StackSimplificationEvidence
simplificationSchemaCategory  StackSimplificationSchema
probeSetVersion               stack-probes-a/v1
```

Before allocating anything, descriptor validation requires all referenced
categories, slots, validators, and comparator operations to exist with the
declared arities; category roles that must be distinct to be distinct;
`1 <= maxLength <= 3`; fixed priority ordering; bounded names and text; at
least `minCorpus` and no more than the configured example cap; a bounded probe
corpus; and an exact candidate count below the cap.

Every primitive must be a defn-only unary endomorphism over the declared value
category, have a nonempty bounded definition, and carry a unique bounded
semantic key in the descriptor-selected slot. Malformed primitives or examples
invalidate the complete descriptor. They are never silently skipped to create
a smaller search.

Every generated candidate, task artifact, schema, and conjecture links back to
the descriptor through `synthesisExperiment`.

## Seed instructions and examples

Neutral instruction units carry these semantic opcodes:

```text
dup swap drop over add mul double
```

Each stores `semanticOpcode` and the self-contained definition
`"<opcode>" stack-exec-op`. Primitive worth starts at 500 so even a
three-occurrence credited program remains below the scalar clamp.

Training examples define the behavior `(a b -- a a+b)`:

```text
[ 2,  3] -> [ 2,  5]
[-1,  4] -> [-1,  3]
[ 0, -2] -> [ 0, -2]
[ 5,  0] -> [ 5,  5]
```

An independent enumerator establishes before comparison that, among all 399
programs, 246 execute successfully on each training input, producing exactly
984 results and 612 invalid observations. Eighteen candidates match one
example, 379 match none, and the two exact candidates are the length-two sequence
`over add` and length-three sequence `over swap add`.

## Candidate identity and credit declaration

Allocated names encode the experiment key, length, and ordered allocated
component identities with boundary-preserving base64url. Occupied bases receive
the first deterministic collision suffix and are never overwritten.

Semantic decision identity is `sha256:v1:<hex>` over canonical JSON containing
the synthesis-method version and ordered, repetition-preserving primitive
semantic keys. Thus aliases preserve the decision, while `A B`, `B A`, `A A`,
and `A` are distinct.

Creditors contain the synthesis heuristic followed by component occurrences.
Roles are `synthesis`, `step-1`, `step-2`, and `step-3`. Repeated primitives
are intentionally occurrence-weighted: ordinary scalar credit applies once per
occurrence, while contextual roles distinguish positions. This can bias global
scalar scheduling toward repeated instructions and is a compatibility
artifact, not a claimed learning advantage. Component-role credit uses
allocated identities and is not alias-invariant; decision credit is semantic
and alias-invariant.

Every declaration must pass `credit.ValidDeclaration`.

## Synthesis and barrier protocol

`H-EnumerateBoundedPrograms` works only on the descriptor's synthesis task.
Its initial worth is 750, matching the evaluation and selection heuristics.
After complete preflight it performs one guarded generation firing that:

1. constructs the authoritative list of all 399 unique candidates;
2. gives each candidate a self-contained concatenated definition, ordered
   components, ordered semantic sequence, experiment link, length, method,
   contextual declaration, and one unique evaluation task;
3. stores `expectedCandidateCount`, `candidateUnits`, and
   `evaluatedCandidateCount=0` on the descriptor; and
4. sets `generationComplete=true` only after allocation finishes.

The store is not transactional. An unexpected interruption may leave partial
units, but they remain structurally ineligible for evaluation or finalization
because generation completion and the authoritative candidate list are absent.

No finalizer is scheduled by priority assumption. Each idempotently evaluated
candidate increments the descriptor once. The transition to
`evaluatedCandidateCount == expectedCandidateCount` schedules exactly one
finalization task and sets `finalizationScheduled=true`.

The finalizer independently revalidates that generation is complete, candidate
identities are unique, every listed candidate belongs to this experiment and
has exactly one evidence unit, and all counts agree. Repeated or premature
finalizer tasks are ineligible.

## Evaluation and shortest selection

`H-EvaluateBoundedProgram` resolves all category, slot, validator, comparator,
and corpus names through the candidate's descriptor. For every example it:

1. validates input and expected values through the declared unary validators;
2. applies the candidate with ordinary `apply-op`;
3. creates a result and direct application only for a valid returned value;
4. compares valid actual and expected values through the declared binary
   comparator; and
5. creates one observation with application validity, match outcome, status,
   actual value or nil, and linked result identity.

Every candidate receives one evidence unit with complete example, result, and
observation lists plus corpus, evaluated, support, failure, and invalid counts.
Evaluation never raises worth: non-exact candidates become 300 and exact
candidates remain at their creation baseline of 500.

`H-SelectShortestExactProgram` runs only after the structural barrier. It
collects candidates with complete corpus support and zero failures/invalids,
records the full exact set and its minimum length, and applies this tie policy:

- no exact candidates: `selectionStatus=no-solution`, no promotion;
- more than one minimum-length candidate: `selectionStatus=co-minimal`, retain
  and promote every co-minimal program while recording the complete class; or
- exactly one minimum-length candidate: `selectionStatus=selected`, promote
  it from 500 to 800.

Every promoted candidate receives its own baseline-safe schema and attributed
conjecture. No lexical or digest tie-break silently turns a real equivalence
class into a unique winner.

The seed therefore retains both exact candidates but uniquely promotes
`over add`. Selection contains no opcode names or expected values.

## Finite-probe simplification lane

After finalization, `H-CompareProgramSimplifications` compares every generated
length-two candidate with every supplied primitive on this declared probe set:

```text
[] [2] [2,3] [-1,4,5] [0,-2] [3,0,1]
```

Both valid outputs are compared by the descriptor comparator. Two undefined
results count as agreement of partial functions, but promotion additionally
requires at least three probes on which both sides are defined. This prevents
vacuous underflow equivalence. The lane first materializes and reuses 336
execution observations (`49 * 6` composite and `7 * 6` primitive), including
213 valid results/applications and 123 undefined observations. It then creates
2,058 pair/probe comparison observations and one aggregate evidence unit for
each of the `49 * 7 = 343` pairs. Evidence distinguishes both-defined equal,
both-undefined, one-undefined, and defined mismatch counts. Negative evidence
is retained.

The preregistered positive seed findings are:

```text
dup swap   probe-equivalent to dup
dup add    probe-equivalent to double
swap add   probe-equivalent to add
swap mul   probe-equivalent to mul
double drop probe-equivalent to drop
```

Each positive pair creates a baseline-safe `StackSimplificationSchema` and
attributed conjecture. These are finite-probe observations, not global
equivalence or authority to rewrite programs. Independent exhaustive held-out
probes over every stack of depth zero through four using values
`{-2,-1,0,1,2}`—781 stacks—must confirm all promoted relations without
entering the store.

A vacuity control replaces the probes with `[]`, `[1]`, and `[2]`. On those
inputs both `swap add` and `add` are undefined three times and are jointly
defined zero times. Their evidence must therefore record
`bothUndefinedCount=3`, `bothDefinedCount=0`, and no promoted schema.

## Credit observation

Only final selection raises a candidate above its creation baseline. The
winner's 500-to-800 change gives an ordinary actual `+150` to
`H-EnumerateBoundedPrograms`, the `over` component, and the `add` component.
It creates one decision contextual record of 300 and three role records of 150.
All other primitives and decisions receive no positive record.

The test must reach a real post-finalization unit-focus cycle divisible by ten,
assert the exact record set/source/evidence counts and scalar worths, then run
the same engine through another eligible interval and byte-compare credit,
scalar, winner, and schema state. Repeated finalizer and evaluation tasks must
also leave artifacts unchanged.

## Alternate, alias, and held-out controls

An opaque-alias trial replaces all canonical primitive identities before VM
construction and preoccupies the winning candidate plus representative result,
observation, evidence, selection-evidence, and schema bases handled by the
single uniform artifact allocator. Every sentinel must survive and every fresh
artifact must remain correctly linked. Conjectures use the kernel's separate
semantic-deduplication policy and are not collision-allocation claims. The
semantic winner and decision remain unchanged. After discovery all primitives
are deleted and the winner still executes, proving its definition is
self-contained.

A runtime alternate replaces the descriptor's category names, primitive
units, corpus, probe units, method, and credit context. Its seven opcodes are:

```text
dup swap drop over add mul neg
```

Its examples define `(a b -- a-b)` through `neg add`:

```text
[ 5,  3] -> [ 2]
[-2,  4] -> [-6]
[ 0, -3] -> [ 3]
[ 7,  0] -> [ 7]
```

The two exact programs are `neg add` and `neg swap add`; the unmodified generic
heuristics must uniquely choose the shorter program. The alternate matrix also
contains 399 candidates, 984 results, and 612 invalid observations. Its
finite-probe lane must find exactly four simplifications: `dup swap -> dup`,
`swap add -> add`, `swap mul -> mul`, and `neg drop -> drop`.

The alternate held-out cases are:

```text
[9,-2]  -> [11]
[-5,-7] -> [2]
[0,0]   -> [0]
[8,5,3] -> [8,2]
```

Both exact programs must pass all four without adding units to the store, and
the four promoted simplifications must pass the same independent 781-stack
exhaustive sweep as the seed relations.

Literal held-out seed examples include new values and an extra lower stack
element:

```text
[10,20]  -> [10,30]
[-7,-8]  -> [-7,-15]
[100,0]  -> [100,100]
[3,4,5]  -> [3,4,9]
```

Held-out values never enter the discovery store.

An ambiguity corpus uses distinct prefixes ending in `[2,2]` or `[0,0]`, with
the expected stack formed by applying either `add` or `mul`:

```text
[9,2,2]  -> [9,4]
[-3,0,0] -> [-3,0]
[2,2]    -> [4]
[0,0]    -> [0]
```

It must produce exactly the two length-one minima `add` and `mul`, record
`selectionStatus=co-minimal`, and create two program schemas and two
selection conjectures.

## Negative controls

Tests separately require:

- malformed descriptors, category overlap, invalid priority ordering,
  duplicate semantic keys, malformed primitives/examples, and exceeded caps
  allocate no candidates;
- removal of either target semantic instruction yields a complete no-solution
  matrix and no promotion;
- duplicate behavior under distinct semantic keys can create two shortest
  exact candidates, producing a recorded co-minimal class with both promoted;
- a literal target requiring `dup dup dup dup` produces a bounded-depth
  no-solution result because no length-three candidate can match its depth;
- a premature or count-forged finalizer cannot select;
- repeated instruction sequences preserve position-sensitive decision and
  credit declarations without accidental deduplication;
- two no-mutation runs have byte-identical canonical stores.

Heuristic mutation is outside this vocabulary trial: it changes the heuristic
population rather than the descriptor-selected synthesis protocol and is
covered by the engine's mutation tests, not treated as evidence for or against
this bounded vocabulary.

## Acceptance and verification

Acceptance requires:

- isolated loading with both selected extensions and no foreign domain units;
- pure, scoped adapter, and primitive-definition parity tests;
- exact independent agreement for all 399 candidates and 1,596 observations,
  including exact results, support/failure/invalid counts, worth, and evidence;
- the unique shortest seed and alternate selections with the complete longer
  exact alternatives retained;
- exactly 336 simplification execution observations, 213 valid execution
  results/applications, 2,058 pair/probe comparisons, 343 pair evidence units,
  and the preregistered five finite-probe schemas in the seed trial;
- alias, collision, primitive-deletion, held-out, alternate-category,
  invalid-descriptor, no-solution, ambiguity, barrier, and deterministic-store
  controls; and
- exact one-shot ordinary and contextual credit.

## Implemented result

The seed run meets the preregistered result: all 399 candidates are evaluated,
`over add` and `over swap add` are exact, and only the shorter program is
selected. The simplification lane creates the exact 336 execution
observations, 213 results, 123 undefined observations, 343 pair evidence units,
and 2,058 comparisons, promoting the five predicted relations. A normal
no-mutation CLI agenda reaches finalization and simplification at cycles 415
and 416.

The fully renamed alternate selects `neg add`, retains `neg swap add`, and
finds its four predicted simplifications. The ambiguity corpus promotes both
`add` and `mul`; the undefined-only control promotes nothing. Independent
oracles validate every candidate, example, result, direct application,
simplification execution, and pair comparison. Held-out, exhaustive,
collision, primitive-deletion, malformed/no-solution, forged-evidence,
one-shot-credit, guarded-replay, and canonical-determinism controls pass.

Verification commands are:

```sh
mise exec -- go test ./...
mise exec -- go test -race ./...
mise exec -- go vet ./...
git diff --check
```
