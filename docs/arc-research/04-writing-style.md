# Arc Documentation Writing Style

This document defines the writing style for Arc documentation. Include this context at the
start of each documentation session.

---

## Core Principles

Avoid sounding like AI-generated documentation; prefer blunt, practical explanations over
polished completeness.

Write in a clear, confident, and conversational tone.

Assume the reader is capable and just wants to get something working.

---

## Voice and Tone

- Avoid help-center or FAQ phrasing
- Do not use "this article", "this guide", or "in this section"
- Light conversational cues are okay (e.g., "If you're just getting started…",
  "In practice…")
- No jokes or emojis
- When there is a recommended approach, state it directly

---

## Content Focus

- Focus on the most common usage patterns
- Edge cases should be mentioned briefly or omitted
- Spend more detail on confusing or error-prone steps
- Keep obvious steps short
- Explain why something matters when it's not obvious

---

## Formatting

- Headings should be short and functional, not full sentences
- Examples should look like real production usage, not abstract placeholders
- Do not use double dashes (--) or em-dashes (—); use commas, periods, or restructure

---

## Session Prompt

Use this prompt when starting a documentation writing session:

```
Write user-facing Markdown documentation for the following feature.

Style guidelines:
- Public product documentation (not internal notes, not a tutorial)
- Clear, confident, and conversational
- Avoid help-center or AI-generated phrasing
- Focus on common workflows over edge cases
- Use short, functional headings
- Explain why something matters when it's not obvious
- Make recommendations directly when appropriate

Assume the reader is competent and wants to get value quickly.
```
