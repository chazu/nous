# Transformation-schema induction vocabulary plan

## Status and authority

Provisional Part 3 Vocabulary 2 implementation plan, revision 8. This document
is not implementation authority until independent architecture, transformation-
semantics, and experimental-validity reviewers all accept the same committed
revision.

Revision 1 was blocked by all three reviewers. Its endpoint made exhaustive PBE
weakly dominate Nous; curriculum ordinal leaked a modulo family assignment;
forest, baseline, statistic, transcript, and protected-evidence semantics were
not closed; oracle audit was causally entangled with promotion; and several
caps were inconsistent. Revision 2 replaces the dominated search with a frozen
factorized-elimination procedure, shuffles sealed family assignments, makes
oracle checks post-freeze only, and closes those contracts below.

Revision 2 was blocked because its generator conditioned acceptance on unique
success of the Nous-specific factor stages, making the intended advantage
constructed rather than empirical. It also had a refinement-order conflict,
random evidence handles, non-invariant ID renaming, impossible event-size
arithmetic, incomplete operation preimages, and ambiguous held-out/oracle
phases. Revision 3 removes the algorithm-conditioned filter and makes ambiguity
and `no-discovery` empirical outcomes, while closing those executable details.

Revision 3 passed the substantive semantic and experimental review but was
blocked on evidence-protocol closure: one event-count sum was wrong, the chain
seed and referenced object wires were incomplete, a policy-visible evidence
digest remained a behavior surface, several secondary controls were unplaced,
and the report omitted its claimed evidence-graph digest. Revision 4 closes
those remaining authority and accounting contracts.

Revision 4 was blocked on the last executable-wire boundary. Its node result
was not field-identifiable, schema predicate and whole-application events were
conflated, factor evidence could not inhabit the result wire, edit validation
had no valid result type, and `digest` necessarily intervened before `attach`.
It also left three serialized preimages undefined and contradicted abstention
scoring. Revision 5 adds exact compound fact/status/application/reference
objects, separates predicate from application events, makes immediate attach
hash the supplied semantic value inside the verifier, freezes success and
failure arities, closes the missing preimages, and normalizes abstention and
false-application scoring.

Revision 5 was blocked because its standalone semantic-reference leaf and
evidence leaf could not both be admitted by one immediate evidence-link event,
and it did not define how a nested application-result projection was
authenticated. Its program-batch cap was also 88 bytes below the computed
legal maximum. Revision 6 inlines the result kind and digest in evidence,
references the already-admitted output object, freezes deterministic nested-
projection authentication, and raises the program-batch cap.

Revision 6 was blocked because a rejected attachment produced no evidence
object, losing the policy-supplied value and making wrong, correct, stale, and
zero-output attachment attempts indistinguishable. Revision 7 makes every
attachment call produce one authenticated attempt object containing its status,
the exact supplied semantic value and digest, and the preceding output/operation
digests. Success and rejection therefore share the same bounded replay path.

Revision 7 was accepted as a plan, but exact implementation review exposed two
contradictions in the accepted protocol. Its operation matrix prohibited
training-validation node/parent/target events that its required exact
schema-application traces necessarily emit, and its locked statistics both
erased every private random seed and required later byte-exact reconstruction
of the resulting inference. No protected v1 panel was executed. Revision 8
closes v1 as unexecuted and defines `transform-schema/v2`: application-internal
fact events inherit the outer application phase, locked statistical randomness
is publicly reproducible from the precommitted private-root commitment, and
stateless PBE policies no longer manufacture Store evidence. All unaffected
term, edit, program, partial, and schema object grammars remain v1.

Vocabulary 1 ended `valid-null`; Vocabulary 2 does not consume its domain,
artifacts, evidence, or learned state. It uses only the existing Store, Agenda,
engine, scoped DSL-extension, and ordinary-heuristic surfaces.

The experiment asks one narrow question: can Nous turn several independently
discovered concrete scalar edits into an inspectable role-parameterized schema,
specialize it with counterexamples, and use the materialized schema to perform
exact coordinated transformations on alpha-renamed held-out term forests under
a fixed lifecycle-work budget?

It does not claim general tree differencing, arbitrary refactoring, competitive
program synthesis, semantic code transformation, or transfer to Kubernetes,
Terraform, Mu, or PUDL.

## Preregistered manifest

The implementation must expose this canonical JSON byte-for-byte from a source
constant and reproduce it in every report:

```json
{
  "experiment_version": "transform-schema/v2",
  "seed_authority": "part3/transform-schema/v2",
  "generator_version": "request-reference-forest/v1",
  "term_version": "typed-reference-forest/v1",
  "edit_grammar_version": "set-scalar-from-request/v1",
  "schema_grammar_version": "anchor-target-scope-old-guard-locality/v1",
  "oracle_version": "independent-forest-transform/v1",
  "baseline_version": "lgg-and-bounded-pbe/v1",
  "cache_version": "disabled/v1",
  "cost_version": "transform-lifecycle-events/v2",
  "statistics_version": "paired-stratified-resampling/v2",
  "report_version": "transform-schema-trials/v2",
  "policy_fixture_version": "transform-policy-curriculum/v1",
  "scorer_fixture_version": "transform-scorer-curriculum/v1",
  "transcript_version": "transform-events/v2",
  "integrity_contract": "budgeted-transcript",
  "training_examples_per_curriculum": 8,
  "training_positive_examples": 4,
  "training_negative_examples": 4,
  "heldout_cases_per_curriculum": 8,
  "heldout_positive_cases": 4,
  "heldout_abstention_cases": 4,
  "development_seeds": {"start": 841001, "count": 48, "step": 1},
  "validation_seeds": {"start": 842001, "count": 96, "step": 1},
  "locked_curricula": 128,
  "maximum_nodes": 12,
  "maximum_groups": 2,
  "maximum_requests": 2,
  "maximum_definitions": 2,
  "maximum_references": 6,
  "maximum_concrete_edits": 4,
  "anchor_modes": ["request-target", "from-value", "first-local"],
  "target_masks": ["definition", "references", "definition+references"],
  "reference_scopes": ["local", "global"],
  "old_value_guards": ["equals-from", "any"],
  "anchor_locality_guards": ["required", "none"],
  "schema_candidates": 72,
  "semantic_families": 9,
  "candidate_refinement_edges": 138,
  "nous_refinement_edges_per_curriculum": 12,
  "nous_candidate_allocations_per_curriculum": 13,
  "schema_application_cap_per_curriculum": 48,
  "competence_schema_application_cap": 26000,
  "competence_program_application_cap": 8000,
  "competence_work_cap": 5000000,
  "generator_schema_application_cap_per_attempt": 1200,
  "generator_schema_application_cap_per_curriculum": 120000,
  "generator_work_cap_per_attempt": 200000,
  "generator_work_cap_per_curriculum": 20000000,
  "oracle_work_cap_per_curriculum": 250000,
  "engine_cycle_cap_per_curriculum": 2000,
  "attributed_unit_cap_per_curriculum": 20000,
  "lifecycle_work_cap_per_curriculum": 12000,
  "report_byte_cap": 16777216,
  "fixture_bundle_byte_cap": 16777216,
  "transcript_event_cap_per_policy_curriculum": 50000,
  "transcript_event_cap_per_policy_locked_panel": 6400000,
  "transcript_raw_byte_cap_per_policy_curriculum": 19200000,
  "transcript_gzip_byte_cap_per_policy_curriculum": 19250000,
  "object_byte_cap_per_policy_curriculum": 67108864,
  "object_leaf_byte_cap": 2560,
  "object_leaf_count_cap_per_policy_curriculum": 24002,
  "object_root_byte_cap_per_policy_curriculum": 4194304,
  "transcript_raw_byte_cap_per_policy_locked_panel": 2457600000,
  "transcript_gzip_byte_cap_per_policy_locked_panel": 2464000000,
  "object_byte_cap_per_policy_locked_panel": 8589934592,
  "minimum_locked_success_advantage": 0.10,
  "minimum_locked_success_rate": 0.80,
  "maximum_false_application_rate": 0.00,
  "maximum_nonmatching_work_ratio": 1.25,
  "alpha": 0.05,
  "confidence_interval": "paired-stratified-bootstrap-two-sided-95",
  "paired_test": "paired-randomization-two-sided",
  "bootstrap_replicates": 10000,
  "randomization_replicates": 10000,
  "bootstrap_indices_zero_based": [249, 9749],
  "development_power_outer_replicates": 2000,
  "development_power_inner_replicates": 2000,
  "minimum_locked_power": 0.80,
  "tie_policy": "minimum-description-length-then-canonical-code-retain-all-ties",
  "mutation_enabled": false
}
```

Panels are disjoint. Public development and validation seeds are exactly the
declared arithmetic sequences. A locked root is generated only after a guarded
one-shot receipt exists; it deterministically derives 128 private curriculum
seeds and is erased before report publication. Generator rejection and cohort
assignment depend only on frozen term semantics, never on a policy result.

Any further change to objects, grammar, fixtures, information rights, costs,
baselines, statistics, thresholds, or execution guards creates
`transform-schema/v3` and preserves the unexecuted v1 preregistration and all
v2 evidence.

## Terms, forests, and exact semantics

A forest uses the exact canonical array encoding below. Array form makes field
order part of the versioned grammar rather than an implementation convention.

```text
["typed-reference-forest/v1", [node...]]
node := [id, kind, parent, key, value, from, to, target]
```

`id`, `parent`, and `target` are JSON integers. IDs are unique and exactly
`0..n-1`; input node order is arbitrary and canonical encoding sorts by ID.
Absent integer fields are `-1`; absent strings are `""`. `kind` is exactly one
of `group`, `request`, `definition`, `reference`, or `decoy`. Every nonempty
string is 1--16 lowercase ASCII letters, each forest is at most 2,048 canonical
bytes, and decoding rejects duplicate fields by construction, unknown versions,
wrong array lengths/types, trailing JSON values, invalid UTF-8, and noncanonical
numbers.

Kind invariants are exact:

| Kind | parent/key | value | from/to | target |
| --- | --- | --- | --- | --- |
| group | `-1` / empty | empty | empty | `-1` |
| request | group ID / nonempty | empty | both nonempty | definition ID |
| definition | group ID / nonempty | nonempty | empty | `-1` |
| reference | group ID / nonempty | nonempty | empty | definition ID |
| decoy | group ID / nonempty | nonempty | empty | `-1` |

All non-group nodes are direct children of a group; therefore `group(node)` is
exactly its parent and "local" always means parent equality. Child keys are
unique within a group. Targets must name a definition. Typed target edges do not
participate in ownership. Maximum kind counts are those in the manifest;
positive documents have exactly one request, while abstention documents may
have zero or two. `from` may equal `to` only in a declared no-op abstention case.

Opaque IDs, keys, and scalar spellings remain part of each concrete forest's
byte identity. IDs are stable structural descriptor positions, not names.
Alpha-renaming is a fixture relation, not canonicalization: it applies
bijections to child keys within each group and scalar strings, then optionally
permutes the serialized node array while retaining IDs. A separate descriptor-
relayout control regenerates IDs and is a competence-only diagnostic for
`request-target` and `first-local`; it never enters a training/held-out case,
`N_i`, `B_i`, false applications, work ratio, or empirical policy row.
`first-local` is explicitly ID-sensitive and is expected to fail some relayout
microcases without invalidating its declared semantics. A schema contains no concrete key or
scalar. Allocated Store unit names are never forest IDs and are excluded from
serialized semantics.

V1 has one primitive edit:

```text
set-value(target-node, literal-value)
```

It changes exactly one node scalar. Targets must be definitions or references;
requests, groups, and decoys cannot be edited. A concrete program is a sorted,
duplicate-free sequence of one through four edits with distinct target nodes. Its
result is obtained by applying edits in target-position order. Since targets
are distinct, order is semantically irrelevant but still canonical and charged.
Invalid targets, duplicate targets, missing values, no-op edits, and programs
outside the bound fail closed.

An edit wire value is
`["set-value/v1", targetID, literal]`. A concrete program is
`["concrete-program/v1", [edit...]]`. Decoders enforce the same limits and EOF
rule as forests. A positive case is
`[opaqueCaseToken,"positive",before,after]`; an abstention case is
`[opaqueCaseToken,"abstain",before,null]`. The case token is exactly 16 ASCII
bytes matching `[0-9a-f]{16}`, drawn independently, and has no seed, ordinal,
attempt, or family encoding.

The oracle independently applies edits, compares complete canonical forests,
and enumerates every bounded concrete program for tiny competence cases. It
shares only version strings and serialized byte slices with production; it has
its own structs, strict decoder, canonical encoder, validator, and executor.

## Schema language

A schema is the tuple

```text
(anchor, targets, reference-scope, old-guard, anchor-locality)
```

with dimensions frozen by the manifest:

- `anchor=request-target` selects the definition named by the request's typed
  target edge;
- `anchor=from-value` selects the unique definition whose scalar equals the
  request's `from` value;
- `anchor=first-local` selects the first definition, by descriptor position,
  owned by the request's group;
- `targets=definition`, `references`, or `definition+references` chooses which
  roles expansion edits;
- `scope=local` includes only inbound references owned by the request's group;
- `scope=global` includes all inbound references to the selected definition;
- `old-guard=equals-from` includes a reference only when its scalar equals the
  request's `from` value;
- `old-guard=any` imposes no reference-scalar guard;
- `anchor-locality=required` requires the selected definition to be owned by
  the request's group; and
- `anchor-locality=none` admits an externally owned selected definition.

The Cartesian grammar therefore contains exactly
`3 * 3 * 2 * 2 * 2 = 72` schemas. Scope and old-value guard remain identity-
bearing even when `targets=definition`; their redundancy is charged in the
description length and retained so the universe stays rectangular and auditable.

The only operative learned artifact is
`["transform-schema/v1",anchor,targets,referenceScope,oldGuard,anchorLocality]`.
It contains exactly those six array elements, is canonical UTF-8 JSON, and is
hashed before precommitted held-out bytes are decoded or released to the policy.
Curriculum tokens, profile digests,
evidence links, worth, Store names, family labels, and provenance are separate
non-operative records. One frozen artifact is used unchanged for all eight
held-out cases. Application accepts only its bytes plus the current forest; no
context key, dispatch table, cache key, or curriculum identifier is an input.

Application scans nodes in increasing ID order and short-circuits in this exact
sequence:

1. collect requests; a count other than one yields `abstain/request-count`;
2. bind an anchor: `request-target` follows the request edge, `from-value`
   scans **all** definitions and requires exactly one with value equal to
   `request.from`, and `first-local` selects the lowest-ID definition whose
   parent equals the request parent; no binding yields `abstain/anchor`;
3. evaluate anchor locality; failure yields `abstain/locality`;
4. scan **all** references in ID order, retain those targeting the anchor,
   filter by parent equality only for local scope, then filter by scalar only
   for `equals-from`; this scan occurs and is charged even for a definition-only
   target so redundant schema fields remain observably costly;
5. build the target set in definition-then-reference order, reject empty or
   more than four targets as `abstain/expansion`;
6. construct edits using `request.to`; the first no-op target yields
   `abstain/no-op` without executing any edit; and
7. validate every edit, execute all edits in target-ID order, and return
   `applied` with the complete output forest.

No other short circuit is legal. Failed and abstaining applications consume an
application credit and every local event reached before the terminal.

The verifier-only certificate contains the schema code, request and definition
positions, ordered reference positions, complete guard outcomes, concrete edit
list, input digest, output digest, terminal, and charged event range. CUE and
policy code receive only terminal and output bytes when terminal is `applied`;
they never receive an evidence ID, binding, reference set, edit list, latent
family, or oracle result. The independent
oracle reproduces the
semantic certificate after the policy/store boundary is frozen.

## Description length and refinement graph

Description length is the fixed integer sum:

| Choice | Cost |
| --- | ---: |
| request-target anchor | 1 |
| from-value anchor | 2 |
| first-local anchor | 3 |
| one target role | 1 |
| definition+references | 2 |
| local scope | 1 |
| global scope | 2 |
| equals-from guard | 2 |
| any old value | 1 |
| anchor locality required | 2 |
| no anchor locality guard | 1 |

The root is an explicit five-hole candidate. Refinement fills holes in the
fixed order `targets`, `anchor`, `reference-scope`, `old-guard`,
`anchor-locality`. Every partial tuple has one edge for every legal value of
the next hole. Thus there are exactly `3 + 9 + 18 + 36 + 72 = 138` refinement
edges. The five phase-closure records are separate evidence events and are not
refinement edges. The implementation may not silently count allocation,
closure, or selection as refinement.

A partial candidate is exactly
`["transform-partial/v1",stage,targets,anchor,referenceScope,oldGuard,
anchorLocality]`. `stage` is `0..5`; fields at or before that stage are enum
values and later fields are empty strings. The root has stage zero and all
empty fields. A child increments stage by one and fills exactly the next field
in the order above. Parent digest plus chosen enum value uniquely determines
child bytes. Complete stage-five bytes are converted to the operative schema's
different canonical field order only by an ordinary finalization heuristic,
with both digests linked in evidence.

The 138 edges describe the finite universe; Nous does **not** traverse all of
them. Its factorized policy allocates one root plus every alternative at each
of five stages along one retained branch: `3 + 3 + 2 + 2 + 2 = 12` traversed
edges and 13 allocated candidates. The policy promotes only when its evidence
barrier leaves exactly one value at every stage. A missing or multiple survivor
is `no-discovery`; fixture acceptance does not filter that outcome, and the
policy may not use description length to force a value through an ambiguous
stage.

The exact candidate state machine is:

1. `concrete-open`: independently recover all four positive concrete programs;
2. `target-open`: compare each of three target-mask projections with every
   recovered edit-kind set; retain only exact values, record others as positive
   counterexamples, then close;
3. `anchor-open`: for each anchor value and positive, use local observations to
   test whether its selected definition equals the recovered definition edit
   (or the definition targeted by recovered reference edits); retain exact
   values, then close;
4. `scope-open`: enumerate inbound references locally. For each scope, CUE
   explicitly pairs it once with `equals-from` and once with `any`, compares
   both predicted edited-reference sets with recovered programs, and retains
   the scope iff at least one of its two explicit pairings is exact on every
   positive; then close. Go never computes the existential. If the retained
   target is definition-only, both values are still materialized but `global`
   is rejected as `redundant-noncanonical` and `local` survives by the public
   normalization rule without inspecting an example;
5. `old-guard-open`: under the retained scope, compare `equals-from` and `any`
   separately with every recovered edited-reference set; retain exact values,
   then close. For definition-only, `equals-from` is instead rejected as
   `redundant-noncanonical` and `any` survives by the same normalization rule;
6. `locality-open`: using explicit node/parent/target observations, compare the
   retained anchor's group with the request group in the wrong-context negative.
   Materialize the predicted terminal for both locality alternatives;
   `none` predicts application and is a counterexample, while `required`
   predicts abstention. This is local factor evidence and never invokes the
   complete-schema capability; then close; and
7. `complete`: assemble the sole five-field tuple, apply it to all eight
   training cases, freeze its bytes and evidence root, and only then permit the
   driver to materialize held-out policy inputs.

Stages and value order are fixed as written. A stage closes only after every
listed alternative has a result for every required example and the independent
transcript reducer recomputes the survivor set. A counterexample therefore
causes a concrete `candidate -> rejected` transition before the retained value
is used to refine the next stage. Deleting any stage heuristic, result, or
counterexample prevents completion. The promoted unit records the operative
schema bytes plus non-operative parent, edge, evidence, and closure links.

## Discovery boundary and capabilities

The scoped adapter exposes exactly these capability classes:

| Capability | Explicit inputs | Policy-visible output | Allowed callers |
| --- | --- | --- | --- |
| `node` | forest bytes, node ID | one exact node-facts value containing kind and present scalar fields | CUE |
| `parent` | forest bytes, node ID | parent ID and key | CUE |
| `target` | forest bytes, node ID | target ID or absent | CUE |
| `compare` | two explicit local values | boolean | CUE |
| `edit` | forest bytes, one explicit edit | terminal and output bytes | CUE |
| `schema` | forest bytes, one complete schema | terminal and output bytes if applied | CUE |
| `refine` | one partial candidate, one explicit next value | child candidate bytes | CUE only |
| `digest` | one canonical policy-visible semantic value | SHA-256 string | CUE only |
| `attach` | the most recent operation's policy-visible semantic result value | boolean attached/rejected | CUE only |

Every call also emits metered events to an unforgeable verifier-side sink.
`schema` may internally derive a binding solely as the deterministic consequence
of its two explicit inputs, but the binding, edit expansion, operation digest,
and full certificate never cross into VM values or Store slots. `attach` is
legal only immediately after `node`, `parent`, `target`, `compare`,
`candidate-allocate`, `refine`, `edit-validate`, `edit-apply`,
`schema-application`, `output-compare`, or `verify`. Internal predicate events
are verifier-only and are never attach targets. `attach`
accepts the canonical semantic value already returned by that operation and
hashes it verifier-side before emitting the evidence-link event. The exact
supplied value, kind, digest, and status are fields of the one evidence-attempt
object, not a standalone semantic-reference object; the attempt also names the
preceding output object when one exists.
No separate `digest` call is required or permitted in between. It can bind only that most
recent unattached operation in the same task, returns one boolean, and exposes
no handle, digest, or ordinal. The sink rejects a second attachment, a value
whose canonical digest is not the recorded result projection, task change,
any intervening capability call, a non-attachable operation, or attachment
after phase closure. Independent reduction reconstructs the hidden
operation/result link.
Tests prove no evidence identifier can enter comparison, candidate identity,
worth, priority, dispatch, or any persisted semantic slot.

Baselines never import or invoke this adapter. They independently strict-decode
the same raw policy bytes, implement byte-level equivalents of these local
operations, and emit the identical event categories/charges through a narrow
meter callback supplied by orchestration. Cross-implementation golden traces
require equal results and vectors for the same explicit edit/schema calls.

It may not diff a complete before/after pair, list changed nodes, align forests,
recover a concrete program, inspect multiple examples in one call, anti-unify,
enumerate the schema universe in one call, choose a candidate, compute the
minimum-description tie set, or expose held-out expectations.

Ordinary CUE heuristics must:

1. scan every before/after training node pair through local observations;
2. propose and validate primitive edits;
3. assemble each positive example's exact concrete program;
4. materialize the 12 factor alternatives and exactly the traversed refinement
   edges;
5. attach the stage-specific positive evidence and negative counterexample;
6. reject inconsistent alternatives and close each factor barrier;
7. construct and execute the sole surviving complete tuple; and
8. freeze it before held-out bytes are exposed.

The driver may load policy-view training bytes, run policies in fresh stores,
enforce budgets, and—after receiving the frozen schema digest and immutable
training-store digest—release held-out **inputs**. It cannot create candidate
units, insert or repair concrete programs, order a queue using scorer metadata,
or write held-out outputs into a store. Expected held-out terminals/outputs are
held only by the external scorer and are compared after each policy result.

The independent post-terminal `transformoracle` is never called until
acquisition, candidate selection, schema
freeze, held-out execution, and the policy terminal are immutable. Oracle
agreement is a post-hoc mechanical gate. Its result cannot alter a Store,
agenda, transcript prefix, application budget, policy outcome, or promotion.

## Package and dependency boundaries

Implementation is confined to:

- `domains/transformschema`;
- `internal/vocab/transformschema`;
- `internal/dsl/builtins_transformschema.go`;
- `internal/transformfixturecore` for strict data-only policy serialization;
- `internal/transformoracle` for independent semantics and enumeration;
- `internal/transformbaseline` for conventional algorithms;
- `internal/transformexp` for orchestration, metering, statistics, and guarded
  evidence;
- `cmd/nous` wiring;
- tests and documentation.

No engine, agenda, mutation, credit, common-domain, math-domain, Store, VM, Mu,
or PUDL file may change. The vocabulary, DSL adapter, fixture core, and domain
pack may not import the experiment, baseline, or oracle.
The oracle may import only the standard library and decodes raw bytes into its
own private types. Baselines may import fixture core but not production
vocabulary, DSL, engine, or experiment code. Dependency tests mechanically
enforce these rules.

## Fixtures and cohorts

There is one canonical definition-only family, whose semantically inactive
reference scope and old-value guard are normalized to `local` and `any`, plus
four references-only and four combined-target families formed by reference
scope and old-value guard. Anchor is always `request-target` and anchor locality
is always `required`.

Family enum order is exactly: definition/local/any; references/local/equals;
references/local/any; references/global/equals; references/global/any;
combined/local/equals; combined/local/any; combined/global/equals; and
combined/global/any.

The orchestrator constructs only a count vector: development
`[6,6,6,5,5,5,5,5,5]`, validation `[11,11,11,11,11,11,10,10,10]`, and locked
`[15,15,14,14,14,14,14,14,14]`. It expands that multiset and Fisher-Yates
shuffles it with the panel-level `family-permutation` stream before assigning
generator seeds. Curriculum index therefore has no arithmetic relationship to
family. The shuffled assignment, seed, accepted attempt, and family exist only
in scorer metadata and never in fixture tokens, Store units, queue order,
policy calls, semantic keys, or engine-emitted events.

For each curriculum the generator creates four positive training pairs, four
negative training cases, four positive held-out cases, and four held-out
abstention cases. Every accepted curriculum contains:

- one single-group positive;
- one two-group token-decoy positive;
- one reordered-children positive;
- one positive with an irrelevant same-token definition;
- one zero-request negative;
- one two-request negative;
- one no-op-request negative; and
- one wrong-context negative whose request targets a definition outside its
  owning group.

Negative cases have an input and the expected terminal `abstain`; they do not
provide an answer forest. Positives provide before/after pairs. Held-out cases
are generated independently and alpha-rename every alias, key, and scalar.
They recombine the same structural conditions in a different order.

The latent schema alone produces positive outputs. The independent generator
rejects a curriculum unless (a) the assigned latent code is the unique minimum-
description complete schema consistent with all eight training cases and (b)
it is exact on all eight held-out cases. It does **not** execute, simulate, or
filter on the Nous factor stages, their survivor counts, concrete-program
recovery, any policy budget, or any baseline result. A factor stage may remain
ambiguous or reject every value; that becomes the empirical `no-discovery`
outcome. Semantic acceptance runs before policy tokens and queues are drawn.
Attempts `0..99` run in order; exhaustion is panel-level `invalid` and consumes
no policy observation.

Each accepted-attempt uniqueness check executes exactly 72 schemas over eight
training and eight held-out cases, at most 1,152 applications, under 1,200-
application and 200,000-work per-attempt caps. Its closed maximum is
`1,152 * 80 = 92,160` schema-application work, plus 8,000
decode/validation + 5,000 uniqueness/ordering + 4,000 serialization + one
terminal = 109,161. Across 100 attempts the caps are
120,000 applications and 20,000,000 work.

Post-terminal oracle audit independently rechecks the 1,152-case acceptance
matrix (92,160), up to `6 * 48 = 288` empirical applications (23,040), at most
16 promoted concrete programs at 64 work each (1,024), and has 25,000 for
strict decode, certificate reconstruction, scoring, and its terminal. The
closed maximum is 141,224, below its 250,000-work cap. These construction/audit
ledgers are reported as integrity diagnostics and never debit or replenish a
policy budget.

Three semantic authorities are distinct. Unexported fixture-construction
semantics inside `transformexp` may execute latent and complete grammar schemas
only while building/sealing fixtures; its outputs are scorer bytes, never
policy facts. Sealed scorer bytes are inaccessible during policy execution.
After a policy terminal freezes, the independent `transformoracle` package
decodes raw policy/scorer bytes with its own implementation and audits fixture
truth, every program, application, and score; that audit can only validate or
invalidate published evidence. Finally, exhaustive `oracle-enumerator` is a
competence-only safe-test role over committed microcases, never an empirical
policy, panel row, or protected-fixture caller.

This acceptance rule makes identifiability part of the task definition, not a
claim that Nous found the answer. Production receives no latent schema,
generator attempt, oracle candidate table, held-out outputs, or semantic
uniqueness witness.

Policy and scorer views are freshly decoded from distinct canonical envelopes:

```text
policy training := ["transform-policy-curriculum/v1", profileDigest,
                    [trainingCase...]]
policy heldout := ["transform-heldout-inputs/v1", profileDigest,
                  [[opaqueCaseToken,before]...]]
scorer := ["transform-scorer-curriculum/v1", family, seedCommitment,
           acceptedAttempt, latentSchema, [heldoutExpected...]]
heldoutExpected := [opaqueCaseToken, terminal, outputOrNull]
```

`profileDigest` is the lowercase SHA-256 of the exact canonical preimage
`["transform-profile/v1","typed-reference-forest/v1",
"set-scalar-from-request/v1",
"anchor-target-scope-old-guard-locality/v1",
"transform-lifecycle-events/v2",12,4,72,48,12000]`. It cannot vary by family.
Before execution, fixture construction generates, canonicalizes,
serializes, and root-commits the held-out inputs and sealed expectations.
The policy training envelope contains no held-out bytes. Only after schema
freeze may orchestration decode the already committed held-out-input envelope
and release one input at a time. The scorer envelope is never
decoded by production, DSL, engine, CUE, or baseline packages. Generic
`transformfixturecore` contains only strict data-only policy-envelope types; it
has no panel, seed, root, family, latent-schema, expected-output, generator, or
oracle API.

At 2,048 bytes per forest, policy training uses at most 12 forests, delayed
held-out input uses eight, and scorer expectations use four output forests.
Including fixed 256-byte overhead per case and 4,096 bytes of envelope metadata
gives less than 61,440 bytes per curriculum and 7,864,320 bytes for 128. The
sorted root manifest is bounded by 1 MiB, so the 16 MiB fixture-bundle cap is
closed with more than 7 MiB margin. Reports retain digests and scalar summaries,
not forests or event bodies; 768 policy/curriculum rows at 4 KiB each plus
fixed metadata is below the 16 MiB report cap. Exceeding either cap is `invalid`.

Family permutation has its own panel-level stream. Development and validation
use the first 16 bytes of
`SHA-256(canonical-json(["part3/transform-schema/v2","family-permutation",
panel,authority]))`, with authorities 841001 and 842001. Locked uses the first 16 bytes
of `HMAC-SHA256(root,canonical-json(["part3/transform-schema/v2",
"family-permutation","locked"]))`. Bytes are two big-endian `uint64` PCG
seeds. Fisher-Yates visits `i=len(multiset)-1..1`, draws exactly
`Uint64N(uint64(i+1))`, and swaps `i` with that index. The resulting whole
family vector is scorer-only. Committed safe golden tests freeze all three
count vectors, public permutations, PCG draws, and a fixed-root locked
permutation before protected execution.

After permutation, public curriculum seeds are the declared panel arithmetic
sequence in ordinal order; locked curriculum seeds use the HMAC derivation in
the protected section. Commitments are exact: public
`panelCommitment = SHA-256(canonical-json(["transform-panel/v1",panel,
authority]))` and `seedCommitment = SHA-256(canonical-json([
"transform-seed/v1",panel,seed]))`. For locked,
`panelCommitment = HMAC-SHA256(root,canonical-json(["transform-panel/v1",
"locked"]))`; curriculum material is the HMAC specified in the protected
section, its first eight bytes are the generator seed, and
`seedCommitment = SHA-256(curriculumMaterial)`. All commitments are lowercase
hex strings; raw curriculum material is never serialized.

Per-curriculum streams are the first 128 bits of
`SHA-256(canonical-json(["part3/transform-schema/v2", panelCommitment,
seedCommitment, attempt, purpose]))` interpreted as two big-endian `uint64`
values for PCG. Public development and guarded validation use commitments to
their declared seed; locked uses a commitment to its derived private seed.
Purposes are exactly `structure`, `aliases`, `scalars`, `child-order`,
`case-tokens`, `case-order`, `production-queue`, `random-policy`, and
`baseline-ties`. Streams never share state. Policy-visible tokens and orders use
their own purposes and reveal no generator stream state.

## Concrete-program acquisition

For each positive training pair, heuristics observe all node positions in
canonical position order. They compare corresponding local node facts and
materialize a proposed `set-value` edit only when kind, parent relation, and
target edge are unchanged but the scalar differs. Each proposal is applied to
the before forest in a fresh child VM. A complete concrete program is promoted
when its execution is byte-equal to the after forest. The training-store digest
and program bytes then freeze. Oracle agreement occurs only in the later
post-policy audit and cannot authorize or withhold promotion.

This algorithm is intentionally simple, but it is not a hidden diff primitive:
every observation, comparison, proposal, application, and rejected alternative
is an attributed transcript event. Deleting the concrete-program construction
heuristic must leave no promoted schema. Seeding the expected concrete programs
or changed-node set is forbidden.

Negative cases are never used to construct edits. They are used only as
candidate-schema counterexamples. A schema "rejects" a negative only when its
own application returns a valid `abstain/<reason>`; producing the unchanged
input counts as a false application unless the schema terminal was already a
valid abstention.

## Policies and controls

Every policy execution has fresh isolated state, identical public training
bytes, its own meter, and the same lifecycle/application caps. CUE policies use
a fresh Store; pure-Go PBE policies strict-decode fresh bytes without a Store.
Every locally observed fact and schema application is charged under the common
ledger.

1. `nous-refine`: ordinary heuristics recover concrete programs, traverse the
   exact five-stage factorized state machine, retain counterexamples, close
   every barrier, and promote its sole survivor.
2. `positive-lgg`: a conventional least-general-generalization baseline over
   recovered positive concrete programs. It sees negatives only when scoring
   its frozen result and cannot specialize afterward.
3. `bounded-pbe`: canonical complete-schema enumeration with early rejection
   on the first failed training case and the shared 48-application cap.
4. `random-pbe`: the same evaluator and candidate set in a frozen random order.
5. `concrete-replay`: exact concrete programs keyed by complete canonical input;
   it performs no schema generalization.
6. `no-equality-guard`: Nous with `old-guard=any` forced before selection.

`oracle-enumerator` is the nonempirical competence role defined above and is
not one of these six policies. Wrong-context behavior is measured by the
declared within-curriculum negative cases, not a cross-curriculum donor policy,
so no artifact or family metadata crosses curricula. Corruption is an integrity
microcase: after a competence schema freezes, a copy toggles anchor locality
from `required` to `none` while retaining the original digest; verification
must reject it before any application, with no empirical row or budget.

`concrete-replay` independently pays acquisition, stores a map from each of the
four complete positive-before forest digests to its program, and returns
`abstain/replay-miss` without edits for every other input. Exact hits execute the
stored program. The map freezes before negative scoring and held-out release.
`no-equality-guard` follows the full Nous state machine and allocates all 12
alternatives, but at `old-guard-open` records `equals-from` as
`ablated-ineligible` and retains `any` if every preceding stage had one survivor.
It still runs the complete eight-case training validation; mismatch yields
`no-discovery`. Allocation, closure, and evidence counts remain otherwise
identical to Nous.

Concrete acquisition is not shared across policy executions. For
`positive-lgg`, orchestration first creates that policy's fresh Store and runs
the same ordinary acquisition-only CUE heuristic used by Nous. After its closed
barrier, orchestration byte-compares and serializes the promoted program units
into the exact narrower envelope `["transform-program-batch/v1",
[[opaqueCaseToken,beforeForestDigest,concreteProgram]...]]`, with four rows
sorted by token, unique tokens and digests, and strict canonical/EOF decoding.
It destroys Store access, then invokes pure-Go LGG on that envelope and the
four positive cases. The LGG package neither acquires nor calls the adapter.
`nous-refine`, `concrete-replay`, and `no-equality-guard`
likewise run acquisition independently in their own stores. The driver never
creates, repairs, or copies a program across policies. `bounded-pbe` and
`random-pbe` consume raw training cases directly and receive no changed-node or
program data, so they pay no acquisition cost but derive no acquisition
shortcut.

`positive-lgg` is frozen pseudocode, not a library-dependent label. From its
own four promoted positive programs it tests each value of target, anchor,
reference scope, and old guard in the same stage order as Nous. Target and
anchor retain exact positive values. Scope explicitly tests both old-guard
pairings and retains a scope if either is exact; lowest description cost then
enum order selects one scope and the full surviving-scope tie is recorded.
Old-guard is then tested only under that selected scope and selected by the same
rule. Definition-only uses the public `local`/`any` normalization. Because LGG
does not inspect negatives, locality is the lower-cost `none`. Every dimension
tie records all equal-cost exact values before canonical selection. LGG
assembles the tuple, applies it to the four positives, freezes it if all are
exact, then externally scores the four training negatives without revision.
Failure returns `no-discovery`. It cannot revise after negative-training or
held-out scoring.

`bounded-pbe` sorts the 72 tuples by `(descriptionLength, anchorIndex,
targetIndex, scopeIndex, oldGuardIndex, localityIndex)`, with enum indices in
manifest order. It sorts training cases by opaque token bytes. For each tuple it
applies cases until the first mismatch, charging every attempt. An exact tuple
is retained; enumeration continues through its complete description-length
tier to retain every exact co-minimal tie, then freezes the first canonical
code. Before each application it atomically checks whether one credit remains;
if not, it returns `budget-exhausted` with no schema. `random-pbe` differs only
by its committed independent tuple permutation and stops on the first exact
tuple rather than making an MDL claim.

No policy receives latent family or held-out truth. The fixed primary comparator
is `bounded-pbe`; there is no development-time comparator selection. LGG,
random, replay, and ablations remain mandatory secondary controls and cannot
replace the comparator after observation.

## Lifecycle-work ledger

There is one wall-clock-independent 12-category vector:

| Ordinal | Event | Charge |
| ---: | --- | ---: |
| 0 | observe one node kind/scalar | 1 |
| 1 | observe one parent/key relation | 1 |
| 2 | observe one typed target edge | 1 |
| 3 | compare one local scalar or edge fact | 1 |
| 4 | allocate one partial or complete candidate | 1 |
| 5 | traverse one refinement edge | 1 |
| 6 | validate one primitive edit | 2 |
| 7 | apply one primitive edit | 1 |
| 8 | evaluate one schema guard or binding predicate | 1 |
| 9 | compare one output node with expected output | 1 |
| 10 | attach or check one evidence link | 1 |
| 11 | hash, canonicalize, verify, or finalize one artifact/application | 1 |

Total work is the checked signed-64-bit dot product of event counts and charges.
Overflow is `invalid`. Cache lookup, hit, miss, duplicate rejection, abstention,
corruption rejection, and no-match paths emit their actual events; none is free.
Candidate construction and all training work are lifecycle costs and are not
amortized away. Held-out scoring shares the remaining cap.

The application cap is 48 per policy/curriculum: exactly eight credits
are reserved for held-out cases, leaving at most 40 during training. A policy
cannot consume the reservation. Factor checks made solely from explicit local
observations are not applications; every `schema-application` and
`replay-application` is. Applied, mismatching, failed, and abstaining calls all
consume one. Primitive edits do not consume application credits.

Worst-case successful Nous work is bounded before fixtures as follows:

| Phase | Maximum charged events | Maximum charged work |
| --- | ---: | ---: |
| decode/observe four positives and recover programs | 2,400 | 2,600 |
| 12 factor alternatives over their required evidence | 4,800 | 4,800 |
| candidates, closures, evidence, hashes | 1,000 | 1,000 |
| eight complete training applications | 608 | 640 |
| eight held-out applications and comparisons | 608 | 640 |
| terminal reserve | 1 | 1 |
| total | 9,417 | 9,681 |

The bound charges every maximum-node scan even where semantics short-circuit.
One maximum-node, four-edit, applied complete-schema call plus its required
immediate evidence attachment and complete-output scoring has one of three
golden category vectors, in anchor enum order:

```text
request-target [12,8,7,0,0,0,4,4,26,12,1,1] = 75 events, 79 work
from-value     [12,8,6,0,0,0,4,4,27,12,1,1] = 75 events, 79 work
first-local    [12,9,6,0,0,0,4,4,27,12,1,1] = 76 events, 80 work
```

The twelve node-facts events scan the forest once and their values remain local
to that call. Parent events are six reference parents plus anchor/locality
parents; target events are six reference targets plus the request edge only for
`request-target`. Predicate events are request count, one or two anchor tests,
locality, three filters for each of six references, expansion bound, and four
no-op checks. The remaining entries are four edit validations charged two,
four edit executions, twelve output-node comparisons, one evidence link, and
one final `schema-application` event. A strict precharge uses the applicable
anchor vector and 80-work overall maximum; cap/golden tests pin all three.

LGG's closed bound is acquisition 2,600 + factor work 4,800 + artifact work
1,000 + four positive validation applications 320 + four externally scored
negative-training applications 320 + eight held-out applications 640 + one
terminal = 9,681. PBE's 48 applications cost at most 3,840 work, with at most
1,000 for 72 candidate allocations, ordering, comparisons, evidence, and its
terminal, totaling 4,840. Replay, random, and ablation paths are bounded by the
larger 9,681 figure. Thus 12,000 work, 50,000 events, 2,000 cycles, and 20,000
units are attainable for every intended policy. The 576
universe matrix is deliberately **not** attainable under the application cap;
bounded search is the experiment.

Before event zero, orchestration writes the identical primary/audit
pre-execution leaf
`pre/<policy>/<opaqueTaskToken>.json` with wire
`["transform-policy-manifest/v2","transform-schema/v2",
"transform-lifecycle-events/v2",panelCommitment,policy,opaqueTaskToken,
trainingFixtureDigest,heldoutInputDigest,queueDigest,
[12000,50000,48,2000,20000]]`. All digests name already committed fixture
leaves; no family, seed, attempt, expected output, ordinal, execution role,
transcript, or output digest appears. Its SHA-256 is `policyManifestDigest` and
is fixed before either execution. Because the execution manifest later refers
to this leaf while the leaf never refers to execution output, the chain has no
cycle.

An internal event has no curriculum ordinal. The unforgeable external sink
attaches it and writes the canonical array
`["transform-events/v2",panelOrdinal,policy,opaqueTaskToken,sequence,phase,
category,operation,subjectDigest,objectDigest,outcome,previousDigest]`.
Policy is at most 32 ASCII bytes, opaque task token exactly 16, phase at most
20, operation at most 24, outcome at most 32, and all three digests exactly 64
lowercase hex bytes. Panel ordinal is at most three decimal digits, sequence at
most five, and category at most two. The maximum including quotes is therefore
`21+3+34+18+5+22+2+26+66+66+34+66+11+2+1 = 377` bytes (fields, commas,
brackets, newline). The hard encoded-event cap is 384 bytes. Sequence starts at
zero and has no gaps. The initial previous digest is
`SHA-256(canonical-json(["transform-chain/v1",policyManifestDigest,
opaqueTaskToken]))`; each later event's `previousDigest` is
`SHA-256(canonical-json(["transform-chain-step/v1",priorCanonicalEvent]))`,
where `priorCanonicalEvent` includes its own previous-digest field.

Every top-level semantic input and output preimage resides in the curriculum's
authenticated object table as `objects/<sha256>.json`, a regular canonical JSON
leaf whose bytes hash to its filename. The only nested semantic values are the
exact `result` and `certificate` children of an application leaf and the exact
attempted value at child index 3 of an evidence-attempt leaf. Application
children are authenticated by their parent: reduction loads the unique application named by
the immediately preceding operation's output digest, strict-decodes and
canonical-reencodes it, selects child array index 1 for result or 2 for
certificate, and hashes those exact child bytes. A nested digest is legal only
in the immediately following evidence link or certificate/report
reconstruction and can never stand in for an object-table path. Missing,
nonunique, wrong-parent, wrong-index, or digest-mismatched projection is
invalid. For an evidence attempt, reduction strict-decodes the attempt leaf,
canonical-reencodes child index 3, verifies `attemptedDigest`, and compares that
value and the preceding operation/output digests with the declared status.
Because the current evidence-link operation hashes only those child/prior
digests and its output names the attempt leaf, this nesting is noncyclic. An
operation object is exactly
`["transform-operation/v1",operation,phase,[inputDigest...],
[outputDigest...],outcome,category,charge]`; digest arrays have at most eight
items and are ordered by semantic operand position.

All semantic object wires and per-object canonical byte caps are frozen:

| Kind | Exact wire | Cap |
| --- | --- | ---: |
| atom | `["transform-atom/v1",type,value]` | 128 |
| node facts | `["transform-node-facts/v1",kind,value,from,to]` | 192 |
| parent facts | `["transform-parent-facts/v1",parentID,key]` | 128 |
| forest | the `typed-reference-forest/v1` wire | 2,048 |
| edit | the `set-value/v1` wire | 128 |
| edit status | `["transform-edit-status/v1",status,editDigest]` | 256 |
| program | the `concrete-program/v1` wire containing one through four edit wires | 640 |
| program batch | the `transform-program-batch/v1` wire | 1,152 |
| partial | the `transform-partial/v1` wire | 256 |
| schema | the `transform-schema/v1` wire | 256 |
| result | `["transform-result/v1",terminal,outputDigest]` | 256 |
| closure | `["transform-closure/v1",stage,parentDigest,[[alternativeDigest,resultDigest,status]...],survivorDigest]` | 1,024 |
| certificate | `["transform-certificate/v1",schemaDigest,inputDigest,requestID,definitionID,[referenceID...],[guardBoolean...],[editDigest...],outputDigest,terminal,firstSequence,lastSequence]` | 2,048 |
| application | `["transform-schema-application/v1",result,certificate]` | 2,560 |
| evidence attempt | `["transform-evidence-attempt/v1",status,attemptedKind,attemptedValue,attemptedDigest,outputObjectDigest,priorOperationDigest]` | 2,560 |
| terminal | `["transform-terminal/v1",policyTerminal,work,applications,lastSequence]` | 256 |
| store boundary | `["transform-store-boundary/v1",phase,storeBytesDigest]` | 256 |
| operation | the `transform-operation/v1` wire | 1,536 |

Atom type is one of `id`, `kind`, `key`, `scalar`, `enum`, `boolean`, or
`digest`, with the corresponding already bounded JSON primitive. Node facts
repeat the node's exact kind and use `""` for every scalar field absent under
the kind table; they never contain ID, parent, key, or target. Parent facts
contain the exact parent ID and child key and exist only for non-group nodes.
Closure has at
most four alternatives. Certificate IDs/reference arrays obey forest/edit
bounds. The exact maximum program batch is 1,104 bytes: four rows, each with a
16-byte token, 64-byte digest, and a four-edit program using the largest four
distinct legal target encodings and 16-byte literals. `edit-status.status` is
exactly `valid`, `no-op`, or `invalid-input`. Evidence-attempt status is exactly
`attached` or `rejected`, and its attempted semantic kind is exactly
`node-facts`, `parent-facts`, `atom`, `partial`, `schema`, `edit-status`,
`forest`, `result`, or `closure`. `attemptedValue` is the exact canonical value
supplied by policy code and `attemptedDigest` hashes its canonical bytes.
`outputObjectDigest` names the immediately preceding operation's admitted
top-level output object, or is `""` if that operation had no output. An
inapplicable digest is `""`; it is never an object-table path.
The event `subjectDigest` is the primary input object and `objectDigest` is this
operation object's digest. Thus the reducer authenticates actual preimages
rather than inferring semantics from category totals.

The normative operation matrix is:

| Operation | Legal phases | Category/charge | Input kinds -> output kinds | Outcomes |
| --- | --- | ---: | --- | --- |
| node | acquire,target,anchor,scope,old-guard,locality,training-validate,heldout | 0/1 | forest,id -> node-facts | ok,invalid-input |
| parent | same as node | 1/1 | forest,id -> parent-facts | ok,absent,invalid-input |
| target | same as node | 2/1 | forest,id -> atom | ok,absent,invalid-input |
| compare | acquire,target,anchor,scope,old-guard,locality | 3/1 | atom,atom -> atom(boolean) | true,false,invalid-input |
| candidate-allocate | target,anchor,scope,old-guard,locality,freeze | 4/1 | partial or schema -> same kind | allocated,duplicate,rejected |
| refine | target,anchor,scope,old-guard,locality | 5/1 | partial,atom(enum) -> partial | refined,rejected,invalid-input |
| edit-validate | acquire,training-validate,heldout | 6/2 | forest,edit -> edit-status | valid,no-op,invalid-input |
| edit-apply | acquire,training-validate,heldout | 7/1 | forest,edit -> forest | applied,invalid-input |
| schema-predicate | training-validate,heldout | 8/1 | forest,schema,atom(selector),atom-or-edit(subject) -> atom(boolean) | true,false,invalid-input |
| output-compare | acquire,training-validate,heldout | 9/1 | forest,forest -> atom(boolean) | equal,different,invalid-input |
| evidence-link | acquire,target,anchor,scope,old-guard,locality,training-validate,freeze,heldout | 10/1 | attempted-value,prior-output-or-empty,prior-operation -> evidence-attempt | attached,rejected |
| canonicalize | all nonterminal phases | 11/1 | any one semantic kind -> same kind | canonical,invalid-input |
| hash | all nonterminal phases | 11/1 | any one semantic kind -> atom(digest) | hashed,invalid-input |
| verify | acquire,training-validate,freeze,heldout | 11/1 | one or two semantic kinds -> atom(boolean) | verified,rejected |
| schema-application | training-validate,heldout | 11/1 plus one application credit | forest,schema -> application | applied,abstain/request-count,abstain/anchor,abstain/locality,abstain/expansion,abstain/no-op,invalid-input |
| replay-application | training-validate,heldout | 11/1 plus one application credit | forest,program-batch -> result | applied,abstain/replay-miss,invalid-input |
| terminal | terminal | 11/1 | closure or schema or store-boundary -> terminal | completed,no-discovery,budget-exhausted |

The schema-predicate selector is exactly `request-count`, `anchor-candidate`,
`anchor-locality`, `reference-target`, `reference-scope`,
`reference-old-guard`, `expansion-bound`, or `edit-no-op`. Its subject is an
ID/count atom for every selector except `edit-no-op`, whose subject is an edit.
The reducer recomputes the boolean from all four inputs; a selector/subject
kind mismatch is `invalid-input`.

Output arity is exact by outcome. `ok`, `allocated`, `refined`, `applied`,
`canonical`, and `hashed` produce the one output shown. Compare `true`/`false`,
output-compare `equal`/`different`, and verify `verified`/`rejected` each produce
one boolean atom. Every edit-validation outcome produces one edit-status.
Every schema-application outcome produces one application object, using `-1`,
empty arrays, and empty digests in its certificate when a field is
inapplicable; every replay-application outcome produces one result. `attached`
and evidence-link `rejected` each produce one evidence-attempt object.
`completed`, `no-discovery`, and
`budget-exhausted` produce one terminal. `absent`, `duplicate`, every other
`rejected`, and every other `invalid-input` produce zero outputs. For an
evidence link, the operation input-digest array is exactly
`[attemptedDigest,outputObjectDigestOrEmpty,priorOperationDigest]`. The one
attempt output authenticates the full supplied value and recomputes its digest;
it also permits replay to distinguish a wrong value, the correct value after an
intervening operation, a non-attachable prior operation, and a prior zero-output
failure. It does not admit another reference object. Other inputs always
have the exact arity shown and are present even on failure. Candidate allocation
therefore records a complete-schema allocation for every PBE tuple rather than
coercing it through a partial candidate.

At a schema/replay call boundary the verifier atomically reserves the
applicable maximum work and one application credit before emitting any internal
predicate/edit event. Exactly one final schema-application or
replay-application event commits that reservation and is the sole event counted
as the application. A missing, duplicated, or nonfinal application event, or a
crash between reservation and commit, is mechanical `invalid`; it cannot turn
partial work into a free empirical attempt.

In `training-validate`, node, parent, and target operations are legal only
inside that exact deterministic reserved schema-application trace. They inherit
the outer call phase. The lifecycle reducer rejects any such fact operation
outside a reservation-derived trace; matrix legality does not authorize
free-standing policy observations during training validation.

No other phase/category/charge/kind/arity/outcome combination is valid. Result
terminal is limited to the schema-application outcomes above plus
`abstain/replay-miss` for the concrete-replay control. `ablated-ineligible`, `survivor`,
`redundant-noncanonical`, and `counterexample` occur only as closure status
strings, not operation outcomes.

Operation is closed by category: `node`; `parent`; `target`; `compare`;
`candidate-allocate`; `refine`; `edit-validate`; `edit-apply`;
`schema-predicate`; `output-compare`; `evidence-link`; or one of
`canonicalize`, `hash`, `verify`, `schema-application`, `replay-application`,
and `terminal`. Caches are disabled; a cache
operation is invalid evidence and duplicate applications are fully charged.
Phase is exactly `acquire`, `target`, `anchor`, `scope`,
`old-guard`, `locality`, `training-validate`, `freeze`, `heldout`, or `terminal`.
Unknown combinations fail reduction.

Caps are abort-before-exceed. One event/work slot is reserved for the terminal;
an action whose complete precomputed maximum charge or application credit does
not fit is not started, and the reserved `budget-exhausted` terminal is written.
No partial action rolls back. Budget exhaustion marks every unattempted held-out
case incorrect and assigns the full 12,000 work cap as that policy's
nonmatching-work diagnostic, preventing early exhaustion from appearing cheap.

Each policy/curriculum transcript is one deterministic raw JSON-lines chunk,
gzip-compressed with fixed header fields and no concatenated members. It has
caps of 50,000 events, 19,200,000 raw bytes (`50,000 * 384`), and 19,250,000
gzip bytes. The gzip cap exceeds the implementation's frozen
`compressBound(19,200,000)+18` result; cap-1/cap/cap+1 tests pin that bound. At
most one operation object and one non-operation object are admitted per charged
event. Compound node-facts, parent-facts, edit-status, and application objects
make every matrix operation obey that rule; the policy-visible result of an
application is a projection of the one authenticated compound object, and an
evidence-link admits only its operation object plus its one evidence-attempt
object. Since
every object is at most 2,560 bytes and work permits at most 12,001 events
including terminal, the table has at most 24,002 leaves using 61,445,120 bytes.
Its sorted root has the exact object-root wire in the evidence section, a
4,194,304-byte cap, and counts inside the 67,108,864-byte curriculum cap,
leaving 1,469,440 bytes margin. A
locked policy panel has exactly 128 chunks and caps of
6,400,000 events, 2,457,600,000 raw bytes, 2,464,000,000 gzip bytes, and
8,589,934,592 object bytes. Primary and audit bundles are separate; combined
caps are exactly twice those values.

Independent reduction validates gzip framing and every event, reconstructs
candidate allocations/refinement parents, stage survivors and closures,
application reservation/consumption, evidence links, frozen artifact bytes,
held-out terminals/output digests, all vectors/work, policy/case/curriculum
terminals, and report statistics. It rejects any event after terminal. Primary
and audit executions separately decode immutable policy/scorer bytes into fresh
stores and scorer state. Equality of complete report semantics and positional
transcript hashes is a mechanical gate.

## Competence suite

Before any empirical label is valid, both production and the independent
oracle must agree on committed microcases covering:

1. all 72 schema encodings round-trip canonically;
2. each of the three anchors with unique, absent, and ambiguous bindings;
3. definition-only, references-only, and combined expansion;
4. local versus global reference scope;
5. equals-from versus any-old behavior;
6. no-op-request and anchor-locality rejection;
7. zero, one, four, and five-edit expansion boundaries;
8. alpha-renaming and child-order invariance;
9. occupied unit names and alternate descriptor aliases;
10. primitive edit invalid/no-op/duplicate-target rejection;
11. wrong-context abstention and digest corruption rejection;
12. concrete-program recovery with no whole-pair helper;
13. minimum-description ties and complete evidence barriers; and
14. byte-identical standalone and experiment-driver application prefixes.

The exhaustive tiny universe fixes group `0`, definition `1`, request `2`, keys
by ID, scalar alphabet `a,b,c`, and zero, one, or two references targeting the
definition. Definition value and request `from/to` range independently over the
alphabet; reference values do likewise. It contains
`3 * 9 * (1 + 3 + 9) = 351` forests and exactly `351 * 72 = 25,272` schema
applications. Across those forests it also contains 7,020 nonempty valid
concrete programs: for `n` editable nodes each program selects a nonempty target
subset and one of the two non-no-op literals per target. Separate committed
microcases cover zero/two requests, two groups, absent/ambiguous anchors, five
references, malformed wire values, and corrupted evidence. One disagreement
makes the run `invalid`.

Competence has its own nonempirical caps of 26,000 schema applications, 8,000
program applications, and 5,000,000 work. It cannot consume or replenish a
curriculum budget, populate policy artifacts, or expose a panel fixture.
Its exact root is `["transform-competence-root/v2",
[[relativePath,sha256,byteLength,"100644"]...]]`, sorted under the same path
rules as the evidence graph, over every canonical competence input and result
leaf and excluding the root file itself. Empty and self-referential roots are
invalid. The competence report's `rootSHA256` hashes these canonical root bytes.

## Endpoint and classification

A held-out case is correct only if a positive output is byte-equal to the oracle
output or an expected-abstention case terminates with any valid
`abstain/<reason>` result and has no edit or output digest. A
curriculum succeeds only when all eight held-out cases are correct within its
full lifecycle budget. A false application occurs only when an
expected-abstention case terminates `applied`; an evaluated schema that returns
a valid abstention still consumes its application credit but is a correct
rejection, not a false application. Producing an unchanged forest with
`applied` remains false even when its bytes equal the input.

Held-out inputs are presented one at a time in opaque-token byte order. A
policy cannot reorder, skip, batch, or inspect a later input. Refusal, crash,
missing result, wrong output, invalid terminal, or budget exhaustion is an
incorrect case. Skipping or exhausting before any held-out case assigns the
12,000-work penalty described above.

For curriculum `i`, let `N_i` and `B_i` be the binary all-eight success values
for Nous and the frozen primary conventional comparator. Let `F_N` be Nous
false applications divided by 4 times the curriculum count. Let `W_N` and
`W_B` be total work on the four held-out abstention cases.

The primary point is

```text
mean_i(N_i - B_i)
```

with a two-sided 95% paired stratified bootstrap and a paired label-swap
randomization test. Strata are the nine latent semantic families and are frozen
before aliases. All strata must be nonempty. `B` is always `bounded-pbe`.

The observed statistic is `T = sum_i(N_i-B_i)` and the point is the exact
rational `T/n`. For bootstrap replicate `r`, families are visited in enum order
and exactly the observed count for that family is drawn with replacement from
its paired rows, in sample-position order. The replicate is its overall
`sum(d)/n`. Ten thousand rational replicates are sorted by checked cross-
multiplication then replicate ordinal; indices 249 and 9,749 are the interval.

For randomization replicate `r`, curricula are visited in stored ordinal order,
one `Uint64N(2)` is drawn per pair, and `d_i` is multiplied by `+1` for zero and
`-1` for one. It is extreme when `abs(sum(s_i*d_i)) >= abs(T)`. The p-value is
`(1+extreme)/10001`. All sums use checked `int64`; fraction comparisons use
`big.Int` cross-products. Any overflow, missing stratum, wrong draw count, or
zero curriculum count is panel-level `invalid`.

Public inference streams derive PCG seeds from the first 128 bits of
`SHA-256(canonical-json(["part3/transform-schema/v2","statistics",panel,
authority,replicate,purpose]))`, where development authority is integer 841001,
validation is 842001, and purpose is exactly `bootstrap/nous-vs-pbe` or
`randomization/nous-vs-pbe`. Locked inference uses the receipt's already
durable `rootCommitment = SHA-256(privateRoot)` as public, pre-outcome
statistical authority. For replicate `r` and either purpose, its PCG seed pair
is the first 16 bytes of
`SHA-256(canonical-json(["part3/transform-schema/v2","statistics","locked",
rootCommitment,r,purpose]))`, interpreted as two big-endian `uint64` values.
The prepared locked fixture bundle contains exactly one
`statistics/authority.json` leaf with wire
`["transform-statistics-authority/v2","locked",rootCommitment,10000,10000]`;
the fixture root and running receipt bind this leaf before policy event zero.
Independent replay requires the leaf commitment to equal the immutable receipt
commitment, rederives all 20,000 pairs, and recomputes the complete inference.
It rejects any serialized private root, curriculum seed, root-derived HMAC
output, or expanded seed-pair leaf.

A locked result is `valid-positive` only when:

- all mechanical and competence gates pass;
- Nous success is at least 0.80;
- the point advantage is at least 0.10;
- the bootstrap lower endpoint is strictly above zero;
- randomization p is below 0.05;
- `F_N = 0`; and
- `W_N / W_B <= 1.25`.

Otherwise a mechanically valid locked result is `valid-null`. Mechanically
valid development and validation use their level-specific `interim-*` labels
while retaining empirical diagnostics; competence, oracle, replay, or other
mechanical failure is `invalid`, never interim or `valid-null`. Only the locked
panel supplies a final empirical label.

Development estimates locked power by 2,000 fixed outer resamples to 128
curricula with locked counts (the first two canonical families receive 15 and
the final seven receive 14). For outer ordinal `o`, a fresh `panel` PCG from
`["part3/transform-schema/v2","power",841001,o,"panel"]` visits family then
sample position and draws development paired rows with replacement within that
family. It copies the complete paired outcome, false-application counts, and
nonmatching work for each sampled curriculum.

Each synthetic panel runs the same point and gates with 2,000 inner bootstrap
and 2,000 inner randomization replicates. Inner replicate `r` uses the first 128
SHA-256 bits of `["part3/transform-schema/v2","power",841001,o,r,purpose]`,
where purpose is exactly `bootstrap` or `randomization`; the panel stream uses
the shorter tuple above. Bootstrap resamples the synthetic rows within each
family at its locked count; randomization draws one bit per synthetic row in
stored order. Streams never share state.
Bootstrap endpoints are sorted indices 49 and 1,949. Randomization p is
`(1+extreme)/2001`. The outer replicate passes exactly the locked criteria.
Power is `passing/2000` and authorizes progression only at 1,600 or more.
There is no comparator selection inside or outside power.

`W_N` and `W_B` sum held-out-phase charged work for the four abstention cases;
the exhaustion penalty substitutes 12,000 for that curriculum. A zero `W_B`
is mechanically impossible because every presented case requires charged
decode and application events; if observed it is `invalid`. The ratio gate is
checked as `W_N * 4 <= W_B * 5` with checked big-integer products.

The positive region is evidence-free attainable. In family enum order, the
frozen PBE answer ranks are exactly `[4,15,7,31,17,32,19,51,34]`.
Thus combined/local-any, combined/global-equals, and combined/global-any have
ranks 19, 51, and 34. Every earlier non-answer consumes
at least one training application, and an exact answer consumes eight. With
only 40 training credits, ranks 34 and 51 require at least 41 and 58 credits and
must exhaust, while Nous needs eight complete training applications plus eight
held-out applications. Those two locked families contribute 28/128 = 21.875%
paired advantage if both policies otherwise succeed, exceeding the 10% floor
without unequal information or uncharged answer-bearing work. Golden ordering
tests freeze all nine answer ranks before any panel run.

## Closed terminal taxonomy

Levels do not share terminal strings:

| Level | Terminals | Meaning |
| --- | --- | --- |
| schema call | `applied`, `abstain/<reason>`, `invalid-input` | one explicit application |
| held-out case | `correct`, `incorrect` | empirical score; wrong answers are data |
| policy curriculum | `completed`, `no-discovery`, `budget-exhausted` | valid empirical policy outcome |
| execution | `mechanically-valid`, `invalid` | integrity/oracle/replay status |
| development | `interim-power-authorized`, `interim-power-unauthorized`, `invalid` | progression decision |
| validation | `interim-valid`, `invalid` | public generalization/integrity decision |
| locked | `valid-positive`, `valid-null`, `invalid` | final empirical label |
| attempt receipt | `claimed`, `running`, `published`, `invalid` | durable one-shot state |

Mechanical invalidity has precedence over every empirical value and finalizes
an existing receipt `invalid`. `no-discovery` and `budget-exhausted` are valid
data, never mechanical invalidity. Development or validation is not generically
"interim" after a mechanical failure. Locked requires committed validation
whose execution terminal is `mechanically-valid` and panel terminal is
`interim-valid`; a merely present report cannot authorize it. A published
receipt cannot transition, and an invalid receipt permanently burns that panel
attempt for the experiment identity.

## Evidence authority and protected sequence

Before implementation, three reviewers must accept the same plan commit. Before
development, those reviewers must accept the same exact implementation commit
and a canonical review manifest must bind their scopes and SHA-256 hashes for
every protected source, domain, fixture, baseline, oracle, command, plan, and
test input.

The repository guard requires:

- the exact canonical top-level repository and `domains/` path;
- a clean `HEAD`, no merge/rebase operation, and no replacement/shallow Git
  authority;
- sanitized Git configuration and object lookup;
- no `go.work`, workspace override, ignored/untracked compiler input, source,
  domain, plan, manifest, or runtime input;
- every reviewed file to equal its committed regular `100644` Git blob; and
- absent destination report, transcript root, and attempt receipt.

Canonical paths are:

```text
.nous/transform-schema-v2-development-report.json
.nous/transform-schema-v2-development-transcripts/
.nous/transform-schema-v2-validation-receipt.json
.nous/transform-schema-v2-validation-report.json
.nous/transform-schema-v2-validation-transcripts/
.nous/transform-schema-v2-locked-receipt.json
.nous/transform-schema-v2-locked-report.json
.nous/transform-schema-v2-locked-transcripts/
```

The only panel execution exports are
`transformexp.ExecuteDevelopment(repoRoot,domainsDir)`,
`ExecuteValidation(repoRoot,domainsDir)`, and
`ExecuteLocked(repoRoot,domainsDir,unlockToken)`. Panel constructors are
unexported `developmentPanel`, `validationPanel`, and `lockedPanel` in the same
guard package and each has exactly one non-test caller: its corresponding
authorized entry. There is no generic `Generate(panel,seed)`, validation/locked
fixture export, private-root parser, scorer export, or protected replay API.
Source-surface tests count declarations and callers and reject aliases,
function variables, reflection, linkname, test backdoors, and indirect wrappers.

Every evidence leaf uses a closed canonical wire:

- review authority is `["transform-reviews/v2",planCommit,
  implementationCommit,[[scope,status,reviewedCommit]...],
  [[protectedPath,sha256]...]]`; reviews are ordered architecture, semantics,
  experiment and paths by UTF-8 bytes;
- an attempt receipt is
  `["transform-attempt/v2",panel,state,head,implementationCommit,planCommit,
  startedUTC,rootCommitment,fixtureRoot,reportDigest,evidenceGraphDigest]`;
- a fixture root is `["transform-fixture-root/v2",panel,
  [[relativePath,sha256,byteLength,"100644"]...]]`, sorted by path, over policy
  training, delayed held-out inputs, sealed scorer bytes, family assignment,
  queue bytes, and, for locked only, the single statistical-authority leaf;
- an object root is `["transform-objects/v2",
  [[relativeObjectPath,sha256,byteLength,"100644"]...]]`;
- an execution manifest is `["transform-execution/v2",role,[row...]]`, where
  each row is `[policy,curriculumOrdinal,opaqueTaskToken,preManifestSHA256,
  chunkSHA256,rawBytes,gzipBytes,eventCount,objectRootSHA256,vector12,work,applications,
  policyTerminal,schemaSHA256,trainingStoreSHA256,heldoutResultsSHA256]`, ordered
  by policy enum then curriculum ordinal; and
- a report is `["transform-schema-trials/v2",classification,payloadDigest,
  payload]`, where payload is the fixed array `[panel,planCommit,
  implementationCommit,manifest,fixtureRootDigest,primaryManifestDigest,
  auditManifestDigest,evidenceGraphDigest,competence,policyRows,inference,power,
  gates,limitations]`.

`trainingStoreSHA256` names execution state, never a synthetic summary.
`nous-refine` and `no-equality-guard` commit their actual frozen CUE Store;
`positive-lgg` and `concrete-replay` commit the actual acquisition-only CUE
Store captured immediately before Store access is destroyed. `bounded-pbe` and
`random-pbe` are stateless and must use `""`; a `training-store.json` leaf for
either PBE policy is forbidden even on successful completion. Their training
evidence is the committed training envelope, exact transcript and reducer
state, frozen artifact or terminal, and independent oracle reconstruction.

Nested report wires are exact. `competence` is
`["transform-competence/v1",351,25272,7020,microcaseCount,passed,rootSHA256]`.
Each policy row is `[curriculumOrdinal,family,policy,policyTerminal,work,
applications,schemaSHA256,heldoutCorrectBits,falseApplications,
nonmatchingWork]`, ordered ordinal then policy enum. `inference` is
`["transform-inference/v1",pointNumerator,pointDenominator,lowerNumerator,
lowerDenominator,upperNumerator,upperDenominator,randomizationExtreme,
pNumerator,pDenominator,nousSuccesses,pbeSuccesses,falseApplications,
nonmatchingNous,nonmatchingPBE]`. `power` is
`["transform-power/v1",passing,2000,authorized]`. `gates` is the 12-boolean
array `[manifest,competence,dualSemantic,transcriptHashes,conservation,
oracleParity,programsExact,applicationsExact,artifactFrozen,heldoutSealed,
sourceAuthority,evidenceGraph]`. `limitations` is a sorted array of at most 16
unique UTF-8 strings of at most 256 bytes. Empty inapplicable digests are `""`;
all other digests are 64 lowercase hex.

`family` is the integer 0 through 8 in the public family order above.
`heldoutCorrectBits` is exactly two lowercase hexadecimal characters. After
sorting the eight held-out cases by opaque task token, correct case `k` sets
bit `1 << k`; no other bit is permitted. The two execution-manifest fields and
the fixture-root field in a report are 64-lowercase-hex SHA-256 digests, not
paths or embedded manifests. The `manifest` field is the complete canonical
experiment-manifest object above; it may not be replaced by a digest, path, or
null. Empty strings are permitted only for explicitly inapplicable schema and
result digest positions associated with abstention or pre-schema termination,
plus the training-Store position for the two stateless PBE policies defined
above.

Receipt time is exactly UTC `YYYY-MM-DDTHH:MM:SS.NNNNNNNNNZ`. `claimed` has
empty root/fixture/report/graph fields; `running` may add root and fixture
commitments monotonically; `published` requires fixture, report, and graph
digests; `invalid` preserves every known field and may never clear one.
Head/plan/implementation are 40 lowercase hex. Counts are nonnegative canonical
JSON integers within manifest caps; vectors have exactly 12 checked `int64`
values.

Strict decoders reject unknown fields/array lengths, duplicate paths, unsorted
leaves, noncanonical JSON, invalid UTF-8, trailing bytes, negative/overflowing
sizes, wrong modes, and digest mismatches. The evidence graph wire is
`["transform-evidence-graph/v2",panel,[[path,sha256,byteLength,"100644"]...]]`.
Its leaves are sorted by raw UTF-8 path bytes. A path is a nonempty relative
POSIX ASCII path with no leading slash, empty component, `.` component, `..`
component, backslash, or duplicate. The graph contains the review-authority
leaf; every policy, scorer, queue, and family-assignment fixture leaf; the
fixture root; every policy premanifest; both execution manifests; every
transcript chunk; every object leaf and object root; and every competence leaf
and competence root; locked additionally contains its single statistical-
authority leaf. Empty graphs are invalid. It
excludes the graph file itself, report, and receipt. `evidenceGraphDigest` is
the SHA-256 of these canonical graph bytes. The report contains that digest and
its outer payload digest hashes the entire payload; after report persistence
the receipt records the report-file digest and graph digest. Thus no hash cycle
exists. Independent replay
rebuilds the graph and every report field from Git-blob fixture/evidence bytes;
reported gates are comparisons, never trusted booleans.

Development creates public fixtures once, then primary and audit executions
independently decode those bytes. Validation is one-shot and requires a
committed, independently replayed development bundle with authorized power.
Locked execution is one-shot and requires committed validation evidence plus
the exact token `transform-schema/v2:<clean-HEAD>`. A receipt is durably claimed
before fixture generation and remains `invalid` after any failed protected
attempt. Evidence leaves and every parent are regular files/directories; every
prerequisite is read from immutable Git object bytes and checked against the
working file.

Validation claims its receipt before its unexported constructor receives the
first seed. Locked claims its receipt, reads 32 random root bytes, writes only
`SHA-256(root)` to the receipt, and derives curriculum seed `j` as the first
eight bytes of
`HMAC-SHA256(root, canonical-json(["part3/transform-schema/v2",
"locked-curriculum",j]))`, big-endian. Fixture streams remain independently
domain-separated from that seed as specified above. The guard constructs and
serializes all policy/scorer fixtures plus the statistical-authority leaf,
commits their root manifest to the receipt, then overwrites the private root,
derived curriculum seeds, and temporary scorer structs before any policy runs.
Statistical pairs are subsequently rederived from the public root commitment,
not the erased root.
Rejected generator attempts are internal to one derived seed and do not derive
a replacement seed. Failure at any point finalizes the already claimed receipt
`invalid`; partial leaves remain for audit and cannot be retried.

Locked reports reveal no private root or derived seeds. They retain only public
fixture commitments, transcript commitments, inference results, gates, and the
implementation/plan commits. No API other than the guarded command may create,
inspect, score, replay, or summarize protected fixtures before publication.

## Required tests and audits

Implementation review must establish, without executing any protected panel:

- exact manifest reproduction and finite-bound arithmetic;
- pure vocabulary and oracle agreement on exhaustive tiny cases;
- dependency-boundary and forbidden-import scans;
- scoped DSL words absent from the base VM;
- single-fact primitives cannot return a diff, alignment, edit set, binding,
  candidate universe, rank, or held-out fact; complete-schema evaluation returns
  no binding/edit certificate or evidence identifier to policy code, and
  `attach` cannot reveal an operation digest, ordinal, or cross-task link;
- ordinary heuristic provenance for concrete edits, all 12 factor alternatives,
  rejections, stage closures, and the promoted schema;
- transcript golden vectors for match, abstain, corruption, budget exhaustion,
  acquisition/application phase separation, and all three exact maximum
  application vectors;
- per-outcome operation arity tests for node-facts, parent-facts, edit status,
  partial/schema allocation, predicate/application separation, replay, and
  immediate attach, including rejection after an intervening `digest`;
- overflow and every cap boundary at `cap-1`, `cap`, and `cap+1`;
- complete primary/audit replay and report recomputation;
- alternate aliases, occupied names, reordered children, and deletion tests;
- strong baseline equal-information/equal-budget tests;
- all nine PBE answer-rank golden vectors and the ranks 34/51 attainability
  lower bound;
- policy-visible fixture scans and deletion/retargeting tests proving seed,
  ordinal, attempt, family, latent code, and held-out truth are absent;
- canonical profile, program-batch, competence-root, ASCII-token, and evidence-
  graph preimage tests;
- exact sole-call-path tests for validation and locked constructors;
- Git environment, ignored-input, symlink, non-blob, dirty-tree, receipt, and
  unlock-token adversarial tests; and
- source scans proving no fixture seed, latent schema, answer table, or panel
  constructor crosses into production.

No development, validation, or locked command may run during implementation
review. After acceptance, development runs once. If its power gate is
unauthorized, the result is documented and the vocabulary stops. If authorized,
development evidence is committed before validation. Validation evidence is
then committed before locked execution. Every valid or invalid terminal is
documented, committed, and pushed without retuning.

## Delivery sequence

1. stabilize and commit this plan;
2. implement pure term/edit/schema semantics and independent oracle first;
3. add the scoped DSL adapter and ordinary heuristic domain;
4. add fixture generation, baselines, metering, transcripts, and competence;
5. add statistics, report reconstruction, and protected evidence guards;
6. run only safe tests and obtain three exact-commit implementation reviews;
7. commit the review authority and run development once;
8. follow the frozen power gate, documenting either stop or progression;
9. commit and push every evidence boundary; and
10. update the Part 3 capability matrix without borrowing success from another
    vocabulary.

## Research relationship

Kutsia, Levy, and Villaret's
[anti-unification for unranked terms and hedges](https://doi.org/10.1007/s10817-013-9285-6)
motivates the explicit generalized-role representation and the conventional
LGG comparator. Gulwani's
[FlashFill programming-by-example work](https://www.microsoft.com/en-us/research/publication/automating-string-processing-spreadsheets-using-input-output-examples/)
motivates a finite transformation DSL, ambiguity controls, and exact held-out
execution. Version-space algebra provides the comparison point for the bounded
PBE baseline.

Nous's distinctive claim is narrower: the schema, refinements, counterexamples,
tie set, and concrete expansion remain first-class inspectable store artifacts,
and the learned schema must causally execute an exact held-out transformation.
