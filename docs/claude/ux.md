# UI and UX

Rules for how the Console, Lyra and Pluto components, the Portal, and the docs site look
and behave. The `ux-work` skill loads this file. A rule holds until the founder changes
it. When the founder locks a new rule, add it here in the same PR as the change.

## Principles

1. **Fix the named problem. Keep the rest.** The grays, radii, type sizes, focus ring,
   and motion are decided. Change a value only to fix a problem that someone named. A
   difference from Radix, Geist, or Linear is not a problem by itself.
2. **Exact decisions, repeated everywhere.** The best products have the same layouts.
   They differ in small values that they apply the same way on every screen. Use an
   existing token, size, or component before you add a value.
3. **Copy conventions, never signatures.** Use the conventions of the category: keyboard
   grammar, the command palette, the layout skeleton. Never copy the signature of
   another brand, such as the shadcn default look or the glow and glass of Linear's
   site. Put new ideas only where Synnax is different: the live plot, the schematic, and
   the mosaic workspace.
4. **Vibrant color is a Synnax signature.** Plot lines, diagram elements, and status
   colors keep their hues. No rule mutes or reserves color.
5. **Precision in the information, flair in the atmosphere.** Data, forms, and controls
   are exact and calm. Character goes in backgrounds, heroes, and start screens.
6. **Never industrial.** Synnax UI is software UI. Do not frame it as an HMI or SCADA
   screen, and do not use their standards as prior art.

## Foundations

### Gray steps

Code asks for gray by step number (`color={9}`, `var(--pluto-gray-l9)`). Each step has
one job. Pick the step by its job:

| Steps        | Band    | Jobs                                               |
| ------------ | ------- | -------------------------------------------------- |
| l0, l1, l2   | Surface | canvas; panes and cards; dialogs, menus, toolbars  |
| l3, l4, l5   | Fill    | rest; hover; press                                 |
| l6, l7, l8   | Border  | separator; control border; strong border           |
| l9, l10, l11 | Text    | secondary text and icons; body text; emphatic text |

Hover moves one fill step from rest, and press moves one more. Disabled text is
`--pluto-text-disabled`. A disabled outlined control (an input, a select trigger, an
outlined button) has fill l2 and border l4, and its caret has the disabled text color.

### Shape and size

- Lyra's rem is 6 px. Write sizes in rem with the existing tokens. Round a fractional
  rem to a whole pixel with `round(..., 1px)`.
- Radius has five steps: tiny 4, small 6, medium 9, large 12, and huge 18 px. Outlined
  and text buttons, inputs, and keycaps use tiny. Filled buttons, tabs, tags, and
  tooltips use small. Modals and context menus use large. Do not add a step.
- A custom icon looks precise, never cartoony: about 1.5-unit strokes on a 24 grid,
  square ends, corner radii of 1 unit or less, and filled shapes cut by clean gaps.

### Type

- Levels: h1 33, h2 27, h3 21, h4 16, h5 14, p 13, small 12, and tiny 11 px. Headings
  are weight 500, h5 is 450, and p, small, and tiny are 400.
- Small text is 12 px: field labels, help and error text, table headings, toast
  descriptions, small buttons, and context menus.
- Tiny text is 11 px, only for small extras: times, keycaps, tags, counts, and task
  subtitles.
- A button label at the p or small level is weight 450. Inputs, switches, list items,
  menu items, tabs, and select triggers keep the weight of their text. Do not change any
  other weight, and do not change font smoothing.
- Numbers that change, or that people compare, use `font-variant-numeric: tabular-nums`.

### Focus and motion

- The focus ring is `outline: 1px solid var(--pluto-primary-z)` at its current offset.
- Menus, dropdowns, dialogs, and the palette keep their motion. Use the durations that
  exist: 0.1 s for dropdowns and menus (`--pluto-dialog-animation`), and 0.15 s for
  modals (`--pluto-modal-animation`) and button colors (`--pluto-btn-transition`). Do
  not add a duration or a curve.
- Live data values never animate.

## Components

### Buttons

- Filled is the one primary action of a region: a footer, a card, or a form. Outlined is
  a secondary action next to it. Text buttons are for toolbars, inline actions, and
  menus.
- An icon-only button has a tooltip that names the action in a word ("Close"), then
  shows its keys.

### Keycaps

- Tooltips and buttons show a shortcut as one chip: 18 px tall, a faint fill, a faint
  hairline, 11 px glyphs at gray step 9, and 3 px between keys. Inside a tooltip, the
  fill is one step darker. On a filled button, the chip mixes the label color.
- Menus and the palette show a shortcut as plain glyph text at gray step 8 ("⌘E"), with
  no chip.
- The label comes first, and the keys follow it. Word keys start with a capital (Esc,
  Tab). On macOS, modifiers follow Apple's order: ⌃ ⌥ ⇧ ⌘. Windows and Linux spell them
  out: Ctrl, Alt, Shift.
- A modal shows its commit shortcut in its primary button ("Create ⌘↵"), never as a hint
  in the footer.

### Modals

- Header: an icon tile, the breadcrumb, and the close button, with no divider. The last
  breadcrumb segment is the brightest. The close tooltip says "Close" and shows the Esc
  cap.
- Body: fields are 18 px (3 rem) apart, with labels above them. Help and error text sits
  in that gap, so a field never moves when an error shows.
- A modal that creates a named object starts with a title field: a bare heading input
  with no label, and a placeholder that names the object ("Channel name"). It sits 24 px
  (4 rem) above the other fields.
- Footer: no divider. Cancel, then the primary button, at the trailing edge. The leading
  edge holds only Back, a mode switch ("Create more"), or a status summary.
- Esc closes the modal from any field. Validation never changes the status of the
  primary button; errors show at their fields.
- Every step of the device connect flow is 930 px (155 rem) wide
  (`PlatformDevice.ConnectFrame`), so the modal keeps its size between steps.
- A confirmation is narrow. Its title names the action and the object ("Delete
  channel"), its body states the result once, and its button is the verb ("Delete").

### Forms

- A required mark (the red asterisk) shows only on a field that a person can empty and
  that the form refuses when empty. A switch, a checkbox, a select with a default, and a
  number with a default never have a mark.
- Use a switch only when the change applies at once, as in toolbar properties and live
  preferences. In a form that commits on a button, use a checkbox. A footer mode switch
  ("Create more") stays a switch, because it changes the dialog at once.
- A checkbox in a form has its label above it or in the label column, like every other
  field. Its box starts at the edge of the label.
- When flags pick one of a few kinds, show one button selector with an icon per kind,
  never a checkbox per flag. The Create channel dialog uses Data, Index, and Virtual.
- A label is a noun phrase in sentence case. It is never a question, and it never
  changes with state: "Index channel", not "Is index".
- An error message is a sentence about its field ("Name is required"), never a raw
  schema message.
- Fields that belong together share a row: host and port, username and password, the
  timing settings. A row is a `Flex.Box x` of two or three fields in a stacked
  `Form.Sections`. Its first label stays in the label column, and its later fields line
  up with the later fields of every other row.
- A row never starts with a checkbox. In a row of three, the later two fields are short:
  a number or a duration. Wide later fields squeeze the first control.
- A number or a duration field is 144 px (24 rem) wide wherever it sits. A form can set
  another width when its values need one.
- A dialog shows its settings in sections, not tabs, while they fit with no scroll. The
  Modbus connect dialog has no tabs.
- A field in a row, or beside another field, has no help text. Its control shows the
  error, and the message shows on hover. A control with a border gets a red border. A
  control with no border, such as a text select or a button group, gets red text.
- A span of time (a timeout, a delay, a window) is a typed duration field
  (`Form.TimeSpanField`, or `Input.TimeSpan inline`), never a number with a unit suffix.
  The person types "500ms", "2.5s", or "1min 30s". A number alone takes the unit of the
  stored value. The readings and a hint show under the field while it has focus, also
  when it already has a value. A bound is a message under the field ("Timeout must be
  1min or less"), never a silent clamp.
- A range duration on the timeline keeps the popover editor (`Input.TimeSpan`), because
  it reads as a value, not as a form field.

### Task forms

- Task-wide settings (Data saving, Auto start) are toggle buttons in the task footer,
  next to the start button. Each shows its icon and "On" or "Off", and its tooltip names
  the setting. They stay, read-only, when the footer hides the start button.
- The header stays compact: the rack name hangs under the utility buttons and adds no
  height.
- A disabled channel dims to half strength. Its enabled dot stays at full strength, as
  the way to enable the channel again.
- A new task takes the only device of its make. With two or more, the device stays
  empty.
- Port banks sit side by side in columns eight ports wide, each under its title. They
  wrap to one column in a narrow pane.
- A port field that lists an accessory's ports, such as the 80 that a Mux80 adds to a
  LabJack T7, shows one bank at a time. A tab for each bank names its connector and its
  port numbers ("X3 48–71"). The tabs wrap to a second line in a narrow pane. The field
  opens on the bank of the channel's port, and the Ports view shows every bank.

### Overflow

- Cut text fades at its end (`Text` with `overflow="fade"`) everywhere: trees, tabs,
  selects, tags, toasts, table cells, and titles. No "…" and no cut in the middle. Text
  with `lineClamp` fades the end of its last line. A cut input value fades until the
  input takes focus.
- A cut name shows its full text in a tooltip on hover, only when it is cut.
  `Tooltip.Overflow`, mounted once, does this for every `overflow="fade"` label. An
  element's own tooltip wins over it.
- A name under edit stays inside its box and scrolls to the caret, with no fade.
- Prose wraps with `overflow-wrap: anywhere` and is never cut: modal text, toasts,
  confirmations, errors, and tooltips.
- A number never clips, because a clipped number reads as a different valid number.
- A flex child that holds cut text needs `min-width: 0`.

### Menus and selects

- Context menus use 12 px text in 24 px items.
- A divider separates groups of actions, never single items.
- In a group, all items have an icon or none do.
- An open select marks the current value and opens scrolled to it.
- A field in a list item uses the `shadow` variant: no box at rest, a border on hover,
  and a box while open. Fields in forms and detail panes keep their box.

### Lists with checkboxes

- Explorers and record lists use `Select.Item` with `checkable`. Only the checkbox, a
  Shift click, or a Cmd click selects. A plain click runs `onActivate`, or does nothing.
- The selection can be empty. The checkbox shows on hover and stays while selected.
- The checkbox has no gap after it. Its own padding spaces it from the content.
- Hover tints the item faintly, and a press does not deepen the tint.

### Record tables

- A task's channel list is a record table: one item per channel, built from
  `Task.Views.ListChannelItem`. The task supplies only its own fields.
- A record table names its columns in a band above the items. Explorers have no
  headings.
- Declare the columns once with `List.Columns`. Each item lays out its own grid, so give
  each column a width that does not follow its content, such as `30rem` or `1fr`.
- A heading over a field lines up with the field's text, not its box.
- A record table that does not fit its pane scrolls sideways, and the headings scroll
  with it. Fields never shrink to fit: a cut value can read as another value.

### Table input

- A ruled grid: the cells join into one box with shared lines, and headings sit in the
  box. Row labels and remove buttons sit outside it. A cell draws its border on the grid
  line, so focus and hover frame the whole cell.
- Every cell can be edited at all times. Empty and filled cells look different at rest.
- The add button sits under the last row and names the item ("Add header"). It moves
  focus to the first cell of the new row. An empty table shows only the add button.
- Every row shows its remove button at all times, at the end of the row. A remove moves
  focus to the row that takes its place.
- Enter moves down, Shift+Enter moves up, and Tab moves right.
- Esc puts back the value the cell held when it took focus. A second Esc leaves the
  cell. Esc never commits typed text.
- A cell with an error has a red frame, and its message shows on hover.
- Paste keeps the values that parse. A cell whose text does not parse keeps its value
  and shows the error until an edit. A first row in which no number parses is a heading,
  and paste drops it.

### Status and toasts

Follow the status text rule in `AGENTS.md`. The name holds the object. The description
says why in a sentence, then gives the raw error.

### Code blocks

- Code to copy uses the colors of the docs site: keywords `--editor-token-keyword`,
  strings `--pluto-secondary-m1`, functions `--pluto-primary-z`, and comments
  `--pluto-gray-l7`.
- The copy button sits at the top right, centered on the first line. A one-line block
  centers its line in the box.

### Plot ranges

- Ranges are bars in a strip above the plot. Overlapping ranges stack in lanes by start,
  longer first, so a parent sits above its children. Three lanes are 20 px and show the
  name. Up to four more are 4 px, with no name.
- A bar is its range color tinted over the background, with a 1 px border in the color.
  The name is in the range hue, centered on its cap height, and fades at its end. A side
  cut by the plot edge has square corners, and the name stays at the edge.
- In the plot, each range draws its start and end lines. Only a range that holds no
  other range gets a fill, so fills do not stack. A range with no color is not drawn.
- Hover fills the range, dims the others, and shows its name, start, end, and duration.
  The tooltip wraps a name past 240 px.
- A click on a range zooms to it and pauses the plot. A second click on that range goes
  back to the view and pause state from before. If the view changes between the two
  clicks, the second click zooms in again. Clicks nest, one level per click.

## Interaction

- Respond to every input in less than 100 ms. Show a spinner only after 150 to 300 ms,
  and keep it for at least 300 ms. A plot draws its axes at once and streams data in.
- Every command that a key runs also has a button or a menu item.
- Hover can show rare actions, never the current state. A control that shows on hover
  keeps its space, so the layout does not move.
- Focus never gets lost. After an add, focus goes to the new item. When a dialog closes,
  focus goes back to where it was.
- Dialogs, drawers, and dropdown lists set `overscroll-behavior: contain`.
- An empty state has one primary action, named as a verb and a noun ("Create channel").

## Prior art

Study these first. Take their conventions, never their signatures.

| Source                                                                                                                        | Study it for                                       |
| ----------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| [Linear](https://linear.app/now/behind-the-latest-design-refresh)                                                             | restraint, keyboard flow, the create modal         |
| [Radix Themes](https://www.radix-ui.com/themes/docs/overview/getting-started), [Geist](https://vercel.com/geist/introduction) | exact component specs, one job per color step      |
| [Primer](https://primer.style/product/)                                                                                       | forms, dialogs, validation                         |
| [Apple HIG](https://developer.apple.com/design/human-interface-guidelines)                                                    | menus, button order, shortcuts, switch or checkbox |
| [Raycast](https://www.raycast.com), [Superhuman](https://blog.superhuman.com/how-to-build-a-remarkable-command-palette/)      | the command palette, keycaps, speed                |
| [VS Code](https://code.visualstudio.com/api/references/contribution-points)                                                   | commands, tabs, a dense workspace                  |
| [Vercel guidelines](https://github.com/vercel-labs/web-interface-guidelines)                                                  | loading, focus, numbers, scrolling                 |
| [NN/g](https://www.nngroup.com/articles/toggle-switch-guidelines/)                                                            | research on switches and response times            |

## Open decisions

Do not change these without the founder: the dark theme surfaces, the plot frame, tabs
joined to their panel, the Portal text and control scale, the docs reading numbers, the
typeface, and the icon set.

## Docs pages

The `docs-refine` skill owns the forms and rhythm of docs pages, in its
`design-system.md`. The rules here cover what the docs share with the Console.

- A picture of the product shows the current Console: a Studio still from
  `site/docs/src/components/hero/stills`, never the screenshot on the www landing page.
- Gradient blooms stay quiet. The Desktop band at the end of a blog post has a 30% blue
  bloom over `#181b27` in dark mode. A 70% bloom over `#282c41` was too intense, and a
  7% tint over the page surface was too flat.
