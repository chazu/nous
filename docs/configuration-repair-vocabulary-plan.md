# Configuration-repair vocabulary plan

## Research question

Can Nous construct a reusable repair plan for small typed configurations,
select it by constraint satisfaction and preservation of declared operator
intent, and retain evidence distinguishing genuine repairs from cheaper
constraint evasion?

The protocol vocabulary discovered behavioral relations over supplied
transforms. The rewrite vocabulary constructed an executable ordered program.
This experiment constructs an executable unordered repair plan in a domain
with interacting constraints, several nonconforming inputs, and tempting valid but
intent-destroying alternatives.

## Review status

Accepted after three adversarial architecture and experiment review rounds.

## Bounded claim

Nous receives six primitive field-setting repairs, two related schemas, and
four well-formed but schema-nonconforming training configurations. A generic heuristic constructs every
non-empty subset of one through three distinct repairs: exactly
`C(6,1) + C(6,2) + C(6,3) = 41` candidates. A second heuristic applies every
candidate to every example and promotes only a plan that:

- produces a well-formed configuration satisfying its example's schema;
- preserves every schema field marked as protected intent;
- succeeds on the complete training corpus of at least four examples; and
- contains no invalid or failed application.

The seed corpus has one qualifying subset. It sets the service port to 443,
raises replicas to two, and disables public administration. The component
identities are neutral and never named by the heuristic.

Success demonstrates bounded repair-plan construction, behavioral selection,
intent-aware rejection, linked evidence, held-out execution, and ordinary plus
contextual credit assignment. It does not demonstrate arbitrary configuration
languages, optimal planning beyond the enumerated depth, learned constraint
inference, general DevOps competence, agenda-level credit-guided pruning, or
safe authority to modify real systems.

## Representation and semantics

A configuration is a `[]string`, represented in the DSL as a list of strings,
containing unique `key=value` records. Keys match
`[A-Za-z][A-Za-z0-9_-]*`; values match `[A-Za-z0-9_.-]+`. Keys are at most 64
bytes, values 128, records 256, total encoding 4096, and record count 32.
Whitespace, duplicates, empty keys or values, and other forms are invalid.
Canonical form sorts records by key.

A schema is a list containing these records:

```text
field:<key>:string
field:<key>:bool
field:<key>:int:<min>:<max>
required:<key>
protected:<key>
eq-if:<guard-key>=<guard-value>,<target-key>=<target-value>
min-if:<guard-key>=<guard-value>,<target-key>=<minimum-int>
```

Schemas are also `[]string`, bounded to 64 records of 256 bytes and 8192 bytes
total. Parsing rejects duplicate field, required, protected, or semantically
duplicate constraint declarations; unknown references; malformed bounds;
type-invalid or out-of-range literals; and required/protected keys without
fields. Booleans are exactly `true` or `false`. Integers use canonical base-ten
`0` or `-?[1-9][0-9]*`; plus signs and leading zeroes are rejected rather than
normalized. Configurations reject unknown fields, missing required fields,
type/range violations, and failed constraints. Guards and targets compare
typed canonical values. Schema validity is structural and typed; a well-formed
but globally unsatisfiable schema is allowed and simply has no satisfying
configuration.

A repair is a lexically valid `(key, value)` assignment. Applying it replaces or adds exactly
that key and returns canonical configuration data. It does not decide whether
the result satisfies a schema. A repair plan is a canonically identity-sorted
set of one to three primitive repair units, executed in that order. Distinct
components may not target the same key in the accepted seed vocabulary;
runtime validation rejects such plans rather than relying on last-write wins.
Plans are semantic sets because distinct-key assignments commute. Execution
uses deterministic primitive-identity order, and permutation tests must prove
the same output. Component names affect allocation and creditor subjects, not
semantic decision identity.

Intent preservation means every `protected` key has the same presence and
typed value before and after repair; type-invalid protected values make the
comparison semantically invalid. Changed-key count is the symmetric count
of keys whose presence or value differs. Applying an accepted plan twice must
be idempotent and produce zero changes on the second application.

Pure semantics live in `internal/vocab/configrepair`, independent of units,
the DSL, and the engine. They expose parsing, canonicalization, satisfaction,
repair application, protected-field comparison, changed keys, plan validation,
and a stable semantic decision key. Tests include malformed encodings,
boundary integers, conditional constraints, canonicalization, idempotence,
and differential checks against a small independent test evaluator.

## Vocabulary-scoped words

`ConfigurationRepairVocabulary` selects `dslExtension: "configrepair"` and
adds:

- `config-valid?` — total representation classifier for configuration data;
- `config-schema-valid?` — total representation classifier for schema data;
- `config-canonicalize` — canonical configuration or semantic nil;
- `config-satisfies?` — `(configuration, schema)` predicate or nil if either
  representation is malformed;
- `config-set` — apply one field assignment or return nil;
- `config-preserves-protected?` — compare two configurations under a schema;
- `config-changed-count` — number of changed keys or nil;
- `config-repair-valid?` — lexical classifier for `(key, value)`;
- `config-plan-valid?` — reject empty, oversized, duplicate, or same-key plans;
- `config-name-less?` — deterministic component-identity ordering;
- `config-plan-name` and `config-artifact-name` — boundary-preserving,
  collision-safe allocated unit identities; and
- `config-plan-defn` — self-contained executable definition generated from
  authoritative structured assignments; and
- `config-decision-key` — stable semantic identity independent of allocated
  candidate names.

Exact stack contracts are:

```text
(configuration -- bool)                         config-valid?
(schema -- bool)                                config-schema-valid?
(configuration -- configuration|nil)            config-canonicalize
(configuration schema -- bool|nil)              config-satisfies?
(configuration key value -- configuration|nil)  config-set
(before after schema -- bool|nil)                config-preserves-protected?
(before after -- int|nil)                        config-changed-count
(key value -- bool)                              config-repair-valid?
(component-names -- bool|nil)                    config-plan-valid?
(left-name right-name -- bool|nil)               config-name-less?
(component-names -- string|nil)                  config-plan-name
(component-names -- string|nil)                  config-plan-defn
(kind program example -- string|nil)             config-artifact-name
(component-names -- string|nil)                  config-decision-key
```

The final six words strictly require string/list ValueKinds and resolve
primitive names through the VM store. They never coerce integers, booleans,
nils, or nested malformed lists with `AsString`. Primitive units carry string
`repairKey`, string `repairValue`, and executable `defn`. A composite receives
its configuration as the unary `apply-op` input and concatenates bodies of the
exact form `"key" "value" config-set`.

The three validity classifiers (`config-valid?`, `config-schema-valid?`, and
`config-repair-valid?`) return false on malformed representations and record
rarity only when invoked through predicate operation units. Other semantic and
allocation words return nil. The words remain absent from
unselected vocabulary VMs, while child VMs inherit them.

## Ontology and seed corpus

The pack defines `ConfigurationRepairVocabulary`, `ConfigurationSchema`,
`Configuration`, `ConfigurationRepairExample`, `PrimitiveConfigurationRepair`,
`CompositeConfigurationRepair`, `ConfigurationRepairResult`,
`ConfigurationRepairObservation`, `ConfigurationRepairEvidence`, and
`ConfigurationRepairSchema`.

`ServiceSchemaV1` declares six fields:

```text
environment:string    protected, required
tls:bool              protected, required
service_port:int      required, range 1..65535
replicas:int          required, range 0..10
admin_public:bool     required
redirect_http:bool    required

tls=true              -> service_port=443
environment=production -> replicas>=2
environment=production -> admin_public=false
```

`GatewaySchemaV1` adds a required `route_count:int:0:100` field but retains the
same obligations. Examples use `configuration: [...string]` and
`schema: <ConfigurationSchema unit name>`. The literal corpus is:

```text
ServiceExampleA / ServiceSchemaV1
  environment=production tls=true service_port=80 replicas=0
  admin_public=true redirect_http=false

ServiceExampleB / ServiceSchemaV1
  environment=production tls=true service_port=443 replicas=0
  admin_public=true redirect_http=false

GatewayExampleC / GatewaySchemaV1
  environment=production tls=true service_port=80 replicas=2
  admin_public=true redirect_http=false route_count=12

GatewayExampleD / GatewaySchemaV1
  environment=production tls=true service_port=80 replicas=2
  admin_public=false redirect_http=false route_count=5
```

All three legitimate repairs are necessary across the corpus. Example B needs
two production repairs; changing only `environment=development` is a strictly
shorter validity-only shortcut. Example D needs only the port repair; changing
only `tls=false` is an equal-cost validity shortcut. The two-repair plan
`{tls=false, environment=development}` satisfies every schema constraint on
all four examples but changes both protected fields and has zero overall
support.

The six neutral primitive repairs are semantically:

```text
service_port = 443
replicas = 2
admin_public = false
redirect_http = true       # irrelevant decoy
tls = false                # validity shortcut that violates protected intent
environment = development # validity shortcut that violates protected intent
```

The preregistered independent-oracle summary is:

| Plan class | Constraint support | Intent support | Overall support |
|---|---:|---:|---:|
| target three repairs | 4 | 4 | 4 |
| plans containing either protected repair | 0..4 | 0 | 0 |
| plans containing neither protected repair except target | 0..2 | 4 | 0..2 |

Exactly one of 41 plans has overall support four. Several partial plans have
nonzero per-example support: the port singleton repairs Example D, the two
production repairs fix Example B, and port plus admin repair Example C.
Several protected plans have constraint support four, including
`{tls=false, environment=development}`,
`{service_port=443, environment=development}`, and
`{tls=false, replicas=2, admin_public=false}`; every one has intent support
zero. Thus no singleton, pair, or other triple qualifies overall, while a
validity-only oracle would promote multiple protected shortcuts. The
independent oracle computes every exact per-plan count rather than inferring
counts from these classes.

## Synthesis lane

`H-ComposeConfigurationRepairPlans` is generic over
`PrimitiveConfigurationRepair`. Each primitive is focused once. It constructs:

- its singleton plan;
- every pair for which its identity is lexically first; and
- every triple for which its identity is lexically first and the remaining
  identities are increasing.

This produces 41 candidates without permutation duplicates only when the
store contains exactly six lexically valid, distinct-key primitive repairs;
invalid primitives are excluded before enumeration. Each candidate
stores canonical components, concatenated executable definition, domain,
range, arity, synthesis method, and a single evaluation task. Candidate and
artifact allocation never overwrite occupied names.

Composite definitions are generated directly from each component's
authoritative structured `repairKey` and `repairValue`, never copied from the
primitive `defn`. Primitive definitions are independently checked for parity.

Plan allocation sorts primitive identities, encodes every boundary with raw
URL-safe base64, and prefixes the arity. Artifact allocation encodes kind,
allocated program identity, and example identity the same way. An occupied
base uses the first free deterministic `-collision-N` suffix; structured slots
remain authoritative and no occupied unit is reused or overwritten.

Creditors contain the synthesis heuristic followed by each component.
`creditContext` is `configuration/repair-subsets-up-to-3/v1`;
`creditDecision` is `sha256:v1:<hex>` over canonical JSON containing the
synthesis-method version plus unordered, sorted raw `(repairKey, repairValue)`
assignments. Schema validation later gives those lexical values their types;
the credit identity deliberately claims assignment identity, not cross-schema
typed equivalence. The digest is invariant under aliases and enumeration order
and fits the kernel's 512-byte credit bound. `config-plan-valid?` also rejects
component unit identities longer than 512 bytes, and tests require the entire
declaration to pass `credit.ValidDeclaration`. Roles are
`synthesis` followed by one `repair` role per component.

## Evaluation lane

`H-EvaluateConfigurationRepairPlans` evaluates an unevaluated candidate over
every `ConfigurationRepairExample`. For each example it:

1. resolves the schema unit and validates both representations;
2. applies the composite plan;
3. checks schema satisfaction and protected-intent preservation;
4. records changed-key count and second-application idempotence;
5. creates a data-bearing result for every well-formed application before
   constraint or intent assessment;
6. creates an observation for every example, including explicit malformed,
   constraint-failure, intent-failure, and non-idempotence statuses; and
7. records a direct application linked to the result and example.

Each observation records orthogonal booleans `applicationValid`,
`schemaSatisfied`, `intentPreserved`, `idempotent`, and `outcome` as their
conjunction. Failure counts are independent and may overlap. A display status
uses deterministic precedence: invalid application, non-idempotent,
constraint-and-intent failure, constraint failure, intent failure, success.
Direct applications are recorded exactly when a result exists.

One evidence unit aggregates all examples, results, observations, support,
constraint failures, intent failures, invalid applications, idempotence
failures, changed-key total, and `exhaustive-training-corpus/v1` as the
comparison method. Every candidate is retained. A candidate rises from worth
500 to 800 only with at least four examples, complete evaluation, full support,
and zero failures of every class; all others end at 300.

Only a fully supported candidate creates a guarded, idempotently allocated
`ConfigurationRepairSchema` and attributed conjecture. It explicitly sets
`worth`, `creationWorth`, and `lastRewardedWorth` to 800 so it cannot produce a
spurious reward. Exactly one schema and one attributed conjecture may exist;
both link to the target evidence, and repeated evaluation or occupied-name
allocation cannot create extras. Promotion does not use
candidate or component names, expected target values, or enumeration position.
Evidence and evaluation guards make repeated focus idempotent.

## Credit observation

The synthesis heuristic starts at 750 and all primitives at 600. The unique
target's worth growth from 500 to 800 must reward the synthesis heuristic and
its three components by the kernel's ordinary actual `+150`
increase. It also creates one decision contextual-credit record of 300 and
four role records of 150. Decoy scalar worth and contextual totals remain
unchanged, the target's `lastRewardedWorth` becomes 800, and a later reward
interval changes no totals.

The synthesis search remains exhaustive. This observes attribution; it does
not claim that credit improved the search that earned it.

## Independent controls and blinding

Tests independently parse, enumerate, and evaluate all 41 subsets without
calling `internal/vocab/configrepair`, configuration DSL words, synthesized
definitions, or production result/evidence units. Fixed fixtures drive
expected outputs. The
oracle must agree on every candidate's support, failure classes, changed-key
total, final worth, and promotion in both seed and alternate trials.

An opaque-alias trial deletes every canonical primitive and recreates the same
repairs under shuffled adversarial identities. Occupied candidate and artifact
base names must survive unchanged. The unique semantic repair set must still
be promoted and execute after primitive deletion, proving that the composite
definition is self-contained.

A runtime alternate trial removes all seed schemas, examples, and primitives.
It installs base and extended worker schemas with:

```text
stage:string protected; encryption:bool protected
ingress_port:int:1:65535; workers:int:0:20; debug:bool
metrics:bool; queue_depth:int:0:100 in the extended schema

encryption=true -> ingress_port=8443
stage=live -> workers>=3
stage=live -> debug=false
```

Its four literal inputs use different keys and values:

```text
A: stage=live encryption=true ingress_port=8080 workers=0 debug=true metrics=false
B: stage=live encryption=true ingress_port=8443 workers=0 debug=true metrics=false
C: stage=live encryption=true ingress_port=8080 workers=3 debug=true metrics=false queue_depth=60
D: stage=live encryption=true ingress_port=8080 workers=3 debug=false metrics=false queue_depth=80
```

Six shuffled repairs set ingress port 8443, workers 3, debug false, metrics
true, encryption false, or stage test. The first three semantic assignments
form the unique qualifying triple. `{encryption=false, stage=test}` satisfies
all constraints but fails intent on all four. Other protected alternatives may
also have constraint support four, but all have intent support zero. The
independent alternate oracle
must establish the same 41 candidates and 164 applications/results/
observations plus uniqueness before comparing Nous.

Held-out configurations and expected outputs are literal fixtures constructed
only after discovery. They include an already-valid input with zero changes,
inputs needing one or two target repairs, both schemas, and protected-shortcut
outputs that satisfy constraints but fail protection. Tests assert exact
output, validity, preservation, and changed-key count; no held-out unit enters
the store. Invalid schemas, malformed configurations, conflicting plans, and
no-solution corpora promote nothing under separate reduced-cardinality
contracts.

The seed held-out fixtures are:

```text
valid service:
  in/out: production,true,443,2,false,true                    changes 0
port only:
  in:     production,true,80,4,false,false
  out:    production,true,443,2,false,false                  changes 2
production obligations only:
  in:     production,true,443,1,true,false
  out:    production,true,443,2,false,false                  changes 2
all obligations plus route_count=99:
  in:     production,true,80,0,true,true,99
  out:    production,true,443,2,false,true,99                changes 3
```

Here each row abbreviates the fixed key order from the schema; test fixtures
contain the complete literal `key=value` lists. Separate literal assertions
show `tls=false` on the port-only input and `environment=development` on the
production-only input satisfy constraints, change one key, and fail protected
intent.

The learned plan is an idempotent normalizer, not a minimum-change patch. Its
literal `replicas=2` assignment may overwrite a larger, already-valid
unprotected value, as the port-only fixture deliberately exposes. Changed-key
count is retained as evidence but is not a selection objective. Conditional
repairs such as `ensure-min` are a possible later vocabulary extension and
would require a new synthesis method and semantic decision identity.

“Reusable” means executable reuse on unseen instances of this same obligation
family and execution after primitive deletion. The alternate trial proves
fresh synthesis on different keys and values, not transfer of the seed plan or
usefulness of its credit.

## Acceptance and verification

Acceptance requires:

- isolated loading without math, protocol, rewrite, or build-graph units;
- scoped DSL exposure and pure/adapter differential tests;
- exactly 41 candidates, 41 evidence units, 164 results, 164 observations, and
  164 direct application records, with four linked examples, results,
  observations, and applications per candidate;
- exact independent-oracle agreement for every candidate;
- exactly one promoted repair schema and one attributed conjecture with the
  expected semantic components and target evidence;
- demonstrated rejection of the corpus-wide shorter validity-only protected
  shortcut and the per-example equal-cost shortcuts;
- held-out success, plan idempotence, and execution after primitive deletion;
- exact ordinary and contextual one-shot credit;
- parity between every primitive's structured repair and executable
  definition, every composite definition and pure plan application, and every
  plan permutation;
- opaque-alias, occupied-name, alternate-runtime, malformed, no-solution, and
  mutation-ineligible controls;
- byte-identical canonical stores for repeated no-mutation runs; and
- preservation of all existing vocabulary and kernel tests.

The engine trial uses a tested `MaxCycles` floor of 700, asserts that the
agenda drains, and completes an eligible unit-focus interval whose global
engine cycle is divisible by ten, demonstrated by `lastRewardedWorth == 800`
and the expected contextual records.

Verification commands are:

```sh
mise exec -- go test ./...
mise exec -- go test -race ./...
mise exec -- go vet ./...
git diff --check
```
