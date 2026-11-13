# 109 Arc diagnostic codes

- **Author**: Nico Alba
- **Date**: 2026-11-12

## 0 Summary

The form of an Arc diagnostic code changes from `ARC####` to `E####` for an error and
`W####` for a warning. This is a change of form only: no kind keeps its number. The
codes move into four bands that cover the full language. Each diagnostic gets a code,
each code has a diagnostic, and the docs page follows the bands.

## 1 Motivation

- **Most diagnostics have no code**: `arc/go` has 11 report sites with a code and about
  240 without one. A type mismatch, the most common error, has none.
- **Four codes are orphans**: `ARC2001`, `ARC2003`, `ARC4001`, and `ARC4002` are
  declared, but the compiler has never reported them.
- **The bands do not cover the language**: Statements, flows, channels, sequences, and
  units have no band.
- **Errors and warnings look the same**: A reader cannot tell the severity from the
  code, and the docs page lists them together.

## 2 Design

### 2.0 Code format

A code is `<letter><number>`, as in the Rust compiler (`E0308`).

- **Letter**: `E` for an error, `W` for a warning. Arc reports no other severity, and a
  new one needs a new letter first.
- **Number**: Four digits. The first digit is the band (§2.1).

The letter and the number together identify a kind. Errors and warnings have separate
numbers, so `E2001` and `W2001` are two kinds with no relation.

A new kind takes the next free number in its band, for its letter. This work assigns the
numbers in one pass, after a review of each diagnostic. An existing kind gets no
preference, so an old number has no meaning in the new codes.

A kind that changes severity takes a new code. Its old code is not used again.

### 2.1 Bands

| Band | Area                         | Examples                                              |
| ---- | ---------------------------- | ----------------------------------------------------- |
| 1xxx | Syntax and statements        | Mismatched input, `return` or `break` out of place    |
| 2xxx | Types                        | Type mismatch, arguments, casts, units, overflow, NaN |
| 3xxx | Names and modules            | Undefined name, deprecated name, imports, `export`    |
| 4xxx | Flows, sequences, and stages | Routing, channel writes, authority, `next`            |

A kind is broad. All type mismatches share one code, as in the Rust compiler (`E0308`).

### 2.2 Coverage

- Each diagnostic that the compiler reports has a code, except the diagnostics of
  `arc/go/graph` (§3).
- Each code has a report site. The four codes with none go to the sites that report
  their problem today with no code.

| Kind                      | Problem                   | Source today                                     |
| ------------------------- | ------------------------- | ------------------------------------------------ |
| `TypeMismatch`            | Type mismatch             | About 20, such as `expression/expression.go:299` |
| `TypeConstraintViolation` | Type constraint violation | `constraints/unify.go:356`, `:366`               |
| `SymbolUndefined`         | Undefined symbol          | `symbol/errors.go:52`, `symbol/symbol.go:780`    |
| `SymbolRedefined`         | Redefined symbol          | `symbol/symbol.go:410`                           |

Some of these sources build a Go error, and a later `diagnostics.Error` call reports it.
One such call can report errors of different kinds, so the call cannot choose the code.
The error carries its kind, and `diagnostics.Error` copies the code to the diagnostic. A
hint works this way today (`HintProvider`, `x/go/diagnostics/diagnostics.go:57`). The
two plain errors, at `symbol/symbol.go:410` and `:780`, become typed errors.

### 2.3 Docs

The errors page is generated from the kinds. `errors.mdx` has no item for each code.

- **One JSON file declares each code**: `go generate` writes `codes.gen.json` from
  `codes.go`. It holds the code, severity, message, problem text, and examples of each
  kind. The docs site and any other tool read only this file.
- **One Markdown file for each code**: It lives beside `codes.go`, as the Rust compiler
  does (`E0308.md`). It has three parts: the problem, an example that fails, and the
  corrected example. The generator copies them into `codes.gen.json`.
- **Real messages**: `go generate` compiles the failing example, checks that it reports
  the code, and writes the message to `codes.gen.json`. The page shows that message, so
  no person names the `%s` values.
- **Layout**: One heading for each band.
- **Anchors**: The anchor of a code is the code in lowercase, such as `#e2001`.

### 2.4 Prior art

| Tool        | Code form      | Bands                                   |
| ----------- | -------------- | --------------------------------------- |
| Rust        | `E0308`        | None. A lint has a name, not a number   |
| TypeScript  | `TS2322`       | 1xxx syntax, 2xxx semantics             |
| pycodestyle | `E501`, `W291` | `E` or `W`, then one digit for the area |
| ShellCheck  | `SC2086`       | 1xxx parser, 2xxx analysis              |

Arc takes the letter for severity from pycodestyle, the bands from TypeScript and
ShellCheck, and broad kinds and one Markdown file for each code from Rust.

## 3 What this RFC does not cover

- **Runtime errors**: A code is for a diagnostic of the compiler. A runtime error keeps
  its typed error (`arc/cpp/runtime/wasm/module.h:177`) and shows in the status of the
  task.
- **Graph diagnostics**: `arc/go/graph` reports a malformed graph, and the graph editor
  is deprecated. Its diagnostics get no code.
- **Oracle diagnostics**: Oracle uses the same Go diagnostics package and has no codes.

## 4 Implementation phases

The work is one pull request, because the numbers of a band are assigned in one pass.
Before the first commit, a review lists each diagnostic, puts it in a kind and a band,
and gives it a number. The commits are in this order:

1. Change the code format and renumber the existing kinds.
2. Add the kinds of each band, one commit for each band.
3. Generate the errors page, with one Markdown file for each code.
4. Change each place that names an old code or its anchor: the links in the docs,
   `arc/docs/spec.md`, and the specs in `arc/go`, `site/docs`, and `pluto`.
5. Add a lint rule that rejects `diagnostics.Errorf` and `diagnostics.Warningf` in
   `arc/go`, except in `arc/go/graph`.

| Phase | Scope                           | Touches                                                      | Depends on |
| ----- | ------------------------------- | ------------------------------------------------------------ | ---------- |
| 1     | Reorganize Arc diagnostic codes | `arc/go`, `arc/docs`, `site/docs`, `pluto`, `.golangci.yaml` | -          |

## 5 Resolved decisions

1. **A lint command needs no new codes**: The code is the rule ID. A style rule takes a
   `W` code, not a new letter.
2. **Bands stay, but fewer**: The codes do not need bands, and a count from 1 also
   works. Bands keep related codes together as the list grows, so a new code does not
   land far from its relatives. The design went from eight bands to five, and then to
   four.
3. **The source is Go, and the JSON is generated**: A check reports through a Go value,
   such as `codes.TypeMismatch`, so the kinds must be Go. A JSON source needs a
   generator that writes Go, which is a second step for the same result. The prose of a
   code stays in Markdown, because Arc examples in JSON strings are hard to review.

## 6 Open questions

- **Check for a reported error**: The lint rule cannot see an error with no kind that
  `diagnostics.Error` reports. A different check must cover it.
- **Page design**: Whether a new design for an item on the errors page is part of this
  work.
