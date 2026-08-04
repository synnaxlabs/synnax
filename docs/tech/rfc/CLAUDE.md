# RFC Conventions

Rules for writing and editing RFCs in this directory.

## Files and structure

- File names are `NNNN-slug.md` — four-digit number, kebab-case slug, no dates.
- The H1 is `# NN Title` in sentence case, using the unpadded number.
- Front matter is a bullet list directly under the H1, in this order:
  `- **Author**: Name`, `- **Date**: YYYY-MM-DD`,
  `- **Related**: [RFC NNNN - Title](./NNNN-slug.md)` (Related is optional).
- Headings are numbered hierarchically from zero: `## 0 Summary`, `### 0.0 ...`.
- Implementation stages are labeled `Phase N` — never `Part N`.

## Headings

- Sentence case: capitalize only the first word, proper nouns, and acronyms.
- A heading word that names a code identifier goes in backticks with its real casing
  (`### 4.0 Parallel offset` `` `domain.DB` ``), not title case.

## Definition lists

- Vocabulary and definition entries use a colon after the term:
  `- **Term**: Definition.` Never `-` or `—` after the term.
- The same applies when the term is a code span: `` `send(payload)`: Sends... ``
- Parentheticals attach to the term, before the colon: `- **Term** (scope): ...`.
- Use unordered bullets unless the order itself carries meaning.
- Acronym expansions are not title-cased: `**DAQ**: data acquisition computer.` Proper
  nouns inside an expansion keep their capitals.

## Prose

- 88-character lines, hand-wrapped and filled to the limit. Measure characters, not
  bytes — em dashes and µ are multi-byte.
- Put a space between a number and its unit, with correct unit casing: `1 Hz`, `25 kHz`,
  `10 ms`, `100 µs`, `200 KB`. Use the micro sign µ (U+00B5), never Greek mu (μ).
- Backtick type names and code identifiers in prose: `uint8`, `float64`, `domain.DB`,
  channel names like `sy_task_set`. CLI commands too: `oracle migrate`.
- Capitalize Synnax component names in prose: Arc, Cesium, Aspen, Core, Console, Pluto,
  Freighter, Alamos, Gorp, Drift, Driver, Oracle. Package and import paths stay
  lowercase in backticks.
- Version references are lowercase: `v1`, `v2`, `v0.2`.
- When citing another RFC's section or heading, match its real number and casing —
  renumbering a section breaks references elsewhere; check for them.
