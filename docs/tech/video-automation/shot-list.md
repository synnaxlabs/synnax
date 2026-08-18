# Docs Video Shot List

Complete inventory of every `<Video id="..." />` under `docs/site/src/pages`.

**Total: 84 unique ids** -> 81 under `reference/` (the docs-site tutorial set) plus 3
legacy clips under `releases/` (release-notes demos, listed in an addendum at the
bottom). Shot descriptions marked "(inferred)" come from thin prose; the flow was
reconstructed from the section heading and neighboring steps. Source paths are
relative to `docs/site/src/pages/`.

## Prerequisites legend

- **Live telemetry** (a channel actively streaming data, e.g. the demo core or a sim
  writer): `console/ui-overview/visualization-toolbar`, `console/ui-overview/multiple-tabs`,
  `console/ranges/plot-create`, all `console/line-plots/*`, `console/tables/create`,
  `console/logs/example`, `console/calculated-channels/create`,
  `console/calculated-channels/edit`, `console/schematics/value`,
  `console/schematics/valve`, `control/arc/get-started/deploy-automation`, all
  `client/resources/build-device-driver/*`, every `device-drivers/*/read-task/*` and
  `device-drivers/task/task-basic`.
- **Pre-seeded ranges**: `console/ui-overview/range-toolbar`,
  `console/ui-overview/palette-search`, `console/ranges/resources`,
  `console/ranges/palette`, `console/ranges/create-child`,
  `console/ranges/add-meta-data`, `console/ranges/add-label`, `console/channels/alias`,
  `console/schematics/snapshot`.
- **Pre-seeded users**: `console/users/modal-change-role`,
  `console/users/drag-change-role` (need at least one non-root registered user).
- **Pre-seeded channels** (index + data channels already created, with or without
  data): `console/channels/alias`, `console/calculated-channels/create`,
  `console/calculated-channels/edit`, `console/line-plots/data-tab`,
  `console/logs/example`, `console/tables/create`, `console/schematics/value`,
  `console/schematics/valve`, `control/arc/get-started/create-automation`.
- **Pre-seeded workspaces/visualizations**: `console/workspaces/load`,
  `console/schematics/download`, `console/schematics/snapshot`.
- **Drag-and-drop**: `console/ui-overview/close-toolbars` (drawer resize-to-edge),
  `console/ui-overview/multiple-tabs` (mosaic tab drag),
  `console/users/drag-change-role`, `console/schematics/upload` (file drop option),
  `console/schematics/connections`, `console/schematics/align-items`,
  `console/schematics/change-color` (drag select box), symbol placement in any
  schematic shot.
- **Context menus**: `console/channels/alias`, `console/calculated-channels/edit`,
  `console/users/modal-change-role`, `console/schematics/download`,
  `console/schematics/snapshot`, `device-drivers/task/toolbar`,
  `device-drivers/ni/configure-device/ni-configure-device`,
  `device-drivers/labjack/configure-device/lj-configure`,
  `device-drivers/ethercat/configure-device/ethercat-configure`,
  `control/arc/get-started/create-automation` (one of the entry points).
- **Hardware simulators / external services**: OPC UA server sim for
  `device-drivers/opc-ua/*`; Modbus TCP server sim for `device-drivers/modbus/*`; an
  HTTP JSON server for `device-drivers/http/*`; real or simulated NI hardware
  (NI-DAQmx + NI MAX) for `device-drivers/ni/*`; a LabJack device (Kipling) for
  `device-drivers/labjack/*`; an EtherCAT network for `device-drivers/ethercat/*`; a
  PagerDuty account + Events API key for `device-drivers/pagerduty/alert-task/create`;
  an Arduino on a serial port for `client/resources/build-device-driver/*`.
- **Running driver (rack connected to the core)**: every `device-drivers/*` id and
  `control/arc/get-started/deploy-automation`.

---

## console/ui-overview

- `console/ui-overview/range-toolbar` (reference/console/ui-overview.mdx) -> open the
  Ranges Toolbar from the left rail -> show the list of favorited ranges and click
  through one -> requires a couple of pre-seeded ranges. (inferred)
- `console/ui-overview/task-toolbar` (reference/console/ui-overview.mdx) -> open the
  Tasks Toolbar from the left rail -> show the list of tasks running on a driver and
  pause/resume one -> requires a running driver with at least one task. (inferred)
- `console/ui-overview/visualization-toolbar` (reference/console/ui-overview.mdx) ->
  with a line plot (or schematic) open in the mosaic, click the visualize button to
  open the bottom Visualization Toolbar -> edit a property and show the plot update ->
  requires an existing visualization with channels selected. (inferred)
- `console/ui-overview/cluster-toolbar` (reference/console/ui-overview.mdx) -> open
  the Core Selector -> show the list of cores, switch the active core, and show the
  connection badge update -> requires two saved core connections. (inferred)
- `console/ui-overview/open-toolbars` (reference/console/ui-overview.mdx) -> start
  from a clean mosaic -> click each toolbar icon on the side rails to open the drawers
  one after another.
- `console/ui-overview/close-toolbars` (reference/console/ui-overview.mdx) -> with
  drawers open, close one by clicking its icon and close another by dragging its edge
  until it collapses.
- `console/ui-overview/multiple-tabs` (reference/console/ui-overview.mdx) -> with one
  visualization open, create a second tab and drag it in the mosaic to split the view
  -> end with two visualizations displayed side by side -> requires channels/data for
  the visualizations to look real.
- `console/ui-overview/documentation` (reference/console/ui-overview.mdx) -> click the
  question-mark icon -> the Synnax documentation site opens inside the Console.
- `console/ui-overview/palette-search` (reference/console/ui-overview.mdx) -> open the
  palette with Cmd/Ctrl+P (or click the top search bar) -> type the name of an
  existing test range -> select the result and show it become the active range ->
  requires a pre-seeded range.
- `console/ui-overview/palette-command` (reference/console/ui-overview.mdx) -> open
  the palette in command mode (Cmd/Ctrl+Shift+P or typing `>`) -> run a command such
  as toggling the color theme or creating a visualization -> show the effect.

## console/clusters

- `console/clusters/connect` (reference/console/cores.mdx) -> click the **Connect**
  button in the Core Selector -> fill host/port/username/password (optionally Secure)
  in the connection dialog -> submit -> the new core appears in the list, becomes
  active, and the connection status indicator in the top-right turns connected ->
  requires a reachable core (localhost or demo.synnaxlabs.com).

## console/workspaces

- `console/workspaces/create` (reference/console/workspaces.mdx) -> open the Workspace
  Selector in the top-left corner -> create a new workspace via its create option and
  name it -> the workspace becomes active in the selector. (inferred)
- `console/workspaces/load` (reference/console/workspaces.mdx) -> locate an existing
  workspace either in the Workspaces Toolbar on the left or in the Workspace Selector
  -> click it -> the saved layout loads into the mosaic -> requires a pre-seeded
  workspace with a saved layout.

## console/users

- `console/users/register` (reference/console/users.mdx) -> open the command palette
  (Cmd/Ctrl+Shift+P) -> run "Register a User" -> fill username/password and pick a
  role in the dialog -> submit -> the new user appears in the Users Toolbar.
- `console/users/modal-change-role` (reference/console/users.mdx) -> open the Users
  Toolbar -> right-click an existing user -> select "Assign to role" -> pick a role
  from the dropdown in the dialog and click "Assign" -> the user moves under the new
  role -> requires a pre-seeded non-root user.
- `console/users/drag-change-role` (reference/console/users.mdx) -> in the Users
  Toolbar resource tree, drag a user node onto a different role node -> drop -> the
  user is reassigned to that role -> requires a pre-seeded non-root user.

## console/channels

- `console/channels/create` (reference/console/channels.mdx) -> open the command
  palette (type `>` or Cmd/Ctrl+Shift+P) -> run "Create Channel" -> fill name,
  virtual/index flags, data type, and index channel in the dialog -> submit -> the
  channel appears in the Channels Toolbar.
- `console/channels/alias` (reference/console/channels.mdx) -> with a range active,
  open the Channels Toolbar -> right-click a channel -> choose the alias option and
  type a range-specific name (e.g. rename `digital_input_1` to `pressure_01`) -> show
  the alias displayed while that range is active -> requires a pre-seeded channel and
  an active range.

## console/calculated-channels

- `console/calculated-channels/create` (reference/console/calculated-channels.mdx) ->
  open the command palette -> run "Create Calculated Channel" -> name the channel and
  write an Arc expression referencing existing channels (ending in `return`),
  optionally pick an operation/window -> submit -> requires pre-seeded source
  channels, ideally with live data so the calculation visibly produces values.
- `console/calculated-channels/edit` (reference/console/calculated-channels.mdx) ->
  open the Channels Toolbar -> right-click an existing calculated channel -> select
  "Edit Calculation" from the context menu -> modify the expression in the dialog and
  save -> requires a pre-seeded calculated channel.

## console/ranges

- `console/ranges/toolbar-create` (reference/console/ranges.mdx) -> open the Ranges
  Toolbar -> click its **create** button -> fill name and from/to timestamps in the
  dialog -> save -> the range appears in the toolbar.
- `console/ranges/palette-create` (reference/console/ranges.mdx) -> open the command
  palette (`>` or Cmd/Ctrl+Shift+P) -> run the create-range command -> fill and save
  the range dialog.
- `console/ranges/plot-create` (reference/console/ranges.mdx) -> on a line plot with
  data, drag-select a region -> use the selection to create a range -> the dialog
  opens with From/To pre-populated from the selected region -> save -> requires a plot
  with plotted (live or historical) data.
- `console/ranges/resources` (reference/console/ranges.mdx) -> open the Ranges Toolbar
  on the left -> browse the core's ranges -> click one to load it into the toolbar as
  active -> requires pre-seeded ranges.
- `console/ranges/palette` (reference/console/ranges.mdx) -> click the Search and
  Command Palette at the top -> type the name of a permanently saved range ->
  arrow-key/click to select it -> the range loads -> requires a pre-seeded saved
  range.
- `console/ranges/create-child` (reference/console/ranges.mdx) -> open a range's
  overview page -> click **Add Child Range** (or set the parent field in the creation
  dialog) -> fill and save the child range -> requires an existing parent range.
- `console/ranges/add-meta-data` (reference/console/ranges.mdx) -> on the range
  overview page, click an empty cell in the metadata section -> type a key and value
  (e.g. a link) -> the field is saved -> requires an existing range.
- `console/ranges/add-label` (reference/console/ranges.mdx) -> on the range overview
  page, click the **Add Label** button -> pick or create a label -> the label chip
  appears on the range -> requires an existing range.

## console/line-plots

All shots require channels with data (live streaming for the rolling-range ones).

- `console/line-plots/toolbar` (reference/console/line-plots.mdx) -> with a line plot
  tab open, click the visualize button in the bottom-left corner -> the Visualization
  Toolbar opens showing the plot's tabs.
- `console/line-plots/data-tab` (reference/console/line-plots.mdx) -> in the Data tab,
  select Y1 channels, pick a range (including a rolling live range), and optionally
  Y2/X1 -> show lines appearing on the plot as channels are selected.
- `console/line-plots/lines-tab` (reference/console/line-plots.mdx) -> in the Lines
  tab, change a line's label, width, downsampling, and color -> show the legend and
  line style updating live.
- `console/line-plots/axes-tab` (reference/console/line-plots.mdx) -> in the Axes tab,
  set lower/upper bounds, tick spacing, and an axis label/direction/size -> show the
  axes re-render.
- `console/line-plots/properties-tab` (reference/console/line-plots.mdx) -> in the
  Properties tab, change the plot title and toggle Show Title / Show Legend -> show
  the plot header and legend appear and disappear.
- `console/line-plots/zoom` (reference/console/line-plots.mdx) -> click the zoom
  button (or hold Alt/Option) -> drag a box on the plot -> the axes rescale to the
  boxed area.
- `console/line-plots/pan` (reference/console/line-plots.mdx) -> click the pan button
  (or hold Shift) -> drag on the plot -> the view translates along the data.
- `console/line-plots/select` (reference/console/line-plots.mdx) -> click the
  selection button -> drag a box over data -> right-click the selection and show the
  menu: copy the time range (Python / TypeScript / ISO), create a range from it, or
  download as CSV.
- `console/line-plots/slope` (reference/console/line-plots.mdx) -> click the slope
  button -> click a first point, then press `2` and click a second point -> the
  overlay shows slope and x/y deltas between the two points.

## console/tables

- `console/tables/create` (reference/console/tables.mdx) -> add a new tab in the
  mosaic and select the table visualization (or run "Create a Table" from the command
  palette) -> show an empty table appear and, ideally, a value cell hooked to a
  channel updating -> requires channels with live data for a convincing end state.

## console/logs

- `console/logs/example` (reference/console/logs.mdx) -> with a log visualization
  open, select multiple channels in the Visualization Toolbar and adjust formatting ->
  show log lines streaming in as telemetry arrives -> requires channels with live
  (low-rate) data.

## console/schematics

- `console/schematics/create` (reference/console/schematics.mdx) -> add a new tab and
  select the schematic visualization -> an empty schematic canvas opens in edit mode
  with the symbols library at the bottom.
- `console/schematics/download` (reference/console/schematics.mdx) -> find the
  schematic under its workspace in the Workspaces Toolbar -> right-click and choose
  "Download as JSON" -> the file saves -> requires an existing schematic in a
  workspace.
- `console/schematics/upload` (reference/console/schematics.mdx) -> import a schematic
  JSON either by dragging the file from the file system onto the Console or by
  right-clicking a workspace name in the Workspaces Toolbar and importing -> the
  schematic opens as a new tab -> requires a schematic JSON file on disk.
- `console/schematics/value` (reference/console/schematics.mdx) -> in edit mode,
  select a value symbol -> in the Telemetry tab of the Visualization Toolbar, pick an
  input channel and set precision/averaging -> the symbol starts displaying the
  channel's live value -> requires a channel with live data.
- `console/schematics/connections` (reference/console/schematics.mdx) -> in edit mode,
  hover a symbol so its attachment points appear -> click an attachment point and drag
  a connection line to another symbol -> release to connect -> requires at least two
  symbols on the canvas.
- `console/schematics/align-items` (reference/console/schematics.mdx) -> drag a
  selection box around several symbols -> click the vertical or horizontal alignment
  button -> the symbols snap into alignment.
- `console/schematics/change-color` (reference/console/schematics.mdx) -> drag a
  selection box around multiple symbols -> open the color picker and choose a new
  color -> all selected symbols recolor together.
- `console/schematics/valve` (reference/console/schematics.mdx) -> switch the
  schematic to control mode -> click a valve (or other actuator) to acquire control
  and toggle it -> show the control indicator circle (blue/green/red) and the colored
  control legend while the state channel reflects the actuation -> requires
  state/command channels, ideally with a sim echoing state.
- `console/schematics/snapshot` (reference/console/schematics.mdx) -> find the
  schematic in the Workspaces Toolbar -> open its context menu -> snapshot the
  schematic to a range -> show the snapshot appear under the range -> requires an
  existing schematic and an existing range.
- `console/schematics/symbol-create-group` (reference/console/schematics.mdx) -> in
  the schematic symbols browser at the bottom, create a new custom symbol group -> the
  empty group appears in the library.
- `console/schematics/symbol-import-svg` (reference/console/schematics.mdx) -> with a
  group created, open the create symbol dialog and import an SVG file -> the editor
  identifies color regions and the new symbol lands in the group -> requires an SVG
  file on disk.

## control/arc

- `control/arc/get-started/create-automation` (reference/control/arc/get-started.mdx)
  -> open the command palette and run "Create an Arc automation" (alternatives: "+"
  button in the toolbar, or right-click a driver in the resources panel) -> name it
  and select **Text** editor mode -> the Arc text editor opens.
- `control/arc/get-started/deploy-automation` (reference/control/arc/get-started.mdx)
  -> with the tutorial program in the editor (and `tank_pressure`/`pressure_scaled`
  virtual channels created so no red squiggles remain), select a driver from the
  editor-toolbar dropdown -> click **Configure** to upload -> click **Play** -> the
  status indicator shows running; optionally show the Pause button stopping it ->
  requires a running driver.

## client/resources (build-device-driver tutorial)

All three require the tutorial's Arduino connected over serial and the corresponding
TypeScript/Python bridge script running against a local core.

- `client/resources/build-device-driver/line-plot`
  (reference/client/advanced/build-device-driver.mdx) -> with the read-only driver
  script streaming `arduino_value`, create a line plot in the Console and select the
  `arduino_value` channel against a rolling range -> live Arduino sensor data plots in
  real time. (inferred)
- `client/resources/build-device-driver/schematic`
  (reference/client/advanced/build-device-driver.mdx) -> with the write-only driver
  script listening on `arduino_command`, create a schematic, add a switch symbol, and
  set its channel(s) to `arduino_command` -> enter control mode and click the switch
  -> the Arduino LED toggles. (inferred)
- `client/resources/build-device-driver/console-setup`
  (reference/client/advanced/build-device-driver.mdx) -> for the read-write driver,
  configure the schematic switch with Command = `arduino_command` and State =
  `arduino_state`, and add a line plot for `arduino_value` -> toggle the switch in
  control mode and show the state confirm plus the analog plot updating. (inferred)

## device-drivers

All shots in this area require a running driver (rack) connected to the core, plus the
integration-specific hardware or simulator noted per shot.

### task basics

- `device-drivers/task/toolbar` (reference/driver/task-basics.mdx) -> open the Devices
  Toolbar via the device icon -> right-click a configured device -> select a task type
  from the context menu -> the task configuration dialog opens -> requires a
  configured device.
- `device-drivers/task/command-palette` (reference/driver/task-basics.mdx) -> open the
  Search and Command Palette (Cmd/Ctrl+Shift+P) -> in command mode type "Create" ->
  pick a task-creation command from the results -> the task configuration dialog
  opens.
- `device-drivers/task/layout-selector` (reference/driver/task-basics.mdx) -> click
  the add icon in the top-right corner of the central mosaic -> select a task type
  from the layout menu -> the task configuration dialog opens as a tab.
- `device-drivers/task/task-basic` (reference/driver/task-basics.mdx) -> full task
  lifecycle in the Console: with a task form open, click **Configure**, then the play
  button to start it (show data/status running), then the pause button to stop ->
  requires a configured device producing data. (inferred)

### pagerduty

- `device-drivers/pagerduty/alert-task/create`
  (reference/driver/pagerduty/alert-task.mdx) -> create a PagerDuty alert task in the
  Console, fill in the trigger/severity configuration, click Configure and start it ->
  requires a PagerDuty device (Events API routing key) already connected and a running
  driver. (inferred)

### ni

All require NI hardware (or NI-DAQmx simulated devices) reserved via NI MAX on the
driver machine.

- `device-drivers/ni/configure-device/ni-configure-device`
  (reference/driver/ni/configure-device.mdx) -> a newly connected NI module triggers a
  Console notification -> find the device in the Devices Toolbar, right-click, select
  "Configure" -> step through the configuration dialog (name, identifier) and save.
- `device-drivers/ni/analog-read-task/ni-analog-read`
  (reference/driver/ni/analog-read-task.mdx) -> create an NI analog read task, add
  analog input channels (port, voltage range), set sample/stream rates -> Configure,
  then start the task and show live voltage data streaming. (inferred)
- `device-drivers/ni/analog-write-task/ni-analog-write`
  (reference/driver/ni/analog-write-task.mdx) -> create an NI analog write task, add
  analog output channels with command/state channel pairs, set the state rate ->
  Configure, start, and command an output value -> show the state channel feedback.
  (inferred)
- `device-drivers/ni/digital-read-task/ni-digital-read`
  (reference/driver/ni/digital-read-task.mdx) -> create an NI digital read task, add
  digital input lines, set sample/stream rates -> Configure and start -> show digital
  line states streaming. (inferred)
- `device-drivers/ni/digital-write-task/ni-digital-write`
  (reference/driver/ni/digital-write-task.mdx) -> create an NI digital write task, add
  digital output lines with command/state pairs -> Configure, start, and toggle an
  output -> show state feedback. (inferred)
- `device-drivers/ni/counter-read-task/ni-counter-read`
  (reference/driver/ni/counter-read-task.mdx) -> create an NI counter read task, add
  counter channels on unique ports, set rates -> Configure and start -> show counter
  values streaming. (inferred)

### ethercat

All require an EtherCAT network attached to the driver machine.

- `device-drivers/ethercat/configure-device/ethercat-configure`
  (reference/driver/ethercat/configure-device.mdx) -> EtherCAT devices are
  auto-discovered on the network -> open the Devices Toolbar via the device icon and
  show the discovered devices, then configure/enable one. (inferred)
- `device-drivers/ethercat/read-task/ethercat-read`
  (reference/driver/ethercat/read-task.mdx) -> create an EtherCAT read task, map TxPDO
  entries to channels, set the sample (cycle) and stream rates -> Configure and start
  -> show deterministic live data streaming. (inferred)
- `device-drivers/ethercat/write-task/ethercat-write`
  (reference/driver/ethercat/write-task.mdx) -> create an EtherCAT write task, map
  RxPDO outputs with command/state channel pairs, set state and execution rates ->
  Configure, start, and command an output -> show state feedback. (inferred)

### opc-ua

All require a reachable OPC UA server (simulator works).

- `device-drivers/opc-ua/connect-server/connect-server`
  (reference/driver/opc-ua/connect-server.mdx) -> open the connect dialog for an OPC
  UA server (via palette/toolbar), enter endpoint URL and security/auth settings ->
  test and save the connection -> the server appears as a connected device in the
  Devices Toolbar. (inferred)
- `device-drivers/opc-ua/read-task/read` (reference/driver/opc-ua/read-task.mdx) ->
  create an OPC UA read task, browse the server's node tree and add nodes as channels,
  set sample/stream rates -> Configure and start -> show live node values streaming.
  (inferred)
- `device-drivers/opc-ua/write-task/write` (reference/driver/opc-ua/write-task.mdx) ->
  create an OPC UA write task, map writable nodes to command channels -> Configure,
  start, and write a value -> show the node update on the server -> server must allow
  write access to the nodes. (inferred)

### http

All require an HTTP server returning JSON (a small local mock server works).

- `device-drivers/http/connect-server/connect-server`
  (reference/driver/http/connect-server.mdx) -> open the "Connect an HTTP server"
  dialog, enter base URL, authentication, and optionally a health-check (JSON pointer
  + expected value) -> save -> the server appears as a connected device. (inferred)
- `device-drivers/http/read-task/read` (reference/driver/http/read-task.mdx) -> create
  an HTTP read task, define the request and JSON-pointer field extraction into
  channels (with optional timestamp_format index) -> Configure and start -> show
  polled response values streaming into channels. (inferred)
- `device-drivers/http/write-task/write` (reference/driver/http/write-task.mdx) ->
  create an HTTP write task, map command channels into the outgoing request
  body/params -> Configure, start, and write a command -> show the request firing
  against the server. (inferred)

### labjack

All require a LabJack device connected to the driver machine (updated via Kipling).

- `device-drivers/labjack/configure-device/lj-configure`
  (reference/driver/labjack/configure-device.mdx) -> a newly connected LabJack
  triggers a Console notification -> find the device in the Devices Toolbar,
  right-click, select "Configure" -> complete the configuration dialog and save.
- `device-drivers/labjack/read-task/lj-read` (reference/driver/labjack/read-task.mdx)
  -> create a LabJack read task, add analog/digital (or thermocouple) input channels,
  set sample/stream rates -> Configure and start -> show live data streaming.
  (inferred)
- `device-drivers/labjack/write-task/lj-write`
  (reference/driver/labjack/write-task.mdx) -> create a LabJack write task, add output
  channels with command/state pairs, set the state rate -> Configure, start, and
  command an output -> show state feedback. (inferred)

### modbus

All require a reachable Modbus TCP server (simulator works).

- `device-drivers/modbus/connect-server/modbus-connect`
  (reference/driver/modbus/connect-server.mdx) -> open the connect dialog for a Modbus
  server, enter name, host, port (502), and byte/word swap settings -> test and save
  -> the server appears as a connected device. (inferred)
- `device-drivers/modbus/read-task/modbus-read`
  (reference/driver/modbus/read-task.mdx) -> create a Modbus read task, add channels
  for holding/input registers, coils, or discrete inputs with addresses and data
  types, set sample/stream rates -> Configure and start -> show live register values
  streaming. (inferred)
- `device-drivers/modbus/write-task/write` (reference/driver/modbus/write-task.mdx) ->
  create a Modbus write task, map command channels to writable coils/holding registers
  -> Configure, start, and write a value -> the register updates on the server (no
  state feedback channels; direct write). (inferred)

---

## Addendum: releases pages (legacy release-notes clips)

These sit outside the `reference/` tutorial set; listed for completeness.

- `releases/0-35-0/schematic-q-cmd` (releases/0-35-0.mdx) -> with a schematic symbol
  selected, press `Q` to open the quick layout menu -> rotate the symbol and drag its
  label/control chip to a new location via the gray boxes.
- `releases/0-36-0/schematic-undo` (releases/0-36-0.mdx) -> while editing a schematic,
  make a change (move/delete a symbol) -> press Ctrl+Z to undo it and Ctrl+Shift+Z to
  redo it.
- `releases/0-56-0/schematic-collaborative-editing` (releases/0-56-0.mdx) -> two
  Console sessions editing the same schematic simultaneously, with one user's edits
  appearing live in the other's view -> requires two clients connected to the same
  core. (inferred)
