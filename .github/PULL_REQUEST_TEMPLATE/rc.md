# Release Candidate Pull Request

## Key Information

- **Version Number**: <!-- MAJOR.MINOR -->

## QA Template

- [ ] I have verified that any changes to the `rc.md` template in this diff have been manually added to this pull request.

## Version Conflicts

I have verified that, when released, the following packages will not conflict with any previously released packages:

- [ ] [`alamos/py`](https://pypi.org/project/alamos/)
- [ ] [`alamos/ts`](https://www.npmjs.com/package/@synnaxlabs/alamos)
- [ ] [`client/py`](https://pypi.org/project/synnax/)
- [ ] [`client/ts`](https://www.npmjs.com/package/@synnaxlabs/client)
- [ ] [`console`](https://github.com/synnaxlabs/synnax/releases)
- [ ] [`drift`](https://www.npmjs.com/package/@synnaxlabs/drift)
- [ ] [`freighter/py`](https://pypi.org/project/synnax-freighter/)
- [ ] [`freighter/ts`](https://www.npmjs.com/package/@synnaxlabs/freighter)
- [ ] [`media`](https://npmjs.com/package/@synnaxlabs/media)
- [ ] [`pluto`](https://npmjs.com/package/@synnaxlabs/pluto)
- [ ] [`server`](https://github.com/synnaxlabs/synnax/releases)
- [ ] [`x/ts`](https://www.npmjs.com/package/@synnaxlabs/x)

## Documentation

### Content Changes

I have verified that user-facing documentation for each of the following services has been updated to match any changes in the release candidate:

- [ ] `guides/analyst`
- [ ] `guides/comparison`
- [ ] `guides/get-started`
- [ ] `guides/operations`
- [ ] `guides/sys-admin`
- [ ] `reference/core`
- [ ] `reference/concepts`
- [ ] `reference/console`
- [ ] `reference/control`
- [ ] `reference/control/python`
- [ ] `reference/control/embedded`
- [ ] `reference/driver`
- [ ] `reference/driver/labjack`
- [ ] `reference/driver/modbus`
- [ ] `reference/driver/ni`
- [ ] `reference/driver/opc-ua`
- [ ] `reference/python-client`
- [ ] `reference/typescript-client`

### Examples

I have verified that code examples for each of the following services run correctly:

- [ ] `client/py`
- [ ] `client/ts`
  - [ ] `@synnaxlabs/client` version in `examples/node` is up-to-date

### Broken Links

- [ ] I have used a broken link checker like [brokenlinkcheck.com](https://www.brokenlinkcheck.com/) or [Dr. LinkCheck](https://www.drlinkcheck.com/) to check that all links work on the live website.

### Release Notes

- [ ] I have verified that release notes exist for this release.

## Console

### Channels

I can successfully:

- **Create Channel Modal**
  - [x] Create a new channel from the command palette. (channel_operations.py)
  - [x] Create multiple channels with the "Create More" flag set to true. (channel_operations.py)
- **Resources Toolbar**
  - [x] Open a channel plot by double-clicking it. (channel_operations.py)
  - [x] Drag and drop a channel onto a line plot. (plot/line_plot.py)
  - [x] Drag and drop a channel onto the line plot toolbar. (plot/line_plot.py)
  - **Context Menu**
    - [x] Rename a channel. (channel_lifecycle.py)
    - [x] Group multiple channels. (channel_operations.py)
    - [ ] Edit the calculation of a calculated channel.
    - [ ] Set an alias for a channel under a range.
    - [ ] Clear an alias for a channel under a range.
    - [x] Delete a channel. (channel_lifecycle.py)
    - [x] Copy a link to a channel. (channel_operations.py)
    - [ ] Hard reload the console.
- **Search and Command Palette**
  - [ ] Open a channel plot by its name.
  - [ ] Open the "Create Channel" modal.
  - [ ] Open the "Create Calculated Channel" modal.
- **Calculated Channels**
  - [ ] Plot a basic calculated channel.
  - [ ] Plot a nested calculated channel.
  - [ ] Intentionally create a channel with an erroneous expression, plot it and make sure the Console and Core remain stable and the error is logged to the Core and the Console.
  - [ ] Run and plot channels from python calc_channel_stress.py setting `--rate` with
    - [ ] 10 Hz
    - [ ] 100 Hz
    - [ ] 1,000 Hz
- [ ] Open a channel plot from a link.
- [ ] Rename a channel and ensure the change synchronizes properly across:
  - Resources Toolbar
  - Line Plot Visualization Toolbar
  - Log Visualization Toolbar
  - Schematic Visualization Toolbar
  - Table Visualization Toolbar
  - Task Configuration Dialog
- [ ] Set an alias for a channel and ensure the change synchronizes properly across:
  - Resources Toolbar
  - Line Plot Visualization Toolbar
  - Log Visualization Toolbar
  - Schematic Visualization Toolbar
  - Table Visualization Toolbar
  - Task Configuration Dialog
- [ ] Remove an alias for a channel and ensure the change synchronizes properly across:
  - Resources Toolbar
  - Line Plot Visualization Toolbar
  - Log Visualization Toolbar
  - Schematic Visualization Toolbar
  - Table Visualization Toolbar
  - Task Configuration Dialog

### Core

I can successfully:

- **Connect core Modal**
  - [ ] Connect to a core.
- **Dropdown**
  - [ ] Add a new core.
  - [ ] Connect to a core by selecting it.
  - [ ] Disconnect from a core by selecting it.
  - [ ] View the default local core in the selector.
  - [ ] View the default demo core in the selector.
  - **Context Menu**
    - [ ] Connect to a core.
    - [ ] Disconnect the active core.
    - [ ] Rename a core.
    - [ ] Remove a core.
    - [ ] Copy a link to a core.
    - [ ] Hard reload the console.
- **Search and Command Palette**
  - [ ] Open the "Connect Core" modal.
  - [ ] Open the "Add a Core" command (replaces "Connect a Core").
  - [ ] Use the "Log Out" command to log out of the active core.
- [ ] Open a core from a link.
- [ ] Receive meaningful feedback when a core connection fails.

### Login Page

I can successfully:

- **Login Screen**
  - [ ] See the core list on the left when multiple cores are configured.
  - [ ] Select a core from the list and see it highlighted.
  - [ ] Switch between cores and see the login form reset (username/password cleared).
  - [x] Log in with valid credentials (username: synnax, password: seldon). (user_login_logout.py)
  - [ ] Receive meaningful error feedback when logging in with invalid credentials.
  - [ ] Add a new core using the "+" button in the core list header.
  - [ ] See connection status indicators for each core in the list.
- **User Badge**
  - [ ] See the user avatar and username in the top-right corner after logging in.
  - [ ] Click the user badge to open the logout menu.
  - [x] Log out using the logout button in the user badge dropdown. (user_logout_badge.py)
  - [x] See the login screen again after logging out. (user_logout_badge.py)

### Devices

I can successfully:

- **Resources Toolbar**
  - [x] See a device appear in the toolbar. (device/lifecycle.py)
  - [x] See a device's status indicator. (device/lifecycle.py)
  - **Context Menu**
    - [ ] Group devices.
    - [ ] Configure an unconfigured device.
    - [ ] Change the identifier on a configured device.
    - [x] Rename a device. (device/lifecycle.py)
    - [x] Delete a device. (device/lifecycle.py)

### Documentation

I can successfully:

- [ ] Open documentation from the command palette.
- [ ] Open documentation from the question mark icon.
- [ ] Close and reopen documentation in the same place as left off.

### Labels

I can successfully:

- **Search and Command Palette**
  - [x] Open the "Edit Label" modal. (label_lifecycle.py)
- **Edit Label Modal**
  - [x] Add a new label. (label_lifecycle.py)
  - [x] Edit an existing label's name. (label_lifecycle.py)
  - [x] Change the color of an existing label. (label_lifecycle.py)
  - [ ] Rename a label and ensure the change synchronizes with the range toolbar.
  - [ ] Change a label's color and ensure the change synchronizes with the range toolbar.

### Layout

I can successfully:

- [ ] Drag and drop a mosaic leaf into a new window.
- [x] Rename a tab by double-clicking its name. (mosaic_operations.py)
- [x] Close layout tabs by clicking the close icon. (pages/open_close.py)
- **Context Menu**
  - [x] Split a mosaic leaf horizontally. (mosaic_operations.py)
  - [x] Split a mosaic leaf vertically. (mosaic_operations.py)
  - [ ] Focus on a leaf.
  - [ ] Rename a mosaic leaf.
  - [ ] Open a leaf in a new window.
  - [ ] Move a mosaic leaf to the main window from a secondary window.
- [x] Rename a tab with `Cmd + E`. (keyboard_shortcuts.py)
- [x] Close layout tabs with `Cmd + W`. (keyboard_shortcuts.py)
- [ ] Focus using `Cmd + L`.
- [ ] Open in a new window with `Cmd + O`.
- [x] Create a new mosaic leaf with `Cmd + T`. (keyboard_shortcuts.py)
- **Search and Command Palette**
  - [ ] Toggle the color theme.

### Line Plots

I can successfully:

- [x] Create a new line plot from the mosaic. (pages/open_close.py)
- **Visualization**
  - [x] Plot a historical range of data. (plot/line_plot.py)
  - [x] Plot a live range of data. (plot/line_plot.py)
  - [x] Move channels between axes. (plot/line_plot.py)
  - [x] Adjust the line thickness. (plot/line_plot.py)
  - [x] Relabel a line. (plot/line_plot.py)
  - [x] Set the plot title. (plot/line_plot.py)
  - [x] Download a range as a CSV. (plot/line_plot.py)
  - [x] Create a range from line plot selection. (plot/line_plot.py)
  - [ ] Use the measuring tool on the line plot.
  - [x] Rename a line plot from its tab title. (plot/line_plot.py)
  - [ ] Export a line plot.
  - [ ] Copy a link to a line plot.
  - [ ] Download a line plot as a CSV from the toolbar.
- **Resources Toolbar**
  - [ ] Open a plot by selecting it.
  - [ ] Drag a plot onto the mosaic.
  - **Context Menu**
    - [ ] Rename a plot.
    - [ ] Delete a plot.
    - [ ] Delete multiple plots.
    - [ ] Export a plot.
    - [ ] Copy a link to a plot.
- **Search and Command Palette**
  - [ ] Open an existing line plot.
  - [x] Create a new line plot. (pages/open_close.py)
  - [ ] Open the "Import Line Plot" dialog.
- [ ] Open a line plot from its link.
- [ ] Import a line plot.
- [x] Rename a line plot and ensure synchronization across: (pages/rename_synchronization.py)
  - [x] Resources Toolbar
  - [x] Mosaic Tab
  - [x] Visualization Toolbar

### Logs

I can successfully:

- [x] Create a new log from the visualization selector. (pages/open_close.py)
- **Visualization**
  - [ ] Rename a log from its mosaic tab title.
  - [ ] Scroll to view historical data.
  - [ ] Stream data from a virtual channel.
  - [ ] Stream data from a persisted channel.
  - [ ] Pause and resume scrolling using the streaming icon in the top right.
  - [ ] Switch the logging channel and observe data switching.
  - [ ] Preserve log data from a virtual channel in the buffer.
  - [ ] Copy a link to a log.
- **Resources Toolbar**
  - [ ] Open a log by selecting it.
  - [ ] Drag a log onto the mosaic.
  - **Context Menu**
    - [ ] Rename a log.
    - [ ] Delete a log.
    - [ ] Delete multiple logs.
    - [ ] Group logs.
    - [ ] Copy a link to a log.
- **Search and Command Palette**
  - [x] Create a new log. (pages/open_close.py)
  - [ ] Open an existing log.
- [ ] Open a log from its link.
- [x] Rename a log and ensure synchronization across: (pages/rename_synchronization.py)
  - [x] Resources Toolbar
  - [x] Mosaic Tab
  - [x] Visualization Toolbar

### Ontology

I can successfully:

- [ ] Create a new group and move resources to it.
- [ ] Move resources between groups.
- [ ] Rename a group.
- [ ] Delete a group.

### Permissions & Roles

I can successfully:

- **Role Management**
  - [ ] View all available roles in the Resources Toolbar.
  - [x] Assign a role to a user. (user_assign_role.py)
  - [ ] Unassign a role from a user.
  - [ ] Cannot delete built-in roles (Owner, Engineer, Operator, Viewer).

- **As an Owner:**
  - [ ] Can register new users and assign roles.
  - [ ] Can create, edit, and delete all resource types.

- **As an Engineer:**
  - [x] Can create and edit schematics, line plots, tables, logs, and workspaces. (role_engineer_permissions.py)
  - [x] Cannot register new users or assign roles. (role_engineer_permissions.py)

- **As an Operator:**
  - [x] Can actuate valves on a schematic. (simple_press_valves.py)
  - [x] Cannot create or edit schematics. (role_operator_permissions.py)

- **As a Viewer:**
  - [ ] Can view schematics, line plots, tables, logs.
  - [ ] Cannot actuate valves on a schematic.
  - [x] Cannot create or edit any resources. (role_viewer_permissions.py)

### Ranges

I can successfully:

- **Create Range Modal**
  - [x] Create a new local range. (range/range_lifecycle.py)
  - [x] Create a new persisted range. (range/range_lifecycle.py)
  - [x] Create a range with a parent range. (range/range_lifecycle.py)
  - [x] Add labels while creating a range. (range/range_lifecycle.py)
  - [x] Update the start and end times through changing the stage of a range (range/range_lifecycle.py)
- **Range Details**
  - [x] Rename a range from the tab name. (range/range_lifecycle.py)
  - [x] Rename a range. (range/range_lifecycle.py)
  - [x] Navigate to a parent range from a range (range/range_lifecycle.py)
  - [x] Copy Python code (range/range_lifecycle.py)
  - [x] Copy TypeScript code (range/range_lifecycle.py)
  - [x] Copy a link to the range (range/range_lifecycle.py)
  - [x] Open and successfully download data as a CSV (range/range_lifecycle.py)
  - [x] Favorite the range (range/range_lifecycle.py)
  - [x] Unfavorite the range (range/range_lifecycle.py)
  - [x] Change start and end times. (range/range_lifecycle.py)
  - [x] Change start and end times via the stage (range/range_lifecycle.py)
  - [x] Add labels. (range/range_lifecycle.py)
  - [x] Remove labels. (range/range_lifecycle.py)
  - **Child Ranges**
    - [x] Click and navigate to a child range (range/range_lifecycle.py)
    - [x] Create child ranges (range/range_lifecycle.py)
    - [x] Change the stage of a child range (range/range_lifecycle.py)
    - [x] Favorite a child range (range/range_lifecycle.py)
    - [x] Unfavorite a child range (range/range_lifecycle.py)
    - **Context Menu**
      - [ ] Rename a child range
      - [ ] Create a child range
      - [ ] Favorite a child range
      - [ ] Favorite multiple child ranges
      - [ ] Unfavorite a child range
      - [ ] Unfavorite multiple child ranges
      - [ ] Copy the link to a range
      - [ ] Delete a child range
      - [ ] Delete multiple child ranges
  - **Metadata**
    - [ ] Set metadata.
    - [ ] Update the value of metadata.
    - [ ] Copy the value of metadata to your clipboard
    - [ ] Open a link from metadata.
    - [ ] Delete metadata.
  - **Snapshots**
    - [ ] Navigate to a snapshot by clicking on it
    - [ ] Remove a snapshot
- **Search and Command Palette**
  - [ ] Open an existing range layout window.
  - [ ] Open the "Create Range" dialog.
  - [x] Open the Range Explorer (range/range_lifecycle.py)
- **Range Toolbar**
  - [ ] Open the "Create Range" modal from the toolbar.
  - [ ] Open the Range Explorer from the toolbar
  - [x] Switch the active range by clicking it. (range/range_lifecycle.py)
  - **Context Menu**
    - [ ] Open the "Create Range" modal.
    - [ ] Open the range layout tab.
    - [ ] Set an active range.
    - [ ] Remove an active range.
    - [ ] Rename a range.
    - [ ] Open the "Create Range" modal with a child range.
    - [ ] Add to the active line plot.
    - [ ] Add to a new line plot.
    - [ ] Remove from favorites.
    - [ ] Delete a persisted range.
    - [ ] Copy a link to a persisted range.
    - [ ] Save a local range to Synnax.
- **Range Explorer**
  - [ ] Open the range overview dialog by clicking on a range.
  - [ ] Search ranges
  - [ ] Filter ranges by labels
  - [x] Favorite and unfavorite ranges and see them added to the range toolbar (range/range_lifecycle.py)
  - [ ] Change the stage of a range
  - **Context Menu**
    - [x] Rename a range (range/range_lifecycle.py)
    - [ ] Create a child range
    - [x] Favorite a range (range/range_lifecycle.py)
    - [ ] Favorite multiple ranges
    - [ ] Unfavorite a range
    - [ ] Unfavorite multiple ranges
    - [ ] Copy the link to a range
    - [x] Delete a range (range/range_lifecycle.py)
    - [ ] Delete multiple ranges
- [ ] Open a range from its link.
- [ ] Rename a range and ensure synchronization across:
  - Range Explorer
  - Range Overview
  - Range Overview Tab Name
  - Ranges Toolbar
- [ ] Change the time of a range and ensure synchronization across:
  - Range Details
  - Ranges Toolbar
  - Range explorer
- [ ] Add or remove child ranges and ensure synchronization across:
  - Range Details (both parent and child view)
- [ ] Snapshot a schematic or task and ensure synchronization across:
  - Range Details
- [ ] Ensure channel aliases synchronize correctly across:
  - Line Plot
  - Line Plot Toolbar
  - Task Configuration Dialog
  - Schematic Value Tooltip

### Schematics

I can successfully:

- [x] Create a new schematic from the mosaic. (pages/open_close.py)
- **Visualization**
  - [ ] Display live data on a value.
  - [x] Actuate a valve. (set_output.py, simple_press_valves.py, setpoint_press_user.py)
  - [x] Select and change the color of multiple elements. (edit_props.py)
  - [ ] View the list of writers in control on the schematic.
  - [x] Acquire absolute control over a control sequence. (setpoint_press_user.py)
  - [ ] Copy a link.
  - [ ] Export a schematic.
- **Resources Toolbar**
  - [ ] Double-click a schematic to load it.
  - [ ] Drag a schematic onto the mosaic to load it.
  - **Context Menu**
    - [ ] Export a schematic.
    - [ ] Rename a schematic.
    - [ ] Delete a schematic.
    - [ ] Delete multiple schematics.
    - [ ] Snapshot a schematic to the active range.
    - [ ] Snapshot multiple schematics to the active range.
    - [ ] Make a copy of a schematic.
    - [ ] Copy multiple schematics.
    - [ ] Copy a link to a schematic.
- **Search and Command Palette**
  - [x] Open an existing schematic. (pages/open_close.py)
  - [x] Create a new schematic. (pages/open_close.py)
  - [ ] Import a schematic from a file.
- [x] Rename a schematic and ensure synchronization across: (pages/rename_synchronization.py)
  - [x] Mosaic Tab
  - [x] Resources Toolbar
  - [x] Visualization Toolbar
- [ ] Rename a schematic snapshot and ensure synchronization across:
  - Mosaic Tab
  - Resources Toolbar
  - Visualization Toolbar
  - Range Details Overview

### Schematic Symbols

I can successfully:

- [ ] Use custom symbols in a schematic.
- [ ] Use custom symbols as actuators in a schematic.
- **Symbol Editor**
  - [ ] Create a new symbol.
  - [ ] Rename a symbol.
  - [ ] Add handles to a symbol.
  - [ ] Add default scaling to a symbol.
  - [ ] Select color and color regions for the symbol.
  - [ ] Have multiple color regions for an actuator.
- **Schematic Symbols Toolbar**
  - [ ] Add a symbol to a symbol group.
  - [ ] Import a symbol group.
  - [ ] Import a symbol to a symbol group.
  - [ ] Create a new symbol group.
    - **Context Menu**
      - [ ] Rename a symbol group.
      - [ ] Delete a symbol group.
      - [ ] Export a symbol group.
      - [ ] Delete a symbol.
      - [ ] Export a symbol.
      - [ ] Edit a symbol.
      - [ ] Rename a symbol.

### Tables

I can successfully:

- [x] Create a new table from the mosaic. (pages/open_close.py)
- [ ] Open a table from a link.
- **Visualization**
  - [ ] Add rows and columns to a table.
  - [ ] Delete rows and columns from a table.
  - [ ] See live data in a table.
  - [ ] Add redlines to a live value cell in a table.
- **Resources Toolbar**
  - [ ] Double-click a table to load it.
  - [ ] Drag a table onto the mosaic to load it.
  - **Context Menu**
    - [ ] Rename a table.
    - [ ] Delete a table.
    - [ ] Delete multiple tables.
    - [ ] Export a table.
    - [ ] Copy a link to a table.
- **Search and Command Palette**
  - [ ] Open an existing table.
  - [x] Create a new table. (pages/open_close.py)
  - [ ] Import a table from a file.
- [x] Rename a table and ensure synchronization across: (pages/rename_synchronization.py)
  - [x] Resources Toolbar
  - [x] Mosaic Tab
  - [x] Visualization Toolbar

### Tasks

I can successfully:

- **Resources Toolbar**
  - [ ] Open task configuration by double-clicking.
  - **Context Menu**
    - [ ] Snapshot a task to the active range.
    - [ ] Snapshot multiple tasks to the active range.
    - [ ] Rename a task.
    - [ ] Group tasks.
    - [ ] Export a task.
    - [ ] Delete a task.
    - [ ] Delete multiple tasks.
    - [ ] Copy a link to a task.
- **Tasks Toolbar**
  - [ ] Pause and play a task.
  - [ ] Open task configuration by double-clicking.
  - **Context Menu**
    - [ ] Pause a task.
    - [ ] Start a task.
    - [ ] Open task configuration.
    - [ ] Copy a link to a task.
- **Search and Command Palette**
  - [ ] Open an existing task configuration.
- [ ] Open a task configuration from a link.
- [ ] Start a task on server boot up when the "Auto start" option is enabled.
- [ ] Import a task from a file via the import task commands
- [ ] Import a task from a file via drag-and-drop
- [ ] Rename a task and ensure synchronization across:
  - Resources Toolbar
  - Task Toolbar
  - Task Configuration Tab Name
  - Task Configuration Title
- [ ] Rename a task snapshot and ensure synchronization across:
  - Resources Toolbar
  - Task Toolbar
  - Task Configuration Tab Name
  - Task Configuration Title

### Users

I can successfully:

- **Resources Toolbar**
  - [ ] Open the "Assign Role" dialog for a user.
  - [ ] Assign a role to a user.
  - [ ] Unassign a role from a user.
  - [ ] View the roles assigned to a user.
  - [ ] Open the "Permissions" dialog.
  - [ ] Rename a user.
  - [ ] Delete a user.
  - [ ] Delete multiple users.
- **Search and Command Palette**
  - [x] Register a new user. (user_register.py)
  - [x] Register a new user with a specific role assigned. (user_register.py)
- [ ] Change a user's username and log in with the new username.
- [ ] Change a user's role and verify their permissions change accordingly.
- [ ] Log in as a user with a specific role and verify permission enforcement.

### Arc

I can successfully:

- **Search and Command Palette**
  - [ ] Created a named arc automation.
  - [ ] Open an existing arc automation.

- **Arc Toolbar**
  - [ ] Toggle Arc toolbar visibility with "A" keyboard shortcut
  - [ ] Create a new Arc automation with the "+" button
  - [ ] Start/stop an Arc automation with the play/pause button
  - [ ] Double-click an Arc to open the editor
  - [ ] Rename the arc automation via the context menu, ensuring that a warning that arcs will get redeployed on rename.
  - [ ] Delete the arc automation via the context menu, ensuring that any arc layouts get removes from the console mosaic.
  - [ ] View Arc status indicators (deployed/not deployed, running/stopped)

- **Arc Editor**
  - [ ] Create an alarm automation that changes statuses and includes the following blocks: channel source, constant, comparison, stable for, select, and status change.
  - [ ] Deploy the arc automation using `press_simulated_daq` and see statuses change based on the constant condition.
  - [ ] Stop the arc deployment.
  - [ ] Rename an arc, re-deploy it, and ensure that the new name is displayed.

### Statuses

I can successfully:

- **Search and Command Palette**
  - [ ] Open the status explorer.
  - [ ] Open the status create modal.

- **Status Create Modal**
  - [ ] Create a new status.
  - [ ] Create a new status with labels.

- **Status Explorer**
  - [ ] Filter statuses by labels.
  - [ ] Delete a single status.
  - [ ] Delete multiple statuses.
  - [ ] Favorite a status.
  - [ ] Unfavorite a status.

- **Status Notifications**
  - [ ] See status notifications in the bottom right corner when creating a new status.

- **Status Toolbar**
  - [ ] Unfavorite a status
  - [ ] Delete a status
  - [ ] Rename a status

### Version

I can successfully:

- [ ] View the correct version in the bottom navbar.
- [ ] Verify that the auto-update functionality works correctly.

### Workspaces

I can successfully:

- [x] Create a new workspace. (workspace.py)
- [ ] Import a workspace by drag and dropping from a directory.
- **Workspace Selector**
  - [x] Create a new workspace. (workspace.py)
  - [x] Switch workspaces in the selector. (workspace.py)
  - [x] Clear workspaces from the selector. (workspace.py)
- **Resources Toolbar**
  - [x] Switch workspaces in the resources view. (workspace.py)
  - **Context Menu**
    - [x] Rename a workspace. (workspace.py)
    - [x] Delete a workspace. (workspace.py)
    - [ ] Export a workspace.
    - [x] Create a new line plot in a workspace. (pages/open_close.py, plot/line_plot.py, layout/mosaic_operations.py, layout/keyboard_shortcuts.py)
    - [x] Create a new log in a workspace. (pages/open_close.py, pages/snap_to_grid.py)
    - [x] Create a new schematic in a workspace. (pages/open_close.py, schematic/alignment.py, schematic/edit_props.py)
    - [x] Create a new table in a workspace. (pages/open_close.py)
    - [ ] Import a line plot.
    - [ ] Import a schematic.
    - [ ] Import a log.
    - [ ] Import a table.
- [ ] Open a workspace from a link.
- **Search and Command Palette**
  - [x] Open the "Create Workspace" dialog. (workspace.py)
  - [ ] Import a workspace.
  - [x] Open an existing workspace. (workspace.py)
- [x] Rename a workspace and ensure synchronization across: (workspace.py)
  - Resources Toolbar
  - Workspace Selector
- [ ] Create a workspace in a previous version of Synnax, add visualizations, and open it in the release candidate.

## Driver

### General

I can successfully:

- [ ] Run the Driver for long periods with minimal memory leakage.
- **Handle invalid device configurations and receive meaningful feedback:**
  - [ ] Invalid ports.
  - [ ] Incorrect task type for devices (e.g., analog read on an analog output device).
  - [ ] Out-of-range values.
  - [ ] Multiple tasks using the same channel.
  - [ ] Device disconnection during a running task.
- [ ] Shut down the Core (`Ctrl + C`) without errors from the Driver routine.

### Control Sequences

I can successfully:

- [ ] Create a new control sequence from the mosaic.
- **Search and Command Palette**
  - [ ] Create a new control sequence.
  - [ ] Open an existing control sequence.
- **Context Menu**
  - [ ] Rename a control sequence.
  - [ ] Delete a control sequence.
- **Sequence Editing**
  - [ ] Edit a control sequence and see auto-complete suggestions for channels.
  - [ ] Edit a control sequence and see auto-complete suggestions for the following built-in functions:
    - [ ] `elapsed_time_within`
    - [ ] `elapsed_time`
    - [ ] `iteration`
    - [ ] `set`
    - [ ] `set_authority`
  - [ ] Accept channel auto-complete suggestions and see the correct channel populated in the `read_from` or `write_to` fields.
  - [ ] Manually configure the `read_from` and `write_to` fields.
  - [ ] Set the sequence control rate.
  - [ ] Configure, start, and correctly operate a minimal bang bang control sequence.

### LabJack

I can successfully:

- [ ] Enable and disable LabJack integration when starting the server.
- [ ] Recognize and connect to a LabJack device available locally.
- [ ] Run the driver on a machine without the LabJack library installed.
- **Read Task**
  - [ ] Plot live analog data.
  - [ ] Plot live digital data.
  - [ ] Tare data from multiple channels on a device.
  - [ ] Plot data with a linear scale applied.
  - [ ] Stop, start, and reconfigure tasks multiple times.
  - [ ] Enable and disable data saving.
  - [ ] Ensure no lag between sensor input and data written to the server.
  - [ ] Configure and run a read task for a thermocouple.
  - [ ] Run a read task with thermocouples, digital, and analog channels.
  - [ ] Disconnect a device while reading, reconnect it, and read data after reconfiguration.
  - **Reliable data plotting at the following sample rates:**
    - [ ] 1 Hz
    - [ ] 10 Hz
    - [ ] 100 Hz
    - [ ] 1 kHz
    - [ ] 10 kHz
    - [ ] 50 kHz
- **Write Task**
  - **Begin a write task and perform control actions with a schematic:**
    - [ ] Actuate a valve via a digital input.
    - [ ] Set an analog output to a specific voltage via a setpoint.
  - [ ] Stop, start, and reconfigure the task.
  - [ ] Disconnect a device while writing, reconnect it, and read data after reconfiguration.
  - **Configure response time based on the specified state rate:**
    - [ ] 1 Hz (should have a visible delay)
    - [ ] 20 Hz (should be nearly immediate)
- [ ] Configure simultaneous write and read tasks and stop or delete either without affecting the other.

### NI

I can successfully:

- [ ] Enable and disable NI integration when starting the server.
- [x] Recognize and connect to an NI device locally. (driver_ni_digital_write.py)
- [ ] Recognize and connect to NI devices over the network.
- [x] Recognize and connect to physical and simulated devices. (driver_ni_digital_write.py)
- [ ] Disconnect a physical device while a task is running without causing faults.
- [ ] Ignore chassis and view devices connected to it.
- [ ] Run the Driver without NI-DAQmx and System Configuration libraries installed.
- [ ] Receive feedback when trying to create an NI task on a machine lacking the necessary libraries.
- **Handle invalid device configurations and receive meaningful feedback:**
  - [ ] Invalid ports.
  - [ ] Incorrect task type for devices.
  - [ ] Out-of-range values.
  - [ ] Multiple tasks using the same channel.
- [ ] Shut down the driver without errors during embedded operation.
- [ ] Run various tasks on a single device.
- [ ] Run multiple tasks across multiple devices concurrently.
- **Reliable data streaming at the following sample rates:**
  - [ ] 1 Hz
  - [ ] 10 Hz
  - [ ] 100 Hz
  - [ ] 1 kHz
  - [ ] 5 kHz
- **Configure the following stream rates:**
  - [ ] 1 Hz
  - [ ] 10 Hz
  - [ ] 30 Hz
- **Analog Read Task**
  - [ ] Plot live data.
  - [ ] Tare data.
  - [ ] Handle device disconnection during active tasks with appropriate feedback.
  - [ ] Start multiple tasks at different times and view live data.
  - [ ] Enable and disable data saving.
  - [ ] Enabled auto-start, and ensure that the task automatically starts after configuration.
  - [ ] Ensure no lag between sensor input and Core data recording.
  - [ ] Configure and run an analog read task for the following channels:
    - [ ] Current (NI-9203)
    - [ ] Resistance (NI-9219)
    - [ ] RTD (NI-9217)
      - [ ] All RTD types and resistance configurations.
    - [ ] Built-in temperature sensor (USB-6289)
    - [ ] Thermocouple (NI-9211A)
      - [ ] All thermocouple types and CJC options.
    - [ ] Voltage (USB-6000)
      - **Terminal configurations:**
        - [ ] Default (USB-6000)
        - [ ] Reference Single-Ended (USB-6000)
        - [ ] Non-Referenced Single-Ended (NI-9206)
        - [ ] Differential (NI-9206)
        - [ ] Pseudo-Differential (NI-9234)
  - **Apply the following scales:**
    - [ ] Linear
    - [ ] Map
- **Digital Read Task**
  - [ ] Plot live data.
  - [ ] Stop, start, and reconfigure tasks.
  - [ ] Enable and disable data saving.
- **Digital Write Task**
  - [ ] Perform control actions using a schematic.
  - [ ] Stop, start, and reconfigure tasks.
  - [ ] Handle device disconnection during active tasks with appropriate feedback.
  - **Configure response time for specified state rates:**
    - [ ] 1 Hz (visible delay)
    - [ ] 20 Hz (near-instant response)

### OPC UA

I can successfully:

- [ ] Enable and disable OPC UA integration when starting the server.
- [x] Connect to an unencrypted OPC UA server. (opcua_read.py)
- [ ] Connect to an encrypted OPC UA server.
- [x] Create additional channels and move them to existing sampling groups. (opcua_read.py)
- [ ] Move and rename channels.
- **Read Task**
  - **Single Sampling**
    - [x] Read from multiple channels. (opcua_read.py)
    - [x] Autogenerate timestamps on the driver. (opcua_read.py)
    - [ ] Read timestamps from the OPC UA server.
  - **Array Sampling**
    - [x] Read from multiple channels. (opcua_read.py)
    - **Test the following array sizes:**
      - [ ] 1
      - [ ] 10
      - [ ] 100
    - [x] Autogenerate timestamps on the driver.
    - [ ] Read timestamps from the OPC UA server.
    - [x] Avoid driver crashes when improper array sizes are specified. (opcua/server.py injects improper arrays)
  - [ ] Obtain recommended Synnax channels based on the configured OPC UA node.
  - [ ] Connect to and read data from a physical device.
  - [ ] Maintain Driver operation during device disconnection or channel removal while a task is running.
  - [ ] Enable and disable data saving.
- **Write Task**
  - [ ] Perform control and verify changes on the connected OPC UA server.
  - [ ] Stop, start, and reconfigure tasks.
  - [ ] Enable and disable data saving.
  - [ ] Perform a write operation on an encrypted server.

### Modbus

I can successfully:

- [ ] Enable and disable Modbus integration when starting the server.
- [x] Connect to a Modbus TCP server. (modbus_read.py)
- [x] Configure connection parameters (IP address, port, unit ID). (modbus_task.py)
- **Read Task**
  - [x] Read holding registers from a Modbus server. (modbus_task.py)
  - [x] Read input registers from a Modbus server. (modbus_task.py)
  - [x] Read coils and discrete inputs from a Modbus server. (modbus_task.py)
  - [ ] Plot live data from Modbus registers.
  - [ ] Apply scaling to register values.
  - [ ] Enable and disable data saving.
  - [ ] Stop, start, and reconfigure read tasks.
  - [ ] Handle device disconnection gracefully.
  - **Reliable data reading at the following sample rates:**
    - [ ] 1 Hz
    - [ ] 10 Hz
    - [ ] 100 Hz
- **Write Task**
  - [ ] Write to holding registers on a Modbus server.
  - [ ] Write to coils on a Modbus server.
  - [ ] Perform control actions using a schematic.
  - [ ] Stop, start, and reconfigure write tasks.
  - **Configure response time for specified state rates:**
    - [ ] 1 Hz (visible delay)
    - [ ] 20 Hz (near-instant response)
- [ ] Run simultaneous read and write tasks on the same device.
- [ ] Run tasks across multiple Modbus servers concurrently.
