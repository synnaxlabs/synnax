# RFC conventions

Rules for writing and editing RFCs in this directory.

## Files and structure

- The H1 is `# NN Title` in sentence case, using the unpadded number. A
  `Component - Subtitle` H1 capitalizes the subtitle: `# 4 Signal - Goroutine manager`.
- Front matter is a bullet list directly under the H1, in this order:
  `- **Author**: Name`, `- **Date**: YYYY-MM-DD`,
  `- **Related**: [RFC NNNN - Title](NNNN-slug.md)` (Related is optional). A Related
  title matches the target's H1; for a `Component - Subtitle` H1 the component alone is
  enough.
- Headings are numbered hierarchically from zero: `## 0 Summary`, `### 0.0 ...`. Every
  heading below the H1 carries a number — appendix and working-notes sections too.
- Implementation stages are labeled `Phase N` — never `Part N`.

## Headings

- Sentence case: capitalize only the first word, proper nouns, and acronyms. Text after
  a colon follows the same rule: `### 8.4 Phase 5: Integration`
- A heading word that names a code identifier goes in backticks with its real casing,
  not title case: ``### 4.0 The `domain.DB` write path``.

## Definition lists

- Vocabulary and definition entries use a colon after the term:
  `- **Term**: Definition.` Never `-` or `—` after the term.
- The same applies when the term is a code span: `` `send(payload)`: Sends... ``
- Parentheticals attach to the term, before the colon: `- **Term** (scope): ...`.
- Use unordered bullets unless the order itself carries meaning.
- Definitions are sentence-cased: capitalize the first word, then normal prose rules.
  Acronym expansions are not title-cased: `**DAQ**: Data acquisition computer.` Proper
  nouns inside an expansion keep their capitals.
- Principle and resolved-decision entries take the same colon form:
  `1. **One Go type per entity**: Owned by its service.` When the bold label leads with
  a section number, it carries no separator, matching headings:
  `**6.1 Predecessor chain, not direct-to-definer.**` These numbers stay gapless like
  headings: an entry added between two others renumbers its successors. Fix the
  references that point at them.

## Prose

- Put a space between a number and its unit, with correct unit casing: `1 Hz`, `25 kHz`,
  `10 ms`, `100 µs`, `200 kB`. Kilo is a lowercase `k`; mega and giga are `M` and `G`.
  Use the micro sign µ (U+00B5), never Greek mu (μ).
- Backtick type names and code identifiers in prose: `uint8`, `float64`, `domain.DB`,
  channel names like `sy_task_set`. CLI commands too: `oracle migrate`.
- Proper nouns follow the root `CLAUDE.md` Prose rules. Package and import paths stay
  lowercase in backticks, and so do commands (`git log`) even when the tool is a proper
  noun (Git).
- Version references are lowercase: `v1`, `v2`, `v0.2`.
- Cite a section with the section sign and no space: `§4.2` within the same RFC,
  `RFC 0033 §4.2` across RFCs. Never spell out `Section 4.2`. Match the target's real
  number — renumbering a section breaks references elsewhere; check for them.
