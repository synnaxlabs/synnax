## Context

One of the key remaining pieces of friction in Synnax is the lack of a clear strategy
for organizing system configurations.

We currently have the following data structures in Synnax:

1. Channels: The core of Synnax, used to stream and store samples from a logical telemetry source.
2. Workspaces: A collection of visualizations and a layout in the console.
3. Racks: A remote Synnax data acquisition process that manages a set of Devices and Tasks.
4. Tasks: A set of instructions for acquiring data, outputting commands, and running sequences (such as an arc task).
5. Devices: Physical hardware devices that can be connected to Synnax.
6. Users: Synnax users with roles and policies.
7. Roles: User roles with permissions.
8. Policies: User policies that define access control.
9. Arcs: Compiled automation sequences.
10. Line Plots: Visualizations of channel data over time.
11. Tables: Tabular representations of channel data.
12. Schematics: Diagrams of hardware and software systems.
13. Schematic Symbols: Customer, user uploaded Icons and shapes used in schematics.
14. Logs: Visualization Records of events and errors.
15. Ranges: Time ranges that organize historical data.
16. Clusters: A cluster of Synnax cores acting as a single logical data space.
17. Nodes: A single Synnax core deployed as part of a cluster.
18. Views: A predefined query for a set of data structures (currently used for ranges and statuses)
19. Statuses: Data structures representing general purpose system statuses.
20. Labels: Used to categorize data structures.
21. Groups: Collections of data structures.

