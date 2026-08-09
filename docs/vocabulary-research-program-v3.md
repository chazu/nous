# Vocabulary research program, Part 3

Status: revision 2, under adversarial review.

Revision record:

- revision 1 was committed at `ebbcf1b17c3c7b787568ed674e4026af8138e975`
  and rejected by adversarial review for incomplete historical isolation from
  Part 2, an insufficient diagnosis information boundary, an answer-bearing
  transformation diff, inconsistent partial-order lifecycle accounting, and
  an exhaustive-audit loophole; and
- revision 2 closes those blockers and the associated soundness, baseline,
  work-ledger, latent-preference, and terminal-classification concerns. It has
  no implementation authority until a new adversarial review accepts it.

Part 3 is a destination-independent curriculum for improving Nous one
reasoning operation at a time. It does not amend, resume, or claim completion
of the gate-chained [Part 2 program](vocabulary-research-program-v2.md). Part 2
remains terminally invalid at its recorded active-causal boundary. This
document instead proposes seven independent micro-vocabularies whose results
stand on their own.

The change in emphasis is deliberate. We are no longer asking which application
domain Nous should eventually serve. We are asking which small, inspectable
reasoning capability would be useful for Nous to acquire next, and how to test
whether it actually acquired it.

## Evidence motivating Part 3

The implemented vocabularies have established several reusable mechanisms:

- finite-state protocols showed blinded relation discovery, counterexample
  retention, alias independence, and held-out schema validation;
- rewrite synthesis recovered 100 of 100 generated hidden programs with zero
  false promotions, while exact contextual credit plus exploration improved
  bounded later-task success over scalar credit;
- configuration repair and tiny-stack programs showed descriptor-driven
  bounded synthesis, complete evidence barriers, behavioral selection, and
  executable artifacts;
- iterated games showed that a Pareto frontier retains consequential
  alternatives that scalar selection discards;
- relational rule induction materialized a learned relation, reused it across
  a task boundary, and sharply reduced downstream search, although its isolated
  materialization benefit missed the preregistered threshold; and
- Kubernetes selector/reference repair showed alias-independent structural
  transfer after training, but its small advantage over a conventional
  constraint ordering could not repay acquisition cost over the fixed horizon.

The detailed records are the [protocol design](finite-state-protocol-vocabulary-plan.md),
[rewrite trials](rewrite-trials.md),
[bounded-synthesis analysis](bounded-program-synthesis-analysis.md),
[game trial](iterated-game-strategies-trials.md),
[relational-rule trial](relational-rule-induction-trials.md), and
[Kubernetes result](kubernetes-selector-reference-results.md).

Together they suggest that Nous is strongest when it constructs small symbolic
artifacts, evaluates them behaviorally, preserves evidence, and reuses exact or
nearly exact structures. Its largest open weakness is abstraction: humans still
supply most of the feature language that says why two successful decisions are
the same. Part 3 therefore moves through increasingly ambitious forms of
learned negative knowledge, generalization, relational compression, defeasible
belief, library formation, explanation, and contextual choice.

## Program objective

Part 3 asks whether ordinary Nous heuristics can acquire seven additional
reasoning moves:

1. convert a failed search branch into a reusable constraint;
2. generalize concrete transformations into a parameterized schema;
3. discover when operations commute, interfere, enable, or disable;
4. refine a conjectured invariant in response to counterexamples;
5. compress recurring programs into a reusable executable library;
6. construct minimal explanations and rank discriminating evidence from a
   fully public model; and
7. learn contextual preferences over a set of nondominated alternatives.

Each vocabulary must produce a first-class artifact that can be inspected,
replayed, deleted, corrupted in controls, and causally connected to later work.
Merely solving the domain problem is insufficient. A conventional solver may
solve every bounded instance; the research question is whether Nous learns an
artifact that changes later reasoning under matched information and work.

## Approach: a curriculum of marginal gains

### Independent lanes, not a gate chain

The proposed priority order is:

1. constraint and nogood learning;
2. transformation-schema induction;
3. commutativity and partial-order reasoning;
4. counterexample-guided invariant learning;
5. subprogram and macro libraries;
6. abductive diagnosis; and
7. preference and trade-off reasoning.

This order expresses expected gain per implementation risk, not authorization.
Every vocabulary receives its own design, preregistration, implementation
review, and result record. A mechanically invalid vocabulary blocks only its
own claim. A valid null remains useful evidence and does not prevent a different
lane from being attempted. No lane may quietly import another lane's learned
store, fixtures, private outcomes, or results.

### Historical isolation and non-supersession

Part 3 is a new research authority, not a continuation mechanism for the
terminated program. The [Part 2 terminal record](active-causal-diagnosis-v6-contract-rejection.md)
remains authoritative: Phase 2 is invalid, no recovery or replay is authorized,
and its Phases 3 through 10 remain unexecuted. No Part 3 plan, implementation,
report, or review may complete, repair, reclassify, resume, or advance a Part 2
phase.

The following identities and distinctions are fixed before any lane-specific
plan:

| Part 3 lane and identity | Closest earlier or terminated work | Frozen distinction |
| --- | --- | --- |
| `domains/nogoods`, seed authority `part3/nogoods/v1` | Part 2 scheduling and failure minimization | learns universally scoped negative constraints in a public finite CSP; it does not learn a scheduling priority rule or minimize a failing artifact |
| `domains/transformschema`, seed authority `part3/transformschema/v1` | Part 2 spatial puzzles and existing configuration repair | anti-unifies explicit typed term-edit programs; it has no grids, perception grammar, Kubernetes objects, or protected configuration fixtures |
| `domains/actionrelations`, seed authority `part3/actionrelations/v1` | terminated Part 2 Phase 9 | learns guarded relations over a fully public deterministic action algebra and uses charged local diamond certificates; it has no hidden SUT, concurrency history, linearizability checker, or schedule-priority program |
| `domains/invariantrefine`, seed authority `part3/invariantrefine/v1` | terminated Part 2 Phase 4 | revises formulas against a fixed public trace corpus; it has no teacher, active probes, hidden machine, probe policy, or Part 2 macro-F1 hypothesis |
| `domains/macrolib`, seed authority `part3/macrolib/v1` | Part 2 spatial concepts and equality saturation | compresses an explicit corpus of solved typed programs into expansion-equivalent macros; it has no grids, e-graphs, rewrite scheduler, or terminal extractor |
| `domains/abduction`, seed authority `part3/abduction/v1` | terminally invalid Part 2 Phase 2 | performs passive model-based diagnosis over a fully public component model and fixed observations; it has no causal hypotheses, interventions, hidden teacher, query responses, acquisition rule, replay, or protected receipt |
| `domains/preferences`, seed authority `part3/preferences/v1` | implemented iterated games | learns a contextual partial preference over an already verified frontier; it does not discover the frontier or treat preference as dominance |

Repository and claim identities are also fixed:

| Lane | Production, fixture, experiment, and CLI identities | Frozen hypothesis, artifact, and primary endpoint |
| --- | --- | --- |
| Nogoods | `internal/vocab/nogoods`; `internal/nogoodfixture`; `internal/nogoodexp`; `nogood-trials` | a scoped learned nogood reduces total held-out CSP lifecycle work while preserving the exact solution set |
| Transformation schema | `internal/vocab/transformschema`; `internal/transformschemafixture`; `internal/transformschemaexp`; `transformschema-trials` | an induced executable schema improves exact held-out transformation success/work under alpha-renaming without false application |
| Action relations | `internal/vocab/actionrelations`; `internal/actionrelationfixture`; `internal/actionrelationexp`; `actionrelation-trials` | a learned guarded relation plus charged local certificates reduces total lifecycle work while preserving terminal behaviors |
| Invariant refinement | `internal/vocab/invariantrefine`; `internal/invariantrefinefixture`; `internal/invariantrefineexp`; `invariantrefine-trials` | counterexample-driven formula lineage improves held-out soundness/recall and lifecycle work on a fixed public trace corpus |
| Macro library | `internal/vocab/macrolib`; `internal/macrolibfixture`; `internal/macrolibexp`; `macrolib-trials` | a learned expansion-equivalent macro reduces total stream proposal/construction work after full amortization |
| Abduction | `internal/vocab/abduction`; `internal/abductionfixture`; `internal/abductionexp`; `abduction-trials` | a learned fault signature reduces passive diagnosis lifecycle work while preserving every subset-minimal diagnosis |
| Preferences | `internal/vocab/preferences`; `internal/preferencefixture`; `internal/preferenceexp`; `preference-trials` | a contextual partial-preference rule reduces held-out latent regret at frozen coverage and lifecycle work without changing the frontier |

Each lane uses a fresh fixture package, experiment package, CLI identity,
generator version, and seed derivation rooted only in its `part3/.../v1`
authority. Lane-specific plans may narrow semantics and bounds but may not
rename these identities or hypotheses without an accepted Part 3 amendment.
They must source-audit the complete dependency graph and reject any import, file
read, seed reuse, fixture reference, report reference, or runtime path into a
closest-overlap pack named above.

All Part 2 panel seeds, hidden fixtures, teachers, evidence bundles, reports,
attempt/replay/diagnostic receipts, and `.git/nous-attempts` state are forbidden
inputs. Part 3 tests must prove their absence from source constants, runtime
file-open traces, dependency graphs, stores, transcripts, and reports. A Part
3 result is reported only in the Part 3 capability matrix; it cannot be cited
as the result of any Part 2 phase.

### One new reasoning operation per vocabulary

Every design must name one distinguishing learned artifact and one causal use:

| Vocabulary | Learned artifact | Required causal use |
| --- | --- | --- |
| Constraint learning | generalized nogood | prune a later legal branch before semantic evaluation |
| Transformation induction | parameterized transformation schema | solve a renamed/recombined transformation task |
| Partial-order reasoning | guarded action relation | omit a redundant interleaving without losing behavior |
| Invariant learning | refined invariant with evidence boundary | reject or classify a held-out trace/state |
| Macro libraries | parameterized executable macro | reduce later synthesis work after full amortization |
| Abductive diagnosis | fault-signature schema plus complete minimal diagnoses | reduce later passive diagnosis search and optionally rank public measurements offline |
| Preference reasoning | contextual preference rule | select among retained Pareto alternatives in a new context |

If the artifact is only a renamed candidate, cached answer, human-written
feature, or conventional algorithm result copied into a unit, the vocabulary
has not supplied its intended marginal gain.

### Two-stage evidence contract

Every lane has two conceptually separate gates:

1. **semantic competence** proves that Nous can represent, construct, execute,
   and independently verify the intended artifacts on tiny complete spaces;
2. **marginal utility** tests a frozen learned artifact on later tasks under a
   budget where search order, pruning, reuse, or evidence choice matters.

Semantic competence may use a complete `exhaustive-matrix` contract. Marginal
utility normally requires a `budgeted-transcript`: every proposed refinement,
duplicate, evaluation, cache lookup, learned-artifact lookup, rejected action,
and terminal event is charged and retained. Passing the first gate must never
be described as passing the second.

### Shared architecture boundary

Part 3 inherits the repository's restored vocabulary boundary:

- a vocabulary is a normal `domains/<name>` pack loaded with
  `domains/common` and no other domain;
- pure production semantics may parse, canonicalize, validate, or execute one
  explicit bounded object or transformation;
- ordinary CUE heuristics own population construction, evidence materialization,
  abstraction proposals, selection, credit declarations, and learned artifacts;
- independent fixtures, baselines, oracles, and experiment drivers live in
  separate packages with source/dependency tests;
- the engine, agenda, VM state, mutation machinery, common pack, math pack, and
  existing vocabulary semantics are fixed; and
- no vocabulary gains authority to inspect or mutate an external system.

Production Go may not hide complete search, anti-unification over the whole
training corpus, conflict analysis, partial-order reduction, invariant learning,
library extraction, hitting-set diagnosis, or preference fitting behind one
builtin. It may expose bounded one-object facts from which Nous heuristics build
those results. Conventional complete implementations belong only in isolated
baseline or oracle packages.

### Information rights and leakage controls

Every specific plan must freeze:

- what is public before a policy starts;
- what becomes visible only after a charged evaluation or measurement;
- which artifact is learned during training and exactly when it is frozen;
- which validation and locked information is unreachable from production;
- semantic identities independent of unit names, aliases, labels, and prose;
- the legal refinement grammar shared by every learned and unlearned policy;
  and
- terminal states including success, ambiguity, no-solution, no-violation,
  budget exhaustion, and mechanical invalidity.

An independent oracle may audit a completed transcript but may not choose a
production proposal or stopping point. V1 lanes use public fixtures except for
post-termination held-out scoring truth explicitly named by their contracts.
Any future lane that authorizes active hidden responses must use a separately
reviewed capability/opaque-handle boundary and cannot inherit authority from
this document. Wrong-context, reset, no-artifact, corrupted-artifact, and
random controls must receive the same public objects and legal action set.

### Marginal utility and acquisition cost

The main endpoint for every lane is relative to the strongest credible
conventional baseline, not merely random ordering. Exact endpoints and effect
thresholds belong in each vocabulary's preregistration, but every reuse claim
must satisfy all of the following:

- equal terminal correctness or an explicitly scored accuracy trade-off;
- equal expressivity and legal refinements;
- acquisition, validation, storage, lookup, and application work charged;
- a declared task horizon long enough to measure amortization without choosing
  it after observing the crossover;
- no-artifact and equal-expressivity inlined or recomputed controls;
- paired seeds and deterministic independent randomness streams; and
- confidence intervals or exact paired tests appropriate to the frozen panel.

Both total lifecycle work and post-training inference work are reported. The
former determines the primary reuse claim; the latter diagnoses whether a null
is caused by bad guidance or an insufficient reuse horizon.

Every lane-specific plan defines one common semantic-work ledger before any
panel is generated. At minimum it has separately reconciled counters for
candidate construction/refinement, semantic execution, artifact validation,
storage, lookup/matching, application/expansion, cache access, certificate
checking, and terminal audit. A scalar primary endpoint may combine them only
through frozen integer weights justified by explicit primitive interpreter
operations and applied identically to Nous and conventional baselines. The raw
vector is always reported. Independent post-termination oracle work is reported
separately and cannot create a search-advantage claim; policy-visible or
artifact-validation work can never be relabelled as oracle audit.

### Evidence status

Each vocabulary ends as:

- `valid-positive`: all mechanical gates pass and the preregistered marginal
  claim passes;
- `valid-null`: all mechanical gates pass but the empirical claim does not;
  or
- `invalid`: a semantic, leakage, accounting, oracle, provenance, or frozen
  protocol gate fails.

Failure to solve, ambiguity, a counterexample, an empty diagnosis set, an
incomplete preference order, or budget exhaustion names a potentially valid
terminal class only when its preregistered predicate is independently verified.
A false no-solution, false empty diagnosis, premature ambiguity, false
exhaustion, incorrect success, or any other terminal misclassification is
mechanical invalidity. Reports must distinguish domain hosting, bounded
discovery, search advantage, and reusable abstraction.

## Vocabulary 1: constraint and nogood learning

### Marginal reasoning gain

This vocabulary asks whether Nous can turn failure into future search control.
Earlier packs retained failed candidates, but usually used them only as
negative evidence. Here the required artifact is a sound generalized nogood:
a compact condition describing a class of partial assignments that cannot lead
to a solution.

### Objects and bounded semantics

The domain represents finite typed constraint problems:

- variables with finite domains;
- partial and complete assignments;
- unary and binary constraints such as equality, inequality, precedence,
  adjacency, incompatibility, and bounded resource capacity;
- decision literals `(variable, value)`;
- observed conflicts linking a partial assignment to violated constraints;
- candidate conflict subsets and generalized role patterns; and
- nogoods with scope, ordered provenance, support, refutations, and a canonical
  semantic key. The scope explicitly names the allowed variable/constraint
  roles, role-respecting substitution universe, value domains, and completion
  universe quantified by the claim.

Initial families should be tiny graph coloring, precedence scheduling, and
package-version compatibility instances expressed through one generic finite
CSP representation. Production semantics may evaluate one explicit partial or
complete assignment and return its explicit violated constraints. They may not
compute a minimal conflict, learn a clause, backjump, or search the problem.

### Discovery loop

Ordinary heuristics:

1. choose an unassigned variable and propose a legal value;
2. materialize the resulting partial assignment and evaluation;
3. when conflict occurs, propose subsets of the decisions participating in the
   explicit violation;
4. test whether removing a literal destroys the conflict witness;
5. replace concrete identities with descriptor roles only when multiple
   alpha-renamed conflicts support the substitution;
6. independently enumerate every role-respecting substitution and every legal
   completion inside the proposed finite scope, retaining each agreement or
   counterexample;
7. retain failed generalizations and counterexamples; and
8. promote a sound, minimal-within-grammar nogood only when the scoped
   completion check is exhaustive.

On later problems, a separate heuristic matches frozen nogoods against a
partial assignment. A match may prune only the represented branch. The
transcript must record the skipped refinement set and an oracle audit must show
that no solution was removed. Before a held-out prune, an instance-specific
certificate binds every schema role to public target variables/constraints and
exhaustively checks the artifact's declared completion universe. Certificate
construction and checking are charged; a schema may be useful only when that
bounded check is cheaper than exploring all skipped continuations.

### Task stream and controls

Training uses related problems with different names, domain values, and graph
layouts. Validation recombines supported motifs and includes sparse, dense,
irrelevant-feature, misleading-local-conflict, already-unsatisfiable, and
no-reusable-conflict cases.

Required comparisons are:

- chronological backtracking with no learning;
- forward checking or another fixed conventional propagation baseline;
- a conventional conflict-learning/backjumping baseline;
- exact concrete conflict memoization without generalization;
- Nous with the frozen generalized nogoods;
- wrong-family and corrupted-nogood controls; and
- an exhaustive tiny-instance oracle.

The primary endpoint is charged branch/evaluation work to the correct terminal,
including learning and matching cost, with exact solution-set preservation.
Secondary endpoints include learned-nogood precision, branch-pruning count,
generalization distance, storage growth, and amortization crossover.

### Main risks and non-claims

A clause observed to fail once is not universally sound. Promotion therefore
requires exhaustive bounded validation of its explicit quantified scope and an
instance certificate for each causal prune. One false prune mechanically
invalidates the pruning claim; it is not merely a precision loss. A
human-written domain constraint copied into a learned unit is not a discovery.
The experiment does not claim competitive SAT/CSP solving.

### Research anchors

- Rina Dechter's [backjumping, learning, and cutset decomposition](https://doi.org/10.1016/0004-3702%2890%2990046-3)
  formalizes constraint learning as an enhancement to backtracking.
- Marques-Silva and Sakallah's [GRASP](https://doi.org/10.1109/12.769433)
  makes conflict analysis and recorded conflict causes central to pruning
  propositional search.

Part 3 borrows the question—can conflict knowledge prevent repeated failure?—
not their complete solvers.

## Vocabulary 2: transformation-schema induction

### Marginal reasoning gain

This vocabulary replaces human-supplied structural feature keys with learned
abstraction. Given several concrete before/after transformations, Nous must
construct a parameterized schema that captures their shared operation and
applies correctly to renamed or recombined objects.

### Objects and bounded semantics

The initial representation is a small typed term forest rather than a real
configuration language:

- nodes have a kind, optional scalar value, and ordered or key-addressed
  children;
- references are explicit typed edges between nodes;
- concrete edits include replace-value, rename-definition, rewrite-reference,
  insert-child, remove-child, and move-child;
- a transformation program is an ordered edit sequence of bounded length;
- metavariables range over node identities, paths, values, or repeated
  definition/reference roles; and
- a schema contains a parameter list, guards, edit template, supporting
  examples, rejected examples, and expansion digest.

Production semantics may compare one explicitly named node/path pair, validate
or apply one proposed primitive edit, or execute one concrete or already-
instantiated schema application. A node/path comparison returns only typed
equality and local kind/value/edge facts; it cannot return an edit, alignment,
partner path, or answer-bearing diff. Production may not align a whole pair,
construct or order its edit set, compare the corpus, or compute an anti-unifier.

### Discovery loop

Nous first enumerates bounded node/path pairings and primitive edit forms as
ordinary artifacts, evaluates proposed edits, and synthesizes concrete ordered
transformations that exactly explain each training pair. Alignment, edit-set
construction, and edit ordering are therefore part of the heuristic transcript.
Nous then proposes generalizations by replacing aligned constants or paths
with shared metavariables, merging compatible edit positions, and adding
equality or role guards when one metavariable appears more than once.

Each proposed schema is expanded against every currently visible positive and
negative example. Over-general schemas retain their counterexamples and may be
specialized by adding a guard or splitting one metavariable. Selection favors
the least complex complete schema under a frozen description-length measure;
all co-minimal schemas remain explicit.

The required artifact is not a label such as `reference-repair`. It is an
executable parameterized transformation whose expansion reproduces concrete
programs and whose variables correspond across definitions and uses.

### Task stream and controls

Families include coordinated renames, wrapper insertion, mirrored field moves,
definition/reference repair, and two-edit recombinations. Held-out tasks change
all names and values, alter irrelevant siblings, permute unordered children,
and combine previously separate roles. Negative cases share surface tokens but
require different variable bindings.

Required comparisons are:

- concrete program replay;
- exact-decision contextual credit;
- a conventional least-general-generalization or anti-unification baseline;
- a bounded programming-by-example baseline using the same edit grammar;
- Nous with learned schemas;
- schemas with variable-equality guards removed;
- wrong-context and corrupted-schema controls; and
- an exhaustive bounded enumerator/oracle.

Every baseline receives the identical public node/path comparison primitive
and no whole-pair diff. If a future lane elects to make an answer-bearing diff
public, concrete program recovery becomes an input rather than evidence and
cannot contribute to a discovery claim.

The primary endpoint is held-out exact transformation success under a fixed
semantic-work budget, including schema-acquisition cost over the declared task
horizon. False application is a hard safety endpoint. Secondary measures are
schema compression, tasks solved per schema, binding ambiguity, and inference
work.

### Main risks and non-claims

Lexical similarity can masquerade as abstraction. Alpha-renaming, adversarial
token reuse, reordered trees, unused definitions, and source-equal decoys are
mandatory. The grammar must remain fixed across learned and unlearned policies;
a schema may shorten a representation but may not add transformations that the
primitive grammar could not express.

### Research anchors

- Kutsia, Levy, and Villaret's open-access
  [anti-unification for unranked terms and hedges](https://doi.org/10.1007/s10817-013-9285-6)
  gives a precise account of generalizing tree-like structures.
- Gulwani's [FlashFill programming-by-example work](https://www.microsoft.com/en-us/research/publication/automating-string-processing-spreadsheets-using-input-output-examples/)
  demonstrates how a bounded transformation language can be synthesized from
  examples while retaining multiple interpretations.

Nous must materialize its own schema and evidence; these algorithms are
conventional baselines and conceptual guides.

## Vocabulary 3: commutativity and partial-order reasoning

### Marginal reasoning gain

This vocabulary asks whether Nous can discover relations among actions and use
them to reason about equivalence classes of executions. The intended gain is
not merely ranking sequences. It is safely avoiding redundant interleavings or
recognizing interference because an inspectable guarded relation justifies the
reduction.

### Objects and bounded semantics

The domain contains:

- finite typed states;
- actions with explicit preconditions and deterministic effects;
- executable histories;
- state observations and terminal predicates;
- relation candidates `commutes`, `enables`, `disables`, and `conflicts`;
- guards over bounded state predicates; and
- equivalence classes or dependency partial orders derived from accepted
  guarded relations.

Examples should use a tiny neutral world: counters with limits, locks and
resources, token movement, independent key updates, and create/use/delete
dependencies. Production may execute one action or explicit history and
compare two terminal states. It may not calculate independence, persistent
sets, ample sets, or a reduced schedule universe.

V1 observes applicability, errors, and an explicit terminal-state projection.
Any event whose order matters is stored in the state projection, so swapping
two event-emitting actions cannot appear commutative. Intermediate-state
temporal properties are deferred. A lane-specific plan may narrow the following
ceilings but not enlarge them: eight action kinds, 64 reachable states, history
length eight, 40,320 complete competence-panel sequences, 65,536 full-universe
utility histories, 512 normalized guards, and 256 action-pair training states.
Generators reject fixtures whose semantic universe crosses a ceiling.

### Discovery loop

For a chosen state and action pair, ordinary heuristics execute `a;b` and
`b;a`, preserving inapplicability, intermediate states, and terminal results.
They form unconditional relation conjectures from repeated agreement, then
specialize them with state guards when counterexamples appear. Enabling,
disabling, and conflict claims require asymmetric evidence, not the absence of
a successful swap.

A frozen reduction heuristic may use an accepted guarded commutativity relation
to propose a canonical representative among adjacent swaps. It may omit the
noncanonical continuation only after constructing a charged local diamond
certificate at the current public state. The certificate proves that both
actions are initially applicable, each remains applicable after the other,
both two-action executions terminate without error, and their explicit
terminal projections are equal. The certificate contains all four charged
transition applications, both guard/match checks, the relation identity, and
the representative identity. Determinism then gives both prefixes the same
future continuation state.

Competence panels independently expand every bounded sequence and verify the
local proof rule against complete behavior. That exhaustive audit proves
semantic competence only and its work is reported separately. Utility panels
never reconstruct skipped schedules. They independently replay each local
certificate and the explored representative transcript. A missing or false
certificate invalidates the pruning claim. When a local certificate cannot be
formed, a learned relation may prioritize the pair but may not prune it.

### Task stream and controls

Training includes unconditional independence, conditional commutativity,
one-way enabling, destructive conflict, and apparent commutativity broken by an
intermediate observation. Held-out tasks rename actions, change numeric values,
embed known motifs in longer histories, and include unseen guard combinations.

Required comparisons are:

- complete interleaving exploration;
- fixed lexical canonicalization;
- static read/write dependency analysis;
- a conventional partial-order-reduction baseline;
- Nous with learned guarded relations;
- relation discovery without guards;
- relation learning without causal use; and
- corrupted-relation and state-renaming controls.

The common utility ledger charges one unit for each candidate or guard
construction/refinement, guard evaluation, relation lookup, stored-artifact
read/write, primitive state transition, local-certificate predicate check, and
terminal classification. Conventional static analysis and partial-order
baselines pay the same units for relation construction, dependency lookup,
transition execution, and certificate checking. Raw category counts are also
reported.

The primary endpoint is total charged lifecycle work across the frozen
relation-training and later-use horizon while preserving the complete terminal
behavior set. Post-freeze explored histories and state transitions are
diagnostic secondary endpoints, along with relation precision, equivalence
class size, guard complexity, and the amortization crossover. Any lost terminal
behavior or missed violation mechanically invalidates the reduction claim.

### Main risks and non-claims

Equal final states alone do not prove safe commutation when applicability,
errors, emitted events, or the declared observation projection differ. The
local diamond and observation boundary are therefore frozen before discovery.
This vocabulary differs from the existing protocol pack: it learns conditional
relations among state-changing actions and then uses locally proved instances
to reduce a search, rather than testing a supplied unary transform against a
supplied protocol relation.

### Research anchors

- Peled's [representative-sequence formulation](https://ai.dmi.unibas.ch/research/reading_group/peled-cav1993.pdf)
  motivates preserving representatives of execution-equivalence classes.
- Godefroid, Peled, and Staskauskas's
  [industrial partial-order validation study](https://patricegodefroid.github.io/public_psfiles/ieee-tse96.pdf)
  demonstrates the state-space motivation for exploiting independence.

Their reduction algorithms are baselines. The Nous claim requires learning
the guarded relation and retaining its counterevidence.

## Vocabulary 4: counterexample-guided invariant learning

### Marginal reasoning gain

This vocabulary gives conjectures a defeasible lifecycle. Nous must propose a
state or temporal invariant, encounter a concrete counterexample, and refine,
split, or abandon the conjecture while preserving the evidence that changed
its status.

### Objects and bounded semantics

The domain represents finite execution traces as typed state/event records. A
bounded invariant grammar includes:

- unary state predicates and comparisons;
- binary relations between current and previous state;
- event preconditions and postconditions;
- bounded `always`, `eventually-within-k`, `until`, and precedence patterns;
- conditional invariants `guard -> property`; and
- supporting states/transitions, counterexample states/transitions, supporting
  traces, counterexample traces, and scope declarations.

Production semantics may evaluate one explicit formula on one explicit trace
and return a shortest local witness or counterexample. It may enumerate legal
one-step formula refinements from one explicit hole. It may not scan the corpus
to select predicates or synthesize the invariant.

### Discovery loop

Heuristics propose simple invariants from repeated observations, test them on
visible traces, and promote them only provisionally. A counterexample creates a
first-class refutation. The heuristic may:

- abandon the conjecture;
- add a guard supported by an observed distinction;
- weaken a bound;
- split one conjecture into two scoped conjectures; or
- mark the evidence ambiguous when the grammar cannot separate cases.

Every refinement points to its parent, counterexample, changed syntax, and
remaining support. The frozen artifact is a minimal-within-grammar invariant
set plus its evidence boundary—not an assertion of universal program truth.

V1 is entirely passive. Every trace belongs to a fixed public corpus generated
before policy execution with a seed independent of policy ordering. Policies
may choose which public trace/formula pair to evaluate next, paying the common
ledger, but there is no teacher, hidden machine, probe, generated response,
query token, or active evidence acquisition. The final competence audit checks
the complete bounded public corpus only after each policy terminates.

### Task stream and controls

Trace families include resource lifecycles, request/retry protocols, bounded
queues, leader/follower modes, and intentionally coincidental correlations.
Training omits selected rare modes so validation can expose overgeneralization.
No-nontrivial-invariant, multiple-co-minimal, insufficient-evidence, and
unexpressible-target cases are mandatory.

Required comparisons are:

- fixed template enumeration similar to dynamic invariant detection;
- a conventional bounded CEGIS/template-refinement learner over the same
  formula grammar, public trace corpus, and trace/formula evaluation budget;
- passive correlation without counterexample refinement;
- Nous with the full refinement lineage;
- no-guard and no-negative-evidence ablations;
- shuffled, alpha-renamed, and adversarial rare-mode controls; and
- an exhaustive bounded model/trace oracle.

The primary endpoint combines held-out soundness and useful-property recall at
a frozen complexity bound with total lifecycle work over the frozen corpus
horizon. A search-advantage claim requires fewer charged formula/trace
evaluations than the strongest equal-grammar baseline at noninferior accuracy.
Reports distinguish observed, bounded-verified, refuted, and undetermined
status.

### Main risks and non-claims

Observed invariants are not proofs outside the bounded model and evidence
boundary. Vacuous implication, unreachable guards, duplicate traces, and data
leakage are mechanical controls. A shorter formula is not better if it erases a
meaningful condition or admits a known counterexample.

### Research anchors

- Ernst and colleagues' [Daikon overview](https://homes.cs.washington.edu/~mernst/pubs/daikon-tool-scp2007-abstract.html)
  motivates dynamically detecting likely invariants from executions.
- Garg, Löding, Madhusudan, and Neider's
  [ICE framework](https://doi.org/10.1007/978-3-319-08867-9_5)
  adds implication counterexamples to positive and negative examples for
  invariant learning.

Part 3 narrows these ideas to tiny inspectable formula and trace spaces in
which Nous's refinement lineage can be audited completely. ICE is a conceptual
anchor, not the named V1 baseline: V1 defines no ICE implication teacher. A
future ICE comparison would require a separate reviewed contract freezing
positive, negative, and implication-sample generation and equal information
rights.

## Vocabulary 5: subprogram and macro libraries

### Marginal reasoning gain

This vocabulary tests whether Nous can change its own useful language of
thought. Across a stream of solved synthesis tasks, it must identify recurring
program structure, construct a parameterized executable macro, and demonstrate
that the frozen library reduces later synthesis work after paying the entire
cost of learning, validating, storing, and applying it.

### Objects and bounded semantics

The initial domain uses small typed expression trees or stack programs with:

- primitive operations and type signatures;
- solved task specifications and complete solution programs;
- explicit subtrees or contiguous fragments;
- candidate macros with typed parameters and expansion templates;
- occurrence, substitution, equivalence, and compression evidence; and
- a versioned library whose definitions expand entirely to original
  primitives.

Production semantics may execute one explicit program, expand one macro, and
enumerate the explicit subfragments of one program. It may not scan the solved
corpus for common fragments, optimize the whole library, or synthesize target
programs in Go.

### Discovery loop

After a frozen batch of tasks is solved using primitives, heuristics group
fragments by structural compatibility, propose parameter positions, and
materialize candidate abstractions. Each macro is checked by expansion against
every claimed occurrence. Selection balances definition cost, replacement
savings, parameter cost, type safety, and behavioral equivalence under a frozen
description-length objective.

The library is then frozen. Later synthesis may propose a macro application as
one search choice, but every application records its primitive expansion and
semantic charge. Deleting the macro must either increase later work or remove a
causally used derivation while leaving primitive expressivity intact.

V1 permits a macro to save only proposal and candidate-construction work: one
typed macro proposal can stand for a primitive subtree that the primitive-only
search would otherwise construct through several refinement nodes. It may not
save or conceal execution work. Before evaluation, every macro application is
expanded; every primitive operation, dispatch, semantic evaluation, and
equivalence check is charged exactly as in the inlined control.

### Task stream and controls

Training and evaluation streams contain repeated motifs, recombined motifs,
superficial syntactic repetition without semantic utility, one-off fragments,
and tasks drawn from a different family. Suggested initial programs are typed
tree transformations rather than another two-token rewrite domain, allowing
parameterized subtrees and repeated variables.

Required comparisons are:

- primitive-only synthesis;
- exact complete-program memoization;
- random frequency-matched macros;
- a conventional compression/library-learning baseline;
- Nous with the frozen macro library;
- an inlined equal-expressivity control;
- libraries scored by frequency without downstream use; and
- corrupted expansion, alias, and unrelated-family controls.

The primary endpoint is total charged work over the full declared stream at
equal solved-task count, including acquisition. Secondary endpoints include
post-freeze work, library description length, fraction of solutions causally
using macros, task-family coverage, and the frozen amortization crossover.
The raw ledger separates proposal generation, candidate construction, macro
lookup, expansion, primitive dispatch, primitive execution, behavioral
verification, and terminal selection. The specific plan preregisters the
expected gain only in proposal/construction counts and forbids interpreting an
execution-count tie as macro acceleration.

### Main risks and non-claims

This lane is especially vulnerable to favorable accounting. A macro application
cannot count as one unit of work while its primitive competitor pays for every
step unless the preregistered ledger explicitly and symmetrically charges
expansion and dispatch. The evaluation horizon must be fixed before training;
running until the library becomes profitable is invalid. Compression alone is
not reasoning improvement unless later search uses the abstraction.

### Research anchors

- Ellis and colleagues' [DreamCoder](https://arxiv.org/abs/2006.08381)
  frames expertise as growing an interpretable library of reusable symbolic
  abstractions across tasks.
- Bowers and colleagues' [Stitch library-learning work](https://par.nsf.gov/biblio/10603573)
  treats reusable abstraction discovery as corpus compression and provides a
  strong symbolic conventional baseline.

Nous need not reproduce their scale or neural components. The relevant test is
whether its own first-class heuristics can propose and justify a small useful
macro.

## Vocabulary 6: abductive diagnosis

### Marginal reasoning gain

This vocabulary asks Nous to reason backward from observations to explanations.
Unlike configuration repair, it does not begin with a desired state and search
for edits. It must retain multiple minimal diagnoses when evidence is
insufficient and may rank declared measurements by how they partition those
explanations. V1 is passive and contains no hidden system interaction.

### Objects and bounded semantics

The domain contains:

- components with normal and explicit fault modes;
- deterministic bounded behavior rules;
- observable propositions and measurement costs;
- observations with true, false, or unavailable values;
- conflict sets inconsistent with the all-normal model;
- candidate diagnoses as sets of abnormal assumptions;
- predictions under a diagnosis;
- measurement partitions over surviving diagnoses; and
- fault-signature schemas with scope, support, and counterexamples.

Initial fixtures should be tiny logic circuits, fluid paths, and request-routing
networks expressed through one neutral component model. Production may evaluate
one explicit diagnosis against one explicit observation set and predict one
explicit measurement. It may not compute all conflicts, minimal hitting sets,
or the best measurement.

### Discovery loop

The passive lane materializes candidate abnormality sets by increasing size,
evaluates consistency, retains complete evidence, and selects every subset-
minimal explanation. Heuristics may generalize recurring observation/fault
relations into guarded signature schemas, but signatures never replace final
consistency checking.

Given several surviving diagnoses, ordinary heuristics may also materialize
each declared measurement's predicted partition and rank its offline
discriminating value and cost. The complete component model, observation set,
measurement definitions, and diagnosis-conditioned predictions are public
before the policy starts. No measurement is executed and no new observation is
returned. Ranking is a secondary artifact, not an active-policy claim.

Frozen fault-signature schemas are used only to prioritize legal diagnosis
candidates on later public cases. Every candidate is still checked against the
complete public model and observations. The claimed causal use is reduced
later diagnosis-search work while preserving the exact complete set of
subset-minimal diagnoses.

### Task stream and controls

Training includes single faults, interacting multiple faults, observationally
equivalent diagnoses, irrelevant symptoms, unavailable measurements, and
incorrect but lexically tempting signatures. Held-out tasks rename components,
change topology, recombine learned motifs, and include faults absent from the
signature curriculum but present in the legal model.

Required comparisons are:

- exhaustive minimal diagnosis;
- a conventional conflict/hitting-set diagnosis baseline;
- greedy information gain per measurement cost and random measurement ranking
  as offline recommendation controls;
- Nous without learned signatures;
- Nous with signatures but no final consistency check;
- wrong-context and corrupted-signature controls; and
- an independent exhaustive oracle over the public model, diagnoses, and
  measurement partitions.

Semantic competence requires complete minimal-diagnosis agreement. The primary
marginal endpoint is total lifecycle diagnosis-search work over the frozen
training/use horizon at exact agreement, including signature acquisition,
validation, storage, matching, and final consistency checks. The
equal-expressivity control independently recomputes the same candidate universe.
Offline measurement-ranking agreement, partition work, and recommendation cost
are secondary.

### Main risks and non-claims

Minimal diagnoses are explanations relative to the supplied model, not causal
truth. Non-identifiability and ambiguity are valid terminals only when the
oracle confirms them. The lane's source/dependency audit forbids imports of
`domains/causal`, every `internal/causal*` package, Part 2 fixtures and reports,
and code that reads `.git/nous-attempts`. V1 has no opaque handles, teacher,
query API, response registry, retry, replay, cache shared across policies, or
hidden truth.

Any future active-measurement extension is a different experiment. It would
require a separately accepted information-rights design with a dependency-
isolated driver/oracle, truth-independent opaque handles, per-policy sealed
response state, independent fixture/policy randomness, durable
before-first-read authority, gap-free query/response prefix digests, no retry
or shared cache, and independent transcript replay. Part 3 V1 supplies no such
authority and cannot be described as a simplified retry of Part 2.

### Research anchors

- Reiter's [theory of diagnosis from first principles](https://doi.org/10.1016/0004-3702%2887%2990062-2)
  connects inconsistent observations, minimal diagnoses, and measurements that
  discriminate among competing explanations.

The conventional model-based diagnosis algorithm belongs in the baseline and
oracle. Nous must construct the conflicts, diagnoses, and measurement evidence
through its ordinary artifact path.

## Vocabulary 7: preference and trade-off reasoning

### Marginal reasoning gain

The games pack proved that retaining a Pareto frontier preserves behavioral
alternatives hidden by a scalar score. This vocabulary adds the next step:
learning which nondominated alternative is preferred in a particular context
without pretending that the preference is an objective fact.

### Objects and bounded semantics

The domain represents:

- alternatives with bounded objective vectors;
- objective directions and explicit feasibility constraints;
- dominance evidence and a complete Pareto frontier;
- contexts described by categorical and ordinal features;
- pairwise choices, ties, abstentions, and inconsistent judgments;
- candidate preference rules with guards and partial orders; and
- predictions with supporting comparisons, exceptions, and uncertainty.

Neutral fixtures may use route, build-plan, resource-allocation, and service-
policy alternatives with objectives such as duration, cost, risk,
reversibility, and resource use. Production semantics may compare two vectors,
evaluate one rule on one context/alternative pair, and validate one explicit
choice. It may not compute the frontier, fit a utility model, rank the corpus,
or choose an elicitation query in one builtin.

### Discovery loop

First, the existing evidence-barrier pattern constructs the complete frontier.
Then heuristics propose contextual rules such as lexicographic priorities,
thresholded trade-offs, or guarded pairwise preferences. Rules are evaluated
against visible choices; contradictions remain explicit and may cause rule
specialization, partial ordering, or abstention.

V1 is offline. Every policy receives the identical frozen training comparison
set; it cannot request another choice. The final artifact is a scoped
preference model and its exceptions, never an assertion that one point
dominates another when it does not.

### Task stream and controls

Each fixture first generates a latent context-conditioned utility/partial-order
function from a seed stream independent of alternatives, context order, and
policy randomness. It then freezes alternatives and contexts. The primary
panel is noiseless: each public training comparison is the deterministic latent
relation `left`, `right`, `tie`, or `abstain`. Held-out latent utilities remain
oracle-only until all policies terminate.

A separately labelled sensitivity panel adds inconsistent/noisy choices. Its
response table is generated once from a distinct frozen noise stream indexed
by semantic `(context,left,right)` identity, so every policy sees byte-identical
realized responses regardless of access order. The sensitivity panel cannot
upgrade the primary claim. Training includes stable conditional preferences,
nonlinear thresholds, genuine indifference, and regions with insufficient
evidence. Held-out tasks change alternative identities and objective
magnitudes, include new frontier shapes, and recombine known context features.

Required comparisons are:

- Pareto frontier without selection;
- fixed equal-weight scalarization;
- tuned global scalar weights;
- fixed lexicographic policies;
- a conventional pairwise preference learner;
- Nous with contextual preference rules;
- no-context, wrong-context, forced-total-order, and random-rule ablations.

The primary endpoint is mean normalized held-out latent regret at a fixed
coverage requirement, combined with total lifecycle semantic work over the
frozen training/use horizon. Ties have zero regret for every latent-maximal
choice; an abstention pays its preregistered coverage penalty; forced guesses
do not convert abstention into correctness. Pairwise realized-choice accuracy,
calibration, abstention rate, and noisy sensitivity are secondary. Frontier
recall must remain exact: learning preferences may choose among nondominated
points but may not rewrite objective values or discard a point before evidence
permits it.

### Main risks and non-claims

Preference learning predicts the supplied decision-maker model; it does not
discover moral, organizational, or operational truth. Scalar objectives and
contexts, latent preferences, response noise, and policy randomness use
independent frozen streams. Hidden latent values are used only in
post-termination scoring and independent audit. The report must separate
Pareto competence, latent-utility prediction, realized noisy-choice prediction,
and abstention.

### Research anchors

- Deb and colleagues' [NSGA-II](https://doi.org/10.1109/4235.996017)
  is a canonical reference for maintaining diverse nondominated alternatives.
- Zintgraf and colleagues' [ordered preference elicitation study](https://ifaamas.org/Proceedings/aamas2018/pdfs/p1477.pdf)
  examines how preferences can be elicited over multi-objective policy sets.

These are comparison points. Nous's distinctive evidence would be a small,
inspectable contextual rule with explicit exceptions and causal held-out use.

## Cross-vocabulary controls

Every vocabulary-specific design must include a table showing that it does not
collapse into an earlier result:

| Vocabulary | Must not collapse into |
| --- | --- |
| Constraint learning | recording the explicit constraint that just fired |
| Transformation induction | exact program memoization or human feature keys |
| Partial-order reasoning | comparing final states without using the relation to reduce search |
| Invariant learning | enumerating templates once without counterexample-driven revision |
| Macro libraries | naming a frequent fragment without later causal use |
| Abductive diagnosis | enumerating diagnosis sets without a causally used learned signature |
| Preference reasoning | scalarizing objectives or recomputing Pareto dominance |

In addition, every lane must test:

- alpha-renaming and occupied-name collisions;
- deterministic reruns and store-byte stability at frozen boundaries;
- malformed, ambiguous, no-solution/no-discovery, and irrelevant-feature cases;
- primitive or artifact deletion proving that promoted behavior is materialized;
- independent semantic reimplementation by the oracle;
- exact transcript/accounting reconciliation; and
- adversarial insertion, deletion, corruption, and wrong-context controls.

## Per-vocabulary delivery protocol

Part 3 authorizes no implementation by itself. For each vocabulary:

1. write a vocabulary-specific plan that freezes objects, grammar, task
   generator, information rights, work ledger, panels, baselines, ablations,
   primary endpoint, effect threshold, tie policy, and terminal taxonomy;
2. obtain independent architecture, domain-theory, and experimental-validity
   acceptance of that plan;
3. commit the accepted plan before implementation;
4. implement semantic competence and run development-only integrity trials;
5. obtain adversarial implementation review and correct all blockers;
6. run validation only if the preregistered feasibility gate passes;
7. expose a locked panel only through a guarded one-shot entry and only if
   earlier gates authorize it;
8. preserve the canonical report and classify it `valid-positive`,
   `valid-null`, or `invalid` without retuning; and
9. commit the implementation, evidence, limitations, review verdicts, and
   exact verification results.

The design should remain small enough that a human can inspect a selected
artifact and follow its evidence chain. If a proposed lane requires a large
framework, external service, production integration, or engine change before
its first honest trial, narrow the vocabulary instead.

## Suggested program-level audit

After all attempted lanes, a Part 3 audit should report a capability matrix
rather than one aggregate success score:

| Capability | Required evidence |
| --- | --- |
| Learn from failure | a sound learned nogood causally prunes later search |
| Generalize structure | an induced parameterized schema transfers across alpha-renaming |
| Compress order | a learned guarded relation safely reduces interleavings |
| Revise belief | a counterexample changes an inspectable conjecture lineage |
| Grow a library | a learned macro repays acquisition under the frozen horizon |
| Explain | minimal diagnoses and evidence choices agree with an independent model |
| Choose by context | a scoped preference rule improves held-out choice without corrupting the frontier |

Rows remain `demonstrated`, `not-demonstrated`, `valid-null`, `invalid`, or
`unattempted`. Success in one row cannot upgrade another. The useful outcome of
Part 3 is a clearer map of which reasoning operations Nous can genuinely
acquire, even if none of the vocabularies points toward a single application
destination.

## Reading guide

The papers linked above are research anchors and sources for conventional
baselines. They do not confer empirical support on Nous. The overarching
inspiration remains Lenat's
[EURISKO account](https://doi.org/10.1016/S0004-3702%2883%2980005-8): concepts,
heuristics, credit, and evidence should remain inspectable objects that the
system can reason about. Part 3 tests that idea through seven deliberately
small additions to the system's reasoning vocabulary.
