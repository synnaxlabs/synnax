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

- [ ] Open a range from its link.
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

- [ ] Open a task configuration from a link.
- [ ] Start a task on server boot up when the "Auto start" option is enabled.
- [ ] Import a task from a file via the import task commands
- [ ] Import a task from a file via drag-and-drop

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
- [ ] View a chassis and it's child devices.
- [ ] Shut down the driver without errors during embedded operation.
- **Digital Read Task**
  - [ ] Plot live data.
- **Digital Write Task**
  - [ ] Perform control actions using a schematic.
  - **Configure response time for specified state rates:**
    - [ ] 1 Hz (visible delay)
    - [ ] 20 Hz (near-instant response)

### OPC UA

I can successfully:

- [ ] Enable and disable OPC UA integration when starting the server.
- [ ] Connect to and read data from a physical device.

### Modbus

I can successfully:

- [ ] Enable and disable Modbus integration when starting the server.
- [ ] Connect to and read data from a physical device.
- [ ] Run tasks across multiple Modbus servers concurrently.
