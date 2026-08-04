---
name: quick-review
description:
  Process and hard rules for a fast software quality audit of work this session just
  produced, or of a solution being proposed before it is built. Use when asked to go
  back and review your changes, to check whether an approach holds up, or to audit a
  plan before implementing it. Runs nine fixed lenses: complexity cost, naming,
  anti-patterns, hackiness, structural avoidance, performance, new patterns, test
  pinning, and robustness. Not a correctness review; `code-review` owns that.
---

# Quick Review

Quick review asks whether this is good software, not whether it works. The subject is
one small, self-contained piece of work: the fix or feature this session just wrote, or
the solution it is about to write. Nine lenses, all of them, every time. Simple problems
get fixed on the spot. Problems that change the shape of the solution go to the user.

**Not a correctness review.** No lens hunts for bugs. A clean quick review is never a
claim that the change works. Correctness belongs to tests, CI, and `code-review`.

**Not a PR review.** No base attribution, no chunked reading guide, no walkthrough.
Reviewing a branch, a PR, or someone else's diff is `code-review`'s job and costs an
order of magnitude more. Reach for this one when the subject is small and yours.

## Invocation

User-invoked only, on explicit language: "go back and do a review of your changes",
"review what you just did", "does this approach hold up", "audit this before we build
it". It never fires on its own at the end of an implementation.

## The two subjects

1. **Written code.** A fix or small feature this session just implemented.
2. **A proposed solution.** An approach or implementation plan still on the table. This
   covers any moment a solution is being proposed, not only Claude Code's plan mode.

The same nine lenses apply either way. Against a proposal they are asked forward
looking: LOC is an estimate, the public API delta is stated exactly, and lens 8 becomes
"what pins this, and can it actually be written".

## The nine lenses

Run every one, in order.

1. **Complexity cost.** What does this cost in complexity? Report the LOC added and
   removed, the new surface area for bugs to arise in, and how much the PUBLIC API
   surface grew. Then answer the real question: is that expansion justified?
2. **Naming.** Do the names follow the naming conventions? The namespace-carries-the-
   context rule is the one broken most often.
3. **Anti-patterns and standards.** Did the work reduce the anti-patterns named in the
   language guide for the area it touches? Did it follow the standards for things like
   writing tests?
4. **Unsafe or hacky code.** Excessive type assertions, type erasure, needless control
   flow, anything load-bearing that works only by accident.
5. **Structural avoidance.** Is the fix working around a deeper structural problem? If
   so, name that problem and say whether solving it is the better move.
6. **Performance cost.** Allocations, repaints, network fetches. What did this add to
   the hot path?
7. **New patterns.** Does this introduce a pattern the codebase did not have before? If
   it does, justify it in blood or drop it. The absence of a pattern across a large
   codebase is itself a declared principle.
8. **Test pinning.** Can the new behavior be pinned with tests, regression tests, or
   both? If it can be and it isn't, that is a finding.
9. **Robustness.** Is this the production grade path, and why? What alternatives were
   considered and rejected, and is the chosen one actually the best of them?

## Fix authority

The dividing line is how many right answers there are.

- **One right answer, and it is local.** Fix it, then report the fix at file:line.
  Naming, comment bloat, needless control flow, an unnecessary type assertion, a missing
  test. All governed by written law, all reversible.
- **More than one right answer, or it changes the shape of the solution.** Do not touch
  it. Report it with a recommendation and let the user decide. Solving the deeper
  structural problem, taking a different path, cutting public API surface, reworking for
  performance.

**Proposals are the exception.** When the subject is a proposed solution rather than
written code, revise the proposal directly and show the result, shape changes included.
Rewriting a proposal is cheap and fully visible; rewriting landed code is neither. The
report says what changed and why.

## The effort ceiling

Quick review is quick because of what it does not do.

- **Read and grep freely.** Mandatory reading: the touched files in full, plus the
  `CLAUDE.md` and `docs/claude/toolchains/<language>.md` for every area touched. Lens 3
  is worthless without them. Targeted grep for callers, prior art, and the nearest
  analog is in bounds.
- **No subagent fan-out by default.** Full parallel inventories are `code-review`'s cost
  profile. On a substantial body of code fan-out is allowed, but the report states why
  it was needed.
- **Builds and focused test runs are allowed.** Run the touched spec file or the
  affected package. Large or extensive suite runs need the user's go-ahead first.
- **Verify renames by grep.** A fix that renames a symbol or changes a signature checks
  every caller before it is reported as done.

## The report

🚨 **Think hard about the presentation before writing a word of it.** Work out what the
user actually needs to see, in what order, and what can be cut. This is the step that
gets skipped under time pressure, and skipping it is what turns a good review into a
wall of text.

- **Lean.** Short. No preamble, no restating the change back to the user, no per-lens
  scorecard.
- **Plain language.** Explain the problem clearly instead of naming it in jargon. A
  reader who does not know the term should still understand the defect.
- **Two groups only.** What was fixed, and what needs a decision. No severity tiers.
- **The complexity numbers always appear**, even when nothing is wrong: LOC added and
  removed from the actual diff, and the count of exported symbols added. That is the
  price of the change, and the user sees it whether or not it is a problem.
- **Detail only for lenses that fired.** Close with one short line naming the ones that
  came back clean, so a skipped lens cannot hide as a quiet one.
- **Every finding is concrete.** file:line, what is wrong, and what to do instead.

## Quick reference

| Situation                         | Gate                                                    |
| --------------------------------- | ------------------------------------------------------- |
| Deciding whether to run this      | Subject is small and this session's; else `code-review` |
| Running the review                | All nine lenses, every time                             |
| Simple, local problem found       | Fix it; report the fix at file:line                     |
| Shape problem found in code       | Do not touch it; report with a recommendation           |
| Shape problem found in a proposal | Revise the proposal directly; show what changed         |
| Tempted to fan out subagents      | Substantial code only, and justify it in the report     |
| Tempted to run a large suite      | Ask first; focused runs of what changed are fine        |
| Before writing the report         | Decide how to present it clearly, then cut              |
| Reporting complexity              | LOC delta and exported-symbol delta, always             |
| Finishing                         | A clean review is not a claim the change is correct     |
