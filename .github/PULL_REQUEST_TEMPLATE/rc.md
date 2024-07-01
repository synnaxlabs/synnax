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
- [ ] console
- [ ] root package.json file

#### Version Conflicts

I have verified that, when released, the following packages will not conflict with any
previously released packages:

- [ ] x/ts
- [ ] alamos/ts
- [ ] freighter/ts
- [ ] client/ts
- [ ] alamos/py
- [ ] freighter/py
- [ ] client/py
- [ ] synnax server
- [ ] console

## CI Tests

- [ ] I have verified that all CI tests pass.

## CD Builds

- [ ] I have verified that all CD builds pass.

## User Documentation

### Content Changes

I have verified that user facing documentation for each of the following services has
been updated to match any changes in the release candidate. If not, I will note why
the documentation has been left stale.

- [ ] reference/concepts
- [ ] references/cluster
- [ ] references/python-client
- [ ] references/typescript-client
- [ ] references/console
- [ ] references/device-drivers/opcua
- [ ] references/device-drivers/ni

### Examples

I have verified that code examples for each of the following services run correctly:

- [ ] client/py
- [ ] client/ts

## Console

### Auto Update

- [ ] I have verified that the nightly builds of the Console are able to auto-update correctly.

### Cluster Connection

I can successfully:

- [ ] Connect to a cluster.
- [ ] Disconnect from a cluster.
- [ ] Switch clusters in the selector.
- [ ] Remove a cluster.
- [ ] Rename a cluster.

### Ranges

I can successfully:

- [ ] Create a new local range.
- [ ] Create a new persisted range.
- [ ] Save a local range to Synnax.
- [ ] Switch the active range.
- [ ] Load a range from the search bar.
- [ ] Create a range from a line plot selection.
- [ ] Delete a range in the resources view.
- [ ] Delete multiple ranges in the resources view.

### Line Plots

I can successfully:

- [ ] Create a new line plot from the mosaic.
- [ ] Move the line plot to a new mosaic.
- [ ] Create a new line plot from the search bar.
- [ ] Plot a historical range of data.
- [ ] Plot a live range of data.
- [ ] Move channels between axes.
- [ ] Adjust the thickness of a line.
- [ ] Re-label a line.
- [ ] Set the title of the plot.

### Schematics

I can successfully:

- [ ] Create a new schematic from the mosaic.
- [ ] Move the schematic to a new mosaic.
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
- [ ] Rename a line plot in a workspace.
- [ ] Rename a schematic in a workspace.
- [ ] Delete a line plot in a workspace.
- [ ] Delete a schematic in a workspace.

### Resources

I can successfully:

- [ ] Create a new group and move resources to it.
- [ ] Move resources between groups.
- [ ] Rename a group.
- [ ] Delete a group.

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
- [ ] Array Sampling - Auto-generate timestamps on the driver.
- [ ] Array Sampling - Read timestamps from the OPC UA server.
- [ ] Array Sampling - The driver will not crash if I specify an improper array size.
 -[ ] Channel Selection - The read task dialog will recommend synnax channels based on
      the configured OPC UA node.

