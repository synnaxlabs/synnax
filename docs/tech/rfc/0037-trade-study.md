# Trade Study: Polymorphism vs Pipe/Threading for `status.set`

Companion document to RFC 0037. Not part of the RFC. Not committed.

## Context

RFC 0037 proposes a `status` module with `set` as the central update primitive. `set` is
polymorphic on:

- which fields are supplied (touch / partial / full),
- whether the first argument is a key or a name,
- and (under the agreed-upon `set`-as-upsert direction) whether the name already exists.

This trade study evaluates that polymorphism against alternatives that model status as a
first-class value passed between functions, with syntax-level constructs (functional
options, Clojure threading, Elixir pipe, method chaining, scope functions, cascade,
record-update, lenses) replacing optional-field dispatch. It recommends a path forward.

## The options

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
first-class records. The cost is per-field constructor functions (`with_message`,
`with_variant`, …) and a higher-order calling convention (passing functions as
arguments) which Arc does not currently support. Verbose at the call site for the common
case.

### Option C1: Clojure-style threading (family of macros)

Status becomes a value type. Updates are pure functions chained through one of several
threading operators, typically `->` for the common case, plus `cond->` for conditional
updates (the partial-update sweet spot), `some->` for nil-aware chains, etc. Modeled on
Clojure's threading macros.

```go
// WASM — full update
status.get("Pressure Alert")
  -> status.message("High Pressure")
  -> status.variant(.error)
  -> status.save()

// WASM — partial update via cond-> (only updates fields whose predicates are truthy)
status.get("Pressure Alert")
  cond-> message  status.message(message)
  cond-> variant  status.variant(variant)
  -> status.save()

// Flow — Flow's `->` already pipes; threading is the natural shape
trigger -> status.get{"Pressure Alert"}
        -> status.message{"High Pressure"}
        -> status.variant{.error}
        -> status.save{}

// Flow — partial update via cond->
trigger -> status.get{"Pressure Alert"}
        cond-> message  status.message{message}
        cond-> variant  status.variant{variant}
        -> status.save{}
```

Each step is pure. `cond->` makes partial-update natural: the chain literally skips the
steps whose predicates are nil. The cost is a family of operators (`->`, `cond->`,
`some->`, `as->`, `->>`) that operators must learn and choose between.

### Option C2: Elixir-style pipe (single operator)

Same first-class value model as C1, but with a **single** pipe operator (`|>`). Partial
updates are expressed by simply omitting the step from the chain. There is no
conditional pipe. Conditional logic is handled with ordinary `if` blocks around chains
or by writing wrapper functions.

```go
// WASM — full update
status.get("Pressure Alert")
  |> status.put_message("High Pressure")
  |> status.put_variant(.error)
  |> status.save()

// WASM — partial update requires manual branching
my_status := status.get("Pressure Alert")
if message != nil {
    my_status = status.put_message(my_status, message)
}
if variant != nil {
    my_status = status.put_variant(my_status, variant)
}
status.save(my_status)

// Flow — same shape as C1 (Flow's `->` already pipes)
trigger -> status.get{"Pressure Alert"}
        -> status.message{"High Pressure"}
        -> status.variant{.error}
        -> status.save{}

// Flow — partial update; no cond->, branch outside the pipeline
if message != nil {
    trigger -> status.get{"Pressure Alert"}
            -> status.message{message}
            -> status.save{}
}
```

One operator to learn, one syntactic transformation. The partial-update case loses the
elegance C1 gets from `cond->` and reads as ordinary imperative branching wrapped around
the pipeline.

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

// Flow — methods don't chain on Flow edges; each "method" becomes its own stage
trigger -> status.find{"Pressure Alert"}
        -> status.message{"High Pressure"}
        -> status.variant{.error}
        -> status.save{}
```

The dot is already familiar, but Arc has no method dispatch today; adding it means
deciding receiver semantics, whether methods are first-class, whether subtyping or
inheritance enters the language, and how this interacts with Flow nodes. A bigger
language commitment than it sounds; "just methods" is rarely just methods.

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

// Partial — just include fewer assignments in the block
status.find("Pressure Alert").apply {
    variant = .error
}.save()

// Flow — apply blocks need a held receiver; collapse to a stage with field assignments
trigger -> status.find{"Pressure Alert"}
        -> status.apply{message="High Pressure", variant=.error}
        -> status.save{}
```

Reads like imperative mutation while still producing a value at the end. The new
language feature is closures with implicit receiver, a non-trivial addition to the type
system, since the body has to type-check against the receiver's fields.

### Option F: Cascade operator (Smalltalk, Dart)

A new operator (`..` in Dart) sends each call to the same receiver, instead of relying
on each method returning `self`. Reads like a block of mutations on one object.

```go
// WASM
status.find("Pressure Alert")
  ..message = "High Pressure"
  ..variant = .error
  ..save()

// Flow — cascades require a held receiver; collapse to a single stage with multi-field config
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

### Option G: Record-update syntax (Haskell, F#, OCaml, Elixir)

Status is a record value, and a literal-update form produces a new record with a subset
of fields replaced. No pipe, no methods, just one expression that says "this status, but
with these fields changed."

```go
// WASM
my_status := status.get("Pressure Alert")
status.save({my_status | message: "High Pressure", variant: .error})

// Partial update — same syntax, fewer fields
status.save({my_status | variant: .error})

// Touch — no field updates at all
status.save(my_status)

// Flow — record values flow through the edge; an update stage replaces fields
trigger -> status.get{"Pressure Alert"}
        -> status.with{message="High Pressure", variant=.error}
        -> status.save{}

// Flow — partial update via fewer fields in the update stage
trigger -> status.get{"Pressure Alert"}
        -> status.with{variant=.error}
        -> status.save{}
```

This pattern most directly fits the "pass around objects" gesture: the value is named,
fields are named, the update is one expression, and preserve-on-omit is a property of
the language rather than of one symbol. The cost is first-class records plus a new
literal form; once those exist, it generalizes everywhere (ranges, devices, channels,
flow node configs).

### Option H: Lenses / optics (Haskell, Scala Monocle)

First-class field accessors that compose. A lens is a value that knows how to focus on
one field of a structure; setting via a lens produces a new structure with that field
changed. Lenses compose, so deeply-nested updates become one expression.

```go
// Hypothetical Arc — lenses applied via & and .~
my_status := status.get("Pressure Alert")
status.save(my_status & status.message .~ "High Pressure"
                      & status.variant .~ .error)

// Flow — lens application as an updater stage carrying lens values on the edge
trigger -> status.get{"Pressure Alert"}
        -> status.update{lenses=[status.message .~ "High Pressure",
                                  status.variant .~ .error]}
        -> status.save{}
```

The academically clean answer to "compose updates on nested data." Pays off enormously
for code that traverses deep object graphs (game state, ASTs, JSON). Almost certainly
overkill for status updates and a poor fit for control engineers. Mentioned only for
completeness.

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

Arc's users are control engineers writing imperative sequences ("when this happens, set
that status"). Option A matches their mental model: one call, named arguments, behavior
controlled by what's passed. Every alternative asks them to reason about something else:
value transformation chains (C1, C2), method receivers (D, E, F), record literals (G),
function-valued options (B), or composable optics (H). That's friction without benefit
for the dominant use case.

Every alternative is also substantially more expensive to build: weeks of language-wide
work versus a few days for Option A. The cost falls into three families of
infrastructure Arc doesn't currently have:

- **First-class records** (C1, C2, G, H): a new kind in the type system, literal syntax,
  WASM struct layout, and Flow edges that carry typed values. H additionally requires
  lens primitives, raising the bar further for an audience that doesn't write functional
  code.
- **Method dispatch / implicit receiver** (D, E, F): receiver semantics, dispatch rules,
  and (for E) closures that type-check against an enclosing receiver.
- **First-class functions** (B): function values and a higher-order calling convention.

Each family is generic infrastructure that should land for its own reasons in a separate
RFC, not as a prerequisite for status updates.

**C1 vs C2.** If a pipe option ever does land, C2 (Elixir-style single operator) is the
cheaper and simpler entry point: one operator, one mental model, one set of compiler
rules. C1 (Clojure-style family) buys better partial-update ergonomics via `cond->`, but
at the cost of multiple operators that users must learn and choose between.
Partial-update ergonomics is exactly the case where Option A is already strongest, so
C1's main advantage over C2 is the case Option A already wins. That makes C2 the more
defensible "if not A, then this" choice, but neither is competitive with A on the time
horizon this RFC is operating on.

## Recommendation

Stay with Option A (polymorphism) as currently described in RFC 0037.

Concerns about a single axis of polymorphism (touch dispatch, name/key dispatch, the
create-vs-update implicit in `set`-as-upsert) are worth discussing individually; several
of those have smaller fixes than rebuilding the type system. A blanket objection to
"polymorphism" is too broad to action, and the alternatives on offer (first-class
records plus either a family of threading macros or a single pipe operator) are
disproportionate to the problem the RFC is trying to solve.
