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
- [ ] [`configs/eslint`](https://www.npmjs.com/package/eslint-config-synnaxlabs)
- [ ] [`configs/stylelint`](https://www.npmjs.com/package/stylelint-config-synnaxlabs)
- [ ] [`configs/ts`](https://www.npmjs.com/package/@synnaxlabs/tsconfig)
- [ ] [`configs/vite`](https://www.npmjs.com/package/@synnaxlabs/vite-plugin)
- [ ] [`console`](https://github.com/synnaxlabs/synnax/releases)
- [ ] [`core`](https://github.com/synnaxlabs/synnax/releases)
- [ ] [`drift`](https://www.npmjs.com/package/@synnaxlabs/drift)
- [ ] [`freighter/py`](https://pypi.org/project/synnax-freighter/)
- [ ] [`freighter/ts`](https://www.npmjs.com/package/@synnaxlabs/freighter)
- [ ] [`media`](https://npmjs.com/package/@synnaxlabs/media)
- [ ] [`pluto`](https://npmjs.com/package/@synnaxlabs/pluto)
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
- [ ] Open a channel plot from a link.
- [ ] Rename a channel and ensure the change synchronizes properly across:
  - Task Configuration Dialog
- [ ] Set an alias for a channel and ensure the change synchronizes properly across:
  - Log Visualization Toolbar
  - Schematic Visualization Toolbar
  - Table Visualization Toolbar
  - Task Configuration Dialog
- [ ] Remove an alias for a channel and ensure the change synchronizes properly across:
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
  - [ ] Receive meaningful error feedback when logging in with invalid credentials.
  - [ ] Add a new core using the "+" button in the core list header.
  - [ ] See connection status indicators for each core in the list.
- **User Badge**
  - [ ] See the user avatar and username in the top-right corner after logging in.
  - [ ] Click the user badge to open the logout menu.

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

### Layout

I can successfully:

- [ ] Drag and drop a mosaic leaf into a new window.
- [ ] Open in a new window with `Cmd + O`.
- **Context Menu**
  - [ ] Open a leaf in a new window.
  - [ ] Move a mosaic leaf to the main window from a secondary window.

### Line Plots

I can successfully:

- **Visualization**
  - [ ] Use the measuring tool on the line plot.
- [ ] Open a line plot from its link.

### Logs

I can successfully:

- **Visualization**
  - [ ] Scroll to view historical data.
  - [ ] Pause and resume scrolling using the streaming icon in the top right.
  - [ ] Switch the logging channel and observe data switching.
  - [ ] Preserve log data from a virtual channel in the buffer.
- [ ] Open a log from its link.

### Permissions & Roles

I can successfully:

- **Role Management**
  - [ ] View all available roles in the Resources Toolbar.
  - [ ] Unassign a role from a user.
  - [ ] Cannot delete built-in roles (Owner, Engineer, Operator, Viewer).

- **As an Owner:**
  - [ ] Can register new users and assign roles.
  - [ ] Can create, edit, and delete all resource types.

- **As a Viewer:**
  - [ ] Can view schematics, line plots, tables, logs.
  - [ ] Cannot actuate valves on a schematic.

### Ranges

I can successfully:

- **Range Explorer**
  - [ ] Search ranges
  - [ ] Filter ranges by labels

- [ ] Open a range from its link.
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

- **Visualization**
  - [ ] Display live data on a value.
- **Search and Command Palette**
  - [ ] Import a schematic from a file.

### Schematic Symbols

I can successfully:

- **Symbol Editor**
  - [ ] Have multiple color regions for an actuator.
- **Schematic Symbols Toolbar**
  - [ ] Import a symbol group.
  - [ ] Import a symbol to a symbol group.
    - **Context Menu**
      - [ ] Export a symbol group.

### Tables

I can successfully:

- **Visualization**
  - [ ] See live data in a table.
  - [ ] See redlines react to live data.

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
  - [ ] Unassign a role from a user.
  - [ ] Open the "Permissions" dialog.
  - [ ] Rename a user.
  - [ ] Delete a user.
  - [ ] Delete multiple users.
- [ ] Change a user's username and log in with the new username.
- [ ] Change a user's role and verify their permissions change accordingly.

### Arc

I can successfully:

- **Arc Toolbar**
  - [ ] Toggle Arc toolbar visibility with "A" keyboard shortcut

- **Arc Editor**
  - [ ] Create an alarm automation that changes statuses and includes the following blocks: channel source, constant, comparison, stable for, select, and status change.

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

- [ ] Verify that the auto-update functionality works correctly.

### Workspaces

I can successfully:

- [ ] Import a workspace by drag and dropping from a directory.
- [ ] Open a workspace from a link.
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
- [ ] Connect to an encrypted OPC UA server.
- [ ] Move and rename channels.
- **Read Task**
  - **Single Sampling**
    - [ ] Read timestamps from the OPC UA server.
    - **Test the following array sizes:**
      - [ ] 10
      - [ ] 100
    - [ ] Read timestamps from the OPC UA server.
  - [ ] Obtain recommended Synnax channels based on the configured OPC UA node.
  - [ ] Connect to and read data from a physical device.
  - [ ] Maintain Driver operation during device disconnection or channel removal while a task is running.
  - [ ] Enable and disable data saving.
- **Write Task**
  - [ ] Perform control and verify changes on the connected OPC UA server.
  - [ ] Enable and disable data saving.
  - [ ] Perform a write operation on an encrypted server.

### Modbus

I can successfully:

- [ ] Enable and disable Modbus integration when starting the server.
- **Read Task**
  - [ ] Plot live data from Modbus registers.
  - [ ] Apply scaling to register values.
  - [ ] Enable and disable data saving.
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
- [ ] Run simultaneous read and write tasks on the same device.
- [ ] Run tasks across multiple Modbus servers concurrently.
