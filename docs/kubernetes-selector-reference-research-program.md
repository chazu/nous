# Kubernetes Selector and Reference Repair Research Program

Status: revised after adversarial review; implementation contract

## Decision and questions

Build `kuberepair`, a synthetic, bounded Kubernetes vocabulary for searching
typed atomic repairs to relations among a Deployment, its Pod template, a
Service, declared container ports, and HTTP readiness probes. Keep the program
entirely inside Nous. Mu, PUDL, live clusters, `kubectl`, arbitrary YAML,
admission, defaulting, rollout behavior, EndpointSlices, and networking are out
of scope.

The program asks two separate questions:

1. Can Nous reuse its existing descriptor-driven enumerate/evaluate/select
   protocol to discover a minimum safe plan made from bound atomic edits?
2. Does structural contextual credit earned by selected edits lower work to the
   first minimum safe plan on later tasks, compared with both blind ordering and
   a conventional constraint-guided repair baseline?

Phase A is exhaustive and cannot show that credit improved its own search.
Phase B is an external bounded-ordering trial over engine-produced credit; it
does not show agenda-level pruning or scalable search.

## Claim boundary and outcomes

Phase A can establish unmodified search-protocol reuse and concrete bounded
patch discovery. It cannot establish repair-rule invention: the typed edit
grammar is human supplied. Phase B can establish only that learned structural
edit preferences are useful on this finite distribution. Exact-sequence reuse,
component recomposition, and cross-role reuse are reported separately; exact
reuse cannot establish structural transfer.

Each phase reports one of:

- `valid-positive`: all integrity gates and the preregistered usefulness gate
  pass;
- `valid-null`: integrity passes but usefulness does not;
- `invalid`: leakage, oracle, determinism, unequal-information, or accounting
  integrity fails.

Evaluating an unsafe candidate and rejecting it is a valid negative observation.
An unsafe candidate counted as success, bypassed safety checks, or oracle
disagreement makes a run invalid.

## Information boundary

Every task has two canonical JSON documents.

The public `kubernetes-bundle/v1` contains the modeled Kubernetes objects and
declares which otherwise-writable fields are operator-protected. Candidate
generation and every ranking policy see only this document.

The private `kubernetes-intent/v1` contains:

- the intended selected Pod-role set;
- the intended Service backend protocol and numeric port;
- the intended numeric readiness endpoint for each container role; and
- a digest of every protected public projection, including presence and value.

Only the terminal evaluator and independent oracle see private intent. The DSL
comparator returns one Boolean. It exposes no target value, expected patch,
intended role, failure name, or predicate bitset. Tests mechanically scan CUE
heuristics and ranking inputs for private fields and target literals.

The task unit contains only a uniformly random 256-bit intent handle. Intent
bytes live in a driver-owned table reachable solely through the terminal
evaluator capability; they never enter the Nous store. The handle is unique per
task and reveals no equality across tasks. Heuristics are source-audited to
ensure they only pass it to the Boolean comparator. Held-out and locked intent
is never used by candidate generation, credit lookup, or policy ordering.

## Pinned semantic profile

A bundle represents exactly one namespace containing:

- one Deployment with a nonempty equality-only `matchLabels` selector, a Pod
  template, one or two containers, at most four uniquely named TCP ports per
  container, and optional HTTP readiness probes;
- one Service with a nonempty equality-only selector and one TCP port; and
- zero to two read-only distractor Pods.

Resource identities, container identities, and collection membership are
unique. Ports are in `1..65535`. Named and numeric references use an explicit
`IntOrString` tagged union. Unknown fields, duplicate identities, invalid
values, excess bounds, and noncanonical encodings are rejected. JSON object
order is immaterial; named collections are canonicalized by identity.

Typed paths use resource, field, map-key, keyed-list-member, and scalar-leaf
components rather than flattened strings or numeric list indexes. Examples:

```text
Deployment[d].spec.selector.matchLabels[key]
Deployment[d].spec.template.metadata.labels[key]
Deployment[d].spec.template.spec.containers[name].ports[name].containerPort
Deployment[d].spec.template.spec.containers[name].readinessProbe.httpGet.port
Service[s].spec.selector[key]
Service[s].spec.ports[name].targetPort
Pod[p].metadata.labels[key]
```

The evaluator implements exactly:

- the Deployment selector is an immutable conjunction and must match template
  labels;
- the virtual Pod inherits the template labels, containers, ports, and probes;
- a Service selects each namespace-local virtual or concrete Pod whose labels
  contain every selector pair;
- missing Service `targetPort` denotes the Service `port`;
- integer `targetPort` denotes that numeric backend;
- named `targetPort` must resolve to exactly one pod-wide named port on every
  selected Pod;
- an integer HTTP readiness port denotes that number;
- a named HTTP readiness port must resolve uniquely within its own container;
- final selected roles, backend number, readiness endpoint numbers, and
  protected digest must equal private intent.

No relationship between `ServicePort.name` and `targetPort` is inferred.
Resolved-but-wrong references fail private intent.

## Atomic edit grammar

Candidate generation walks only public typed nodes and materializes every legal
bound edit. An edit contains its complete destination path and, where needed,
complete source path. It contains no private value. The Go vocabulary may
enumerate public nodes, validate one edit, and apply one edit; it may not rank
edits, compose plans, inspect intent, or select a repair.

The grammar is:

1. `PutLabel(dst,key,ValueAt(src,key))`, where `dst` is template labels or the
   Service selector and `src` is a selector/label map containing `key`;
2. `RemoveLabel(Service.selector,key)`;
3. `SetPortRefByName(refPath,declaredPortPath)`;
4. `SetPortRefByNumber(refPath,declaredPortPath)`; and
5. `UnsetServiceTargetPort(Service.targetPort)`.

Reference destinations are the Service `targetPort` and individual HTTP
readiness port leaves. Reference sources are explicit declared container-port
nodes. A readiness edit changes exactly one named container's probe. No
operation changes more than one semantic leaf.

Deployment selectors, concrete Pods, identities, Service public port,
container-port declarations, readiness presence/path/scheme, and any
fixture-protected Service/template leaf are illegal destinations. An edit that
does not strictly change canonical public state is ineligible. The experiment
envelope carries a write-history set outside the modeled Kubernetes objects.
`kube-apply-edit` returns semantic nil when a prior step wrote the same
destination, so the unchanged repetition-producing enumerator can generate but
cannot validate duplicate-destination plans. History is ignored by terminal
Kubernetes semantics but included in evidence replay.

Candidate identity is canonical typed edit JSON. Plan identity is an ordered
list of edit identities. The trial charges all syntactic plans even when two
plans reach the same state, reports semantic duplicate counts, and accepts all
co-minimal correct plans as ties. The empty plan is explicit.

Terminal precedence is:

1. malformed input;
2. already correct (empty plan is the unique minimum);
3. minimum safe solution set, possibly co-minimal;
4. no solution within the grammar and length bound;
5. budget exhausted when a policy lacks a charged complete proof.

## Phase A: Nous artifact search

The current bounded-program engine is reused only after a task-specific setup
step materializes the complete legal edit set as unary primitives. V1 fixture
generation is constrained to one selector key, one template-label source, one
Service-selector source, two declared ports, one Service target, and one
readiness target. After removal of no-ops and protected destinations this
formula yields five through eight edits; fixtures outside that range are
rejected before intent assignment. No edit is truncated or selected by a
policy. Primitive definitions contain the canonical edit and call
`kube-apply-edit`; they do not contain an expected value. The descriptor's
existing maximum length is three, so a task has at most 584 nonempty syntactic
plans. The empty-plan terminal is handled before synthesis.

The first three CUE heuristics are extracted verbatim from
`domains/tinystack/heuristics.cue`:

- `H-EnumerateBoundedPrograms`;
- `H-EvaluateBoundedProgram`;
- `H-SelectShortestExactProgram`.

A source test compares the exact UTF-8 source slices from each heuristic's
opening object brace through its closing brace, including prose. The fourth
tiny-stack heuristic is stack-specific; `kuberepair` uses the same
simplification evidence fields and replay rules but emits a Kubernetes-specific
conjecture kind and statement. It is not counted as generic reuse.

The seed task contains eight eligible edits and therefore exactly 584 nonempty
plans. The unique minimum safe plan has two or three atomic edits. Its uniqueness
must arise from final intent and typed legality, not an artificial intermediate
validity precondition; intermediate states need only be structurally valid.

An independent `internal/kuberepairoracle` package reimplements decoding,
atomic edits, final semantics, and exhaustive plan enumeration. It must not
import `internal/vocab/kuberepair`, DSL, engine, seed, domain, credit, or trial
packages. A dependency test enforces the boundary.

Phase-A panels are fixed before implementation:

- seed: 1 task used for domain-pack integration;
- development: seeds `731001..731015` (15 tasks);
- validation: seeds `732001..732031` (31 tasks);
- locked: seeds `733001..733063` (63 tasks), not opened until implementation,
  tests, and thresholds are committed.

The generator assigns exact target values only after public bundles and edit
sets are frozen, then rejects solely for oracle terminal class and minimum-plan
multiplicity—not for any tested policy's performance. Development may tune code.
Validation may reject the implementation once. The locked panel is one-shot;
post-observation changes require `v2` and preserve the v1 result.

Phase-A integrity requires:

- descriptor and exact candidate-count agreement;
- complete replayable candidate evidence;
- exact minimum-set agreement with the independent oracle;
- no private datum in candidates or ranking inputs;
- all protected projections preserved by selected outputs;
- explicit unique, co-minimal, already-correct, no-solution, malformed, and
  budget-exhausted controls;
- alpha-renaming, reordering, irrelevant-node, collision, primitive-deletion,
  forged-evidence, and deterministic-store controls; and
- mechanically valid scalar and contextual credit provenance from one unique
  selected training plan before Phase B.

Phase A is positive only if every solvable validation/locked task returns its
complete minimum set, every null terminal is correct, no unsafe plan is
selected, and at least one minimum plan contains two distinct atomic edit kinds.

## Structural credit

Concrete edit identity includes bound names and values and remains the unique
`semanticSequence` and ordinary `creditDecision` used by the unchanged
synthesizer's completeness checks. It is deliberately not used as a transfer
key. Each edit also has two separately validated structural keys. The component
feature key contains:

- edit kind;
- destination resource/field role;
- source resource/field role, when present; and
- reference representation (`name`, `number`, or `default`).

The relation key contains only the edit family (`label-copy`, `label-remove`,
or `reference-to-declared-port`) and reference representation. It deliberately
omits source and destination field roles and is used only by the exploratory
cross-role panel.

Both keys exclude resource names, container names, label keys/values, port
names/numbers, list positions, fixture IDs, and private intent. Stable feature
and relation units exist for every key.

The extension does not change the CUE heuristic text. Each concrete primitive
may declare `creditFeatureSubject`, `creditFeatureKey`,
`creditRelationSubject`, and `creditRelationKey`. When an evaluated synthesized
program grows from worth 500 to 800, existing scalar credit still gives each
ordinary creditor its clamped 150-point reward. In addition, for every
primitive occurrence whose declaration validates, the engine:

- increases the stable feature unit's scalar worth by 150;
- upserts 150 to `{context, feature-subject, component}`;
- upserts 150 to `{context, feature-subject, step-N}`; and
- upserts 150 to `{context, relation-subject, relation}`.

Repeated occurrences are credited per occurrence; position-independent records
aggregate them. The engine also upserts 300 to a structural decision tuple
whose subject is SHA-256 over the synthesis method and ordered component feature
keys. It continues to emit the existing concrete decision record. Structural
declarations are ignored atomically unless subjects exist, keys validate, and
the feature sequence exactly corresponds to the program's concrete component
creditors. Replay tests recompute every derived tuple and reward from source
evidence. This is an explicit, generic credit-mechanics extension; current
unit-name component credit is not claimed to transfer.

Context is exactly `kubernetes-selector-reference/atomic-edits/v1`. Multiple
concrete plans may share one structural decision tuple. This is intentionally a
learned preference over a rule shape, not proof that Nous invented the shape.

Phase B requires one unique phase-A training winner and the exact preregistered
credit record set. If that precondition fails, Phase B is invalid rather than
silently pooling tied answers.

## Phase B: matched ordering trial

All policies receive the same frozen public task, candidate edits, syntactic
plans, base permutation, length bound, terminal evaluator, and total-work
ceiling. Oracle terminal labels and minimum sets are unavailable to ranking.

Policies are:

- `contextual`: structural-sequence reward, then position-independent component
  reward, then position reward, then exploratory relation reward, then the
  shared base order;
- `no-credit`: the shared uniform base permutation;
- `wrong-context`: the contextual algorithm under an empty context; it must be
  byte-identical to no-credit;
- `scalar`: ordinary feature-unit worth, then shared base order;
- `reset`: seed worths, which must be byte-identical to no-credit;
- `constraint`: the frozen public-constraint best-first baseline below;
- `exhaustive`: canonical complete enumeration used only for oracle/work
  reference.

The complete eligible plan list, including duplicate-destination rejection, is
constructed once and byte-shared by every policy. The base permutation is
constructed once from `seed xor 0x51f15e`; no policy-specific RNG exists.

The constraint baseline independently applies every plan to public state and
computes this fixed public vector: malformed-state bit, Deployment
selector/template mismatch count, empty-Service-selector bit, unresolved
Service-reference count, unresolved-readiness-reference count, changed-leaf
count, and plan length. It stable-sorts lexicographically ascending by the
vector, then by base position. It receives no private-intent result during
ranking, does no adaptive reordering, and traverses the resulting list under the
same terminal stop rule. Thus it is a reproducible conventional repair ordering,
not a second oracle.

The primary panel is component recomposition only, and its primary comparator
is `contextual` versus `constraint`. Contextual versus no-credit is the secondary
causal check. Exact reuse is descriptive and cannot make the result positive.

Work is a wall-clock-independent integer ledger. Each typed-node visit, edit
materialization, eligibility check, normalization, hash, dedup lookup, credit
lookup, score comparison, stable-merge comparison, primitive application,
changed-leaf write, reference candidate examined, public predicate check,
private predicate check, and emitted evidence field costs one unit. Cache lookup
and cache insertion each cost one; a hit does not erase the lookup charge.
Permutation construction costs one per exchanged element. Evaluation never
short-circuits: it traverses every modeled predicate and charges the full
result. Stable merge sort is the only sort and charges exactly one per comparator
invocation. The ledger is observable only after a policy terminates.

Every policy has the same `393216` total-work ceiling. Exhaustion has capped
loss `393217`. Candidate generation and the common plan list are charged
identically to every policy; policy-specific ranking is then charged according
to the tariff.

The analytic eight-edit worst case is fixed as follows. Common generation costs
at most 512 public-node visits, 40 edit construction events, 584 plan
materializations, 1,752 edit references, 1,752 destination checks, 584 each of
normalizations, hashes, and dedup lookups, and 583 permutation exchanges: fewer
than 7,500 units. Stable merge sort performs at most
`584 * ceil(log2(584)) = 5,840` comparator invocations total, not per plan.
Contextual ranking costs at most ten lookups and ten score operations per plan
plus the merge-comparator invocation and five charged integer comparisons per
merge comparator: 46,720 units.
Constraint ranking costs at most 96 public-application units, 64 public-predicate
units, and seven vector-normalization units per plan, plus seven charged integer
comparisons and the invocation itself per merge comparator: 144,248 units. A terminal attempt, including
application, full public/private traversal, cache events, and evidence fields,
is capped structurally at 304 units; all 584 attempts cost 177,536. Therefore
the contextual complete-proof bound is 231,756 and the constraint bound is
329,284, leaving at least 63,932 units of headroom under the common ceiling.
Tests assert both the category bounds and total bound on the maximum-size
fixture before any protected run.

Training discovery cost is amortized over the fixed 32-task primary component
panel and is included in the primary contextual loss, not merely reported:

```text
primary contextual loss = phase-A training work / 32
                            + mean phase-B contextual work
```

The v1 panels, generator, and RNG streams are:

- development: seed `741001`, 24 tasks;
- validation: seed `742001`, 48 tasks;
- locked: seed `743001`, 96 tasks;
- fixture RNG: `seed`; base-permutation RNG: `seed xor 0x51f15e`;
  inference RNG: `seed xor 0x9e3779b9`.

The locked panel contains 32 exact-reuse, 32 component-recomposition, 8
cross-role, 8 unrelated, 4 co-minimal, 4 already-correct, and 8 no-solution
tasks. Every minimum plan in a component task consists entirely of credited
component-feature keys, while no minimum ordered feature sequence has structural
decision credit. Sixteen are strict-subset cases at sequence edit distance one
from training; sixteen recombine two or three credited features at distance at
least two. Results and gates are reported for both strata. Cross-role tasks use
credited relation keys but uncredited component keys and are explicitly
exploratory, not evidence for the v1 transfer claim. Unrelated tasks contain no
credited component or relation key in any minimum plan and are descriptive
negative-transfer controls. Observable object, edit, plan, and minimum-set
counts are matched within each paired solvable comparison. Null cohorts are
never included in positive-rate aggregation.

For every paired task, loss is total work to the first co-minimal safe plan, or
`393217` on exhaustion. Success rate and unsafe outcomes are reported separately.
Paired bootstrap percentile intervals use 10,000 fixed inference replicates;
the primary effect is `(contextual loss - constraint loss) / constraint loss`.
Safety noninferiority uses a two-percentage-point margin. No multiplicity claim
is made beyond the single primary comparator; secondary comparisons are labeled
descriptive.

The locked run is positive only if:

- contextual safe-solution rate is at least 90% in each component-distance
  stratum;
- on the 32 component tasks, its safety-rate difference from constraint has a
  95% interval lower bound no worse than `-0.02`;
- on those same tasks, primary contextual loss including `training/32` is at
  least 15% below constraint and the 95% paired-bootstrap interval excludes
  zero;
- component contextual loss is at least 10% below no-credit with an interval
  excluding zero in each distance stratum;
- no unsafe or protected-intent-violating plan is counted as success;
- wrong-context equals no-credit and reset equals no-credit byte-for-byte.

Exact, cross-role, unrelated, co-minimal, already-correct, and no-solution
results are descriptive integrity and limitation evidence. No positive or
negative population claim is made from their small cohort sizes.

A development power/feasibility report must be committed before the locked run.
If the frozen 96-task panel cannot resolve these margins, the locked result is
valid-null rather than changing the sample or threshold.

## Implementation topology

1. `internal/vocab/kuberepair`: strict public/private models, typed paths,
   public edit enumeration, atomic application, feature identity, and Boolean
   terminal evaluation.
2. `internal/kuberepairoracle`: source-independent model and exhaustive oracle.
3. `internal/dsl/builtins_kuberepair.go`: scoped validation, enumeration,
   application, and Boolean comparator words.
4. `internal/vocab/programsynth` and the generic synthesis adapter: optional
   structural feature-credit subjects and position-independent component credit.
5. `domains/kuberepair`: task descriptor, one seed task, bound primitive edits,
   three verbatim generic heuristics, and the Kubernetes simplification
   heuristic.
6. `internal/seed`: integration, source-equality, leakage, evidence, oracle,
   adversarial, credit, and determinism tests.
7. `internal/kuberepairexp` and `nous kuberepair-trials`: panels, baselines,
   work ledger, inference, and deterministic reports.
8. `docs/kubernetes-selector-reference-results.md`: commands, immutable run
   identities, outcome classification, claims, and limitations.

## Stop conditions

Record an invalid run rather than patching around evidence if a private target
reaches generation or ranking, an expected patch is encoded in a primitive, an
oracle disagrees, policies receive unequal information or ceilings, source
reuse differs, unsafe behavior bypasses validation, or a protected panel is
redesigned after observation. Any semantic, generator, threshold, or panel
change after a result creates a new version and preserves v1.
