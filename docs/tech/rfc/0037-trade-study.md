# Trade Study: Upsert API Shape and Update Expression Syntax for `status.set`

Companion document to RFC 0037. Records the alternatives considered for the upsert API
shape and update expression syntax, and the reasoning behind the choices the RFC makes.

## Context

RFC 0037 proposes a `status` module with `set` as the central update primitive. The
design space for that primitive splits along two independent axes:

- **Axis 1: Upsert API shape.** What does the API surface itself look like? How is
  identity separated from content, and how do create-time and update-time payloads
  relate? This is the question the major ORMs (Prisma, Django, Rails, SQLAlchemy, Mongo,
  Postgres `ON CONFLICT`) answer differently.
- **Axis 2: Update expression syntax.** Once a payload is being assembled at the call
  site, how does the caller express it? One polymorphic call with optional fields, a
  pipe of pure transforms, methods on an object, a record-update literal? This is the
  question languages (Go, Clojure, Elixir, Kotlin, Smalltalk/Dart, Haskell) answer
  differently.

The two axes are orthogonal in theory: any shape from Axis 1 could in principle be
paired with any syntax from Axis 2. In practice each syntactic style has one or two
shapes it pairs with naturally, and the remaining combinations are forced. A "natural
pairings" matrix follows the option list.

The current RFC picks Shape 1 (shared payload, preserve-on-omit) on Axis 1, and Option A
(polymorphism) on Axis 2. This document evaluates both axes and recommends that pairing.

## Background: first-class records

Several shapes and options below assume Arc has **first-class records**, which it does
not today. The term carries the cost analysis for those alternatives, so it's worth
defining once up front rather than repeating per-section.

A **record** is a named, heterogeneous bundle of fields: a `struct` in Go, a class
instance in Python, an `interface` value in TypeScript:

```go
// Go
type Status struct {
    Name    string
    Message string
    Variant string
}
s := Status{Name: "Pressure Alert", Message: "High Pressure", Variant: "error"}
```

**First-class** means the language treats record values as ordinary values: store them
in a variable, pass them to a function, return them, put them in a channel, compare them
for equality. The type system knows what a `Status` is.

**What Arc has today.** Primitives (numbers, strings, bools, timestamps), series (typed
arrays), and channels. There is no way to declare a `Status` type, construct a `Status`
value, or pass one around. Everything Arc operates on is either a scalar or a series of
scalars.

**What adding first-class records would require.** This is the cost line in the
conclusion:

- **Type system.** A new kind of type, with field names and field types known to the
  analyzer.
- **Literal syntax.** Some way to write a record value
  (`Status{name="...", message="..."}`). Parser and grammar work.
- **Field access.** `my_status.message` has to typecheck and compile.
- **WASM layout.** A record needs a memory representation: packed fields, pointer-boxed,
  garbage-collected. Each choice has runtime cost.
- **Flow edges.** Today Flow channels carry scalars or series. Carrying a record means
  edges typed by record schemas, with serialization across the host boundary.
- **Mutability and equality.** Value-typed (copied) or reference-typed (shared)? Mutable
  or immutable? Decisions that propagate through the entire compiler.

A multi-week, RFC-sized undertaking. **Shapes 4 and 5** on Axis 1, and **Options C1, C2,
G, and H** on Axis 2, all presuppose it. Their Arc examples are written in a
hypothetical Arc that has records.

## Axis 1: Upsert API shape

### Shape 1: Shared payload, preserve-on-omit (current RFC)

One field set, applied as overrides on update or as defaults on create. Used by MongoDB
(`{$set: {...}}` with `upsert: true`) and by Synnax's existing Python and TS clients
(whose `set` upserts by primary key).

MongoDB:

```javascript
db.statuses.updateOne(
  { name: "Pressure Alert" },
  { $set: { message: "High Pressure", variant: "error" } },
  { upsert: true },
);
```

Synnax Python client:

```python
client.statuses.set(Status(
    name="Pressure Alert",
    message="High Pressure",
    variant="error",
))
```

Synnax TypeScript client:

```typescript
await client.statuses.set({
  name: "Pressure Alert",
  message: "High Pressure",
  variant: "error",
});
```

In Arc:

```go
// WASM
status.set("Pressure Alert", "High Pressure", "error")

// Flow
trigger -> status.set{identifier="Pressure Alert", message="High Pressure"}
```

Single payload; behavior depends on whether the row exists. Cheapest at the call site,
cheapest in the type system. Forces preserve-on-omit semantics so an absent field on
update doesn't clobber existing state, which §4.0 of the RFC already specifies. Matches
the API shape of the Synnax Python and TS clients, so callers crossing language
boundaries see one mental model.

### Shape 2: Split payloads (Prisma)

Caller writes two distinct field sets, one applied on update and one on create. Lets
them diverge: create-time defaults can differ from what an update touches.

Prisma:

```typescript
prisma.status.upsert({
  where: { name: "Pressure Alert" },
  update: { message: "High Pressure" },
  create: { name: "Pressure Alert", message: "High Pressure", variant: "info" },
});
```

In Arc:

```go
// Flow
trigger -> status.upsert{
    identifier="Pressure Alert",
    on_update={message="High Pressure"},
    on_create={message="High Pressure", variant="info"},
}
```

WASM positional form is impractical (nested config doesn't translate to flat positional
args). Forces nested config blocks in Flow, or two parameter sets across two call
shapes. Most flexible, most verbose. Useful when create and update genuinely diverge,
which is rare for status updates, where the same fields are meaningful in both
directions.

### Shape 3: Lookup + defaults (Django, Postgres `ON CONFLICT`)

Identity args are syntactically distinct from content args; content acts as
defaults-on-create and as overrides-on-update. Django writes
`update_or_create(defaults=..., **lookup_kwargs)`; Postgres writes
`INSERT (...) VALUES (...) ON CONFLICT (key) DO UPDATE SET ...`.

Django:

```python
Status.objects.update_or_create(
    name="Pressure Alert",
    defaults={"message": "High Pressure", "variant": "error"},
)
```

Postgres:

```sql
INSERT INTO statuses (name, message, variant)
VALUES ('Pressure Alert', 'High Pressure', 'error')
ON CONFLICT (name) DO UPDATE
SET message = EXCLUDED.message, variant = EXCLUDED.variant;
```

In Arc:

```go
// Hypothetical: defaults block separates identity from content
status.set("Pressure Alert", defaults={message="High Pressure", variant="error"})
```

With only one lookup field (`identifier`), this collapses onto Shape 1: there is no
identity-vs-content distinction worth surfacing because identity is one arg and content
is the rest. Shape 3 earns its weight when the lookup is multi-field (e.g.,
`WHERE rack_key = X AND name = Y`), which the status module does not need: statuses live
in one flat cluster-global namespace, identified by a single `name` (or its derived
`key`), with no scoping axis like rack, device, or task. If statuses ever grow such an
axis (e.g. per-rack statuses where two racks can each have their own "Pressure Alert"),
Shape 3 becomes the right answer.

### Shape 4: Whole-object merge (SQLAlchemy, TypeORM)

Caller builds a status value, hands it to a `merge` or `save` primitive, and the
persistence layer figures out create-vs-update from primary-key presence.

SQLAlchemy:

```python
status = Status(name="Pressure Alert", message="High Pressure", variant="error")
session.merge(status)
session.commit()
```

TypeORM:

```typescript
const status = repository.create({
  name: "Pressure Alert",
  message: "High Pressure",
  variant: "error",
});
await repository.save(status);
```

In Arc:

```go
// Hypothetical: requires first-class records (record literal, no DB call)
my_status := Status{name="Pressure Alert", message="High Pressure", variant="error"}
status.merge(my_status)
```

Requires first-class status records. The value construction (the record literal) is not
a DB call: only `merge` writes. Adjacent to (but distinct from) Option G on Axis 2: G is
_how to express a partial change to a record value_ (e.g.,
`Status{my_status | message="High"}`); Shape 4 is _what API call persists that value_.
They compose, but each is an independent decision.

### Shape 5: Find-then-modify (Rails ActiveRecord)

Caller fetches an existing-or-fresh value, mutates it, then saves. Rails exposes this as
`find_or_initialize_by(...)` plus block-style mutation plus `save`.

Rails:

```ruby
status = Status.find_or_initialize_by(name: "Pressure Alert")
status.message = "High Pressure"
status.variant = "error"
status.save!
```

In Arc:

```go
// Hypothetical: requires records, mutable refs, and a save primitive
my_status := status.find_or_init("Pressure Alert")
my_status.message = "High Pressure"
status.save(my_status)
```

The most language-heavy shape: first-class records _and_ mutable refs _and_ a save
primitive. Pairs naturally with imperative styles (Options D, E, F on Axis 2) but
doesn't compose with pipe-style (C1, C2) without extra machinery.

## Axis 2: Update expression syntax

### Option A: Polymorphism (current RFC, with `set`-as-upsert)

`set` takes a name (or key) plus named optional fields. Supplied fields overwrite,
omitted fields preserve. If the name doesn't exist, create.

```go
// WASM (positional only; see RFC §4.0)
status.set("Pressure Alert")                                  // touch
status.set("Pressure Alert", "High Pressure")                 // partial (message only)
status.set("Pressure Alert", "High Pressure", "error")        // full
// variant-only is not expressible in WASM today (no skip-middle, no name=value
// inside parens); use the Flow form below.

// Flow
trigger -> status.set{identifier="Pressure Alert", message="High Pressure"}
trigger -> status.set{identifier="Pressure Alert", variant="error"}
trigger -> status.set{identifier="Pressure Alert"}
```

One symbol, several call shapes, all reading the same at the surface.

**Natural shape pairing:** Shape 1 (shared payload). The polymorphism on field presence
_is_ the preserve-on-omit semantics; pairing with any other shape strips the benefit.

### Option B: Functional options (Go)

Each "option" is a function that mutates a config value. `set` takes a name plus a
variadic list of options. Idiomatic Go; used by gRPC, the standard `http` server,
Kubernetes client-go, and most modern Go libraries.

```go
// WASM
status.set("Pressure Alert",
    status.message("High Pressure"),
    status.variant(.error))

status.set("Pressure Alert", status.variant(.error))   // partial
status.set("Pressure Alert")                           // touch

// Flow
trigger -> status.set{
    identifier="Pressure Alert",
    options=[status.message("High Pressure")],
}
```

Solves the optional-args problem without polymorphism on `types.Param` and without
first-class records. The cost is per-field constructor functions (`status.message`,
`status.variant`, …) and a higher-order calling convention (passing functions as
arguments) which Arc does not currently support. Verbose at the call site for the common
case.

**Natural shape pairing:** Shape 1 (shared payload). Each option mutates one shared
config; the upsert API surface is unchanged from Option A's, only the call form differs.

### Option C1: Clojure-style threading (family of macros)

Status becomes a value type. Updates are pure functions chained through one of several
threading operators, typically `->` for the common case, plus `cond->` for conditional
updates (the partial-update sweet spot), `some->` for nil-aware chains, etc. Modeled on
Clojure's threading macros.

```go
// WASM: full update
status.get("Pressure Alert")
  -> status.message("High Pressure")
  -> status.variant(.error)
  -> status.save()

// WASM: partial update via cond-> (only updates fields whose predicates are truthy)
status.get("Pressure Alert")
  cond-> message  status.message(message)
  cond-> variant  status.variant(variant)
  -> status.save()

// Flow: Flow's `->` already pipes; threading is the natural shape
trigger -> status.get{"Pressure Alert"}
        -> status.message{"High Pressure"}
        -> status.variant{.error}
        -> status.save{}

// Flow: partial update via cond->
trigger -> status.get{"Pressure Alert"}
        cond-> message  status.message{message}
        cond-> variant  status.variant{variant}
        -> status.save{}
```

Each step is pure. `cond->` makes partial-update natural: the chain literally skips the
steps whose predicates are nil. The cost is a family of operators (`->`, `cond->`,
`some->`, `as->`, `->>`) that operators must learn and choose between.

**Natural shape pairing:** Shape 4 (whole-object merge). Threading needs a value flowing
through the chain; that value is a status record built up by pure transforms and
persisted by a terminal `save`. Forces first-class records.

### Option C2: Elixir-style pipe (single operator)

Same first-class value model as C1, but with a **single** pipe operator (`|>`). Partial
updates are expressed by simply omitting the step from the chain. There is no
conditional pipe. Conditional logic is handled with ordinary `if` blocks around chains
or by writing wrapper functions.

```go
// WASM: full update
status.get("Pressure Alert")
  |> status.put_message("High Pressure")
  |> status.put_variant(.error)
  |> status.save()

// WASM: partial update requires manual branching
my_status := status.get("Pressure Alert")
if message != nil {
    my_status = status.put_message(my_status, message)
}
if variant != nil {
    my_status = status.put_variant(my_status, variant)
}
status.save(my_status)

// Flow: same shape as C1 (Flow's `->` already pipes)
trigger -> status.get{"Pressure Alert"}
        -> status.message{"High Pressure"}
        -> status.variant{.error}
        -> status.save{}

// Flow: partial update; no cond->, branch outside the pipeline
if message != nil {
    trigger -> status.get{"Pressure Alert"}
            -> status.message{message}
            -> status.save{}
}
```

One operator to learn, one syntactic transformation. The partial-update case loses the
elegance C1 gets from `cond->` and reads as ordinary imperative branching wrapped around
the pipeline.

**Natural shape pairing:** Shape 4 (whole-object merge), same as C1. The pipe carries a
status record; partial-update branching wraps the pipeline rather than living inside it.

### Option D: Object-oriented method chaining (Java, Ruby, JavaScript, Rust builders)

Status is an object with methods. Each setter returns the receiver (or a new value),
enabling top-down chains via the dot operator. No new operator; the dot already exists.

```go
// WASM
status.find("Pressure Alert")
      .message("High Pressure")
      .variant(.error)
      .save()

status.find("Pressure Alert").message("High Pressure").save()   // partial
status.find("Pressure Alert").save()                            // touch

// Flow: methods don't chain on Flow edges; each "method" becomes its own stage
trigger -> status.find{"Pressure Alert"}
        -> status.message{"High Pressure"}
        -> status.variant{.error}
        -> status.save{}
```

The dot is already familiar, but Arc has no method dispatch today; adding it means
deciding receiver semantics, whether methods are first-class, whether subtyping or
inheritance enters the language, and how this interacts with Flow nodes. A bigger
language commitment than it sounds; "just methods" is rarely just methods.

**Natural shape pairing:** Shape 5 (find-then-modify); the chain is exactly "find,
mutate, save". Shape 4 also fits if methods are pure (return a new value); the choice
depends on whether Arc's records are mutable or persistent.

### Option E: Scope functions (Kotlin: `apply` / `with` / `let`)

A block scoped to one object, where field assignments inside the block bind to that
object implicitly. Familiar to anyone who's written Kotlin or Groovy. Pairs naturally
with OO method chaining (Option D) and is essentially unusable without it.

```go
// WASM
status.find("Pressure Alert").apply {
    message = "High Pressure"
    variant = .error
}.save()

// Partial: just include fewer assignments in the block
status.find("Pressure Alert").apply {
    variant = .error
}.save()

// Flow: apply blocks need a held receiver; collapse to a stage with field assignments
trigger -> status.find{"Pressure Alert"}
        -> status.apply{message="High Pressure", variant=.error}
        -> status.save{}
```

Reads like imperative mutation while still producing a value at the end. The new
language feature is closures with implicit receiver, a non-trivial addition to the type
system, since the body has to type-check against the receiver's fields.

**Natural shape pairing:** Shape 5 (find-then-modify). The `apply` block needs a held
receiver, and the receiver originates from a `find_or_init`-style call.

### Option F: Cascade operator (Smalltalk, Dart)

A new operator (`..` in Dart) sends each call to the same receiver, instead of relying
on each method returning `self`. Reads like a block of mutations on one object.

```go
// WASM
status.find("Pressure Alert")
  ..message = "High Pressure"
  ..variant = .error
  ..save()

// Flow: cascades require a held receiver; collapse to a single stage with multi-field config
trigger -> status.set{
    identifier="Pressure Alert",
    message="High Pressure",
    variant=.error,
}
```

Cheaper than full OO methods (no return-self contract, no dispatch question), but still
requires a new operator and an implicit receiver inside the cascade. Smalltalk's
original form treated each cascaded call as an independent message send; Dart's `..` is
a syntactic sugar over the same idea. Reads cleanly for "do N things to one object," but
loses the value-transformation semantics that make pipes composable.

**Natural shape pairing:** Shape 5 (find-then-modify). Cascades need a held receiver; in
Flow the cascade collapses back onto Shape 1 because edges don't carry an implicit
receiver across nodes.

### Option G: Record-update syntax (Haskell, F#, OCaml, Elixir)

Status is a record value, and a literal-update form produces a new record with a subset
of fields replaced. No pipe, no methods, just one expression that says "this status, but
with these fields changed."

```go
// WASM
my_status := status.get("Pressure Alert")
status.save({my_status | message: "High Pressure", variant: .error})

// Partial update: same syntax, fewer fields
status.save({my_status | variant: .error})

// Touch: no field updates at all
status.save(my_status)

// Flow: record values flow through the edge; an update stage replaces fields
trigger -> status.get{"Pressure Alert"}
        -> status.with{message="High Pressure", variant=.error}
        -> status.save{}

// Flow: partial update via fewer fields in the update stage
trigger -> status.get{"Pressure Alert"}
        -> status.with{variant=.error}
        -> status.save{}
```

This pattern most directly fits the "pass around objects" gesture: the value is named,
fields are named, the update is one expression, and preserve-on-omit is a property of
the language rather than of one symbol. The cost is first-class records plus a new
literal form; once those exist, it generalizes everywhere (ranges, devices, channels,
flow node configs).

**Natural shape pairing:** Shape 4 (whole-object merge). The record-update literal
produces a status value; that value is persisted by a terminal `save`/`merge` primitive.

### Option H: Lenses / optics (Haskell, Scala Monocle)

First-class field accessors that compose. A lens is a value that knows how to focus on
one field of a structure; setting via a lens produces a new structure with that field
changed. Lenses compose, so deeply-nested updates become one expression.

```go
// Hypothetical Arc: lenses applied via & and .~
my_status := status.get("Pressure Alert")
status.save(my_status & status.message .~ "High Pressure"
                      & status.variant .~ .error)

// Flow: lens application as an updater stage carrying lens values on the edge
trigger -> status.get{"Pressure Alert"}
        -> status.update{lenses=[status.message .~ "High Pressure",
                                  status.variant .~ .error]}
        -> status.save{}
```

The academically clean answer to "compose updates on nested data." Pays off enormously
for code that traverses deep object graphs (game state, ASTs, JSON). Almost certainly
overkill for status updates and a poor fit for control engineers. Mentioned only for
completeness.

**Natural shape pairing:** Shape 4 (whole-object merge). Lens application produces a new
record; that record is persisted by a terminal `save`/`merge`.

## Natural pairings

The two axes are independent decisions, but each Axis 2 syntactic style has one or two
Axis 1 shapes it pairs with cleanly. Forced pairings (e.g., Shape 4 with Option A, or
Shape 1 with Option G) read awkwardly because they fight each option's premise.

| Option (Axis 2)           | Natural shape (Axis 1)        | Why                                                        |
| ------------------------- | ----------------------------- | ---------------------------------------------------------- |
| **A: Polymorphism**       | **1** Shared payload          | Polymorphism on field presence _is_ preserve-on-omit       |
| **B: Func options**       | **1** Shared payload          | Each option mutates one shared config                      |
| **C1: Clojure threading** | **4** Whole-object merge      | Threading needs a value flowing through                    |
| **C2: Elixir pipe**       | **4** Whole-object merge      | Pipe carries a record; terminal `save` persists            |
| **D: OO methods**         | **5** Find-then-modify (or 4) | Chain is literally "find → mutate → save"                  |
| **E: Scope functions**    | **5** Find-then-modify        | `apply` block needs a held receiver                        |
| **F: Cascade**            | **5** Find-then-modify        | Cascade needs a receiver; Flow collapses back to Shape 1   |
| **G: Record-update**      | **4** Whole-object merge      | Update literal produces a record; terminal `save` persists |
| **H: Lenses**             | **4** Whole-object merge      | Lens application produces a record; terminal `save`        |

The current RFC picks **Shape 1 + Option A**. The pairing is mutually reinforcing:
shared-payload upsert needs preserve-on-omit, and polymorphism on field presence is the
call-site syntax that _expresses_ preserve-on-omit most directly.

## Evaluation

| Option                    | Type system change                   | Compiler/runtime change             | New surface syntax               | Partial-update UX               | Effort               |
| ------------------------- | ------------------------------------ | ----------------------------------- | -------------------------------- | ------------------------------- | -------------------- |
| **A: Polymorphism**       | `Optional bool` on `types.Param`     | Handle 0 dispatch in host fns       | None                             | Native (omit field)             | **Few days**         |
| **B: Functional options** | First-class functions (Arc lacks)    | Function values + variadic args     | Per-field constructor fns        | Verbose; pass options each call | Weeks                |
| **C1: Clojure threading** | First-class records                  | Records + macro family              | `->`, `cond->`, `some->`, `as->` | Strong via `cond->`             | Weeks                |
| **C2: Elixir pipe**       | First-class records                  | Records + pipe operator             | One operator (`\|>`)             | Manual `if` blocks              | Weeks                |
| **D: OO method chaining** | Method dispatch (subtyping question) | Receiver semantics + dispatch rules | None (dot exists)                | Drop chain steps                | Weeks + design churn |
| **E: Scope functions**    | Closures with implicit receiver      | Body type-checks against receiver   | `apply{}` block                  | Drop assignments                | Weeks; needs D first |
| **F: Cascade**            | Implicit receiver in cascade scope   | Cascade desugaring                  | One operator (`..`)              | Drop assignments                | Weeks                |
| **G: Record-update**      | First-class records                  | Records + literal-update form       | `{x \| f: v}` literal            | Native (fewer fields)           | Weeks                |
| **H: Lenses**             | First-class lenses + composition     | Lens application/composition        | `&` and `.~` (or equivalent)     | Drop lens steps                 | Weeks; academic fit  |

Notes on the cells:

- **"Weeks"** means substantial language work in Arc's compiler, type system, or
  runtime: first-class records, function values, method dispatch, or new operator
  semantics. Each is a standalone RFC.
- **"Native" partial-update** means the language form expresses partial-vs-full
  directly, with no caller branching.
- **Option E** is conditional on Option D landing first; scope functions don't compose
  without method chaining.
- **Options C1, C2, G** all share the "first-class records" type-system cost. If records
  land for any of those reasons, the others become cheaper.

## Conclusion

> **The pipe model is elegant for programmers writing data pipelines; it's friction for
> engineers writing control logic.**

**Axis 1 (upsert shape).** Status updates have one identity field (`identifier`), one or
two content fields (`message`, `variant`), and no meaningful divergence between
create-time defaults and update-time payloads. Shape 1 (shared payload,
preserve-on-omit) sits exactly at that point: identity is one positional/config arg,
content is the rest, and the same payload covers both branches. Shape 2 (split payloads)
earns its weight when create and update genuinely differ, and they don't here. Shape 3
(lookup + defaults) earns its weight when the lookup is multi-field, and it isn't here.
Shapes 4 and 5 (whole-object merge, find-then-modify) only pay off if Arc gains
first-class records, which is a separate RFC. Shape 1 also matches the existing Python
and TS clients, so callers crossing language boundaries see one mental model. The same
consistency argument that already justified collapsing `create` and `update` into a
single `set` (RFC §4.2) is the same argument for adopting the shape and syntax those
clients use.

**Axis 2 (update syntax).** Arc's users are control engineers writing imperative
sequences ("when this happens, set that status"). Option A matches their mental model:
one call, named arguments, behavior controlled by what's passed. Every alternative asks
them to reason about something else: value transformation chains (C1, C2), method
receivers (D, E, F), record literals (G), function-valued options (B), or composable
optics (H). That's friction without benefit for the dominant use case.

Every Axis 2 alternative is also substantially more expensive to build: weeks of
language-wide work versus a few days for Option A. The cost falls into three families of
infrastructure Arc doesn't currently have:

- **First-class records** (C1, C2, G, H): a new kind in the type system, literal syntax,
  WASM struct layout, and Flow edges that carry typed values. H additionally requires
  lens primitives, raising the bar further for an audience that doesn't write functional
  code.
- **Method dispatch / implicit receiver** (D, E, F): receiver semantics, dispatch rules,
  and (for E) closures that type-check against an enclosing receiver.
- **First-class functions** (B): function values and a higher-order calling convention.

Each family is generic infrastructure that should land for its own reasons in a separate
RFC, not as a prerequisite for status updates. Note that the same first-class-records
prerequisite also gates Shapes 4 and 5 on Axis 1, so the two axes' costs are correlated:
once records exist, several alternatives become cheaper simultaneously, but none of them
are cheap today.

**C1 vs C2.** If a pipe option ever does land, C2 (Elixir-style single operator) is the
cheaper and simpler entry point: one operator, one mental model, one set of compiler
rules. C1 (Clojure-style family) buys better partial-update ergonomics via `cond->`, but
at the cost of multiple operators that users must learn and choose between.
Partial-update ergonomics is exactly the case where Option A is already strongest, so
C1's main advantage over C2 is the case Option A already wins. That makes C2 the more
defensible "if not A, then this" choice, but neither is competitive with A on the time
horizon this RFC is operating on.

## Recommendation

Stay with **Shape 1 + Option A** as currently described in RFC 0037: shared-payload
upsert with preserve-on-omit semantics, expressed as a single polymorphic call with
optional named fields.

Concerns about a single axis of polymorphism (touch dispatch, name/key dispatch, the
create-vs-update implicit in `set`-as-upsert) are worth discussing individually; several
of those have smaller fixes than rebuilding the type system. A blanket objection to
"polymorphism" is too broad to action, and the alternatives on offer (first-class
records plus either a family of threading macros or a single pipe operator, plus either
whole-object merge or find-then-modify on Axis 1) are disproportionate to the problem
the RFC is trying to solve.
