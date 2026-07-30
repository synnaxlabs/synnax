# Go Development

## Modules

`/core/` (server), `/aspen/` (distributed KV), `/cesium/` (time-series DB),
`/freighter/go/` + `/freighter/integration/` (transport + its test server), `/arc/go/`
(Arc compiler), `/alamos/go/` (instrumentation), `/oracle/` (schema codegen CLI),
`/x/go/` (shared utilities).

Modules reference siblings via relative-path `replace` directives in each `go.mod`.
`go.work` is gitignored but fine for local convenience.

## Commands

- Build: `cd <module> && go build ./...`
- Test: **always `ginkgo`, never `go test`**. `ginkgo` (package), `ginkgo -r`
  (recursive), `ginkgo -v`, `ginkgo --focus "Name"`.

## Style

golangci-lint built-in formatters (gofmt, gofumpt, gci, goimports, golines, swaggo;
configured in the root `.golangci.yaml`), 88-char lines, standard Go idioms. Imports
grouped: stdlib, external, internal. Format with `golangci-lint fmt` in the module,
check with `golangci-lint fmt --diff`. `golangci-lint run` (CI) also fails on
unformatted files.

## Packages & Naming

- Package names: lowercase, one word, singular (`channel`, `framer`, `writer`). When the
  natural name collides with a keyword, extend it (`ranger`, not `range`).
- One file per concern inside a package: `service.go`, `writer.go`, `retrieve.go`,
  `ontology.go`, `transport.go`. Tests co-located.
- Exported names are contextual to the package: `channel.Service`, `channel.Key`,
  `channel.Writer` (see the root namespace rule). The core type may share the package
  name: `channel.Channel`.
- `internal/` hides implementation packages consumers must not import
  (`cesium/internal`, `aspen/internal`).

## General Rules

### Rule 1: Never ignore errors

No `_ = fn()`, no silently dropped returns, no bare `defer conn.Close()`. Handle every
error by: **returning** it (wrapped via `errors.Wrap` for context), **logging** it via
instrumentation when the contract can't return it (deferred cleanup, background
goroutines), **asserting** on it in tests, or **panicking** only on unrecoverable
programmer bugs. No exceptions for "can't fail in practice" or "best-effort cleanup".

```go
defer func() {
    if err := conn.Close(); err != nil { s.L.Warn("close conn", zap.Error(err)) }
}()
```

### Rule 2: Only `github.com/synnaxlabs/x/errors`

`fmt.Errorf` is forbidden in all cases (including `%w`), as are the std `errors` package
and third-party error packages. Use `errors.New(msg)`, `errors.Newf(format, ...)`,
`errors.Wrap(err, msg)`, `errors.Wrapf(err, format, ...)`.

### Rule 3: Sentinel errors for matchable conditions

If callers (including tests) will branch on an error condition, represent it as an
exported package-level `var Err<Condition>` — never a one-off `errors.New` that forces
substring matching. **Reuse existing sentinels first** by wrapping them
(`query.ErrNotFound`, `validate.ErrValidation`, `freighter.EOF`) — this keeps
`errors.Is` matching for both the specific and general shape:

```go
var ErrNotFound = errors.Wrap(query.ErrNotFound, "channel not found")
```

Create a new sentinel only when no existing one fits and the error is part of the
package's public contract. Never speculatively. Message starts lowercase, no trailing
period, doc comment says when it's returned.

### Rule 4: Validation errors wrap `validate.ErrValidation`

Any error for caller-provided data failing a rule, format, or completeness check wraps
`validate.ErrValidation` (`github.com/synnaxlabs/x/validate`), never a bare
`errors.New`. Prefer its helpers: `validate.New(scope)` plus `Ternary`/`Ternaryf` join
multiple field failures; `NotNil`, `Positive`, `InBounds`, `NonZero`, `NotEmptySlice`,
`NotEmptyString` cover common checks; `PathedError` prefixes a field path for nested
structs.

```go
v := validate.New("cert.SourceConfig")
validate.NotEmptyString(v, "address", cfg.Address)
return v.Error()
```

This generalizes: wrap the best-fitting existing general-purpose error
(`validate.ErrValidation`, `query.ErrNotFound`, `freighter.EOF`, ...) instead of a fresh
one-off. Introduce a new error type only when none fits, and justify it in conversation,
not a comment.

### Rule 5: No throwaway debug scripts

Never create ad-hoc `main` packages or scratch files to explore behavior. Write a
focused Ginkgo `It("reproduces ...")` in the package under investigation and iterate
with `ginkgo --focus`. Temporary `fmt.Println` inside the test is fine (remove before
committing).

### Rule 6: Never edit generated code

Identified by `Code generated ... DO NOT EDIT.` headers and `.gen.` / `_gen` suffixes.
Edit the source `.oracle` schema in `/schemas/` instead, then run `oracle sync` — always
installing first via `./oracle/install.sh`, the only blessed install method (see
`oracle/CLAUDE.md` Sync Workflow). If the schema language can't express the change,
investigate the generator in `/oracle/` (`analyzer/`, `formatter/`, `resolution/`,
`exec/`), describe the gap and blast radius, and wait for user approval before touching
it.

### Rule 7: `any`, never `interface{}`

Everywhere — params, returns, map/slice elements, constraints, fields, assertions. Fix
`interface{}` to `any` when editing existing code.

### Rule 8: Use `any` with extreme caution

Default answer is no. Prefer, in order: concrete type → focused interface → generic type
parameter → sealed sum-like interface with one documented type switch → `any`.
Legitimate uses (exceptions, not the pattern): variadic logging/formatting args,
reflective serialization boundaries (`json.Marshal`), truly heterogeneous framework-edge
containers (DI registry). Justify each use in conversation with the user, not in an
inline comment.

### Rule 9: `set.Set[T]` for set membership

Use `set.Set[T]` from `github.com/synnaxlabs/x/set` — never hand-rolled `map[T]struct{}`
or `map[T]bool`. Same performance, clear intent. A map-of-bool is legitimate only when
the bool carries real tri-state/toggle meaning ("explicitly disabled" vs "not
configured").

### Rule 10: No `init()` functions, no ambient imports

Never use `init()` to run package-load-time side effects, and never import a package
solely for its `init()` (`import _ "pkg"`). Both hide behavior behind an import graph
instead of an explicit call. Wire the setup explicitly at the call site (a constructor,
an app-startup function) instead.

## Comments

The universal body-comment and doc-comment rules in the root CLAUDE.md apply.
Go-specific form:

**Doc comments are encouraged on all identifiers, exported and unexported.** Standard Go
conventions: start with the identifier's name, complete sentences ending in periods,
single `//` block directly above the declaration.

```go
// Create persists a new channel with the given name and returns its populated
// record. It returns validate.Error if name is empty or already taken.
func (s *Service) Create(ctx context.Context, name string) (Channel, error)
```

**Struct fields, including unexported, get a tight one-line `//` comment above the
field**: semantic role, units, valid range, invariants, or lock discipline
(`// mu guards pending and closed.`). Not mutation history, not a restatement of the
type, never trailing inline.

## Testing (Ginkgo/Gomega)

Suites in `*_suite_test.go`, tests in `*_test.go`, blackbox `package foo_test`. BDD
structure: `Describe`/`Context`/`It`/`Specify`; `Ordered` for sequential execution;
`Eventually`/`Consistently` for async; goroutine leak detection via
`ShouldNotLeakGoroutines()`. Utilities from `github.com/synnaxlabs/x/testutil`:
`MustSucceed`, `MustSucceed2`, `MustBeOk`, `MustOpen`, `DeferClose`.

Hard rules — no exceptions:

### Rule 1: Unwrap `(T, error)` with `MustSucceed` / `MustSucceed2`

Never bind the error and assert `Expect(err).ToNot(HaveOccurred())` separately.

```go
ch := MustSucceed(svc.Create(ctx, "my-channel"))
frame, release := MustSucceed2(reader.Read(ctx))
```

### Rule 2: Assert errors inline with `Expect(fn()).Error().To(MatchError(...))`

Never capture the error into a variable first, not even the other return with `_`: that
discard means the test can't catch a partial-result bug, where the function wrongly
returns non-zero alongside the error. `.Error()` matches the error and asserts every
other return value is zero-valued.

```go
// Bad: discards the Source; a half-built Source on this error path would still pass.
_, err := auto.Factory{}.NewSource(cfg)
Expect(err).To(MatchError(myError))

// Good: asserts the Source is zero-valued too.
Expect(auto.Factory{}.NewSource(cfg)).Error().To(MatchError(myError))
```

### Rule 3: Unwrap `(T, bool)` with `MustBeOk`

```go
ch := MustBeOk(registry.Get("my-channel"))
```

When asserting `ok` is false (absence), use a matcher directly instead.

### Rule 4: Closer teardown via `MustOpen` / `DeferClose` / `DeferCleanup`

Never call `Close` from `AfterEach`/`AfterAll`/`AfterSuite` — cleanup belongs next to
the open. `MustOpen(v, err)` unwraps and registers `DeferCleanup(v.Close)`;
`DeferClose(v)` registers for an already-opened closer; raw `DeferCleanup` only for
multi-step shutdowns. `DeferCleanup` runs LIFO within scope.

```go
db := MustOpen(pebble.Open("", opts))
w := DeferClose(svc.NewWriter(ctx, cfg))
```

### Rule 5: Never read raw from a channel

`<-ch` blocks forever on failure; non-blocking `select` is flaky. Use
`Eventually(ch).Should(Receive())` / `Receive(&v)` (prefer when the value is needed) /
`Receive(Equal(expected))` / `BeClosed()`, and `Consistently(ch).ShouldNot(Receive())`
sparingly (it takes real time).

### Rule 6: Errors from `defer` in tests must be asserted

Prefer `MustOpen`/`DeferClose`/`DeferCleanup` (assertion built in). If plain `defer` is
genuinely clearer: `defer func() { Expect(w.Close()).To(Succeed()) }()`.
`defer w.Close()` and `defer func() { _ = w.Close() }()` are both discards.

### Rule 7: `DescribeTable` for parameterized tests

Same assertion over varying inputs → `DescribeTable` with one `Entry` per case (failures
name the case; focus/skip work per-entry). Never a `for` loop of cases inside one `It`,
never repeated near-identical `It`s. Separate `It`s only when cases need meaningfully
different setup or assertions.

### Rule 8: Never assert only that an error occurred

`HaveOccurred()`, `Not(Succeed())`, `Not(BeNil())` on errors are forbidden — they pass
when the code fails for the wrong reason. Always identify the error:
`MatchError(sentinel)`, `MatchError(ContainSubstring("..."))`, or both via
`SatisfyAll(...)`.

### Rule 9: No export_test.go

Never create `export_test.go` files (or any pattern that re-exports unexported
identifiers for tests). Tests exercise the public API; if a behavior is only reachable
through an unexported symbol, test it through the exported surface that uses it, or
reconsider whether the symbol should exist.

### Rule 10: Every suite wires goroutine leak detection

Every `*_suite_test.go` must call one of the two leak checkers from
`github.com/synnaxlabs/x/testutil`, no exceptions:

- `var _ = ShouldNotLeakGoroutinesPerSpec()` at file scope (bottom of the file, after
  `RunSpecs`) — the default. Snapshots goroutines before each spec and asserts none leak
  after, catching leaks with per-spec precision.
- `ShouldNotLeakGoroutines()` called inside `BeforeSuite` — only when the suite needs a
  single suite-wide snapshot (e.g. shared expensive setup whose goroutines shouldn't
  count as steady-state per spec).

```go
func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Distribution Channel Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()
```
