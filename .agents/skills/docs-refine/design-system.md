# Docs design system

Each kind of content has one form, and each form has one look. Choose the form by what
the content means, not by how it should look. The model page is
`/reference/control/authority`. Check every change in Chrome on the dev server.

## Foundations

### Column and rhythm

- Every block sits flush with the prose column: code, tab panes, tables, lists,
  diagrams, media, and notes. No inset, no bleed. The one exception is a hero, which
  bleeds 5rem on wide screens.
- A diagram never widens the column. Below 560px, a block or nest diagram switches to
  its narrow layout: block diagrams flow from top to bottom, and a rank too wide for a
  phone breaks its labels onto two lines.
- When a block has a lead-in sentence, the sentence ends with a colon and sits 2rem
  above the block. A table keeps 4rem above it, because its header is text. A block has
  4rem below it.
- A heading owns the gap below it. The block under a heading has no top margin.
- A block that scrolls (a table) takes its margin on its wrapper, never inside it.
- A `<Divider.Divider x />` and 12rem go before each h2. An h3 gets 8rem.
- The text column is at most 122rem (732px). The sidebar shows from 1024px, and below
  that the header menu opens the page list. The outline shows from 1280px. The outline
  hides first, so the text never drops below 640px on a desktop.

### Type roles

Use only these four.

- **Body** -> 15px, line height 175%, weight 350, `--docs-body-color`. Prose, list
  items, values.
- **Title** -> body size, weight 550, gray-l11. The name of an entry: a rule, a field, a
  stacked table entry, a problem.
- **Label** -> small size, weight 500, gray-l8. Names a value and is never the value:
  table headers, stacked labels, meta lines, legends.
- **Mono** -> Geist Mono. Code, identifiers, values, and the numbers in markers.

Headings sit above these roles: the page title is 42px, h2 28px, h3 21px, and h4 18px,
all at weight 550 in the text color (gray-l10), never gray-l11. Size and weight lead the
section; a brighter color would pull the eye from the prose.

An icon in text, such as **Arcs** (`<Icon.Arc />`), is 1em and centers on the capitals.

### Color roles

- **Gray-l8** -> markers, labels, and secondary text.
- **Gray-l3 hairline** -> a split inside a block: list entries, table rows, code bands.
  A hairline opens a list, and hairlines split its entries. A block never ends on one.
- **Gray-l4 frame** (`--pluto-border`) -> the edge of a block that has its own surface.
- **Primary** -> links, the state that matters in a diagram, the Enterprise mark, and
  the brand accent: the rule under a page title and the landing use cases.
- **Error and warning** -> the severity dot of a problem entry, and notes. Error also
  marks a required field: the asterisk before **Required**.
- **Arc code colors** (`tokens` in `@synnaxlabs/arc`) -> Arc code blocks, and the type
  (teal) and default (green) of a field entry. Never Lyra primary for a type: primary
  means a link.
- **Warning (amber) and secondary (green)** -> the accents of the outer and inner group
  of a nest diagram: border, hatch, and header icon. Nothing else uses color.

### Markers

- **Dot** -> 5px, gray-l8. One item of a set, in no order.
- **Square** -> 24px, 6px corners, gray-l2 fill and 1px gray-l3 border (the inline code
  chip), mono 12px. 20px next to small text. An order: actions to do, events that
  happen, or parts that come in a fixed order.
- **Index** -> `01`, mono 12px, gray-l7. A number that keeps the reader's place in a set
  that has no order of its own.
- **Eyebrow** -> mono 12px, uppercase, gray-l8, above a heading. A part of a tutorial
  (`STEP 1`).

### Frames

A block gets a frame only when it has its own surface: code, tab panes, media, notes,
disclosures, and the `Diagram.Block` canvas. Lists, tables, and diagrams that draw their
own regions (`Diagram.Nest`, `Diagram.Handoff`) sit on the page.

### Heroes

A hero is the cover picture of a section page or a release: one Console picture on a
two-color gradient, from `Hero.Hero`.

- A section page, and each release from 0.60 on, gets a Studio still. Write a script
  that saves one 1280 x 800 still named `window`, add it to `studio/videos.ts` with
  `hero(id, script)`, run `pnpm batch` and `pnpm upload`, then add
  `<Hero.Hero still="<id>" ... />`. The tag never names a version: a new capture updates
  the page by itself.
- A release before 0.60 frames its existing picture or video with `image` or `video`.
  `framed` shows all of it. `bleed` cuts it at the right and bottom, for media whose
  subject is at its top left. A video is always `framed`, so its controls show.
- A still is `bleed`, `layered`, or `centered` when its subject is at the left of the
  window. A subject at the middle, such as a dialog, needs `framed`.
- A page or release has one cover: the hero at its top. A hero for a feature inside it
  is `inset`, which keeps it in the text column, shorter, and in a faint tint. An inset
  hero takes no `tone` and cannot be `layered`.
- Give neighbors a different palette and tone. `layered` needs a `toolbar` region.

### Navigation

- The breadcrumb, the sidebar, and the previous and next links use the sidebar names, in
  sentence case. An overview page takes the name of its section.
- Previous and next: a label over the page title. The next link aligns right. A link
  into another group names the group in its label: "Next in HTTP".
- The outline title is a Label. At the top of a page, the outline marks its first entry.
  A page loads with its sidebar entry in view.
- Cards: 3rem gaps, and the card name is a Title.
- On a phone, the breadcrumb wraps between segments, never inside one. Nothing on a page
  scrolls sideways except code, tables, and tab bars.

## Forms

- **Items of a set, in no order** -> bullets.
- **A thing, then what it does** -> bullets with `**Term** -> description`.
- **Actions in order** -> steps (any ordered list).
- **The parts of a tutorial** -> step headings.
- **Statements of how the system decides** -> `Rule.List`.
- **Options, parameters, flags, config keys, properties, menu items, controls** ->
  `Field.List`.
- **Many variants, each with its own fields** -> `Channel.Types`, one `Channel.Type` per
  variant, each holding a table by the same rule as the page.
- **Short values across a few items, or a lookup by key** -> a table.
- **A message or a symptom, then its cause and fix** -> problem entries.
- **A question that readers ask** -> an h3 that is the question, then the answer.
- **Control or state that changes over time** -> `Diagram.Handoff`.
- **Parts and how they connect** -> `Diagram.Block` or `Diagram.Nest`.
- **One task in more than one client or OS** -> `Client.Tabs` or `Platform.Tabs`.
- **A risk to data, hardware, or security** -> `Note.Note` with `warning`.

### Bullets

- 2 to 7 items, each one or two sentences. Nest one level at most.
- All items are fragments, or all are sentences.
- A term item is `**Term** -> description` or `[Link](...) -> description`. Never a
  colon or a dash.
- A next-steps section is a bullet list of `[Link](...) -> description`.

### Steps

- Every ordered list renders as steps. Use an ordered list only when the order matters.
- One action per step. A step can hold a code block, a table, or an image under its
  text.
- A finished program goes after the list, under "The finished ...:". Step 1 never holds
  code that a later step adds.
- A tutorial with parts uses `## <Step step={n}>` headings, and each part holds its own
  steps.

### Rule list

- 3 to 8 statements of how the system decides, each one to three sentences long.
- The title is the statement as a short sentence. The anchor comes from the title.
- Text cites a rule by a link with its title, never by its number. The numbers come from
  CSS, so they change when a rule is added.
- Statements with worked examples, or with anchors that code prints
  (`write-data#rule-1`), stay h3 sections.

### Field list

- One entry per option. The first line holds the name (the title), then `type = default`
  in Arc code colors, then **Required** after a red asterisk, as on a Lyra form label.
  The description follows, then the meta line with the other values.
- When an option has a Console label and a client field, the Console label is the title
  and the field goes in the meta line in mono. When one label sets two fields, the meta
  line lists both (**Range** -> `min_val`, `max_val`).
- When the Python and TypeScript names differ, the meta line shows both. When the
  Console label differs from the title, the meta line shows it after **Console**.
- A properties list marks each entry **Can change** or **Cannot change**, unless one
  sentence above the list covers all of them.
- Use it when a description runs past about 100 characters. Options with short values
  and short notes stay a table, whatever the number of columns. Pages of one family (the
  Arc standard library, the NI tasks) use one form.

### Tables

- A matrix or a lookup: at most 5 columns, and each cell holds a value, a phrase, or a
  link. A matrix whose cells are a mark or a word or two (roles by resource,
  integrations by platform) can have more.
- A cell holds at most about 100 characters: a value, a phrase, or one or two short
  sentences. When cells need more, use a field list or problem entries.
- A lookup by key stays a table, even when it reads as terms: keyboard shortcuts, OS
  paths, units.
- The first column names the item and aligns left. A name of 24 characters or less stays
  on one line. Numbers in other columns align right. No frame.
- The first column is never bold, even for a Console label or a key. Its position and
  color mark it as the label, and the column header says what the items are.
- On a phone, a table with at most 3 columns and short cells stays a table. Other tables
  stack into entries, and an entry leaves out a field with no value.
- Where one integration runs is two labeled lines, **Supported** and **Not supported**,
  never a table of one row. `<Platforms integration="..." />` draws them.
- Never put a table in `Details` to hide its width or its length. Variants go in
  `Channel.Types`, and a page with one shape has one table.
- A settings table has the columns Setting, Default, and Notes. A setting is its Console
  label with its code field in `<small>` under it. A field with no Console label is the
  `<small>` field alone.

### Problem entries

- One entry per message or symptom. The title is the exact message in mono, or the
  symptom in words. Then **Cause** and **Fix**.
- An error or log message is the whole title, in code, so a reader who searches for it
  finds it. A symptom may quote interface text: The top bar shows “Reconnecting”.
- A fix with one action is a sentence. A fix with two or more actions is steps.
- Several causes, each with its own fix: **Cause** is one sentence for what they share,
  and **Fix** is term bullets, `**cause** -> fix`. Steps would say "do all of these".
- A symptom with no known cause has a **Fix** only. A problem that the system handles by
  itself has a **Cause** only, which says what happens.
- The exact error text, as a code block or tabs, goes under the heading, above the body.
- A reference list of many short entries (compiler errors, task errors) uses
  `Problem.List` and `Problem.Item`, grouped under h3s. Each entry keeps its anchor. The
  troubleshooting section of a task, device, or server page is such a list.
- A troubleshooting page, where readers scan the outline for their error, keeps one
  heading per problem, with `Problem.Body` holding the cause and the fix. The heading
  slug never changes, because code prints some of them
  (`client/troubleshooting#old-core-version`).

### Code

- A standalone block has a frame and sits flush with the column.
- Code in a block is 13px with a line height of 1.5, on every screen size.
- A line of code is at most 80 characters, so a block never scrolls on a desktop.
- In a tab pane, a block is a full-width band between hairlines, and its code lines up
  with the prose in the pane.
- Inline code is a chip. A chip of 32 characters or less stays on one line. A longer
  chip, such as an error message, wraps and keeps its frame on each line. A long path
  that does not fit breaks where it must.
- In a heading or the outline, code is plain mono at 0.875em with the weight of the
  heading. A single token stays whole, and a message wraps at spaces.
- A key or shortcut is bold, the same as a label in the interface: **Ctrl+T**. Never
  code, never an icon. Name both platforms: **Ctrl+T** (**Cmd+T** on macOS).
- A Synnax data type is always code, in prose, tables, and titles: `float64`, a
  `timestamp` channel. A problem symptom marks code with backticks.

### Tab panes and editions

- A tab set is a pane: a header bar with Lyra tabs, then the panel. A `Details` is the
  same pane with a summary bar.
- A channel reference shows one type at a time in one pane. The types are Lyra chips
  (`Tabs.Selector variant="chip"`) at the top of the panel, each with the Console icon
  of its type. When the types are many, groups are the tabs in the header bar, each with
  the icon of its first type. A group is a measured quantity or a signal source (Force,
  Encoder), never a mix. A link to a type, or a find in page, opens its group and type.
- A tab keeps its full label. When the tabs do not fit, the bar scrolls, fades each edge
  that hides a tab, and keeps the selected tab in view.
- In a pane, a code block, a video, or an image is a band across the full pane between
  hairlines. It never draws a frame inside the frame of the pane.
- The Enterprise mark is a small primary label. It shows once, at the right end of the
  pane header, only while an Enterprise tab is open. The same label marks an Enterprise
  page under its title, an Enterprise block above the block, and an Enterprise item in a
  table or a list after its name. Never put it in a tab label.

### Notes

- Use them rarely. `warning` for a risk to data, hardware, or security. `info` for a
  fact that a reader will miss otherwise. Never two in a row.
- `Note.Note` from `@/components/note`: the Lyra note with the icon of its variant
  before the text. The fill is 5% of the note color. Text, links, and code take the note
  color mixed 70% with the body color, so a link in a note is underlined. The icon keeps
  the full note color.

### Downloads

- A download is a row of buttons aligned left, like every other block. The line about
  older versions follows as a normal paragraph. Never center it or add a `<br />`.
