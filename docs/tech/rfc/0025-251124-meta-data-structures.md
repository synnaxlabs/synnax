# 25 - Meta Data Structures

**Feature Name**: Improvements to Meta-Data Structure Handling

# 0 - Summary

In this RFC I propose a set of standard practices, patterns, and tools for working with
meta-data structures in Synnax. Meta-data structures are any permanently stored
resources with a Synnax deployment that are not telemetry. This generally includes
any data structure stored in Pebble/Aspen as opposed to Cesium: think Users, Access
Control Policies, Schematics, Logs, Tables, etc

# 1 - Motivation

Although generally considered somewhat secondary in importance and complexity when
compared to Synnax's core telemetry engine, the significant increase in functionality,
logic, and code volume related to meta-data structures in the Synnax codebase. Although
some patterns (gorp, services, clients, flux queries, slices) have been introduced to
standardize the toolchains for working with these data structures, the high
variability in their presentation and core logic has resulted in a number of poor
practices that are now significantly hindering the maintainability and stability
of the Synnax platform.

## 1.0 - Toolchain for a Meta-Data Structure

A "toolchain" for a data structure represents the set of services
(in the core, client libraries, driver, pluto, and console) used to support the feature
set that data structure aims to implement.

For examples, the toolchain for a range includes:

1. The service in the core used to create, retrieve, and validate ranges (core/service).
2. Mechanisms for serializing, storing, and indexing ranges on disk (gorp, aspen, pebble).
3. Signal propagation mechanisms for real-time synchronization and collaborative editing (distribution/signals).
4. API endpoints and access control mechanisms for those endpoints (core/api).
5. Transport mechanisms for encoding and transmitting data structures over the network (freighter).
6. Client libraries for adding language-native wrappers around API calls,
   and exposing standardized data structures (client/ts, client/py, client/cpp)
7. Reactive queries for updating UI's in real-time (flux)
8. General purpose components for exposing functionality (pluto).
9. Specific UI views for user interaction (console).

Toolchains vary across structures, although a large portion of their components clearly
overlap.

## 1.1 - Problems with Current Toolchains

### 1.1.0 - Lack of Efficient Query Mechanisms for Meta-Data Structures

The initial implementation of gorp was designed around simplicity and minimum
viability: *create a wrapper around a key-value store to provided access and rudimentary
query mechanisms for go data structures*.

Gorp's implementation has evolved very little over the course of several years in
production, while the complexity and performance requirements of the services that
use it have evolved significantly.

Gorp uses inefficient `msgpack` or `json` encoding/decoding mechanisms that leverage
heavy reflection, chewing through heap allocations and cpu instruction time. We have
access to a performant, optimized cache for meta-data inside of Pebble, but we can't
leverage it because of the overhead of serialization and/or deserialization.

Gorp supports fast lookups of data structures by key, but that's about it. Nowadays,
we're doing large numbers of lookups by filtering through fields (name, data_type, etc.).
For any of these non-key based lookups, gorp iterates through *every single* item for
a particular class of data structure.

For example, if you want to look up a range by its name, gorp will iterate through
*every* range stored in the database.

### 1.1.1 - Large Amounts of Boilerplate

A significant proportion of the Synnax codebase is replicated boilerplate allocated
to supporting services. Although patterns are generally clear and well established,
maintaining this boilerplate has a non-negligible overhead.

Almost every data structure has a similar shape to its core service, API, and client
library implementations (x3). It's hard to believe there's not some way to consolidate
and simplify the process of maintaining these 'boilerplatey' services without resulting
in excessive coupling.

### 1.1.2 - Lack of Server Side Undo/Redo Support, and Limited Versioning Support for Data Structures

One of the hallmarks of a good user experience is providing leniency when the user makes
a mistake. There comes a huge amount of pain from accidentally deleting something
that is unrecoverable, and a fair amount of joy that comes when the `Ctrl+Z` button
works as expected.

Our leniency for user mistakes is remarkably poor. Very few of our data structures have
undo/redo support, and the support we do have is not very good. Messing something
up in Synnax usually has a high consequence.

### 1.1.3 - Client Side Migrations for Server Side Data Structures

### 1.1.4 - Inefficient Network Transport for Synchronization of Complex Data Structures

Whenever the user pans a schematic within a workspace, the *entire* schematic,
including nodes, edges, and symbol props gets communicated to the core. Needless to
say, this is a huge waste of resource bandwidth.

It also makes supporting real-time collaboration nearly impossible.

### 1.1.5 - Limited Support for Collaborative Editing

### 1.1.6 - Data Structure Complexity Due to Negligence and Technical Debt

### 1.1.7 - Over Reliance on Runtime Validation + General Lack of Type Safety

### 1.1.8 - Multiple Sources of Truth in Client-Side State
