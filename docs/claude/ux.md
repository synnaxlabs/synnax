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
| l0, l1, l2   | Surface | canvas; panes and cards; toolbars                  |
| l3, l4, l5   | Fill    | rest; hover; press                                 |
| l6, l7, l8   | Border  | separator; control border; strong border           |
| l9, l10, l11 | Text    | secondary text and icons; body text; emphatic text |

Non-modal dialogs and menus use `--pluto-gray-l1-l2`, halfway between l1 and l2. Modals
keep the modal surface.

A card on an l1 page uses l0, the same as the shell islands. Settings on the project
splash is one: its page is l1, so its nav and cards are l0.

Hover moves one fill step from rest, and press moves one more. Disabled text is
`--pluto-text-disabled`. A disabled outlined control (an input, a select trigger, an
outlined button) has fill l2 and border l4, and its caret has the disabled text color.

### Status colors

Success is primary blue, warning is yellow (amber in light), and error is red. Every
status color is a step of the palette ramps in `lyra/src/theme/theme.ts`, never a hex
from another system. Text on a 15% tint of its status (badges, notes) uses
`--pluto-primary-text`, `--pluto-warning-text`, or `--pluto-error-text`. Each is the
step nearest the status color that reads on the tint: a darker step in light mode, and a
lighter step or the base step in dark mode.

### Series colors

The Core ships 16 series colors for data, in a palette of named colors named "Series"
(`core/pkg/service/color/builtin.go`). Each has a light and a dark value with 3:1
contrast on the canvas of its theme. The first eight keep the old hues and stay apart
under each color vision deficiency. The last eight fill the hue gaps and only pass the
contrast check, so lines past the eighth can look alike to a color-blind user.

A palette lists its colors in the order that a user sets by drag, as Figma lists styles.
A new line takes the first series color in that order that no other line uses.

### Shape and size

- Lyra's rem is 6 px. Write sizes in rem with the existing tokens. Round a fractional
  rem to a whole pixel with `round(..., 1px)`.
- Radius has five steps: tiny 4, small 6, medium 9, large 12, and huge 18 px. Outlined
  and text buttons, inputs, and keycaps use tiny. Filled buttons, tags, tooltips, and
  tabs use small. View tabs, filter chips, and cards use medium. Modals, menus, explorer
  bars, and page sections use large. The Settings section list uses huge, and medium
  when it shows only icons. Do not add a step.
- A custom icon looks precise, never cartoony: about 1.5-unit strokes on a 24 grid,
  square ends, corner radii of 1 unit or less, and filled shapes cut by clean gaps.
- An icon centers on its ink, not its box. A star moves up 3.6% of its box, so it lines
  up with the text and the icons next to it.

### Compound icons

- A compound icon is a base icon with a modifier (add, import, export) or a stack. Draw
  it as one glyph with `Icon.createFromPath`. Do not layer icons at run time.
- The modifier sits at top right: 9 units wide, centered at (17.5, 6.5), with a 2.5-unit
  stem (2 on a line base). The base scales to 0.9 from its bottom-left corner.
- Cut the base 1.75 units clear of the modifier, with no backing disc. Cut with the
  modifier's box, with a circle on a round base, or with the modifier's outline when a
  box removes a small detail. A cut line keeps its round caps. Remove slivers.
- A folder holds its modifier as a knockout. The project icon replaces its top dot with
  the modifier.
- An explorer icon is a diagonal stack: a solid back copy at top right, cut 1.75 units
  clear of the front copy at bottom left.
- An item that adds to a thing that exists uses the plain base icon ("Add to active
  plot").

### Type

- Levels: h1 33, h2 27, h3 21, h4 16, h5 14, p 13, small 12, and tiny 11 px. Headings
  are weight 500, h5 is 450, and p, small, and tiny are 400.
- Small text is 12 px: field labels, help and error text, table headings, toast
  descriptions, and small buttons.
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
- In a resource toolbar header, each create action is outlined (`Toolbar.CreateAction`).
  The other actions in that header are text buttons.
- An icon-only button has a tooltip that names the action in a word ("Close"), then
  shows its keys.

### Top bar

- The right end of the top bar holds what applies to the whole Console. It ends with
  docs ("?"), the Settings gear, then the Core badge, the order Jira and Confluence use.
- The left bar holds toolbars only. Settings is a tab, not a toolbar, so its gear never
  goes in the left bar.
- The islands of a pre-workspace surface, such as login or the project splash, are not
  the top bar. Their right end is the Core island, then the island of the theme toggle.
  On the project splash, the Settings gear shares the island of the theme toggle.

### Toolbar search

- A search action in a resource toolbar header turns the title into a field. The search
  glyph sits where the title icon was, and the term starts at the title text.
- While the field is open, the header holds a toggle that shows the results in their
  groups, and a close button. Esc or the close button ends the search, and the tree
  comes back as it was.
- In the flat layout, each match shows the name of its group at gray step 8. The start
  of each word that the term matches takes the primary color.
- At most 100 matches show. A closing line says when more exist.

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

### Tags and badges

- A tag (`Tag.Tag`) names an object that a person acts on: a label, a range, or a user.
  A person can open, remove, or drag it.
- A badge (`Badge.Badge`) shows a short fact that the program writes about the item
  beside it: "Connected", "Auto", or "Dev3". It is not interactive. It has a color only
  when the color means something.
- An indicator (`Status.Indicator`) is a dot that shows the color of a state, with no
  words.
- A choice between options is a button or a tab, never a pill. Fluent, Polaris,
  Atlassian, and Primer split these roles the same way.

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
- A row never starts with a checkbox, except a checkbox that reveals the field beside
  it, such as Optional with Fill value. In a row of three, the later two fields are
  short: a number or a duration. Wide later fields squeeze the first control.
- A number or a duration field is 144 px (24 rem) wide wherever it sits. A form can set
  another width when its values need one.
- A row of three fits the dialog with no overflow into its padding, so its labels are
  short ("Channel lifetime", not "Secure channel lifetime").
- Two text fields on one line, such as a username and a password, are equally wide. The
  second sits in the control of the first, not in the later field columns. An empty
  field shows what empty means: "Anonymous" for the username, "None" for the password.
- Two number fields on one line, such as Scale and Offset, sit together at the start of
  the control of the first, each 144 px wide, with the form's gap between them. They
  never spread to the ends of the line.
- A dialog shows its settings in sections, not tabs, while they fit with no scroll. The
  Modbus connect dialog has no tabs.
- A section holds a whole topic, so a short topic joins its neighbor: the OPC UA login
  sits in Connection, and its certificates sit in Security. An action that fills a field
  sits beside that field ("Find endpoints" beside Mode), not on a line of its own.
- A value in a dialog field, such as a thumbprint, sits on one line with its actions as
  icon buttons at the end. Each icon button has a tooltip that names it.
- A field in a row, or beside another field, has no help text. Its control shows the
  error, and the message shows on hover. A control with a border gets a red border. A
  control with no border, such as a text select or a button group, gets red text.
- An error about the whole form, such as a refused login, shows in an error note under
  the submit button. The note keeps its space while empty, so nothing moves when it
  shows. It is one plain sentence ("Incorrect username or password."), and it is empty
  before the first attempt.
- A span of time (a timeout, a delay, a window) is a typed duration field, never a
  number with a unit suffix. Use `Form.TimeSpanField` when the stored unit is the unit
  of a number typed alone, and `Input.TimeSpan inline` with its own `unit` when they
  differ. The person types "500ms", "2.5s", or "1min 30s". A number alone takes the unit
  people think in for that field, such as milliseconds for a click delay and seconds for
  a timeout, whatever unit the value is stored in. The readings and a hint show under
  the field while it has focus, also when it already has a value. A bound is a message
  under the field ("Timeout must be 1min or less"), never a silent clamp.
- A range duration on the timeline keeps the popover editor (`Input.TimeSpan`), because
  it reads as a value, not as a form field.

### Auto values

An auto field holds a value that the Console picks until a person sets one: an axis
bound, a log precision, or the line plot X channel.

- The field shows the "Auto" badge (`Input.AutoBadge`) at its end, where a unit sits.
  Violet (`--pluto-auto-z`, `--pluto-auto-text`) means auto. Primary blue stays for
  focus and selection.
- The field never shows a value that it does not use, such as 0 for an auto bound. A
  field with room shows the prompt "Enter a value to override". A narrow field shows
  only the badge.
- A set value removes the badge and shows the Auto button (`Input.AutoButton`): the Auto
  icon in violet, with the tooltip "Auto". The button, or an empty field, puts the field
  back in auto.
- A select whose empty value is an auto mode shows the mode as a value ("Time") with the
  badge, never "Select channel". The mode is a `Select.Item` in the select's `footer`,
  so it stays under the list and the search never hides it. It has a one-line hint that
  ends "Best for most needs." Its `label` leaves the hint out of the trigger.

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
- A task body without a device names the next step. With no device of its make, it says
  so and offers to connect one ("Connect MQTT broker"), then selects the new device.
  Otherwise it says "No device selected" and offers "Select device", which opens the
  device select.
- A task that reads fields of an ICD message, such as MQTT read, lists them in a tree
  under their topic, like a group tree. Each field shows once, in the tree. The topic
  details hold the subscription and the timestamp. A field's details hold its format,
  which saves to the ICD, then its channel. A field that the task does not read is
  struck through, and its dot starts reading it. Rejected: a list of topics with a
  channel table, and a fields table in the topic details, which showed each field twice.
- A topic that reads several messages holds a node for each, with its fields beneath. A
  message's details hold its name and match, from the ICD, then its timestamp.
- A topic that reads one message picks it in **Message format**, from the JSON messages
  of every ICD, each shown with its ICD's name. A new topic's message goes into the ICD
  that most of the task's topics read. A pick deletes the message that the topic read
  before when it has no fields and no other topic reads it, so new topics leave no empty
  messages behind. Rejected: a stored task-level ICD, which repeats each topic's ICD
  key.
- A field's format stays short, so its channel shows with no scroll: Scale and Offset
  share a line, Units shows only for a number type, and Enum mapping is one line until
  it holds a mapping.
- A task form is at most 1000 px wide. A task whose channel table needs more room sets
  `--console-task-form-max-width`. Modbus uses 1200 px.
- Port banks sit side by side in columns eight ports wide, each under its title. They
  wrap to one column in a narrow pane.
- A port field that lists an accessory's ports, such as the 80 that a Mux80 adds to a
  LabJack T7, shows one bank at a time. A tab for each bank names its connector and its
  port numbers ("X3 48–71"). The tabs wrap to a second line in a narrow pane. The field
  opens on the bank of the channel's port, and the Ports view shows every bank.
- As the Arc task controls get narrow, the Driver name keeps its space longest. Below
  450 px, the performance select and redeploy show only their icons. Below 280 px, the
  Driver select shows only its icon too.

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

### Pickers

- A card that creates something shows the item's own icon in a large icon tile, then its
  name and a short description. Never draw a custom illustration or a preview of the
  result: line art and colored drawings were both rejected.
- A card is a Lyra `card` button, alone or in `Select.Buttons`. It rests halfway between
  gray steps 1 and 2, hovers halfway between steps 2 and 3, and presses to step 3. It
  has a gray step 6 border and the medium radius. Its icon tile has fill step 3 and
  border step 5, so the tile stands out from the card. An outlined button's fill is too
  strong for a card, and its tiny radius too sharp.
- The header is a quiet title (h4, gray step 10, weight 450) over a huge borderless
  search across the content width, with a gray step 8 search icon and a gray step 6
  hairline under it. The hairline takes the primary color while the search has focus,
  because a picker that is not a dialog loses focus, and typed keys then go elsewhere.
- A search disables each item that does not match it and leaves it in place, so the
  layout does not move. The arrow keys and the pointer move one highlight, and Enter
  picks the highlighted item.

### Color picker

- With named colors, two tabs split the picker's width: Library (`Icon.Library`) and
  Custom (`Icon.Sliders`). It opens on Library while the value is linked or absent, and
  on Custom while the value is a fixed color.
- One 1.5rem inset holds the tab boxes, the search icon, the titles, and the swatches.
  Swatches are 21 px with 1rem gaps across and down. Eight across set the width: 38rem.
- Library has the select dialog's search under the tab bar's hairline, with a hairline
  under it. Each palette sits under its title, which ends with the name of the hovered
  color, else the linked one. Swatches have no tooltips.
- A swatch shows the color's value for the current theme only.
- An edit in Custom stores a fixed color and unlinks the value. There is no unlink
  button.
- An optional color puts a square Auto button at the right of the tab bar, on both tabs.
- The saturation and brightness plane has the tiny radius.
- The hex and alpha inputs pack into one control, and the alpha input keeps its opacity
  icon. Hex codes use the code font, so every code takes the same width and fits.

### Menus and selects

- Menus use 13 px text in 28 px items. This includes context menus, the filter menu, and
  its values.
- The filter menu, its submenus, and the pickers of a filter chip use the context menu
  surface: an l6 border, the large radius, 6 px padding, and the overlay shadow.
- Menus and non-modal dialogs share one fill, `--pluto-gray-l1-l2`. A select keeps its
  bordered search and its framed list. In a submenu, the list has no frame and sits on
  the menu surface.
- A submenu opens 100 ms after the pointer stops on its item, and at once on a click or
  ArrowRight. ArrowLeft and Esc close one level. While the pointer moves toward an open
  submenu, the items it crosses do not light or open.
- A submenu opens with its first item level with its parent item, and laps 4 px over the
  edge of the menu on either side. It fades in when it opens first, and it replaces
  another submenu with no animation.
- The item whose submenu is open stays lit (l4). Only one item is lit at a time. A
  highlight follows the pointer with no fade.
- The search of a menu or a submenu looks like the search of a select: the l3 fill and
  an l7 hairline under it. It spans the top of the menu.
- Typing in an open context menu or the filter menu filters its items, as in Linear. The
  first printable key opens a search at the top, with the caret after that key. The
  search is a normal field: a click places the caret. Items whose text does not match
  hide, and so do dividers with no match on one side. With no match, one muted line says
  "No matches". While the search has the focus, the first match is lit, and Enter picks
  the lit item. The arrow keys move into the items, and a key typed on an item goes on
  in the search. Esc closes the search, then the menu. Every menu is at least 32rem (192
  px) wide, filtering or not. Keys typed into a field, such as the search of a picker,
  never reach the filter.
- The filter also searches in submenus. Each word must be in the item or in the names of
  its submenus, in any order, so "copy link" finds Copy › Link. A match shows its icon,
  the submenu name with a caret, then its label. A submenu whose name matches shows
  itself, not its items. A picker, such as Color, shows nothing. A select shows its
  matching options only when it opts in: Labels does, and Parent does not, because range
  names would fill the menu. A key typed in an open submenu, outside a field, closes the
  submenu and goes to the search.
- Every explorer filters through `View.Filter`: one menu item for each property, which
  opens its picker as a submenu. A chip opens the same picker below the chip.
- Values have 4 px corners. A picked value tints, with no checkbox, and picked values
  next to each other join into one highlight.
- A divider separates groups of actions, never single items. Delete and Reload Console
  are the exception: each one is a group of its own, as in every Console menu.
- A resource has one context menu on every surface that shows it. Ranges use the same
  menu in the toolbar, the explorer, the overview, and the plot strip. A surface can add
  items, and the action of a click on the resource comes first.
- An item for an integration, such as "Upload to Sift", shows only while the Core has
  that integration.
- A submenu of targets keeps a target that already has the change, with a check, and
  disables it. Add to plot does this for a plot that shows the range.
- In a selection, an option that only some of the items have shows a minus mark. The
  first click adds it to all of them, and the next click removes it from all of them.
- A picker leaves out a choice that the Core would reject. The Parent picker leaves out
  the edited ranges and every range nested in them.
- A single select whose value can be empty has a "No <noun>" option with the none icon
  (`Icon.None`), such as "No parent". It is pinned under the list, below a divider, and
  it shows during a search. A click on the current value to clear it is not sufficient,
  because users do not find it.
- A select of things users make, such as labels or ranges, creates them in place. With
  an empty search, a pinned "New <noun>" opens the create modal. While the user types,
  it changes to "Create <noun> "text"", which creates the item at once and picks it. An
  item that needs a choice from the user opens the create modal with the text as its
  name: a label asks for its color. The Console never picks a color for the user. The
  option hides when the text matches a name in any case, and after a click until the
  text changes, so a second click makes no copy. It needs create permission. A filter
  never creates. A new parent range covers the ranges that move into it. Under the
  divider, the create option comes first, then "No <noun>".
- In a group, all items have an icon or none do.
- "Copy" puts data on the clipboard. "Duplicate" makes a new resource. A menu never
  calls a duplicate "Copy".
- An open select marks the current value and opens scrolled to it.
- An open `Select.Multiple` and the line plot's Ranges select list their selected values
  first, with no divider: their joined tint marks where they end. The order is set when
  the select opens and holds while it stays open, so an item never moves under the
  pointer. A search lists its matches in their own order. GitHub and GitLab do the same.
  A filter menu keeps its own order.
- The placeholder of an empty `shadow` or `text` multiple select is a text button. A
  click on it opens the select beside it, as a click on a tag does. The placeholder and
  the add button take the select's size, medium by default, so they stand a step above
  the small tags. A list item passes small, so the item keeps one height.
- A tag or count that opens a select rests at l0, and fills l2 on hover and l3 on press.
  Its outline goes from l6 to l8.
- A click on a tag opens its select beside that tag. While the select is open, a change
  to the tags never moves it sideways. It keeps its first x until the window resizes. If
  the tag goes or folds into the count, the select stays where it is, in the window.
- A group of tags that runs short stacks its tags, then folds the last ones into a
  count: the dots of the first hidden tags, then "+N". With no tag shown, the count
  reads "N labels", then "+N" with its dots. Only a group too narrow for that shows
  nothing.
- Only a tag that shows a color dot shows as a dot in the count. Tags with other icons,
  such as channels, fold into a plain "+N".
- An outlined group, such as a select in a form, keeps its outline at every step.
- A field in a list item uses the `shadow` variant: no box at rest, a border on hover,
  and a box while open. Fields in forms and detail panes keep their box.
- A `shadow` select hides its caret at rest and shows it on hover, on focus, and while
  open. The caret keeps its space while hidden, so the value never moves. A select that
  shows only its icon has no caret.
- An option with a description shows the icon beside the name and the description under
  the name, through `Select.ItemDescription`. The dropdown is wide enough for each
  description on one line. Only a narrow window wraps it, and it is never cut.
- A select with a few described options, such as the Arc performance level and the plot
  line sampling, has no search. A user reads the options instead.
- A role select shows each role's icon (`Access.Role.icon`) beside its name, in the
  dropdown and on the closed select. Its floating dropdown stops at 60rem
  (`--pluto-dialog-max-width`), so long descriptions wrap.
- A select that shows only its icon is a square button, never a wide box.
- A device select shows the selected device's status as a badge at the end of the
  trigger, before the caret (`Device.StatusBadge`). The badge says "Connected" or
  "Disconnected" from the status `connected` flag, in the color of the status variant.
  Its tooltip shows the message and the description. A word from the variant was
  rejected, because one variant covers both states.

### Devices

- Status sits at the right end. In a list, such as the device tree, select items, and
  the find view, a device shows its status as a dot (`Device.StatusIndicator`), and a
  Driver shows its heart. A closed device select shows a badge instead (see Menus and
  selects). A list is for scanning, and a chosen device answers one question, so it gets
  words.
- The integration says how a device's location reads (`Device.LocationFormat`). The
  telling part is gray l9 and the rest l7: the slot of an EtherCAT path, the host of an
  address, and the first address of a list with a count. An address hides the schemes of
  its integration, such as `https://` and `mqtts://`, because a scheme never tells two
  of its devices apart. While any part is hidden or cut, a tooltip shows the whole
  location.
- A device tree item is one line: the logo, the name, and the location at the right end.
  The location gives up its width first, down to its telling part, and then the name
  fades. Two lines were rejected for the tree.
- A device select item is two lines in 48 px: the name, then the location 3 px below it,
  with the status dot at the right. It has no logo, because the select lists one make
  and the closed select shows the logo.

### Lists with checkboxes

- Explorers and record lists use `Select.Item` with `checkable`. Only the checkbox, a
  Shift click, or a Cmd click selects. A plain click runs `onActivate`, or does nothing.
- The selection can be empty. The checkbox shows on hover and stays while selected.
- The checkbox sits 12 px from the item's left, top, and bottom edges and 6 px before
  the content (`margin-inline` in `lyra/src/select/Item.css`). The checkbox in a
  headings band sits the same way. Gaps of 9 and 12 px to the content were rejected.
- Hover tints the item faintly, and a press does not deepen the tint.
- A timeline lane is checkable too, and spaces its checkbox as a tree item does. Its
  label gives up the checkbox width, so the bars stay on the time axis.
- The first timeline lane sits 6 px below the time axis, as the first tree item sits
  below the top of its list. The axis lines and now go on through that room.

### Explorer bars

- An explorer has two bars. The first holds the search, then the views. The second holds
  Filter and its chips on the left, and the sort, layouts, and Create on the right.
- The search is 270 px wide. A hairline that runs the full height of the bar separates
  it from the views. Its icon is above the Filter icon.
- Both bars sit in one island: an l1 card with an l5 border and the large radius, inset
  6 px from the top and 9 px from the sides of the panel. An l4 line separates the two
  bars. The second bar's controls are 28 px with 13 px text.
- Each bar is 42 px (7rem) tall. The filter bar grows when its chips wrap.
- The bars have no edit toggle. The search shows on every view. On a static view, and on
  a saved view that the user may update, the user changes the filters, the sort, and the
  layout. Other saved views show the filter chips, the sort, and the layout read-only.
  Create view shows when the user may create views.

### View tabs

- A view is a 28 px pill with the medium radius: an icon, then the name, with a 6 px
  gap. Names are l9. The open view takes the text color, a fill, and an l6 border.
- The add view button sits right after the last view, not at the end of the bar.
- A view has an icon and a color, picked from "Change icon" in its context menu with the
  same picker as panels. The fixed view shows the explorer's icon.
- Create view opens a modal like the project modal: an icon button, then the name. The
  name has no default value.
- The panel stores the view each explorer tab shows, so the view lasts through a reload
  and a move to another window, and everyone who opens the panel sees it. Opening a view
  changes the tab in place. A user who may not change the panel cannot open another view
  in it.
- A panel shows a saved view in one tab at most. Opening a view that another tab shows
  focuses that tab.
- The panel tab name follows the view: the explorer's name ("Range explorer") on the
  static view, and "Ranges › <view>" on a saved view. A rename of the view changes it,
  and a rename of the tab renames the view. When the view is deleted, the tab shows that
  it was deleted, with Close and Restore, like the tab of any deleted resource.
- "Open in new tab" in a view's context menu, or Cmd+click (Ctrl+click on Windows and
  Linux) on a view, opens it in a new explorer tab beside the current one. The current
  tab keeps its view. The saved view the current tab shows has no "Open in new tab".
- Opening the explorer from the sidebar or the command palette focuses its first tab in
  the panel, whichever view it shows, and opens one only when the panel has none.

### Filter chips

- A chip is a row of parts: name, operator, value, and remove. It has an l2 fill, an l5
  border, the medium radius, and l6 lines between its parts.
- A chip is 26 px tall, and each part has 10.5 px of padding on each side. The text is
  13 px.
- The name icon and the operator are muted (l9). The remove part is 28 px wide. Its icon
  is 12 px and l8, and it takes the text color on hover.
- A part that opens something gets l3 on hover and l4 on press.
- The dots of a labels value overlap a little, as one group.

### Record headers

- The header of a record page, such as a range overview, puts its actions on the top
  line, beside the parents. The name takes the full width under them.
- A parent shows its color square, as the explorer does, at 80% opacity, then its name
  at weight 500 in gray l9. A caret the size of the text separates two parents. The
  parents never wrap: the furthest up fades first.
- In a narrow pane, the parents move to their own line under the actions. When the pane
  is narrower still, the copy, download, and link buttons fold into a "⋮" menu. Plot and
  the star stay.
- A word of a time line ("started", "ended") wraps with its time, never alone.
- In a narrow pane, a field of a record shows its value under its name.

### Page sections

- A section of a record page, such as Child ranges or Metadata, is an outlined box like
  the channel box of a task form: an l6 outline, the large radius, an l1 body, and an l2
  header bar 37 px tall with a line under it.
- The header holds one text button with the section's icon, title, and caret. It
  collapses the box to its bar. An action in the header opens the box again.
- An empty section shows no box, only one muted "+ Noun" line that adds its first item.

### Settings

- The section list shows names at 800 px and wider, and only icons below. A Sidebar
  button at its top collapses it to icons and expands it again. The choice is not saved,
  so Settings always opens expanded.
- The list animates over 0.15 s: the width changes, the names and group headings fade,
  and a line fades in where each heading was. A pane that turns narrow snaps to icons.
- The toggle and each icon keep their place in both modes, so the icon-only list is as
  wide as it takes to center the icons.
- The page padding steps down below 100rem and 72rem, as on the range overview.
- An entry's control moves under its label when the label has less than 36rem. In a card
  narrower than 72rem, every control goes under its label.
- A name column is at least 24rem wide, and a short value column 12rem to 16rem. A list
  that does not fit scrolls sideways.
- A button beside a field, such as a create button or "Set password", goes under the
  field when the field has less than 30rem.
- The Profile name field is at most 50rem wide. The license stats stack below 96rem.

### Group headings

- A list that groups its items, such as the palettes on the Colors page or the groups of
  the Ranges toolbar, heads each group with `Header.Group`: one full-width bar that
  opens and closes the group.
- The bar is l2 with the item radius. Its height is the Lyra size nearest the item
  height: `large` above a 40 px item. Hover and press fill the whole bar with a button's
  transition.
- The title is a plain string or a span at the button label weight. The count follows in
  tiny l9 text, then an optional badge. The caret at the end points down when open.
- Actions, such as a "+" that creates an item in the group, sit before the caret and do
  not open or close it. Rename edits the title in place.

### Ranges toolbar

- The toolbar is one list in three groups: Running, Favorites, and Recent. Never call
  the middle group "Pinned". Each heading has an icon: the in-progress stage icon in
  amber, a filled star in goldenrod, and the completed stage icon in blue.
- The heading icon sits over the item carets, and the title over the color marks.
  Collapsed headings are 3 px apart, and the first is 6 px below the header. A collapsed
  group stays collapsed across sessions.
- A line is the caret, the color mark, the name, the muted parent, the time, then Stop
  or the star. The time is the elapsed time in Running, the start in Favorites, and
  "ago" in Recent. The caret's hover box never touches the color mark.
- Running nests each range under its running parent. Favorites keeps star order; with
  none, it shows "Star a range to add it here". Recent sorts by end, newest first. A
  finished child of a running range shows in Recent with its muted parent, and nests
  under the parent when the parent ends.
- Only the star and the Favorite menu item add a favorite. Plot, create, search, and
  links never add one. The plot's Ranges picker searches every range, newest first, so
  no range needs a star to be plotted. A range that has not started shows in the toolbar
  only as a favorite.
- The header holds the explorer, then + (the create form set to the last hour), then
  Start. + and Start are outlined `Toolbar.CreateAction` buttons. The toolbar has no
  active range, and new plots open on Rolling 30s.
- Next shows beside Stop on a running range with a parent. It stops the range, and each
  running range below it, at the click time, and opens a draft below the range. Enter
  starts the new range at that same time; Escape cancels both. Start child shows on
  hover on each running range, opens a draft as the last child, and stops nothing.
- A draft suggests a name from the order of the children of the parent's previous
  sibling, or else adds one to a trailing number. It suggests the color of the matched
  range, and its swatch changes the color.
- The hover card (0.7 s, to the right) shows the name, color, stage, path, start, end,
  duration, a strip of the children in time, the first 6 children, and the labels. It
  has no buttons.
- The menu is the explorer's menu. In the toolbar, Start child and Next work in place. A
  line drags onto a line plot's Ranges field.
- At 46rem the parents hide, and Start child and Next move to a hover cover over the
  time. At 38rem finished times hide. At 34rem live times and the explorer button hide.
  A line without Next keeps its slot, so the times line up.

### Record tables

- A task's channel list is a record table: one item per channel, built from
  `Task.Views.ListChannelItem`. The task supplies only its own fields.
- The Users page of Settings is a record table with checkable items, like the range
  explorer: the name, then a role select in place. A right click opens the menu of the
  selected users. A "You" badge follows the current user's name.
- The Labels page of Settings is a record table with checkable items, like the range
  explorer: a color swatch, then the name. Click the swatch to recolor. A right click
  opens the menu of the selected labels. You create a label from the page or from any
  label field.
- Secrets and certificates are two record tables in Settings, not a toolbar. A secret
  shows where its value comes from and a "Used by" count that opens what uses it. A
  certificate shows when it is valid.
- The Settings lists share `Settings.Table`: search and a filled create button above,
  headings, checkable items, and a right-click menu.
- Create in place when every field you set is a column, such as a label or a secret. The
  create item opens at the top with check and X buttons, and Enter and Esc. It stays
  open and empty after each create, and the new item fades in under it.
- Use a modal when the item needs a field the list does not show, such as a user's
  password, or when the create returns a value to copy, such as an API key's token.
- A record table names its columns in a band above the items. Explorers have no
  headings.
- Declare the columns once with `List.Columns`. Each item lays out its own grid, so give
  each column a width that does not follow its content, such as `30rem` or `1fr`.
- A heading over a field lines up with the field's text, not its box.
- A write task names two channel columns, Command and State
  (`Task.Views.LIST_WRITE_CHANNEL_COLUMNS`). The enabled switch follows the state name.
- A record table that does not fit its pane scrolls sideways, and the headings scroll
  with it. Fields never shrink to fit: a cut value can read as another value.
- A field whose glyph shows its value, such as the Modbus byte order, drops its name
  when the form is below its maximum width. Its menu still names each option.
- A task channel's icon shows its state, in place of the flow arrow in a record table:
  the arrow for a channel that exists, a green `Icon.New` for one that deploying will
  create, and an amber `Icon.None` for one that is missing, with its name in gray. The
  two state circles look the same size: the tinted `Icon.New` disc is drawn a little
  wider than the dashed ring, because a faint fill looks smaller. Each state but the
  first has a tooltip that says what deploying does, below the name at its left edge.
  The name never takes a state color.
- A channel that deploying will create shows the name deploying gives it, from the
  function that deploy uses. "No channel" in gray is only for a channel whose name
  cannot be known yet.
- A channel that deploying skips, such as a disabled one, shows no `Icon.New`. In a
  list, its tooltip says that deploying skips it.
- A missing channel shows the name deploying will use, and a rename edits that name. Its
  data type unlocks too, because deploying creates a new channel.
- A channel that the form picks, as in Kafka write, is never new. It reads "No channel"
  before a pick and "Channel not found" after its channel is deleted.
- A channel name input shows the same state icons at its start. An item in a group tree
  shows them at its end, before its switch, so the pointers keep one left edge. Neither
  shows an icon or a tooltip for a skipped channel: the tree strikes through the item or
  its group.

### Port maps

- A task whose channels sit on ports shows its port map above its record table, in the
  `map` slot of `Task.Views.List`, with no switch to a Ports view: LabJack write, NI
  digital read and write, and the Modbus register map. Other port tasks show the map in
  the Ports view of the narrow list pane.
- Above the list, a section's title and its count share one line, with a thin line
  between sections. Each bank or port is one row, with its label in a 6.5 rem gutter. A
  port of more than eight lines wraps every eight. The rows fill columns read down, each
  as wide as a row of eight (`Task.Port.Rows`). Titles beside the rows, rows read
  across, and a matrix with no cell numbers were rejected.
- The Ports view keeps each title over its grid, with a thin line between sections or
  devices and 2.5 rem above and below it. Labels beside or above each bank were
  rejected.
- An NI device's location ("Dev3") is a chip at the right end of its title line:
  monospace, small, on gray step 3. The usage text does not repeat it. Monospace text
  alone was rejected, because it stands out too little.
- In a map, a click on a used port selects its channel. A click on a free port adds a
  channel there: the port shows a + under the pointer, and its tooltip reads "Add a
  channel on FIO3". The pointer stays an arrow. A used port keeps its tint, and its
  tooltip names its channels.
- A port field or picker moves its channel. A click on any port moves the channel there,
  and a free port's tooltip reads "Move to FIO3". It shows no +.
- A click on a Modbus register selects its channel, and an add selects the new channel.
- The pointer on an item marks its port in the map, and the pointer on a port marks its
  item (`Task.Port.HoverProvider`). A port in conflict darkens on hover.

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

### Logs

- A log follows new lines only while its view is at the end. A scroll up or a text
  selection holds the lines, and "Jump to latest" goes back to the end.
- An error or a warning tints its line and colors its level word with
  `--pluto-error-text` or `--pluto-warning-text`. Times, logger names, and debug lines
  are gray l9.
- Long lines wrap with a hanging indent. A log never scrolls sideways.
- "Copy logs" copies a short header (product, version, state, time), then the lines as
  plain text with UTC ISO times.

### Plot ranges

- Ranges are bars in a strip above the plot. Overlapping ranges stack in lanes by start,
  longer first, so a parent sits above its children. Three lanes are 20 px and show the
  name. Up to four more are 4 px, with no name.
- A bar is its range color tinted over the background, with a 1 px border in the color.
  The name is in the range hue, centered on its cap height, and fades at its end. A side
  cut by the plot edge has square corners and fades out over 40 px, and the name stays
  at the edge. A fade means only "more of this range is outside the view".
- The time sits at the bar's end in the code font: `T+h:mm:ss` for a running range, and
  the short length (`10min`) for any other. The name gets the room first.
- A running bar ends at the newest sample of a live plot, or at now. Its end is a 2 px
  cap in the range color that bends around the corners, with a glow that pulses every
  1.6 s, drawn at 15 Hz. 30 Hz doubled the CPU cost for no visible gain. Rejected: a
  fade, a dot, stripes, or a hollow bar for "running", and elapsed time after the name.
- In the plot, each range draws its start and end lines. Only a range that holds no
  other range gets a fill, so fills do not stack. A range with no color is not drawn.
- Hover fills the range, dims the others, and shows its name, start, end, and duration.
  A running range shows its start, how long it has run, and its planned end if it has
  one. The tooltip wraps a name past 240 px.
- A click on a range zooms to it and pauses the plot. A second click on that range goes
  back to the view and pause state from before. If the view changes between the two
  clicks, the second click zooms in again. Clicks nest, one level per click.
- The menu of a range starts with "Zoom to range", which does what a click does. While
  the plot shows that zoom, the item reads "Zoom out". The tooltip hides while the menu
  is open.

### Plot range select

- The dialog opens above the field. The Live row holds one chip per preset and an Off
  chip, which shows while the plot has no live window.
- A duration typed into the search (`15s`, `2h`, `1:30`) lists a live option at the top
  of the list. Picking it replaces the live window. Enter sets it and closes the dialog.

### Plot axes

- The toolbar's axes sit in a `Properties.Rail` on the left, the same rail as the
  schematic element config. Each tab shows `LinePlot.axisIcon`: a faint plot frame with
  the axis side drawn solid. An inset band was rejected for the first style.

### Plot data tab

- The labels share one column as wide as the widest label, so every field starts at one
  edge. X1, the field people change least, sits at the end of the Ranges row, as both
  set the x axis. Below a toolbar width of 144rem it takes its own row at the bottom.
- The X1 dialog floats, so it can be wider than its trigger.

### Plot lines tab

- The aggregation and detail of a line are compact selects, not rows of buttons. The
  trigger shows an icon and a label that says what it sets ("Min/max", "Medium detail").
  Each option says what it does to the plot in a few words ("Keeps spikes visible"), not
  how it works, so the dropdown stays narrow.
- A waveform marks min/max, a sine wave marks average, and `Icon.Meter` marks the
  detail.
- Icon buttons alone were rejected: a user must hover each one to learn it.

### Plot legend

- A padded Lyra `Divider` separates the axis groups: the shared border, with 0.5rem
  above and below. A 1 px line with no space was too heavy.

## Interaction

- Respond to every input in less than 100 ms. Show a spinner only after 150 to 300 ms,
  and keep it for at least 300 ms. A plot draws its axes at once and streams data in.
- Every command that a key runs also has a button or a menu item.
- Cmd+C on selected tree items copies their names, one per line. The HTML form makes
  each name a link when a link opens its type, so a paste into chat or email opens the
  resource. The tree menu has the same Copy item. A text selection, such as a name in a
  rename, copies as text.
- Hover can show rare actions, never the current state. A control that shows on hover
  keeps its space, so the layout does not move.
- Focus never gets lost. After an add, focus goes to the new item. When a dialog closes,
  focus goes back to where it was.
- Dialogs, drawers, and dropdown lists set `overscroll-behavior: contain`.
- An empty state has one primary action, named as a verb and a noun ("Create channel").

### Text selection

- Selection is off by default. The Console's `body` turns it off. Selection inherits, so
  a component turns it back on with `-webkit-user-select: text` and `user-select: text`,
  never with `!important`. WebKit reads only the prefixed form.
- Text that people paste somewhere else is selectable: dialog messages, error text,
  read-only values (IDs, keys, addresses, thumbprints, versions), code, and logs. A
  toast is not, because its copy button copies the whole status.
- Controls are not selectable: buttons, tabs, menu items, tree and list items, field
  labels, and toolbar titles. A record list copies through its context menu.
- Plots, schematics, tables, and the log keep their own selection, and Cmd+C copies it.
- A copy button shows a check while it stays in view. A menu item that copies shows a
  toast, because the menu closes.

### Drag previews

A drag shows the smallest true form of the object, with the facts that matter. It is
never so small that it is hard to read.

- **Objects** use `Haul.Preview`: a tile, the name, then one line of facts. The frame
  stays the same, and each type fills the tile and the facts. A channel shows its data
  type and index. A range shows its color, start, and length, or "Planned" when it has
  no start. Other tree resources show their type name.
- **Colors** fill the tile: a swatch, a palette color, or a range with a color.
- **Many objects** show the grabbed object's preview with two plates behind it and a
  count.
- **Schematic symbols and Arc stages** drag as the drawing alone, at canvas size, with
  the top-left corner on the pointer, where the drop places the node.
- **Tabs and panel tabs** drag as `Haul.Copy` of the tab at rest: its icon stays, its
  close button goes, and the copy holds the grab point.
- **The source dims** to 40% opacity until the drag ends. A selection dims every
  selected item.

Each source passes its own preview to `startDrag`, which requires one. There is no
preview map. Each object type has one preview component in its own module, such as
`Channel.HaulPreview`. A generic part (a select trigger, the tree) gets its preview from
its domain wrapper, through `renderHaulPreview` or the tree item's `haulPreview`. A
preview renders above every provider, so it takes plain values, not hooks.

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
