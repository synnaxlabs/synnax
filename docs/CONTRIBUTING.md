# New Contributor Guide

Thanks for your interest in Synnax! We'd love to have you contribute!

> But in the machine of today we forget that motors are whirring: the motor, finally,
> has come to fulfill its function, which is to whirr as a heart beats - and we give no
> thought to the beating of our heart. Thus, precisely because it is perfect the machine
> dissembles its own existence instead of forcing itself upon our notice.
>
> ― Antoine de Saint-Exupéry, Wind, Sand and Stars

# 0 - Read About the Product

Making meaningful contributions to Synnax starts with having a clear understanding of
the product we're building. The best place to start is by reading our
[thesis](product/thesis.md). This document defines the problem with the status quo and
the approach we're taking to solve it.

# 1 - Find Something to Work On

The next best step is to start thinking about the area or feature you'd like to
contribute to. There are a few ways to do this:

1. Browse our [open issues list](https://github.com/synnaxlabs/synnax/issues) for
   something that catches your eye. Issues
   marked ["good first issue"](https://github.com/synnaxlabs/synnax/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)
   are a great place to start.
2. Read our [product strategy documents](product/psd), our [architecture overview](./tech/architecture.md) 
   document, and our technical [requests for comments (RFCs)](tech/rfc). These documents 
   outline critical product strategies and technical design decisions we've made, and 
   provide insight into the different product focuses we're delivering.
3. Explore the codebase! Read
   the [repository organization](../README.md#repository-organization)
   section of our [README](../README.md) and start digging through the code we've
   written; there are many interesting algorithms:

    - [Distributed counters](../aspen/internal/cluster/pledge/pledge.go)
    - [Gossip algorithms](../aspen/internal/kv/gossip.go)
    - [GPU rendering](../pluto/src/core/vis)
    - [Transport protocols](../freighter)
    - And more!

# 2 - Technical

If you haven't already, please star the project on GitHub! This helps us grow our
community and attract more contributors. Now that you've found an area you're intersted
in working on, it's time to get into the technical foundations of the project. All of
our technical documentation can be found in the [docs/tech](../docs/tech) directory
where the [README](./tech/README.md) is the best place to get started.

