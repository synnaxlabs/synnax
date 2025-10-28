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
  - [ ] Create a new channel from the command palette.
  - [ ] Create multiple channels with the "Create More" flag set to true.
- **Resources Toolbar**
  - [ ] Open a channel plot by double-clicking it.
  - [ ] Drag and drop a channel onto a line plot.
  - [ ] Drag and drop a channel onto the line plot toolbar.
  - **Context Menu**
    - [ ] Rename a channel.
    - [ ] Group multiple channels.
    - [ ] Edit the calculation of a calculated channel.
    - [ ] Set an alias for a channel under a range.
    - [ ] Clear an alias for a channel under a range.
    - [ ] Delete a channel.
    - [ ] Copy a link to a channel.
    - [ ] Hard reload the console.
- **Search and Command Palette**
  - [ ] Open a channel plot by its name.
  - [ ] Open the "Create Channel" modal.
  - [ ] Open the "Create Calculated Channel" modal.
- **Calculated Channels**
  - [ ] Plot a basic calculated channel.
  - [ ] Plot a nested calculated channel.
  - [ ] Intentionally create a channel with an erroneous expression, plot it and make sure the Console and Core remain stable and the error is logged to the Core and the Console.
  - [ ] Plot a calculated channel that uses channels with hyphenated names.
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

### Clusters

I can successfully:

- **Connect Cluster Modal**
  - [ ] Connect to a cluster.
- **Dropdown**
  - [ ] Add a new cluster.
  - [ ] Connect to a cluster by selecting it.
  - [ ] Disconnect from a cluster by selecting it.
  - [ ] View the default local cluster in the selector.
  - [ ] View the default demo cluster in the selector.
  - **Context Menu**
    - [ ] Connect to a cluster.
    - [ ] Disconnect the active cluster.
    - [ ] Rename a cluster.
    - [ ] Remove a cluster.
    - [ ] Copy a link to a cluster.
    - [ ] Hard reload the console.
- **Search and Command Palette**
  - [ ] Open the "Connect Cluster" modal.
- [ ] Open a cluster from a link.
- [ ] Receive meaningful feedback when a cluster connection fails.

### Devices

I can successfully:

- **Resources Toolbar**
  - [ ] See a device's state get updated.
  - **Context Menu**
    - [ ] Group devices.
    - [ ] Configure an unconfigured device.
    - [ ] Change the identifier on a configured device.
    - [ ] Rename a device.
    - [ ] Delete a device.

### Documentation

I can successfully:

- [ ] Open documentation from the command palette.
- [ ] Open documentation from the question mark icon.
- [ ] Close and reopen documentation in the same place as left off.

### Labels

I can successfully:

- **Search and Command Palette**
  - [ ] Open the "Edit Label" modal.
- **Edit Label Modal**
  - [ ] Add a new label.
  - [ ] Edit an existing label's name.
  - [ ] Change the color of an existing label.
  - [ ] Rename a label and ensure the change synchronizes with the range toolbar.
  - [ ] Change a label's color and ensure the change synchronizes with the range toolbar.

### Layout

I can successfully:

- [ ] Drag and drop a mosaic leaf into a new window.
- [ ] Rename a tab by double-clicking its name.
- [ ] Close layout tabs by clicking the close icon.
- **Context Menu**
  - [ ] Split a mosaic leaf horizontally.
  - [ ] Split a mosaic leaf vertically.
  - [ ] Focus on a leaf.
  - [ ] Rename a mosaic leaf.
  - [ ] Open a leaf in a new window.
  - [ ] Move a mosaic leaf to the main window from a secondary window.
- [ ] Rename a tab with `Cmd + E`.
- [ ] Close layout tabs with `Cmd + W`.
- [ ] Focus using `Cmd + L`.
- [ ] Open in a new window with `Cmd + O`.
- [ ] Create a new mosaic leaf with `Cmd + T`.
- **Search and Command Palette**
  - [ ] Toggle the color theme.

### Line Plots

I can successfully:

- [ ] Create a new line plot from the mosaic.
- **Visualization**
  - [ ] Plot a historical range of data.
  - [ ] Plot a live range of data.
  - [ ] Move channels between axes.
  - [ ] Adjust the line thickness.
  - [ ] Relabel a line.
  - [ ] Set the plot title.
  - [ ] Download a range as a CSV.
  - [ ] Create a range from line plot selection.
  - [ ] Use the measuring tool on the line plot.
  - [ ] Rename a line plot from its tab title.
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
  - [ ] Create a new line plot.
  - [ ] Open the "Import Line Plot" dialog.
- [ ] Open a line plot from its link.
- [ ] Import a line plot.
- [ ] Rename a line plot and ensure synchronization across:
  - Resources Toolbar
  - Mosaic Tab
  - Visualization Toolbar

### Logs

I can successfully:

- [ ] Create a new log from the visualization selector.
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
  - [ ] Create a new log.
  - [ ] Open an existing log.
- [ ] Open a log from its link.
- [ ] Rename a log and ensure synchronization across:
  - Visualization Toolbar
  - Resources Toolbar
  - Mosaic Tab Name

### Ontology

I can successfully:

- [ ] Create a new group and move resources to it.
- [ ] Move resources between groups.
- [ ] Rename a group.
- [ ] Delete a group.

### Permissions

I can successfully:

- **As a user without schematic permissions:**
  - [ ] Cannot open the "Create Schematic" dialog from the command palette.
  - [ ] Cannot create a new schematic from the workspace resources toolbar.
  - [ ] Cannot import a schematic from the workspace resources toolbar.
  - [ ] Cannot import a schematic via drag-and-drop.
  - [ ] Can actuate valves on a schematic.
  - [ ] Cannot switch to edit mode on a schematic.
- **As a user without admin permissions:**
  - [ ] Cannot open the "Register User" dialog from the command palette.
  - [ ] Cannot delete users, open the permissions dialog, or change a username from the resources toolbar.

### Racks

I can successfully:

- **Devices Toolbar**
  - [ ] See a rack's state get updated.
  - **Context Menu**
    - [ ] Rename a rack.
    - [ ] Delete a rack.
    - [ ] Copy a rack's key.
    - [ ] Create a control sequence from a rack.

### Ranges

I can successfully:

- **Create Range Modal**
  - [ ] Create a new local range.
  - [ ] Create a new persisted range.
  - [ ] Create a range with a parent range.
  - [ ] Add labels while creating a range.
  - [ ] Update the start and end times through changing the stage of a range
- **Range Details**
  - [ ] Rename a range from the tab name.
  - [ ] Rename a range.
  - [ ] Navigate to a parent range from a range
  - [ ] Copy Python code
  - [ ] Copy TypeScript code
  - [ ] Copy a link to the range
  - [ ] Open and successfully download data as a CSV
  - [ ] Favorite the range
  - [ ] Unfavorite the range
  - [ ] Change start and end times.
  - [ ] Change start and end times via the stage
  - [ ] Add labels.
  - [ ] Remove labels.
  - **Child Ranges**
    - [ ] Click and navigate to a child range
    - [ ] Create child ranges
    - [ ] Change the stage of a child range
    - [ ] Favorite a child range
    - [ ] Unfavorite a child range
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
  - [ ] Open the Range Explorer
- **Range Toolbar**
  - [ ] Open the "Create Range" modal from the toolbar.
  - [ ] Open the Range Explorer from the toolbar
  - [ ] Switch the active range by clicking it.
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
  - [ ] Favorite and unfavorite ranges and see them added to the range toolbar
  - [ ] Change the stage of a range
  - **Context Menu**
    - [ ] Rename a range
    - [ ] Create a child range
    - [ ] Favorite a range
    - [ ] Favorite multiple ranges
    - [ ] Unfavorite a range
    - [ ] Unfavorite multiple ranges
    - [ ] Copy the link to a range
    - [ ] Delete a range
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

- [ ] Create a new schematic from the mosaic.
- **Visualization**
  - [ ] Display live data on a value.
  - [ ] Actuate a valve.
  - [ ] Select and change the color of multiple elements.
  - [ ] View the list of writers in control on the schematic.
  - [ ] Acquire absolute control over a control sequence.
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
  - [ ] Open an existing schematic.
  - [ ] Create a new schematic.
  - [ ] Import a schematic from a file.
- [ ] Rename a schematic and ensure synchronization across:
  - Mosaic Tab
  - Resources Toolbar
  - Visualization Toolbar
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
- **Search and Command Palette**
  - [ ] Open the "Create Symbol" modal.
  - [ ] Create a symbol from the "Create Symbol" modal.

### Tables

I can successfully:

- [ ] Create a new table from the mosaic.
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
  - [ ] Create a new table.
  - [ ] Import a table from a file.
- [ ] Rename a table and ensure synchronization across:
  - Mosaic Tab
  - Resources Toolbar
  - Visualization Toolbar

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
  - [ ] Open the "Permissions" dialog.
  - [ ] Rename a user.
  - [ ] Delete a user.
  - [ ] Delete multiple users.
- **Search and Command Palette**
  - [ ] Register a new user.
- [ ] Change a user's username and log in with the new username.

### Arc

I can successfully:

- **Search and Command Palette**
  - [ ] Created a named arc automation.
  - [ ] Open an existing arc automation.

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

- **Status Notifications**
  - [ ] See status notifications in the bottom right corner when creating a new status.

### Version

I can successfully:

- [ ] View the correct version in the bottom navbar.
- [ ] Verify that the auto-update functionality works correctly.

### Workspaces

I can successfully:

- [ ] Create a new workspace.
- [ ] Import a workspace by drag and dropping from a directory.
- **Workspace Selector**
  - [ ] Create a new workspace.
  - [ ] Switch workspaces in the selector.
  - [ ] Clear workspaces from the selector.
- **Resources Toolbar**
  - [ ] Switch workspaces in the resources view.
  - **Context Menu**
    - [ ] Rename a workspace.
    - [ ] Delete a workspace.
    - [ ] Export a workspace.
    - [ ] Create a new line plot in a workspace.
    - [ ] Create a new log in a workspace.
    - [ ] Create a new schematic in a workspace.
    - [ ] Create a new table in a workspace.
    - [ ] Import a line plot.
    - [ ] Import a schematic.
    - [ ] Import a log.
    - [ ] Import a table.
- [ ] Open a workspace from a link.
- **Search and Command Palette**
  - [ ] Open the "Create Workspace" dialog.
  - [ ] Import a workspace.
  - [ ] Open an existing workspace.
- [ ] Rename a workspace and ensure synchronization across:
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
- [ ] Recognize and connect to an NI device locally.
- [ ] Recognize and connect to NI devices over the network.
- [ ] Recognize and connect to physical and simulated devices.
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
- [ ] Connect to an unencrypted OPC UA server.
- [ ] Connect to an encrypted OPC UA server.
- [ ] Create additional channels and move them to existing sampling groups.
- [ ] Move and rename channels.
- **Read Task**
  - **Single Sampling**
    - [ ] Read from multiple channels.
    - [ ] Autogenerate timestamps on the driver.
    - [ ] Read timestamps from the OPC UA server.
  - **Array Sampling**
    - [ ] Read from multiple channels.
    - **Test the following array sizes:**
      - [ ] 1
      - [ ] 10
      - [ ] 100
    - [ ] Autogenerate timestamps on the driver.
    - [ ] Read timestamps from the OPC UA server.
    - [ ] Avoid driver crashes when improper array sizes are specified.
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
- [ ] Connect to a Modbus TCP server.
- [ ] Configure connection parameters (IP address, port, unit ID).
- **Read Task**
  - [ ] Read holding registers from a Modbus server.
  - [ ] Read input registers from a Modbus server.
  - [ ] Read coils and discrete inputs from a Modbus server.
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
