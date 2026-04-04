# Formal Grammars and Automata Domain

## Why this fits Mode 1

Regular languages are closed under composition. Every operation produces a concrete, checkable result. Equivalence is decidable. The concept space has deep structure with known theorems the system could rediscover, plus room for non-obvious discoveries.

## Concept space

### Core types

- **Regex** — a regular expression. Data slot is the pattern string or an AST.
- **DFA** — deterministic finite automaton. Data slot is a transition table (states × alphabet → state, plus accept set).
- **NFA** — nondeterministic finite automaton. Same but with sets of target states and epsilon transitions.
- **Language** — a finite sample of accepted/rejected strings. Used for empirical equivalence testing.
- **RegexOp** — an operation on regular expressions (unary or binary).
- **AutomatonOp** — an operation on automata.
- **LanguageProperty** — a predicate on languages (empty, finite, infinite, contains-epsilon, etc.)

### Data representation

A DFA as a unit:
```
name: "DFA-abc-star"
data: {
    states: [0, 1, 2, 3],
    alphabet: ["a", "b", "c"],
    transitions: {
        "0,a": 1, "0,b": 0, "0,c": 0,
        "1,a": 1, "1,b": 2, "1,c": 0,
        "2,a": 1, "2,b": 0, "2,c": 3,
        "3,a": 3, "3,b": 3, "3,c": 3,
    },
    start: 0,
    accept: [3]
}
```

A Language (finite sample) as a unit:
```
name: "L-abc-star"
data: {
    accepts: ["abc", "abcabc", "aabcbc", ...],
    rejects: ["ab", "ac", "abcab", ...],
}
```

For the DSL, these would be stored as nested maps or as specialized data structures with dedicated builtins to manipulate them.

### Operations

**On regular expressions:**
- **Union** (r1 | r2) — binary, produces a regex matching either language
- **Concat** (r1 r2) — binary, produces a regex matching r1 followed by r2
- **Star** (r*) — unary, zero or more repetitions
- **Plus** (r+) — unary, one or more repetitions
- **Optional** (r?) — unary, zero or one
- **Complement** (¬r) — unary, everything the original doesn't match
- **Intersection** (r1 & r2) — binary, strings matched by both
- **Reverse** (r^R) — unary, reverse all strings in the language

**On automata:**
- **Determinize** (NFA → DFA) — subset construction
- **Minimize** (DFA → minimal DFA) — Hopcroft's or Brzozowski's algorithm
- **Complement** (DFA → DFA) — flip accept/reject states
- **Product** (DFA × DFA → DFA) — for intersection/union via product construction
- **Reverse** (DFA → NFA) — reverse all transitions, swap start/accept

**Conversions:**
- **RegexToDFA** — Thompson's construction + determinization + minimization
- **DFAToRegex** — state elimination

### Quality attributes / evaluable properties

- `stateCount` — number of states (fewer is more elegant)
- `isMinimal` — is this DFA already minimal?
- `languageSize` — finite, infinite, or empty
- `equivalentTo` — does this accept the same language as another automaton?
- `subsumes` — does this language contain another?
- `alphabetSize` — how many symbols
- `acceptRatio` — on random strings of length ≤ N, what fraction are accepted?

### What the system could discover

**Known theorems to rediscover:**
- Every NFA has an equivalent DFA (but possibly exponentially larger)
- Every DFA has a unique minimal equivalent
- Regular languages are closed under union, intersection, complement, concatenation, star
- (r*)* = r* — star is idempotent
- (r1 | r2)* ⊇ r1* — star of union contains star of either part
- Reverse(Complement(r)) ≠ Complement(Reverse(r)) in general
- Minimize(Determinize(Reverse(Minimize(Determinize(Reverse(DFA)))))) = Minimize(DFA) — Brzozowski's algorithm

**Non-obvious discoveries:**
- Which regex transformations preserve minimality?
- Which compositions of operations are equivalent to simpler operations?
- Families of regexes where complementation doesn't blow up state count
- Empirical relationships between regex complexity and DFA state count

### DSL builtins needed

- `regex-union`, `regex-concat`, `regex-star`, `regex-complement`
- `nfa-to-dfa`, `dfa-minimize`, `dfa-complement`, `dfa-product`
- `dfa-accepts?` — test a string against a DFA
- `dfa-equivalent?` — check language equivalence (via product + emptiness)
- `dfa-state-count`, `dfa-is-minimal?`
- `sample-language` — generate N accepted/rejected strings for empirical testing
- `regex-to-dfa`, `dfa-to-regex`

### Implementation complexity

**Moderate.** The DFA/NFA operations are well-known algorithms, all O(n²) or better for small automata. The main challenge is the data representation — nested maps for transition tables are clunky in the current DSL. Might want a dedicated automaton type rather than encoding everything as maps.

**Recommended approach:** Keep automata small (≤ 20 states) to keep operations fast. Use a binary alphabet {a, b} initially to limit combinatorial explosion. Add alphabet symbols as the system discovers that two-symbol automata are well-explored.

### Seed content

Start with:
- A handful of simple regexes: `a*`, `(ab)*`, `a*b*`, `(a|b)*`, `a(a|b)*b`
- Their DFA representations
- The basic operations: Union, Concat, Star, Complement
- The conversion operations: RegexToDFA, Minimize
- A few known equivalences as seed conjectures
