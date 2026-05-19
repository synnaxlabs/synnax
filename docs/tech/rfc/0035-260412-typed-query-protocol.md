# 35 - Typed Query Protocol

**Feature Name**: Typed Composable Query Filters Across the API Layer

# 0 - Summary

This RFC describes a typed, composable query protocol that exposes gorp's filter algebra
(AND/OR/NOT, ordering, cursor pagination) through the API layer to all client languages.
Today, retrieve requests are flat structs with implicit AND semantics and no ordering
support. By generating per-entity filter tree types from oracle schemas, we can give
clients full boolean composition and sorted ordering while maintaining compile-time type
safety in Go, TypeScript, Python, and C++ with zero `any`, zero reflection, and zero
string-based DSLs.

# 1 - Vocabulary

- **Filter tree**: A recursive structure where each node is either a leaf (field
  comparison) or a combinator (AND/OR/NOT over child nodes).
- **Leaf filter**: A comparison on a single field (e.g., `name in ["sensor_a"]`).
- **Combinator**: A boolean operator that composes child filters (`and`, `or`, `not`).
- **Comparison struct**: A per-scalar-type struct that holds the operator and value(s)
  for a leaf filter (e.g., `StringFilter { eq, in, contains }`).
- **Filter node**: The per-entity generated struct that has one optional field per
  filterable attribute plus `and`/`or`/`not` for recursion.
- **Interpreter**: The generated Go function that walks a deserialized filter node and
  produces `gorp.Filter[K, E]` calls.

# 2 - Motivation

RFC 0034 introduced in-memory secondary indexes for gorp tables. The gorp layer now
supports:

- `gorp.And(filters...)` with key-set intersection
- `gorp.Or(filters...)` with key-set union
- `gorp.Not(filter)` with eval inversion
- `Retrieve.OrderBy()` with typed cursor pagination via `SortedQuery.After(cursor)`
- Index-backed O(1) lookups and O(log n) sorted walks

None of this is reachable from clients. The API layer uses flat request structs where
every field is implicitly ANDed:

```go
type RetrieveRequest struct {
    Keys         channel.Keys
    Names        []string
    DataTypes    []telem.DataType
    Virtual      *bool
    Limit        int
    Offset       int
    // ...
}
```

The API handler mechanically translates each non-nil field into a `.Where()` call. There
is no way for a client to express:

- "channels named X **or** with data type Y" (OR across fields)
- "channels that are **not** internal" as a composable filter
- "channels ordered by name descending" (ordering)
- "next page of results after cursor C" (cursor pagination)

These are real user needs. The Console search, resource browsers, and pagination all
require these capabilities. Today they are worked around with multiple requests, client-
side filtering, or offset-based pagination. The gorp layer is ready. The gap is entirely
in the transport and client layers.

## 2.0 - Why Not a String DSL?

Google's AIP-160 and OData both use string-based filter expressions
(`"name = 'foo' AND age > 18"`). This is the simplest approach to implement on the wire,
but it provides zero compile-time safety. The server must parse, validate, and translate
the string at runtime. Clients build filter strings via concatenation, which is error-
prone and untestable at compile time. This contradicts our core principle of type safety
at every layer.

## 2.1 - Why Not GraphQL?

GraphQL provides exactly the filter tree model we want (Hasura's `_and`/`_or`/`_not`
boolean expression types). But adopting GraphQL as the transport protocol requires
replacing Freighter, adding a GraphQL schema layer, and adopting GraphQL codegen
tooling. The protocol overhead (parsing, introspection, N+1 resolution) is unnecessary
for our use case. We want the data model, not the protocol.

## 2.2 - Prior Art

The design draws directly from the Hasura/Prisma pattern: per-entity typed filter input
objects where `and`/`or`/`not` are structural fields and leaf nodes are per-field
comparison objects. The key adaptation: instead of GraphQL SDL as the schema source, we
use `.oracle` files. Instead of graphql-codegen, we use oracle plugins. Instead of
GraphQL transport, we use Freighter with JSON/msgpack.

Other systems examined:

| System       | Approach               | Type Safety          | Multi-Language  |
| ------------ | ---------------------- | -------------------- | --------------- |
| Hasura       | Recursive JSON tree    | Codegen per language | Yes             |
| Prisma       | Recursive TS objects   | TS inference         | No              |
| tRPC         | Zod recursive schemas  | TS inference only    | No              |
| AIP-160      | String expression      | None                 | Yes (string)    |
| OData        | URL query string       | With .NET LINQ only  | Partial         |
| Ent (Go)     | Typed closures         | Go compiler          | Via entgql only |
| Ent + entgql | Ent closures + GraphQL | Codegen per language | Yes             |
| go-jet       | Expression tree nodes  | Go compiler          | No              |
| Drizzle      | Typed AST `SQL<T>`     | TS compiler          | No              |

Ent + entgql is the closest existing analog: typed predicate closures on the server,
typed filter input objects on the wire, codegen for client types. We replace entgql with
oracle and replace GraphQL with Freighter.

# 3 - Design

## 3.0 - Guiding Principles

1. **Zero type erasure.** No `any`, no `interface{}`, no reflection, no type assertions.
   Every field, operator, and value is statically typed in generated code across all
   languages.

2. **Oracle is the single source of truth.** The `.oracle` schema file defines which
   fields are filterable, which are orderable, and what comparison operators each field
   supports. Oracle generates the filter types, interpreters, and client schemas.

3. **The interpreter bridges wire format to gorp.** The generated `toFilter()` method on
   the server calls the same `MatchNames()`, `MatchDataTypes()`, `MatchVirtual()` filter
   functions that the API handler calls today. The index optimization is preserved
   transparently.

4. **Backward compatible.** The existing flat request fields (`Keys`, `Names`,
   `SearchTerm`, `Limit`, `Offset`) remain. The new `Where` and `OrderBy` fields are
   additive. Clients adopt incrementally.

5. **Structural recursion over tag discrimination.** Filter trees use field-level
   discrimination (`and`/`or`/`not` are fields on the filter struct), not a tagged union
   with an `op` string. This maps cleanly to generated structs in all languages without
   requiring a discriminator enum.

## 3.1 - Wire Format

### 3.1.0 - Filter Tree

A retrieve request carries an optional `where` field containing a recursive filter node.
Each node has one optional field per filterable attribute (leaf filters) plus `and`,
`or`, and `not` for recursion:

```json
{
  "where": {
    "or": [
      { "name": { "in": ["sensor_a", "sensor_b"] } },
      { "and": [{ "dataType": { "eq": "float64" } }, { "virtual": { "eq": false } }] }
    ]
  },
  "orderBy": [{ "field": "name", "direction": "asc" }],
  "limit": 10,
  "offset": 20
}
```

When `and`/`or`/`not` are absent, all present leaf fields on a single node are
implicitly ANDed. This means `{"name": {"eq": "x"}, "virtual": {"eq": true}}` is
equivalent to `{"and": [{"name": {"eq": "x"}}, {"virtual": {"eq": true}}]}`. This keeps
simple queries compact while allowing full composition when needed.

### 3.1.1 - Comparison Operators

Each scalar type gets a comparison struct with operators appropriate to its type:

**StringFilter:**

```json
{
  "eq": "exact_value",
  "in": ["val_a", "val_b"],
  "contains": "substring",
  "regex": "^sensor_.*$"
}
```

**BoolFilter:**

```json
{ "eq": true }
```

**NumericFilter** (for int, uint, float types):

```json
{
  "eq": 42,
  "in": [1, 2, 3],
  "gt": 10,
  "lt": 100,
  "gte": 10,
  "lte": 100
}
```

**TimeFilter** (for timestamps):

```json
{
  "eq": 1700000000000000000,
  "gt": 1700000000000000000,
  "lt": 1800000000000000000
}
```

Multiple operators on the same comparison struct are ANDed (e.g.,
`{"gt": 10, "lt": 100}` means "between 10 and 100 exclusive").

### 3.1.2 - Ordering

The `orderBy` field is a per-field typed struct, mirroring the filter node design. Each
sortable field gets its own typed slot containing direction and an optional typed
cursor:

```json
{ "orderBy": { "name": { "direction": "asc", "after": "sensor_m" } } }
```

or for a timestamp-sorted field:

```json
{ "orderBy": { "createdAt": { "direction": "desc", "after": 1700000000000000000 } } }
```

Only one field may be set per request (single-field ordering). The `after` value
provides cursor-based pagination: the walk skips entries whose sort value is less than
or equal to (asc) or greater than or equal to (desc) the cursor. Because each field has
its own typed struct, the cursor value is statically typed (string for name, int64 for
timestamps) with no `any` or `json.RawMessage`.

This design is consistent with the filter node pattern: per-field typed slots, only one
active at a time for ordering, oracle generates the struct and interpreter. Multi-field
ordering (tiebreakers) is deferred until gorp supports composing multiple `OrderQuery`
handles.

### 3.1.3 - Coexistence with Flat Fields

The new `where` and `orderBy` fields coexist with the existing flat request fields. The
API handler merges them: flat fields are converted into an implicit AND tree, then ANDed
with the explicit `where` tree if present. This allows incremental adoption:

```json
{
  "keys": [1, 2, 3],
  "where": { "virtual": { "eq": false } },
  "limit": 10
}
```

This request fetches channels 1, 2, 3 that are not virtual, limited to 10 results.

## 3.2 - Go Server Layer

### 3.2.0 - Comparison Structs (Shared)

A small set of comparison structs are defined once in a shared package (e.g.,
`x/go/filter` or within `gorp` itself). These are not per-entity; they are per-scalar-
type:

```go
package filter

// StringFilter holds comparison operators for string fields.
type StringFilter struct {
    Eq       *string  `json:"eq" msgpack:"eq"`
    In       []string `json:"in" msgpack:"in"`
    Contains *string  `json:"contains" msgpack:"contains"`
    Regex    *string  `json:"regex" msgpack:"regex"`
}

// BoolFilter holds comparison operators for boolean fields.
type BoolFilter struct {
    Eq *bool `json:"eq" msgpack:"eq"`
}

// NumericFilter holds comparison operators for numeric fields.
type NumericFilter[T constraints.Integer | constraints.Float] struct {
    Eq  *T  `json:"eq" msgpack:"eq"`
    In  []T `json:"in" msgpack:"in"`
    Gt  *T  `json:"gt" msgpack:"gt"`
    Lt  *T  `json:"lt" msgpack:"lt"`
    Gte *T  `json:"gte" msgpack:"gte"`
    Lte *T  `json:"lte" msgpack:"lte"`
}
```

The generic `NumericFilter[T]` avoids defining separate structs for int32, uint32,
int64, float64, etc. The `T` parameter ensures comparison values are the correct type at
deserialization time.

### 3.2.1 - Per-Entity Filter Node (Oracle-Generated)

Oracle generates a filter node struct per `@retrieve` entity. Each `@filter`-annotated
field gets a typed slot:

```go
// Generated in core/pkg/api/channel/filter.gen.go

// ChannelFilterNode is a composable filter tree for channel retrieval.
// Each field-level filter is optional; present fields on a single node
// are implicitly ANDed. Use And/Or/Not for explicit boolean composition.
type ChannelFilterNode struct {
    Name     *filter.StringFilter             `json:"name" msgpack:"name"`
    DataType *filter.StringFilter             `json:"dataType" msgpack:"dataType"`
    Virtual  *filter.BoolFilter               `json:"virtual" msgpack:"virtual"`
    IsIndex  *filter.BoolFilter               `json:"isIndex" msgpack:"isIndex"`
    Internal *filter.BoolFilter               `json:"internal" msgpack:"internal"`
    And      []ChannelFilterNode              `json:"and" msgpack:"and"`
    Or       []ChannelFilterNode              `json:"or" msgpack:"or"`
    Not      *ChannelFilterNode               `json:"not" msgpack:"not"`
}
```

No `any`. No `interface{}`. Every field is concrete.

### 3.2.2 - Per-Entity Interpreter (Oracle-Generated)

Oracle generates a `toFilter()` method that walks the struct and produces the same typed
`channel.Filter` values that hand-written API code produces today:

```go
// Generated alongside ChannelFilterNode

func (n ChannelFilterNode) toFilter() channel.Filter {
    var filters []channel.Filter

    // Leaf filters: each non-nil field becomes a typed filter call.
    if n.Name != nil {
        if n.Name.Eq != nil {
            filters = append(filters, channel.MatchNames(*n.Name.Eq))
        }
        if len(n.Name.In) > 0 {
            filters = append(filters, channel.MatchNames(n.Name.In...))
        }
        if n.Name.Contains != nil {
            filters = append(filters, channel.MatchNameContains(*n.Name.Contains))
        }
        if n.Name.Regex != nil {
            filters = append(filters, channel.MatchNames(*n.Name.Regex))
        }
    }
    if n.DataType != nil {
        if len(n.DataType.In) > 0 {
            filters = append(filters, channel.MatchDataTypes(
                castDataTypes(n.DataType.In)...,
            ))
        }
    }
    if n.Virtual != nil && n.Virtual.Eq != nil {
        filters = append(filters, channel.MatchVirtual(*n.Virtual.Eq))
    }
    if n.IsIndex != nil && n.IsIndex.Eq != nil {
        filters = append(filters, channel.MatchIsIndex(*n.IsIndex.Eq))
    }
    if n.Internal != nil && n.Internal.Eq != nil {
        filters = append(filters, channel.MatchInternal(*n.Internal.Eq))
    }

    // Combinators: recursive composition.
    for _, child := range n.And {
        filters = append(filters, child.toFilter())
    }

    // Implicit AND for all leaf + And children.
    result := channel.And(filters...)

    // OR children: each child is a sub-tree, composed via channel.Or.
    if len(n.Or) > 0 {
        orFilters := make([]channel.Filter, len(n.Or))
        for i, child := range n.Or {
            orFilters[i] = child.toFilter()
        }
        orResult := channel.Or(orFilters...)
        if len(filters) > 0 {
            result = channel.And(result, orResult)
        } else {
            result = orResult
        }
    }

    // NOT: invert the child sub-tree.
    if n.Not != nil {
        notFilter := channel.Not(n.Not.toFilter())
        if result != nil {
            result = channel.And(result, notFilter)
        } else {
            result = notFilter
        }
    }

    return result
}
```

This is the critical bridge. The interpreter calls the exact same generated `MatchX`
functions from RFC 0034, which route through indexes when available and fall back to
scans when not. The index optimization is fully preserved.

### 3.2.3 - OrderBy Types and Interpreter (Oracle-Generated)

Oracle generates per-field typed OrderBy structs from `@index sorted` annotations,
mirroring the per-field filter design. Each sortable field gets its own struct with a
typed cursor:

```go
// Generated in core/pkg/api/channel/filter.gen.go

// StringOrderBy holds ordering parameters for a string-typed sorted field.
type StringOrderBy struct {
    Direction string  `json:"direction" msgpack:"direction"`
    After     *string `json:"after" msgpack:"after"`
}

// TimestampOrderBy holds ordering parameters for a timestamp-typed sorted field.
type TimestampOrderBy struct {
    Direction string           `json:"direction" msgpack:"direction"`
    After     *telem.TimeStamp `json:"after" msgpack:"after"`
}

// ChannelOrderBy is a per-field typed ordering specification. Only one field
// may be set per request.
type ChannelOrderBy struct {
    Name      *StringOrderBy    `json:"name" msgpack:"name"`
    CreatedAt *TimestampOrderBy  `json:"createdAt" msgpack:"createdAt"`
}
```

The interpreter routes to the typed `OrderByX()` functions from RFC 0034:

```go
func (o ChannelOrderBy) toOrder() channel.Order {
    if o.Name != nil {
        dir := parseDirection(o.Name.Direction)
        if o.Name.After != nil {
            return channel.OrderByName(dir, *o.Name.After)
        }
        return channel.OrderByName(dir)
    }
    if o.CreatedAt != nil {
        dir := parseDirection(o.CreatedAt.Direction)
        if o.CreatedAt.After != nil {
            return channel.OrderByCreatedAt(dir, *o.CreatedAt.After)
        }
        return channel.OrderByCreatedAt(dir)
    }
    return nil
}

func parseDirection(s string) gorp.Direction {
    if s == "desc" {
        return gorp.Desc
    }
    return gorp.Asc
}
```

No `any`. No `json.RawMessage`. No string-to-field dispatch. The cursor value is typed
at the struct level. The `After *string` on `StringOrderBy` and `After *telem.TimeStamp`
on `TimestampOrderBy` are known at compile time. Oracle generates the correct OrderBy
struct per field from the field's type in the `.oracle` schema.

### 3.2.4 - API Handler Simplification

The API handler collapses from N if-checks to a single filter tree interpretation:

```go
func (s *Service) Retrieve(
    ctx context.Context,
    req RetrieveRequest,
) (RetrieveResponse, error) {
    var resChannels []channel.Channel
    q := s.internal.NewRetrieve().Entries(&resChannels)

    // Legacy flat fields (backward compatible).
    if len(req.Keys) > 0 {
        q = q.WhereKeys(req.Keys...)
    }
    if len(req.SearchTerm) > 0 {
        q = q.Search(req.SearchTerm)
    }

    // New: structured filter tree.
    if req.Where != nil {
        q = q.Where(req.Where.toFilter())
    }

    // New: structured ordering.
    if req.OrderBy != nil {
        if o := req.OrderBy.toOrder(); o != nil {
            q = q.OrderBy(o)
        }
    }

    if req.Limit > 0 {
        q = q.Limit(req.Limit)
    }
    if req.Offset > 0 {
        q = q.Offset(req.Offset)
    }

    if err := q.Exec(ctx, nil); err != nil {
        return RetrieveResponse{}, err
    }
    // ... access control, translation, response
}
```

The `Where` and `OrderBy` handling is two lines each, regardless of how many filterable
fields the entity has. Oracle generates all the routing logic.

### 3.2.5 - Request Type Changes

The retrieve request gains two new optional fields:

```go
type RetrieveRequest struct {
    // Existing flat fields (preserved for backward compatibility).
    Keys         channel.Keys     `json:"keys" msgpack:"keys"`
    Names        []string         `json:"names" msgpack:"names"`
    SearchTerm   string           `json:"search_term" msgpack:"search_term"`
    DataTypes    []telem.DataType `json:"data_types" msgpack:"data_types"`
    // ...

    // New: structured filter tree.
    Where   *ChannelFilterNode `json:"where" msgpack:"where"`

    // New: per-field typed ordering.
    OrderBy *ChannelOrderBy    `json:"order_by" msgpack:"order_by"`

    // Preserved.
    Limit  int `json:"limit" msgpack:"limit"`
    Offset int `json:"offset" msgpack:"offset"`
}
```

## 3.3 - TypeScript Client Layer

### 3.3.0 - Filter Types (Oracle-Generated)

Oracle generates Zod schemas for each entity's filter node. The recursive type uses
`z.lazy()`:

```typescript
// Generated in client/ts/src/channel/filter.gen.ts

import { z } from "zod";
import { DataType } from "@synnaxlabs/x";

export const stringFilterZ = z.object({
  eq: z.string().optional(),
  in: z.string().array().optional(),
  contains: z.string().optional(),
  regex: z.string().optional(),
});

export const boolFilterZ = z.object({
  eq: z.boolean().optional(),
});

export const channelFilterZ: z.ZodType<ChannelFilter> = z.lazy(() =>
  z.object({
    name: stringFilterZ.optional(),
    dataType: stringFilterZ.optional(),
    virtual: boolFilterZ.optional(),
    isIndex: boolFilterZ.optional(),
    internal: boolFilterZ.optional(),
    and: z.array(channelFilterZ).optional(),
    or: z.array(channelFilterZ).optional(),
    not: channelFilterZ.optional(),
  }),
);
export interface ChannelFilter extends z.infer<typeof channelFilterZ> {}

export const stringOrderByZ = z.object({
  direction: z.enum(["asc", "desc"]),
  after: z.string().optional(),
});

export const timestampOrderByZ = z.object({
  direction: z.enum(["asc", "desc"]),
  after: z.bigint().optional(),
});

export const channelOrderByZ = z.object({
  name: stringOrderByZ.optional(),
  createdAt: timestampOrderByZ.optional(),
});
export interface ChannelOrderBy extends z.infer<typeof channelOrderByZ> {}
```

### 3.3.1 - Client Usage

The TypeScript client gains `where` and `orderBy` on retrieve:

```typescript
// Existing usage still works.
const channels = await client.channels.retrieve({ names: ["sensor_a"] });

// New: composable filters.
const channels = await client.channels.retrieve({
  where: {
    or: [
      { name: { in: ["sensor_a", "sensor_b"] } },
      { and: [{ dataType: { eq: "float64" } }, { virtual: { eq: false } }] },
    ],
  },
  orderBy: { name: { direction: "asc" } },
  limit: 50,
});

// Cursor pagination: resume after "sensor_m".
const nextPage = await client.channels.retrieve({
  orderBy: { name: { direction: "asc", after: "sensor_m" } },
  limit: 50,
});

// Simple equality is still compact.
const channels = await client.channels.retrieve({
  where: { name: { eq: "sensor_a" }, virtual: { eq: false } },
});
```

TypeScript enforces at compile time that `name` accepts `StringFilter` fields (`eq`,
`in`, `contains`, `regex`), `virtual` accepts `BoolFilter` (`eq` only), and `orderBy`
`field` is constrained to `"name" | "createdAt"`.

### 3.3.2 - Comparison Structs (Shared)

The shared comparison filter Zod schemas (`stringFilterZ`, `boolFilterZ`,
`numericFilterZ`) are defined once in `@synnaxlabs/x` or a shared client utilities
package, imported by each entity's generated filter module.

## 3.4 - Python Client Layer

### 3.4.0 - Filter Types (Oracle-Generated)

Oracle generates Pydantic models for each entity's filter node:

```python
# Generated in client/py/synnax/channel/filter_gen.py

from __future__ import annotations
from pydantic import BaseModel, Field

class StringFilter(BaseModel):
    eq: str | None = None
    in_: list[str] | None = Field(None, alias="in")
    contains: str | None = None
    regex: str | None = None

class BoolFilter(BaseModel):
    eq: bool | None = None

class ChannelFilter(BaseModel):
    name: StringFilter | None = None
    data_type: StringFilter | None = Field(None, alias="dataType")
    virtual: BoolFilter | None = None
    is_index: BoolFilter | None = Field(None, alias="isIndex")
    internal: BoolFilter | None = None
    and_: list[ChannelFilter] | None = Field(None, alias="and")
    or_: list[ChannelFilter] | None = Field(None, alias="or")
    not_: ChannelFilter | None = Field(None, alias="not")

class StringOrderBy(BaseModel):
    direction: str = "asc"
    after: str | None = None

class TimestampOrderBy(BaseModel):
    direction: str = "asc"
    after: int | None = None

class ChannelOrderBy(BaseModel):
    name: StringOrderBy | None = None
    created_at: TimestampOrderBy | None = Field(None, alias="createdAt")
```

### 3.4.1 - Client Usage

```python
# Existing usage still works.
channels = client.channels.retrieve(names=["sensor_a"])

# New: composable filters.
channels = client.channels.retrieve(
    where=ChannelFilter(
        or_=[
            ChannelFilter(name=StringFilter(in_=["sensor_a", "sensor_b"])),
            ChannelFilter(
                and_=[
                    ChannelFilter(data_type=StringFilter(eq="float64")),
                    ChannelFilter(virtual=BoolFilter(eq=False)),
                ]
            ),
        ]
    ),
    order_by=ChannelOrderBy(name=StringOrderBy(direction="asc")),
    limit=50,
)
```

Pydantic validates the filter tree at construction time. Invalid field names or wrong
value types raise `ValidationError`.

## 3.5 - C++ Client Layer

### 3.5.0 - Filter Types (Oracle-Generated)

Oracle generates C++ structs with nlohmann/json serialization:

```cpp
// Generated in client/cpp/channel/filter.gen.h

namespace channel {

struct StringFilter {
    std::optional<std::string> eq;
    std::optional<std::vector<std::string>> in;
    std::optional<std::string> contains;
    std::optional<std::string> regex;
};

struct BoolFilter {
    std::optional<bool> eq;
};

struct ChannelFilter {
    std::optional<StringFilter> name;
    std::optional<StringFilter> data_type;
    std::optional<BoolFilter> virtual_;
    std::optional<BoolFilter> is_index;
    std::optional<BoolFilter> internal;
    std::optional<std::vector<ChannelFilter>> and_;
    std::optional<std::vector<ChannelFilter>> or_;
    std::optional<std::unique_ptr<ChannelFilter>> not_;
};

}  // namespace channel
```

`std::unique_ptr` for the `not_` field avoids infinite recursion in the struct layout.

## 3.6 - Oracle Schema Annotations

### 3.6.0 - Existing Annotations

The `@filter` and `@index` annotations from RFC 0034 already mark which fields are
filterable and indexable. The query protocol builds directly on these:

```
Channel struct {
    name        Name           { @index lookup  }
    data_type   telem.DataType { @filter         }
    is_index    bool           { @filter         }
    virtual     bool           { @filter         }
    internal    bool           { @filter         }
    created_at  telem.TimeStamp { @index sorted  }
}
```

### 3.6.1 - New Annotations

No new schema annotations are needed for the basic design. The `@filter` annotation
already identifies filterable fields. The `@index sorted` annotation already identifies
orderable fields. Oracle can derive comparison operators from the field's primitive
type:

| Primitive  | Comparison struct | Operators                            |
| ---------- | ----------------- | ------------------------------------ |
| `string`   | `StringFilter`    | `eq`, `in`, `contains`, `regex`      |
| `bool`     | `BoolFilter`      | `eq`                                 |
| integers   | `NumericFilter`   | `eq`, `in`, `gt`, `lt`, `gte`, `lte` |
| floats     | `NumericFilter`   | `eq`, `in`, `gt`, `lt`, `gte`, `lte` |
| timestamps | `TimeFilter`      | `eq`, `gt`, `lt`, `gte`, `lte`       |

If a field needs to restrict its operator set (e.g., a string field that should only
support `eq` and `in`, not `contains`), a future `@filter ops "eq,in"` annotation can
narrow it. This is not needed for the initial implementation.

## 3.7 - Oracle Plugin Architecture

### 3.7.0 - Server-Side: Extension to `go/query`

The existing `go/query` oracle plugin already generates `retrieve.gen.go` with filter
functions, index structs, and query builders. The filter node struct and interpreter are
natural extensions:

1. For each `@retrieve` entity, emit a `FilterNode` struct with one optional field per
   `@filter` field, plus `And`/`Or`/`Not`.
2. Emit a `toFilter()` method that routes each non-nil field to the corresponding
   `MatchX()` function (already generated by `go/query`).
3. Emit an `OrderBySpec` struct and `toOrder()` method that routes each `@index sorted`
   field to the corresponding `OrderByX()` function (already generated by `go/query`).

This code lives in the API package (e.g., `core/pkg/api/channel/`) alongside the
request/response types, not in the service package. The service layer's
`retrieve.gen.go` is unchanged.

### 3.7.1 - Client-Side: New Plugins

Oracle currently generates types for TypeScript (Zod), Python (Pydantic), and C++, but
does not generate query builders for clients. New plugins (or extensions to existing
type plugins) generate:

- **`ts/filter`**: Zod schemas for filter nodes and comparison structs.
- **`py/filter`**: Pydantic models for filter nodes and comparison structs.
- **`cpp/filter`**: C++ structs with serialization for filter nodes.

These plugins read the same `@filter` and `@index` annotations that `go/query` reads.
The shared comparison structs (`StringFilter`, `BoolFilter`, etc.) are emitted once as
static library types; the per-entity filter nodes are generated per `@retrieve` entity.

# 4 - What This RFC Does Not Cover

- **Range queries on sorted indexes.** The interpreter supports `eq`, `in`, and ordering
  via `OrderBy`. Range predicates (`gt`, `lt` on sorted fields used as WHERE filters
  rather than ORDER BY cursors) require gorp support for range walks, which is not yet
  implemented. Deferred to a follow-up.

- **Multi-field ordering (tiebreakers).** The initial implementation supports single-
  field ordering. Multi-field ordering requires gorp to compose multiple `OrderQuery`
  handles, which is not yet supported.

- **Index intersection for OR.** When an OR combines two index-backed filters, gorp
  already performs key-set union. The protocol does not need to do anything special
  here; it falls out of the existing `gorp.Or` implementation.

- **Aggregation queries.** Count, sum, min, max, and other aggregation operations are
  separate from filtering. They may be added to the protocol later but are out of scope
  here.

- **Deprecation of flat fields.** The existing flat request fields are preserved
  indefinitely. A future RFC may propose deprecation once all clients have adopted the
  structured filter format.

- **Custom filter functions.** Some entities have non-trivial filter logic (e.g.,
  channel's `MatchVirtual` excludes calculated channels from the virtual bucket). The
  interpreter calls these existing functions, but the protocol does not expose a way to
  express arbitrary custom predicates from the client. Custom server-side filters that
  don't map to a single field comparison remain API-handler logic.

# 5 - Implementation Order

## Phase 1: Shared Comparison Structs

1. Define `StringFilter`, `BoolFilter`, `NumericFilter[T]`, and `TimeFilter` in a shared
   Go package (e.g., `x/go/filter/`).
2. Define the equivalent Zod schemas in `@synnaxlabs/x`.
3. Define the equivalent Pydantic models in the Python client utilities.
4. Define the equivalent C++ structs.
5. Add JSON/msgpack serialization tests for all comparison structs.

## Phase 2: Oracle Go Server Plugin

1. Extend the `go/query` plugin (or create a sibling `go/filter` plugin) to emit per-
   entity `FilterNode` structs from `@filter` annotations.
2. Emit the `toFilter()` interpreter method that calls the existing `MatchX()`
   functions.
3. Emit the `OrderBySpec` struct and `toOrder()` interpreter from `@index sorted`
   annotations.
4. Generate for a pilot entity (e.g., `rack`, which has few fields) and validate end-to-
   end.

## Phase 3: API Layer Integration

1. Add `Where *FilterNode` and `OrderBy []OrderBySpec` fields to `RetrieveRequest` for
   the pilot entity.
2. Update the API handler to call `toFilter()` and `toOrder()`.
3. Write integration tests covering AND, OR, NOT, ordering, and cursor pagination
   through the HTTP API.
4. Roll out to remaining entities (channel, device, task, ranger, etc.).

## Phase 4: Oracle Client Plugins

1. Extend the `ts/types` plugin (or create `ts/filter`) to emit Zod schemas for per-
   entity filter nodes.
2. Extend the `py/types` plugin (or create `py/filter`) to emit Pydantic models.
3. Extend the `cpp/types` plugin (or create `cpp/filter`) to emit C++ filter structs.
4. Update client retriever implementations to accept `where` and `orderBy` parameters.

## Phase 5: Client Adoption

1. Update Console search and resource browsers to use structured filters.
2. Update Python client `retrieve()` to support `where` parameter.
3. Update C++ driver code if applicable.
4. Document the new query protocol in client API docs.

# 6 - Resolved Decisions

1. **Structural fields over tag discrimination.** Filter trees use `and`/`or`/`not` as
   fields on the filter node struct, not a tagged union with
   `{"op": "and", "filters": [...]}`. The structural approach maps directly to generated
   struct types in all languages without requiring a discriminator enum. It matches the
   Hasura/Prisma pattern that has proven ergonomic across large codebases.

2. **Per-field typed slots over `any`-based dispatch.** Each filterable field gets its
   own typed optional field on the filter node (e.g., `Name *StringFilter`). The
   alternative would be `Field string, Values []any` with runtime type assertion. The
   per-field approach preserves compile-time safety, avoids `any`, and allows oracle to
   generate exhaustive interpreters. The cost is more generated code per entity, but
   this code is mechanical and produced by oracle.

3. **Shared comparison structs over per-entity comparison types.** `StringFilter`,
   `BoolFilter`, and `NumericFilter` are defined once and reused across all entities.
   Per-entity comparison types would allow custom operator sets per field but add
   significant code volume for little benefit. The shared approach is extended later
   with `@filter ops` annotations if needed.

4. **Interpreter calls existing `MatchX()` functions.** The generated `toFilter()`
   method does not implement filter logic itself. It delegates to the same
   `channel.MatchNames`, `channel.MatchVirtual`, etc. that the API handler calls today.
   This ensures index optimizations from RFC 0034 are preserved without duplication, and
   existing tests of `MatchX()` continue to provide coverage.

5. **Flat fields preserved for backward compatibility.** `Keys`, `Names`, `SearchTerm`,
   `Limit`, `Offset`, and entity-specific flat fields remain on `RetrieveRequest`. They
   are not deprecated. Clients that don't need boolean composition or ordering can
   continue using the flat fields with no changes. The API handler merges both sources.

6. **Single-field ordering first.** Gorp's `OrderBy` currently accepts one `OrderQuery`.
   Multi-field ordering (e.g., order by name asc, then by created_at desc as tiebreaker)
   requires extending `Retrieve` to accept a sequence of `OrderQuery` handles. This is a
   gorp-layer change that is deferred.

7. **Per-field typed OrderBy struct (mirrors filter design).** The `orderBy` field uses
   the same per-field typed slot pattern as the filter node. Each `@index sorted` field
   gets its own typed struct (e.g., `StringOrderBy`, `TimestampOrderBy`) with a typed
   `After` cursor. This avoids `any`, `json.RawMessage`, or string-based field dispatch
   for the cursor value. Only one field may be set per request, which is correct for
   single-field ordering. The cursor value is statically typed at the struct level: a
   `string` for name ordering, a `telem.TimeStamp` for timestamp ordering.

8. **Cursor pagination uses typed cursor values, not opaque tokens.** The `After` field
   on each OrderBy struct carries the raw cursor value rather than an opaque base64
   token. This is simpler, more debuggable, and sufficient for single-field ordering. If
   we later need multi-field cursors or server-side cursor state, opaque tokens can be
   introduced as a separate mechanism.

9. **Generated filter types live in the API package.** The filter node structs and
   interpreters are generated into the API package (e.g.,
   `core/pkg/api/channel/filter.gen.go`). These are wire types that belong alongside
   request/response types. The API package already imports the service package (e.g.,
   `api/channel` imports `distribution/channel` for `MatchX()` functions), so no new
   dependency direction is introduced.

10. **Custom filter logic behind leaf operators is acceptable.** Some `MatchX()`
    functions have non-trivial logic (e.g., `MatchVirtual` excludes calculated
    channels). The wire format exposes `{"virtual": {"eq": true}}`, which looks like a
    pure field comparison, but the server-side behavior is a semantic filter. This is
    acceptable because the interpreter delegates to the same `MatchX()` functions the
    API handler calls today. The protocol is a transport layer for invoking named
    filters, not a promise about their implementation. The `MatchX()` function is the
    contract, and it is tested independently.

# 7 - Open Questions

1. **Should the protocol support `not` on individual leaf operators?** For example,
   `{"name": {"not": {"eq": "x"}}}` vs requiring `{"not": {"name": {"eq": "x"}}}`. The
   former is more ergonomic for simple negation; the latter is simpler to generate and
   interpret. Hasura supports both; Prisma uses the latter. Recommend starting with
   node-level `not` only (the latter) and adding field-level `not` if needed.

2. **Validation depth limits.** Recursive filter trees can be arbitrarily deep. Should
   the server enforce a maximum depth to prevent abuse? Recommend a generous default
   (e.g., 16 levels) that can be adjusted.
