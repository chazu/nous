# Kubernetes Selector and Reference Repair Research Program V2

Status: revised after first v2 adversarial review

V1 is closed and preserved in `kubernetes-selector-reference-results.md`. V2
uses a corrected contract, new development and validation seeds, and a private
locked seed generated only after its one-shot guard is acquired.

## Question and claim boundary

V2 asks whether Nous can (a) exhaustively discover minimum plans of bound,
typed, single-leaf Kubernetes edits through its existing generic synthesis
heuristics, and (b) reuse structural credit from three one-edit discoveries to
reduce expensive terminal evaluations on deliberately matched later
recompositions.

The edit grammar and structural feature vocabulary are human supplied. A
positive result demonstrates bounded repair discovery and curriculum-specific
reward reuse. It does not demonstrate Kubernetes expertise, repair-rule
invention, production readiness, or scalable search. Mu, PUDL, live clusters,
YAML, admission/defaulting, rollout behavior, and network execution are out of
scope.

## Self-contained semantic subset

A canonical `kubernetes-bundle/v1` contains exactly one namespace, one
Deployment, one Service, zero to two concrete distractor Pods, one or two
containers per Pod template/Pod, at most four TCP ports per container, and
optional HTTP readiness probes. V2 fixtures contain one equality-only label key
under test, one Deployment-template container, one or two declared ports, one
Service port, at most one readiness probe, and at most one distractor Pod.
Fixture generation rejects a public bundle unless the independent edit sets
agree and contain `1..8` edits. No edit is truncated or sampled.

Names are lowercase DNS-label atoms of at most 63 characters. Label keys and
values in this synthetic subset use the same atom grammar; the subset does not
claim full Kubernetes label syntax. Identities and label keys are unique.
Container port names are unique across the complete Pod, not merely within one
container. Ports are `1..65535`. A `PortRef` is exactly one of `name`, `number`,
or `default`; readiness forbids `default`.

The Deployment selector is an immutable conjunction and must match its template
labels. The virtual Deployment Pod inherits template labels, containers, ports,
and probes. The Service selects every virtual or concrete Pod containing every
selector pair. V2 private intent always requires the virtual Deployment Pod as
the sole selected backend; concrete Pods are distractors. Consequently Service
reference sources are Deployment-template declared ports. A missing/default
Service target denotes its public Service port; an integer denotes that number;
a name must resolve to exactly one Pod-wide declared port. A readiness reference
resolves by name within its owning container or directly by number.

Private intent contains desired selected role, numeric backend, numeric
readiness endpoint per probed container, and a digest of protected projections.
Protected projections include presence and value for any protected selector or
label leaf, Service public port and target, declared port, and readiness
presence/path/port. Candidate construction and ranking see only public state.
The Nous store contains only a task-unique opaque 256-bit handle; only the
Boolean terminal capability can resolve it. Development/validation handles are
domain-separated deterministic digests of nonpublic driver seeds and ordinal;
the locked root seed is generated after the guard and recorded only afterward.

## Paths, edits, and independent oracle

Every typed path is resolved against the concrete bundle before a read,
protection check, write-history check, or mutation. Deployment/template paths
must name the actual Deployment; Service paths the actual Service; Pod paths an
actual Pod; and keyed members must exist at the declared owner. Protection and
write history use the same resolved semantic leaf identity. Forged resource
aliases are rejected.

The atomic grammar is:

1. copy a public label value to a template-label or Service-selector leaf with
   the same key;
2. remove one Service-selector label;
3. set the Service target or owning-container readiness reference from a bound
   Deployment-template port by name or number; and
4. unset the Service target to `default`.

Public label destination keys are the union of keys at all public label
sources. A legal edit changes one unprotected leaf. Rewriting a destination in
one plan is invalid. Syntactic identity is the ordered canonical edit sequence.
Semantic result identity is canonical final public state with write-history
metadata removed. Complete minimum sequences and deduplicated minimum result
states are both reported.

The standard-library-only `internal/kuberepairoracle` entrypoint receives only
canonical public JSON, independent private intent, and length bound. It
independently decodes, validates, enumerates canonical edits, applies them, and
evaluates final states; it accepts no production edit list. A source/dependency
test forbids imports or shared/generated code from the production vocabulary,
fixture, DSL, engine, seed, credit, or trial packages. Production and oracle
edit sets are compared byte-for-byte before either solution is used.

## Phase A: synthesis and oracle parity

The first three `tinystack` enumerate/evaluate/select heuristics remain
byte-identical. After exact edit-set agreement, a task setup materializes every
production edit as a primitive and Nous performs ordinary bounded synthesis.
With `n` edits, the unchanged synthesizer must materialize exactly
`n + n^2 + n^3` nonempty syntactic candidates, including sequences whose
duplicate destination later evaluates to semantic nil.

Before primitive materialization, production and oracle independently evaluate
the empty plan. If safe, the terminal is `already-correct` with minimum set
`{[]}` and synthesis/evidence requirements are inapplicable. Otherwise, Phase A
compares Nous's complete selected minimum sequence set and semantic result set
with the independent oracle.

Panels fixed before implementation are:

- development seeds `761001..761006` (6 tasks);
- validation seeds `762001..762012` (12 tasks);
- locked: 24 tasks derived without rejection from the private locked root seed.

The direct seed-to-case mapping cycles through unique one-edit,
two-feature/co-minimal, three-feature/co-minimal, already-correct, no-solution,
and reference-repair cases. Public state and edit sets are frozen before intent.
No task is regenerated. A structural mismatch makes the panel invalid.

Phase A is positive only if edit sets and candidate counts agree, evidence is
complete, every selected plan replays, protected digests survive, and every
terminal, minimum length, minimum sequence set, and semantic result set agrees.
Unit tests additionally cover malformed input, forged/protected aliases,
duplicate destinations, canonical ordering, alpha-renaming, irrelevant Pods,
primitive deletion, source-equal generic heuristics, structural-credit
provenance, and deterministic reruns.

## Training and structural credit

Training seeds are exactly `750001`, `750002`, and `750003`. They yield one
unique one-edit winner apiece: template label from Deployment selector, Service
selector from template label, and extra Service-selector removal. Validated
components emit position-independent feature credit, step-specific feature
credit, relation credit, and an ordered structural-decision record. Keys exclude
names, label keys/values, port names/numbers, positions, task IDs, and intent.
Malformed declarations emit no partial credit.

Acquisition cost is the exact number of times Nous actually invokes the Boolean
terminal evaluator across the three discoveries, including the three initial
empty-plan calls and every evaluated nonempty candidate. A test reconciles the
counter with the evaluator call log.
It is measured in the same primary unit as Phase B and amortized over the fixed
32 locked component tasks. Candidate creation and credit-update event counts are
reported separately and are not mixed into the primary endpoint.

## Phase B: matched terminal-evaluation trial

All policies receive one byte-shared eligible-plan list and one byte-shared
base permutation. Every policy first performs and charges one Boolean evaluation
of unchanged public state. Plans are then searched by increasing length;
policies reorder only within a length stratum. The first Boolean-safe plan is
therefore minimum without oracle information. No-solution requires traversing
the complete bounded eligible list. Oracle results are consulted only after
policy termination to audit safety, terminal class, and minimum identity.

The primary loss is Boolean terminal calls to first minimum solution, plus
`training calls / 32` for contextual. Failure loss is `401`, one above the
maximum eight-edit eligible list. Construction, scoring-key, stable-sort
comparator, and oracle-audit calls are reported as separate exact event counts;
they do not enter the primary claim. This avoids assigning arbitrary exchange
rates to unlike computations.

Within each length, all ties use ascending position in the one shared
Fisher-Yates permutation. Every deterministic stream encodes canonical JSON
`["kuberepair-v2-stream/v1", domain, rootHex, ordinal]`, hashes its UTF-8 bytes
with SHA-256, interprets digest bytes `[0:8]` and `[8:16]` as unsigned big-endian
integers, and passes them in that order to Go `math/rand/v2.NewPCG`. Ordinals are
zero-based. Integer development/validation roots are lowercase 64-digit hex.
The domains are `phase-a-case`, `phase-b-case`, `permutation`,
`intent-handle`, `inference/<comparator>`, `power-outer`, and
`power-inner/<comparator>`. A power-inner stream uses simulation index as its
ordinal. Ranking keys are computed once per plan and cached. No policy adapts
after terminal observations.

- `contextual` stable-sorts lexicographically descending by structural-decision
  reward, summed position-independent feature reward, summed matching step
  reward, and summed relation reward;
- `scalar` sorts descending by summed feature-unit worth;
- `constraint` sorts ascending by `[deployment selector/template mismatch
  count, empty Service selector bit, no selected backend bit, unresolved
  Service reference bit, unresolved readiness count, changed-leaf count]`;
- `no-credit` retains base order. `wrong-context` executes the complete
  contextual key/cache/stable-sort path against a domain-separated empty
  context. `reset` executes that same path against an independently constructed
  seed-worth profile. Their resulting orders and terminal traces, rather than
  aliases assigned by construction, must be byte-identical to no-credit.

Development root `751001` maps directly to 24 tasks: 6 exact reuse, 6
`two-feature`, 6 `three-feature-recombined`, 2 cross-role, and one each
unrelated, co-minimal, already-correct, and no-solution. Validation root `752001`
maps directly to twice each development count (48 tasks). The locked root yields
96 tasks: 32 exact reuse, 16 `two-feature`, 16
`three-feature-recombined`, 8 cross-role, 8 unrelated, 4 co-minimal, 4
already-correct, and 8 no-solution. Every minimum plan in `two-feature` has
length two with exactly two distinct credited feature keys. Every minimum plan
in `three-feature-recombined` has length three with all three credited keys.
Neither has structural-decision credit for a minimum sequence. Each panel uses
this ordinal-to-cohort mapping without rejection or replacement. The generator
is independent of learned scores; cohort classification is post-generation and
a mismatch invalidates the panel.

This deliberately matched curriculum does not establish broad transfer.
Misleading credited-feature decoys occur in the eligible list. Cross-role and
unrelated cohorts report negative transfer. The unrelated gate excludes
training, assigns 401 to failure, and requires `(mean contextual Phase-B calls -
mean no-credit calls) / mean no-credit calls <= 0.10`; otherwise the run is
`valid-null`.

For paired loss effects, each of 10,000 replicates samples task indices with
replacement using a domain-separated PCG inference stream, adds the fixed
`training/32` term to the contextual mean, and computes
`(contextual - comparator) / comparator`. The percentile interval uses sorted
indices 249 and 9749. V2 is `valid-positive` only if all integrity checks and
Phase A pass, pooled contextual component loss is at least 15% below constraint
with interval upper bound below zero, contextual loss is at least 10% below
no-credit in each component stratum with each interval upper bound below zero,
and the unrelated negative-transfer gate passes. Otherwise an integrity-clean
run is `valid-null`; any unsafe acceptance, evaluator/oracle disagreement,
cohort drift, unequal inputs, or accounting mismatch is `invalid`.

The development report supplies the only variance/effect population for a fixed
power simulation. Using development paired observations only, perform exactly
2,000 simulated locked panels. Each simulation independently samples 16
two-feature pairs and 16 three-feature pairs with replacement. It then applies
all three confirmatory gates: pooled contextual-versus-constraint and the two
stratum contextual-versus-no-credit gates, using 2,000 inner percentile
bootstrap replicates and the same point thresholds/training term as locked.
The outer stream uses domain `power-outer`, development root, and ordinal zero;
the inner streams use the `power-inner/<comparator>` domain, development root,
and simulation index. Each inner interval sorts 2,000 effects and selects
zero-based indices 49 and 1949.
Power is the fraction satisfying all three gates and must be at least 0.80.
Validation and locked execution are forbidden otherwise. Validation may reject
the implementation once; acceptance requires Phase-A positive, all integrity
checks true, deterministic byte-equal rerun, and power at least 0.80. Phase-B
validation may be `valid-positive` or `valid-null`; it is evidence, not a tuning
gate. After its committed result, thresholds, mappings, and code freeze.

## Locked-run authority

One guarded `RunLockedV2` API, used by CLI and package callers, runs Phase A
first and Phase B only if Phase A is valid and positive. Before generating the
random locked root it requires an explicit unlock token, verifies the supplied
commit equals `HEAD`, rejects any output from `git status --porcelain=v1
--untracked-files=all`, verifies the canonical repository/domains path and no
active `go.work` or module replacement, and verifies a committed prerequisite
manifest. The fixed tracked manifest is
`.nous/kuberepair-v2-prerequisites.json`. It contains the protocol version, the
approved implementation parent commit, and unique `{path, sha256}` entries for
the protocol, generator sources, training-profile report, complete test log,
development report, validation report, power report, and accepted adversarial
review. The manifest is the only change in the current guard commit relative to
its approved parent. Before receipt creation, the guard verifies that relation,
verifies every path is relative, nonescaping, regular, tracked, unique, and not
a symlink, and verifies both working-tree bytes and `HEAD:path` bytes against the
recorded SHA-256. The unlock token contains the current guard commit, and the
guard requires `HEAD == supplied commit`.

Before receipt creation the guard parses the manifest-hashed power report,
requires matching protocol and development-report hash, the exact 2,000/2,000
simulation parameters and percentile indices, and recorded power at least
`0.80`. It similarly requires the manifest-hashed validation report to record
Phase-A positive, integrity true, and a byte-equal deterministic rerun.

The only receipt is `.nous/kuberepair-v2-locked-receipt.json`; the only report is
`.nous/kuberepair-v2-locked-report.json`. Before root generation the guard uses
`lstat` on both paths and every path component, rejects any existing output or
symlink, then exclusively creates and fsyncs a `claimed` receipt containing
version, commit, and command. Any existing or incomplete receipt permanently
refuses another v2 run. It then reads exactly 32 bytes from `crypto/rand.Reader`,
encodes 64 lowercase hexadecimal characters, and atomically replaces/fsyncs the
receipt as `started` with that root before deriving any task. All task and
inference streams then use the canonical JSON derivation and domain names pinned
above.
The full report is exclusively created and fsynced; its SHA-256 and terminal
status are then installed into the receipt by fsynced temporary-file rename. A
crash consumes the attempt. An integrity-clean Phase-A performance miss
finalizes `valid-null`; an edit-set, evaluator, oracle, information, cohort, or
accounting disagreement finalizes `invalid`. Either skips Phase B. Locked cohort
validation happens only inside this guard. No other public API accepts, exposes,
or derives locked v2 tasks.

Inside the guard, the three fixed training tasks are rerun through Nous before
Phase B. The run requires one unique one-edit winner each, validates the complete
concrete and structural credit tuple set, and requires its canonical training
report to byte-equal the manifest-hashed profile report. Only that newly
produced in-memory profile ranks locked tasks; mismatch is `invalid`, and the
observed evaluator-call count is the acquisition term.

After locked observation, any change to semantics, fixtures, ranking,
accounting, gates, or generator creates v3 and preserves v2.
