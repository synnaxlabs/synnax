# Release Candidate Pull Request Template

## Key Information

- **Platform Version**:

## Versioning

### Public Packages

#### Version Consistency

I have verified that the following packages have the same minor version number:

- [ ] x/ts
- [ ] alamos/ts
- [ ] freighter/ts
- [ ] client/ts
- [ ] alamos/py
- [ ] freighter/py
- [ ] client/py
- [ ] synnax server
- [ ] drift
- [ ] pluto
- [ ] console
- [ ] root package.json file

#### Version Conflicts

I have verified that, when released, the following packages will not conflict
with any previously released packages:

- [ ] x/ts
- [ ] alamos/ts
- [ ] freighter/ts
- [ ] client/ts
- [ ] alamos/py
- [ ] freighter/py
- [ ] client/py
- [ ] synnax server
- [ ] drift
- [ ] pluto
- [ ] console

## CI Tests

- [ ] I have verified that all CI tests pass.

## CD Builds

- [ ] I have verified that all CD builds pass.

## User Documentation

### Content Changes

I have verified that user facing documentation for each of the following
services has been updated to match any changes in the release candidate. If not,
I will note why the documentation has been left stale.

- [ ] reference/concepts
- [ ] references/cluster
- [ ] references/python-client
- [ ] references/typescript-client
- [ ] references/console
- [ ] references/pluto
- [ ] references/device-drivers/opcua
- [ ] references/device-drivers/ni

### Examples

I have verified that code examples for each of the following services run
correctly:

- [ ] client/py
- [ ] client/ts
- [ ] pluto

## Console

### Auto Update

- [ ] I have verified that the nightly builds of the Console are able to
  auto-update correctly.

### Cluster Connection

I can successfully:

- [ ] Open connect a cluster dialog from cluster toolbar.
- [ ] Open connect a cluster dialog from command search bar.
- [ ] Test a cluster connection from connect a cluster dialog.
- [ ] Connect a cluster by selecting it from cluster toolbar.
- [ ] Connect a cluster by opening context menu from cluster toolbar.
- [ ] Disconnect from a cluster by selecting it from cluster toolbar.
- [ ] Disconnect a cluster by selecting it from the cluster toolbar.
- [ ] Copy a link to a cluster and open it from outside Synnax.
- [ ] Remove a cluster.
- [ ] Rename a cluster.

### Ranges

I can successfully:

- [ ] Open create range dialog from command search bar.
- [ ] Open create range dialog from range toolbar.
- [ ] Open create range dialog from context menu in range toolbar.
- [ ] Create a new local range.
- [ ] Create a new persisted range.
- [ ] Save a local range to Synnax in the range toolbar.
- [ ] Switch the active range in the range toolbar.
- [ ] Load a local range from the search bar.
- [ ] Load a persisted range from the search bar.
- [ ] Rename a range from the range toolbar.
- [ ] Edit a range from the range toolbar.
- [ ] Remove a range from the range toolbar.
- [ ] Delete a persisted range from the range toolbar.
- [ ] Delete a range in the resources view.
- [ ] Delete multiple ranges in the resources view.
- [ ] Set a range as an active range from the resources view.
- [ ] Edit a range from the resources view.
- [ ] Add a range to a plot from the resources view.
- [ ] Copy a link to a range and open it from the resources view.
- [ ] Rename a range from the range toolbar.

### Channels

I can successfully:

- [ ] Create a new channel from the search bar.
- [ ] Rename a channel in the resources view.
- [ ] Delete a channel in the resources view.
- [ ] Delete multiple channels in the resources view.
- [ ] Set the alias for a channel under a range.
- [ ] Clear the alias for a channel uner a range.
- [ ] Copy a link to a channel in the resources view and open it from outside
  Synnax.

### Line Plots

I can successfully:

- [ ] Create a new line plot from the mosaic.
- [ ] Move the line plot to a new mosaic.
- [ ] Rename a line plot from the resourcs view.
- [ ] Rename a line plot from its tab title.
- [ ] Copy the link to a line plot and open it from the resources view.
- [ ] Create a new line plot from the search bar.
- [ ] Create a new line plot from a workspace in the resources view.
- [ ] Plot a historical range of data.
- [ ] Plot a live range of data.
- [ ] Move channels between axes.
- [ ] Adjust the thickness of a line.
- [ ] Re-label a line.
- [ ] Set the title of the plot.
- [ ] Download a range as a CSV.
- [ ] Create a range from line plot selection.
- [ ] Use the measuring tool on the line plot.

### Schematics

I can successfully:

- [ ] Create a new schematic from the mosaic.
- [ ] Create a new schematic from the workspace context menu in the resources
  view.
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

### OPC UA

#### Configuration

I can successfully:

- [ ] Connect to an OPC UA server running unencrypted.
- [ ] Connect to an OPC UA server running encrypted.
- [ ] Create additional sampling groups and move channels to them.
- [ ] Create additional channels and move them to existing sampling groups.
- [ ] Move and rename channels.
- [ ] Save the device configuration.

#### Read Task

I can successfully:

- [ ] Single Sampling - Read from multiple channels.
- [ ] Single Sampling - Auto-generate timestamps on the driver.
- [ ] Single Sampling - Read timestamps from the OPC UA server.
- [ ] Array Sampling - Read from multiple channels in array-sampling mode.
- [ ] Array Sampling - Read from multiple channels in array-sampling mode with
  different array sizes. 
    - [ ] 1
    - [ ] 10
    - [ ] 100
- [ ] Array Sampling - Auto-generate timestamps on the driver.
- [ ] Array Sampling - Read timestamps from the OPC UA server.
- [ ] Array Sampling - The driver will not crash if I specify an improper array size.
- [ ] Channel Selection - The read task dialog will recommend synnax channels
based on the configured OPC UA node.
- [ ] Connect to a physical device and read data from it.
- [ ]	Maintain operation of the driver when a device is disconnected or a channel is 
removed from the device while it is running.
- [ ]	Enable and disable data saving at will.

### National Instruments

#### Configuration

I can successfully:
- [ ] Recognize and connect to a National Instruments device available on local
  machine.
- [ ] Recognize and connect to a National Instruments devices available on
  network.
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
- [ ] Disconnect a device while tasks are active and provide meaningful feedback
  in the task dialogue.
- [ ] Begin several tasks at different times and see them all plotting live
  data.
- [ ] Enable and disable data saving at will.
- [ ] verify there is no lag between sensor input and data written to sever.

##### Special Purpose Channels

- [ ] I can succesfully configure and run an analog read task for each of the
  following channels:
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
          - [ ] all strain guage configurations.
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
- [ ] Disconnect a device while tasks are active and provide meaningful feedback
  in the task dialogue.
- [ ] Configure response time based on state rate specified:
     - [ ] 1 Hz (should have visible delay in response)
     - [ ] 20 Hz (should feel almost immediate)

#### General Usage

I can successfully:

- [ ] Run multiple types of tasks on a single device
- [ ] Run multiple tasks across multiple devices simultaneously
- [ ] I can reliably stream data at the following sample rates
     = [ ] 1 Hz
     - [ ] 10 Hz
     - [ ] 50 Hz
     - [ ] 100 Hz
     - [ ] 500 Hz
     - [ ] 1 kHz
     - [ ] 2 kHz
     - [ ] 5 kHz
- [ ] Configure the following stream rates
     - [ ] 1 Hz
     - [ ] 5 Hz
     - [ ] 10 Hz
     - [ ] 20 Hz
     - [ ] 30 Hz

#### Error Handling

I can successfully:

- [ ] Pass in an invalid device configuration and receive meaningful feedback.
     - [ ] Invalid ports
     - [ ] Invalid task type for devices (e.g. analog read on a analog output
       device)
     - [ ] Out of range values
     - [ ] Multiple tasks using the same channel

### Other

I can successfully:
- [ ] Toggle the color theme from the command search bar.
