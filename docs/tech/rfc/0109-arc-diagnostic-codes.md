# 109 Arc diagnostic codes

- **Author**: Nico Alba
- **Date**: 2026-11-12

## 0 Summary

The form of an Arc diagnostic code changes from `ARC####` to `ARC-E####` for an error
and `ARC-W####` for a warning. This is a change of form only: no kind keeps its number.
The codes move into five bands that cover the full language. Each diagnostic gets a
code, each code has a diagnostic, and the docs page follows the bands.

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

A code is `ARC-<letter><number>`.

- **Letter**: `E` for an error, `W` for a warning. Arc reports no other severity, and a
  new one needs a new letter first.
- **Number**: Four digits. The first digit is the band (§2.1).

The letter and the number together identify a kind. Errors and warnings have separate
numbers, so `ARC-E2001` and `ARC-W2001` are two kinds with no relation.

A new kind takes the next free number in its band, for its letter. This work assigns the
numbers in one pass, after a review of each diagnostic. An existing kind gets no
preference, so an old number has no meaning in the new codes.

A kind that changes severity takes a new code. Its old code is not used again.

### 2.1 Bands

| Band | Area                     | Examples                                            |
| ---- | ------------------------ | --------------------------------------------------- |
| 1xxx | Syntax                   | Mismatched input                                    |
| 2xxx | Types and expressions    | Type mismatch, casts, units, constant overflow, NaN |
| 3xxx | Functions and statements | Arguments, `return`, `break`, assignment            |
| 4xxx | Names and modules        | Undefined name, deprecated name, imports, `export`  |
| 5xxx | Flows and sequences      | Routing, channel writes, authority, stages, `next`  |

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

- **One Markdown file for each code**: It lives beside `codes.go`, as the Rust compiler
  does (`E0308.md`). It has three parts: the problem, an example that fails, and the
  corrected example.
- **Real messages**: `go generate` compiles the failing example, checks that it reports
  the code, and writes the message to `codes.gen.json`. The page shows that message, so
  no person names the `%s` values.
- **Layout**: One heading for each band.
- **Anchors**: The anchor of a code is the code in lowercase, such as `#arc-e2001`.

## 3 What this RFC does not cover

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

## 6 Open questions

- **Type mismatch**: One code for all sites, or one code for each cause. The review of
  the sites decides it.
- **Check for a reported error**: The lint rule cannot see an error with no kind that
  `diagnostics.Error` reports. A different check must cover it.
- **Page design**: Whether a new design for an item on the errors page is part of this
  work.
