# Synnax: Product Thesis

# 0 - Positioning Statement

Synnax is for real-time hardware operations teams dissatisfied with the long, inefficient cycles between acquiring data and using it to make actionable decisions. Unlike traditional systems that disregard data handling beyond writing to a file, Synnax considers the entire data lifecycle; its modular, open architecture delivers locality aware distributed data storage and transport, extensible interfaces for integrating analysis tools, and a performant pipeline for manual and programmatic control at any scale.

# 1 - Mantras

_Synnax enables teams to operate and understand their hardware systems._

_Synnax is the operating system for large scale hardware._

_Synnax is the framework for data driven hardware teams._

_Synnax is the software infrastructure for large scale hardware._

# 2 - Problem Definition

## 2.0 - Long Cycle Times

## 2.1 - Local File Systems

## 2.2 - Closed, Proprietary Software

## 2.3 - Antiquated Programming Paradigms

## 2.4 - Archaic User Interfaces

# 3 - Product Definition

At the highest level, Synnax’s only role is to make the readings of an instrument accessible to any entity that requires them. These instruments can be data acquisition computers recording sensor telemetry, human operators executing commands, or analysis pipelines publishing post-processed results. Subscribers can consume the readings of another instrument to control actuators, render plots, or calculate values in real-time. To provide this functionality, Synnax implements four key components.

## 4.1 - Distributed Telemetry Engine

At the core of the platform lies a distributed telemetry engine that stores telemetry and delivers it to wherever it’s required. This core is packaged as a single binary, called a node, that can interact with other nodes, called peers, to coordinate data storage and transfer. A group of synchronized nodes forms a cluster.

Unlike traditional distributed databases, such as Apache Cassandra, Synnax tailors to hardware by implementing locality aware data storage. Nodes are deployed in close physical proximity to the instruments they send and receive data to, increasing write throughput and reducing network congestion. Synnax is eventually consistent and highly fault tolerant. A single node can be completely separated from its peers and still accept both read and write requests; this makes it ideal for global deployments under unreliable network conditions, such as fleets of self-driving cars.

Each node also includes a relay that transfers telemetry to other nodes and connected clients. This relay forms the basis for real-time control and stream data processing. Multiple entities, called writers, can send telemetry to a node, where the relay then broadcasts it to multiple readers. These readers then process the information, using it for analysis

## 4.2 - Analysis Tooling

## 4.3 - Device Interfaces

## 4.4 - Framework for Real-Time Operation

4 - Hypotheses

5 - Ideal Customer Profile
