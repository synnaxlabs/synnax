<br />
<p align="center">
    <a href="https://synnaxlabs.com/">
        <img src="x/media/static/logo/icon-white-padded.png" width="20%"/>
    </a>
    <br />
    <br />
    <a href="https://docs.synnaxlabs.com">
        <img src="https://img.shields.io/badge/_-documentation-3b84e5?style=for-the-badge&link=https://docs.synnaxlabs.com&logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAA4AAAAOCAQAAAC1QeVaAAAABGdBTUEAALGPC/xhBQAAACBjSFJNAAB6JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAAAAmJLR0QA/4ePzL8AAAAJcEhZcwAALiMAAC4jAXilP3YAAAAHdElNRQfmCg0KKDIsDui/AAAA2UlEQVQY023QLUtDARjF8ecOp4jrirCiIJgMCybLikUsBoPBaFkWBKNpyWhZMi74An4DMSkoLIgYLKKiMMbmS9rPcG+4u3rq/5zD85yITEJo6NsRoiCh6hbXZv1BYR8M7RaywqInNxruPFowgkoOfVkXNnxrKslVLnvXNilMOfWmllULE451rUiNdT0tZVluVd+RMam1rKWnnsKKC8+W0iIh1Lw6UQlh048Difx5TZ+2w7RLD+aMPjbvSifsGTozrjhJ1VbiJWaiG+34iCTHO3Eeg3DvPw2siV92RsVo+XSmkwAAACV0RVh0ZGF0ZTpjcmVhdGUAMjAyMi0xMC0xM1QxMDo0MDo1MCswMDowMIzzdxUAAAAldEVYdGRhdGU6bW9kaWZ5ADIwMjItMTAtMTNUMTA6NDA6NTArMDA6MDD9rs+pAAAAAElFTkSuQmCC" />
    </a>
</p>

# Synnax

The software infrastructure for data driven hardware teams.

- [Synnax](#synnax)
- [What is Synnax?](#what-is-synnax)
- [Development Status](#development-status)
- [How to Contribute](#how-to-contribute)
- [Repository Organization](#repository-organization)

# What is Synnax?

Synnax is for real-time hardware operations teams dissatisfied with the long,
inefficient cycles between acquiring data and using it to make actionable decisions.
Unlike traditional systems that disregard data handling beyond writing to a file, Synnax
considers the entire data lifecycle; its modular, open architecture delivers locality
aware distributed data storage and transport, extensible interfaces for integrating
analysis tools, and a performant pipeline for manual and programmatic control at any
scale.

# Development Status

Synnax is currently in beta and is under active development. The APIs are stable
and are unlikely to change significantly.

Versions prior to 1.x.x follow modified Semantic Versioning. Versions with the same
patch (e.g. 0.0.1 and 0.0.2) are guaranteed to maintain the same API, while minor
versions may include API changes.

Our team is targeting a v1 release before the end of 2024, at which point all APIs
will be stable and follow strict semantic versioning.

# How to Contribute

Help us modernize industrial control! Reach out
to [Emiliano](mailto:ebonilla@synnaxlabs.com)(ebonilla@synnaxlabs.com) if you'd like to
get involved. While you wait for a response, check out
the [New Contributor Guide](docs/CONTRIBUTING.md) to get up to speed.

# Repository Organization

Synnax is built as a collection of several projects, all of which are collected
in this monorepo. The following is an alphabetically sorted summary of each:

- [Alamos](alamos) - Dependency injected code instrumentation that provides
  observability into the Synnax platform.
- [Aspen](aspen) - A gossip based distributed key-value store used for propagating and
  persisting metadata between nodes, such cluster topology, state, and configuration.
- [Cesium](cesium) - An embedded time-series engine optimized for high performance reads
  and writes of time-series sensor data.
- [Client](client) - Client libraries for synnax available in multiple languages.
- [Console](console) - Data visualization and cluster management user interface for Windows,
  macOS, and Linux.
- [Documentation Site](docs/site) - The user-facing documentation for Synnax. Contains
  the code for the Synnax documentation website, technical RFCs, and additional media
  such as logos.
- [Freighter](freighter) - A protocol agnostic network transport for cross-language
  unary and streaming communication with implementations in several languages.
- [Pluto](pluto) - A component library for building modular user interfaces on top of
  the Synnax telemetry engine.
- [Synnax](synnax) - The core Synnax server, which integrates all other services to
  provide a complete telemetry system.
- [ X](x) - Common utilities used by other projects. The most relevant are:
  - [Telem](x/go/telem) - Core telemetry primitives used across the Synnax stack
    (timestamps, data types, arrays, etc.)
  - [Confluence](x/go/confluence) -
    Assemble and run concurrent data processing and message passing pipelines.
  - [Gorp](x/go/gorp) - Efficient querying of go-types to and from a key-value store.
  - [Signal](x/go/signal) - A library for controlling goroutine lifecycle.
