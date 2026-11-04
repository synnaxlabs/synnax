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
generates a Go interpreter that turns a filter into the service's existing matchers, an
`Apply` that runs a request on a retrieve, and a TypeScript `matches` that keeps cached
lists live. Saved views store filters. The flat filter fields on retrieve requests go
away. Endpoints and handlers stay hand-written. Clients write a filter as a partial
record of the entity, and a key or a name still retrieves one record.

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

- **Bare line**: Names a required field of the target. Oracle picks its operators from
  the field type (§2.1), and `go/query` emits its matcher. `key` always maps to a key
  match, and takes its type from `@retrieve key` when the target has no key field.
- **Typed line**: Calls a matcher written by hand in the target's service package. The
  generated Go calls `MatchOverlaps(telem.TimeRange)`, and a `T[]` line calls a variadic
  matcher: `MatchLabels(...label.Key)`. A missing or mistyped function fails the Core
  build.
- **Target**: A struct or a union in the same schema (`Where filter View`). A schema
  declares at most one filter.

A line has no `?`, default, or domain removal. `and`, `or`, `not`, `where`, and the
option names of §2.7 are reserved. An optional field, or a field whose type is a type
parameter, needs a typed line. A union's discriminator needs a named enum
(`union on type Type`) to get a bare line.

The analyzer expands the definition into an ordinary struct, as `@create` expands into
`New` (`oracle/analyzer/create.go`). The type plugins emit it as a struct. They needed
fixes for recursive types, generic instances, and strict decoding (§3).

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

`go/query` builds one matcher list from bare lines and `@filter` fields, and the
interpreter reads names from it. A field can have both; it gets one matcher.

### 2.1 Operators

`schemas/x/filter.oracle` declares the operator types as ordinary Oracle structs. The
field type selects one:

| Field type          | Operator type        | Operators                                      |
| ------------------- | -------------------- | ---------------------------------------------- |
| `string`            | `filter.String`      | `eq`, `in`, `contains`, `starts_with`, `regex` |
| `bool`              | `filter.Bool`        | `eq`                                           |
| keys, UUIDs, enums  | `filter.Set<T>`      | `eq`, `in`                                     |
| numbers, timestamps | `filter.Ordered<T>`  | `eq`, `in`, `gt`, `gte`, `lt`, `lte`           |
| `T[]`               | `filter.Contains<T>` | `any`                                          |

A key is a `@key` field or a field whose type has the `@key` mark, such as `rack.Key`.
The mark goes on the type declaration, so a foreign key filters by value, not by range.
A distinct alias of `string` or `bool` gets `filter.Set<T>`, so its values keep their
type. All operators set on one slot are ANDed. "All of these labels" is an AND of nodes.
A regex is RE2 and unanchored. `filter.String` also has a `case_insensitive` flag, which
makes every operator on the slot ignore case, a regex included. Case is folded by the
Unicode simple case folding rule of the `(?i)` regex flag, in the Core and in the
TypeScript cache. An operator object that sets no operator, such as `name: {}` or
`case_insensitive` alone, is a validation error at its path, since it would match every
record.

The generic operator types are `@pb omit`. Protobuf gets one message per use, such as
`RackKeySet`, and no shared generic message.

### 2.2 The retrieve request

Oracle derives the request and the response from the definition:

```
RetrieveRequest {
    where          Where?         // absent selects every record
    search_term    string = ""    // the target has @search fields
    limit          uint32?        // absent sets no limit
    offset         uint32 = 0
    include_labels bool = false   // Range.labels has @include
    include_parent bool = false   // Range.parent has @include
}

RetrieveResponse {
    ranges Range[]
}
```

An absent field with a default has the default value, so a caller sets only what it
needs. The Core fills the defaults. The TypeScript request types leave them unset, so
the cache sees only the fields that a caller set.

The response holds the target. A generic target holds `T<any>`, since one table holds
records of every type argument. `@response type T` on the definition replaces the target
with T, and the `include_*` fields then come from the fields of T. Channels return the
wire-only `APIChannel`. The request and the response go wherever the definition goes, so
an `omit` on the definition omits them too. In Go, `@go request_output` on the
definition puts them in the API package (`core/pkg/api/ranger`).

The gRPC services return the generated `RetrieveResponse`. Their requests keep a
transport message that holds the generated request, since the generated field numbers
would reuse the reserved numbers of the old flat fields. Protobuf wraps each list in a
message, so an empty `in` arrives as an empty list and matches no record, as it does
over JSON.

### 2.3 The Core

The service owns what a filter means: the `Where` type, its interpreter, its validation,
and the typed-line matchers. The API owns the request, the response, and the access
checks. Clients only build trees.

A new Go plugin generates the interpreter. `MatchWhere` walks a `Where` and returns the
service's bound filter, built from `And`, `Or`, `Not`, and the matchers.
`RetrieveRequest.Apply` applies the where, the search term, the limit, and the offset to
a retrieve. For a generic target such as `status.Status<D>`, `MatchWhere` is generic,
and `Apply` takes the retrieve over `any`. The API handler stays hand-written: it calls
`Apply`, runs the query, the joins that `include_*` asks for, and access checks. The
status join that channel, device, rack, and task each copy today becomes one shared
helper.

The generated request decodes strictly. An unknown key anywhere in the request is an
error, so a stale client that sends `keys` fails instead of matching every record. Go
rejects unknown JSON members at any depth. Every filter, operator, and request is a
`z.strictObject` in TypeScript and has `extra="forbid"` in Python. A saved view decodes
leniently in the Core, since stored data is not validated on read. The gRPC translators
call `pb.RejectUnknown` (`x/go/pb/unknown.go`), which walks nested messages. A bad
filter, such as an invalid regex, returns a `validate.PathError`
(`x/go/validate/errors.go:33`) with the path to the bad value (`where.or.1.name.regex`).
The TypeScript client receives it as a `PathError`.

A filter never returns `ErrNotFound`. `Apply` sets Gorp's `SkipMissingKeys` option on
the retrieve, so a key with no record is a non-match anywhere in the tree. A typed line
that names another resource, such as `subjects` on policies or `for` on labels, matches
nothing when that resource does not exist. Lookups that the Core does for itself stay
strict. The `ignore_not_found_error` field goes away.

The channel filter has a `name_patterns` line. A valid channel name matches exactly, and
any other pattern is a regular expression, anchored when unanchored. Clients send names
as the caller gives them.

### 2.4 The TypeScript cache

Oracle generates `matches` and `serverComputed` for each entity. Bare lines are checked
in generated code. Typed lines take functions from a `Lines` table that the hand-written
client passes in: `matches(lines, record, where)`. An entity with no typed lines has no
table. An entry is a function or an explicit `null` for a line only the Core can check;
a missing entry is a type error. A filter with a `null` line or a regex is
server-computed and refetches (RFC 0047 §5.1). An absent `where` matches every record.

One case table, `core/testdata/filter/vectors.json`, runs against the Go interpreter and
the TypeScript `matches`, as `x/testdata/crdt/vectors.json` does for CRDTs. It lives in
the Core, since it holds Synnax entities.

The list pager sends `searchTerm: ""` and `offset: 0` on the first page
(`lyra/src/list/pager.ts:51-71`). The cache counts any non-null server field as set
(`client/ts/src/query/query.ts:703`), so every explorer list refetches today. The cache
counts an empty string and a zero `offset` as unset, since they ask the Core for
nothing. The pager keeps `offset: 0`, which marks the first page as fetched.

### 2.5 Saved views

A view stores only `where`, never a search term, limit, or offset. The `range` and
`status` variants of `View` replace `query` with `where`.

A stored `Where` has versions in its target's chain, as `color.Color` does in
`schemas/x/versions/color` and ranger's `Ref` does in `schemas/synnax/versions/ranger`.
The view pins that version. A change to a shipped `Where` mints a new version of the
target's chain, where the target itself stays an alias line (`Range = v1.Range`), so
stored ranges do not migrate. The pin check then makes the view follow. Version files
accept `filter` definitions; they still reject actions.

The view writer checks a stored `where` with the target's `MatchWhere`, and rejects a
view of an unknown variant.

Core v0.60.0 shipped view v1 (#3306) with `query`, so the change mints view v2. Its
migration maps `has_labels` to the `labels` line with the same meaning, and an empty
`query` to an absent `where`.

### 2.6 Single-record retrieves

A client method that returns one record raises not-found when no record matches, and
multiple-found when more than one matches. This is true in every client and language, as
in Django `get`, SQLAlchemy `one`, and Entity Framework `Single`. In Python,
`ignore_not_found=True` returns `None` when no record matches, and still raises when
more than one matches. A list retrieve drops missing keys. A caller that needs every
record checks for itself, as the framer and the C++ channel client do.

The error states the filter as a sentence of one form in every client, with the field
names of the client's language:
`Expected one range where name starts with "press", but found 2`, or
`Found no range where key = 12`.

### 2.7 The client API

Three rules shape the client API. The common call is the shortest. The same names and
structure work in every language, in that language's idiom. A mistake fails in the
editor, not on the Core.

A filter is a partial record. A call names fields of the entity and gives each a value,
and the client turns them into the `where` of §2.2. Nothing is exported per entity.

- **A plain value**: Matches records whose field equals it. On a list field, such as
  `labels`, it matches records that hold it.
- **A list**: Matches records whose field equals any element. On a list field, it
  matches records that hold any element.
- **An operator**: An object of the operators in §2.1 in TypeScript
  (`{ startsWith: "Hot" }`), or a function from `sy.filter` in Python and `x::filter` in
  C++ (`starts_with("Hot")`).

```ts
await client.channels.retrieve("pressure_1");
await client.tasks.retrieve({ type: "ni_analog_read", rack: rack.key });
await client.channels.retrieve({
  name: { contains: "pressure", caseInsensitive: true },
  limit: 20,
});
await client.ranges.retrieve({
  or: [{ name: { startsWith: "Hotfire" } }, { labels: anomaly.key }],
  overlaps: lastWeek,
});
```

```python
client.channels.retrieve("pressure_1")
client.tasks.retrieve(type="ni_analog_read", rack=rack.key)
client.channels.retrieve(
    name=sy.filter.contains("pressure", case_insensitive=True), limit=20
)
client.ranges.retrieve(
    ranger.Where(name=sy.filter.starts_with("Hotfire"))
    | ranger.Where(labels=anomaly.key),
    overlaps=last_week,
)
```

```cpp
client.channels.retrieve("pressure_1");
rack.tasks.retrieve({.type = "ni_analog_read", .rack = rack.key});
client.channels.retrieve(
    {.name = x::filter::contains("pressure", {.case_insensitive = true})},
    {.limit = 20}
);
client.ranges.retrieve(
    ranger::Where{.name = x::filter::starts_with("Hotfire")} ||
    ranger::Where{.labels = {anomaly.key}}
);
```

- **Fields and options**: TypeScript takes one object and Python takes keywords. Both
  hold filter fields and the options `limit`, `offset`, `searchTerm`, `one`, and each
  `include*` field, side by side. The client splits them. No field can collide with an
  option, since the filter definition reserves the option names (§2.0). C++ takes a
  `Where` and an options struct, since `Where` is a generated struct.
- **Combining**: All fields in one call must match. TypeScript nests with the `and`,
  `or`, and `not` keys. Python combines `Where` values with `&`, `|`, and `~`, and C++
  with `&&`, `||`, and `!`. Python cannot overload `and` and `or`, so `Where.__bool__`
  raises a `TypeError`, and `a and b` cannot silently drop `a`.
- **Shortcuts**: A key or a name as the first argument returns one record by the rule of
  §2.6, and a list of either returns a list. Options follow it:
  `retrieve("pressure_1", { includeStatus: true })`, `include_status=True` in Python,
  and `{.include_status = true}` in C++. In TypeScript and C++, a device takes only a
  key, since a device key is free text and a name can look like a key.
- **One record**: A plain `key` or `name` value in a filter returns one record by the
  rule of §2.6. A list or an operator on those fields returns a list. Any other filter
  returns a list unless `one` is set. `one` rejects a limit or an offset, since a page
  could hide a second match. C++ picks a return type when it compiles, so
  `{.name = "x"}` returns a list there, and a filter that returns one record takes the
  tag `x::filter::one` before the options.
- **Absent values**: A field whose value is `undefined` or `None` throws, since dropping
  it would match every record.
- **The tree**: The shorthand is only input. Saved views, the cache, and errors hold the
  tree of §2.0. TypeScript turns the shorthand into the tree with the generated Zod
  input, Python with the generated Pydantic model, and C++ with the generated field
  type, which takes a value or a list and keeps the `std::optional` interface. A stored
  tree, such as a saved view's `where`, goes into a call as it is.
- **Types**: A field that the entity does not have, or a value of the wrong type, fails
  to compile in TypeScript and C++, and fails `mypy` in Python.
- **Names**: Names differ only by casing across languages. A name that a language
  reserves takes a trailing underscore there: Python `and_`, `or_`, `not_`, `in_`, and
  `for_`, and C++ `and_`, `or_`, `not_`, and `for_`.
- **Shipped fields**: For one minor release, TypeScript and Python still take the flat
  fields that they shipped, such as `names` and `hasLabels` (`has_labels=` in Python).
  Each one warns with its new name, through `console.warn` and a `FutureWarning`. C++
  has no such release.

Each choice follows prior art. A partial record with operator objects follows Prisma,
Drizzle v2, the Linear SDK, Hasura, and Directus. Plain values for equality follow
Prisma and MongoDB, and Python keyword fields follow Django. Python `&`, `|`, and `~`
follow SQLAlchemy, Django `Q`, and Polars, and C++ `&&`, `||`, and `!` follow sqlpp11
and ODB. Drizzle throws on `undefined` in a filter, since an absent value once dropped a
condition and returned the wrong user. `one` follows SQLAlchemy `one()`, LINQ `Single`,
and Rails `sole`. The case flag follows Elasticsearch's `case_insensitive`. We differ in
one place: no C++ library we checked writes a filter as designated initializers, so the
C++ form rests only on C++20 rules.

## 3 Implementation phases

Each phase is one PR into `main`. An entity moves to `where` in one PR: its callers move
and its flat fields go away together, since a client must match the Core's version.

- **Phase 1: This RFC.**
- **Phase 2: Oracle.** All generator work, with specs for each shape and no schema that
  uses `filter` yet:
  - The matcher list in `go/query`, and a generator error for `@filter` on a `@key`
    field.
  - The operator types in `schemas/x/filter.oracle`. The protobuf plugin emits one
    message per generic instantiation; it ignores type arguments today.
  - The `filter` definition: the grammar, the analyzer expansion into `Where`, the
    formatter, the LSP, the TextMate grammar, and a recursion test for every type
    plugin.
  - Type plugin fixes: recursive TypeScript types, a protobuf message per generic
    instance, C++ type parameter fields, union decoding under strict decoding, and codec
    fixtures for recursive types.
  - Request derivation: `RetrieveRequest`, `RetrieveResponse`, defaults, and `@include`.
  - The `@key` mark on type declarations.
  - Version files that accept `filter` definitions, and `oracle migrate` support.
  - The Go interpreter plugin with `Apply`, strict decoding, and path errors, Gorp's
    `SkipMissingKeys` option, and the TypeScript `matches` plugin.
  - `@go request_output`, and protobuf list messages that keep an empty list present.
- **Phase 3: Ranges and statuses.** The two entities with saved views move to `where` in
  every language:
  - Schemas, handlers, and behavior specs for AND, OR, NOT, and each operator.
  - The shared case table, the server-computed rule in the cache, and the empty-value
    fix.
  - `pb.RejectUnknown` for the gRPC translators, and `query.keysRequest` for the
    key-list requests of the TypeScript clients.
  - The TypeScript, Python, and C++ clients, with retrieve-by-key specs, and the Pluto
    queries.
  - Saved views that store `where`.
  - The flat fields, `ignore_not_found_error`, and the hand-written `requestMatches` go
    away. Old protobuf field numbers are reserved.
- **Phase 4: Channels, devices, racks, and tasks.** The same move, plus the shared
  status-join helper that all four use. A channel retrieve takes an optional range next
  to the request (`AliasedRetrieveRequest` in Go). The Core fills each channel's alias
  in that range. The range alias service puts the channels whose alias matches the
  search term first, and pages the merged list. Clients send channel names as
  `name_patterns`.
- **Phase 5: Every other entity with an API retrieve.** The same move. Control, license,
  ontology, and range alias retrieves keep their requests. The access API splits into
  policy and role packages, since each generates its own `RetrieveRequest`.
- **Phase 6: Single-record retrieves.** Every client follows §2.6, with specs. Python
  gets one shared helper. The TypeScript cache sends a single answer back to the Core
  when a second record matches it. The Driver checks for tasks of a type with a list
  retrieve, since more than one is not an error there.
- **Phase 7: Text operators.** `contains`, `starts_with`, and `case_insensitive` on
  `filter.String`, in the interpreter, the TypeScript `matches`, and the case table.
- **Phase 8: TypeScript partial records.** The TypeScript type plugin generates filter
  inputs that take plain values and lists. Every `retrieve` takes the flat form of §2.7,
  and the Console and Pluto move to it. The shipped flat fields warn for one release.
- **Phase 9: Python partial records.** The Python type plugin generates filter models
  that take plain values and lists, with `&`, `|`, and `~`, and `sy.filter` gets the
  operator functions. `retrieve` takes fields as keywords, and the plural keywords warn
  for one release. A docs page teaches filters in both languages.
- **Phase 10: C++ partial records.** The C++ type plugin generates filter fields that
  take plain values and lists, with `&&`, `||`, and `!`, and `x::filter` gets the
  operator functions and `one`. `retrieve` takes a `Where` and options, and replaces
  `RetrieveOptions` and the `retrieve_by_*` methods. The Driver moves to it.
- **Phase 11: Empty operators.** The generated matcher rejects an operator object that
  sets no operator, with specs in the Core and the TypeScript client.
- **Phase 12: Filter sentences.** TypeScript, Python, and C++ each get a `describe` that
  writes a filter as a sentence, and the single-record errors of §2.6 use it.

| Phase | Scope                                                    | Touches                                                   | Depends on |
| ----- | -------------------------------------------------------- | --------------------------------------------------------- | ---------- |
| 1     | Write the Oracle filter tree RFC                         | `docs/tech/rfc`                                           | -          |
| 2     | Generate filter trees from Oracle schemas                | `oracle`, `schemas/x`, `x`, `gorp`                        | 1          |
| 3     | Move ranges and statuses to filter trees                 | `schemas`, `core`, `client`, `pluto`, `console`           | 2          |
| 4     | Move channels, devices, racks, and tasks to filter trees | `schemas`, `core`, `client`, `pluto`, `console`, `driver` | 3          |
| 5     | Move every other entity to filter trees                  | `schemas`, `core`, `client`, `pluto`, `console`           | 4          |
| 6     | Check single-record retrieves in every client            | `client`, `driver`                                        | 5          |
| 7     | Add text operators to string filters                     | `schemas/x`, `x`, `client/ts`                             | 6          |
| 8     | Write TypeScript filters as partial records              | `oracle/plugin/ts`, `client/ts`, `pluto`, `console`       | 7          |
| 9     | Write Python filters as partial records                  | `oracle/plugin/py`, `x/py`, `client/py`, `site/docs`      | 8          |
| 10    | Write C++ filters as partial records                     | `oracle/plugin/cpp`, `x/cpp`, `client/cpp`, `driver`      | 9          |
| 11    | Reject filter operators that set no operator             | `oracle/plugin/go/filter`, `core`, `client/ts`            | 10         |
| 12    | Describe filters in single-record errors                 | `x`, `client`, `driver`, `console`                        | 11         |

## 4 What this RFC does not cover

- **The Console filter builder**: The UI to add conditions and group them with AND, OR,
  and NOT is a separate design. The explorers keep their current label and variant
  filters, run on the tree.
- **Endpoints**: Routes, `core/pkg/api/layer.go`, `.proto` services, gRPC translators,
  and client calls stay hand-written.
- **Index types**: A filter line without an index scans the table.
- **Sorting**: Requests have no sort field. A sort needs a `SortedIndex`
  (`gorp/index.go:207`), since a scan and sort would be a second sort path, and no
  entity has one yet. How `@index sorted` names a sub-field, such as `time_range.start`,
  waits for that work too.

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
6. **Flat-field compatibility at cutover, rejected**: A client must match the Core's
   major and minor version (`client/ts/src/connection/status.ts:261`). Strict decoding
   makes a stale client fail loudly.
7. **The name `Filter`, rejected**: `go/query` already emits `Filter` in every service
   package (`core/pkg/service/ranger/retrieve.gen.go:60`).
8. **A handler that skips `ErrNotFound`, rejected**: The skip also hides a not-found
   from a lookup inside a matcher, which is a real fault. The retrieve drops missing
   keys instead (§2.3).
9. **Requests in the service package, rejected**: Only the API uses them, and the API
   owns envelopes (`core/CLAUDE.md`).
10. **A lenient key matcher, rejected**: A flag on one matcher spreads through `and` and
    `or`, so a strict sibling drops its missing keys too. One retrieve option states the
    rule once.
11. **Name patterns in the clients, rejected**: Each client would hold its own copy of
    the rule, and the C++ client had none.
12. **A generated builder, rejected**: Each entity exported its lines, `and`, `or`,
    `not`, and option helpers: 245 exports in TypeScript alone, each a thin wrapper over
    the tree. The Console never called it, since Flux parameters must be data, and it
    wrote 146 `where` objects. Of the libraries we checked, only Ent and the deprecated
    Prisma Go client generate an `or` per entity.
13. **A callback that receives the fields, rejected**: `retrieve((r) => r.name.eq(n))`,
    as in Kysely, Convex, and Zero, removes the exports. In Python the lambda only
    carries a namespace that a module already gives, and TypeScript keeps a second form
    beside the data that the Console passes.
14. **Equality through an operator only, rejected**: `{ type: { eq: t } }` makes the
    most common filter the longest. The real risk of a plain value is an absent one:
    Prisma ignores an `undefined` key, so a delete by `{ id: undefined }` deletes every
    record. The client throws on `undefined` and `None` instead.
15. **A `where` key around the fields, rejected**: `{ where: { type: t }, limit: 20 }`
    nests one level for no gain. Prisma and Drizzle need the key, since their records
    can have a field named `limit`. A filter definition here reserves the option names.
16. **Case-insensitive operators, rejected**: `containsIgnoreCase` and its siblings
    double the text operators. One flag covers all of them. Prisma's flag fails on
    databases that cannot run it, but the Core runs every match itself.
17. **Lower-casing to ignore case, rejected**: Go `ToLower`, JavaScript `toLowerCase`,
    and the `(?i)` flag disagree on letters such as the Greek final sigma and the long
    s, so the Core and the cache would disagree on a record. Simple case folding is the
    rule of the regex engines in both languages.
18. **Errors that print the filter as JSON or as code, rejected**: JSON lists every
    unset field, and code differs by language. A sentence reads the same in every
    client.
19. **An empty retrieve as an error, rejected**: SQL, Prisma, MongoDB, and Kubernetes
    all return every record for an empty query, and an empty `and` matching every record
    keeps the tree rules free of special cases. No delete or update takes a filter, so
    an empty filter cannot destroy data.

## 6 Open questions

1. **Optional fields**: Whether bare lines on optional fields get an `is set` operator.
