# Go Development

## Go Modules

Individual Go modules in the monorepo:

- `/core/` - Synnax server (4-layer architecture)
- `/aspen/` - Distributed key-value store and cluster management
- `/cesium/` - Time-series database engine
- `/freighter/go/` - Transport layer (gRPC, HTTP, WebSocket)
- `/freighter/integration/` - Freighter integration test server (used to validate
  Freighter implementations in other languages)
- `/arc/go/` - Arc programming language compiler
- `/alamos/go/` - Instrumentation and observability
- `/oracle/` - Database migration CLI and tooling for Synnax schema evolution
- `/x/go/` - Shared Go utilities

### Local Module Resolution

Each Go module uses `replace` directives in its `go.mod` to reference sibling modules
via relative paths. For example, `core/go.mod` contains:

```go
replace (
    github.com/synnaxlabs/alamos => ../alamos/go
    github.com/synnaxlabs/arc => ../arc/go
    github.com/synnaxlabs/aspen => ../aspen
    github.com/synnaxlabs/cesium => ../cesium
    github.com/synnaxlabs/freighter => ../freighter/go
    github.com/synnaxlabs/x => ../x/go
)
```

This allows modules to reference each other without publishing. `go.work` and
`go.work.sum` are gitignored but can be used locally for convenience.

## Development Commands

### Building

```bash
cd core && go build ./...
cd cesium && go build ./...
```

### Testing

**IMPORTANT**: Always use `ginkgo` to run tests, not `go test`. The codebase uses
Ginkgo/Gomega for testing.

```bash
# Run all tests in a package
cd core/pkg/distribution/channel && ginkgo

# Run tests with verbose output
cd cesium && ginkgo -v

# Run specific tests by focus
cd core/pkg/distribution/channel && ginkgo --focus "Name Validation"

# Run all tests recursively
cd core && ginkgo -r
```

### Integration Tests

- Integration tests located in `/integration/` directory
- Python-based conductor framework
- Run with `pytest` from integration directory

## Code Style

- **Formatter**: `gofmt` (default Go formatting)
- **Conventions**: Standard Go idioms, interfaces for abstraction
- **Line length**: 88 characters (configured in editor, not enforced by gofmt)
- **Imports**: Group standard library, external, and internal packages
- **Error handling**: Explicit error returns, wrapped with context

### Go Patterns

- **Interfaces for abstraction**: Define small, focused interfaces
- **Dependency injection**: Pass dependencies as parameters, not globals
- **Context propagation**: Use `context.Context` for cancellation and values
- **Structured concurrency**: Use goroutines with proper synchronization
- **Table-driven tests**: Parameterize tests with test cases

## General Rules

### Rule 1: Never ignore errors

If a function returns an `error`, that error **must** be handled. Ignoring it — via
`_ = fn()`, by omitting the error in a multi-value assignment when the language allows
it, or by simply not checking the value — is not allowed. There are no exceptions for
"can't fail in practice", "best-effort cleanup", or "we don't care about this path".

The acceptable ways to handle an error are:

- **Return it** to the caller, usually wrapped with `errors.Wrap` for context.
- **Log it** via the configured instrumentation when the function's contract does not
  allow returning an error (e.g., a deferred cleanup, a background goroutine).
- **Assert on it in a test** via `MustSucceed`, `Expect(...).Error().To(...)`, or
  similar (see the testing rules above).
- **Panic** only when the error indicates a programmer bug that cannot be recovered
  from.

**Incorrect — never do this:**

```go
_ = file.Close()                 // ❌ ignored error
svc.Flush(ctx)                   // ❌ error dropped silently
v, _ := parseConfig(raw)         // ❌ discarded with blank identifier
defer conn.Close()               // ❌ Close returns an error that is silently dropped
```

**Correct:**

```go
if err := file.Close(); err != nil {
    return errors.Wrap(err, "close config file")
}

if err := svc.Flush(ctx); err != nil {
    return errors.Wrap(err, "flush pending writes")
}

v, err := parseConfig(raw)
if err != nil {
    return errors.Wrap(err, "parse config")
}

// Deferred cleanup that cannot return an error: log it.
defer func() {
    if err := conn.Close(); err != nil {
        s.L.Warn("close conn", zap.Error(err))
    }
}()
```

### Rule 2: Always use `github.com/synnaxlabs/x/errors` — never `fmt.Errorf`

`github.com/synnaxlabs/x/errors` is the **only** error package permitted in this
codebase. Every error construction or wrapping call must go through it. `fmt.Errorf` is
forbidden in all cases, including `%w` wrapping — there are no exceptions. The standard
library's `errors` package and any third-party error package (`pkg/errors`,
`cockroachdb/errors` directly, etc.) are likewise not permitted; the
`synnaxlabs/x/errors` package already wraps `cockroachdb/errors` and adds stack capture,
so go through it.

Use the following functions:

- `errors.New(msg)` — construct a new error with a static message.
- `errors.Newf(format, args...)` — construct a new error with a formatted message.
- `errors.Wrap(err, msg)` — wrap an existing error with a static context message.
- `errors.Wrapf(err, format, args...)` — wrap an existing error with a formatted context
  message.

**Correct:**

```go
import "github.com/synnaxlabs/x/errors"

if name == "" {
    return errors.New("channel name is required")
}

if !validName.MatchString(name) {
    return errors.Newf("invalid channel name %q", name)
}

if err := s.db.Put(ctx, ch); err != nil {
    return errors.Wrapf(err, "write channel %s", ch.Name)
}
```

**Incorrect — never do this:**

```go
import "fmt"
import stderrors "errors"

return fmt.Errorf("invalid channel name %q", name)                 // ❌ fmt.Errorf
return fmt.Errorf("write channel %s: %w", ch.Name, err)            // ❌ fmt.Errorf with %w
return stderrors.New("channel name is required")                   // ❌ std errors
return pkgerrors.Wrap(err, "write channel")                        // ❌ third-party errors
```

### Rule 3: Use sentinel errors when an error will be matched on

Whenever there is a reasonable expectation that callers (including tests) will need to
**match on** a specific error condition, that condition should be represented by an
exported sentinel error value — not a one-off `errors.New("...")` inside the function
body. Matching on a string substring is brittle; matching on an identity is not.

**Reuse existing sentinels first.** Before defining a new one, check whether an
appropriate sentinel already exists in a lower-level package and wrap it. Common shared
sentinels include:

- `query.ErrNotFound` — missing resource lookups.
- `validate.Error` — input validation failures.
- `freighter.EOF` — stream termination.

Wrapping a shared sentinel keeps `errors.Is` / `MatchError` semantics intact while
adding domain-specific context to the message:

```go
// cesium/internal/channel/errors.go
var ErrNotFound = errors.Wrap(query.ErrNotFound, "channel not found")

// aspen/internal/cluster/cluster.go
var ErrNodeNotFound = errors.Wrap(query.ErrNotFound, "node not found")
```

A caller can match either the specific flavor
(`errors.Is(err, cluster.ErrNodeNotFound)`) or the general shape
(`errors.Is(err, query.ErrNotFound)`).

**Create a new sentinel only when clearly justified.** A new sentinel is warranted when
there is no existing sentinel that captures the same condition and the error is part of
the package's public contract (documented return, caller is expected to branch on it).
Do **not** introduce a sentinel speculatively for an error that no one matches on today.

Conventions for sentinels:

- **Exported, package-level `var`**, named `Err<Condition>` (e.g., `ErrNotFound`,
  `ErrDiscontinuous`, `ErrLeaseNotTransferable`).
- **Constructed with `errors.New` or `errors.Wrap`** from
  `github.com/synnaxlabs/x/errors` (see Rule 2).
- **Message starts with lowercase** and does not end with a period (standard Go
  convention).
- **Documented** with a short doc comment describing when the error is returned (see
  Comments Rule 2).

**Correct — new sentinel for a package-specific condition:**

```go
// ErrVirtual is returned when a unary database is opened on a virtual channel,
// which does not have persistent storage.
var ErrVirtual = errors.New("cannot open a unary database on a virtual channel")
```

**Incorrect — one-off error that callers are expected to branch on:**

```go
// Inside a handler — callers have to string-match "already exists" to detect
// duplicates. Define an ErrAlreadyExists sentinel instead.
if exists {
    return errors.New("channel already exists")
}
```

**Incorrect — duplicating an existing sentinel from another package:**

```go
// This re-invents query.ErrNotFound. Wrap it instead.
var ErrChannelMissing = errors.New("channel not found")
```

### Rule 4: Never write throwaway debug scripts — write a real test

Do **not** create ad-hoc Go files (e.g. `temp_debug.go`, `scratch.go`, `main_test.go` in
an unrelated package, a `main` package under `/tmp`) to explore behavior, reproduce a
bug, or "just run something quickly". These files are brittle, bypass the module's build
graph, don't get committed or reviewed, and tend to be forgotten until they break CI or
leak secrets.

Instead:

- **Write a proper Ginkgo test** in the package being investigated. A focused
  `It("reproduces ...", ...)` is just as fast to write as a `main` function, runs under
  the same build flags as production code, and lives where the next person debugging the
  same area will find it.
- **Run it with `ginkgo --focus`** to iterate (see the Testing section).
- **Add `fmt.Println` / `fmt.Printf`** inside the test or the code under test for
  temporary visibility, and remove them before committing. For anything that should
  survive the commit, use the package's configured instrumentation (`alamos` / `zap`) at
  the appropriate level, not `fmt`.

**Incorrect — never do this:**

```go
// temp_debug.go
package main

import "fmt"

func main() {
    svc := channel.NewService(...)
    ch, err := svc.Create(ctx, "foo")
    fmt.Println(ch, err) // ❌ throwaway script
}
```

**Correct:**

```go
// channel/service_test.go (in an existing *_test.go file)
It("repros the duplicate-name bug", func() {
    _ = MustSucceed(svc.Create(ctx, "foo"))
    Expect(svc.Create(ctx, "foo")).Error().To(MatchError(channel.ErrAlreadyExists))
})
```

Run: `ginkgo --focus "repros the duplicate-name bug"` in the package directory.

### Rule 5: Never edit generated code

Generated files are identified by a `Code generated ... DO NOT EDIT.` header (Go's
standard convention) and, for Oracle output, by the `.gen.` suffix in the filename
(e.g., `types.gen.ts`, `types_gen.py`, `*_gen.go`). Do **not** edit these files. Any
edit will be silently overwritten on the next `oracle sync`, and the real source of
truth will still be wrong.

**When the generated file needs to change, follow this flow:**

1. **Identify the generator.** Oracle-generated files come from `.oracle` schema
   definitions in `/schemas/` at the repo root (e.g., `/schemas/status.oracle`,
   `/schemas/channel.oracle`). Find the schema whose output maps to the file you would
   have edited.
2. **Edit the schema.** Make the change in the `.oracle` source file, not the generated
   output.
3. **Prompt the user to run `oracle sync`.** Do not run it yourself — per project
   convention, the user runs oracle (see user memory). Ask them to run the sync and
   report the result.

**If the change cannot be expressed in the current schema language,** the generator
itself in `/oracle/` may need to change. In that case:

1. **Investigate first.** Read the relevant parts of `/oracle/` (`analyzer/`,
   `formatter/`, `resolution/`, `exec/`) to confirm what the generator currently
   supports and where the gap is.
2. **Prompt the user before making generator changes.** Describe what's missing, what
   you propose to change in the generator, and the blast radius (what other generated
   files would shift). Wait for approval — generator changes often touch every
   language's output and are easy to regress.

**Incorrect — never do this:**

```go
// Code generated by Oracle. DO NOT EDIT.
type Channel struct {
    Name    string `json:"name"`
    NewField string `json:"new_field"` // ❌ added by hand to generated file
}
```

**Correct:**

Edit `/schemas/channel.oracle` to add `new_field`, then ask the user to run
`oracle sync` and commit the regenerated `.gen` files alongside the schema change.

### Rule 6: Always prefer `any` over `interface{}`

`any` is a built-in alias for `interface{}` (added in Go 1.18) and is the canonical
spelling throughout the codebase. Always write `any`. Do not write `interface{}` in new
code, and prefer fixing it to `any` when editing existing code.

This applies everywhere: function parameters, return types, map/slice element types,
type parameters' constraints, struct fields, and type assertions.

**Correct:**

```go
func Log(ctx context.Context, msg string, args ...any) { /* ... */ }

type Registry map[string]any

func As[T any](v any) (T, bool) { /* ... */ }
```

**Incorrect — never do this:**

```go
func Log(ctx context.Context, msg string, args ...interface{}) { /* ... */ }       // ❌
type Registry map[string]interface{}                                                // ❌
func As[T interface{}](v interface{}) (T, bool) { /* ... */ }                       // ❌
```

### Rule 7: Use `any` with extreme caution — justify every occurrence

Rule 6 is about spelling: when you _must_ use the empty interface, it's `any`, not
`interface{}`. This rule is about whether to reach for it at all. In Synnax, the default
answer is **no**. `any` erases the type system, forcing runtime type assertions,
defeating the compiler, and pushing errors from compile time to production. Every `any`
in new code must have a clear, written justification.

**Prefer, in order:**

1. **A concrete type.** If you know the type, name it.
2. **A focused interface.** `Reader`, `Writer`, `Closer`, a domain interface with two or
   three methods — describes what the value _does_, not that it exists.
3. **A generic type parameter.** `func F[T Encodable](v T)` preserves type identity at
   the call site and still lets the function be reused.
4. **A sum-like interface with a sealed set of implementers**, used together with a type
   switch in one well-documented place.
5. **`any` — only when none of the above are possible.**

**Legitimate uses of `any`** (these are the exceptions, not the pattern):

- Variadic formatting / logging arguments (`fmt.Sprintf`-shaped APIs).
- Reflective serialization boundaries (`json.Marshal`, generic encoders).
- Truly heterogeneous containers at a framework edge (e.g., a DI registry) where the
  type diversity is the feature.

When you do use `any`, explain to the user in conversation why a typed alternative
doesn't work — why a concrete type, focused interface, generic parameter, or sealed sum
won't fit. Do **not** bury the justification in an inline comment; inline comments about
"why `any`" are noise (see Comments Rule 1). The justification is a conversation the
user can push back on, not a tag in the file.

**Incorrect — `any` used because typing would take a moment of thought:**

```go
func Handle(event any) { /* ❌ what is event? */ }

type Cache struct {
    items map[string]any // ❌ caller must assert on every read
}

func Retry(fn func() (any, error)) (any, error) { /* ❌ use generics */ }
```

**Correct — typed alternatives:**

```go
func Handle(event channel.Event) { /* ... */ }                     // concrete

type Cache[V any] struct {                                          // generic
    items map[string]V
}

func Retry[T any](fn func() (T, error)) (T, error) { /* ... */ }    // generic
```

**Correct — `any` in a legitimate place:**

```go
func (l *Logger) Infof(format string, args ...any) { /* ... */ }
```

(Justification lives in the conversation with the user, not in the file.)

### Rule 8: Use `set.Set` for set membership — never `map[T]struct{}` or `map[T]bool`

For set semantics (membership, insertion, deletion, union, intersection), always use
`set.Set[T]` from `github.com/synnaxlabs/x/set`. Do not hand-roll a `map[T]struct{}` or
`map[T]bool` — these obscure intent, force every caller to re-implement the same
membership helpers, and produce inconsistent behavior across packages (e.g., does
`false` count as absent?).

`set.Set[T]` is a `map[T]struct{}` under the hood, so the zero-allocation and
constant-time-lookup properties are identical.

**Correct:**

```go
import "github.com/synnaxlabs/x/set"

seen := set.New[channel.Key]()
seen.Add(k)
if seen.Contains(k) { /* ... */ }

allowed := set.New("read", "write", "admin")
```

**Incorrect — never do this:**

```go
seen := map[channel.Key]struct{}{}                 // ❌ use set.Set
seen[k] = struct{}{}
if _, ok := seen[k]; ok { /* ... */ }

allowed := map[string]bool{"read": true, "write": true} // ❌ same — use set.Set
if allowed[role] { /* ... */ }                          //    and this is wrong on missing keys
```

**When a map-of-bool is legitimate:** if the value meaningfully carries tri-state or
toggle information (e.g., "explicitly enabled" vs "explicitly disabled" vs "not
configured"), it's a map, not a set. Set-like shapes only.

## Comments

### Rule 1: No excessive comments

Comments inside function or test bodies are **discouraged**. Well-named variables,
functions, and types should make the code self-explanatory. Only add an inline comment
when it genuinely clarifies **obscure or surprising behavior** that the code itself
cannot convey — a subtle invariant, a non-obvious reason for an ordering constraint, a
workaround for an upstream bug, or a warning about a trap a reader would otherwise fall
into.

Do **not** write comments that:

- Restate what the next line of code does.
- Label sections of a function (`// setup`, `// act`, `// assert`).
- Narrate each step of a procedure.
- Describe the current task, ticket, or author.

**Incorrect:**

```go
func (s *Service) Create(ctx context.Context, name string) (Channel, error) {
    // validate the name
    if name == "" {
        return Channel{}, errors.New("name required")
    }
    // build the channel
    ch := Channel{Name: name}
    // write it to the database
    return ch, s.db.Put(ctx, ch)
}
```

**Correct:**

```go
func (s *Service) Create(ctx context.Context, name string) (Channel, error) {
    if name == "" {
        return Channel{}, errors.New("name required")
    }
    ch := Channel{Name: name}
    return ch, s.db.Put(ctx, ch)
}
```

**Correct — a comment that earns its place:**

```go
// Must hold s.mu before calling; gossip's ordering guarantee depends on the caller
// serializing writes against incoming SIR updates.
func (s *Service) applyUpdate(u Update) { /* ... */ }
```

Doc comments on identifier declarations (types, functions, constants, package-level
vars) are a separate matter and follow standard Go conventions — this rule is about
_in-body_ comments. See Rule 2.

### Rule 2: Document identifiers from the caller's perspective

Comments explaining functions, methods, constants, variables, or types are
**encouraged** for both exported and unexported identifiers. A good doc comment tells
the reader what they need to know to use the identifier correctly: what it does, what
the arguments mean, what is returned, what errors can occur, and any preconditions or
side effects.

**Doc comments describe behavior, not implementation.** Do not narrate which
collaborators are called, what underlying systems back the call, what data structures
are used internally, the order of internal steps, or any other detail the reader could
not observe by using the API. These details belong in the code itself, where they can
change without leaving stale comments behind. The only implementation facts that earn a
place in a doc comment are ones the caller's correctness depends on — concurrency
safety, complexity, blocking behavior, ordering guarantees, lock discipline the caller
must follow, or similar contracts. If you find yourself writing "calls X", "uses Y",
"stores in Z", "then does W", stop: that information is not for the doc comment.

**Narrow exception for extreme cases.** A brief note about an implementation choice is
acceptable when the choice is genuinely **unintuitive or unconventional** and a reader
would otherwise be confused, assume a bug, or "fix" it in a follow-up — a workaround for
a known upstream issue, a deliberately non-standard algorithm chosen for a specific
reason, an ordering that contradicts what a careful reader would expect. The bar is
high: this is for choices the next reader will stop and question, not for ordinary
judgment calls or for any implementation detail you happen to find interesting. Keep the
note to a sentence or two and explain the _why_, not the mechanics. When in doubt, leave
it out — the default is silence.

This applies the same way to **unexported** identifiers. The "caller" of an unexported
function or type is the rest of the package, and the doc comment exists for the next
person who will read or use it — not as a transcript of the body. Internal narration is
no more useful on unexported code than on exported code.

Keep doc comments **short, tight, and focused**. One or two sentences is usually enough.
Prefer adding a second paragraph or a bulleted list only when the API has real nuance
the reader must understand.

Follow standard Go doc conventions:

- **Start with the identifier's name.** `// Create ...`, `// Service ...`,
  `// ErrNotFound ...`.
- **Write complete sentences** with a period at the end.
- **Use a single `//` block** directly above the declaration, no blank line between the
  comment and the declaration.
- **Use `//` for all doc comments**, not `/* ... */`.
- **Do not indent code samples inside doc comments** except by a single tab (gofmt
  handles this); `go doc` renders indented blocks as code.
- **Reference other identifiers by unqualified name** when they are in the same package
  (`See Reader.Read.`), or with the package prefix otherwise.

**Correct:**

```go
// Create persists a new channel with the given name and returns its populated
// record. It returns validate.Error if name is empty or already taken.
func (s *Service) Create(ctx context.Context, name string) (Channel, error) { /* ... */ }

// Reader streams frames from a range of channels. Reader is not safe for
// concurrent use; open one per goroutine.
type Reader struct { /* ... */ }

// ErrNotFound is returned when a lookup targets a channel that does not exist in
// the cluster.
var ErrNotFound = errors.New("channel not found")
```

**Incorrect — restates the signature without adding value:**

```go
// Create creates a channel. It takes a context and a name and returns a channel
// and an error.
func (s *Service) Create(ctx context.Context, name string) (Channel, error) { /* ... */ }
```

**Incorrect — leaks irrelevant implementation details:**

```go
// Create builds a Channel struct, calls s.validator.Validate, then invokes
// s.db.Put which writes through the Pebble WAL before returning.
func (s *Service) Create(ctx context.Context, name string) (Channel, error) { /* ... */ }
```

**Incorrect — narrates internal sequencing the caller cannot observe:**

```go
// Apply acquires s.mu, appends the update to the journal, signals the gossip
// subsystem, and then releases the lock before returning.
func (s *Service) Apply(u Update) error { /* ... */ }
```

The caller doesn't see the journal, the lock, or the gossip signal. They see a function
that applies an update. If the locking is something the caller must coordinate with, say
_that_ — not the internal sequence.

**Incorrect — same standard applied to an unexported helper:**

```go
// resolveKey looks up the channel in s.cache, falls back to s.db.Get on miss,
// and stores the result back in the cache before returning.
func (s *Service) resolveKey(ctx context.Context, name string) (Key, error) { /* ... */ }
```

The caller of `resolveKey` (somewhere else in the package) needs to know it returns the
key for a channel name and that it returns `ErrNotFound` if the name is unknown. The
cache, the fallback, and the write-back are implementation choices that may change
tomorrow.

**Incorrect — doesn't start with the identifier name, uses the wrong comment style:**

```go
/* Creates a new channel. */
func (s *Service) Create(ctx context.Context, name string) (Channel, error) { /* ... */ }
```

### Rule 3: Document struct fields, including unexported ones

Comments on struct fields and member variables are **strongly encouraged**, even when
the field is unexported. Struct layouts are where a reader builds their mental model of
a type, and an unexplained field is almost always a source of confusion later.

Keep field comments **tight, short, and focused** — usually a single line. Follow the
same Go doc conventions as Rule 2: `//` style, start with the field's name, complete
sentence ending in a period, placed directly above the field with no blank line between.

The same implementation-detail rule from Rule 2 applies here. A field comment should
describe **what the field represents** — its semantic role, units, valid range, or
invariants the rest of the package must uphold. Do **not** narrate when or how the field
is mutated, which methods touch it, or what the methods do. Method bodies are the source
of truth for that; a comment that restates them only goes stale.

The exceptions are the same kinds of contracts Rule 2 calls out: lock discipline ("must
hold `mu` to read or write"), invariants other fields depend on, or units and ranges
that the type's name does not already convey. The same narrow extreme-case exception
applies — a brief note on a genuinely unintuitive choice (e.g., a field intentionally
typed wider than it needs to be to work around an upstream library bug) is acceptable,
but the bar is high.

**Correct:**

```go
type Writer struct {
    // Channels is the set of channels this writer will emit frames for. Must be
    // non-empty and must not contain duplicates.
    Channels []ChannelKey
    // Start is the timestamp of the first sample. Subsequent writes must be
    // strictly monotonic relative to Start.
    Start telem.TimeStamp

    // db is the underlying storage handle.
    db *pebble.DB
    // mu guards pending and closed.
    mu sync.Mutex
    // pending holds frames buffered for the next flush.
    pending []Frame
    // closed reports whether the Writer has been closed.
    closed bool
}
```

**Incorrect — narrates when fields are mutated and what methods do:**

```go
type Writer struct {
    // db is the underlying storage handle. Opened in NewWriter, closed in Close.
    db *pebble.DB
    // pending buffers frames that have not yet been flushed; appended to in
    // Write, drained in flush, cleared after a successful Put.
    pending []Frame
    // closed is set to true on the first call to Close; subsequent Close calls
    // see it as true and return nil immediately without touching db.
    closed bool
}
```

The names already say what these fields are. The behavior of `Close`, `Write`, and
`flush` lives in those methods, not in the struct definition.

**Incorrect — trailing inline comments instead of `//` blocks above the field:**

```go
type Date struct {
    Year  uint16 // calendar year.
    Month uint8  // month of year [1, 12].
    Day   uint8  // day of month [1, 31].
}
```

**Incorrect — no field comments at all on non-trivial state:**

```go
type Writer struct {
    Channels []ChannelKey
    Start    telem.TimeStamp
    db       *pebble.DB
    mu       sync.Mutex
    pending  []Frame
    closed   bool
}
```

**Incorrect — field comment restates the type without adding meaning:**

```go
type Writer struct {
    // Channels is a slice of ChannelKey.
    Channels []ChannelKey
}
```

### Rule 4: Do not modify existing comments unless you are confident they should change

Existing comments and annotations in the codebase represent decisions made by someone
who had context you may not. Treat them as **load-bearing until proven otherwise**. Do
not rewrite, "improve", reformat, or delete a comment as a side effect of editing the
code around it. Drive-by comment churn is one of the easiest ways to silently destroy
context — a one-line note explaining a non-obvious invariant or workaround is gone the
moment someone "tightens the wording".

You may modify or remove a comment only when you have a concrete reason and you are
confident the change is correct. Acceptable reasons:

- The code the comment describes has changed and the comment is now factually wrong.
- The comment violates one of the rules above (Rules 1–3) and you have read it carefully
  enough to be sure the information it conveys is either redundant or belongs elsewhere
  — not just that it _looks_ like the kind of comment a rule would flag.
- The user has explicitly asked you to revise the comment.

If you are not sure whether a comment is still accurate or still earning its place,
**leave it alone**. The cost of an out-of-date comment surviving one more PR is trivial;
the cost of deleting a workaround note that turns out to have been the only record of a
subtle bug is not.

**Incorrect — drive-by rewrite while editing nearby code:**

```go
// Before (existing in the file):
// Must hold s.mu before calling; gossip's ordering guarantee depends on the caller
// serializing writes against incoming SIR updates.
func (s *Service) applyUpdate(u Update) { /* ... */ }

// After your unrelated edit:
// applyUpdate applies an update to the service.   ❌ deleted real context
func (s *Service) applyUpdate(u Update) { /* ... */ }
```

**Incorrect — deleting a comment because it "looks like" implementation narration:**

```go
// Before:
// We poll every 50ms instead of using a ticker because the underlying clock
// drifts on NI Linux Real-Time and a ticker accumulates the drift.
for { /* ... */ }

// After:                                          ❌ that wasn't narration; it was the why
for { /* ... */ }
```

**Correct — leave it; if you suspect it's wrong, ask or verify before changing.**

## Testing with Ginkgo/Gomega

### Structure

- Suite files: `*_suite_test.go`
- Test files: `*_test.go`
- Package naming: `package_name_test` for blackbox testing

### Example

```go
package cesium_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "testing"
)

func TestCesium(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Cesium Suite")
}

var _ = Describe("Feature Behavior", func() {
    Context("FS: "+fsName, Ordered, func() {
        BeforeAll(func() {
            // Setup
        })

        AfterAll(func() {
            // Teardown
        })

        Describe("Sub-feature", func() {
            It("Should do something", func() {
                Expect(result).To(Equal(expected))
            })

            Specify("Specific behavior", func() {
                Expect(value).To(BeTrue())
            })
        })
    })
})
```

### Key Features

- **BDD Structure**: `Describe`, `Context`, `It`, `Specify` for organizing tests
- **Lifecycle Hooks**: `BeforeAll`, `AfterAll`, `BeforeEach`, `AfterEach`,
  `JustBeforeEach`
- **Matchers**: Rich Gomega matchers (`Equal`, `BeTrue`, `Succeed`, `HaveLen`,
  `MatchError`)
- **Custom Matchers**: Domain-specific matchers (e.g., `telem.MatchSeriesDataV`)
- **Ordering**: `Ordered` decorator for sequential test execution
- **Async Support**: `Eventually` matcher for polling assertions

### Common Patterns

- Parameterized tests using loops over file systems or configurations
- Table-driven tests via loops with test cases
- Context-based test organization for different scenarios
- Helper functions for database/service setup in suite files
- Goroutine leak detection: `ShouldNotLeakGoroutines()`

### Helpful Utilities

- `MustSucceed(result, err)` - Unwrap single-value result or fail
- `MustSucceed2(a, b, err)` - Unwrap two-value result or fail
- `MustBeOk(value, ok)` - Unwrap `(T, bool)` result or fail if `ok` is false
- `MustOpen(v, err)` - Unwrap an opener returning `(io.Closer, error)` and schedule
  `Close` via `DeferCleanup`
- `DeferClose(v)` - Schedule `Close` on an already-opened `io.Closer` via `DeferCleanup`
- `Eventually(func() bool).Should(BeTrue())` - Poll until condition met
- `Consistently(func() bool).Should(BeTrue())` - Assert condition stays true

### Strict Rules

#### Rule 1: Always use `MustSucceed` / `MustSucceed2` to unwrap nominal returns

When a function returns `(T, error)` (or `(A, B, error)`) and the test expects the error
to be nil so it can use the nominal return values, you **must** use `MustSucceed` /
`MustSucceed2` from `github.com/synnaxlabs/x/testutil`. Do **not** assign the error to a
variable and then assert it separately with `Expect(err).ToNot(HaveOccurred())` (or
`Expect(err).To(Succeed())`, `Expect(err).To(BeNil())`, etc.). This is a hard rule —
there are no exceptions within test code.

**Correct:**

```go
import . "github.com/synnaxlabs/x/testutil"

ch := MustSucceed(svc.Create(ctx, "my-channel"))
frame, release := MustSucceed2(reader.Read(ctx))
```

**Incorrect — never do this:**

```go
ch, err := svc.Create(ctx, "my-channel")
Expect(err).ToNot(HaveOccurred()) // ❌ use MustSucceed instead
// ... use ch

frame, release, err := reader.Read(ctx)
Expect(err).To(Succeed()) // ❌ use MustSucceed2 instead
```

#### Rule 2: Always use `Expect(fn()).Error().To(MatchError(...))` to assert errors

When a function returns `(T, error)` (or `(A, B, error)`) and the test is asserting that
the error is non-nil (or matches a specific error), you **must** inline the call into
`Expect(...).Error().To(...)`. Do **not** capture the error into a variable and then
assert on it separately. This is also a hard rule — no exceptions.

This pattern is strictly better because Gomega's `.Error()` transformer asserts that
**every other return value is the zero value for its type**, in addition to matching the
error. That catches bugs where a failing call still produces a non-zero partial result.

**Correct:**

```go
Expect(reader.Read(ctx)).Error().To(MatchError(myError))
Expect(parser.Parse(input)).Error().To(MatchError(ContainSubstring("invalid")))
```

**Incorrect — never do this:**

```go
_, err := reader.Read(ctx)
Expect(err).To(MatchError(myError)) // ❌ does not assert zero values on other returns

frame, release, err := reader.Read(ctx)
Expect(err).To(MatchError(myError)) // ❌ same problem, and you shouldn't bind frame/release at all
```

#### Rule 3: Always use `MustBeOk` to unwrap `(T, bool)` returns

When a function returns `(T, bool)` (the "comma-ok" idiom, e.g. map lookups, cache hits,
type assertions, `Find` helpers) and the test expects `ok` to be true so it can use the
value, you **must** use `MustBeOk` from `github.com/synnaxlabs/x/testutil`. Do **not**
bind the `ok` to a variable and assert it separately with `Expect(ok).To(BeTrue())`.

**Correct:**

```go
ch := MustBeOk(registry.Get("my-channel"))
node := MustBeOk(tree.Find(key))
```

**Incorrect — never do this:**

```go
ch, ok := registry.Get("my-channel")
Expect(ok).To(BeTrue()) // ❌ use MustBeOk instead
// ... use ch
```

**When not to use `MustBeOk`:** if the test is asserting that `ok` is _false_ (e.g.,
verifying a key is absent), use `Expect(fn()).To(...)` with the appropriate matcher
directly — `MustBeOk` is only for the nominal success path.

#### Rule 4: Use `MustOpen` / `DeferClose` / `DeferCleanup` for closer teardown

When a test opens a resource that implements `io.Closer` (databases, ontologies,
writers, servers, etc.), you **must** schedule its `Close` via one of:

- `MustOpen(v, err)` — unwraps an `(io.Closer, error)` return and registers
  `DeferCleanup(v.Close)` in one call. Use this when the constructor returns an error.
- `DeferClose(v)` — registers `DeferCleanup(v.Close)` for an already-opened closer. Use
  this when the value was obtained without an error return (or was unwrapped
  separately).
- `ginkgo.DeferCleanup(...)` directly — use only when the teardown is more than a single
  `Close` call (e.g., multi-step shutdown with ordering).

Do **not** call `Close` from `AfterEach` / `AfterAll` / `AfterSuite`. Cleanup belongs
next to the `Open`, not scattered across lifecycle hooks. `DeferCleanup` runs in LIFO
order within the current Ginkgo scope, which gives correct teardown ordering for free
and keeps each resource's setup and teardown visually adjacent.

**Correct:**

```go
import . "github.com/synnaxlabs/x/testutil"

var _ = Describe("Ontology", func() {
    var otg *ontology.Ontology
    BeforeAll(func() {
        db := MustOpen(pebble.Open("", &pebble.Options{FS: vfs.NewMem()}))
        otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
    })

    It("does a thing", func() { /* ... */ })
})
```

**Also correct — `DeferClose` when the value is already in hand:**

```go
w := DeferClose(svc.NewWriter(ctx, cfg))
```

**Incorrect — never do this:**

```go
var otg *ontology.Ontology
BeforeAll(func() {
    otg = MustSucceed(ontology.Open(ctx, cfg))
})
AfterAll(func() {
    Expect(otg.Close()).To(Succeed()) // ❌ use MustOpen / DeferClose instead
})
```

#### Rule 5: Never read raw from a channel in tests — always use `Eventually`

A raw `<-ch` read in a test can block forever if the expected send never happens,
causing the test to hang until the Ginkgo spec-level timeout (often many seconds or
minutes) instead of failing fast with a useful message. You **must** use Gomega's
channel matchers, which are bounded by `Eventually`'s timeout and produce clear
diagnostics on failure.

- `Eventually(ch).Should(Receive())` — assert that a value is received (value
  discarded).
- `Eventually(ch).Should(Receive(&v))` — assert that a value is received and bind it to
  `v` for subsequent assertions. **Prefer this form** when you need the value.
- `Eventually(ch).Should(Receive(Equal(expected)))` — assert on the received value
  inline without binding.
- `Eventually(ch).Should(BeClosed())` — assert that the channel is closed.
- `Consistently(ch).ShouldNot(Receive())` — assert nothing is sent within the
  consistency window (use sparingly; it takes real time).

**Correct:**

```go
var frame Frame
Eventually(frames).Should(Receive(&frame))
Expect(frame.Len()).To(Equal(10))

Eventually(done).Should(BeClosed())
```

**Incorrect — never do this:**

```go
frame := <-frames            // ❌ blocks forever on failure
Expect(frame.Len()).To(Equal(10))

<-done                       // ❌ same — use Eventually(done).Should(BeClosed())
```

**Non-blocking select is also wrong:**

```go
select {
case frame = <-frames:       // ❌ flaky; timing-dependent
default:
    Fail("no frame")
}
```

Use `Eventually(...).Should(Receive(...))` instead — it polls with a proper timeout and
surfaces the channel's state (empty, closed, value received) in the failure message.

#### Rule 6: Errors returned from `defer` in tests must be asserted on

When a `defer` statement inside a test (or inside `BeforeEach` / `BeforeAll` / `It` /
any test-scoped function) invokes a function that returns an error, that error **must**
be asserted on. A deferred error that is silently dropped is a regression waiting to
happen: the cleanup path can fail indefinitely without the suite ever turning red.

In almost every case, the right answer is **not** `defer` at all — use `DeferCleanup`,
`MustOpen`, or `DeferClose` (see Rule 4), which already build the assertion in. Plain
`defer` should only appear when the cleanup is a single inline expression that is easier
to read this way, and even then the returned error must be wrapped in an assertion.

**Preferred:**

```go
db := MustOpen(pebble.Open("", opts))            // DeferCleanup + Close assertion built-in
w := DeferClose(svc.NewWriter(ctx, cfg))         // same
DeferCleanup(func() { Expect(svc.Flush(ctx)).To(Succeed()) })
```

**Acceptable when plain `defer` is genuinely clearer:**

```go
defer func() { Expect(w.Close()).To(Succeed()) }()
```

**Incorrect — never do this:**

```go
defer w.Close()                   // ❌ Close returns an error that is silently dropped
defer func() { _ = w.Close() }()  // ❌ explicit discard is still a discard
defer svc.Flush(ctx)              // ❌ Flush error dropped
```

#### Rule 7: Use `DescribeTable` for parameterized tests

When a test asserts the **same behavior across a set of inputs** — multiple data types,
multiple input permutations, multiple error conditions, multiple file-system or config
variations — use Ginkgo's `DescribeTable` with one `Entry` per case. Do not open-code it
as a `for` loop of `It` blocks or as repeated near-identical `It` specs.

Why:

- Each `Entry` becomes its own spec, so failures name the failing case directly instead
  of a loop index.
- Focus (`--focus`) and skip work per-entry.
- Parallelism and randomization still apply across entries.
- The table header makes the test's intent readable at a glance.

**Correct:**

```go
DescribeTable("parses valid timestamps",
    func(input string, expected telem.TimeStamp) {
        Expect(telem.ParseTimeStamp(input)).To(Equal(expected))
    },
    Entry("RFC 3339",       "2026-01-01T00:00:00Z", telem.TimeStamp(1767225600_000_000_000)),
    Entry("unix seconds",   "1767225600",           telem.TimeStamp(1767225600_000_000_000)),
    Entry("unix millis",    "1767225600000",        telem.TimeStamp(1767225600_000_000_000)),
)

DescribeTable("rejects invalid timestamps",
    func(input string) {
        Expect(telem.ParseTimeStamp(input)).Error().To(MatchError(telem.ErrInvalidTimeStamp))
    },
    Entry("empty string", ""),
    Entry("garbage",      "not-a-date"),
    Entry("negative",     "-1"),
)
```

**Incorrect — per-case `It` specs that duplicate the assertion:**

```go
It("parses RFC 3339", func() {
    Expect(telem.ParseTimeStamp("2026-01-01T00:00:00Z")).To(Equal(ts))
})
It("parses unix seconds", func() {
    Expect(telem.ParseTimeStamp("1767225600")).To(Equal(ts))
})
It("parses unix millis", func() {
    Expect(telem.ParseTimeStamp("1767225600000")).To(Equal(ts))
})
```

**Incorrect — a `for` loop over cases inside one `It`:**

```go
It("parses timestamps", func() {
    for _, tc := range []struct{ in string; want telem.TimeStamp }{
        {"2026-01-01T00:00:00Z", ts},
        {"1767225600", ts},
    } {
        Expect(telem.ParseTimeStamp(tc.in)).To(Equal(tc.want)) // ❌ failure doesn't name the case
    }
})
```

**When not to use `DescribeTable`:** if each case requires meaningfully different setup
or different assertions, separate `It` specs are clearer. `DescribeTable` is for _the
same assertion over varying inputs_.

#### Rule 8: Never assert only that an error occurred — always match type or message

`HaveOccurred()` (and `Not(Succeed())`, `Not(BeNil())` on errors) is **forbidden**. An
error assertion must always identify _which_ error occurred, either by matching a
specific sentinel/typed error or by matching a substring of the message. Asserting that
"some error happened" lets regressions slip through: the test still passes if the code
fails for an entirely different reason than the one under test.

**Correct — match a specific error:**

```go
Expect(svc.Create(ctx, "")).Error().To(MatchError(validate.Error))
```

**Correct — match a message substring (when no typed error is available):**

```go
Expect(svc.Create(ctx, "")).Error().To(MatchError(ContainSubstring("name is required")))
```

**Correct — match both type and message with `SatisfyAll`:**

```go
Expect(svc.Create(ctx, "")).Error().To(SatisfyAll(
    MatchError(validate.Error),
    MatchError(ContainSubstring("name is required")),
))
```

**Incorrect — never do this:**

```go
Expect(svc.Create(ctx, "")).Error().To(HaveOccurred())       // ❌ which error?
Expect(err).To(HaveOccurred())                                // ❌ also violates Rule 2
Expect(err).ToNot(BeNil())                                    // ❌ same
```

## 4-Layer Architecture (Server)

The Synnax server (`/core/`) follows strict layering:

```
Interface Layer (HTTP/gRPC APIs)
         ↓
Service Layer (Business logic)
         ↓
Distribution Layer (Aspen clustering)
         ↓
Storage Layer (Cesium + Pebble)
```

**Rules:**

- Dependencies only flow downward
- Each layer exposes interfaces, not implementations
- Services use dependency injection for testability

## Common Patterns

### Dependency Injection

```go
type Service struct {
    db     *DB
    client *Client
}

func New(db *DB, client *Client) *Service {
    return &Service{db: db, client: client}
}
```

### Interface Segregation

```go
// Small, focused interfaces
type Reader interface {
    Read(ctx context.Context) (Frame, error)
}

type Writer interface {
    Write(ctx context.Context, frame Frame) error
}
```

### Error Wrapping

```go
import "github.com/cockroachdb/errors"

if err != nil {
    return errors.Wrap(err, "failed to process frame")
}
```

## Common Gotchas

- **Aspen**: Eventual consistency means metadata updates may take up to 1 second to
  propagate
- **Cesium**: Requires careful handling of overlapping time ranges to prevent write
  conflicts
- **Context**: Always pass `context.Context` as first parameter to functions
- **Goroutines**: Use `sync.WaitGroup` or `errgroup.Group` for proper cleanup
- **Testing**: Ensure tests don't leak goroutines using `ShouldNotLeakGoroutines()`

## Development Best Practices

- **Interfaces over concrete types**: Define interfaces for dependencies
- **Dependency injection**: Pass dependencies explicitly, avoid global state
- **Context propagation**: Use `context.Context` for cancellation and request-scoped
  values
- **Error wrapping**: Add context to errors using `errors.Wrap`
- **Table-driven tests**: Parameterize tests with slices of test cases
- **Goroutine safety**: Use mutexes or channels for shared state
- **Clean shutdown**: Implement graceful shutdown with context cancellation
