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
