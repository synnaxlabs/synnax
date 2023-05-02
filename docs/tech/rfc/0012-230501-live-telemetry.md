# 12 - Live Telemetry

**Feature Name**: Live Telemetry <br />
**Start Date** 2023-05-01 <br />
**Authors**: Emiliano Bonilla <br />
**Status**: Draft <br />

# 0 - Summary

Live telemetry transportation lies at the foundation for any industrial control system.
Until now, Synnax's focus has been solely on distributed telemetry storage and
retrieval. While it has been used for plotting in real-time scenarios at rates of up to
1Hz, dedicated real-time infrastructure is necessary for supporting active control and
higher data rates.

In this RFC I propose an architecture for integrating live telemetry into the existing
Synnax ecosystem. I'll discuss modifications to several existing components, including
the core read and write pipeline, as well as establishing several new components to
support real-time needs.

# 1 - Vocabulary

# 2 - Motivation

# 3 - Philosophy

# 3.0 - Leveraging Properties of Telemetry

Building an efficient telemetry engine starts with having a clear understanding of the
core characteristics of telemetry as it relates to hardware systems. We can leverage
these properties to define constraints that allow us to build simpler, more efficient
software while still providing our users with sufficient flexibility to meet their
needs.

This approach is not novel, and, most notably, has played a major role in the simplicity
of Cesium's storage architecture. As we extend beyond storage, it's important that we
continue to leverage these fundamentals to our advantage. The telemetry overview
document describes these properties.
