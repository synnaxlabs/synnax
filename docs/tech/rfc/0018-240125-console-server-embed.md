# 18 - Embed Local Server in Console

- **Feature Name**: Embed Local Server in Console
- **Start Date**: 2024-01-25
- **Authors**: Emiliano Bonilla
- **Status**: Draft

# 0 - Summary

This short RFC outlines the technical design for embedding a local instance of Synnax
server into the Console for introductory and benchtop usage. This will make it
dramatically easier for less software-savvy users to get started with Synnax. This RFC
will cover:

    1. Embedding the server and automating the process with CI.
    2. Managing the server's lifecycle from within the Console.
    3. Changes we need to make to the UI.

# 1 - Vocabulary

- **Server** - The core Synnax server (also known as "Synnax Database" or "Synnax
  Engine") that serves reads and writes of telemetry, stores metadata, and handles
  control negotiation.
- **Console** - A desktop application for controlling and visualizing data within a
  Synnax cluster.

# 2 - Design

## 2.0 - Embedding the Server

To embed the Synnax server into the Console, we'll use Tauri's
[sidecar](https://tauri.app/v1/guides/building/sidecar/) feature set. All we need to do
is place a binary for each platform into the `src-tauri/bin` directory and make a few
changes to the `tauri.conf.json` file. Integrating this into CI will require pulling the
latest server binaries from the Synnax repository and placing them into the appropriate
directory. This part of the process shouldn't be too challenging.

One of the major concerns with embedding the server is ensuring that the running console
has sufficient permissions to actually execute the binaries. We'll need to test on all
three operating systems to ensure everything works. There also may be unexpected
permissions issues on different users' machines. We'll need to be prepared to correctly
communicate and resolve these issues.

Another, minor issue with embedding the server is that the server binaries are about
50MB each (90MB on Windows). This will increase the size of the Console download
substantially, but shouldn't be a major issue.

## 2.1 - Keeping the Server Alive

There are four major pieces to starting and keeping the server alive:

1. Starting the server when the Console is opened.
2. If the server crashes, restart it with any changes parameters.
3. Shutting down the server when the Console closes.
4. Making sure we correctly clean up any stranded P&IDs.

### 2.1.1 - Starting the Server
