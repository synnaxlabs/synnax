# New Contributor Guide

Thanks for your interest in Synnax! We'd love to have you contribute!

> But in the machine of today we forget that motors are whirring: the motor, finally,
> has come to fulfill its function, which is to whirr as a heart beats - and we give no
> thought to the beating of our heart. Thus, precisely because it is perfect the machine
> dissembles its own existence instead of forcing itself upon our notice.
>
> ― Antoine de Saint-Exupéry, Wind, Sand and Stars

## 1 - Find Something to Work On

The next best step is to start thinking about the area or feature you'd like to
contribute to. There are a few ways to do this:

1. Browse our [open issues list](https://github.com/synnaxlabs/synnax/issues) for
   something that catches your eye. Issues marked
   ["good first issue"](https://github.com/synnaxlabs/synnax/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)
   are a great place to start.
2. Read our [product strategy documents](product/psd), our
   [architecture overview](./tech/architecture.md) document, and our technical
   [requests for comments (RFCs)](tech/rfc). These documents outline critical product
   strategies and technical design decisions we've made, and provide insight into the
   different product focuses we're delivering.
3. Explore the codebase! Read the
   [repository organization](../README.md#repository-organization) section of our
   [README](../README.md) and start digging through the code we've written; there are
   many interesting algorithms:
   - [Distributed counters](../aspen/internal/cluster/pledge/pledge.go)
   - [Gossip algorithms](../aspen/internal/kv/gossip.go)
   - [GPU rendering](../pluto/src/core/vis)
   - [Transport protocols](../freighter)
   - And more!

## 2 - Technical

If you haven't already, please star the project on GitHub! This helps us grow our
community and attract more contributors. Now that you've found an area you're interested
in working on, it's time to get into the technical foundations of the project. All of
our technical documentation can be found in the [docs/tech](../docs/tech) directory
where the [README](./tech/README.md) is the best place to get started.

## 3 - Developer Expectations

To ensure software reliability and good practices, all contributors must adhere to the
following principles.

1. All PRs must
   - be approved by at least 2 people.
   - be approved by [Elham Islam](https://github.com/Lham42).
   - adhere to and every checklist in the existing PR template.
   - not exceed 2000 lines of changes.

2. Any changes to the codebase must be done through a PR. No direct pushes to `main` or
   `rc` branches.

## 4 - Issue priority

| Priority Level | Feature                                                                                | Bug                                                                                                                                                                                                 |
| -------------- | -------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Urgent         | This feature blocks progress on upcoming pilots.                                       | This bug will stop the user from making progress and will cause significant loss of time and/or resources. Examples are critical bugs which cause complete failure of core elements of the product. |
| High           | This feature has been requested by users and is important to their use of our product. | This bug significantly degrades the quality of the user's experience and would take priority over most new feature development. Bugs could cause slowing of the user's workflow or repeated work.   |
| Medium         | There has been some interest in this feature or would improve the product              | This bug makes a noticeable impact on the user experience and should be targeted for next release or there is an existing workaround for now.                                                       |
| Low            | This feature is not critical to users or the product.                                  | These bugs have low impact on the user and could be put off to future releases as necessary.                                                                                                        |
