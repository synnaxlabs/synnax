# 40 Arc function-call unification

- **Author**: Nico Alba
- **Date**: 2026-05-13
- **Related**: [RFC 0030 - Arc Module System](./0030-arc-modules.md),
  [RFC 0031 - Arc Scheduler Semantics](./0031-arc-scheduler-semantics.md),
  [RFC 0037 - Arc Status Module Updates](./0037-arc-status-updates.md)

## 0 Summary

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

The same change demotes `Exec` (the `ExecWASM` / `ExecFlow` / `ExecBoth` enum) from a
structural property to a **superficial gate**. Today `Exec` co-drives composition: it
selects which parameter array a symbol reads and feeds the trigger heuristic. After this
RFC a symbol's composition is just its `Inputs` list plus its `Trigger`. `Exec` survives
only as a thin permission flag describing where a symbol may be called, decoupled from
how it is built; every symbol is treated as usable in any context by default, and the
superficial gate is consulted in two places only (Section 4.1).

User-facing Arc syntax does not change. `name(...)`, `name{...}`, and
`wire -> name{...}` continue to parse, analyze, compile, and run exactly as they do
today. The change is structural: one parameter list replaces two, one analyzer path
replaces two, one hook replaces two, and one declaration replaces a runtime heuristic.

The refactor establishes the foundation for the future trigger-as-argument feature
(`{message: incoming_ch} -> status.set("X", message)`), which a separate RFC will
specify. That feature becomes a call-site override of the symbol-level `Trigger` default
and works against any param, since there is no longer any param-property distinction at
the type level for it to interact with.

## 1 Goals

**Eliminate the `Config`/`Inputs` dichotomy at the type level.** A function has one
parameter list. There is no per-param flag distinguishing "configuration" from "runtime
input." Whatever conceptual distinction the original split was reaching for is
recoverable from each symbol's own semantics; the type system does not need to encode
it.

**Replace the `ExecBoth` heuristic with an explicit declaration.** Today the flow
analyzer guesses whether an upstream edge is a value source or a pure activation pulse
by inspecting `Exec` and `len(Config)`. The guess happens to be correct for every stdlib
symbol today because of the mirror trick, but the rule is implicit. Replace it with a
`Trigger` field on `Symbol` that names the edge-fed param (or declares `TriggerOnly`).

**Demote `Exec` to a superficial gate.** A symbol's composition is its `Inputs` list and
its `Trigger`. `Exec` no longer selects a parameter array or participates in trigger
detection; it is reduced to a thin permission flag, consulted only where the analyzer
and the expression compiler decide whether a symbol may be called in the context it
appears in (Section 4.1). This RFC strips `Exec` of structural meaning and keeps it only
as that superficial gate.

**Collapse the two analyzer call-validation paths.** `validateFunctionCall` (parens
form) and `validateFuncConfig` (brace form) silently agree only because `Inputs` is
mirrored to `Config` in every `ExecBoth` symbol. Replace both with a single
`call.Analyze` routine.

**Collapse the two analyzer hook surfaces.** A symbol that wants to validate its
arguments (e.g. `status.set` constraining the `variant` literal; see
[RFC 0037 §5.0.1](./0037-arc-status-updates.md#501---literal-value-constraints))
registers `AnalyzeCall` for the parens form AND `AnalyzeFlowConfig` for the brace form.
The two hooks walk different AST shapes to find the same argument and run the same
literal check. Replace both with a single `AnalyzeArguments` hook that receives a
unified `[]Argument` view.

## 2 Non-goals

- **User-facing Arc syntax.** `name(...)`, `name{...}`, and `wire -> name{...}` parse,
  analyze, and run identically; the grammar is untouched. This is a deliberate scope
  limit, not a punt: the goal here is parity at the type level, and holding the surface
  fixed keeps the change non-breaking for customers and bounds the test churn to the
  refactor itself rather than a grammar migration layered on top of an already large
  diff. Which bracket should mean what in the long term is the natural next decision,
  explored but deliberately not taken in
  [Section 8.2](#82-higher-order-functions-and-the-syntax-question).
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
- **Optional parameter semantics.** Arc already has partial default-value optionality
  (`name type = literal`; a defaulted param may be omitted, with the "required cannot
  follow optional" ordering rule). This RFC preserves that behavior unchanged, adds no
  `Optional` field, and changes nothing about how defaults are substituted or
  dispatched. Completing optional semantics (an explicit field, reliable call-site
  default substitution, preserve-on-omit dispatch) is future work that the unification
  makes easier; see [Section 8.1](#81-optional-parameters).

## 3 The problem

### 3.0 How the split became debt

Early Arc symbols were single-context: a flow node was flow-only (`time.interval`,
`time.wait`), a function was func-only (early `math` and `op` symbols). In that world
the `Config`/`Inputs` split read as load-bearing: each field corresponded to the context
the symbol participated in, and the timing/binding semantics of its params were
unambiguous within that context.

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

Once parity made symbols cross-context, the `Config`/`Inputs` split stopped being
load-bearing. In the pre-parity world it carried real semantic weight: a symbol could
appear in one context and one only, and the field it landed in named that context. Once
a symbol could appear in both, the field could no longer name the context, and the
mirror trick is the proof: copying the same data into both fields produces correct
symbols, which it could not if the two fields still encoded distinct content.

What the split was actually tracking, for the symbols where it still meant anything, is
**which param the upstream wire feeds**. A mixed-shape flow symbol like `channel.write`
puts its configured `channel` in one array and its edge-fed `input` in the other, but
those are not two kinds of param. They are one param list with one entry singled out as
the wire target. That single fact is the only thing worth preserving, and it now lives
explicitly on `Symbol.Trigger`. The reframing is the whole RFC in one line: a symbol has
an `Inputs` list, period, plus a `Trigger` that names which input (if any) a wire sets.
Everything the `Config`/`Inputs` split and the `ExecBoth` mirror were reaching for falls
out of those two pieces.

This RFC finishes what `ExecBoth` started. The mirror trick was a tactical fix that
exposed the strategic problem; the strategic fix is to collapse the field whose
redundancy the workaround already demonstrated, before more symbols accrue the same
scaffolding.

`Exec` is demoted in the same motion. With composition reduced to `Inputs` plus
`Trigger`, the `ExecWASM` / `ExecFlow` / `ExecBoth` enum no longer selects a param array
or feeds the trigger decision. It is kept only as a superficial gate, a thin permission
flag the analyzer consults to decide where a symbol may be called (Section 4.1); every
symbol is otherwise treated as usable in any context.

### 3.1 The mirror trick

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

### 3.2 The trigger heuristic

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
in both contexts too, but on a separate path: in flow form, `f"x: {ch}" -> log`
registers a synthetic `KindFunction` (Exec is the zero value, _not_ `ExecBoth`) whose
interpolated channel reads happen via host calls and whose activation is stratum
membership rather than a trigger edge
([arc/go/analyzer/flow/expression.go](../../../arc/go/analyzer/flow/expression.go)). So
the rule that decides "is this wire a trigger" is now expressed two incompatible ways
across the codebase, and the `AST` half of it is entangled with cross-context mechanisms
that have nothing to do with `ExecBoth`.

It happens to produce the right answer for every symbol today because of the mirror
trick and the `AST` accident. But none of the three sites ever says "this function takes
its wire as a trigger, not a value." The truth is implicit in structural properties
(`Exec` mode, `Config` length, AST presence), none of which is the property being
tested. A future function whose intent diverges from these patterns has no way to
express that today.

### 3.3 The two analyzer paths

`validateFunctionCall` in
[arc/go/analyzer/expression/expression.go](../../../arc/go/analyzer/expression/expression.go)
and `validateFuncConfig` in
[arc/go/analyzer/flow/flow.go](../../../arc/go/analyzer/flow/flow.go) implement two
versions of the same validation. Reduced to essentials, the forms look like this:

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
agree only because `Inputs` mirrors `Config` in every `ExecBoth` symbol. The divergent
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

### 3.4 The two-hook duplication

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

## 4 Proposed design

### 4.0 Unified inputs

`arc/go/types/types.gen.go` (regenerated from `schemas/arc/types.oracle`):

```go
type FunctionProperties struct {
    Inputs  Params `json:"inputs"  msgpack:"inputs"`
    Outputs Params `json:"outputs" msgpack:"outputs"`
    // Config field removed. No replacement.
}

type Param struct {
    Name  string `json:"name"  msgpack:"name"`
    Type  Type   `json:"type"  msgpack:"type"`
    Value any    `json:"value" msgpack:"value"`
}
```

No `BindMode` field, no `Captures()` / `Arguments()` helpers, no per-param tag. Whatever
"config" vs "input" was trying to encode at the type level is gone. A function has
params; each param has a name, a type, and an optional default value. This RFC adds no
new optionality mechanism: the existing default-value optionality is preserved and no
`Optional` field is introduced (see [Section 2](#2-non-goals)).

Param ordering is preserved as the user declared it. The compiler already produces a
deterministic order at the WASM ABI layer (`slices.Concat(Config, Inputs)`); the
migration appends the old `Config` list before the old `Inputs` list to preserve that
order byte-for-byte (Section 5).

### 4.1 Explicit trigger binding

`arc/go/symbol/symbol.go`:

```go
type TriggerBinding struct {
    // Target names the param the upstream wire binds its value to in flow
    // context. Empty (the zero value, TriggerOnly) means the wire activates the
    // symbol without binding a value.
    Target string
}

// TriggerOnly is the zero value: an upstream wire activates the symbol without
// binding a value to any param.
var TriggerOnly = TriggerBinding{}

// TriggerInput declares that an upstream wire binds its value to the named param.
func TriggerInput(name string) TriggerBinding { return TriggerBinding{Target: name} }

type Symbol struct {
    // ... existing fields ...
    Trigger TriggerBinding
}
```

The two states map directly to today's behavior: a symbol a wire activates without
feeding a value is `TriggerOnly` (the zero value); a symbol whose wire sets a named
param is `TriggerInput`. Whether a symbol may appear in flow context at all is the
superficial gate's concern, not `Trigger`'s (see "Exec as a superficial gate" below).
`Trigger` describes only what the wire does with its value once the symbol is in flow.

Every symbol whose wire feeds a param gains an explicit `TriggerInput:` line; everything
else keeps the zero value (`TriggerOnly`). The assignment is a mechanical audit driven
by today's behavior. Symbols are grouped by where they're declared, since the
implementer touches each set in a different file sweep.

**Arc stdlib** (`arc/go/stl/`):

| Symbol                                                            | Today (`Exec`, `Config`, `Inputs`)                | `Trigger`                     |
| ----------------------------------------------------------------- | ------------------------------------------------- | ----------------------------- |
| `time.interval`                                                   | `ExecFlow`, `[period]`, none                      | `TriggerOnly`                 |
| `time.wait`                                                       | `ExecFlow`, `[duration]`, none                    | `TriggerOnly`                 |
| `time.now`                                                        | `ExecBoth`, none, none                            | `TriggerOnly`                 |
| `channel.on`                                                      | `ExecFlow`, `[channel]`, none                     | `TriggerOnly`                 |
| `channel.write`                                                   | `ExecFlow`, `[channel]`, `[input]`                | `TriggerInput("input")`       |
| `constant.constant`                                               | `ExecFlow`, `[value]`, none                       | `TriggerOnly`                 |
| `stable.for`                                                      | `ExecFlow`, `[duration]`, `[input]`               | `TriggerInput("input")`       |
| `math.avg` / `min` / `max`                                        | `ExecFlow`, `[duration, count]`, `[input, reset]` | `TriggerInput("input")`       |
| `math.derivative`                                                 | `ExecFlow`, none, `[input]`                       | `TriggerInput("input")`       |
| `op.{ge,gt,le,lt,eq,ne,and,or}`                                   | `ExecFlow`, none, `[a, b]`                        | `TriggerInput("a")`           |
| `op.not`                                                          | `ExecFlow`, none, `[input]`                       | `TriggerInput("input")`       |
| `selector.select`                                                 | `ExecFlow`, none, `[output]` (see note)           | `TriggerInput("output")`      |
| `authority.set`                                                   | `ExecFlow`, `[value, channel]`, `[output]`        | `TriggerInput("output")`      |
| `math.pow`                                                        | `ExecWASM`, none, varies                          | `TriggerOnly` (not consulted) |
| `series.len`, `series.{element_*, series_*}` (internal host syms) | `ExecWASM`, none, varies                          | `TriggerOnly` (not consulted) |
| `string.{from_literal, concat, equal, ...}`                       | `ExecWASM`, none, varies                          | `TriggerOnly` (not consulted) |
| `state.{load, store, load_series, store_series, ...}`             | `ExecWASM`, none, varies                          | `TriggerOnly` (not consulted) |
| `error.panic`                                                     | `ExecWASM`, none, `[ptr, len]`                    | `TriggerOnly` (not consulted) |

**Service layer** (`core/pkg/service/arc/`):

| Symbol       | Today (`Exec`, `Config`, `Inputs`)                     | `Trigger`                   |
| ------------ | ------------------------------------------------------ | --------------------------- |
| `status.set` | `ExecBoth`, mirrored `[key_or_name, message, variant]` | `TriggerOnly` (post-dedupe) |

**User-defined functions:** every user-defined function is now `ExecBoth`
([`arc/go/analyzer/function/function.go`](../../../arc/go/analyzer/function/function.go)),
with `Config` holding the brace-block params and `Inputs` the parens-block params. Under
this RFC they are auto-assigned `TriggerInput(<first parens-block param>)` if a parens
block is present, else `TriggerOnly`.

The "mixed-shape" rows (`channel.write`, `stable.for`, `math.*`, `authority.set`) are
the ones that today have **both** `Config` and `Inputs` populated with **different**
params. The unification collapses them to a single `Inputs:` list with old `Config`
items first (matching `slices.Concat(Config, Inputs)` and the codec migration's append
order); the trigger param is the first old-`Inputs` param. Mirror-trick rows
(`status.*`) deduplicate; non-mirror rows concat.

Pure-computation symbols (`math.pow`, `series.*`, `string.*`, `state.*`, `error.panic`)
leave `Trigger` at the zero value (`TriggerOnly`); the superficial gate keeps them out
of flow context, so `Trigger` is never consulted for them and the value is immaterial.
The registration invariant accommodates an empty `Target`.

**Note on `selector.select`.** Its sole input is named with the `DefaultOutputParam`
constant (`"output"`) rather than something like `"condition"`, so the `Trigger` target
must be `TriggerInput("output")` to match the param the code actually declares. Renaming
the param to `"condition"` is a reasonable cleanup to fold into the stdlib sweep, but it
is an IR-visible change; absent that rename, the audit uses `"output"`.

A unit test in `stl_test.go` guards the field: `Trigger.Target`, if non-empty, must name
an existing param on the same symbol. This catches the rename/delete failure mode at
`go test` time, not at the first flow-call site.

**Why symbol-level, not param-level.** The semantic "which param does the wire feed by
default" is a function-level decision. It varies per function regardless of which params
are present. Encoding it on `Symbol` puts it where it logically lives.

**Exec as a superficial gate.** This RFC strips `Exec` of structural meaning. It no
longer selects a parameter array (there is one `Inputs` list) and no longer feeds the
trigger decision (that is `Trigger`). What remains is a thin permission flag: a symbol
is usable in any context by default, and `Exec` is the developer's override naming the
contexts it is restricted to. The gate is one check,
`symbol.Exec.Compatible(contextExec)`: an unrestricted symbol passes anywhere, a
restricted one only where its flag and the calling context agree. It runs in two places:

- [arc/go/analyzer/expression/expression.go](../../../arc/go/analyzer/expression/expression.go):
  the analyzer-time check.
- [arc/go/compiler/expression/compiler.go](../../../arc/go/compiler/expression/compiler.go):
  the same check at compile time.

Both generalize today's one-directional `scope.Exec == symbol.ExecFlow` test to the
symmetric `Compatible` form: a func-only symbol (`ExecWASM`) is gated out of flow and a
flow-only symbol out of func by the same rule. LSP completion already uses this filter
in [arc/go/lsp/completion.go](../../../arc/go/lsp/completion.go) and is unchanged. The
gate forbids the exceptions the developer declares; it does not define what a symbol is.

### 4.2 Unified call analyzer

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

### 4.3 Unified hook

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

### 4.4 Flow analyzer collapse

The three-branch upstream-handling block at
[arc/go/analyzer/flow/flow.go:102-213](../../../arc/go/analyzer/flow/flow.go)
(`prevIDNode` / `prevExpr` / `prevFuncNode`) collapses around a direct consult of the
symbol's `Trigger`:

```go
target := fn.Trigger.Target
switch {
case target == "":
    // TriggerOnly: wire is pure activation; do not type-check the upstream value.
case suppliedAtCallSite[target]:
    diag("param '%s' is bound by both call-site args and upstream wire", target)
default:
    // TriggerInput: bind the upstream value to the named param.
    targetParam, ok := fn.Type.Inputs.Get(target)
    if !ok {
        diag("symbol '%s' declares Trigger target '%s' but has no such param", name, target)
    } else {
        atypes.Check(upstreamType, targetParam.Type, ...)
        // emit edge: upstream → targetParam
    }
}
```

The `upstreamIsTrigger` heuristic is removed. The zero value is `TriggerOnly` (pure
activation); a flow symbol whose wire feeds a param declares `TriggerInput` explicitly.

## 5 Codec migration

The schema change is a wire-format break. Persisted Arc programs (IR JSON, IR msgpack,
proto) deserialize through a codec migration that translates the pre-refactor shape into
the unified shape.

**Snapshot.** Bumping the codec version makes Oracle freeze the pre-refactor
`arc/go/types/types.gen.go` and `arc/go/ir/types.gen.go` under the respective
`migrations/vN/` directories. The snapshot is generated
(`Code generated by oracle. DO NOT EDIT.`), not hand-copied.

**Translation.** `MigrateFunction` and `MigrateNode` walk the vN snapshot:

- Concatenate the old `Config` list and the old `Inputs` list into the unified `Inputs`
  list, in that order. This preserves the WASM ABI ordering the compiler already
  produces today.
- For stdlib `ExecBoth` symbols where `Config == Inputs` (the mirror trick), the
  duplicate is detected and dedupes to a single list. The mirror was a workaround for
  the `Config`/`Inputs` split; with the split gone, the workaround is no longer needed.

**Verification.** `migrations/vN/migrate_test.go` round-trips a pre-vN snapshot through
the migrator and asserts the resulting IR Type-shape matches what a fresh declaration in
the new shape produces. Persisted Arc programs from the pre-refactor codec version
deserialize identically.

**Scope: Go is assumed to be the deserialization boundary for persisted IR.** The
existing migrations all live on the Go side
([core/pkg/service/arc/migrations/v54/migrate_test.go](../../../core/pkg/service/arc/migrations/v54/migrate_test.go),
with no TS or C++ counterpart), which suggests the Go server deserializes persisted Arc
programs and hands post-migration data to the C++ and TS bindings. If that holds, those
bindings never see pre-vN bytes and no hand-written C++ or TS migration is required.
This assumption should be confirmed with whoever owns the persistence path before
relying on it; if any non-Go runtime loads pre-vN bytes directly, it needs its own
migration.

## 6 Changes by layer

### 6.0 Type system and IR

**Files:** `arc/go/types/types.gen.go`, `arc/go/types/type.go`, `arc/go/types/fresh.go`,
`arc/go/types/migrations/vN/`, `arc/go/ir/types.gen.go`, `arc/go/ir/function.go`,
`arc/go/ir/node.go`, `arc/go/ir/migrations/vN/`

`types.gen.go` (regenerated): drop `Config` from `FunctionProperties`. `type.go`: update
`Equal` and `paramsEqual` to compare against the single `Inputs` list. `fresh.go`: drop
`Config` freshen. `ir/types.gen.go` (regenerated): drop `Config` from `Function` and
`Node`. `ir/function.go` and `ir/node.go`: drop the `Config` rendering branch from
`Type()` and string output. Both `migrations/vN/` directories snapshot the pre-refactor
shape; the IR side additionally gains a hand-written `migrate.go` (Section 5).

### 6.1 Symbol table

**Files:** `arc/go/symbol/symbol.go`, `arc/go/symbol/hooks/hooks.go`,
`arc/go/symbol/scope.go`

Replace `AnalyzeCall` + `AnalyzeFlowConfig` with `AnalyzeArguments`. Add
`Trigger TriggerBinding` field and the `TriggerBinding` / `TriggerOnly` /
`TriggerInput(...)` types and helpers. `ResolveConfigChannel` reads `Inputs.Get(...)`.

### 6.2 Analyzer

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
// With an edge-fed trigger
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

### 6.3 Compiler and runtime

**Files:** `arc/go/compiler/compiler.go`, `arc/go/runtime/node/state.go`

`slices.Concat(i.Config, i.Inputs)` → `i.Inputs`. Edge alignment iterates `Inputs`
directly. Param ordering is preserved across the migration, so WASM offsets stay stable.

### 6.4 Graph and text compilation

**Files:** `arc/go/graph/analyze.go`, `arc/go/text/analyze.go`

Merge `Config`+`Inputs` `bindParams` in graph compilation. In text-to-IR, `Config:`
field assignments fold into `Inputs:`; the `upstreamIsTrigger` special case is deleted.
IR rendering (covered in 6.0) emits the unified `Inputs` list.

### 6.5 Standard library

**Files:**
`arc/go/stl/{time,stable,channel,authority,math,constant,op,series,selector,strings,stateful,errors,wasm}/*.go`,
`arc/go/stl/stl_test.go`

Every stdlib symbol gains an explicit `Trigger:` (per Section 4.1's audit table).
Symbols that previously declared `Config:` rename the field to `Inputs:`. Symbols that
previously declared both `Config:` and `Inputs:` (the mirror-trick `ExecBoth` symbols)
collapse to a single `Inputs:` list. Factory call sites that read
`cfg.Node.Config.ValueMap()` become `cfg.Node.Inputs.ValueMap()`.

`stl_test.go` deletes the "ExecBoth structural contract" test (the
`Inputs`-mirrors-`Config` invariant is no longer expressible) and adds:

- `AnalyzeArguments` hooks are only attached to `KindFunction` symbols.
- `Trigger.Target`, if non-empty, names an existing param on the symbol's `Inputs` list.

### 6.6 LSP

**Files:** `arc/go/lsp/hover.go`, `arc/go/lsp/completion.go`

Hover renders the unified `Inputs` list. Completion proposes named-config keys from the
same list. The `Trigger` target is shown in hover where set.

### 6.7 C++ runtime

**Files:** `arc/cpp/runtime/state/state.cpp`, `arc/cpp/runtime/wasm/factory.h`,
`arc/cpp/stl/stable/stable.h`, `arc/cpp/stl/math/math.h`

Generated headers regenerate from the updated schemas. Hand-written sites that read
`cfg.node.config` switch to `cfg.node.inputs`. State alignment buffers iterate the
unified list.

### 6.8 Service layer

**Files:** `core/pkg/service/arc/status/status.go`

The visible win. `newSetSymbolType` declares its param list once and assigns it only to
`Inputs`. The two analyzer hooks collapse to one `AnalyzeArguments`. `status.set`
declares `Trigger: TriggerOnly`.

### 6.9 Test fixtures

**Files:** `arc/go/**/*_test.go`, `arc/cpp/**/*_test.cpp`,
`core/pkg/service/arc/**/*_test.go`, `client/ts/src/arc/**/*.spec.ts`

Mechanical fixture sweep. Every test that constructs a `types.FunctionProperties`,
`ir.Function`, or `ir.Node` literal with a `Config:` (or `config:`) field folds it into
`Inputs:` (or `inputs:`). Symbols that previously declared both `Config:` and `Inputs:`
deduplicate to a single list. New fixtures cover `call.Analyze` validation, `Trigger`
consult, the `Trigger` registration invariant, and the `status.set` hook collapse
(Section 10).

## 7 Implementation plan

The change ships as a **stack of PRs, one per logical step**, each branched off the
previous and reviewed on its own diff. Schema sync and full codegen land **first**, in
their own PR, so the large generated diff is isolated and every later PR is a pure
hand-written change against the final generated shape. Bumping the codec version makes
Oracle freeze the pre-refactor snapshot under `migrations/vN/` as part of that codegen;
there is no separate hand-copied snapshot step, and no generated file is hand-edited.

Because codegen drops `Config` up front, the tree does not build until the last reader
of `.Config` is converted near the end of the stack. Intermediate PRs are reviewed on
their diff, not on a green CI run; the change merges to `rc` only once the full stack is
green, so `rc` never observes the broken intermediate state. The test sweep is
consolidated into a single late PR rather than spread inline.

A **design-evaluation gate** follows the analyzer collapse (`call.Analyze` plus the flow
`Trigger` consult), which is the core of the change. The reviewer decides whether the
unified `Inputs` + `Trigger` shape holds up against the hardest stdlib symbols before
the mechanical spread into stdlib, C++, tests, and migration. The service-layer shrink
and the migration round-trip are later confirmations, not prerequisites for that
decision.

The hand-written codec migration (`arc/go/ir/migrations/vN/migrate.go`) wraps the
generated `migrate_auto.gen.go`, merging the old `Config + Inputs` lists into the
unified `Inputs` (old `Config` first, to preserve WASM ABI ordering) and deduplicating
the mirror-trick symbols; the auto-mapper only does 1:1 field mapping and cannot infer
the merge. The types-side migration needs no hand-written `migrate.go` (types are
referenced by IR, not persisted independently). The final PR builds and tests
end-to-end, runs the integration suite, and reconciles any `pluto/src/arc/` sites that
construct `ir.Node` / `ir.Function` payloads with a `config:` literal.

## 8 Future work

This RFC lands the structural foundation. Two features become substantially easier to
implement on top of it, and a separate RFC will specify each (Sections 8.0 and 8.1).
Section 8.2 addresses where higher-order functions fit, and Section 8.3 records the
long-term topics this refactor deliberately leaves untouched, and why.

### 8.0 `Trigger`-as-argument syntax

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

#### 8.0.0 Scaling to N bindings

The mechanism extends to multiple bindings at one call site without structural change:

```go
{message: msg_ch, variant: variant_ch} -> status.set("My_status", message, variant)
```

The analyzer loop is identical whether one binding or five: parse the table, push each
`LocalName` into a transient scope visible only to the call's args, type-check arg
expressions against each bound channel's value type, emit one edge per used binding. The
cost is linear in the number of bindings; nothing new is required beyond what handles
the one-binding case.

#### 8.0.1 Firing semantics with multiple bindings

When multiple wires are bound at one site, when should the node fire? Three credible
policies, all of which the foundation here accommodates without structural change:

| Model              | Behavior                                                                                         | Trade-off                                         |
| ------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------- |
| **Any**            | Fire when any bound wire updates. Other inputs read their last-known value.                      | Simplest. Mixed-freshness args possible.          |
| **All** (zip/sync) | Fire only once every bound wire has produced a new value since last fire.                        | Frame-aligned. Can stall on slow inputs.          |
| **User-specified** | User declares the firing predicate. Likely reuses `and`/`or` so users don't learn new operators. | Most flexible, most complex to parse and compile. |

No stance is taken here. The trigger-as-argument RFC will pick one.

### 8.1 Optional parameters

Arc's default-value optionality today is partial: a param with a `= literal` default may
be omitted, but the mechanism is incomplete (e.g. `status.set`'s empty-string defaults
exist only to make the brace/flow form parse, and fail their own validation when relied
on). A follow-on RFC will complete it. **The following is purely hypothetical**,
included only to show the unification leaves room for it. That RFC would add an
`Optional bool` field to `Param` and specify the semantics behind it: likely a mix of
compile-time default substitution (reading the existing `Value` field) and runtime
preserve-on-omit dispatch (a handle-Nil sentinel for host functions), along the lines
RFC 0037 §5.0.0 sketches. None of that field or behavior exists in this RFC.

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

### 8.2 Higher-order functions and the syntax question

Higher-order functions are unaffected by this refactor, since the goal is the parity
tenet, not the choice of syntax. Unifying `Config` and `Inputs` is a statement at the
type level (a function has one parameter list); it deliberately takes no position on
which surface bracket spells which kind of binding. Collapsing the two brackets
forecloses nothing about higher-order functions: that decision is left fully open, and
the unified model makes it cleaner to revisit, not harder.

One direction the unified model leaves room for is to standardize `()` for a function's
ordinary inputs and `{}` for an upstream-trigger binding, so that `{}` reads as
"trigger." Under that convention the two forms below are equivalent:

```arc
a -> f(b) -> output
{a} -> f(b) -> output
f{a}(b) -> output
```

A function takes its inputs through `()` and may accept any input type, higher-order
functions included; the `{}` form supplies the wire-bound trigger. A reusable
supervisor, written once and reused across signals with a different classifier each
time, shows the shape:

```go
// `value` is the trigger (edge-fed); `severity` (a function) and `limit`
// are ordinary inputs, fixed when the graph is built.
func limit_status{value f32}(severity fn(f32) string, limit f32) string {
    if value <= limit { return "ok" }
    return severity(value - limit)
}

// Plain f32 -> string classifiers (Pressure)
func pressure_severity(excess f32) string {
    if excess > 65.0 { return "emergency" }
    if excess > 60.0  { return "severe" }
    if excess > 55.0  { return "caution" }
    return "elevated"
}
line_psi -> limit_status(pressure_severity, 50) -> line_status
tank_psi -> limit_status(pressure_severity, 50) -> tank_status

// Plain f32 -> string classifiers (Temperature)
func temp_severity(excess f32) string {
    if excess > 445 { return "emergency" }
    if excess > 430 { return "severe" }
    if excess > 415 { return "warning" }
    return "elevated"
}
chamber_temp_1 -> limit_status(temp_severity, 400) -> chamber_status_1
chamber_temp_2 -> limit_status(temp_severity, 400) -> chamber_status_2
chamber_temp_N -> limit_status(temp_severity, 400) -> chamber_status_N

```

These are illustrative, not a commitment: this RFC ships the parity foundation, and a
separate RFC picks the surface syntax.

### 8.3 Topics deliberately not considered

A topic belongs in the future work above only if the `Config`/`Inputs` unification
**opens, closes, or constrains a door** for it. The topics below do not: the refactor
leaves each exactly as it found it, so each is the concern of a different RFC, if any.
They are listed here only to answer why a function-call refactor does not reach them,
not for any lack of merit:

- **Removing stateful variables.** An execution-model concern (persistence of values
  across reactive firings), orthogonal to how a call's parameters are bound. A
  non-trigger param is "held" across firings today and stays held after the refactor;
  the mechanism is untouched.
- **For-loops and other statement-level control flow.** This lives in the grammar above
  the call expression, where the parameter-binding collapse never reaches.
- **Global variables.** A scoping and lifetime question, independent of function
  signatures.
- **Error handling and propagation.** Whether a function can fail, and how that failure
  surfaces (a panic, a `Result`-style return, propagation up the flow graph), is an
  execution-and-return concern. It is decided after arguments are bound, so the shape of
  the parameter list never reaches it.
- **Generic, type-parameterized functions.** Abstracting a function over the types of
  its params is a type-system feature. The unification changes how many lists a
  signature carries, not how the types within it are resolved; type-variable inference
  is neither simplified nor obstructed by the collapse.

## 9 Change footprint

Productive code (everything except codec migration scaffolding) is net-negative,
matching the intuition for a tech-debt-paydown refactor: the analyzer collapse and the
service-layer status cleanup dominate the deletions, while `Config`-to-`Inputs` renaming
is a wash and the dual-list `ExecBoth` symbols dedupe. The codec migration scaffolding
is one-time boilerplate (the Oracle-generated pre-refactor snapshot plus a mechanical
`Migrate*`) and is the only thing that tips the gross total positive.

## 10 Test coverage

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
- **`call.Analyze` shared validation**: argument count, type, name resolution,
  unknown-name detection, duplicate-name detection, verified once against both parens
  and brace surface forms. Today's coverage duplicates each scenario across two test
  files.
- **`Trigger` consult**: a flow call to a `TriggerOnly` symbol does not type-check the
  upstream; a flow call to a `TriggerInput("x")` symbol type-checks the upstream against
  the `x` param's type and rejects type mismatches.
- **`Trigger` invariant**: the `stl_test.go` symbol walker fails when a registered
  symbol declares `Trigger.Target = "missing_param"`.
- **Surface-syntax preservation**: the smoke program in Section 11 parses, analyzes,
  compiles, and runs identically to the pre-refactor branch.
- **`status.set` collapse**: the literal-validation `AnalyzeArguments` hook fires
  correctly for both `status.set(..., "errpr")` and `status.set{variant="errpr"}`,
  emitting one diagnostic per call site (today's hooks would each fire once).

## 11 Verification

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

LSP smoke: hover over `status.set` in the Console editor renders the unified `Inputs`
list correctly. Completion proposes named-config keys for the brace form exactly as it
does today.

## 12 Risks and open questions

**Pluto direct IR construction.** Some tile components may construct `ir.Node` with
`config:` literals at the API layer. Needs a grep during the Pluto reconciliation step.
The migration is mechanical (`config: {...}` folds into `inputs: [...]`), but the sites
have to be found first.

**WASM ABI ordering.** WASM offsets in
[arc/go/stl/wasm/wasm.go](../../../arc/go/stl/wasm/wasm.go) depend on the order the
compiler concat already produces (`Config` first, then `Inputs`). The migration appends
in that same order, so offsets stay stable. The migration round-trip tests verify
byte-equal IR for the same source.

**Hook ctx type assertion.** Today's hooks switch on the concrete `acontext.Context[T]`
type. The unified `AnalyzeArguments` could keep `ctx any` and let each hook type-assert
what it needs (smaller diff), or introduce a `DiagBearer` interface that exposes just
the diagnostic surface hooks actually use (typed and discoverable, larger diff). Open
question; no decision taken here.

**Codec version bump.** This is a wire-format break. The codec version increments as
part of the major release this refactor ships with, skipping the next sequential number.
The exact target version is fixed once the release is named; this RFC uses `vN` as a
placeholder throughout.

## Appendix A - Running notes

An append-only log of decisions, observations, and deferred ideas surfaced while
implementing this RFC. Add new entries at the bottom; do not edit or reorder existing
ones. Strict append-only keeps this section free of merge conflicts across the stacked
phase branches and leaves the numbered sections above untouched. When a note hardens
into a real design decision, promote it into the relevant section above (or §8 Future
work) and leave the note here as a record. Entry heading:
`### <phase or date> - <short title>`.

#### Phase 3 - Brace/parens surface convention may invert

Today a user-defined function's brace block holds the non-trigger inputs and the parens
block holds the trigger (its first param is the edge-fed one):

```go
func my_func{inputs}(trigger) { ... }
```

A future direction worth weighing is inverting this, so `{}` denotes the flow/trigger
surface and `()` the inputs. That would match how `{}` already reads for sequences and
stages, leaving `()` intuitively for inputs:

```go
// Function definition
func my_func{trigger}(inputs) { ... }
func {trigger} my_func(inputs) { ... }
func {trigger} -> my_func(inputs) { ... }

// Usage in flow context:
my_channel -> my_func(inputs) -> output
{ch1, ch2} -> my_func(inputs) -> output
```

This is surface syntax only. The unified `Inputs` list and explicit `Trigger` binding
from this RFC are unaffected either way, which is exactly why the foundation was kept
policy-free (see §8.0). Deferred; no decision yet.

#### Phase 3 - Grammar still uses "config" terminology

Phase 1 removed the `Config` field from the type/IR model (Oracle schema). The ANTLR
grammar, a separate generator, still names its productions `config`: the parser exposes
`ConfigBlock()`, `ConfigValues()`, `NamedConfigValues()`, `AnonymousConfigValues()`,
`ConfigList()`, and `AllConfig()`, which the analyzer calls to read the `{...}` form. §2
currently lists the grammar as a non-goal, so these names survive this RFC.

Since `config` is being deprecated, the grammar terminology should follow. This is a
standalone effort, not a sub-part of the analyzer collapse: the analyzer is indifferent
to what the parser methods are named. It is a wide mechanical sweep (edit the `.g4`,
regenerate the parser, update every caller across arc/go and other grammar consumers).

Do it last, after Phase 9, when the tree is green. A mechanical rename on a compiling
tree can lean on the compiler and tests to catch every missed caller; done mid-stack on
the broken-by-design tree there is no such safety net. This is strictly better than
doing it earlier, not just acceptable: the rename is independent of the type/analyzer
collapse, so nothing in Phases 1 to 9 is blocked by the `config` names surviving, and
nothing in the rename is blocked by the refactor. Promote §2 from non-goal to goal when
it lands.

Open question for that phase: what the rules rename to (e.g. `braceBlock` / `parenBlock`
by surface form). Not just find-and-replace.

#### Phase 4 - `symbol.KindConfig` should be deprecated

Distinct from both the type-level `Config` collapse (this RFC) and the grammar `config`
rename (above): the symbol-table scope `Kind` still carries a `KindConfig` / `KindInput`
split. Plan §Phase 3 deliberately kept it ("the symbol `Kind` distinction stays via the
block the param came from"). Since this RFC's intent is that "config" is no longer a
load-bearing concept, the scope `Kind` should follow.

Phase 4 narrowed it without removing it: the graph analyzer now binds all params
`KindInput`, and `KindInput` absorbed `KindConfig`'s channel behavior at the two sites
where they diverged (`compiler/statement/variable.go` channel-source resolution and
`analyzer/expression/expression.go` channel-read tracking). The remaining producer is
`addConfigToScope` (text path, `analyzer/function/function.go`), which still emits
`KindConfig` for brace-block params; every `KindConfig` reader therefore has to match
both kinds for now.

Full removal: flip `addConfigToScope` to `KindInput`, delete the `KindConfig` enum value
and its `kind_string.go` entry, and collapse the readers. Most are already trivial: the
`KindConfig` cases in `compiler/expression/identifier.go` and the write path of
`compiler/statement/variable.go` are byte-identical to their `KindInput` neighbors, and
`analyzer/statement/statement.go` only special-cases `KindChannel`; the LSP sites
(`lsp/semantic.go`, `lsp/hover.go`) are cosmetic.

Do it after the tree is green (post-Phase 9), same reasoning as the grammar rename: a
mechanical `Kind` collapse on a compiling tree lets the compiler and tests catch every
missed reader, whereas mid-stack on the broken-by-design tree there is no safety net.
Independent of the grammar rename; neither blocks the other.

#### Phase 5 - Runtime inputs are addressed by name, retiring the config/input split

The plan scoped the runtime touch as a one-liner (`len(Config)` to `len(Inputs)`), but
the merge broke more than that: the runtime read inputs by position and split params via
a `configCount` prefix (`params[configCount+j]`). With the lists unified there is no
contiguous boundary to recover (`math.avg`'s edge-fed `reset` carries a default value
while its edge-fed `input` does not), and a host node reading `Input(0)` for its data
now hits a former config param.

So the split is retired from the runtime, not patched. `node.State` already materializes
literal-fed inputs as constant series, so the model was half-realized; three changes
finish it: host nodes read inputs by name (`State.InputNamed`, etc.) so declaration
order stops mattering to them; `wasm.go`/`node.go` drop `configCount` and fill each
param slot from its source (edge-fed streams per sample, literal-fed set once, strings
the only real edge-vs-literal branch); and the `AnalyzeArguments` hook receives the
diagnostics sink directly, so it needs neither the analyzer context nor a type switch.

Order is now cosmetic for the runtime, surviving only for positional argument binding
and the self-consistent WASM param sequence. This is the first change to the runtime
input files (`runtime/node/state.go`, `stl/wasm/node.go`), which Phase 4's
node-construction change left reading positionally, and is the largest non-mechanical
change in the stack; it is code-complete but unverifiable until the tree is green, and
warrants focused review plus the Phase 7 fixture sweep.
