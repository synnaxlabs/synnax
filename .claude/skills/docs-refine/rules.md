# Docs house rules

Every page obeys these. They come from user corrections and locked decisions. When the
user corrects something that applies beyond one page, add it here and say so in one
line.

## Contents

- Prose
- Section and page structure
- Editions and clients
- Presentation
- Components and front matter
- Links and moves
- Content patterns that worked
- Content principles from page review

## Prose

- ASD-STE100 Simplified Technical English, as the root `CLAUDE.md` Prose section says.
  Short sentences, one idea each, active voice, second person.
- Sentence case for titles, headings, nav labels, and UI names in prose.
- Never use em dashes. Use "and", not "&".
- Term bullets use `->` as the separator:
  `- [Ranges](/reference/ranges) -> mark a test in time`. Never an em dash or a colon.
- Product names: full name first on a page ("Synnax Core"), short name after ("the
  Core"). Say "the Core" or "a Core", never "server", "node", or "cluster". The docs
  never describe multi-node clusters; they are not supported.
- Console components are lowercase common nouns ("ranges toolbar", "Core selector"),
  with exact singular or plural. Task names are sentence case ("NI digital read task").
- UI labels a reader clicks go in bold and match Console source exactly. Commands in the
  command palette go in quotes ("Create channel").
- Name the thing. No bare "this", "that", or "it" pointing back at a sentence.
- Write for the end user's flow: what they see, what they do, what they get. Leave out
  internal details (retry counts, backoff, probe intervals, storage layout, file
  formats) unless the reader acts on them. A behavior the UI makes obvious, such as an
  update prompt, gets a sentence, not a page.
- Never name a customer or show real customer deployment details.
- Positioning copy (what Synnax is, what a capability is for) comes from Synnax's own
  materials: the website (`site/www`), the README, `llms.txt`, and the blog. Never a
  generic category label such as "a platform for hardware data". Say the concrete work:
  acquire, control, visualize, review, and automate.

## Section and page structure

- The docs never describe multi-node clusters; they are not supported.
- One page per task. No "Manage" titles and no catch-all pages that hold many tasks.
- Exception: a page for one Console component or one hardware task type (line plots,
  schematics, an NI analog read task) holds that component's tasks as verb h2s, then its
  reference. The title stays the component noun. A task that needs more than ten steps
  or its own tutorial gets its own page.
- Each section opens with a concept page. The concept page has a section on what the
  thing is for: short canonical use cases, each linking to the page that shows how.
- Nav labels drop the section noun ("Create"). Page titles keep it for search ("Create
  channels").
- Media sits on the task page that it shows, not on the section overview.
- Capability sections never mention licensing, license limits, or Core start flags.
  Those belong in the Enterprise section. A reference page may link to a CLI flag.
- Desktop is the default reader. Pages lead with the Console, and the Console tab comes
  first.
- The restructured site ships together with Synnax Desktop. Do not hide Desktop pages or
  content behind a flag, and do not write a fallback for a site without Desktop.

## Editions and clients

- The nav (`PageNavNode.edition`) marks whole pages and groups as Enterprise or Desktop.
  Never put an edition in front matter.
- Content that only applies to the Python or TypeScript client goes inside that client's
  tab. The tab already carries the Enterprise badge, so it needs no `Edition.Enterprise`
  block. Repeating a sentence in two tabs is acceptable.
- `Edition.Enterprise` blocks are for Enterprise-only content outside the client tabs,
  such as the users toolbar.

## Presentation

- Videos stay videos. Each video has one purpose the reader needs, is zoomed out enough
  to read the UI, and follows `studio/AUTHORING.md`.
- A block diagram uses `Diagram.Block` (`site/docs/src/components/diagram/`), never an
  image of boxes and arrows drawn by hand.
- Lists, tables, steps, code, and editions follow `design-system.md`.
- A new component or CSS change gets a look in Chrome before the user sees it. Visual
  choices the user has not approved get one real prototype on the real page.

## Components and front matter

- Every page starts with the `mdxOverrides` boilerplate:
  `import { mdxOverrides } from "@/components/mdxOverrides"; export const components = mdxOverrides;`
- A `<Divider.Divider x />` goes before every h2.
- Import components by path. There is no `@/components` barrel. Do not use `client:load`
  or `client:only`.
- Files that start with `_` are partials, not routes.
- Front matter: `title`, `description`, `layout: "@/layouts/Reference.astro"`, and
  optional `heading` (the displayed title when it differs from `title`). Never `prev`,
  `next`, or `edition`: order and edition come from the nav.
- A page behind a flag has `flag:` in front matter and on its nav node.
- Every page is in the nav. `check-site` fails an orphan page.
- Tabs: `Client.Tabs` (slots `console`, `python`, `typescript`, `arc`; `exclude` to drop
  one), `Platform.Tabs` (slots `Linux`, `Windows`, `macOS`, `Docker`). The Arc tab shows
  only in blocks that fill its `arc` slot. A page with an Arc tab lists `arc` in its
  `clients` front matter.
- OS tabs always use `Platform.Tabs` or `Driver.Tabs`, never a hand-built `Tabs` with OS
  names.
- `Note.Note` variants are `info`, `warning`, and `error` only. Use notes rarely.
- `<Video id>` and `<Image id>` resolve to `-light` and `-dark` files on the CDN.

## Links and moves

- Link text names the target. Never put a link on "here" or "page".
- When a page moves, add an exact redirect in `site/docs/astro.config.ts`.
- Code outside the docs links into the docs, and the link check does not see it. Before
  moving a page or renaming a heading, grep the repo for `docs.synnaxlabs.com/<path>`. A
  heading anchor cannot be redirected: keep its slug, or update the code that prints it.
  Examples: the Console docs button, Cesium write errors (`write-data#rule-1`), Python
  client docstrings.
- Never move anything under `/releases/`: the Console updater reads it.

## Content patterns that worked

Use these when they fit. Each came from prior art and the user accepted it.

- A kinds table with a "use when" column (Ignition, Prometheus).
- A properties list: Console label, client field, and whether it can change (Ignition,
  Stripe).
- Worked input and output tables for computed values (InfluxDB).
- Name rule, then convention, then bad and good pairs (Prometheus, Tailscale).
- A limits table (Cloudflare).
- Symptom-first troubleshooting, then an FAQ (QuestDB).

## Content principles from page review

The user's page notes, made general. Apply them in passes over other sections.

- When Synnax often makes a thing for the reader, the task page first says when it does
  and when the reader must make the thing. Tell the reader not to do work that the
  product already does.
- Say when to use each interface, not only how. A task with a Console and a client
  method says which one fits which job (a few things set up once, or data that a script
  writes).
- Keep only the cases the reader acts on. A "when" section names the paths that the
  reader chooses between, and leaves out side facts, even true ones.
- Each way to do the task on a page gets a section, even when the full steps are on
  another page. A short section with one example and a link is enough.
- Each Console task with its own h2 gets a video of its steps. Request it from the media
  session when it is missing.
- A link to a client-only page says that it uses the Python or TypeScript client.
- Use `Client.Tabs` only when the Console and the clients do the same job (create a
  channel). When the jobs differ (find a channel to plot it, or retrieve it in a script)
  or a task exists only in code, give the code part its own h2 with "in code" in the
  name, and open it with one sentence on why a script does it. Model:
  `channels/find.mdx`. Pages to check: ranges find, data, and aliases; hardware tasks;
  control authority; Enterprise users.
- Arc is a client. When an Arc automation does the same job as the other tabs, give the
  block an Arc tab with one short example. Patterns that need several worked examples go
  on their own page in the section, linked from the tab. Model: `ranges/create.mdx` and
  `ranges/arc.mdx`.
