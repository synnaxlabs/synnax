---
name: controversial
description:
  Find the five most controversial decisions in a logical block of changes (a PR, a
  diff, or a set of PRs), judged against this codebase's architecture and standards, and
  present them one at a time for a ruling. Use when the user invokes /controversial or
  asks which decisions in a change a maintainer would dispute. Covers significant added
  complexity, violated or extended architectural boundaries, new architectural patterns,
  and choices that go against the status quo. Not a bug hunt; `code-review` owns
  correctness.
---

# Controversial decisions

Find the choices in a change that a maintainer of the touched area would want to argue
about, and give the user what they need to rule on each one. A correct decision can
still be controversial. The goal is not to prove a decision wrong. The goal is to make
sure that a person made it on purpose.

The most useful results are structural choices that the PR never mentions.

This skill only reads. It changes no code and posts nothing to GitHub unless the user
asks. `code-review` asks if a change works. `eb-review` audits small work from this
session. This skill asks which choices need a ruling.

## What counts

A candidate must pass every gate:

1. **The block made it.** The change introduces the decision. Pre-existing code is out.
2. **A real alternative exists.** Name one that works in this repo. If only one answer
   is right, the candidate is a bug or a fix, and it belongs to `code-review`.
3. **It changes the shape.** It affects structure, dependencies, interfaces, stored or
   wire shapes, or how the next feature gets built. Style, naming, comments, test
   coverage, and PR size are out.
4. **Someone pays.** Name what breaks, degrades, or gets harder to reason about, and who
   carries that cost. Being different is not enough.
5. **It is not settled.** Drop a candidate that disputes a rule or a Resolved decision
   in an RFC. Keep one where the change goes against a rule or a Resolved decision.
   Report a breach of an `AGENTS.md` ban, but do not debate the ban.

## What to look for

Use size and churn only to choose where to read. They are never a finding.

**New patterns.** The most common dispute in this repo.

- The first use of a mechanism: an `init()`, a registry, a new dependency, a new way to
  hold state.
- A second way to do a solved job: "solve the same problem 3 different ways" (#3022).
- A local convention with no agreement behind it: "introducing a new pattern here
  without a shared agreement" (#2678).

**Boundaries.**

- Logic at the wrong layer, against `docs/claude/architecture.md` or a component
  `AGENTS.md`.
- A dependency that points up or across: `x/` imports the Core, a new cycle, a sibling
  feature import.
- Surface widened for convenience: "went public only so `test/driver.ts` could call it"
  (#2696).
- One concept that now spreads across more languages or components.

**Complexity.** Measure it: lines, exported symbols, packages, config fields.

- Nothing asks for it: "What problem is this solving for our users?" (#2864).
- Speculative generality: an interface, option, or flag with one user.
- A shallow layer that only forwards calls.
- New state, concurrency, or lifecycle that every later reader must learn.
- Work pushed onto others: packages that must migrate or adapt.

**Against the status quo.**

- A stored or wire shape, a migration, or a generator change.
- An exception to a written rule.
- An alternative that an RFC already rejected.
- A removed guard or behavior with no stated reason.
- A new observable behavior that callers can come to depend on.

## Process

1. **Set the scope.** If the user names nothing, use the PRs this session owns, or else
   the current branch. Find the true base and recover renames (`code-review` Rule 1).
   For a set of PRs, read the whole set as one diff and attribute each item to its PR.
   Skip generated output, but read its source, such as the `.oracle` schema.
2. **Read the code first.** List candidates from the diff before you read the PR
   description, commit messages, or RFC. The author's framing changes what a reviewer
   sees, and it hides the choices that the author did not notice.
3. **Load the written rules.** Read the sources that `code-review` Rule 4 names for
   every touched area, plus `docs/claude/architecture.md`. Never work from memory of a
   rule.
4. **Count the status quo.** For each candidate, search the repo and state a count: "11
   of 12 services do X". Then judge if the pattern is load-bearing or incidental with
   the six signals in `design` Rule 3. A broken incidental pattern is a weak candidate.
5. **Read the author's side.** Now read the description, the commits, the RFC, and the
   review threads. Drop what the gates now fail. A decision that the PR never mentions
   stays strong, and one that the PR defends well gets weaker.
6. **Get a second opinion.** Send the candidates and their files to one fresh subagent.
   Tell it to argue for the author, check each count and citation, and look for the
   constraint that forces each choice. Drop a candidate if that constraint holds. Do
   this for your own work too.
7. **Rank.** Keep the top five. Use the order below.

Disputes in this repo ended with no code change in two cases: the author named a
constraint that the reviewer did not have (#2357, a resize that fires on each frame of a
drag), or the cost was private and easy to undo ("a two way decision", #2678). The code
changed when the reviewer named an in-repo alternative or a written rule, so bring one.

## Ranking

Sort by each key in order. Use no scores. A numeric score only looks precise.

1. **Reversal cost.** A shipped stable format or public API, then a required migration,
   then a plain revert. Formats that only candidates shipped are easy to reverse.
2. **Precedent.** The next author can copy it. A first use or a rule exception ranks
   high.
3. **Blast radius.** Packages, callers, and languages, counted.
4. **Who pays.** Other people before the author.
5. **Evidence.** A cited rule, then a counted pattern, then judgment.

Five is a limit, not a target. Never pad. Zero is a valid result.

## Present

Follow the `itemize` skill: one item per message, then stop for a ruling. Lead with the
top item. Frame each item as a trade, not a defect.

The orientation line gives the number of decisions, the number of candidates that you
dropped, and what you did not examine. Example: "Three decisions from 11 candidates,
biggest first. Not examined: generated Oracle output."

Each item uses this form, in plain language:

```markdown
**1 of 3: <the decision in one short line>** (`file:line`, #PR)

**Decision.** Facing <concern>, the change chose <choice> over <alternative>, to get
<goal>, and accepts <downside>.

**Status quo.** <The count, and the rule or RFC, if any.>

**For it.** <The strongest honest case, in the author's terms.>

**Against it.** <Who pays, and what the next author can now copy.>

**Undo.** <What reversal costs today, and after a stable release.>

**Recommendation.** <Keep, change, or record>. <Why.> Confidence: <low, medium, or
high>. <What would change this view.>
```

The three recommendations:

- **Keep:** the decision holds. Name the test or lint that should pin it, if any.
- **Change:** name the alternative and its scope in files and lines.
- **Record:** the choice can stand, but it needs a written reason in the PR, an
  `AGENTS.md`, or an RFC.

The ruling belongs to the user. Record each one. After the last item, list the rulings
in one line each. A clean result is not a claim that the change works.
