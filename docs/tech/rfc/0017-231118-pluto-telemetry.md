# 17 - Pluto Telemetry Refactor

**Feature Name**: Pluto Telemetry Refactor <br />
**Start Date**: 2023-11-18 <br />
**Authors**: Emiliano Bonilla <br />
**Status**: Draft <br />

# 0 - Summary

Introducing multi-threaded telemetry processing and visualization in
[RFC 13](./0013-230526-pluto-visualization.md) has played a critical role in making
Synnax capable of handling real-time hardware operations at scale. Now that we're using
this functionality in production, we're finding areas where we'd like to step up the
capability of how we can represent and display telemetry within Synnax.

The current implementation of telemetry processing hinders our ability to flexibly
define new interfaces and visualizations. This RFC examines why this is the case,
and proposes a new design that will allow our users to flexibly define how they
want to view and work with their data in Synnax.

# 1 - Vocabulary

**Pluto** - The Synnax React component library. Source code is [here](../../../pluto/).
**Telemetry** - Data samples recorded from sensors and/or sent to actuators; typically
stored and transferred by the Synnax Server. More details available [here](../../../docs//tech/telemetry.md).

# 2 - Motivation

Motivation for this RFC comes from [PSD 3 - Telemetry Assembler](../../../docs/product/psd/003-231118-telemetry-builder.md).

# 3 - Philosophy

# 5 - Detailed Design

