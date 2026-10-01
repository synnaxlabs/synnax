# 76 Arc brace and paren swap

- **Author**: Nico Alba
- **Date**: 2026-10-01
- **Related**:
  [RFC 0040 - Arc function-call unification](0040-arc-function-call-unification.md)

## 0 Summary

Arc writes function inputs in braces and triggers in parens. This RFC swaps them. Inputs
go in parens and triggers go in braces, in the declaration and in the flow.

**Is:**

```go
func name{param} (trigger1, trigger2) output {
// body
}

{
    trigger1: a,
    trigger2: b
} -> name{param = c} -> avg_output
```

**Proposed:**

```go
func name(param) {trigger1, trigger2} output {
// body
}

{
    trigger1: a,
    trigger2: b
} -> name(param = c) -> avg_output
```

In a flow, the upstream triggers are written with `{}`. The function declaration now
writes its triggers with `{}` too. Inputs use `()`, which is how a function is called
inside a function body and how most other languages write parameters.

Note: the text compiler does not implement input routing tables yet
(`arc/go/text/analyze.go:1779`). The spec documents them (`arc/docs/spec.md:543`), and
support is planned.

The swap should be a hard change. There is no period in which both forms are valid. If
both were valid, `func name(x f64)` would have two meanings, a trigger in the old form
and an input in the new form, and nothing in the text says which one.

## 1 Design

### 1.0 Declaration

```
FunctionDeclaration ::= 'func' Identifier '(' InputList? ')' '{' TriggerList? '}'
                        OutputType? Block
```

Today the trigger parens are required and the input braces are optional
(`arc/go/parser/ArcParser.g4:77`). After the swap, the input parens and the trigger
braces are both required. Each can be empty.

A declaration has one form. The first `{` is always the trigger braces, so the parser
never has to tell them from the body. The outputs do not change. More than one output
keeps its parens, after the trigger braces.

Nothing below the syntax changes:

- A function still has one parameter list. The order is the inputs, then the triggers
  (`arc/go/analyzer/function/function.go:50`).
- A single upstream edge still feeds the first trigger (`function.go:58`). A routing
  table maps the other triggers by name.
- A function with no triggers has only inputs. An upstream edge starts it and binds no
  value. `func f(x f64) {}` declares an input `x`, not a trigger.

### 1.1 Flow

A function is instantiated in a flow with parens.

```go
sensor -> filter(threshold=50.0) -> out
time.interval(1s) -> tick
sensor -> band(limit=50.0) -> {
    high: "high" -> log,
    low: "low" -> log
}
```

The values are all named or all positional, as in the brace form today
(`ArcParser.g4:221`). Routing tables keep their braces.

### 1.2 Rewrite of existing code

Three rules. `{inputs}` becomes `(inputs)`. `(triggers)` becomes `{triggers}`. A missing
pair is added empty.

| Old                                                    | New                                                    |
| ------------------------------------------------------ | ------------------------------------------------------ |
| `func scale{factor f64} (value f64) f64`               | `func scale(factor f64) {value f64} f64`               |
| `func add(x f64, y f64) f64`                           | `func add() {x f64, y f64} f64`                        |
| `func split(value f64) (high f64, low f64)`            | `func split() {value f64} (high f64, low f64)`         |
| `func band{limit f64} (value f64) (high f64, low f64)` | `func band(limit f64) {value f64} (high f64, low f64)` |
| `func log{ch chan f64} ()`                             | `func log(ch chan f64) {}`                             |
| `func tick()`                                          | `func tick() {}`                                       |
| `sensor -> scale{factor=2.0} -> out`                   | `sensor -> scale(factor=2.0) -> out`                   |
| `sensor -> split{} -> {high: 1 -> hi, low: 1 -> lo}`   | `sensor -> split() -> {high: 1 -> hi, low: 1 -> lo}`   |
| `add(1, 2) -> out`                                     | `add{1, 2} -> out`                                     |
| `time.wait{2s}`                                        | `time.wait(2s)`                                        |

## 2 Parser notes

### 2.0 `name(...)` in a flow is the instantiation

A flow node is tried as `identifier`, then `function`, then `expression`
(`ArcParser.g4:193`). Today `name(...)` in a flow matches `expression`, except a bare
name after an arrow with one value, which is read as a name and a separate `(value)`
item (§2.1). After the swap it matches `function`.

### 2.1 In a flow, `(` must touch the name

Newlines are hidden tokens (`arc/go/parser/ArcLexer.g4:167`), and the items of a
program, a stage, or a sequence have no separator. These two programs give the same
tokens:

```go
sensor -> filter(50.0) -> out
```

```go
sensor -> filter
(50.0) -> out
```

The second shape is in use. `arc/go/sequence_test.go:3450` ends one line with a channel
name and starts the next with `(temp_a > 90 and temp_b > 90) => off`.

The rule is that `name(` is an instantiation only when no whitespace separates the name
and the paren. The grammar already has this type of rule for unit suffixes. `300ms` is
one literal and `300 ms` is not (`TokensAdjacent`, `arc/go/parser/parser.go:217`, used
at `ArcParser.g4:459`). When `(` touches the name, the flow node is not read as
`identifier`.

## 3 Implementation phases

One pull request. The old syntax stops parsing, so the grammar and every Arc example and
test in the repo change together to keep `main` green.

- Grammar and the regenerated parser.
- The function analyzer, the formatter, and the LSP.
- `arc/docs/spec.md` and the Arc reference on the docs site.
- Every Arc snippet in tests, integration tests, and examples.

## 4 Migration of stored programs

Open to the opinions of the reviewers.

- **Migrate**: Rewrite each stored program on upgrade with the three rules of §1.2.
- **Do not migrate**: Rewrite the programs with each customer.
