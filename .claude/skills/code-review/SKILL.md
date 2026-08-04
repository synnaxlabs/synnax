---
name: code-review
description:
  Process and hard rules for conducting code reviews in this repo: PRs, branches,
  commit ranges, and working diffs. Use whenever reviewing any diff-shaped artifact,
  walking the user through a change, or assessing mergeability. Extends and overrides
  the built-in code-review skill. Enforces true-base attribution, a chunked reading
  guide, verification before reporting, and severity discipline. Triaging incoming
  comments on a PR you authored belongs to the `pr-comments` skill.
---

# Code Review

A review finding is a claim that something in the diff is wrong, and claims are earned
with evidence. The recurring reviewer failure modes this skill exists to prevent:
reviewing against the wrong base, judging a hunk without its surrounding code, flagging
intentional patterns as bugs, findings too vague to evaluate, and severity inflation.
The deliverable is never a dump of findings — it is a reading guide to the diff that the
user is walked through in tight chunks, with verified findings attached where they
belong.

**Base skill.** This skill extends the built-in `code-review` skill and wins wherever
they conflict. Effort-level semantics and the `--comment` / `--fix` flags carry over;
`--comment` counts as the explicit instruction required before anything is written to
GitHub. The built-in's process, output format, and severity model are replaced by the
rules below.

## The process spine

1. **Establish the artifact and its true base** (Rule 1). What is being reviewed, what
   it is actually diffed against, and which defects are pre-existing.
2. **Map the diff into a reading guide** (Rule 2). Chunk by architectural concern,
   classify, order by risk.
3. **Read beyond the diff** (Rule 3). Surrounding code for every hunk; blast radius for
   shared primitives; existing coverage before any coverage claim.
4. **Assemble the criteria in force** (Rule 4). Load the written law for the touched
   areas; never review from memory of it.
5. **Verify every candidate finding** (Rule 5, gated by the trap list in Rule 6).
   Unverified findings do not survive to the report.
6. **Report and walk through** (Rules 7–9). Severity-disciplined findings, delivered as
   a user-paced walkthrough, collapsible into postable review comments.

Findings go to the user in conversation, always; nothing is posted to GitHub without an
explicit instruction in that session.

## Rule 1: Attribute against the true base

Reviewing the wrong diff poisons every downstream finding.

- **Find the real base.** Feature/fix PRs target `rc`, hotfixes `main`, stacked PRs the
  parent branch (`gh pr view --json baseRefName`). A stacked PR reviewed against `rc`
  shows the whole stack; sibling content already merged into the base washes out. Review
  only what the PR actually introduces.
- **Recover renames locally.** GitHub disables rename detection on large diffs, so moves
  render as delete + add. Run `git diff -M -C <base>...<head>` before treating any "new
  file" as new code, and before mourning any "deleted" one.
- **Record the baseline.** Failures that exist on the base (check-types errors, red CI,
  lint debt) are never attributed to the diff. When attribution is in question, check
  the base's CI, not just the head's.
- **PR hygiene, briefly.** Title matches `SY-####: Title Case`, template filled with
  real content, base branch correct. One line in the report; never the headline.

## Rule 2: The reading guide — chunk the diff by concern

The map is the first deliverable. Partition every changed file into named chunks by
architectural concern — never by directory listing. Each chunk carries:

- a one-line statement of what it does,
- its file list with rough line counts,
- a classification: **behavioral**, **mechanical/rename**, **generated**, **test-only**,
  or **config**.

Order chunks load-bearing-first: behavioral changes where risk lives, then tests, then
mechanical and generated churn the user can skim. Two guarantees: every file in the diff
appears in exactly one chunk (silent omission is a defect in the review itself), and
every finding cites its chunk, so the findings list and the guide stay navigable
together.

**Correct:**

> 1. **Wire format** (behavioral, 2 files, ~60 lines) — adds `end` bounds to the panel
>    schema. 2. **Service guard** (behavioral, 1 file, ~30 lines) — rejects end-before-
>    start on write. 3. **Specs** (test-only, 3 files, ~200 lines). 4. **Generated**
>    (generated, 14 files) — oracle output for chunk 1; skim only.

**Incorrect:** a file-by-file tour in directory order, or a diff summary that names
"various changes across core and console" without partitioning them.

## Rule 3: Read beyond the diff

- **Hunks lie alone.** Before judging any hunk, read its enclosing function or file in
  the current tree. Most false findings come from extrapolating the missing context.
- **Shared primitives get the three-layer blast radius.** When the diff changes a shared
  utility's semantics, audit: (1) every caller; (2) the provenance of each caller's
  inputs — constants inflated to compensate for the old behavior (`MAX + 1`,
  `length + 1`, sibling constants that differ) are the tell; (3) downstream validators a
  changed value now flows into (zod `.max`, `bounds.contains`, cross-package schemas).
  Parallel Explore agents build these inventories well.
- **Coverage claims require an audit.** "Needs test" is the house's most common review
  demand and is legitimate for new behavior — but only after checking what already
  covers the path, including the Playwright integration suite under `/integration/`,
  which is the primary frontend surface. Name the specific missing case; "we need better
  coverage" is not a finding.

## Rule 4: Criteria come from written law — load, don't restate

Assemble the criteria in force for the touched areas and treat every 🚨 / "never" /
"always" rule in them as a review criterion:

- Root `CLAUDE.md` (naming, namespace-context, comments, architectural principles, git
  rules) and the component `CLAUDE.md`s for every touched directory.
- `docs/claude/toolchains/<language>.md` for every touched language.
- The `console-testing` skill for any `console/src/**/*.spec.ts[x]` in the diff;
  `oracle/CLAUDE.md` for schemas or generated code.
- The Resolved Decisions sections of related RFCs (`docs/tech/rfc/`). A diff that
  re-proposes a rejected alternative gets flagged with the RFC number.
- Session memories for the touched area — but re-verify any file:line they cite before
  repeating it (Rule 5).

This skill never restates those rules; a drifted copy is worse than a pointer. It states
directly only what is codified nowhere else: the process above, the trap list, the
severity model, and the output form below.

## Rule 5: The verification gate

Nothing is reported that has not been confirmed real. The gate between "I noticed" and
"I report":

- **State the failure concretely.** A reportable finding names the trigger and the wrong
  outcome, with file:line read in the current tree — not inferred from the diff view,
  not quoted from a memory without re-reading.
- **Check the trap list** (Rule 6). If the pattern is there, the finding dies silently.
- **Mutation-check guard claims.** Before asserting a test protects a condition (or
  fails to), break the condition mentally or actually; a test that stays green tested
  nothing.
- **Read CI, never wave at it.** A non-zero CI exit is never dismissed as flaky or
  environmental without reading the actual failed assertion.
- **Bots and humans get the same gate, in both directions.** Greptile has been wrong
  (flagged accepted designs) and right (caught a real caseconv break). Verify per
  finding; never blanket-dismiss, never blanket-trust.
- **Unverified findings are dropped.** The Question tier (Rule 7) is for verified
  behavior whose _intent_ is unknown — never a laundering channel for facts that were
  too expensive to check.

**Correct:** "`writer.go:142` — `SetEnd` skips the start/end ordering guard that
`Rename` applies via `NewUpdate`; a range can end before it starts. Confirmed: no check
in `SetEnd` or its validator."

**Incorrect:** "The writer might not validate bounds properly." (No trigger, no line,
nothing was read.)

## Rule 6: The trap list — intentional patterns that look like bugs

The codebase contains deliberate patterns that pattern-match to bugs and have been
wrongly flagged before, by past sessions and by bots. They live in session memories
(e.g. the actionObserver pre-commit Notify, `wrapReader`'s value return,
`signal.Isolated` in `Open*()`). Before reporting a finding, check the loaded memories
for a trap covering it; when one fires, say nothing. When a finding dies to a trap not
yet recorded, record it.

## Rule 7: Severity discipline — four tiers and a silence rule

Every finding carries exactly one tier, stated up front:

1. **Blocking** — must fix before merge. Correctness bugs; violations of the hard-rule
   ledger (deleted or trimmed tests, backwards-compat re-export aliases, visibility
   loosening, type erasure, wire-format or generator-wide changes without sign-off,
   customer names / NDA leaks); missing tests for new behavior.
2. **Should-fix** — a real defect that can defer. Say explicitly that it is deferrable
   and suggest a Linear follow-up over expanding the PR, matching house practice.
3. **Nit** — prefixed `Nit:`, and the author is genuinely free to ignore it.
4. **Question** — the behavior is verified; whether it is intended is not. This codifies
   the house "shouldn't this…?" form and accepts "intentional, because…" as a full
   answer.

**The silence rule.** Not reported anywhere: taste with no house rule behind it; style a
formatter owns; trivia of the kind reviewers themselves dismiss ("trivial, ignored");
anything on the trap list; and pre-existing defects the diff didn't introduce — with one
carve-out: a serious pre-existing bug confirmed along the way is reported separately,
labeled out-of-diff, never mixed into the PR's findings.

**No drama.** Before calling a fix expensive or risky, scope it concretely: which files,
which tests, roughly how many lines. Severity describes the defect, never the effort of
fixing it.

## Rule 8: Findings — short, targeted, fix-proportional

A finding is one to three sentences: tier, `file:line`, the defect, the evidence that
makes it real, and a fix sized by the proportionality rule:

- **Mechanical and unambiguous** → the fix rides inline, one clause or a short snippet.
- **Structural** → lead with the long-term structural direction; a tactical mitigation
  is a footnote at most, never the recommendation.
- **The fix is a decision the user owns** (wire change, contract widening, API rename
  with churn) → present options with one recommendation and stop for sign-off.
- **Needs discussion** — a first-class mode, not a failure. Name the defect, say why the
  fix isn't obvious, and resolve it together during the walkthrough.

**Correct:** "Should-fix — `queries.ts:88` retrieves the channel twice per render;
memoize the second lookup or lift it into the existing `useMemo` above."

**Incorrect:** "This file has some performance concerns and the error handling could be
more robust." (No line, no defect, no evaluable fix.)

## Rule 9: The walkthrough

- **Map first.** Deliver the Rule 2 reading guide with a one-line verdict summary: N
  blocking, M should-fix, K nits, J questions.
- **Then one chunk at a time, tight and concise.** Each step: the chunk's purpose, its
  findings, anything needing discussion. Wait for the user; they set the pace and may
  reorder, skip, or drill in.
- **Resolve as you go.** Discussion-mode findings get worked out in their chunk;
  verdicts the user hands down (fix, defer, reject) are recorded against the finding.
- **Collapse on request.** The findings double as a buildable list of postable review
  comments: terse, one finding per comment, `Nit:` prefix intact, question form where
  the tier is Question, no customer names, no em dashes. Posting anything to GitHub
  still requires the explicit instruction.

## Rule 10: Author-side triage lives in `pr-comments`

Triaging incoming review comments on a PR you authored is owned by the `pr-comments`
skill. Load it instead of working from this one. Rule 5's verification gate and Rule 6's
trap list still apply per comment; everything else about that workflow, discovery of the
unresolved threads, the one-at-a-time loop, and the batched GitHub writes, lives there.

## Rule 11: Tooling — CI runs the checks, the reviewer reads

Reviews do not run lint, check-types, or test suites; CI owns those, and the review
reads CI's results instead (through the Rule 5 never-unread gate). Executable tooling
appears only as investigation for what CI cannot answer: `git diff -M -C` rename
recovery, blame and history archaeology, caller inventories, or a targeted probe when a
single finding genuinely hinges on it. Builds, integration runs, and anything against a
live cluster belong to the user. The reviewer's instruments are reading and search.

## Quick reference

| Situation                       | Gate                                                            |
| ------------------------------- | --------------------------------------------------------------- |
| Starting any review             | True base found; renames recovered; baseline recorded           |
| Producing the first deliverable | Reading guide: concern-chunked, classified, risk-ordered        |
| Diff touches a shared primitive | Three-layer blast radius: callers, input provenance, validators |
| Claiming missing coverage       | Existing tests audited (incl. integration); specific case named |
| Any candidate finding           | Verified at file:line in the current tree; trap list checked    |
| Assigning severity              | One of Blocking / Should-fix / Nit / Question; else silence     |
| Attaching a fix                 | Proportional: inline, structural-first, sign-off, or discussion |
| Delivering the review           | Map + verdict counts, then user-paced chunk walkthrough         |
| Incoming review comment         | Not this skill; load `pr-comments`                              |
| Writing anything to GitHub      | Explicit user instruction in this session                       |
| Tempted to run lint/tests       | Don't — read CI; tooling is investigative only                  |
