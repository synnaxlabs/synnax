# 85 Oracle filter trees

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-25
- **Related**: [RFC 0034 - Gorp in-memory indexes](0034-gorp-indexes.md),
  [RFC 0047 - Client cache, unified reads, and Console session state](0047-client-cache-unified-reads-session-state.md),
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md)

## 0 Summary

A new Oracle definition, `filter`, lists what an entity can be filtered by. From that
one block, Oracle generates a recursive filter type with `and`, `or`, and `not`, and the
retrieve request and response, in Go, TypeScript, Python, C++, and protobuf. It also
generates a Go interpreter that turns a filter into the service's existing matchers, and
a TypeScript `matches` that keeps cached lists live. Saved views store filters. The flat
filter fields on retrieve requests go away. Endpoints and handlers stay hand-written.

## 1 Motivation

The explorers can only filter by what a retrieve request has a field for. A range view
filters by labels and nothing else (`RangeQuery` in `schemas/synnax/view.oracle`). There
is no OR, no NOT, no name match, and no sort.

Each retrieve request is a flat AND of fixed fields, written by hand in each language:
Go (`core/pkg/api/ranger/ranger.go:133`), TypeScript
(`client/ts/src/ranger/client.ts:247`), Python
(`client/py/synnax/ranger/retrieve.py:19`), and gRPC
(`core/pkg/transport/grpc/ranger/ranger.proto:51`). The copies drift. The gRPC request
has only `keys` and `names`, so C++ cannot filter ranges by label. The TypeScript cache
needs a hand-written copy of the Core's filter logic for each entity to keep lists live
(`client/ts/src/ranger/client.ts:783`).

The `sy-4057-generate-api-level-retrieve-logic` branch tried this first and stopped. Its
tree shape was right. It failed on the rest: operator structs written by hand in three
languages, separate TypeScript and Python plugins, `@filter` on a `@key` field emitting
a second `MatchKeys`, flat fields deleted while callers still used them, and bad input
ignored instead of rejected. This design keeps the tree and drops the rest.

## 2 Design

### 2.0 The `filter` definition

A `filter` is a top-level definition, next to `struct`, `enum`, and `union`:

```
Where filter Range {
    key
    name
    overlaps telem.TimeRange {
        @doc value "matches ranges whose time range overlaps the given range."
    }
    labels   label.Key[]
}
```

- **Bare line**: Names a field of the target. Oracle picks its operators from the field
  type (§2.1), and go/query emits its matcher. `key` always maps to `MatchKeys`.
- **Typed line**: Calls a matcher written by hand in the target's service package. The
  generated Go calls `MatchOverlaps(telem.TimeRange)`, and a `T[]` line calls a variadic
  matcher: `MatchLabels(...label.Key)`. A missing or mistyped function fails the Core
  build.
- **Target**: A struct or a union (`Where filter View`).

The analyzer expands the definition into an ordinary struct, as `@create` expands into
`New` (`oracle/analyzer/create.go`), so every type plugin emits it with no change.

```
Where struct {
    and      Where[]?
    or       Where[]?
    not      Where?
    key      filter.Set<Key>?
    name     filter.String?
    overlaps telem.TimeRange?
    labels   label.Key[]?
}
```

The set slots on one node are ANDed. `and` matches when all children match, `or` when
any child matches, and `not` when its child does not. An empty node or an empty `and`
matches every record. An empty `or`, `in`, or `any` matches none. A key that does not
exist is a non-match, never `ErrNotFound`.

go/query builds one matcher list from bare lines and `@filter` fields, and the
interpreter reads names from it. A field can have both; it gets one matcher.

### 2.1 Operators

`schemas/x/filter.oracle` declares the operator types as ordinary Oracle structs. The
field type selects one:

| Field type                  | Operator type        | Operators                            |
| --------------------------- | -------------------- | ------------------------------------ |
| `string`                    | `filter.String`      | `eq`, `in`, `regex`                  |
| `bool`                      | `filter.Bool`        | `eq`                                 |
| `@key` fields, UUIDs, enums | `filter.Set<T>`      | `eq`, `in`                           |
| numbers, timestamps         | `filter.Ordered<T>`  | `eq`, `in`, `gt`, `gte`, `lt`, `lte` |
| `T[]`                       | `filter.Contains<T>` | `any`                                |

All operators set on one slot are ANDed. "All of these labels" is an AND of nodes. A
regex is RE2, and the TypeScript API takes a pattern string, with `(?i)` for case.

### 2.2 The retrieve request

Oracle derives the request and the response from the definition:

```
RetrieveRequest {
    where          Where?
    search_term    string?     // the target has @search fields
    order_by       Order?      // the target has @index sorted fields
    limit          uint32?
    offset         uint32?
    include_labels bool?       // Range.labels has @include
    include_parent bool?       // Range.parent has @include
}

RetrieveResponse {
    ranges Range[]
}
```

`Order` holds a `field`, from the `OrderField` enum of `@index sorted` fields, and a
`direction`. Gorp's `OrderBy` walks a `SortedIndex` (`gorp/index.go:207`), so a field
without the index cannot sort. When both are set, `order_by` replaces search relevance.
The response holds the target type, unless `@response type T` on the definition names
another. Channels return the wire-only `APIChannel`.

### 2.3 The Core

A new Go plugin generates the interpreter. `MatchWhere` walks a `Where` and returns the
service's bound filter, built from `And`, `Or`, `Not`, and the matchers. `OrderingOf`
turns an `Order` into the service's `Ordering`. For a generic target such as
`status.Status<D>`, both are generic functions. The API handler stays hand-written: it
runs the filter, the joins that `include_*` asks for, and access checks. The status join
that channel, device, rack, and task each copy today becomes one shared helper.

The generated request decodes strictly. An unknown key anywhere in the request is an
error, so a stale client that sends `keys` fails instead of matching every record. A bad
filter, such as an invalid regex, returns a `validate.PathError`
(`x/go/validate/errors.go:33`) with the path to the bad value (`where.or.1.name.regex`).
The TypeScript client receives it as a `PathError`.

The handler skips `ErrNotFound` once, because a filter is a predicate. The
`ignore_not_found_error` field goes away. Client methods that return one record still
raise not-found.

### 2.4 The TypeScript cache

Oracle generates `matches(record, where)` for each entity. Bare lines are checked in
generated code. Typed lines take functions from a table that the hand-written client
passes in. An entry is a function or an explicit `null` for a line only the Core can
check; a missing entry is a type error. A filter with a `null` line or a regex is
server-computed and refetches (RFC 0047 §5.1).

One case table, `x/testdata/filter/vectors.json`, runs against the Go interpreter and
the TypeScript `matches`, as `x/testdata/crdt/vectors.json` does for CRDTs.

The list pager sends `searchTerm: ""` and `offset: 0` on the first page
(`lyra/src/list/pager.ts:51-71`). The cache counts any non-null server field as set
(`client/ts/src/query/query.ts:703`), so every explorer list refetches today. The cache
counts an empty string and a zero `offset` as unset, since they ask the Core for
nothing. The pager keeps `offset: 0`, which marks the first page as fetched.

### 2.5 Saved views

A view stores `where` and `order_by`, never a search term, limit, or offset. The `range`
and `status` variants of `View` replace `query` with these two fields.

A stored `Where` has versions in its target's chain, as `color.Color` does in
`schemas/x/versions/color` and ranger's `Ref` does in `schemas/synnax/versions/ranger`.
The view pins that version. A change to a shipped `Where` mints a new version of the
target's chain, where the target itself stays an alias line (`Range = v1.Range`), so
stored ranges do not migrate. The pin check then makes the view follow. Version files
accept `filter` definitions; they still reject actions.

View v1 (#3306) is in no stable release. The change edits v1 in place if it lands before
the next stable release, and mints v2 otherwise. `has_labels` maps to the `labels` line
with the same meaning.

## 3 Implementation phases

Each phase is one PR into `main`. An entity moves to `where` in one PR: its callers move
and its flat fields go away together, since a client must match the Core's version.

- **Phase 1: This RFC.**
- **Phase 2: Oracle.** All generator work, with specs for each shape and no schema that
  uses `filter` yet:
  - The matcher list in go/query, and a generator error for `@filter` on a `@key` field.
  - The operator types in `schemas/x/filter.oracle`. The protobuf plugin emits one
    message per generic instantiation; it ignores type arguments today.
  - The `filter` definition: the grammar, the analyzer expansion into `Where`, the
    formatter, the LSP, and a recursion test for every type plugin.
  - Request derivation: `RetrieveRequest`, `RetrieveResponse`, `Order`, and `@include`.
  - Version files that accept `filter` definitions, and `oracle migrate` support.
  - The Go interpreter plugin with strict decoding and path errors, and the TypeScript
    `matches` plugin.
- **Phase 3: Ranges and statuses.** The two entities with saved views move to `where` in
  every language:
  - Schemas, handlers, and behavior specs for AND, OR, NOT, and each operator.
  - The shared case table, the server-computed rule in the cache, and the empty-value
    fix.
  - The TypeScript, Python, and C++ clients, with retrieve-by-key specs, and the Pluto
    queries.
  - Saved views that store `where` and `order_by`.
  - The flat fields, `ignore_not_found_error`, and the hand-written `requestMatches` go
    away. Old protobuf field numbers are reserved.
- **Phase 4: Channels, devices, racks, and tasks.** The same move, plus the shared
  status-join helper that all four use.
- **Phase 5: Every other entity with an API retrieve.** The same move.

## 4 What this RFC does not cover

- **The Console filter builder**: The UI to add conditions and group them with AND, OR,
  and NOT is a separate design. The explorers keep their current label and variant
  filters, run on the tree.
- **Endpoints**: Routes, `core/pkg/api/layer.go`, `.proto` services, gRPC translators,
  and client calls stay hand-written.
- **Index types**: A filter line without an index scans the table.

## 5 Resolved decisions

1. **Hand-written filter types, rejected**: Each entity would carry `and`, `or`, `not`,
   and an operator type per field, written by hand. The definition derives all of it.
2. **`@filter` rules or a struct member, rejected**: Version files alias one top-level
   definition at a time. Inside `Range`, each shipped filter change would mint a new
   `Range` version and migrate every stored range. Actions can live inside a struct
   because nothing stores them. A member form needs a new member-level version rule.
3. **Generated endpoints and client methods, rejected**: Oracle would need to declare
   every endpoint, not only retrieves. A client call is one line.
4. **Hand-declared requests, rejected**: Every entity gets the same request shape. Only
   `@include` is per field.
5. **Search as a filter line, rejected**: Search sorts by relevance
   (`gorp/search.go:99-101`). A nested line cannot.
6. **Sorting without an index, rejected**: A scan and sort would be a second sort path
   next to `SortedIndex`.
7. **Flat-field compatibility at cutover, rejected**: A client must match the Core's
   major and minor version (`client/ts/src/connection/status.ts:261`). Strict decoding
   makes a stale client fail loudly.
8. **The name `Filter`, rejected**: go/query already emits `Filter` in every service
   package (`core/pkg/service/ranger/retrieve.gen.go:60`).

## 6 Open questions

1. **Optional fields**: Whether bare lines on optional fields get an `is set` operator.
2. **Nested sort keys**: How `@index sorted` names a sub-field, so ranges sort by
   `time_range.start`.
