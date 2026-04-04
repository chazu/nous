# Code Transformation Patterns Domain

## Motivation

Refactoring is compositional. Extract function, inline, rename, move, introduce parameter, encapsulate field — these are operations on code structure with computable effects on quality metrics. An agent performing refactoring could use nous-discovered patterns: which transform sequences improve which metrics, which combinations are identity (undo each other), which sequences are equivalent to simpler transforms.

This is the software architecture domain but at the AST level — operating on actual code structure patterns rather than abstract module diagrams.

## Why this fits Mode 1

- Code structures are representable as trees/graphs (ASTs, call graphs, dependency graphs)
- Transforms are well-defined operations on these structures
- Quality metrics are computable: function length, parameter count, coupling, cyclomatic complexity, duplication
- Transform composition is meaningful: Extract then Move = ExtractToModule
- Equivalence is checkable: do two transform sequences produce the same structure?

## Concept space

### Core types

- **CodeUnit** — an abstract representation of code structure. Not actual source code — a structural skeleton.
  - Functions: name, parameter count, body complexity, return type
  - Types: name, field count, method count, visibility
  - Modules: name, function count, type count, import count, export count

- **CallGraph** — who calls whom. Edges are function-to-function dependencies.

- **Transform** — a named refactoring operation with preconditions and effects.

- **TransformSequence** — an ordered list of transforms applied to a code unit.

- **CodeMetrics** — computed quality measurements.

### Data representation

A CodeUnit as a unit:
```
name: "UserService"
data: {
    functions: [
        {name: "CreateUser",   params: 3, complexity: 5, lines: 20, calls: ["ValidateEmail", "HashPassword", "InsertDB"]},
        {name: "GetUser",      params: 1, complexity: 2, lines: 8,  calls: ["QueryDB"]},
        {name: "UpdateUser",   params: 3, complexity: 7, lines: 30, calls: ["ValidateEmail", "GetUser", "UpdateDB"]},
        {name: "DeleteUser",   params: 1, complexity: 3, lines: 12, calls: ["GetUser", "DeleteDB", "ClearCache"]},
        {name: "ValidateEmail", params: 1, complexity: 4, lines: 15, calls: []},
        {name: "HashPassword", params: 1, complexity: 3, lines: 10, calls: []},
        {name: "InsertDB",     params: 2, complexity: 2, lines: 8,  calls: []},
        {name: "QueryDB",      params: 1, complexity: 2, lines: 8,  calls: []},
        {name: "UpdateDB",     params: 2, complexity: 2, lines: 8,  calls: []},
        {name: "DeleteDB",     params: 1, complexity: 2, lines: 8,  calls: []},
        {name: "ClearCache",   params: 1, complexity: 1, lines: 5,  calls: []},
    ],
}
metrics: {
    functionCount: 11,
    avgComplexity: 3.0,
    maxComplexity: 7,
    avgParams: 1.5,
    maxParams: 3,
    avgLines: 12,
    totalLines: 132,
    callGraphDepth: 2,
    callGraphFanOut: 1.7,
    cohesion: 0.6,
    couplingInternal: 8,
}
```

### Transforms

**Function-level:**
- **ExtractFunction** — take a section of a function body, move it to a new function, replace with a call. Reduces complexity of the original, increases function count.
- **InlineFunction** — replace a call with the function body. Inverse of Extract. Reduces function count, increases complexity.
- **IntroduceParameter** — replace a hardcoded value in a function with a parameter. Increases param count, increases reusability.
- **RemoveParameter** — remove an unused or derivable parameter. Decreases param count.
- **MoveFunction** — move a function from one module to another. Changes coupling.

**Type-level:**
- **ExtractType** — pull fields out of a type into a new type. Reduces field count, adds a type.
- **InlineType** — merge a type's fields into its only user. Inverse of ExtractType.
- **EncapsulateField** — make a field private, add getter/setter. Increases method count, decreases coupling.
- **IntroduceInterface** — extract an interface from a concrete type. Adds a type, enables substitution.

**Module-level:**
- **ExtractModule** — move a set of related functions/types into a new module. Increases module count, may reduce coupling.
- **MergeModules** — combine two modules. Inverse of ExtractModule.
- **InvertDependency** — extract an interface so that the dependency direction reverses. Changes coupling topology.

### Quality metrics (computable from CodeUnit structure)

- `functionCount` — number of functions
- `avgComplexity` — average cyclomatic complexity
- `maxComplexity` — highest complexity function (hotspot)
- `avgParams` — average parameter count
- `maxParams` — highest parameter count (smell: too many params)
- `totalLines` — total lines of code
- `callGraphDepth` — deepest call chain
- `callGraphFanOut` — average outgoing calls per function
- `cohesion` — ratio of internal calls to total possible internal calls
- `duplication` — number of function pairs with similar structure
- `interfaceCount` — number of abstract interfaces
- `couplingScore` — sum of inter-module dependencies

### What the system could discover

**Transform equivalences:**
- `Extract(f, section) + Move(newFn, moduleB)` = `ExtractToModule(f, section, moduleB)` — a compound transform is equivalent to a simpler named one.
- `Inline(f) + Extract(caller, section)` ≈ `Move(section, caller)` in simple cases.

**Transform anti-patterns:**
- `Extract + Inline` on the same section is identity — wasted effort. The system creates an avoidance rule.
- Extracting a function that's only called once increases function count without reducing complexity. The system discovers this has negative value.

**Metric correlations:**
- "Functions with complexity > 10 always have params > 3" — the system discovers that high complexity and high parameter count co-occur, suggesting that IntroduceParameter makes complex functions worse, not better.

**Optimal sequences:**
- For a function with complexity=15 and params=5, the best sequence is: ExtractFunction (split the complex body), then RemoveParameter on the original (the extracted part took some params with it). The system discovers this two-step pattern.

**Refactoring priorities:**
- "Reducing maxComplexity has more impact on overall quality than reducing avgComplexity." The system discovers that focusing transforms on the worst function is always better than spreading them evenly.

### DSL builtins needed

- `code-unit-new` — create from function/type/module descriptions
- `code-extract-function`, `code-inline-function`
- `code-introduce-param`, `code-remove-param`
- `code-move-function`, `code-extract-module`, `code-merge-modules`
- `code-extract-type`, `code-introduce-interface`
- `code-complexity`, `code-function-count`, `code-coupling`
- `code-call-graph-depth`, `code-call-graph-fan-out`
- `code-equivalent?` — do two code units have the same call graph and metric profile?
- `code-metrics` — compute all metrics for a code unit

### Implementation complexity

**Moderate.** The code structure representation is the main challenge — functions with call graphs, modules with dependencies. The transforms need to maintain structural consistency (can't move a function that's called from the source module without updating the call graph). The metrics are straightforward computations on the structure.

The key simplification: we're not operating on real code. We're operating on structural skeletons — function stubs with parameter counts and call lists. No parsing, no type checking, no actual ASTs. Just the structural properties that transforms affect.

### Connection to agentic coding

This domain is the most directly applicable to what coding agents do:

1. **Refactoring suggestions:** Given a codebase profile (extracted by pudl), nous suggests transform sequences ranked by impact.
2. **Transform planning:** "To reduce complexity from 15 to 5, apply these 3 transforms in this order."
3. **Anti-pattern detection:** Discovered avoidance rules flag transform sequences that are known to be counterproductive.
4. **Quality prediction:** "Applying ExtractModule here will reduce coupling by ~30% based on discovered patterns."

### Seed content

- 2-3 example code units: a simple service (5 functions), a complex module (12 functions with high coupling), a well-structured module (8 functions with high cohesion)
- Basic transforms: ExtractFunction, InlineFunction, MoveFunction, IntroduceParameter
- Module-level transforms: ExtractModule, InvertDependency
- Known patterns as seed conjectures: "Extract reduces complexity," "Inline reduces function count but increases complexity"
