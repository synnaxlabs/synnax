# Release Candidate Pull Request

## Key Information

- **Version Number**:

## Versioning

### Version Consistency

I have verified that the following files have the same minor version number:

- [ ] alamos/py
- [ ] alamos/ts
- [ ] client/py
- [ ] client/ts
- [ ] console
- [ ] drift
- [ ] freighter/py
- [ ] freighter/ts
- [ ] pluto
- [ ] x/media
- [ ] x/ts
- [ ] root package.json file

#### Version Conflicts

I have verified that, when released, the following packages will not conflict with any previously released packages:

- [ ] [alamos/py](https://pypi.org/project/alamos/)
- [ ] [alamos/ts](https://www.npmjs.com/package/@synnaxlabs/alamos)
- [ ] [client/ts](https://www.npmjs.com/package/@synnaxlabs/client)
- [ ] [client/py](https://pypi.org/project/synnax/)
- [ ] [console](https://github.com/synnaxlabs/synnax/releases)
- [ ] [drift](https://www.npmjs.com/package/@synnaxlabs/drift)
- [ ] [freighter/py](https://pypi.org/project/synnax-freighter/)
- [ ] [freighter/ts](https://www.npmjs.com/package/@synnaxlabs/freighter)
- [ ] [pluto](https://npmjs.com/package/@synnaxlabs/pluto)
- [ ] [server](https://github.com/synnaxlabs/synnax/releases)
- [ ] [x/ts](https://www.npmjs.com/package/@synnaxlabs/x)

## User Documentation

### Content Changes

I have verified that user facing documentation for each of the following services has
been updated to match any changes in the release candidate.

- [ ] reference/concepts
- [ ] reference/cluster
- [ ] reference/python-client
- [ ] reference/typescript-client
- [ ] reference/console
- [ ] reference/control
- [ ] reference/pluto
- [ ] reference/device-drivers/opcua
- [ ] reference/device-drivers/ni

### Examples

I have verified that code examples for each of the following services run correctly:

- [ ] client/py
- [ ] client/ts
  - [ ] version number in examples/node is up-to-date
- [ ] pluto

## Console

### Auto Update

- [ ] I have verified that the nightly builds of the console are able to auto-update
      correctly.

### Clusters

I can successfully:

- Dropdown
  - [ ] Add a new cluster
  - [ ] Connect a cluster by selecting
  - [ ] Disconnect a cluster by selecting
  - Context Menu
    - [ ] Connect a cluster
    - [ ] Disconnect the active cluster
    - [ ] Rename a cluster
    - [ ] Remove a cluster
    - [ ] Copy a link to a cluster
- [ ] Open connect a cluster dialog from the command palette
- [ ] Test a cluster connection in the connect a cluster modal
- [ ] Open a cluster from a link

### Labels

I can successfully:

- [ ] Open the label edit dialog from the command palette.
- [ ] Add a new label.
- [ ] Edit an existing label.

### Ranges

I can successfully:

- Create Range Modal
  - [ ] Create a new local range.
  - [ ] Create a new persisted range.
  - [ ] Set parent range
  - [ ] Add labels
- Range Layout
  - [ ] Rename range.
  - [ ] Rename range from tab.
  - [ ] Change start and end times.
  - [ ] Add labels.
  - [ ] Set metadata.
  - [ ] Delete metadata.
  - [ ] Add child ranges.
  - [ ] Open snapshots.
  - [ ] Navigate to and from child ranges
- Search and Command Palette
  - [ ] Open an existing range layout window
  - [ ] Open create range dialog
- Range Toolbar
  - [ ] Open create range modal from toolbar link when no range exists
  - [ ] Switch the active range by clicking
  - Context Menu
    - [ ] Open create range modal
    - [ ] Rename range
    - [ ] Set active range
    - [ ] Open create range modal with child range
    - [ ] Add to active line plot
    - [ ] Add to new line plot
    - [ ] Remove from range toolbar
    - [ ] Delete persisted range
    - [ ] Copy link of persisted range
    - [ ] Save local range to Synnax
- [ ] Resources Toolbar
  - [ ] Open the range overview dialog by clicking on a range
  - [ ] Context Menu
    - [ ] Set active range
    - [ ] Rename range
    - [ ] Open create range modal with child range
    - [ ] Group ranges
    - [ ] Add to active line plot
    - [ ] Add multiple ranges to active line plot
    - [ ] Add to new line plot
    - [ ] Add multiple ranges to new line plot
    - [ ] Delete range
    - [ ] Delete multiple ranges
    - [ ] Copy link to range
- [ ] Open a range from its url
- [ ] Make changes to a range with resources toolbar, overview, and ranges toolbar open
      and see changes propagate to all three.

### Channels

I can successfully:

- [ ] Create a new channel from the search palette.
- [ ] Create several channels with the 'Create More' flag set to true.
- [ ] Rename a channel in the resources view.
- [ ] Delete a channel in the resources view.
- [ ] Delete multiple channels in the resources view.
- [ ] Set the alias for a channel under a range.
- [ ] Clear the alias for a channel under a range.
- [ ] Copy a link to a channel in the resources view and open it from outside Synnax.

### Line Plots

I can successfully:

- [ ] Create a new line plot from the mosaic.
- [ ] Move the line plot to a new mosaic.
- [ ] Rename a line plot from the resources view.
- [ ] Rename a line plot from its tab title.
- [ ] Copy the link to a line plot and open it from the resources view.
- [ ] Create a new line plot from the search bar.
- [ ] Create a new line plot from a workspace in the resources view.
- [ ] Export and import a line plot.
- [ ] Plot a historical range of data.
- [ ] Plot a live range of data.
- [ ] Move channels between axes.
- [ ] Adjust the thickness of a line.
- [ ] Re-label a line.
- [ ] Set the title of the plot.
- [ ] Download a range as a CSV.
- [ ] Create a range from line plot selection.
- [ ] Use the measuring tool on the line plot.
- [ ] I can repeatedly start and stop an acquisition read task, and tooltips will appear
      in the correct location.

### Schematics

I can successfully:

- [ ] Create a new schematic from the mosaic.
- [ ] Create a new schematic from the workspace context menu in the resources view.
- [ ] Create a new schematic from the command search bar.
- [ ] Move the schematic to a new mosaic.
- [ ] Download a schematic from the context menu.
- [ ] Drag a schematic from the files folder into the mosaic and have it load.
- [ ] Rename a schematic in the resources view.
- [ ] Delete a schematic in the resources context menu.
- [ ] Snapshot a schematic in the resources context menu.
- [ ] Copy a schematic in the resources context menu.
- [ ] Create a new schematic from the search bar.
- [ ] Add a value to the schematic and display live data.
- [ ] Add a valve to the schematic and actuate it.

### Logs

I can successfully:

- [] Create a new log from the mosaic.
- [] Create a new log from the workspace context menu in the resources view.
- [] Create a new log from the command search bar.
- [] Stream data from a virtual channel in the schematic.
- [] Stream data from a persisted channel in the schematic.
- [] Scroll to see historical data.

### Workspaces

I can successfully:

- [ ] Create a new workspace.
- [ ] Switch workspaces in the selector.
- [ ] Switch workspaces in the resources view.
- [ ] Delete a workspace in the resources view.
- [ ] Create a new line plot in a workspace.
- [ ] Create a new schematic in a workspace.
- [ ] Import a schematic from the context menu.
- [ ] Rename a line plot in a workspace.
- [ ] Rename a schematic in a workspace.
- [ ] Delete a line plot in a workspace.
- [ ] Delete a schematic in a workspace.
- [ ] Create a workspace in a previous version of Synnax, add visualizations, and open
      it in the release candidate.

### Resources

I can successfully:

- [ ] Create a new group and move resources to it.
- [ ] Move resources between groups.
- [ ] Rename a group.
- [ ] Delete a group.

### Documentation

I can successfully:

- [ ] Open the documentation from the command search bar.
- [ ] Open the documentation from the question mark icon.

### Users

I can successfully:

- [ ] Register a new user from the command palette
- [ ] Give the user permissions from the resources toolbar
- [ ] Login as the new user and open a schematic without schematic permissions and then
      with schematic permissions

### Devices

I can successfully:

- [ ] Group devices in the resources view.
- [ ] Rename a device in the resources view.
- [ ] Delete a device in the resources view.

### Tasks

I can successfully:

- [ ] Group tasks in the resources view.
- [ ] Rename a task in the resources view.
- [ ] Delete a task in the resources view.

### OPC UA

#### Configuration

I successfully:

- [ ] Updated the license on TwinCat/Beckhoff before continuing QA

I can successfully:

- [ ] Enable and disable OPC UA integration when starting the server.
- [ ] Connect to an OPC UA server running unencrypted.
- [ ] Connect to an OPC UA server running encrypted.
- [ ] Create additional channels and move them to existing sampling groups.
- [ ] Move and rename channels.
- [ ] Save the device configuration.

#### Read Task

I can successfully:

- [ ] Single Sampling - Read from multiple channels.
- [ ] Single Sampling - Auto-generate timestamps on the driver.
- [ ] Single Sampling - Read timestamps from the OPC UA server.
- [ ] Array Sampling - Read from multiple channels in array-sampling mode.
- [ ] Array Sampling - Read from multiple channels in array-sampling mode with different
      array sizes.
  - [ ] 1
  - [ ] 10
  - [ ] 100
- [ ] Array Sampling - Auto-generate timestamps on the driver.
- [ ] Array Sampling - Read timestamps from the OPC UA server.
- [ ] Array Sampling - The driver will not crash if I specify an improper array size.
- [ ] Channel Selection - The read task dialog will recommend synnax channels based on
      the configured OPC UA node.
- [ ] Connect to a physical device and read data from it.
- [ ] Maintain operation of the driver when a device is disconnected or a channel is
      removed from the device while it is running.
- [ ] Enable and disable data saving at will.
- [ ] Leave a task paused for an hour and resume it again without needing to reconfigure.

#### Write Task

I can successfully:

- [ ] Perform control and see changes reflected on the connected OPC UA server.
- [ ] Stop, start and reconfigure task at will.
- [ ] Enable and disable data saving at will.
- [ ] Leave a write tasking played but idle for an hour and perform control without losing connection.
- [ ] Perform a write on an encrypted server.

### National Instruments

#### Configuration

I can successfully:

- [ ] Enable and disable NI integration when starting the server.
- [ ] Recognize and connect to a National Instruments device available on local machine.
- [ ] Recognize and connect to a National Instruments devices available on network.
- [ ] Recognize and connect to physcial and simulated devices.
- [ ] Disconnect a physical device from the machine with a task running without faulting.
- [ ] Save device configuration.
- [ ] Not see chassis devices connected to the machine
- [ ] See devices connected to a chassis
- [ ] Run the driver without the Daqmx and Syscfg libraries installed on the machine.
- [ ] Get feedback when trying to create an NI task on a machine that doesn't have the
      required libraries installed.

#### Read Task

I can successfully:

- [ ] Plot live data from an analog read task.
- [ ] Plot live data from a digital read task.
- [ ] Stop, start and reconfigure task at will.
- [ ] Disconnect a device while tasks are active and provide meaningful feedback in the
      task dialogue.
- [ ] Begin several tasks at different times and see them all plotting live data.
- [ ] Enable and disable data saving at will.
- [ ] verify there is no lag between sensor input and data written to sever.

##### Special Purpose Channels

- [ ] I can succesfully configure and run an analog read task for each of the following
      channels:
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

#### Write Task

I can successfully:

- [ ] Begin a digital write task and perform control actions with a schematic.
- [ ] Stop, start and reconfigure task at will.
- [ ] Disconnect a device while tasks are active and provide meaningful feedback in the
      task dialogue.
- [ ] Configure response time based on state rate specified:
  - [ ] 1 Hz (should have visible delay in response)
  - [ ] 20 Hz (should feel almost immediate)

#### General Usage

I can successfully:

- [ ] Run multiple types of tasks on a single device
- [ ] Run multiple tasks across multiple devices simultaneously
- [ ] Reliably stream data at the following sample rates:
  - [ ] 1 Hz
  - [ ] 10 Hz
  - [ ] 50 Hz
  - [ ] 100 Hz
  - [ ] 500 Hz
  - [ ] 1 kHz
  - [ ] 2 kHz
  - [ ] 5 kHz
- [ ] Configure the following stream rates:
  - [ ] 1 Hz
  - [ ] 5 Hz
  - [ ] 10 Hz
  - [ ] 20 Hz
  - [ ] 30 Hz

#### Error Handling

I can successfully:

- [ ] Pass in an invalid device configuration and receive meaningful feedback.
  - [ ] Invalid ports
  - [ ] Invalid task type for devices (e.g. analog read on a analog output device)
  - [ ] Out of range values
  - [ ] Multiple tasks using the same channel
- [ ] Shut down the driver with an embedded driver without receiving an error from the
      driver routine.

### Other

I can successfully:

- [ ] Toggle the color theme from the command search bar.
- [ ] Run the driver for long periods of time with minimal memory leakage.


### LabJack

#### Configuration

I can successfully:

- [ ] Enable and disable LabJack integration when starting the server.
- [ ] Recognize and connect to a LabJack device available on local machine.
- [ ] Save device configuration.
- [ ] Run the driver without the LabJack library installed on the machine.

#### Read Task

I can successfully:

- [ ] Plot live analog data from a read task
- [ ] Plot live digital data from a read task
- [ ] Stop, start and reconfigure tasks numerous times at will
- [ ] Enable and disable data saving at will
- [ ] Verify there is no lag between sensor input and data written to server
- [ ] Configure and run a read task for a thermocouple
- [ ] Run a read task with thermocouples, digital and analog channels
- [ ] Reiliably plot data at the following sample rates:
  - [ ] 1 Hz
  - [ ] 10 Hz
  - [ ] 50 Hz
  - [ ] 100 Hz
  - [ ] 500 Hz
  - [ ] 1 kHz
  - [ ] 5 kHz
  - [ ] 10 kHz
  - [ ] 50 kHz

#### Write Task

I can successfully:

- [ ] Begin a write task and perform control actions with a schematic.
  - [ ] actuate a valve via a digital input
  - [ ] set an analog output to a specific voltage via a setpoint
- [ ] Stop, start and reconfigure task at will.
- [ ] configure a write and read task to run simultaneously and stop, delete either one without affecting the other