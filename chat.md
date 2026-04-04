# Conversation Export
*Symbolic Knowledge Layers, Eurisko, Pudl, Mu, CUE DSL*
*April 4, 2026*

---

## Eurisko and Argumentative Hypermedia

**User:** I've been thinking about eurisko and looking at its source code. I've also been thinking about discursive or argumentative hypermedia for agentic/human hybrid systems. Would something like eurisko - even just a modern port of the concept - be useful in such a system as a means of automating the production of symbolic semantic content to be used as grist for the argumentative mill?

**Claude:** This is a rich intersection worth unpacking carefully, because there are a few distinct Eurisko-capabilities that map differently onto what you're describing.

### What Eurisko Actually Offers

Eurisko's core machinery is: **faceted concept representation** + **worth-guided agenda** + **heuristics as first-class objects that rewrite themselves**. The thing that made it genuinely strange is that meta-heuristics could modify object-level heuristics — the generative process was itself subject to the same evaluation/selection pressure as the concepts it produced.

For argumentative hypermedia, these map to meaningfully different affordances:

**Concept invention** (the AM lineage) → generating novel *nodes* in the argument graph: new issue framings, claims, positions, distinctions that no participant has yet articulated. This is where a Eurisko-port is most compelling — LLMs generate *plausible* argumentative content, but not necessarily *novel-relative-to-the-existing-discourse* content in any principled way.

**Heuristic self-modification** → learning which *argument schemas* are productive in this domain. Which warrant types get accepted. Which rebuttal patterns tend to be effective. This is harder to port because Eurisko's heuristic quality degraded badly over time — Lenat attributed this partly to the "brittleness horizon" where meta-heuristics couldn't distinguish genuinely useful mutations from lucky ones.

**Worth estimation** → this is the crux of the problem. In AM, interestingness was partly pre-baked into mathematical elegance criteria. In a discursive system, what's the analogue? Dialectical coverage? Novelty vs. the existing issue map? Human upvote signal? This is where a naive port probably fails.

### The Argumentative Hypermedia Context

You're essentially describing something in the lineage of:

- **IBIS** (Rittel/Webber) — Issue-Based Information System, the original wicked-problem argumentation structure: issues, positions, arguments
- **Compendium/QuestMap** — the IBIS hypermedia instantiation
- **Toulmin schemes + Walton's argument schemes** — typed warrants, backing, rebuttal slots
- Rahwan et al.'s **argument web** work and Dung's abstract argumentation frameworks

The key structural insight is that IBIS nodes (issue, position, argument for/against) are almost exactly Eurisko-style *slots* on a concept. A Eurisko-like system operating over an IBIS vocabulary would be doing: given the current state of the argument map, what new *issue* can I open, what new *position* can I generate for an existing issue, what *argument* (warrant + evidence) can I construct or challenge?

### Where This Gets Interesting

The strongest case for a Eurisko-style system in this context is **structured novelty under dialectical pressure**. The problem with LLM-based argument generation is that it tends toward the *central* distribution of positions on any topic — it generates the expected moves. Eurisko's worth-guided exploration with an explicit novelty bonus was specifically designed to find the *edges* of a concept space.

In a hybrid human/agent argumentative system, you want agents that:
1. Identify **unexplored regions** of the issue map (no one has argued for this position yet)
2. Generate **non-obvious warrants** (connecting claims across domains via analogical heuristics)
3. Detect **structural weaknesses** in the argument graph (claims with no backing, uncontested positions that should have challengers)

Eurisko's agenda mechanism is actually well-suited to (1) and (3). The worth/interestingness metric could be defined over the *argument graph topology* rather than semantic content — find the isolated node, find the uncontested position, find the claim that would maximally bridge two sub-debates.

### What a Modern Port Might Look Like

Rather than porting the Interlisp machinery literally, the interesting synthesis is:

- **LLM as generative substrate** (replacing Interlisp's macro rewriting) — generate candidate nodes given a slot description and heuristic
- **Symbolic argument graph** (IBIS or Toulmin schema) for the slot/facet structure — this is where the actual discourse lives
- **Worth estimation** over graph topology + human feedback signal + novelty metric (embedding distance from existing nodes)
- **Heuristics as prompts + filters** — reified in something like your tuplespace vocabulary, so they're inspectable and modifiable
- **Meta-heuristic layer** as a slower outer loop — periodically evaluating which heuristic-prompts are generating high-worth nodes and adjusting

### The Genuine Risk

Lenat himself noted that Eurisko's productivity was *front-loaded* — the early heuristics were the richest because they were hand-seeded with mathematical taste. The meta-heuristic loop degraded the heuristic quality over time. In an argumentative context, this might manifest as the system learning to generate *formally valid* but *dialectically uninteresting* argument moves — technically novel but not grist the humans actually want to engage with.

The mitigation is probably: keep the worth signal tightly coupled to human engagement (which nodes get responded to, which get ignored), and treat the heuristic inventory as something that needs *human curatorial* intervention periodically, not just self-modification. The system as heuristic co-author rather than autonomous heuristic evolver.

The concept is genuinely worth pursuing. The part of Eurisko that ports cleanly is the agenda + faceted representation + novelty-seeking. The part that doesn't port is the self-modifying heuristics — that needs a human in the loop to avoid quality collapse.

---

## Symbolic Knowledge Layers for Systems Building

**User:** A lot of what I do is around computing and systems building. Managing complexity of systems as they grow, trying to help agentic systems maintain quality output and avoid accelerating the aging of their software output. My intuition tells me that a symbolic superstructure or lattice is a big enabler for this but I feel at a certain point an ontology has to develop and I'm not sure how that happens. Thoughts on how to accelerate the accretion of a symbolic knowledge layer for both humans and machines in this context? How to represent? How to reason over it? How to extend and validate?

**Claude:** This question has several interlocking sub-problems that need to be disentangled, because the answers to "how to represent," "how to reason," and "how to extend/validate" are actually partially in tension with each other, and the choices you make early constrain the others badly.

### The Core Difficulty: Ontologies Don't Bootstrap Gracefully

The historical pattern is grim. CYC tried top-down at scale — enormous human effort producing something perpetually behind the curve of what the system actually needed. The Semantic Web OWL approach tried bottom-up formalization of existing practice — produced a zoo of incompatible ontologies nobody wanted to align. Schema.org succeeded by being deliberately impoverished: enough to be useful for narrow interop, not enough to do real reasoning.

The failure mode you're most at risk of is **premature closure** — committing to categories before you have enough instances to know what the natural kinds actually are. In software systems this is especially bad because the domain is synthetic and self-modifying; the ontology has to track the evolution of the thing it describes.

The failure mode in the other direction — in agentic contexts specifically — is **vocabulary inflation**: agents producing tokens that look like ontological commitments but are actually just naming noise.

### Representation: Stratified, Not Flat

The right structure is layers with different stability properties:

**Upper layer — foundational categories.** Your eight-word moor vocabulary (repo, agent, task, observation, convention, dependency, signal, resource) is a workable upper ontology for your domain. The upper layer should be nearly frozen once seeded.

**Domain layer — typed relationships and constraints.** This is where software-specific concepts live: interface contracts, architectural boundaries, invariants, design rationale, lifecycle states. This layer should encode *why* things are the way they are, not just what they are. An ADR (architecture decision record) is roughly a domain-layer tuple: problem, forces, decision, consequences, status.

**Instance layer — the tuplespace itself.** Anomalies at the instance level (tuples that don't fit existing categories cleanly) are the signal that the domain layer needs extension.

For physical representation: **Datalog over a bitemporal fact store** is probably the right answer. XTDB (formerly Crux) gives you this — immutable, bitemporal, Datalog-queryable. The bitemporality matters: you need to reason about "what did the system believe about itself at time T" to diagnose aging and drift, not just what's true now.

Avoid OWL/RDF for the hot path. Use CUE as the constraint layer of the domain ontology, with Datalog handling the relational/inferential layer above it.

### Reasoning: Datalog + Defeasibility

Datalog over the tuplespace gets you: reachability queries, pattern matching over graph structure, recursive rules for transitive relationships. The gap is **non-monotonic reasoning**. Two options:

**Answer Set Programming (ASP)** — specifically clingo — gives you defeasible reasoning with clean semantics. Use it for reasoning over summaries/abstractions, not raw instance data.

**Defeasible Datalog** — priority-ordered rules where newer/more-specific rules defeat older/more-general ones. Maps well onto convention management.

For software aging specifically, the most powerful reasoning pattern is **abductive**: given observed system state S and expected invariants I, what is the minimal set of violated assumptions that explains the divergence?

### Extension: Formal Concept Analysis as the Engine

**Formal Concept Analysis (FCA)** is the underused tool here. Given a set of objects and their attributes, FCA computes the complete **concept lattice** — all maximally consistent clusters. New categories emerge from the data rather than being stipulated top-down.

The workflow: as instances accumulate, periodically run FCA over the instance layer. The resulting concept lattice reveals natural kinds that the current domain ontology doesn't have names for. These become **ontology extension candidates** that enter the discursive system as issues for deliberation.

This gives you the accretion mechanism: **stigmergic accumulation at the instance layer → FCA-driven candidate generation → deliberative validation → domain-layer commitment**.

### Validation: Invariant Pressure + Instance Fit

**Invariant checking** — the domain ontology makes architectural claims that compile down to Datalog rules running continuously against the instance layer. Violations are signals, not errors.

**Instance fit** — each time a new concept is proposed for the domain layer, check whether it cleanly partitions the instance space. Ambiguous classification signals a bad category.

### The Software Aging Connection

If design rationale and architectural invariants are first-class objects in the domain ontology, an agent can query them before acting: "what conventions constrain the module I'm about to modify? what invariants does my proposed change need to preserve?" The symbolic layer functions as **extended working memory for the system's own architecture**.

---

## Upper Ontologies: BFO, SUMO, DOLCE

**User:** I notice you didn't mention lower ontologies like bfo sumo or dolce. Are those unimportant?

**Claude:** Minor terminological note first: BFO, SUMO, and DOLCE are actually **upper** ontologies (foundational/top-level), not lower. Lower ontologies are domain-specific ones that sit beneath them. Worth being precise because the distinction matters to the question.

### Why I Didn't Mention Them

They're genuinely important in certain contexts, but a poor fit for yours, and committing to one early could be actively harmful.

**BFO** makes a hard ontological commitment to a realist, continuant/occurrent distinction. Software artifacts aren't natural kinds — they're intentional objects whose identity conditions are contested and socially negotiated. BFO's realist commitments don't map cleanly onto "what is a microservice, really."

**DOLCE** is more interesting for your context because it's explicitly **descriptive** rather than realist — it tries to capture how humans conceptualize the world. DOLCE's treatment of **roles** and **social objects** is genuinely relevant — a `convention` in your vocabulary is arguably a DOLCE social object, and DOLCE has machinery for reasoning about that.

**SUMO** is the most comprehensive but also the most unwieldy. Treat it as a reference to check alignment against rather than something to build on.

### The Real Argument Against Adopting One

The interoperability argument for upper ontologies assumes you're trying to **federate with external knowledge sources** that already use them. If you're building a closed or semi-closed system for your own infrastructure, that interoperability tax is pure overhead with no near-term dividend.

The deeper problem is what Nicola Guarino calls the **ontological commitment problem**: when you adopt an upper ontology, you inherit its commitments even where they're silent.

### Where They Do Matter

If your work eventually needs to interoperate with external ontological infrastructure, reason about time and causality rigorously, or ground natural language interpretation at scale — then aligning to DOLCE specifically makes sense for your context. But **align to**, not **build on**.

### The Practical Synthesis

Your eight-word upper vocabulary is already functioning as a minimal upper ontology. The intellectually honest move is to check it against DOLCE's categories explicitly — not to adopt DOLCE wholesale, but to use it as a **stress test**. `task` is probably a perdurant (process). `agent` is probably an endurant with a role. `convention` is a social object with a fuzzy temporal boundary. Working through this against DOLCE's ontological machinery is the philosophy seminar you run once to sharpen the vocabulary, not the library you import into production.

---

## Failures as Ontological Objects

**User:** Thanks for that. Analysing and formalising failures so as to learn from them and integrate them into the knowledge base/lattice is one possible use case so the potential alignment with dolce seems relevant.

**Claude:** Yes, and this is actually where DOLCE's machinery earns its keep most concretely — because failures have a peculiar ontological structure that naive representations get wrong in ways that matter for reasoning.

### Why Failures Are Ontologically Tricky

A failure exists simultaneously at multiple levels that need to be kept distinct:

- **The failure event** — a particular occurrence at a specific time (perdurant in DOLCE terms). This deployment failed at 14:32 UTC.
- **The failure mode** — the type of which this event is a token. Cascading timeout under high fan-out.
- **The failure mechanism** — the causal process that instantiated the mode.
- **The failure effect** — the consequent degraded states that propagated from the event.
- **The latent condition** — the pre-existing structural property that made the system susceptible. This existed *before* the event and is the thing you actually want to learn.

The failure you observe is the token. What you want to accumulate in the knowledge lattice is typed knowledge about the **mode**, the **mechanism**, and the **latent condition** — because that's what lets you reason prospectively.

### The FMEA Connection

The FMEA schema maps reasonably cleanly onto DOLCE categories. Function is a **role** played by a component (DOLCE roles are external — they depend on context). Failure mode is a **type of perdurant**. Failure effect is a **consequent state**. Failure cause is a **generative mechanism**.

The gap in standard FMEA is that it treats RPN as the terminal product — a scalar that prioritizes remediation. It doesn't accumulate structural knowledge about *why* systems become susceptible. That's where integrating FMEA-derived knowledge into a concept lattice changes the game: you start building a library of **susceptibility patterns** as first-class ontological objects.

### Failures as Negative Constraints on the Lattice

FCA builds the concept lattice from positive attribute assignments. But failures give you **negative constraints**: systems with this combination of structural properties reliably fail under these conditions.

The synthesis: run FCA over both the population of **system instances** (with structural properties as attributes) and the population of **failure events** (with mode/mechanism/effect as attributes). The alignment between the two lattices is where the generative learning happens. This is essentially computational epidemiology applied to software systems: find the structural risk factors.

### Provenance as the Connective Tissue

W3C PROV has DOLCE-compatible commitments and provides the **causal chain** representation. PROV's core triad — entity, activity, agent — maps onto DOLCE endurant/perdurant/agent with minor reconciliation effort.

An agent-produced artifact that fails carries information about which agent produced it, under what constraints, with what context — and that's the chain you need to reason about software aging and agentic output quality degradation.

### The Learning Loop

1. **Failure event** enters the tuplespace as an `observation` with structured FMEA-like slots
2. **Abductive reasoning** over the ontology proposes candidate mechanisms and latent conditions
3. **Human deliberation** validates or refutes the abductive hypothesis
4. **Validated failure analysis** gets promoted to the domain layer as a typed **susceptibility pattern**
5. **FCA** periodically runs over accumulated patterns to surface higher-order generalizations
6. **Invariant rules** are derived from validated patterns and enter the continuous checking loop

---

## Bitemporal Datalog in Go with CUE

**User:** Could a bitemporal database with datalog style querying be built in go using cuelang?

**Claude:** Short answer: CUE is a poor fit as the query/reasoning engine, but a genuinely good fit as the schema/constraint layer. The Datalog evaluation needs to be built in Go directly.

### Why CUE Doesn't Work as the Query Layer

**No recursion.** This is by design — CUE is guaranteed to terminate. Datalog's entire power (transitive closure, reachability, fixpoint computation) depends on recursive rule application. You cannot express "all ancestors of X" in CUE.

**Wrong result semantics.** Datalog produces a **set of tuples**. CUE unification produces a **single value**. These are categorically different result shapes.

**Disjunction is open, not enumerating.** CUE's `|` operator expresses a type-level choice, not set union over ground facts.

### Where CUE Earns Its Keep

Schema definition for fact types:

```cue
#BitemporalFact: {
    id:               string & =~"^[a-z]{5}-[a-z]{5}$"
    relation:         string
    args:             [...#Term]
    validTimeStart:   time.Time
    validTimeEnd:     time.Time | *#Infinity
    txTimeStart:      time.Time
    txTimeEnd:        time.Time | *#Infinity
    provenance:       #Provenance
}

#Provenance: {
    agent:    string
    activity: string
    entity:   string
}
```

### The Go Implementation

**Bitemporal Storage Layer** — SQLite as the backing store:

```sql
CREATE TABLE facts (
    id              TEXT PRIMARY KEY,
    relation        TEXT NOT NULL,
    args            TEXT NOT NULL,
    valid_start     INTEGER NOT NULL,
    valid_end       INTEGER,
    tx_start        INTEGER NOT NULL,
    tx_end          INTEGER,
    provenance      TEXT
);
```

Four canonical bitemporal query modes: `AsOfNow()`, `AsOfValid(t)`, `AsOfTransaction(t)`, `AsOf(validT, txT)`.

**Datalog Evaluator** — semi-naive bottom-up evaluation with stratification for correct negation handling:

```go
type Rule struct {
    Head Atom
    Body []Literal
}

type Literal struct {
    Atom    Atom
    Negated bool
}
```

**The Bitemporal-Datalog Integration** — for the failure analysis use case, treat time as a first-class term so rules can reason about temporal overlaps: did the susceptibility condition exist before the failure event? Was the invariant violated during this deployment window?

### End-to-End

```
CUE schemas
    ↓ (validate at ingestion)
Bitemporal SQLite fact store
    ↓ (temporal view selection)
Go Datalog EDB
    ↓ (semi-naive evaluation over stratified rules)
Go Datalog IDB (derived facts)
    ↓ (abductive query)
Candidate explanations / susceptibility patterns
    ↓ (promote validated patterns)
Domain-layer CUE schema extension
```

The rules themselves live in the bitemporal store too — temporally situated, auditable, replayable. There's no mature Go library that gives you all of this — it would genuinely need to be built.

---

## EDB and IDB

**User:** What are edb and idb?

**Claude:** Short answer: they're Datalog terminology for the two kinds of facts in a program.

**EDB — Extensional Database.** The base facts. Things that are true by assertion, not by derivation. In your context: every fact that mu pushes into the bitemporal store is EDB. "This artifact was produced by this action at this time." "This service depends on this component." "This deployment failed."

**IDB — Intensional Database.** The derived facts. Things that are true because Datalog rules computed them from EDB. "This service is transitively dependent on that library" (computed via recursive reachability). "This component has no circuit breaker convention" (computed by absence). "This cluster of services shares a susceptibility pattern" (computed by joining failure events against structural facts).

The distinction matters because:
- EDB is what you store and maintain. IDB is recomputed on demand — it's a view, not ground truth.
- EDB facts can be retracted. IDB facts just stop being derivable when their supporting EDB changes.
- The richer your rules, the more you get from IDB without having to assert it explicitly.

In practice: pudl-validated, mu-produced facts land in the bitemporal store as EDB. Everything the Datalog evaluator derives — the inferences, the patterns, the invariant violations — is IDB.

---

## Pudl and Mu

**User:** Okay forget moor here. I'm concerned right now with pudl and mu. There is no tuplespace.

**Claude:** Let me search past conversations for context on pudl and mu...

**User:** It's a schema store using cue. It defines types and ingests data about instances of those types. Where mu executes, pudl defines and validates.

**Claude:** Good, that clarifies the architecture considerably. Now the picture is:

- **pudl** — the schema/type layer. CUE's value lattice gives you the structural ontology: every type defined in pudl is an ontological commitment about what a valid instance looks like.
- **mu** — the execution layer. It produces events, artifacts, action results — the raw instance data that pudl's schemas type-check.

### What Pudl Already Gives You

Pudl's CUE type definitions are already doing something ontologically significant — the CUE lattice is a genuine lattice in the mathematical sense, with meet (conjunction of constraints) and join (least upper bound). Every schema in pudl implicitly defines a concept with subsumption relationships. A type that embeds another is a subtype in the ontological sense.

Pudl is already your **domain layer** from the stratified ontology discussion. The schemas define the vocabulary; the validation enforces it at instance ingestion time.

### What's Actually Missing

**Temporality.** Mu's artifact graph is content-addressed and monotonic but there's no first-class notion of *when* something was true.

**Inference.** CUE can tell you whether an instance satisfies a schema. It cannot derive new facts from existing ones.

**Cross-instance reasoning.** Pudl reasons about one instance at a time against a schema. Datalog reasons over the population of all instances simultaneously.

### The Architecture

```
pudl (CUE schemas)
    ↓  type definitions flow down as EDB schema
    ↓  instances validated before ingestion

bitemporal SQLite fact store
    ← mu pushes artifact/action/failure events here, validated by pudl schemas
    ← human assertions pushed here too

Datalog evaluator (Go)
    ← reads EDB from bitemporal store at chosen temporal view
    → derives IDB: susceptibility patterns, invariant violations, dependency closures

rules (also in the store, temporally situated)
    ← promoted from validated failure analyses
    ← encode architectural invariants derived from pudl schemas
```

The key integration point: **pudl schemas compile to EDB shape constraints**. You generate the SQL schema and Datalog relation declarations from pudl types, keeping the two in sync automatically.

### The Interesting Tension

Pudl's type lattice and a Datalog-derived concept lattice (FCA over instances) are both lattices but orthogonal. Pudl's lattice is over *constraints*. An FCA concept lattice is over *extensions*. Their conflicts are informative — a pudl type that seemed like a natural kind turns out to split into two FCA concepts, signalling the type needs refinement.

The full loop: pudl defines → mu produces instances → bitemporal store accumulates → Datalog infers → FCA surfaces new candidates → pudl types get refined.

---

## Eurisko Adaptation with CUE and Stack DSL

**User:** I've also been playing with an adaptation of eurisko. It uses cue as data representation layer and a stack based dsl for heuristics. How could such a system fit in here?

**Claude:** This fits remarkably cleanly — and it resolves a gap I left implicit in the earlier architecture.

### What Your Eurisko Adaptation Is Doing

CUE as the concept representation layer is a natural fit because Eurisko's slot/facet model maps directly onto CUE structs — a concept is a CUE value with named fields for each facet, and the CUE lattice gives you subsumption between concepts for free.

The stack-based DSL for heuristics makes heuristics **inspectable and serializable** — a heuristic is a value you can store, transmit, and reason about, not just a function pointer. That's crucial for the self-modification property and for putting heuristics in the bitemporal store with full temporal provenance.

### Where It Fits In The Stack

```
Eurisko adaptation
    ↓ reads concept definitions from pudl schemas (EDB shape)
    ↓ reads derived facts from Datalog IDB (what's currently true)
    ↓ runs heuristics against the agenda
    ↓ produces candidate concepts, rules, and type refinements

pudl  ←── receives validated concept promotions

bitemporal store  ←── receives new EDB assertions from heuristic firing

Datalog  ←── new rules promoted from validated heuristic output
```

The Eurisko layer sits **between** the Datalog IDB and the pudl schema layer. It reads derived facts upward from Datalog and pushes validated novelty upward into pudl and back down into the rule base.

### The Three Loops

**Fast loop — Datalog inference.** Runs continuously over EDB as mu pushes new facts. Derives IDB: invariant violations, dependency closures, susceptibility pattern matches. Reactive and deterministic.

**Medium loop — Eurisko agenda.** Runs periodically, driven by worth estimates over the current IDB state. Fires heuristics against high-worth agenda items. Produces candidate concepts, rules, type refinements. Generative and exploratory.

**Slow loop — deliberative validation.** Candidates from Eurisko enter the human-review layer. Validated candidates get promoted: new pudl types, new Datalog rules, new EDB conventions. Human judgment gates what accretes into the stable knowledge base.

The timescale separation is critical — Eurisko must not directly modify the rule base without the slow loop. That's how Lenat's heuristic quality degradation occurs.

### The Stack DSL as Heuristic Bytecode

The stack-based DSL makes heuristics **first-class data objects** that can be the subject of Datalog queries. You can ask: which heuristics have fired most often against high-worth items? Which have produced candidates that survived the slow loop? Which are generating noise?

This sidesteps Lenat's degradation problem because heuristic modification is mediated by the same validation loop as concept promotion — not by direct self-rewriting.

### The CUE Coherence

There's an elegant coherence in using CUE throughout:

- Pudl schemas are CUE — structural constraints on instance types
- Eurisko concepts are CUE values — slot/facet representations of domain knowledge
- Heuristic programs are CUE values — serialized stack DSL bytecode

A concept produced by Eurisko can be directly checked against pudl schemas for structural validity before promotion. The meet operation in CUE is how you detect that a new candidate concept is a refinement of an existing pudl type rather than genuinely novel.

### The Failure Analysis Connection

For software aging and failure learning, the Eurisko layer adds something neither Datalog nor pudl can do alone: **analogical generalization across failure instances**.

A heuristic that says "when two failure events share a mechanism and their antecedent structural conditions overlap by more than N fields, generate a candidate susceptibility pattern" is doing analogical reasoning over the IDB. The output is a candidate concept in CUE — a new susceptibility pattern type to be validated and potentially promoted into pudl.

That's the accretion mechanism made concrete: **Eurisko finds the pattern, CUE represents it, Datalog makes it inferentially active, pudl stabilizes it as a type.** Each layer does exactly what it's suited for.

---

## Rules vs. Heuristics

**User:** Are rules in this design the same as heuristics? No, right?

**Claude:** Right, they're distinct — and the distinction matters.

### Rules

Rules are **deterministic inference machinery**. They derive facts from other facts with no uncertainty and no judgment involved. Given the EDB, a rule either fires or it doesn't. There's no notion of "this rule is more or less applicable" — if the body is satisfied, the head is derived.

```cue
transitiveDep: #Rule & {
    head: { rel: "depends_transitive", args: { from: "$X", to: "$Z" } }
    body: [
        { rel: "depends",            args: { from: "$X", to: "$Y" } },
        { rel: "depends_transitive", args: { from: "$Y", to: "$Z" } },
    ]
}
```

This either applies or it doesn't. No weight, no threshold, no context-sensitivity.

### Heuristics

Heuristics are **context-sensitive, worth-weighted suggestions**. They don't derive facts — they propose actions on the agenda. They can fire on the same inputs and produce different outputs depending on context. They have a worth score that determines whether they're even tried. They can fail to produce anything useful and that's fine — it just affects their worth.

A heuristic isn't "if body then head." It's closer to "given this pattern in the IDB, here is a candidate worth considering, with this confidence, for this reason."

### The Crisp Boundary

| | Rules | Heuristics |
|---|---|---|
| Output | Derived facts (IDB) | Agenda candidates |
| Firing | Deterministic | Worth-weighted, context-sensitive |
| Representation | `#Rule` CUE struct | Stack DSL program |
| Evaluated by | Datalog engine | Eurisko runtime |
| Validated by | Pudl schema + stratification check | Slow loop + worth tracking |
| Live in | Bitemporal store as `#Rule` values | Bitemporal store as stack bytecode values |

Both live in the bitemporal store with temporal provenance. Both can be produced by the Eurisko layer and promoted through the slow validation loop. But what they produce is categorically different — rules extend the IDB deterministically, heuristics extend the agenda speculatively.

The Eurisko layer reads the IDB (produced by rules) and fires heuristics against it. The output of heuristics, when validated, can include new rules — but a heuristic is never itself a rule. That's the layering.

---

## CUE as Datalog Surface Syntax

**User:** Ok so I asked earlier about cue being the query language. I meant syntax not evaluation. How could we use cue for datalog like querying and representation?

**Claude:** Good clarification — that's a much more tractable and interesting question. Using CUE as the *surface syntax* for a Datalog implementation is a legitimate design choice with real advantages.

### What You're Actually Proposing

CUE becomes the representation language for facts (EDB assertions), rules (inference programs), and queries (patterns to match against the IDB). A Go Datalog engine evaluates them. CUE never evaluates anything — it just structures the data that the evaluator reads.

### Facts as CUE Values

```cue
#Depends: {
    from: string
    to:   string
    kind: "runtime" | "compile" | "optional"
}

edb: {
    service: [
        { name: "api",  tier: "edge" },
        { name: "auth", tier: "core" },
    ]
    depends: [
        { from: "api",  to: "auth", kind: "runtime" },
        { from: "auth", to: "db",   kind: "runtime" },
    ]
}
```

### The Variable Convention

Variables are strings prefixed with `$` — easy to parse, no collision with CUE semantics:

```cue
rules: {
    transitiveDep: {
        head: { rel: "depends_transitive", from: "$X", to: "$Z" }
        body: [
            { rel: "depends",            from: "$X", to: "$Y" },
            { rel: "depends_transitive", from: "$Y", to: "$Z" },
        ]
    }
}
```

### Rules as CUE Values

The full rule schema in pudl:

```cue
#Var:  string & =~"^\\$[A-Z][A-Za-z0-9]*$"
#Term: string | #Var

#Atom: {
    rel:  string
    args: { [string]: #Term }
}

#Literal: #Atom & {
    negated: bool | *false
}

#Rule: {
    head: #Atom
    body: [...#Literal]
}
```

Named args rather than positional because CUE structs are named — more idiomatic and readable:

```cue
rules: {
    transitiveDep: #Rule & {
        head: { rel: "depends_transitive", args: { from: "$X", to: "$Z" } }
        body: [
            { rel: "depends",            args: { from: "$X", to: "$Y" } },
            { rel: "depends_transitive", args: { from: "$Y", to: "$Z" } },
        ]
    }

    uncoveredDep: #Rule & {
        head: { rel: "at_risk", args: { service: "$S", dependency: "$D" } }
        body: [
            { rel: "depends",    args: { from: "$S", to: "$D" } },
            { rel: "convention", args: { covers: "$D" }, negated: true },
        ]
    }
}
```

### Queries as CUE Values

CUE's `_` (top — matches anything) works naturally as an anonymous wildcard:

```cue
// What does "api" transitively depend on?
query: {
    rel:  "depends_transitive"
    args: { from: "api", to: _ }
}
```

Results are themselves valid CUE values, validatable by pudl schemas:

```cue
[
    { service: "api", dependency: "db" },
    { service: "auth", dependency: "db" },
]
```

### Temporal Queries

```cue
#TemporalQuery: {
    rel:     string
    args:    { [string]: #Term | _ }
    validAt: time.Time | *"now"
    asOf:    time.Time | *"now"
}

query: #TemporalQuery & {
    rel:     "depends_transitive"
    args:    { from: "api", to: _ }
    validAt: "2026-01-01T00:00:00Z"
    asOf:    "2026-02-01T00:00:00Z"
}
```

### What This Buys You

**Unified tooling.** Everything in the system — schemas, facts, rules, queries, results — is CUE. One formatter, one linter, one IDE integration.

**Rule validation at write time.** Pudl's `#Rule` schema catches malformed rules before they reach the evaluator.

**Rules as storable values.** A rule is a CUE struct. It lives in the bitemporal store with full temporal provenance.

**Eurisko heuristics as rules.** When the Eurisko layer produces a candidate rule, it produces a `#Rule` CUE value. That value enters the slow validation loop exactly like any other candidate. The whole pipeline is homogeneous.

---

## Heuristics in CUE

**User:** Now that everything but the heuristics is in cue do you think it makes sense to put them in cue as well?

**Claude:** The honest answer is: partially, and the boundary is meaningful.

### What CUE Can Usefully Represent

A heuristic has two distinct parts — its **structure** (metadata, preconditions, output shape, worth parameters) and its **execution logic** (the actual stack program). These have different relationships to CUE.

The structural part benefits from CUE validation:

```cue
#Heuristic: {
    name:  string
    worth: number & >=0 & <=1
    produces: "agenda-item" | "rule-candidate" | "type-candidate" | "fact"

    // Preconditions are IDB patterns — same shape as rule body literals
    preconditions: [...#Literal]

    // Stack program — typed but opaque to CUE's semantics
    program: [...#StackOp]
}
```

The preconditions especially make sense in CUE — they're the same `#Literal` type as rule body atoms. CUE validating their structure catches malformed patterns at write time.

### Where CUE Stops Helping

The stack program itself is where CUE's value proposition runs out. CUE unification has no concept of sequencing, conditional branching, stack manipulation, or iteration. You can represent the bytecode as a CUE list of typed opcodes:

```cue
#StackOp: {
    op:  "push" | "pop" | "match" | "score" | "emit" | "branch" | ...
    arg?: _
}
```

This makes the program a valid CUE value — storable, inspectable, toolable — but CUE's type system isn't doing meaningful work on it beyond basic structural validation. You're using CUE as a container, not as a reasoning layer.

### The Right Synthesis

Represent heuristics as CUE structs where CUE adds genuine value, and embed the stack program as a typed but semantically opaque field. The structural/metadata layer is fully CUE-native. The execution layer is CUE-contained but Go-interpreted.

---

## DSL Vocabulary Extension and CUE Word Definitions

**User:** For the eurisko layer, part of the challenge is going to be finding ways to model new domains. I suspect this will mean creating new program words for the heuristic programs. Is this intuition correct? Additionally, could we use cue to hold definitions of new dsl words?

**Claude:** Your intuition is correct, and it maps precisely onto something Eurisko itself did — Lenat called it **concept-driven vocabulary extension**. When AM/Eurisko moved into a new domain, the heuristics operating there would discover that existing operations were insufficient and propose new primitive operations as concepts in their own right.

### Why New Words Are Necessary

A stack DSL for heuristics starts with a kernel of domain-agnostic operations. But when the Eurisko layer encounters a new domain, the kernel operations become compositionally expensive. At some point a recurring composition deserves a name. That name is a new word.

Named words are:
- **The unit of worth tracking** — you can observe that this word appears in high-worth heuristics and that one doesn't
- **The unit of analogical transfer** — does this word from the failure-analysis domain have an analogue in the dependency-graph domain?
- **The unit of meta-heuristic operation** — a meta-heuristic that rewrites heuristic programs operates on words, not raw bytecode sequences

### CUE for Word Definitions

```cue
#StackWord: {
    name:   string
    domain: string
    worth:  number & >=0 & <=1

    // Stack effect declaration — what the word consumes and produces
    // CUE validates this interface contract
    effect: #StackEffect

    // Preconditions on stack state before the word can execute
    requires: [...#StackTypeConstraint]

    // Implementation — primitive (Go runtime) or composite (other words)
    impl: #Primitive | #Composite
}

#Primitive: {
    kind:    "primitive"
    handler: string  // name of Go function to dispatch to
}

#Composite: {
    kind:    "composite"
    program: [...#StackOp]
}

#StackEffect: {
    consumes: [...string]
    produces: [...string]
}
```

The **stack effect declaration** is where CUE earns its keep specifically. Stack effect typing is a real type system — it's what Forth and Factor use to reason about program correctness. CUE can validate that a composite word's program is stack-effect consistent.

### Composite Words as the Bridge

When the Eurisko layer discovers a recurring composition in high-worth heuristics, a meta-heuristic can propose abstracting it into a named word:

```cue
structural_risk_match: #StackWord & {
    name:   "structural-risk-match"
    domain: "failure-analysis"
    worth:  0.0  // provisional — unproven
    effect: {
        consumes: ["failure_event", "threshold"]
        produces: ["scored_candidate"]
    }
    impl: #Composite & {
        program: [
            { op: "match_structural_overlap", arg: "$threshold" },
            { op: "filter_by_mechanism" },
            { op: "score_by_severity" },
        ]
    }
}
```

This candidate enters the slow validation loop exactly like a rule candidate or type candidate.

### The Dictionary as a Bitemporal Object

The word store is itself bitemporal — you can ask "what words existed at time T" and replay heuristic evaluation using exactly the vocabulary available then. Worth tracking on words enables **domain atrophy detection**: if a domain's words steadily decline in worth, that's a signal the domain model is becoming stale relative to the actual system state.

### The Full Vocabulary Picture

- **Kernel words** — primitive, Go-implemented, domain-agnostic, stable
- **Domain words** — composite, CUE-defined, introduced as the system encounters new domains, worth-tracked
- **Meta-heuristics** — watch for recurring compositions in high-worth programs, propose new composite words
- **Slow loop** — validates word candidates before promotion
- **Bitemporal store** — holds the full word dictionary with temporal provenance, queryable by Datalog

---

## Custom CUE Functions via the Go API

**User:** We can also define custom cue functions using the go API.

**Claude:** This changes the design meaningfully — and closes the gap I was treating as unavoidable.

### What It Actually Means

The primitive/composite distinction was predicated on primitives being opaque to CUE — Go handlers that CUE could only reference by name. If you can register Go functions as CUE builtins via the API, that boundary collapses. Primitives become **callable within CUE expressions**, not just named in CUE structs.

Which means composite words don't need to be stack programs at all. They can be **CUE expressions** that call registered builtins directly:

```cue
// Composite word — now a genuine CUE expression, not opaque bytecode
structural_risk_match: #StackWord & {
    name:   "structural-risk-match"
    domain: "failure-analysis"
    impl: #Composite & {
        expr: matchStructuralOverlap & filterByMechanism & scoreBySeverity
    }
}
```

### What This Unlocks

**Stack effect validation becomes native.** If the composite is a CUE expression, CUE's own unification checks consistency at definition time — before the program ever runs. A composite that tries to call a builtin with the wrong input type fails CUE validation, not runtime execution.

**CUE's lattice applies to word composition.** Two composite words that overlap in their expressions have a meet — the most specific composite that satisfies both. Detecting that two heuristics are doing similar things is now a CUE unification operation, not a semantic comparison of bytecode sequences.

**Meta-heuristics generate CUE, not bytecode.** When a meta-heuristic identifies a recurring composition and proposes a new word, it produces a CUE expression as the word definition — immediately validatable, immediately inspectable, immediately composable.

**The dictionary becomes a CUE package.** If words are CUE builtins plus CUE expressions over those builtins, the entire vocabulary for a domain is a CUE package — importable, versionable, with all of CUE's tooling applying to it. Domain transfer becomes a question about whether a word from one package unifies with a definition in another.

### The Revised Word Schema

```cue
#Primitive: {
    kind:    "primitive"
    builtin: string
    effect:  #StackEffect
}

#Composite: {
    kind:   "composite"
    expr:   _  // typed by the effect declaration
    effect: #StackEffect
}
```

### The Remaining Question

The one thing to think through carefully is whether CUE's evaluation model — convergent, terminating, no side effects — is a constraint on what composite words can express. Heuristics need to produce candidates speculatively, score them, conditionally emit them.

The boundary is probably: **word definitions in CUE, word execution mediated by Go**. CUE defines what a word is and validates its composition with other words. The Go runtime calls the registered builtins in the right order, maintaining the stack state that CUE's model can't directly represent. CUE does the static reasoning; Go does the dynamic execution.

But that's a much thinner Go layer than the full stack interpreter imagined earlier — and a much richer CUE layer than "container for opaque bytecode."

---

*End of conversation export*
