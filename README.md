<a href="https://synnaxlabs.com/" style="display: flex; justify-content: center;">
    <img src="lyra/static/logo/icon-white-padded.png" width="20%"/>
</a>

# Synnax

Synnax is mission control and data review for hardware operations. It is one platform
for real-time data acquisition, hardware control, visualization, and analysis. Synnax
records every sample at full rate and gives engineers live plots, operator schematics,
control, automated sequences, and data review. Synnax can be accessed programmatically
through our client libraries and visually through our Console application.

## Download

[Synnax Desktop](https://docs.synnaxlabs.com/reference/get-started/install-desktop) is
free for any use, including commercial use, up to 50 channels. It runs on macOS (Apple
silicon) and Windows. It connects to LabJack, EtherCAT, and HTTP devices, OPC UA and
Modbus TCP servers, and NI hardware on Windows. Teams that share one system use
[Synnax Enterprise](https://docs.synnaxlabs.com/reference/get-started/editions).

## Documentation

All official documentation can be found on our [website](https://docs.synnaxlabs.com).
If you are interested in building on top of Synnax, please read our
[Contributing Guide](CONTRIBUTING.md) and our [technical documentation](docs/tech).

## Development status

Synnax is currently under active development. The APIs are stable and are unlikely to
change significantly.

Releases follow [semantic versioning](https://semver.org/). Versions with the same minor
version number (e.g. 0.40.1 and 0.40.2) are guaranteed to maintain the same API, while
releases with different minor version numbers (e.g. 0.40.0 and 0.41.0) may change APIs.

Our team is targeting a v1 release before the end of 2026, at which point all APIs will
be considered stable and will not change until a v2 release.

## Repository organization

Synnax is built as a collection of several projects, all of which are collected in this
monorepo. The following is a summary of each:

- [Alamos](alamos) - Dependency-injected code instrumentation that provides
  observability into the Synnax platform.
- [Arc](arc) - A domain-specific language and compiler for hardware automation and
  control sequences, with a reactive execution model and a WebAssembly compilation
  target.
- [Cesium](cesium) - An embedded time series database engine optimized for
  high-performance reads and writes of time series sensor data.
- [Client](client) - Client libraries for communicating with the Synnax server in C++,
  Python, and TypeScript.
- [Console](console) - A data-visualization and graphical control application for macOS
  and Windows.
- [Documentation Site](site/docs) - The code for the Synnax documentation website.
- [Portal](site/portal) - The account site at portal.synnaxlabs.com, where users manage
  licenses and Synnax Desktop machines.
- [Technical Documentation](docs/tech) - Technical documentation such as RFCs and
  contribution guides.
- [Drift](drift) - Redux state synchronization and declarative window management for
  building multi-window Tauri applications.
- [Driver](driver) - An application that connects to NI, LabJack, EtherCAT, and HTTP
  devices and OPC UA and Modbus servers, and runs control sequences on real-time
  operating systems.
- [Freighter](freighter) - A protocol-agnostic network transport for cross-language
  unary and streaming communication with implementations in several languages (C++, Go,
  Python, and TypeScript) and protocols (GRPC, HTTP, and WebSockets).
- [Integration](integration) - A Python-based test conductor framework for end-to-end
  integration and stress testing across the Synnax server and its clients.
- [Oracle](oracle) - A schema definition language, code generator, and migration CLI
  used to keep Synnax's data models consistent across Go, TypeScript, Python, and C++.
- [Lyra](lyra) - The React UI primitives (buttons, inputs, forms, menus, theming) shared
  by Pluto, the Console, and the documentation site.
- [Pluto](pluto) - A React component library for building modular user interfaces on top
  of the Synnax telemetry engine.
- [Core](core) - The core Synnax server, which integrates Cesium and Gorp to provide a
  unified telemetry system.
- [X](x) - Common utilities used by other projects.

# Attributions

The Synnax server uses [Pebble](https://github.com/cockroachdb/pebble), and usage must
follow their provided [disclaimer](licenses/BSD-3-Clause.txt).
