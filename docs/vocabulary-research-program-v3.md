# Vocabulary research program, Part 3

Status: draft committed for adversarial review.

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
6. construct minimal explanations and choose discriminating evidence; and
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

### One new reasoning operation per vocabulary

Every design must name one distinguishing learned artifact and one causal use:

| Vocabulary | Learned artifact | Required causal use |
| --- | --- | --- |
| Constraint learning | generalized nogood | prune a later legal branch before semantic evaluation |
| Transformation induction | parameterized transformation schema | solve a renamed/recombined transformation task |
| Partial-order reasoning | guarded action relation | omit a redundant interleaving without losing behavior |
| Invariant learning | refined invariant with evidence boundary | reject or classify a held-out trace/state |
| Macro libraries | parameterized executable macro | reduce later synthesis work after full amortization |
| Abductive diagnosis | minimal diagnosis or fault-signature schema | explain observations or select a discriminating measurement |
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
production proposal or stopping point. Hidden answers must use opaque handles
or post-termination audit material. Wrong-context, reset, no-artifact,
corrupted-artifact, and random controls must receive the same public objects and
legal action set.

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

### Evidence status

Each vocabulary ends as:

- `valid-positive`: all mechanical gates pass and the preregistered marginal
  claim passes;
- `valid-null`: all mechanical gates pass but the empirical claim does not;
  or
- `invalid`: a semantic, leakage, accounting, oracle, provenance, or frozen
  protocol gate fails.

Failure to solve, ambiguity, a counterexample, an empty diagnosis set, an
incomplete preference order, or budget exhaustion is normally a valid outcome,
not mechanical invalidity. Reports must distinguish domain hosting, bounded
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
  semantic key.

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
6. retain failed generalizations and counterexamples; and
7. promote a sound, minimal-within-grammar nogood.

On later problems, a separate heuristic matches frozen nogoods against a
partial assignment. A match may prune only the represented branch. The
transcript must record the skipped refinement set and an oracle audit must show
that no solution was removed.

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
requires either exhaustive bounded validation of its quantified scope or a
guard narrow enough that every represented substitution can be checked. A
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

Production semantics may execute one concrete or already-instantiated schema
application. They may expose the syntactic differences between one before/after
pair. They may not compare the entire corpus or compute its anti-unifier.

### Discovery loop

Nous first synthesizes concrete transformations that exactly explain each
training pair. It then proposes generalizations by replacing aligned constants
or paths with shared metavariables, merging compatible edit positions, and
adding equality or role guards when one metavariable appears more than once.

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

### Discovery loop

For a chosen state and action pair, ordinary heuristics execute `a;b` and
`b;a`, preserving inapplicability, intermediate states, and terminal results.
They form unconditional relation conjectures from repeated agreement, then
specialize them with state guards when counterexamples appear. Enabling,
disabling, and conflict claims require asymmetric evidence, not the absence of
a successful swap.

A frozen reduction heuristic may use an accepted guarded commutativity relation
to choose a canonical representative among adjacent swaps. Every omitted
sequence is recorded with its relation, guard witness, representative, and
expansion certificate. A terminal audit expands reduced classes and verifies
behavioral coverage on bounded instances.

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

The primary endpoint is explored histories or semantic transitions needed to
preserve the complete set of terminal behaviors and shortest counterexamples.
Any lost terminal behavior or missed violation mechanically invalidates the
reduction claim. Secondary measures include relation precision, equivalence
class size, guard complexity, and reduction work.

### Main risks and non-claims

Equal final states do not prove safe commutation when intermediate observations,
errors, or enabledness differ. The observation boundary must therefore be
frozen before discovery. This vocabulary differs from the existing protocol
pack: it learns conditional relations among state-changing actions and then
uses them to reduce a search, rather than testing a supplied unary transform
against a supplied protocol relation.

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
- positive states, negative states, transition implications, supporting
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

### Task stream and controls

Trace families include resource lifecycles, request/retry protocols, bounded
queues, leader/follower modes, and intentionally coincidental correlations.
Training omits selected rare modes so validation can expose overgeneralization.
No-nontrivial-invariant, multiple-co-minimal, insufficient-evidence, and
unexpressible-target cases are mandatory.

Required comparisons are:

- fixed template enumeration similar to dynamic invariant detection;
- a conventional ICE-style learner over the same predicate grammar;
- passive correlation without counterexample refinement;
- Nous with the full refinement lineage;
- no-guard and no-negative-evidence ablations;
- shuffled, alpha-renamed, and adversarial rare-mode controls; and
- an exhaustive bounded model/trace oracle.

Primary endpoints combine held-out soundness and useful-property recall at a
frozen complexity bound. A search-advantage claim additionally requires fewer
charged formula evaluations or traces than the strongest equal-grammar
baseline. Reports distinguish observed, bounded-verified, refuted, and
undetermined status.

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
which Nous's refinement lineage can be audited completely.

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
insufficient and, in the active lane, choose a measurement that distinguishes
the remaining explanations.

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

The active lane begins only after passive semantic competence. Given several
surviving diagnoses, ordinary heuristics materialize each legal measurement's
predicted partition, score its discriminating value and cost, choose one, and
recompute the posterior after an opaque fixture returns the observation. Every
query, unavailable result, posterior, and ambiguity terminal is charged.

### Task stream and controls

Training includes single faults, interacting multiple faults, observationally
equivalent diagnoses, irrelevant symptoms, unavailable measurements, and
incorrect but lexically tempting signatures. Held-out tasks rename components,
change topology, recombine learned motifs, and include faults absent from the
signature curriculum but present in the legal model.

Required comparisons are:

- exhaustive minimal diagnosis;
- a conventional conflict/hitting-set diagnosis baseline;
- greedy information gain per measurement cost;
- random legal measurement;
- Nous without learned signatures;
- Nous with signatures but no final consistency check;
- wrong-context and corrupted-signature controls; and
- an independent exhaustive oracle over model, diagnoses, and measurements.

Passive endpoints are complete minimal-diagnosis agreement and charged work.
Active endpoints are terminal identification/ambiguity accuracy, measurement
cost, and total semantic work. A reuse claim charges signature acquisition over
the fixed task horizon and compares it with an equal-expressivity recomputation
control.

### Main risks and non-claims

Minimal diagnoses are explanations relative to the supplied model, not causal
truth. Non-identifiability and ambiguity are valid terminals. Active fixtures
must expose observations only after an irreversible charged query and must not
repeat the elaborate external replay machinery that invalidated Part 2; a tiny
in-process sealed table with independently audited opaque handles is sufficient
for this vocabulary.

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

An optional elicitation lane lets heuristics choose which pair of frontier
alternatives to compare next. The sealed preference fixture returns `left`,
`right`, `tie`, or `unavailable` only after the charged query. The final
artifact is a scoped preference model and its exceptions, never an assertion
that one point dominates another when it does not.

### Task stream and controls

Training contexts exhibit stable conditional preferences, nonlinear thresholds,
genuine indifference, inconsistent/noisy choices, and regions with insufficient
evidence. Held-out tasks change alternative identities and objective magnitudes,
include new frontier shapes, and recombine known context features.

Required comparisons are:

- Pareto frontier without selection;
- fixed equal-weight scalarization;
- tuned global scalar weights;
- fixed lexicographic policies;
- a conventional pairwise preference learner;
- Nous with contextual preference rules;
- no-context, wrong-context, and forced-total-order ablations; and
- random and conventional active-query baselines if elicitation is tested.

Primary endpoints are held-out pairwise choice accuracy or regret against the
sealed declared preferences, with abstention and inconsistency scored under a
frozen rule. Active endpoints add queries and total semantic work. Frontier
recall must remain exact: learning preferences may choose among nondominated
points but may not rewrite objective values or discard a point before evidence
permits it.

### Main risks and non-claims

Preference learning predicts the supplied decision-maker model; it does not
discover moral, organizational, or operational truth. Scalar objectives and
contexts must be generated independently of policy randomness. The report must
separate Pareto competence, preference prediction, and active elicitation.

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
| Abductive diagnosis | enumerating repairs or reading the hidden fault directly |
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
