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

### Status colors

Success is primary blue, warning is yellow (amber in light), and error is red. Every
status color is a step of the palette ramps in `lyra/src/theme/theme.ts`, never a hex
from another system. Text on a 15% tint of its status (badges, notes) uses
`--pluto-primary-text`, `--pluto-warning-text`, or `--pluto-error-text`. Each is the
step nearest the status color that reads on the tint: a darker step in light mode, and a
lighter step or the base step in dark mode.

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
- A span of time (a timeout, a delay, a window) is a typed duration field
  (`Form.TimeSpanField`, or `Input.TimeSpan inline`), never a number with a unit suffix.
  The person types "500ms", "2.5s", or "1min 30s". A number alone takes the unit of the
  stored value. The readings and a hint show under the field while it has focus, also
  when it already has a value. A bound is a message under the field ("Timeout must be
  1min or less"), never a silent clamp.
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

### Menus and selects

- Menus use 13 px text in 28 px items. This includes context menus, the filter menu, and
  its values.
- The filter menu, its submenus, and the pickers of a filter chip use the context menu
  surface: l2, an l6 border, the large radius, 6 px padding, and the overlay shadow.
- A submenu opens 100 ms after the pointer stops on its item, and at once on a click or
  ArrowRight. ArrowLeft and Esc close one level. While the pointer moves toward an open
  submenu, the items it crosses do not light or open.
- A submenu opens with its first item level with its parent item, edge to edge with the
  menu. It fades in when it opens first, and it replaces another submenu with no
  animation.
- The item whose submenu is open stays lit (l4). Only one item is lit at a time. A
  highlight follows the pointer with no fade.
- The search in a submenu is a line of text over a hairline, with no box.
- Values have 4 px corners. A picked value tints, with no checkbox, and picked values
  next to each other join into one highlight.
- A divider separates groups of actions, never single items.
- In a group, all items have an icon or none do.
- "Copy" puts data on the clipboard. "Duplicate" makes a new resource. A menu never
  calls a duplicate "Copy".
- An open select marks the current value and opens scrolled to it.
- A field in a list item uses the `shadow` variant: no box at rest, a border on hover,
  and a box while open. Fields in forms and detail panes keep their box.
- An option with a description shows the icon beside the name and the description under
  the name, through `Select.ItemDescription`. The dropdown is wide enough for each
  description on one line. Only a narrow window wraps it, and it is never cut.
- A select with a few described options, such as the Arc performance level and the plot
  line sampling, has no search. A user reads the options instead.
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

### Explorer bars

- An explorer has two bars. The first holds the search, then the views. The second holds
  Filter and its chips on the left, and the sort, layouts, and Create on the right.
- The search is 270 px wide. A hairline that runs the full height of the bar separates
  it from the views. Its icon is above the Filter icon.
- Both bars sit in one island: an l1 card with an l5 border and the large radius, inset
  6 px from the top and 9 px from the sides of the panel. An l4 line separates the two
  bars. The second bar's controls are 28 px with 13 px text.
- Each bar is 42 px (7rem) tall. The filter bar grows when its chips wrap.
- The bars have no edit toggle. The search shows on every view. The filters show on a
  static view, and on a saved view when the user may update it. Create view shows when
  the user may create views.

### View tabs

- A view is a 28 px pill with the medium radius: an icon, then the name, with a 6 px
  gap. Names are l9. The open view takes the text color, a fill, and an l6 border.
- The add view button sits right after the last view, not at the end of the bar.
- A view has an icon and a color, picked from "Change icon" in its context menu with the
  same picker as panels. The fixed view shows the explorer's icon.
- Create view opens a modal like the project modal: an icon button, then the name. The
  name has no default value.

### Filter chips

- A chip is a row of parts: name, operator, value, and remove. It has an l2 fill, an l5
  border, the medium radius, and l6 lines between its parts.
- A chip is 26 px tall, and each part has 10.5 px of padding on each side. The text is
  13 px.
- The name icon and the operator are muted (l9). The remove part is 28 px wide. Its icon
  is 12 px and l8, and it takes the text color on hover.
- A part that opens something gets l3 on hover and l4 on press.
- The dots of a labels value overlap a little, as one group.

### Page sections

- A section of a record page, such as Child ranges or Metadata, is an outlined box like
  the channel box of a task form: an l6 outline, the large radius, an l1 body, and an l2
  header bar 37 px tall with a line under it.
- The header holds one text button with the section's icon, title, and caret. It
  collapses the box to its bar. An action in the header opens the box again.
- An empty section shows no box, only one muted "+ Noun" line that adds its first item.

### Record tables

- A task's channel list is a record table: one item per channel, built from
  `Task.Views.ListChannelItem`. The task supplies only its own fields.
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
  cut by the plot edge has square corners, and the name stays at the edge.
- In the plot, each range draws its start and end lines. Only a range that holds no
  other range gets a fill, so fills do not stack. A range with no color is not drawn.
- Hover fills the range, dims the others, and shows its name, start, end, and duration.
  The tooltip wraps a name past 240 px.
- A click on a range zooms to it and pauses the plot. A second click on that range goes
  back to the view and pause state from before. If the view changes between the two
  clicks, the second click zooms in again. Clicks nest, one level per click.

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
