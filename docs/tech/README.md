# Synnax - Technical Documentation

# 0 - Summary

Synnax implements a platform for distributed data acquisition, storage and analysis.
It's a complex project with many moving parts. The document guides developers through
the resources available to them, to provide a rational approach for working with the
Synnax codebase.

# 1 - Purpose

The purpose of this document, and the entire `docs/tech` directory, is to:

1. Characterise the core elements that make up the platform, and how they work together
   to provide a cohesive solution. Note that this information is high-level, and
   **deeper explanations are provided in directories related to specific components.**
2. Provide a guide for **navigating the codebase**, and to explain the rationale behind
   the design decisions made.
3. Establish **clear standards on the quality of code we expect**, and to provide
   references
   to make it easier to adhere to these standards.
4. Put in place the **software engineering foundations** required to work on a project
   with this level of complexity, and to provide a reference for developing these
   foundations.

# 2 - Concepts

The first step to working with the Synnax platform is to understand the high level
components that make up the system. The best way to do this is to read the
[concepts](https://docs.synnaxlabs.com/concepts/overview?) section of the official
documentation.

As a supplement, read through the [telemetry concepts](telemetry.md) document. This
provides a detailed guide on what telemetry is, and how Synnax leverages the properties
of hardware generated telemetry to implement a performant database.

# 3 - Architecture

After you have a solid grasp on Synnax's high level concepts, it's time to read about
the architecture of the platform. The [architecture](architecture.md) document provides
a birds eye view of the fundamental design decisions we've made and how they've evolved
into the current architecture.

# 4 - Engineering Foundations

A lot can be learned while contributing to the codebase, but it's important to have a
solid software engineering foundation. Different areas of the codebase require different
types of expertise, and the [foundations](foundations.md) document provides insight into
what we expect from incoming contributors.
