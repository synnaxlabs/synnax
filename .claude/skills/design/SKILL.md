---
name: design
description:
  Process and hard rules for designing and planning complex new features, refactors, and
  re-architectures. Use whenever producing an RFC or implementation plan, evaluating
  whether to follow or break an existing pattern, deciding package boundaries or where a
  behavior lives, researching prior art or user workflows, or interviewing the user to
  lock design decisions. Enforces research-before-questions, one-decision-at-a-time
  interviews, and earned phase boundaries.
---

# Design & Planning

A design is a set of claims: this responsibility belongs here, this boundary protects
that, users need this, and the work can land in these increments without leaving the
system broken in between. Every claim must be earned with evidence. The session gathers
the evidence, proposes with a recommendation, and locks decisions with the user one at a
time. The verdict on hard calls stays with the user; the session's job is to feed that
judgment the best possible evidence, never to substitute for it.

## 🚨 THE INTERVIEW RULE 🚨

ALL USER INTERVIEWS ARE GOVERNED BY THE `interview` SKILL. LOAD IT BEFORE ASKING THE
FIRST QUESTION. ITS CORE: ONE QUESTION PER MESSAGE, PLAIN PROSE, PLAIN TEXT RESPONSES,
NEVER AskUserQuestion.

## The process spine

1. **Scope the deliverable.** RFC when the design chooses a shape: an interaction model,
   a wire or persisted format, an architectural boundary. Plan when the shape is settled
   (by an RFC or by precedent) and the work is execution: sequencing, kill lists,
   gotchas. Both are interview-driven; a plan's interview is shorter.
2. **Research.** Implementation research (Rule 1) and the principles-in-force brief
   (Rule 2), plus UX grounding (Rule 6) and prior art (Rule 7) where they apply. Produce
   the artifacts; do not start interviewing without them.
3. **Interview.** Conducted under the `interview` skill; Rule 8 adds the design-specific
   gates. Lock decisions; persist locked answers into documents or memories so the same
   ground is never re-interviewed.
4. **Draft the deliverable** in the house RFC form (Rule 9), rejections folded in.
5. **Phase the work** with earned boundaries only (Rule 10).

Register for the whole process: lead with the long-term structural answer, never a
tactical unblock framed as the path forward. No half measures: complications get worked
through, not punted to hypothetical future PRs. Scope concretely (files, tests, lines)
before calling anything expensive.

## Rule 1: Implementation research — no hallucinated APIs, no reinvented wheels

The recurring design failures are hallucinating APIs that don't exist, recreating
utilities that already exist, and ignoring common practices and architectural seams.
Research must be disciplined enough to make each impossible:

1. **The existence ledger.** Every symbol the design references (function, type,
   service, channel, oracle field) is marked either _exists_, with file:line, actually
   read, signature confirmed, or _NEW_. No third state. Hallucination happens exactly
   when a design names an API without having read it.
2. **Search before invent.** Nothing is marked NEW until the design records where it
   looked for an existing version (`x/`, the owning package, the nearest sibling
   feature) and why each candidate doesn't fit. The answer is often "already exists":
   audit existing tests and machinery before designing replacements.
3. **The precedent rule.** Everything NEW names its nearest existing analog and either
   follows its shape or justifies divergence in blood. Cautionary example: a session
   added a Go `init()`, almost unheard of in this codebase, and on re-examination it was
   completely unjustified. The absence of a pattern across a large codebase is itself a
   declared principle.

Research is complete when the session can produce: the current flow narrated end-to-end
across every layer and language touched, with file:line cites; the pattern
classification (Rule 3); the kill list, everything the design would delete (sentinels,
workarounds, compensating constants, dead flows); and the reference implementation, the
nearest analogous feature traced end-to-end with the why behind its choices extracted
from code, never from PR summaries. Parallel Explore agents are the right tool for the
inventories. For shared-primitive changes, the blast radius is three layers: every
caller, the provenance of each caller's inputs (constants inflated to compensate for old
behavior), and downstream validators.

## Rule 2: The principles-in-force brief

Before designing, assemble the governing principles for the affected area from five
sources:

1. **Written law** -> root and component CLAUDE.mds, `docs/claude/*`, and the Principles
   and Resolved Decisions sections of related RFCs. Resolved Decisions especially: past
   designs already rejected alternatives this design may be about to re-propose.
2. **Grid placement** -> the cell(s) the change occupies on the three axes (layer ×
   domain/feature × general-purpose-vs-Synnax-specific), and the invariants each axis
   imposes there.
3. **Negative space** -> what the surrounding code conspicuously never does (no
   `init()`, no reflection, no singletons). Absences never announce themselves; finding
   them requires breadth-reading neighbors, not just the changed files.
4. **Unwritten whys** -> the extracted reasons behind the load-bearing patterns
   identified via Rule 3's signals.
5. **Violation precedents** -> past decisions where a pattern WAS deliberately violated,
   and why. A sanctioned violation is evidence that a boundary flexes under those
   conditions.

Every structural claim in the design cites the brief, or explicitly proposes a new
principle for user sign-off.

## Rule 3: Load-bearing or incidental — the six signals

Every codebase pattern is either an invariant protecting something real or an incident
of history that looks like architecture. The verdict is intuition and cannot be
codified; the evidence feeding it can. Score each touched pattern on:

1. **Breadth** -> conformers span multiple domains, packages, or languages:
   load-bearing. Confined to one domain: weaker claim to being architecture at all.
2. **Nameable casualty** -> can you state concretely what breaks, degrades, or becomes
   impossible to reason about if the rule vanished? No nameable casualty: suspect
   incident.
3. **Instantiation of a general principle** -> a pattern backed by dependency direction,
   single ownership of state, or single source of truth inherits that principle's
   authority. A sui generis rule must earn authority alone, and usually can't.
4. **Deliberate arrival** -> introduced by an RFC or a sweeping refactor: declared
   invariant. Accreted from a one-off commit or copied sideways: incident. Read the
   actual code and history for the why.
5. **Enforcement and defense** -> documented in a CLAUDE.md, guarded by lint or codegen,
   or visibly defended in review: load-bearing by revealed preference. Violations that
   survived review unchallenged: the pattern may already be dead.
6. **Conformers benefit or merely comply** -> conformance yields reuse, safety, or
   simpler reasoning: structure. Boilerplate everyone works around: friction wearing
   architecture's clothes.

**The protocol.** The design classifies every pattern it touches as _conform_, _extend_,
or _replace everywhere_. Silent deviation is banned. Replace verdicts, and any case
where the signals conflict, go to the user as a short evidence summary with a
recommendation. The asymmetry: conforming unnecessarily is cheap, rotting a boundary is
not, so under uncertainty defer; violation requires affirmative evidence. There is no
middle state where new code quietly deviates while old code keeps the old way, except as
an explicit migration phase with a committed endpoint.

## Rule 4: Package extraction — seams, not counters

Extract a package when:

1. It is a **common utility that needs to be shared**.
2. It creates a **clean architectural seam**, especially across layers, where the
   boundary is a clean interface with clear separation of responsibility.
3. It **splits a large parent into smaller components**, also seam-driven.

**The naming tell:** if the split makes names simpler and clearer, extract. The
namespace absorbs context the identifiers were carrying (`task.ConfigService` wanting to
become `config.Service` is the signal firing). Conversely, a package whose contents can
no longer be covered by one sentence of the form "this package protects X" is overloaded
and wants splitting.

**The testability tell:** a good boundary makes the package's functionality easy to test
in isolation. Dependencies arrive by injection at the seam, so tests construct the
package with test-friendly real dependencies without standing up its siblings or parent.
If testing a candidate package still requires dragging in the whole parent, the seam is
not clean; if a tangle of functionality is hard to test where it sits, that is the
extraction signal firing.

## Rule 5: Altitude — where a behavior lives

When a behavior could live at several layers, decide by:

1. **Deep modules.** Which placement produces a simpler interface over a more powerful
   implementation? The placement that deepens a module beats one that flattens it. This
   is the golden rule; most of the others are consequences.
2. **Cross-language dedup.** Logic placed in core is written once; logic placed in
   clients is written four times (Go, TS, Python, C++) and drifts.
3. **Core owns persisted data.** Persisted data structures live in core, and core is the
   source of truth for migrations. Exceptions: session state for driver, console, and
   clients.
4. **The general-purpose seam.** Some infrastructure could build a SaaS CRM (pluto
   buttons, x math and spatial utilities); some is Synnax domain. Boundaries often split
   exactly on this seam; keep it visible when placing new code.
5. **The grid.** Place every new piece explicitly: vertical axis = layer, horizontal
   axis = domain/feature, third axis = general-purpose vs Synnax-specific.

Corollaries of deep modules, useful as tiebreakers: put the decision where its inputs
already live (ship the decision to the data, not the data to the decision); keep the
transport dumb and the edges smart (broadcast plus consumer-side filter over server-side
routing); lower layers export mechanism, upper layers inject policy (a generic host
parameterized by a registry the domain owner provides), and the lower layer never
reaches up.

## Rule 6: UX claims must be grounded

"The operator wants X" is unfalsifiable on its own. Every UX claim names a **role**, a
**concrete workflow moment**, and a **consequence if we're wrong**. The two corpus
moves: scenario prose that grounds a hard budget ("close a valve within 5 ms of
detecting an overpressure condition"), and norm-mapping against the tools users already
know.

The canonical roles:

1. **Test operator** -> runs procedures against live hardware; safety-critical; low
   tolerance for surprise.
2. **Instrumentation/controls engineer** -> configures channels, devices, tasks,
   calibrations; the Console power user.
3. **Test/data engineer** -> post-run review and analysis.
4. **Automation engineer** -> writes Arc, builds schematics.
5. **Admin/IT** -> standard across industries.

Sources, in trust order: **interviewing the user is best**, and interview answers get
persisted into documents or memories for reuse. Then the docs site (the intended mental
model), the integration test suite (encoded real workflows), demo schematics, and web
research on industry tools (Ignition, TIA Portal, Wonderware, LabVIEW) and adjacent
modern tools (Grafana, VS Code, n8n, Retool). Never name customers (NDA); generalize.

## Rule 7: Prior art — mandatory for shapes others have already chosen

External prior-art research is mandatory whenever the design chooses an interaction
model or a data/lifecycle model other products already had to choose:
autosave-vs-publish, tabs-vs-mosaic, migration strategies. Skippable for internal
plumbing with no external analog. Fixed output form: which tools were checked, what each
chose, and one sentence on why we align or deviate. A pointed paragraph, never a survey.
Internal prior art (related RFCs by number) is cited throughout.

## Rule 8: Interview conduct — the `interview` skill, plus design gates

Conduct (one decision per message, keystone-first ordering, the "it's intuition"
build-it-together move, termination, persistence) is owned by the `interview` skill.
Load it before the first question. Design adds these gates on top:

- **Research first means Rules 1 and 2.** No question is asked until the existence
  ledger and the principles-in-force brief exist for the affected area; every question
  leads with an evidence-backed recommendation drawn from them.
- **Broad before narrow.** UX and architectural principle/boundary questions come before
  implementation-specific ones.
- **Termination lands in the deliverable.** Remaining parameters go to the RFC's Open
  Questions tail.
- **Locked means locked.** Persist locked decisions (working doc, then memory). Deliver
  the full implementation of what was locked; sequencing is the session's choice and
  never a question.

## Rule 9: The deliverable takes the house RFC form

RFCs live in `docs/tech/rfc/`, filename `NNNN-YYMMDD-kebab-title.md` (next sequence
number, start date). The corpus spine, sections numbered from 0:

`# 0 - Summary` -> `Motivation` and/or `Vocabulary` -> `Principles` -> `Design` ->
`Implementation Phases` -> `Resolved Decisions` -> `Open Questions`.

Conventions that matter:

- **Resolved Decisions ledger.** Every rejected alternative is recorded with its
  enumerated downsides and an honest "the trade is real" acknowledgment. Inline prose
  rejection right after a decision is the dominant style for smaller calls.
- **"What This RFC Does Not Cover."** Explicit scope exclusion, its own section.
- **Interview-locked register.** Decisions read terse and settled, rejections folded in;
  the open-questions tail is short and bounded, each item a parameter choice.
- **Evidence style.** Claims cite actual code (file:line) and sibling RFCs by number.
  Never reference RFCs from production code comments.
- Tables for enumerable matrices (state × action); prose for rationale.
- Short RFCs are fine when they justify their existence. A plan persisted in the
  codebase takes RFC form; otherwise plans live in session memory.

## Rule 10: Phase boundaries are earned

A phase (or PR) boundary is earned only when it buys one of:

1. **A reviewable unit** -> a reviewer can hold the whole diff's intent in their head.
2. **A green intermediate state** -> the system builds and tests pass at the boundary;
   nothing half-wired.
3. **Risk isolation** -> mechanical or wire changes land separately from behavior
   changes, so bisection points at one culprit.

If a proposed split buys none of these, merge the phases. Split along architectural
seams (identity/wire vs behavior vs UX), never along file counts. Sequence so the
lowest-risk, dependency-unblocking work lands first; migrations are first-class citizens
of the phase list, and a compatibility statement closes the plan.

🚨 **The session's known bias is OVER-splitting.** Default to fewer phases. A boundary
proposed "to keep PRs small" with no reviewability, greenness, or risk argument is the
bias firing. Additive-introduce followed by atomic-cutover is the sanctioned two-phase
shape for migrations that would otherwise be unreviewable, and zero-coexistence cutovers
(no flags, no parallel old/new) are preferred over long coexistence windows.

## Quick reference

| Activity                            | Gate                                                                             |
| ----------------------------------- | -------------------------------------------------------------------------------- |
| Naming any API in a design          | Existence ledger: _exists_ (file:line, read) or _NEW_                            |
| Marking anything NEW                | Search recorded (x/, owning pkg, siblings) + named precedent                     |
| Diverging from a codebase norm      | Justified in blood; absence of a pattern is itself a principle                   |
| Touching any existing pattern       | Classify conform / extend / replace-everywhere; never silent                     |
| Proposing replace-everywhere        | Six-signal evidence summary + recommendation to the user                         |
| Extracting a package                | Shared utility, clean seam, or parent split; naming improves; tests in isolation |
| Placing a behavior                  | Deep modules first; grid placement explicit                                      |
| Any UX claim                        | Role + workflow moment + consequence if wrong                                    |
| Choosing an externally-solved shape | Prior-art paragraph: tools checked, choices, why we differ                       |
| Asking the user anything            | Research done; recommendation attached; ONE question, prose                      |
| Ending the interview                | Remaining unknowns are parameters, not shapes                                    |
| Adding a phase boundary             | Buys reviewability, greenness, or risk isolation; else merge                     |
