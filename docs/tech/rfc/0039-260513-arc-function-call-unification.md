# 39 - Arc Function-Call Unification

**Feature Name**: Arc Function-Call Unification <br /> **Start Date**: 2026-05-13 <br />
**Authors**: Nico Alba <br />

**Related:** [RFC 0030 - Arc Module System](./0030-260221-arc-modules.md),
[RFC 0031 - Arc Scheduler Semantics](./0031-260311-arc-scheduler-semantics.md),
[RFC 0037 - Arc Status Module Updates](./0037-260427-arc-status-updates.md)

# 0 - Summary

Arc's type system today carries two separate parameter arrays on every function
signature: `Config` (read by the flow analyzer for the `{...}` brace block) and `Inputs`
(read by the expression analyzer for the `(...)` parens form). For an `ExecBoth`
function to be callable in both contexts, the same parameter list is **mirrored** into
both fields. The flow analyzer then disambiguates "wire value feeds a param" from "wire
is pure activation" via the heuristic
`upstreamIsTrigger := Exec == ExecBoth && len(Config) > 0`.

This RFC collapses the two arrays into a single `Inputs` list. There is no replacement
tag, no `BindMode` field, no "capture vs argument" distinction at the type level. A
function has params. Period. The `ExecBoth` heuristic is replaced by an explicit
`Trigger` field on `Symbol` that names which param (if any) receives the upstream wire's
value in flow context.

User-facing Arc syntax does not change. `name(...)`, `name{...}`, and
`wire -> name{...}` continue to parse, analyze, compile, and run exactly as they do
today. The change is structural: one parameter list replaces two, one analyzer path
replaces two, one hook replaces two, and one declaration replaces a runtime heuristic.

The refactor establishes the foundation for the future trigger-as-argument feature
(`{message: incoming_ch} -> status.set("X", message)`), which a separate RFC will
specify. That feature becomes a call-site override of the symbol-level `Trigger` default
and works against any param, since there is no longer any param-property distinction at
the type level for it to interact with.

# 1 - Goals

**Eliminate the Config/Inputs dichotomy at the type level.** A function has one
parameter list. There is no per-param flag distinguishing "configuration" from "runtime
input." Whatever conceptual distinction the original split was reaching for is
recoverable from each symbol's own semantics; the type system does not need to encode
it.

**Replace the `ExecBoth` heuristic with an explicit declaration.** Today the flow
analyzer guesses whether an upstream wire is a value source or a pure activation pulse
by inspecting `Exec` and `len(Config)`. The guess happens to be correct for every stdlib
symbol today because of the mirror trick, but the rule is implicit. Replace it with a
`Trigger` field on `Symbol` that names the wire-fed param (or declares `TriggerOnly`).

**Collapse the two analyzer call-validation paths.** `validateFunctionCall` (parens
form) and `validateFuncConfig` (brace form) silently agree only because Inputs is
mirrored to Config in every `ExecBoth` symbol. Replace both with a single `call.Analyze`
routine.

**Collapse the two analyzer hook surfaces.** A symbol that wants to validate its
arguments (e.g. `status.set` constraining the `variant` literal; see
[RFC 0037 §5.0.1](./0037-260427-arc-status-updates.md#501---literal-value-constraints))
registers `AnalyzeCall` for the parens form AND `AnalyzeFlowConfig` for the brace form.
The two hooks walk different AST shapes to find the same argument and run the same
literal check. Replace both with a single `AnalyzeArguments` hook that receives a
unified `[]Argument` view.

# 2 - Non-Goals

- **User-facing Arc syntax.** `name(...)`, `name{...}`, and `wire -> name{...}` parse,
  analyze, and run identically. The grammar is untouched.
- **The trigger-as-argument feature.** `{message: incoming_ch} -> status.set(...)` is
  out of scope. This RFC only lands the structural foundation it will sit on. A
  follow-on RFC will specify the grammar and analyzer for the override.
- **Firing semantics with multiple bound wires.** Any / All / user-specified: the policy
  decision is deferred to the trigger-as-argument RFC. Nothing in this RFC precludes any
  choice.
- **Renaming the unified field to `Params`.** The grammar already calls the parens form
  `inputList`, the compiler already concatenates the two arrays into an "Inputs"-named
  local, and the slice type is `types.Params`. Renaming the field creates a
  `Params Params` redundancy. `Inputs` stays.
- **Implementing optional parameter semantics.** The `Optional` field on `Param` exists
  in the struct but is set to `false` everywhere in this RFC; every param is required.
  Default substitution at the call site and preserve-on-omit dispatch in host functions
  remain future work. See [Section 8.1](#81---optional-parameters).

# 3 - The Problem

## 3.0 - How the split became debt

Early Arc symbols were single-context: a flow node was flow-only (`time.interval`,
`time.wait`), a function was func-only (early `math` and `op` symbols). In that world
the Config/Inputs split read as load-bearing: each field corresponded to the context the
symbol participated in, and the timing/binding semantics of its params were unambiguous
within that context.

That assumption broke under the **parity tenet**: if a symbol works in one context, it
should work the same in the other. The first attempt at parity was `ExecBoth` plus the
mirror trick: declare the params once, assign the same list to both `Config` and
`Inputs`, and let each context read from its own field. It worked. `status.set` (in
[`core/pkg/service/arc/status/status.go`](../../../core/pkg/service/arc/status/status.go))
ships today on this mechanism. (`time.now` is also `ExecBoth` but takes no params at
all, so the mirror trick is degenerate there.) Most recently, **every user-defined
function** became `ExecBoth` as well
([`arc/go/analyzer/function/function.go`](../../../arc/go/analyzer/function/function.go)),
so that a user function with inputs is callable from both the parens and brace forms;
this is the change that first exercised `ExecBoth` on a symbol carrying actual params,
and it is what forced the trigger heuristic to grow a second discriminator (Section
3.2).

Once parity made symbols cross-context, the Config/Inputs split stopped being
load-bearing. In the pre-parity world it carried real semantic weight: a symbol could
appear in one context and one only, and the field it landed in named that context. Once
a symbol could appear in both, the field could no longer name the context, and the
mirror trick is the proof. If the two fields had still been encoding distinct semantic
content, copying the same data into both could not possibly have produced correct
symbols; the mismatch would surface as a real failure. It didn't. The fields had
collapsed to **declaration site** (which slot a param was placed in: the `{...}` block
of a user-defined function, or the `Config:` field of a stdlib symbol's Go declaration)
with no remaining semantic content. What _was_ doing semantic work (wire-feedability) is
the only piece that needed preservation, and now lives on `Symbol.Trigger`.

This RFC finishes what `ExecBoth` started. The mirror trick was a tactical fix that
exposed the strategic problem; the strategic fix is to collapse the field whose
redundancy the workaround already demonstrated, before more symbols accrue the same
scaffolding.

## 3.1 - The Mirror Trick

For an `ExecBoth` function to be callable in both contexts, the same parameter list must
appear in both `Config` (read by the flow analyzer) and `Inputs` (read by the expression
analyzer).
[`core/pkg/service/arc/status/status.go`](../../../core/pkg/service/arc/status/status.go)
makes this concrete:

```go
// newSetSymbolType returns a fresh function type per call so analysis never
// mutates a shared symbol. Empty-string defaults mark the inputs optional so
// flow-form usage (Config-fulfilled, no edges) analyzes.
func newSetSymbolType() types.Type {
    params := types.Params{
        {Name: "key_or_name", Type: types.String(), Value: ""},
        {Name: "message",     Type: types.String(), Value: ""},
        {Name: "variant",     Type: types.String(), Value: ""},
    }
    return types.Function(types.FunctionProperties{
        Inputs:  params,  // mirrored
        Config:  params,  // mirrored
        Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
    })
}
```

The mirroring is required, not idiomatic. A regression test in
[arc/go/stl/stl_test.go](../../../arc/go/stl/stl_test.go) ("Should obey the ExecBoth
structural contract on every ExecBoth symbol") enforces that any `ExecBoth` symbol's
`Inputs` mirrors its `Config` one-for-one. The empty-string `Value` defaults exist
solely so the brace form parses when the user supplies no `{...}` block.

## 3.2 - The Trigger Heuristic

[arc/go/analyzer/flow/flow.go](../../../arc/go/analyzer/flow/flow.go) decides whether an
upstream wire carries a value into a param or is pure activation via:

```go
// Dual-shape ExecBoth (see symbol.ExecBoth): upstream is a trigger, not
// a typed input.
upstreamIsTrigger := funcType.Exec == symbol.ExecBoth && len(freshType.Config) > 0
```

This is a heuristic, and it has since **proliferated to three sites** that no longer
agree on the discriminator:

- [arc/go/analyzer/flow/flow.go](../../../arc/go/analyzer/flow/flow.go) — the primary
  upstream-handling block, using `len(freshType.Config) > 0`.
- [arc/go/analyzer/flow/flow.go](../../../arc/go/analyzer/flow/flow.go) — a second copy
  in the routing-table (`select`) branch, using `len(fnType.Type.Config) > 0`.
- [arc/go/text/analyze.go](../../../arc/go/text/analyze.go) — text-to-IR, which uses a
  **different** discriminator entirely: `sym.Exec == symbol.ExecBoth && sym.AST == nil`.

The `sym.AST == nil` clause in the third site exists because `AST` presence has become a
load-bearing discriminator in its own right. Its immediate cause is user-defined
functions becoming `ExecBoth` (Section 3.0): a user function carries a non-nil `AST` and
must keep its `Inputs`, whereas a stdlib `ExecBoth` symbol (`AST == nil`) drops them in
flow form. But the clause fences off more than user functions — `AST != nil` covers
**every synthetic or user-authored node that carries a source AST**: user functions,
inline flow expressions, and format-string (`f"..."`) nodes. Format strings are callable
in both contexts too, but on a separate path: in flow form, `f"x: {ch}" -> log` registers
a synthetic `KindFunction` (Exec is the zero value, _not_ `ExecBoth`) whose interpolated
channel reads happen via host calls and whose activation is stratum membership rather
than a trigger edge ([arc/go/analyzer/flow/expression.go](../../../arc/go/analyzer/flow/expression.go)).
So the rule that decides "is this wire a trigger" is now expressed two incompatible ways
across the codebase, and the `AST` half of it is entangled with cross-context mechanisms
that have nothing to do with `ExecBoth`.

It happens to produce the right answer for every symbol today because of the mirror
trick and the `AST` accident. But none of the three sites ever says "this function takes
its wire as a trigger, not a value." The truth is implicit in structural properties
(`Exec` mode, Config length, AST presence), none of which is the property being tested.
A future function whose intent diverges from these patterns has no way to express that
today.

## 3.3 - The Two Analyzer Paths

`validateFunctionCall` in
[arc/go/analyzer/expression/expression.go](../../../arc/go/analyzer/expression/expression.go)
and `validateFuncConfig` in
[arc/go/analyzer/flow/flow.go](../../../arc/go/analyzer/flow/flow.go) implement two
versions of the same validation. Reduced to essentials, the forms looks like this:

```go
// validateFunctionCall (parens form). Reads from funcType.Inputs.
args := funcCall.ArgumentList().AllExpression()
if len(args) < funcType.Inputs.RequiredCount() || len(args) > len(funcType.Inputs) {
    diag("function %s expects %d arg(s), got %d", ...)
    return
}
for i, arg := range args {
    paramType := funcType.Inputs[i].Type
    argType  := types.InferFromExpression(arg)
    if !types.Compatible(argType, paramType) {
        diag("argument %d of %s: expected %s, got %s", ...)
    }
}

// validateFuncConfig (brace form). Reads from fnType.Config.
if named := configBlock.NamedConfigValues(); named != nil {
    for _, cv := range named.AllNamedConfigValue() {
        key := cv.IDENTIFIER().GetText()
        expectedType, exists := fnType.Config.Get(key)
        if !exists { diag("unknown config parameter '%s' for func '%s'", ...); continue }
        exprType := atypes.InferFromExpression(cv.Expression())
        atypes.Check(ctx.Constraints, expectedType.Type, exprType, ...)
    }
} else if anon := configBlock.AnonymousConfigValues(); anon != nil {
    exprs := anon.AllExpression()
    if len(exprs) > len(fnType.Config) { diag("too many config values..."); return }
    for i, expr := range exprs {
        param    := fnType.Config[i]
        exprType := atypes.InferFromExpression(expr)
        atypes.Check(ctx.Constraints, param.Type, exprType, ...)
    }
}
for _, param := range fnType.Config {
    if !supplied[param.Name] && param.Value == nil {
        diag("missing required config parameter '%s' for func '%s'", ...)
    }
}
```

Each routine does roughly the same work: argument-count check, per-argument type check,
and (in the brace form) name resolution and required-coverage check. They reach into
different fields on the same `Type`: `funcType.Inputs` vs `fnType.Config`. They silently
agree only because Inputs mirrors Config in every `ExecBoth` symbol. The divergent
capabilities make the duplication worse than pure repetition:

- Only `validateFunctionCall` rejects multiple-named-outputs callables. A flow-form call
  to such a function would pass analysis today.
- Only `validateFuncConfig` supports named arguments, detects unknown names, and emits
  the missing-required diagnostic. The parens-form call relies entirely on positional
  count matching.

Any future divergence beyond these would be a quiet bug class: a function whose parens
form accepts an arg the brace form rejects, or vice versa, with no test that would catch
it. `call.Analyze` (Section 4.2) merges both paths into one routine that does the union
of their checks against the unified `Inputs` list.

## 3.4 - The Two-Hook Duplication

A symbol that wants to validate its arguments (e.g. compile-time check of the `variant`
literal in `status.set`) registers two hooks:

```go
// In core/pkg/service/arc/status/status.go
member := &symbol.Symbol{
    Name:              setMemberName,
    Kind:              symbol.KindFunction,
    Exec:              symbol.ExecBoth,
    Type:              newSetSymbolType(),
    // ...
    AnalyzeCall:       analyzeStatusSetCall,        // parens form
    AnalyzeFlowConfig: analyzeStatusSetFlowConfig,  // brace form
}
```

The two hook implementations are essentially identical: both find the `variant` argument
and call `checkVariantLiteral`. They differ only in which AST shape they walk
(`IFunctionCallSuffixContext` vs `IConfigValuesContext`). The duplication exists because
the same logical argument list lives in two parser nodes.

# 4 - Proposed Design

## 4.0 - Unified Inputs

`arc/go/types/types.gen.go` (regenerated from `schemas/arc/types.oracle`):

```go
type FunctionProperties struct {
    Inputs  Params `json:"inputs"  msgpack:"inputs"`
    Outputs Params `json:"outputs" msgpack:"outputs"`
    // Config field removed. No replacement.
}

type Param struct {
    Name     string `json:"name"     msgpack:"name"`
    Type     Type   `json:"type"     msgpack:"type"`
    Value    any    `json:"value"    msgpack:"value"`
    Optional bool   `json:"optional" msgpack:"optional"` // Stub, set all to False (Sec. 8.1)
}
```

No `BindMode` field, no `Captures()` / `Arguments()` helpers, no per-param tag. Whatever
"config" vs "input" was trying to encode at the type level is gone. A function has
params; each param has a name, a type, an optional default, and an optional flag.

**The `Optional` field is a stub in this RFC.** Every stdlib symbol declaration sets
`Optional: false`; every param is required. This RFC does not implement the semantics
that would make a `true` value behave any differently: no default-value substitution at
the call site, no preserve-on-omit handling in host functions. Section 8.1 sketches how
those land later; the only thing this RFC commits to is leaving the field in place for
that future work to attach to.

A `Param` gains one internal field (json:`"-"`), `AST antlr.ParserRuleContext`, so the
symbol-table builder can construct symbols with source-location info without re-walking
the parse tree. It is not part of the serialized shape.

Param ordering is preserved as the user declared it. The compiler already produces a
deterministic order at the WASM ABI layer (`slices.Concat(Config, Inputs)`); the
migration appends the old Config list before the old Inputs list to preserve that order
byte-for-byte (Section 5).

## 4.1 - Explicit Trigger Binding

`arc/go/symbol/symbol.go`:

```go
type TriggerBinding struct {
    // Target names the param that receives the upstream wire's value in flow
    // context. Empty means the wire is pure activation (no value bound).
    Target string
}

var TriggerOnly = TriggerBinding{}
func TriggerInput(name string) TriggerBinding { return TriggerBinding{Target: name} }

type Symbol struct {
    // ... existing fields ...
    Trigger TriggerBinding
}
```

Every symbol with `Exec` including `ExecFlow` gains an explicit `Trigger:` line. The
assignment is a mechanical audit driven by today's behavior. Symbols are grouped by
where they're declared, since the implementer touches each set in a different file
sweep.

**Arc stdlib** (`arc/go/stl/`):

| Symbol                                                            | Today (`Exec`, `Config`, `Inputs`)                | Trigger                      |
| ----------------------------------------------------------------- | ------------------------------------------------- | ---------------------------- |
| `time.interval`                                                   | `ExecFlow`, `[period]`, none                      | `TriggerOnly`                |
| `time.wait`                                                       | `ExecFlow`, `[duration]`, none                    | `TriggerOnly`                |
| `time.now`                                                        | `ExecBoth`, none, none                            | `TriggerOnly`                |
| `channel.on`                                                      | `ExecFlow`, `[channel]`, none                     | `TriggerOnly`                |
| `channel.write`                                                   | `ExecFlow`, `[channel]`, `[input]`                | `TriggerInput("input")`      |
| `constant.constant`                                               | `ExecFlow`, `[value]`, none                       | `TriggerOnly`                |
| `stable.for`                                                      | `ExecFlow`, `[duration]`, `[input]`               | `TriggerInput("input")`      |
| `math.avg` / `min` / `max`                                        | `ExecFlow`, `[duration, count]`, `[input, reset]` | `TriggerInput("input")`      |
| `math.derivative`                                                 | `ExecFlow`, none, `[input]`                       | `TriggerInput("input")`      |
| `op.{ge,gt,le,lt,eq,ne,and,or}`                                   | `ExecFlow`, none, `[a, b]`                        | `TriggerInput("a")`          |
| `op.not`                                                          | `ExecFlow`, none, `[input]`                       | `TriggerInput("input")`      |
| `selector.select`                                                 | `ExecFlow`, none, `[condition]`                   | `TriggerInput("condition")`  |
| `authority.set`                                                   | `ExecFlow`, `[value, channel]`, `[output]`        | `TriggerInput("output")`     |
| `math.pow`                                                        | `ExecWASM`, none, varies                          | zero value (never consulted) |
| `series.len`, `series.{element_*, series_*}` (internal host syms) | `ExecWASM`, none, varies                          | zero value (never consulted) |
| `string.{from_literal, concat, equal, ...}`                       | `ExecWASM`, none, varies                          | zero value (never consulted) |
| `state.{load, store, load_series, store_series, ...}`             | `ExecWASM`, none, varies                          | zero value (never consulted) |
| `error.panic`                                                     | `ExecWASM`, none, `[ptr, len]`                    | zero value (never consulted) |

**Service layer** (`core/pkg/service/arc/`):

| Symbol       | Today (`Exec`, `Config`, `Inputs`)                     | Trigger                     |
| ------------ | ------------------------------------------------------ | --------------------------- |
| `status.set` | `ExecBoth`, mirrored `[key_or_name, message, variant]` | `TriggerOnly` (post-dedupe) |

**User-defined functions:** every user-defined function is now `ExecBoth`
([`arc/go/analyzer/function/function.go`](../../../arc/go/analyzer/function/function.go)),
with `Config` holding the brace-block params and `Inputs` the parens-block params. Under
this RFC they are auto-assigned `TriggerInput(<first parens-block param>)` if a parens
block is present, else `TriggerOnly`.

The "mixed-shape" rows (`channel.write`, `stable.for`, `math.*`, `authority.set`) are
the ones that today have **both** `Config` and `Inputs` populated with **different**
params. Phase 5 collapses them to a single `Inputs:` list with old `Config` items first
(matching `slices.Concat(Config, Inputs)` and the codec migration's append order); the
trigger param is the first old-`Inputs` param. Mirror-trick rows (`status.*`)
deduplicate; non-mirror rows concat.

`ExecWASM`-only symbols leave `Trigger` at the zero value. The runtime never consults
`Trigger` for symbols that can't appear in flow context, so populating it would be
noise. The registration invariant accommodates an empty `Target`.

A unit test in `stl_test.go` guards the field: `Trigger.Target`, if non-empty, must name
an existing param on the same symbol. This catches the rename/delete failure mode at
`go test` time, not at the first flow-call site.

**Why symbol-level, not param-level.** The semantic "which param does the wire feed by
default" is a function-level decision. It varies per function regardless of which params
are present. Encoding it on `Symbol` puts it where it logically lives.

## 4.2 - Unified Call Analyzer

New shared package `arc/go/analyzer/call/call.go`:

```go
package call

type Argument struct {
    Index int
    Name  string                       // empty for positional
    Expr  parser.IExpressionContext
    AST   antlr.ParserRuleContext      // for diagnostics
}

func Analyze[T antlr.ParserRuleContext](
    ctx acontext.Context[T],
    fnName string,
    fnType types.Type,
    args []Argument,
    site antlr.ParserRuleContext,
)
```

No `Form` parameter. No subset to filter against. `call.Analyze` matches `args` against
`fnType.Inputs`: every arg, every param, one list. Whether the arg matches by name or by
index is derivable from `args[i].Name != ""`.

The routine performs the union of today's two paths: argument-count check, per-argument
type check via `atypes.Check`, name resolution for named args, unknown-name and
duplicate-name detection, required-param coverage check, and invocation of the symbol's
`AnalyzeArguments` hook. The two AST-shape adapters (parens → `[]Argument`, brace →
`[]Argument`) are tiny; the substance of the validation lives in one place.

## 4.3 - Unified Hook

`arc/go/symbol/hooks/hooks.go`:

```go
// Replaces CallHook and FlowConfigHook.
type ArgumentsHook func(ctx any, args []Argument)
```

`status.set`'s two hook bodies collapse into one:

```go
const variantParamName = "variant"
const variantIndex = 2

func analyzeStatusSetArguments(ctx any, args []hooks.Argument) {
    diags := extractDiagnostics(ctx)
    if diags == nil { return }
    for _, a := range args {
        if a.Name == variantParamName || (a.Name == "" && a.Index == variantIndex) {
            checkVariantLiteral(diags, a.Expr)
            return
        }
    }
}
```

`checkVariantLiteral` is unchanged. The hook sees a unified `[]Argument` view regardless
of which surface form produced it.

## 4.4 - Flow Analyzer Collapse

The three-branch upstream-handling block at
[arc/go/analyzer/flow/flow.go:102-213](../../../arc/go/analyzer/flow/flow.go)
(`prevIDNode` / `prevExpr` / `prevFuncNode`) collapses around a direct consult of the
symbol's `Trigger`:

```go
target := fn.Trigger.Target
if target == "" {
    // wire is pure activation; do not type-check the upstream value
} else {
    targetParam, ok := fn.Type.Inputs.Get(target)
    if !ok {
        diag("symbol '%s' declares Trigger target '%s' but has no such param", name, target)
    } else if suppliedAtCallSite[target] {
        diag("param '%s' is bound by both call-site args and upstream wire", target)
    } else {
        atypes.Check(upstreamType, targetParam.Type, ...)
        // emit edge: upstream → targetParam
    }
}
```

The `upstreamIsTrigger` heuristic is removed.

# 5 - Codec Migration

The schema change is a wire-format break. Persisted Arc programs (IR JSON, IR msgpack,
proto) deserialize through a codec migration that translates the pre-refactor shape into
the unified shape.

**Snapshot.** Copy the pre-refactor `arc/go/types/types.gen.go` and
`arc/go/ir/types.gen.go` into `arc/go/types/migrations/vN/` and
`arc/go/ir/migrations/vN/` respectively. Frozen; never regenerates again.

**Translation.** `MigrateFunction` and `MigrateNode` walk the vN snapshot:

- Concatenate the old `Config` list and the old `Inputs` list into the unified `Inputs`
  list, in that order. This preserves the WASM ABI ordering the compiler already
  produces today.
- For stdlib `ExecBoth` symbols where `Config == Inputs` (the mirror trick), the
  duplicate is detected and dedupes to a single list. The mirror was a workaround for
  the Config/Inputs split; with the split gone, the workaround is no longer needed.

**Verification.** `migrations/vN/migrate_test.go` round-trips a pre-vN snapshot through
the migrator and asserts the resulting IR Type-shape matches what a fresh declaration in
the new shape produces. Persisted Arc programs from the pre-refactor codec version
deserialize identically.

**Scope: Go-only deserialization of historical IR.** Persisted Arc programs are
deserialized exclusively in the Go server (the existing v54 migration test lives only at
[core/pkg/service/arc/migrations/v54/migrate_test.go](../../../core/pkg/service/arc/migrations/v54/migrate_test.go),
with no TS or C++ counterpart). The C++ and TypeScript codec bindings regenerated in
Phase 10 are output consumers only; they receive post-migration data from the Go server
and never see pre-vN bytes directly. No hand-written C++ or TS migration is required.

# 6 - Changes by Layer

## 6.0 - Type System and IR

**Files:** `arc/go/types/types.gen.go`, `arc/go/types/type.go`, `arc/go/types/fresh.go`,
`arc/go/types/migrations/vN/`, `arc/go/ir/types.gen.go`, `arc/go/ir/function.go`,
`arc/go/ir/node.go`, `arc/go/ir/migrations/vN/`

`types.gen.go` (regenerated): drop `Config` from `FunctionProperties`; add internal
`AST` field to `Param`. `type.go`: update `Equal` and `paramsEqual` to compare against
the single Inputs list. `fresh.go`: drop Config freshen. `ir/types.gen.go`
(regenerated): drop `Config` from `Function` and `Node`. `ir/function.go` and
`ir/node.go`: drop the Config rendering branch from `Type()` and string output. Both
`migrations/vN/` directories snapshot the pre-refactor shape; the IR side additionally
gains a hand-written `migrate.go` (Section 5).

## 6.1 - Symbol Table

**Files:** `arc/go/symbol/symbol.go`, `arc/go/symbol/hooks/hooks.go`,
`arc/go/symbol/scope.go`

Replace `AnalyzeCall` + `AnalyzeFlowConfig` with `AnalyzeArguments`. Add
`Trigger TriggerBinding` field and the `TriggerBinding` / `TriggerOnly` /
`TriggerInput(...)` types and helpers. `ResolveConfigChannel` reads `Inputs.Get(...)`.

## 6.2 - Analyzer

**Files:** `arc/go/analyzer/call/call.go` (new),
`arc/go/analyzer/expression/expression.go`, `arc/go/analyzer/flow/flow.go`,
`arc/go/analyzer/flow/expression.go`, `arc/go/analyzer/function/function.go`,
`arc/go/analyzer/resolve.go`

`call.Analyze` is new and centralizes call validation. The bodies of
`validateFunctionCall` (parens form) and `validateFuncConfig` (brace form) are deleted;
both sites delegate to `call.Analyze`. `upstreamIsTrigger` is removed; flow.go consults
`fn.Trigger` directly.

For user-defined functions, the surface syntax is unchanged:

```arc
// With a wire-fed trigger
func my_tally{threshold f32} (sample f32) u8 { return sample > threshold }

// Without a trigger
func my_compute{a i32, b i32} i32 { return a + b }
```

The two declaration-walking passes in `function/function.go` collapse into one. The
brace-block params land in `Inputs` directly; the parens-block param (if present)
appends to `Inputs` AND becomes the symbol's `Trigger.Target`. There is no type-level
"config" or "input" distinction left; the two grammar blocks are sugar for the same
underlying shape.

`my_tally` registers as:

```go
Symbol{
    Name: "my_tally",
    Kind: KindFunction,
    Type: FunctionProperties{
        Inputs: Params{
            {Name: "threshold", Type: F32()},
            {Name: "sample",    Type: F32()},
        },
        Outputs: Params{{Name: ir.DefaultOutputParam, Type: U8()}},
    },
    Trigger: TriggerInput("sample"),
}
```

`my_compute` registers as:

```go
Symbol{
    Name: "my_compute",
    Kind: KindFunction,
    Type: FunctionProperties{
        Inputs: Params{
            {Name: "a", Type: I32()},
            {Name: "b", Type: I32()},
        },
        Outputs: Params{{Name: ir.DefaultOutputParam, Type: I32()}},
    },
    Trigger: TriggerOnly,
}
```

The scope-population passes (which add the params to the symbol scope for use inside the
function body) collapse the same way: one pass over `Inputs`, no per-param branching.

## 6.3 - Compiler and Runtime

**Files:** `arc/go/compiler/compiler.go`, `arc/go/runtime/node/state.go`

`slices.Concat(i.Config, i.Inputs)` → `i.Inputs`. Edge alignment iterates `Inputs`
directly. Param ordering is preserved across the migration, so WASM offsets stay stable.

## 6.4 - Graph and Text Compilation

**Files:** `arc/go/graph/analyze.go`, `arc/go/text/analyze.go`

Merge Config+Inputs bindParams in graph compilation. In text-to-IR, `Config:` field
assignments fold into `Inputs:`; the `upstreamIsTrigger` special case is deleted. IR
rendering (covered in 6.0) emits the unified `Inputs` list.

## 6.5 - Standard Library

**Files:**
`arc/go/stl/{time,stable,channel,authority,math,constant,op,series,selector,strings,stateful,errors,wasm}/*.go`,
`arc/go/stl/stl_test.go`

Every stdlib symbol gains an explicit `Trigger:` (per Section 4.1's audit table).
Symbols that previously declared `Config:` rename the field to `Inputs:`. Symbols that
previously declared both `Config:` and `Inputs:` (the mirror-trick `ExecBoth` symbols)
collapse to a single `Inputs:` list. Factory call sites that read
`cfg.Node.Config.ValueMap()` become `cfg.Node.Inputs.ValueMap()`.

`stl_test.go` deletes the "ExecBoth structural contract" test (the Inputs-mirrors-Config
invariant is no longer expressible) and adds:

- `AnalyzeArguments` hooks are only attached to `KindFunction` symbols.
- `Trigger.Target`, if non-empty, names an existing param on the symbol's Inputs list.

## 6.6 - LSP

**Files:** `arc/go/lsp/hover.go`, `arc/go/lsp/completion.go`

Hover renders the unified `Inputs` list. Completion proposes named-config keys from the
same list. The Trigger target is shown in hover where set.

## 6.7 - C++ Runtime

**Files:** `arc/cpp/runtime/state/state.cpp`, `arc/cpp/runtime/wasm/factory.h`,
`arc/cpp/stl/stable/stable.h`, `arc/cpp/stl/math/math.h`

Generated headers regenerate from the updated schemas. Hand-written sites that read
`cfg.node.config` switch to `cfg.node.inputs`. State alignment buffers iterate the
unified list.

## 6.8 - Service Layer

**Files:** `core/pkg/service/arc/status/status.go`

The visible win. `newSetSymbolType` declares its param list once and assigns it only to
`Inputs`. The two analyzer hooks collapse to one `AnalyzeArguments`. `status.set`
declares `Trigger: TriggerOnly`.

## 6.9 - Test Fixtures

**Files:** `arc/go/**/*_test.go`, `arc/cpp/**/*_test.cpp`,
`core/pkg/service/arc/**/*_test.go`, `client/ts/src/arc/**/*.spec.ts`

Mechanical fixture sweep. Every test that constructs a `types.FunctionProperties`,
`ir.Function`, or `ir.Node` literal with a `Config:` (or `config:`) field folds it into
`Inputs:` (or `inputs:`). Symbols that previously declared both `Config:` and `Inputs:`
deduplicate to a single list. New fixtures cover `call.Analyze` validation, Trigger
consult, the Trigger registration invariant, and the status.set hook collapse (Section
10).

# 7 - Implementation Plan

The implementation runs in two halves separated by a **review-and-evaluate gate**.
Phases 0-8 are all hand-written productive code; Phases 9-13 are mechanical (schema
sync, full codegen, integration). The gate exists so reviewers can read the hand-written
diff in isolation, before generated noise enters the picture.

**Phase 0: Snapshot.** Copy four current generated files into `migrations/vN/`:
`arc/go/types/{types,codec}.gen.go` and `arc/go/ir/{types,codec}.gen.go`. Pure
mechanical move, no logic change. Oracle regenerates these in Phase 10; the manual
snapshot exists so Phase 8 can reference the frozen pre-refactor shape while writing
hand-written migration logic before the gate.

**Phase 1: Minimal type-system seed.** The one unavoidable hand-edit to a `.gen.go`
file: remove `Config` from `FunctionProperties` and add the internal `AST` field to
`Param`. Phase 10 codegen overwrites this with identical content.

**Phase 2: Symbol, hooks, Trigger.** `TriggerBinding` type and helpers,
`AnalyzeArguments` hook, registration-time invariant for `Trigger.Target`.

**Phase 3: Analyzer collapse.** The main payoff. New `arc/go/analyzer/call/` package.
`validateFunctionCall` and `validateFuncConfig` bodies are deleted; their call sites
delegate to `call.Analyze`. flow.go upstream-handling collapses around `fn.Trigger`
consult.

**Phase 4: Downstream consumers.** Compiler, graph, text-to-IR, IR rendering, runtime
edge alignment.

**Phase 5: Stdlib, LSP, service layer.** Every stdlib symbol gains explicit `Trigger:`.
`Config:` declarations rename to `Inputs:`. Mirror-trick symbols deduplicate.
`status/status.go` collapses its dual-hook code.

**Phase 6: C++ runtime hand-written sites.** `cfg.node.config` → `cfg.node.inputs`.

**Phase 7: Test fixture updates.** Mechanical: `Config:` declarations across Go, C++,
and TS test files fold into `Inputs:`.

**Phase 8: Codec migration logic.** Hand-write `arc/go/ir/migrations/vN/migrate.go` with
`MigrateFunction` and `MigrateNode` overrides that merge the old `Config + Inputs` lists
into the unified `Inputs` (the auto-generated `migrate_auto.gen.go` produced in Phase 10
does only 1:1 field mapping and cannot infer the merge). Add round-trip tests in
`migrate_test.go`. The types-side migration is purely the codec snapshot from Phase 0;
no hand-written migrate.go is needed there. Both this file and Phase 1's seed reference
shapes that don't exist until Phase 10 codegen completes, so the branch will not compile
at the gate.

> ### 🛑 Review-and-Evaluate Gate 🛑
>
> **Commit, push, request review.** At this point the diff is entirely hand-written
> productive code. The reviewer can read the design without scrolling past generated
> noise. The branch does NOT compile: most references to the new shape are dangling
> until Phase 9 edits the schema and Phase 10 regenerates the generated bindings (and
> the migration auto-mapper that Phase 8's `migrate.go` wraps). That is intentional.
> Inspection targets: Phase 3 analyzer collapse, Phase 5 stdlib and service-layer
> cleanup, Phase 8 migration translation.

**Phase 9: Schema sync.** Edit `schemas/arc/types.oracle` and `schemas/arc/ir.oracle`.

**Phase 10: Full codegen.** Run the project's code generators end-to-end. Verification:
the Phase 1 hand-edit to `arc/go/types/types.gen.go` should be overwritten with an
_identical_ result. A non-zero diff is a bug to fix.

**Phase 11: Build, test, lint.** Everything compiles end-to-end for the first time. Go
build + test, C++ build + test, TS test, `golangci-lint run ./...`, and
`scripts/check_gofmt.sh`.

**Phase 12: Integration tests.** Full build sequence per
[docs/claude/testing.md](../../../docs/claude/testing.md), then
`cd integration && uv run tc arc`.

**Phase 13: Pluto reconciliation.** Grep `pluto/src/arc/` for any site directly
constructing `ir.Node` or `ir.Function` payloads with a `config:` literal. The surface
is expected to be narrow: most `config:` references in `pluto/src/arc/` are unrelated
wrapper objects (e.g. `{ arcKey }`). Update any hits to fold `config: {...}` into
`inputs: [...]`. Smoke test in console.

# 8 - Future Work

This RFC lands the structural foundation. Two features become substantially easier to
implement on top of it. A separate RFC will specify each.

## 8.0 - Trigger-as-Argument Syntax

> **Syntax in this section is illustrative.** Grammar fragments and code examples sketch
> how the feature would plug into `call.Analyze`; the follow-on RFC picks the actual
> surface syntax.

The follow-on RFC will add a call-site override that the foundation here enables. The
grammar adds a preflow binding table mirroring the existing right-side `routingTable`:

```
flowStatement: (argBindTable)? flowNode (flowOperator (routingTable | flowNode))+
argBindTable:  LBRACE argBindEntry (COMMA argBindEntry)* COMMA? RBRACE
argBindEntry:  IDENTIFIER COLON IDENTIFIER  // paramName: channelName
```

Single-binding shape, where one wire is pre-bound to a named param and the rest are
supplied at the call site:

```go
{message: msg_ch} -> status.set("My_status", message, "info")
```

The same syntax works against any function (stdlib or user-defined) and composes with
brace-form config on the right-hand side:

```go
func my_tally{threshold f32} (sample f32) u8 { return sample > threshold }

{sample: sensor_ch} -> my_tally{threshold=5.0}
```

Here the wire feeds `sample` (the trigger param declared in `()`) via the proposed
`argBindTable` shape on the left, while the existing brace-form on the right supplies
`threshold` (a brace-block param) using its `=` syntax. Both halves coexist because both
kinds of params live in the same `Inputs` list.

The analyzer builds synthetic `[]call.Argument` entries from the `argBindTable`, merges
with whatever the brace form supplies, and runs through `call.Analyze` as normal. The
call-site bindings override the symbol's `Trigger` default: `status.set`'s `TriggerOnly`
does not prevent a call site from explicitly binding `message` to a wire.

Because the type system carries no "this param cannot be wire-bound" flag, the override
works against any param. There is no allowlist to maintain.

### 8.0.0 - Scaling to N bindings

The mechanism extends to multiple bindings at one call site without structural change:

```go
{message: msg_ch, variant: variant_ch} -> status.set("My_status", message, variant)
```

The analyzer loop is identical whether one binding or five: parse the table, push each
`LocalName` into a transient scope visible only to the call's args, type-check arg
expressions against each bound channel's value type, emit one edge per used binding. The
cost is linear in the number of bindings; nothing new is required beyond what handles
the one-binding case.

### 8.0.1 - Firing semantics with multiple bindings

When multiple wires are bound at one site, when should the node fire? Three credible
policies, all of which the foundation here accommodates without structural change:

| Model              | Behavior                                                                                         | Trade-off                                         |
| ------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------- |
| **Any**            | Fire when any bound wire updates. Other inputs read their last-known value.                      | Simplest. Mixed-freshness args possible.          |
| **All** (zip/sync) | Fire only once every bound wire has produced a new value since last fire.                        | Frame-aligned. Can stall on slow inputs.          |
| **User-specified** | User declares the firing predicate. Likely reuses `and`/`or` so users don't learn new operators. | Most flexible, most complex to parse and compile. |

No stance is taken here. The trigger-as-argument RFC will pick one.

## 8.1 - Optional Parameters

Today every Arc function call must supply every declared param. The `Optional` field on
`Param` (Section 4.0) gives the type system a place to mark a param as optional, but
this RFC does not implement the call-site or runtime behavior that flag would control. A
follow-on RFC will specify the semantics: likely a mix of compile-time default
substitution (for the `Value` field) and runtime preserve-on-omit dispatch (handle-0
sentinel for host functions), along the lines RFC 0037 §5.0.0 sketches.

The unification here makes that follow-on substantially smaller. Under today's split,
optional-arg handling has to be wired through two analyzer paths (`validateFunctionCall`
and `validateFuncConfig`) and two hook surfaces, and the preserve-on-omit sentinel has
to be threaded through both the parens-form compiler emit and the brace-form flow-node
factory. Each is a separate code site with its own argument-walking logic. After
unification, the implementation reduces to a localized change inside `call.Analyze`
(treat missing args for `Optional: true` params as a non-error and emit the default /
sentinel) plus one update to the shared host-function and node-factory plumbing. One
code site instead of two, one analyzer pass instead of two.

This is a concrete second beneficiary of the refactor, separate from
trigger-as-argument: the same structural simplification that opens the door to
wire-bound args also opens the door to optional args with default values.

# 9 - Change Footprint

Order-of-magnitude estimates per area. Productive code (everything except codec
migration scaffolding) is net-negative, matching the intuition for a tech-debt-paydown
refactor. The codec migration scaffolding is one-time boilerplate (snapshot of
pre-refactor types + a mechanical `Migrate*`) and tips the gross total positive.

| Area                                                                   |     Added |   Removed |      Net |
| ---------------------------------------------------------------------- | --------: | --------: | -------: |
| Schema + generated bindings (Go, C++, TS)                              |      ~140 |      ~175 |      -35 |
| Type-system hand-written (`type.go`, `fresh.go`)                       |       ~15 |       ~20 |       -5 |
| Symbol & hooks (`symbol.go`, `hooks.go`, `scope.go`)                   |       ~50 |       ~43 |       +7 |
| **Analyzer core** (`call/`, `expression.go`, `flow.go`, `function.go`) |  **~400** |  **~620** | **-220** |
| Per-stdlib `Trigger:` assignments (~30 symbols × 1 line)               |       ~30 |         0 |      +30 |
| Compiler & runtime (`compiler.go`, `state.go`, C++ runtime)            |       ~40 |       ~50 |      -10 |
| Text → IR (`text/analyze.go`)                                          |       ~25 |       ~60 |      -35 |
| Graph compilation (`graph/analyze.go`)                                 |       ~20 |       ~50 |      -30 |
| LSP (`hover.go`, `completion.go`)                                      |       ~15 |       ~25 |      -10 |
| Stdlib (Go: `time`, `stable`, `channel`, `op`, `math`, etc.)           |      ~120 |      ~140 |      -20 |
| **Service-layer (`status/status.go`)**                                 |   **~25** |  **~110** |  **-85** |
| Test fixture updates                                                   |      ~270 |      ~310 |      -40 |
| Pluto frontend                                                         |       ~15 |       ~15 |        0 |
| **Subtotal (productive code)**                                         | **~1165** | **~1618** | **-453** |
| Codec migration vN (one-time scaffolding)                              |      ~550 |         0 |     +550 |
| **Grand total**                                                        | **~1715** | **~1618** |  **+97** |

Productive code lands at **-453 net lines**. The analyzer alone removes ~220 net lines;
the service-layer status code shrinks by ~85 net lines; stdlib registrations contract
because Config-to-Inputs renaming is a wash and the dual-list ExecBoth symbols dedupe.
The codec migration scaffolding is the only thing keeping the gross total above zero.

# 10 - Test Coverage

Most coverage moves rather than appearing fresh. The mirroring contract test in
`stl_test.go` is deleted; in its place sit the two new structural invariants from
Section 6.5. End-to-end coverage of "ExecBoth callable in both forms" moves from
`stl_test.go` (where it was a symbol-table invariant) into `expression_test.go`,
`flow_test.go`, and `flow_upstream_trigger_test.go` (where it is a call-site property).
Fixture updates across `arc/go/`, `core/pkg/service/arc/`, and `arc/cpp/` fold `Config:`
declarations into `Inputs:`.

New tests required by this change:

- **Migration round-trip**: pre-vN IR JSON / msgpack / proto deserialize, migrate, and
  produce IR Type-shape identical to a fresh declaration in the new shape.
- **`call.Analyze` shared validation**: argument count, type, name resolution, default
  substitution, unknown-name detection, duplicate-name detection, verified once against
  both parens and brace surface forms. Today's coverage duplicates each scenario across
  two test files.
- **Trigger consult**: a flow call to a `TriggerOnly` symbol does not type-check the
  upstream; a flow call to a `TriggerInput("x")` symbol type-checks the upstream against
  the `x` param's type and rejects type mismatches.
- **Trigger invariant**: the `stl_test.go` symbol walker fails when a registered symbol
  declares `Trigger.Target = "missing_param"`.
- **Surface-syntax preservation**: the smoke program in Section 11 parses, analyzes,
  compiles, and runs identically to the pre-refactor branch.
- **`status.set` collapse**: the literal-validation `AnalyzeArguments` hook fires
  correctly for both `status.set(..., "errpr")` and `status.set{variant="errpr"}`,
  emitting one diagnostic per call site (today's hooks would each fire once).

# 11 - Verification

The end-to-end correctness check is that existing Arc programs are unaffected. Two smoke
programs cover the two trigger flavors.

**TriggerOnly path** (`status.set`, exercising the mirror-trick collapse plus the
dual-hook collapse):

```go
func my_example {} () {
    log_a = status.set("Same_Name", "msg", "success")
}
time.interval{1s} -> my_example{}
time.interval{1s} -> status.set("my_status", "msg", "info") -> log_b
```

**TriggerInput path** exercises the wire-as-input branch that today consults
`upstreamIsTrigger=false` and post-refactor consults `Trigger.Target != ""`:

```go
sensor_ch -> math.avg{count=10} -> output_ch
sensor_ch -> op.gt{b=5.0} -> alert_ch
```

(Confirm at implementation time that the grammar permits both `op.gt(5.0)` parens-form
calls and `op.gt{b=5.0}` brace-form calls inside a flow chain. If the parens form is not
currently legal in flow position, the smoke uses brace form for both.)

Both must parse, analyze, compile, and run identically to the pre-refactor branch. The
IR JSON for each source should be structurally the same.

LSP smoke: hover over `status.set` in the console editor renders the unified Inputs list
correctly. Completion proposes named-config keys for the brace form exactly as it does
today.

# 12 - Risks and Open Questions

**Pluto direct IR construction.** Some tile components may construct `ir.Node` with
`config:` literals at the API layer. Needs grep during Phase 13. The migration is
mechanical (`config: {...}` folds into `inputs: [...]`), but the sites have to be found
first.

**WASM ABI ordering.** WASM offsets in
[arc/go/stl/wasm/wasm.go](../../../arc/go/stl/wasm/wasm.go) depend on the order the
compiler concat already produces (Config first, then Inputs). The migration appends in
that same order, so offsets stay stable. Phase 8's migration tests verify byte-equal IR
for the same source.

**Hook ctx type assertion.** Today's hooks switch on the concrete `acontext.Context[T]`
type. The unified `AnalyzeArguments` could keep `ctx any` and let each hook type-assert
what it needs (smaller diff), or introduce a `DiagBearer` interface that exposes just
the diagnostic surface hooks actually use (typed and discoverable, larger diff). Open
question; no decision taken here.

**Codec version bump.** This is a wire-format break. The codec version increments as
part of the major release this refactor ships with, skipping the next sequential number.
The exact target version is fixed once the release is named; this RFC uses `vN` as a
placeholder throughout.
