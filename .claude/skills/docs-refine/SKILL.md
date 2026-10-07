---
name: docs-refine
description:
  Process and hard rules for refining, rewriting, and upgrading one section of the
  Synnax docs site (`site/docs`) page by page. Use when asked to work through, polish,
  improve, restructure, or rewrite a docs section or page. Researches code facts, the
  current pages, and prior art, then drives three user gates (section charter, page
  brief, written page review), writes the pages, verifies every claim against source,
  re-records videos, and edits whatever the pages need.
---

# Docs refine

The job is to bring one docs section up to par: the right pages, each with one clear
responsibility, every claim true against the code, and the best presentation the site
can give it. The skill edits and upgrades. It is not a review tool: when a page needs a
new component, a CSS fix, a moved paragraph in another section, or a re-recorded video,
the session does that work.

The user steers structure at three gates. Everything between the gates runs without the
user.

## 🚨 THE INTERVIEW RULE 🚨

EVERY QUESTION TO THE USER FOLLOWS THE `interview` SKILL. LOAD IT BEFORE THE FIRST
QUESTION. ONE QUESTION PER MESSAGE, PLAIN PROSE, A RECOMMENDATION WITH EVIDENCE, NEVER
AskUserQuestion. PLAIN WORDS: NO FILE:LINE, COMPONENT NAMES, OR INTERNAL JARGON IN THE
QUESTION ITSELF. A QUESTION THE USER CANNOT PARSE IN ONE PASS GETS `/restate`.

## Read first

1. `rules.md` in this folder: the house rules for docs content. Every page obeys it.
2. `page-types.md` in this folder: the checklist for each page type.
3. `design-system.md` in this folder: the one form and look for each kind of content.
4. The section memory file `project_docs_<section>_section.md`, if it exists. It holds
   the locked decisions and verified facts from earlier runs. Never re-ask a locked
   decision.
5. `site/docs/CONTRIBUTING.md` and the Prose section of the root `CLAUDE.md`.

## The process spine

1. **Research** the whole section once (below).
2. **Gate 1: section charter.** Wait for approval.
3. For each page, in nav order:
   1. **Gate 2: page brief.** Lock its shape decisions one question at a time.
   2. **Write** the page, re-checking each claim against code as it goes.
   3. **Verify** (below).
   4. **Gate 3: review.** Give the dev URL and wait for the user's reaction.
4. **Close out** the section: report, memory status, media upload.

## Research

Start three read-only agents in parallel, then spot-check their key claims yourself
before the charter.

1. **Code facts.** For every behavior the section describes: data model, fields and
   whether they can change, defaults, limits, name rules, errors, what the Console shows
   (exact button, menu, command, and dialog labels from Console source), what each
   client does. Each fact with `file:line`. Include wrong claims found on the current
   pages.
2. **Docs audit.** The current pages, what `main` or older versions had that the current
   pages dropped (`git log` and `git show <rev>:<path>`), content about this section
   that lives in other sections, inbound links to the section, and the media inventory
   (every `<Video>` and `<Image>` id, with its studio script if one exists).
3. **Prior art.** Fetch real pages from comparable products (for example Ignition,
   Grafana, InfluxDB, Foxglove, Prometheus, QuestDB, Tailscale, Stripe, Supabase,
   Docker, Home Assistant) for the same section's topics. Report the patterns that
   transfer: page sets, tables, examples, ordering, what each page owns. Cite URLs.

Start the section memory file when research is done (template below). Record the
verified facts there.

## Gate 1: the section charter

One message, then stop. It holds:

- **What the section is for**: the reader, the questions they arrive with, and what they
  can do when they leave. Two or three sentences.
- **The page list** in nav order. For each page: title, nav label, primary type (from
  `page-types.md`), and one responsibility in one sentence. Mark pages added, split,
  merged, moved, or removed, with the reason.
- **Prior art** that shaped the list, in a compact list with URLs.
- **Wrong claims** found in the current pages, in plain words.

Nothing gets written before the user approves the charter. Changes to the charter go
through the interview, one decision at a time.

## Gate 2: the page brief

One page at a time. The brief holds:

- **Responsibility and type** from the charter, and the h2 outline with the type of each
  section.
- **Keep, cut, add, present better.** What stays, what goes (and where it moves), what
  is missing (use cases, examples, limits, errors, troubleshooting), and what needs a
  better form (a table, a worked example, a diagram, a video, tabs).
- **Prior art** for this page, where it adds something beyond the charter.
- **Design and layout.** How the page looks, not only what it says: hierarchy, what the
  eye lands on first, spacing, columns, and how it reads at phone width. Look at the
  current page in Chrome and at real prior art pages before proposing. Content-only
  briefs are incomplete.
- **Media plan.** Each video and image: keep, change, re-record, add, or remove, with
  the reason the reader needs it. See Media.
- **Open claims** that code reading could not verify.
- **Shape decisions**: then ask them one at a time, under the interview skill. Decisions
  with one obvious answer are not questions: state them in the brief and move on.

## Writing

- Obey `rules.md` and the page type checklist.
- **Claim ledger.** Every UI label, default, limit, error, and behavior statement on the
  page, old text and new, is verified against source with `file:line` in the section
  memory file, never on the page. Old text gets the same check as new text: the old text
  is where the errors were. A claim that cannot be verified goes to the user as a
  question. It is never reworded to sound safe.
- No live Core runs and no manual tests to verify prose. Code reading is the standard.
- Code bugs found along the way go into the report. Do not fix them.
- Edit whatever the page needs: other sections' pages, shared components, CSS, layouts,
  redirects in `astro.config.ts`. When a page moves, add an exact redirect.
- A new component or new CSS gets a look in Chrome (light and dark, desktop and phone
  width) before the user sees it. Routine prose needs no screenshot.
- A visual choice the user has not seen (a badge style, a table form, a layout) gets a
  real prototype on the real page, one problem at a time. Never ship a guess, and never
  build several unseen variants behind a switcher.

## Verify

From `site/docs`:

```bash
pnpm build:vite-plugin                       # from the repo root, once per session
pnpm exec prettier --write <changed files>
pnpm check-site:build && pnpm check-site
```

A failure in a page this session did not touch gets reported, not fixed silently. Filter
network timeouts to external sites, and report them. For component or CSS changes, also
run `pnpm exec vitest run`, `pnpm lint`, and `pnpm check-types`.

## Gate 3: review

Run the dev server (`pnpm dev` in `site/docs`, on the port the user names, else the
first free one from 4321). End the message with the exact localhost URL of each written
page. Then stop. Apply the user's corrections, re-verify, and give the URLs again.

## Media

Videos come from the studio pipeline (`studio/README.md`, `studio/AUTHORING.md`). Each
`<Video id>` maps to a script in `studio/scripts` through `studio/videos.ts`.

- After the user approves a page's media plan, start background agents to write or edit
  the scripts and render the videos. Give each agent the shot description, the reason
  the video exists, the names used in the page examples, and the rules in
  `studio/AUTHORING.md`. One agent at a time: each capture uses the studio Core port and
  the Console dev server.
- Keep the video id when re-recording, so the page does not change.
- Show the rendered videos to the user at gate 3.
- Ask the user for approval before `pnpm upload`. After the upload, confirm each CDN
  file with `curl -I`. The CDN can serve the old file for up to an hour.
- Still images have no capture tool. List them in the brief for the user.

## Learning

When the user corrects something that applies beyond this page or section, edit this
skill at once: `rules.md` for content rules, this file for process, `page-types.md` for
a type rule. Tell the user in one line what changed. Do not ask first. Section-specific
locks go in the section memory file instead.

## The section memory file

`project_docs_<section>_section.md` in the memory directory, with a line in `MEMORY.md`.

```markdown
---
name: project-docs-<section>-section
description:
  "<Section> docs section refine: charter, claim ledger, locks, media plan, status"
metadata:
  type: project
---

Charter (locked <date>): pages, types, one responsibility each.

Claim ledger: <claim> -> <file:line>, one line each.

Locked decisions: **<X1> (<date>)**: <decision>, with the user's words where they
matter.

Media plan: <id> -> keep | re-record | add | remove, reason, status.

Code bugs found: <bug> -> <file:line>.

Status: <which pages are written, reviewed, uploaded>.
```

## Close out

A short report: pages written, wrong claims fixed, code bugs found, media status, and
anything reported outside the section. Update the memory file status.
