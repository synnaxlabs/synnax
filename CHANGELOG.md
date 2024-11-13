# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

Versions prior to 1.x.x follow modified [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Versions with the same patch (e.g. 0.0.1 and 0.0.2) are guaranteed to maintain the same API, while minor versions may include API changes.

Our team is targeting a v1 release before the end of 2024, at which point all APIs will be stable and follow strict semantic versioning.

## Upcoming

### Additions

- LabJack T-Series support on macOS and Linux.

## v0.34.0 - 2024-11-13

### Additions

- Scientific and engineering notation to values on a schematic.
- Ability to toggle the NI scanner on and off from the command palette.
- [Python client](https://docs-1gg5z500t-synnax.vercel.app/reference/python-client) support for [OPC UA read tasks](https://docs-1gg5z500t-synnax.vercel.app/reference/device-drivers/opc-ua/read-task)
- Configurable font sizes on [schematic](https://docs-1gg5z500t-synnax.vercel.app/reference/console/schematic) values.

### Fixes

- Issues with renaming multiple channels of different types (virtual, free virtual, persisted).
- Issues with deleting channels on Windows.
- Reconnection/disconnection issues with labjack devices that are actively streaming data.
- Logical and physical position calculations issues due to an open issue in Tauri.
- Port indices in the LabJack task configuration dialogue.
- Improved schematic connection line pathing.

## v0.33.0 - 2024-11-04

### Additions

- A [log component](https://docs-1qlj556hd-synnax.vercel.app/reference/console/logs) to the synnax console, allowing you to view channel data in the form of logs.
- Direct [LabJack Integration](https://docs-1qlj556hd-synnax.vercel.app/reference/device-drivers/labjack) for T-Series devices.
  - [Write Tasks](https://docs-1qlj556hd-synnax.vercel.app/reference/device-drivers/labjack/write-task) to control digital and analog outputs from the device.
  - [Read Tasks](https://docs-1qlj556hd-synnax.vercel.app/reference/device-drivers/labjack/read-task) to acquire data from inputs
- A snooze button on the version update notification to silence version updates.
- Taring functionality for National Instruments and LabJack read tasks to zero out channels actively reading from devices.

### Changes

- Upgraded to Tauri v2.
