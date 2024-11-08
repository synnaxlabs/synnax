# Release Candidate Pull Request

## Key Information

- **Version Number**: <!-- MAJOR.MINOR.PATCH -->

## Versioning

### Version Consistency

I have verified that the following files have the same minor version number:

- [ ] `alamos/py`
- [ ] `alamos/ts`
- [ ] `client/py`
- [ ] `client/ts`
- [ ] `console`
- [ ] `drift`
- [ ] `freighter/py`
- [ ] `freighter/ts`
- [ ] `media`
- [ ] `pluto`
- [ ] `server`
- [ ] `x/ts`
- [ ] Root `package.json` file

### Version Conflicts

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

## User Documentation

### Content Changes

I have verified that user-facing documentation for each of the following services has been updated to match any changes in the release candidate:

- [ ] `reference/cluster`
- [ ] `reference/concepts`
- [ ] `reference/console`
- [ ] `reference/control`
- [ ] `reference/device-drivers/labjack`
- [ ] `reference/device-drivers/ni`
- [ ] `reference/device-drivers/opc-ua`
- [ ] `reference/pluto`
- [ ] `reference/python-client`
- [ ] `reference/typescript-client`

### Examples

I have verified that code examples for each of the following services run correctly:

- [ ] `client/py`
- [ ] `client/ts`
  - [ ] Version number in `examples/node` is up-to-date
- [ ] `drift/electron`
- [ ] `drift/tauri`
- [ ] `pluto`

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
    - [ ] Delete a channel.
    - [ ] Delete multiple channels.
    - [ ] Group channels.
    - [ ] Add a channel to an active plot.
    - [ ] Add multiple channels to an active plot.
    - [ ] Add a channel to a new plot.
    - [ ] Add multiple channels to a new plot.
    - [ ] Set an alias for a channel under a range.
    - [ ] Clear an alias for a channel under a range.
    - [ ] Copy a link to a channel.
- **Search and Command Palette**
  - [ ] Open a channel plot by its name.
  - [ ] Open the "Create Channel" modal.
- [ ] Open a channel plot from a link.
- [ ] Rename a channel and ensure the change synchronizes properly across:
  - Resources Toolbar
  - Line Plot Visualization Toolbar
  - Log Visualization Toolbar
  - Schematic Visualization Toolbar
  - Task Configuration Dialog
- [ ] Set an alias for a channel and ensure the change synchronizes properly across:
  - Resources Toolbar
  - Line Plot Visualization Toolbar
  - Log Visualization Toolbar
  - Schematic Visualization Toolbar
  - Task Configuration Dialog
- [ ] Remove an alias for a channel and ensure the change synchronizes properly across:
  - Resources Toolbar
  - Line Plot Visualization Toolbar
  - Log Visualization Toolbar
  - Schematic Visualization Toolbar
  - Task Configuration Dialog

### Clusters

I can successfully:

- **Connect Cluster Modal**
  - [ ] Test a cluster connection.
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
- **Search and Command Palette**
  - [ ] Open the "Connect Cluster" modal.
- [ ] Open a cluster from a link.
- [ ] Receive meaningful feedback when a cluster connection fails.

### Devices

I can successfully:

- **Resources Toolbar**
  - **Context Menu**
    - [ ] Group devices.
    - [ ] Configure an unconfigured device.
    - [ ] Rename a device.
    - [ ] Delete a device.

### Documentation

I can successfully:

- [ ] Open documentation from the command palette.
- [ ] Open documentation from the question mark icon.
- [ ] Close and reopen documentation.

### Labels

I can successfully:

- [ ] Open the "Edit Label" modal from the command palette.
- [ ] Add a new label.
- [ ] Edit an existing label's name.
- [ ] Change the color of an existing label.
- [ ] Rename a label and ensure the change synchronizes with the range toolbar.
- [ ] Change a label's color and ensure the change synchronizes with the range toolbar.

### Layout

I can successfully:

- [ ] Drag and drop a mosaic leaf into a new window.
- [ ] Split a mosaic horizontally.
- [ ] Split a mosaic vertically.
- [ ] Rename a tab by double-clicking on its name.
- [ ] Close layout tabs by clicking the close icon.
- **Context Menu**
  - [ ] Split a mosaic horizontally.
  - [ ] Split a mosaic vertically.
  - [ ] Focus on a leaf.
  - [ ] Open a leaf in a new window.
- [ ] Rename a tab with `Cmd + R`.
- [ ] Close layout tabs with `Cmd + W`.
- [ ] Focus using `Cmd + F`.
- [ ] Open in a new window with `Cmd + O`.
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
- **Resources Toolbar**
  - [ ] Open a plot by selecting it.
  - [ ] Drag a plot onto the mosaic.
  - **Context Menu**
    - [ ] Rename a plot.
    - [ ] Delete a plot.
    - [ ] Delete multiple plots.
    - [ ] Export a plot.
    - [ ] Export multiple plots.
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
  - [ ] Switch the logging channel and see data switch.
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

### Ranges

I can successfully:

- **Create Range Modal**
  - [ ] Create a new local range.
  - [ ] Create a new persisted range.
  - [ ] Create a range with a parent range.
  - [ ] Add labels while creating a range.
- **Range Details**
  - [ ] Rename a range.
  - [ ] Rename a range from the tab name.
  - [ ] Change start and end times.
  - [ ] Add labels.
  - [ ] Set metadata.
  - [ ] Open a link from metadata.
  - [ ] Delete metadata.
  - [ ] Add child ranges.
  - [ ] Open snapshots.
  - [ ] Navigate to and from child ranges.
- **Search and Command Palette**
  - [ ] Open an existing range layout window.
  - [ ] Open the "Create Range" dialog.
- **Range Toolbar**
  - [ ] Open the "Create Range" modal from the toolbar.
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
    - [ ] Remove from the range toolbar.
    - [ ] Delete a persisted range.
    - [ ] Copy a link to a persisted range.
    - [ ] Save a local range to Synnax.
- **Resources Toolbar**
  - [ ] Open the range overview dialog by double-clicking a range.
  - **Context Menu**
    - [ ] Set an active range.
    - [ ] Remove an active range.
    - [ ] Open the range layout.
    - [ ] Rename a range.
    - [ ] Open the "Create Range" modal with a child range.
    - [ ] Group ranges.
    - [ ] Add to the active line plot.
    - [ ] Add multiple ranges to the active line plot.
    - [ ] Add to a new line plot.
    - [ ] Add multiple ranges to a new line plot.
    - [ ] Delete a range.
    - [ ] Delete multiple ranges.
    - [ ] Copy a link to a range.
- [ ] Open a range from its link.
- [ ] Rename a range and ensure synchronization across:
  - Resources Toolbar
  - Range Overview
  - Range Overview Tab Name
  - Ranges Toolbar
- [ ] Change the time of a range and ensure synchronization across:
  - Range Details
  - Ranges Toolbar
- [ ] Add or remove child ranges and ensure synchronization across:
  - Resources Toolbar
  - Range Details
- [ ] Snapshot a schematic or task and ensure synchronization across:
  - Resources Toolbar
  - Range Details
- [ ] Ensure channel aliases synchronize correctly across:
  - Resources Toolbar
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
  - [ ] Acquire absolute control over an autosequence.
  - [ ] Copy a link.
  - [ ] Export a schematic.
- **Resources Toolbar**
  - [ ] Double-click a schematic to load it.
  - [ ] Drag a schematic onto the mosaic to load it.
  - **Context Menu**
    - [ ] Export a schematic.
    - [ ] Export multiple schematics.
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
  - [ ] Open the "Create Schematic" dialog.
- [ ] Rename a schematic and ensure synchronization across:
  - Mosaic Tab
  - Resources Toolbar
  - Visualization Toolbar
- [ ] Rename a schematic snapshot and ensure synchronization across:
  - Mosaic Tab
  - Resources Toolbar
  - Visualization Toolbar
  - Range Details Overview

### Tasks

I can successfully:

- **Resources Toolbar**
  - [ ] Open task configuration by double-clicking.
  - **Context Menu**
    - [ ] Snapshot a task to the active range.
    - [ ] Snapshot multiple tasks to the active range.
    - [ ] Rename a task.
    - [ ] Group tasks.
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
- [ ] Grant permissions to a user from the resources toolbar.
- [ ] Change a user's username and log in with the new username.

### Workspaces

I can successfully:

- [ ] Create a new workspace.
- **Workspace Selector**
  - [ ] Create a new workspace.
  - [ ] Switch workspaces in the selector.
  - [ ] Clear workspaces from the selector.
- **Resources Toolbar**
  - [ ] Switch workspaces in the resources view.
  - **Context Menu**
    - [ ] Rename a workspace.
    - [ ] Delete a workspace.
    - [ ] Create a new line plot in a workspace.
    - [ ] Create a new log in a workspace.
    - [ ] Create a new schematic in a workspace.
    - [ ] Import a line plot.
    - [ ] Import a schematic.
- [ ] Open a workspace from a link.
- **Search and Command Palette**
  - [ ] Open the "Create Workspace" dialog.
  - [ ] Open an existing workspace.
- [ ] Rename a workspace and ensure synchronization across:
  - Resources Toolbar
  - Workspace Selector
- [ ] Create a workspace in a previous version of Synnax, add visualizations, and open it in the release candidate.

### Version

I can successfully:

- [ ] View the correct version in the bottom navbar.
- [ ] Verify that the auto-update functionality works correctly.

## Driver

### General

I can successfully:

- [ ] Run the driver for long periods of time with minimal memory leakage. <!-- Let's quantify this!-->
- **Pass in an invalid device configuration and receive meaningful feedback:**
  <!-- Should we add more / only check these 4? -->
  - [ ] Invalid ports
  - [ ] Invalid task type for devices (e.g. analog read on a analog output device)
  - [ ] Out of range values
  - [ ] Multiple tasks using the same channel
- [ ] Shut down the driver with an embedded driver without receiving an error from the driver routine. <!-- Not really sure what this means-->

### LabJack

I can successfully:

- [ ] Enable and disable LabJack integration when starting the server.
- [ ] Recognize and connect to a LabJack device available on local machine.
- [ ] Save device configuration.
- [ ] Run the driver without the LabJack library installed on the machine.
- <!-- we should probably add some edge case tests like disconnecting a device while running -->
- **Read Task**
  <!-- we probably should add more tests here, right? -->
  <!-- like tares, scaling, etc. -->
  - [ ] Plot live analog data
  - [ ] Plot live digital data
  - [ ] Stop, start and reconfigure tasks numerous times
  - [ ] Enable and disable data saving
  - [ ] Verify there is no lag between sensor input and data written to server
  - [ ] Configure and run a read task for a thermocouple
  - [ ] Run a read task with thermocouples, digital and analog channels
  - **Reliably plot data at the following sample rates:**
    <!-- Let's just do like three, if it works at 10 and 100 then it works at 50 -->
    - [ ] 1 Hz
    - [ ] 10 Hz
    - [ ] 50 Hz
    - [ ] 100 Hz
    - [ ] 500 Hz
    - [ ] 1 kHz
    - [ ] 5 kHz
    - [ ] 10 kHz
    - [ ] 50 kHz
- **Write Task**
  <!-- there's probably other tests we can do on configuring it right! -->
  - **Begin a write task and perform control actions with a schematic**:
    - [ ] actuate a valve via a digital input
    - [ ] set an analog output to a specific voltage via a setpoint
  - [ ] Stop, start and reconfigure task at will.
- [ ] configure a write and read task to run simultaneously and stop, delete either one without affecting the other

### NI

I can successfully:

- [ ] Enable and disable NI integration when starting the server.
- [ ] Recognize and connect to a National Instruments device available on local machine.
- [ ] Recognize and connect to a National Instruments devices available on network. <!--what does this mean?>
- [ ] Recognize and connect to a physcial device
- [ ] Recognize and connect to a simulated device
- [ ] Disconnect a physical device from the machine with a task running without faulting.
- [ ] Save device configuration.
- [ ] Not see chassis devices connected to the machine
- [ ] See devices connected to a chassis
- [ ] Run the driver without the Daqmx and Syscfg libraries installed on the machine.
- [ ] Get feedback when trying to create an NI task on a machine that doesn't have the required libraries installed.
- **Pass in an invalid device configuration and receive meaningful feedback:**
  - [ ] Invalid ports
  - [ ] Invalid task type for devices (e.g. analog read on a analog output device)
  - [ ] Out of range values
  - [ ] Multiple tasks using the same channel
- [ ] Shut down the driver with an embedded driver without receiving an error from the driver routine
- [ ] Run multiple types of tasks on a single device
- [ ] Run multiple tasks across multiple devices simultaneously
- **Reliably stream data at the following sample rates**:
  - [ ] 1 Hz
  - [ ] 10 Hz
  - [ ] 50 Hz
  - [ ] 100 Hz
  - [ ] 500 Hz
  - [ ] 1 kHz
  - [ ] 2 kHz
  - [ ] 5 kHz
- **Configure the following stream rates**:
  - [ ] 1 Hz
  - [ ] 5 Hz
  - [ ] 10 Hz
  - [ ] 20 Hz
  - [ ] 30 Hz
- **Analog Read Task**
  - [ ] Plot live data from an analog read task.
  - [ ] Plot live data from a digital read task.
  - [ ] Stop, start and reconfigure task at will.
  - [ ] Disconnect a device while tasks are active and provide meaningful feedback in the task dialogue.
  - [ ] Begin several tasks at different times and see them all plotting live data.
  - [ ] Enable and disable data saving at will.
  - [ ] verify there is no lag between sensor input and data written to sever.
  - [ ] I can succesfully configure and run an analog read task for each of the following channels:
    - [ ] Acceleration
    - [ ] Acceleration 4 wire
    - [ ] Bridge
      - [ ] All bridge configurations
    - [ ] Charge
    - [ ] Current
    - [ ] Force bridge polynomial
    - [ ] Force bridge table
    - [ ] Force bridge two point linear
    - [ ] Force iepe
    - [ ] Microphone
    - [ ] Pressure bridge polynomial
    - [ ] Pressure bridge table
    - [ ] Pressure bridge two point linear
    - [ ] Resistance
    - [ ] RTD
      - [ ] All RTD types.
      - [ ] All resistance configs.
    - [ ] Strain gauge
      - [ ] all strain guage configurations
    - [ ] Built in temperature sensor
    - [ ] Thermocouple
      - [ ] All thermocouple types.
      - [ ] All cjc options.
    - [ ] Torque bridge polynomial
    - [ ] Torque bridge table
    - [ ] Torque bridge two point linear
    - [ ] Velocity iepe
    - [ ] Voltage
      - [ ] All terminal configurations.
  - [ ] I can sucessfully configure the following scales:
    - [ ] Linear
    - [ ] Map
- **Digital Read Task**
  - [ ]
- **Digital Write Task**
- [ ] Begin a digital write task and perform control actions with a schematic.
- [ ] Stop, start and reconfigure task at will.
- [ ] Disconnect a device while tasks are active and provide meaningful feedback in the task dialogue.
- **Configure response time based on state rate specified**:
  - [ ] 1 Hz (should have visible delay in response)
  - [ ] 20 Hz (should feel almost immediate)

### OPC UA

I can successfully:

- [ ] Enable and disable OPC UA integration when starting the server.
- [ ] Connect to an OPC UA server running unencrypted.
- [ ] Connect to an OPC UA server running encrypted.
- [ ] Create additional channels and move them to existing sampling groups.
- [ ] Move and rename channels.
- [ ] Save the device configuration.
- **Read Task**
  - [ ] Single Sampling - Read from multiple channels.
  - [ ] Single Sampling - Auto-generate timestamps on the driver.
  - [ ] Single Sampling - Read timestamps from the OPC UA server.
  - [ ] Array Sampling - Read from multiple channels in array-sampling mode.
  - **Array Sampling - Read from multiple channels in array-sampling mode with different array sizes**:
    - [ ] 1
    - [ ] 10
    - [ ] 100
  - [ ] Array Sampling - Auto-generate timestamps on the driver.
  - [ ] Array Sampling - Read timestamps from the OPC UA server.
  - [ ] Array Sampling - The driver will not crash if I specify an improper array size.
  - [ ] Channel Selection - The read task dialog will recommend synnax channels based on the configured OPC UA node.
  - [ ] Connect to a physical device and read data from it.
  - [ ] Maintain operation of the driver when a device is disconnected or a channel is removed from the device while it is running.
  - [ ] Enable and disable data saving at will.
  - [ ] Leave a task paused for an hour and resume it again without needing to reconfigure.
- **Write Task**
  - [ ] Perform control and see changes reflected on the connected OPC UA server.
  - [ ] Stop, start and reconfigure task at will.
  - [ ] Enable and disable data saving at will.
  - [ ] Leave a write tasking played but idle for an hour and perform control without losing connection.
  - [ ] Perform a write on an encrypted server.
