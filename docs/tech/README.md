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
   references to make it easier to adhere to these standards.
4. Put in place the **software engineering foundations** required to work on a project
   with this level of complexity, and to provide a reference for developing these
   foundations.

# 2 - Concepts

The first step to working with the Synnax platform is to understand the high level
components that make up the system. The best way to do this is to read the
[concepts](https://docs.synnaxlabs.com/reference/concepts/overview) section of the
official documentation.

As a supplement, read through the [telemetry concepts](telemetry.md) document. This
provides a detailed guide on what telemetry is, and how Synnax leverages the properties
of hardware generated telemetry to implement a performant database.

# 3 - Architecture

After you have a solid grasp on Synnax's high level concepts, it's time to read about
the architecture of the platform. The [architecture](architecture.md) document provides
a birds eye view of the fundamental design decisions we've made and how they've evolved
into the current architecture.

# 4 - Setting up your Development Environment

It's time to set up your development environment! Here are guides for
[macos](setup-macos.md) and for [windows](setup-windows.md).

# 5 - Engineering Foundations

While there's a lot to be learned by actively contributing to the project, it's
important to have solid software engineering foundations in place. The
[foundations](foundations.md) document walks you through all the essential (and
advanced) technologies and concepts you need to be familiar with when working on a
specific area of the codebase.

# 6 - Language Specific Guides

We also have language specific guides for developing in [python](./python/python.md) and
[typescript](./typescript/typescript.md). These includes information on the correct
processes for working on the codebase in that language.
