<a href="https://synnaxlabs.com/" style="display: flex; justify-content: center;">
    <img src="x/media/static/logo/icon-white-padded.png" width="20%"/>
</a>

# Synnax

Synnax is an observability and control platform designed for high-performance hardware
teams. Synnax aims to control the entire data lifecycle, from reading data from a sensor
to writing bytes to a file, including a visualization engine and mechanisms for
real-time propagation of data, allowing for the control of hardware. Synnax can be
accessed both programatically through our client libraries and visually through our
Console application.

## Documentation

All official documentation can be found on our [website](https://docs.synnaxlabs.com).
If you are interested in building on top of Synnax, please read our
[Contributing Guide](docs/CONTRIBUTING.md) and our [technical documentation](docs/tech).

## Development Status

Synnax is currently under active development. The APIs are stable and are unlikely to
change significantly.

Releases follow [semantic versioning](https://semver.org/). Versions with the same minor
version number (e.g. 0.40.1 and 0.40.2) are guaranteed to maintain the same API, while
releases with different minor version numbers (e.g. 0.40.0 and 0.41.0) may change APIs.

Our team is targeting a v1 release before the end of 2025, at which point all APIs will
be considered stable and will not change until a v2 release.

## Repository Organization

Synnax is built as a collection of several projects, all of which are collected in this
monorepo. The following is a summary of each:

- [Alamos](alamos) - Dependency-injected code instrumentation that provides
  observability into the Synnax platform.
- [Aspen](aspen) - A gossip-based distributed key-value store used for propagating and
  persisting metadata between nodes, such as cluster topology, state, and configuration.
- [Cesium](cesium) - An embedded time series database engine optimized for
  high-performance reads and writes of time series sensor data.
- [Client](client) - Client libraries for communicating with the Synnax server in C++,
  Python, and TypeScript.
- [Console](console) - A data-visualization and graphical control application for macOS
  and Windows.
- [Documentation Site](docs/site) - The code for the Synnax documentation website.
- [Technical Documentation](docs/tech) - Technical documentation such as RFCs and
  contribution guides.
- [Driver](driver) - An application that can connect to LabJack or National Instruments
  hardware or OPC UA servers and run control sequences on real-time operating systems.
- [Freighter](freighter) - A protocol-agnostic network transport for cross-language
  unary and streaming communication with implementations in several languages (C++, Go,
  Python, and TypeScript) and protocols (GRPC, HTTP, and WebSockets).
- [Pluto](pluto) - A React component library for building modular user interfaces on top
  of the Synnax telemetry engine.
- [Synnax](synnax) - The core Synnax server, which integrates Aspen and Cesium to
  provide a unified telemetry system.
- [X](x) - Common utilities used by other projects.

# Attributions

The Synnax server uses [Pebble](https://github.com/cockroachdb/pebble), and usage must
follow their provided [disclaimer](licenses/BSD-3-Clause.txt).
