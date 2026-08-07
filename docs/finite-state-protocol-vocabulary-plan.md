# Finite-state protocol vocabulary plan

## Research question

Can the source-parity EURISKO kernel form and retain a reusable behavioral
relation over small transition systems, rather than merely execute a
hand-written protocol checker? The implementation separates an integration
control from a blinded discovery trial so plumbing success cannot be reported
as discovery.

## Status and review record

Implemented on 2026-08-07 after three adversarial architecture/test review
rounds. The first review rejected a named, scripted conjecture as a discovery
claim. The stabilized design therefore separates the purpose-built control
from a blinded Cartesian search. Later reviews added exact trace semantics,
shortest counterexamples, vocabulary-scoped words, complete evidence units,
opaque aliases, independent exhaustive comparison, held-out evaluation,
invalid-input behavior, repeated-focus guards, and process/store determinism.
Both reviewers accepted the final plan before implementation began.

The implemented 3-transform by 2-relation trial creates six candidates and 18
subject observations. It promotes canonicalization/equivalence,
canonicalization/same-encoding, and unreachable-state-removal/equivalence. It
does not promote either relation for the destructive transition-removal decoy,
nor exact encoding equality for unreachable-state removal. The target
unreachable-state-removal/equivalence schema also passes the held-out corpus
after discovery.

## Two experimental lanes

### Lane A: integration control

The control proves that a non-mathematical vocabulary can load selected
semantic primitives, execute them through operation units, create credited
result/evidence units, and emit attributed conjectures. A purpose-built control
heuristic may report rejecting traps. Its result is an integration baseline,
not evidence that EURISKO discovered the analysis.

### Lane B: blinded relation discovery

The discovery trial exposes several unary protocol transforms, several binary
relations, and training protocols. A relation-agnostic heuristic evaluates
every type-compatible `(transform, relation)` pair. It creates a candidate,
records applications and evidence, changes candidate worth according to
support and failure, and promotes only pairs supported by every training
protocol.

The heuristic never names `RemoveUnreachableProtocolStates` or
`EquivalentProtocols`. Decoy transforms and relations are evaluated by the
same path. A held-out corpus is constructed only after discovery and is never
inserted into the running store. Tests interpret promoted schema units against
that corpus.

The first target schema is:

```text
EquivalentProtocols(P, RemoveUnreachableProtocolStates(P))
```

Success means Nous forms that reusable schema, records why it formed it, and
the schema passes the unseen corpus. Merely creating named protocol facts does
not satisfy Lane B.

## Encoding and semantics

The `protocols` pack represents a partial deterministic finite automaton as a
list of canonical records:

```cue
data: [
    "state:idle",
    "state:awaiting",
    "state:established",
    "event:ack",
    "event:begin",
    "start:idle",
    "accept:established",
    "trans:awaiting,ack>established",
    "trans:idle,begin>awaiting",
]
```

Names match `[A-Za-z0-9_-]+`. Parsing rejects whitespace, unknown tags,
malformed delimiters, undeclared references, duplicate starts, and conflicting
transitions. Exact duplicate state, event, accept, and transition declarations
are accepted and canonicalized away. There must be exactly one start state;
zero accepting states are allowed.

Missing transitions enter an internal rejecting sink whose representation
cannot collide with user state names. Unknown but syntactically valid events
also enter that sink. Empty traces are accepted exactly when the start state is
accepting.

Canonical output orders states, events, start, accepting states, and
transitions lexicographically within that fixed section order. Canonicalization
is permutation-invariant and idempotent. Trimming retains all declared events
but removes unreachable state, accept, and transition records.

The control reports `RejectingTrapStates`, not “deadlocks”: reachable states
from which no accepting state is reachable. This includes terminal rejecting
states and rejecting trap cycles without pretending that an acceptance
automaton models progress obligations.

“Equivalence” means accepted-trace language equivalence. The pure comparison
API returns both its decision and a certificate:

```go
Compare(a, b Machine) (equivalent bool, witness []string)
```

Product-state breadth-first traversal uses the sorted union event alphabet.
The initial pair is checked before consuming an event, so an empty witness is
possible. A non-equivalence witness is shortest, deterministic among equal
length witnesses, and accepted by exactly one machine.

## Vocabulary-scoped primitive words

Protocol algorithms live in `internal/vocab/protocol`, independent of the DSL
and engine. Thin adapters provide:

- `protocol-valid?`
- `protocol-canonicalize`
- `protocol-reachable-states`
- `protocol-rejecting-trap-states`
- `protocol-remove-unreachable`
- `protocol-drop-first-transition`
- `protocol-accepts?`
- `protocol-equivalent?`
- `protocol-same-encoding?`

These words are not global kernel words. A minimal per-VM vocabulary registry
starts with base words. `ProtocolVocabulary` is a `Vocabulary` instance with
`dslExtension: "protocol"`. VM construction enumerates `Vocabulary` instances,
sorts their extension identifiers, rejects empty, duplicate, or unregistered
identifiers, and enables each registered word set in that order. Forked sub-VMs
clone the resolved word registry; they do not rescan mutable store state. Tests
must prove that a math or build-graph VM rejects protocol words while a
protocols VM executes them, that unknown and duplicate markers fail
initialization, and that multiple extensions resolve deterministically. This
is selection, not a public plugin API.

`ValidProtocol` is a total representation classifier: malformed input returns
false and does update that predicate's rarity. Every other protocol operation
or predicate returns `nil` on malformed input. Predicate rarity is otherwise
updated only for non-nil results, so invalid semantic applications do not
become false behavioral evidence.

## Ontology, seeds, operations, and decoys

The pack defines `Vocabulary`, `ProtocolVocabulary`, `Protocol`,
`ProtocolTrainingExample`, `ProtocolTransform`, `ProtocolRelation`,
`ProtocolRelationCandidate`, `ProtocolRelationSchema`, `ProtocolEvidence`,
`StateList`, and `Trace`.

Training machines use neutral names and include multiple structures: fully
reachable machines, machines with unreachable structure, and a machine with a
rejecting trap. CUE does not assert their semantic relationships.

Transform units:

- `RemoveUnreachableProtocolStates` — expected language-preserving transform;
- `CanonicalizeProtocol` — positive control;
- `DropFirstProtocolTransition` — plausible but generally destructive decoy.

Relation units:

- `EquivalentProtocols` — accepted-trace language equivalence;
- `SameProtocolEncoding` — exact canonical encoding equality decoy.

Ordinary control operations also expose validity, reachability, rejecting
traps, and trace acceptance. Every operation declares domain, range, arity,
definition, and prose.

## Heuristics, scheduling, and credit

Every training protocol has an `initialTasks` analysis task. Every transform
receives the kernel's ordinary operation task.

`H-ProtocolControlAnalysis` is Lane A. It is guarded by
`protocolControlAnalyzed`, creates a rejecting-trap evidence unit when needed,
and emits at most one attributed control conjecture per source machine.

`H-DiscoverProtocolRelations` is Lane B. It is guarded per transform by
`protocolRelationsExplored`. For each transform/relation pair it:

1. applies the transform to every training protocol;
2. creates a credited result unit and records the transform application;
3. evaluates the relation between source and result;
4. creates one evidence unit holding training subjects, result units, support,
   failures, comparison method, transform, and relation;
5. creates one candidate with initial worth 500, then adjusts worth by
   `+50 * support - 100 * failures`; and
6. promotes an immutable schema unit only when support is at least three and
   failures are zero.

Promotion also creates one `ProtoConjec` about the transform and relation, then
replaces its generic evidence with the evidence-unit identity and records the
comparison method. Persistent guards and canonical pair names prevent repeated
focus from inflating support. Existing result, evidence, candidate, or schema
names are never overwritten.

This trial exercises operation enumeration, application records, creditors,
evidence-based worth changes, concept creation, and conjecture formation. It
does not claim that the math-specific H1-H29 corpus transfers unchanged; it
tests a new relation-agnostic heuristic using EURISKO's representation,
scheduling, credit, and concept-formation mechanisms.

## Held-out evaluation and baseline

After the engine stops, tests construct unseen machines with alpha-renamed
states, different alphabets, unreachable trap cycles, no unreachable states,
and no accepting states. They read each promoted schema's transform and
relation slots and execute that pair against the hidden corpus.

An exhaustive control evaluates every exposed pair directly. Nous must identify
the same valid schema; the research record compares candidates evaluated,
schemas retained, support/failure evidence, and final worth. This first slice
does not claim search-efficiency superiority over enumeration.

## Verification contract

Pure algorithm tests cover:

- every malformed encoding class and exact-duplicate normalization;
- input permutation invariance and canonicalization idempotence;
- reachability and rejecting terminal/trap-cycle detection;
- empty, accepted, rejected, missing-transition, and unknown-event traces;
- equivalence reflexivity, symmetry, transitivity, and state alpha-renaming;
- arbitrary unreachable-structure insertion preserving language;
- trim idempotence and language preservation;
- inequivalence at the initial pair and after events;
- deterministic shortest witnesses accepted by exactly one side; and
- differential checking against exhaustive trace enumeration for generated
  tiny machines up to the product-state distinguishing bound.

Adapter and engine tests cover:

- selected word exposure and sub-VM inheritance;
- invalid semantic predicate inputs returning nil without changing rarity,
  while `ValidProtocol` returns false and records that classification;
- independent `protocols` loading with no math-only units;
- Lane A positive and negative controls;
- relation-candidate evidence, application records, creditors, and exact worth;
- promotion of the expected schema without promotion of destructive decoys;
- an opaque-alias trial that removes the canonical transform/relation unit
  names and still discovers the corresponding semantic pair;
- the complete Cartesian matrix: exactly one candidate and evidence unit per
  transform/relation pair, one result and relation observation per training
  subject, and `support + failures == training count` for every candidate;
- exact agreement of every candidate's support, failures, final worth, and
  promotion decision with the independent exhaustive control;
- no duplicate support after repeated focus;
- held-out schema success without held-out units entering the store; and
- no result or conjecture from malformed semantic inputs and no rarity change
  except the explicit `ValidProtocol` classification.

Determinism verification runs three separate CLI processes with a fixed cycle
count and `-no-mutate`, comparing stdout byte-for-byte. Tests also compare a
canonical final-store snapshot with sorted units, slots, conjecture evidence,
and support counts. Fixed-seed mutation-enabled runs must also produce
identical stdout and canonical snapshots. Future diversity experiments must
expose an explicit seed and remain reproducible separately for each seed.

Repository verification remains `mise exec -- go test ./...`,
`mise exec -- go test -race ./...`, `mise exec -- go vet ./...`, and
`git diff --check`.

## Scope exclusions

- nondeterministic or epsilon transitions;
- timed, probabilistic, concurrent, or communicating automata;
- protocol synthesis from logs;
- live-service or deployment mutation;
- promotion into PUDL or execution through Mu;
- claiming that Lane A is discovery; and
- a public plugin API before another vocabulary requires it.

## Completed implementation sequence

1. Implemented the pure parser, canonicalizer, analyses, comparison certificate,
   and generated/metamorphic tests.
2. Added the selected per-VM vocabulary registry, protocol adapters, sub-VM
   inheritance, and nil-rarity regression test.
3. Added the `domains/protocols` ontology, neutral seeds, operations, decoys,
   evidence model, and both heuristics.
4. Added integration-control, blinded-discovery, held-out, repeat-focus, and
   canonical-store tests.
5. Ran process-level determinism checks and the complete repository contract.
