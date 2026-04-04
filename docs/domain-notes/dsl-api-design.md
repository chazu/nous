# DSL / API Design Domain

## Motivation

An agent building a system needs to design interfaces — function signatures, data shapes, configuration schemas, protocol contracts. These are compositional objects with computable properties. An EURISKO-style system could explore the space of interface designs and discover which compositions produce clean, minimal, orthogonal APIs.

The output is directly actionable: "when building a CRUD service, these 4 endpoint shapes cover all cases" or "this configuration schema is equivalent to that one but has half the surface area."

## Why this fits Mode 1

- Types compose: product (struct), sum (union/enum), function (A → B), optional, list, map
- Type properties are computable: inhabited?, subtype?, equivalent?, surface area
- Operations on types are well-defined: combine, restrict, extend, project, embed
- Evaluation is immediate: compare two API schemas for equivalence, check subsumption

## Concept space

### Core types

- **Type** — a data type. Can be primitive, composite, or function.
  - Primitives: `String`, `Int`, `Bool`, `Float`, `Bytes`, `Timestamp`
  - Composites: `Product(fields...)`, `Sum(variants...)`, `List(element)`, `Map(key, value)`, `Optional(inner)`
  - Functions: `Fn(params...) → return`

- **Schema** — a named collection of types with constraints. An API is a schema.
  - Data slot: list of named types with their definitions
  - Invariants: field count, depth, optional ratio, polymorphism degree

- **Endpoint** — a function type with metadata: method, path pattern, auth requirement, idempotency.

- **Contract** — a pair of schemas (request, response) with protocol metadata.

- **SchemaOp** — an operation on schemas (combine, restrict, project, etc.)

### Data representation

A Schema as a unit:
```
name: "UserAPI"
data: {
    types: {
        "User":       {kind: "product", fields: {"id": "Int", "name": "String", "email": "String"}},
        "CreateUser": {kind: "product", fields: {"name": "String", "email": "String"}},
        "UserList":   {kind: "product", fields: {"items": "List(User)", "total": "Int"}},
    },
    endpoints: [
        {method: "GET",    path: "/users",     input: "Empty",      output: "UserList"},
        {method: "POST",   path: "/users",     input: "CreateUser", output: "User"},
        {method: "GET",    path: "/users/:id",  input: "Empty",      output: "User"},
        {method: "DELETE", path: "/users/:id",  input: "Empty",      output: "Empty"},
    ]
}
invariants: {
    typeCount: 3,
    endpointCount: 4,
    fieldCount: 7,
    maxDepth: 2,
    optionalRatio: 0.0,
    avgFieldsPerType: 2.33,
}
```

### Operations

**On types:**
- **Product** — combine two types into a struct: `Product(A, B)` has all fields of both
- **Sum** — create a union: `Sum(A, B)` can be either
- **Project** — keep only certain fields from a product type
- **Extend** — add fields to a product type
- **Restrict** — add constraints to field types (e.g., `String` → `String & len ≤ 255`)
- **Optional** — wrap a type in optional
- **Unwrap** — remove optional wrapper
- **Embed** — include one type inside another (like Go embedding)

**On schemas:**
- **MergeSchemas** — combine two schemas, unifying shared types
- **IntersectSchemas** — keep only types/endpoints present in both
- **ProjectSchema** — keep only a subset of endpoints
- **SpecializeEndpoint** — narrow an endpoint's input/output types
- **GeneralizeEndpoint** — widen an endpoint's input/output types
- **AddEndpoint** — add a new endpoint
- **DeduplicateTypes** — find equivalent types and merge them

**Conversions:**
- **SchemaToContract** — extract request/response pairs
- **ContractToSchema** — reconstruct a schema from contracts
- **CRUDGenerate** — given a type, generate the standard CRUD endpoints

### Quality attributes

- `typeCount` — fewer types with more reuse is better
- `endpointCount` — measure of API surface
- `fieldCount` — total fields across all types
- `maxDepth` — deepest nesting of composite types
- `optionalRatio` — fraction of fields that are optional (high = unclear contract)
- `duplication` — number of equivalent or near-equivalent type pairs
- `orthogonality` — are endpoints independent or do they overlap in capability?
- `coveredOperations` — CRUD coverage: how many of create/read/update/delete/list are supported?
- `subtypeRelations` — how many types are subtypes of others (indicates polymorphism/reuse)

### What the system could discover

- **Canonical CRUD patterns:** Given a base type, the system generates all possible endpoint sets and discovers that 4-5 endpoints (list, get, create, update, delete) is a natural attractor — adding more creates redundancy, fewer leaves gaps.

- **Type equivalences:** `Product(A, Optional(B))` with B defaulting to empty is equivalent to `Sum(A, Product(A, B))` for certain usage patterns. The system finds this by comparing which inputs each accepts.

- **Schema minimization:** Two schemas that look different but are functionally equivalent — the system discovers that a "flat" schema with 10 fields and a "nested" schema with 3 types of 3-4 fields accept the same data. The nested one has lower cognitive load.

- **Anti-patterns:** An endpoint that takes `Product(all fields)` but only uses 2 of them — the system discovers this is equivalent to a projected input type and the projection is always preferable (lower surface area, same capability).

- **Composition patterns:** Merging a UserAPI and an AuthAPI produces a schema with redundant user types — the system learns to deduplicate after merge.

### DSL builtins needed

- `type-product`, `type-sum`, `type-list`, `type-optional`
- `type-project`, `type-extend`, `type-restrict`
- `type-equivalent?`, `type-subtype?`, `type-inhabited?`
- `schema-merge`, `schema-intersect`, `schema-project`
- `schema-type-count`, `schema-endpoint-count`, `schema-duplication`
- `schema-crud-coverage` — what fraction of CRUD operations are present
- `schema-generate-crud` — given a type, produce standard CRUD endpoints

### Implementation complexity

**Moderate.** Types as data are tree structures — products, sums, lists, optionals. Type equivalence is structural comparison. Subtyping is well-defined (A is a subtype of B if every valid A is also a valid B). The main complexity is representing nested type structures in the DSL, which may need dedicated builtins rather than generic map operations.

### Seed content

- 3-4 primitive types: String, Int, Bool, Timestamp
- 2-3 example schemas: UserAPI (CRUD), AuthAPI (login/logout/refresh), simple key-value store
- The basic type operations: Product, Sum, Optional, Project, Extend
- Schema operations: Merge, GenerateCRUD, Deduplicate
- Known equivalences as seed conjectures
