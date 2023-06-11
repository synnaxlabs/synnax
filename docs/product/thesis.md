# Synnax: Product Thesis

# 0 - Positioning Statement

Synnax is for real-time hardware operations teams dissatisfied with the long,
inefficient cycles between acquiring data and using it to make actionable decisions.
Unlike traditional systems that disregard data handling beyond writing to a file, Synnax
considers the entire data lifecycle; its modular, open architecture delivers locality
aware distributed data storage and transport, extensible interfaces for integrating
analysis tools, and a performant pipeline for manual and programmatic control at any
scale.

# 1 - Mantras

_Synnax enables teams to operate and understand their hardware systems._

_Synnax is the operating system for large scale hardware._

_Synnax is the framework for data driven hardware teams._

_Synnax is the software infrastructure for large scale hardware._

# 2 - The Problem

From factories to rockets, **productive iteration is the means of improving hardware
systems.** Teams continuously design, build, and evaluate new versions of their products
or operations. Increasing the pace and quality of each iteration means organizations can
deliver better solutions faster.

**Data lies at the core of this process**. From initial design and testing to long term
operational deployment, the information teams create and collect plays a role in every
decision. The faster teams can acquire, analyze, and act on this data, the faster they
can improve.

Existing data acquisition and control (DAQ) systems pay little attention to the
importance
of this loop. They focus on recording data and controlling hardware, but disregard the
rest of the lifecycle, often dumping data to a local file system in a proprietary
format. This leaves teams with a crevasse between acquisition and meaningful analysis.
They bridge this gap with a patchwork of tools and processes unsuited for the task at
hand, leading to a slow, error-prone development cycle.

## 2.1 - Local File Systems

Existing systems export recorded data to a local file system. These files use
proprietary formats (like `.tdms`) and are only accessible by the small number of users
with access to the machine. To share data, engineers must manually convert it to a more
accessible format and upload it to a network drive. The files on these drives can have
obscure names, making it difficult for an engineer to locate specific data sets.

Engineers typically query a small subset of the data stored in a file. When accessing a
file with 1000 channels, they may only examine 3, yet they still need to download the
entire file to their local machine and load it into their analysis tool of choice.

Accessing data at scale is impossible. To understand the evolution of a system over
time, an engineer must manually search for dozens or even hundreds of files,
post-process each one, and aggregate the results by hand.

The process of taking recorded data from a file and extracting meaningful insights is
slow and error-prone. In consequence, engineers often under-analyze the systems they
operate, leading to less effective, slower, and more expensive development cycles.

## 2.3 - Antiquated Programming Paradigms

Existing platforms are iterations on software developed in the 80s and 90s. LabVIEW,
a visual language first released in 1986, remains the front-runner in data acquisition
programming.

## 2.2 - Closed, Proprietary Software

- Restricted to a small set of programming languages and paradigms.
- Closed source ecosystem with few extensions for modern data aggregation and processing
  systems.

## 2.3 - Antiquated Programming Paradigms

- Advancements in cloud computing and distributed data processing.
- Lack of hire-able engineers.
- Open source software means you can contribute to the system you use.

## 2.4 - Archaic User Interfaces

- Modern UIs are intuitive and gorgeous, existing industrial control interfaces are not.
- We think there's a long way to go towards improving the user experience.

# 3 - The Opportunity

# 3 - The Solution

At the highest level, Synnax’s only role is to make the readings of an instrument
accessible to any entity that requires them. These instruments can be data acquisition
computers recording sensor telemetry, human operators executing commands, or analysis
pipelines publishing post-processed results. Subscribers can consume the readings of
another instrument to control actuators, render plots, or calculate values in real-time.
To provide this functionality, Synnax implements four key components.

## 4.1 - Distributed Telemetry Engine

At the core of the platform lies a distributed telemetry engine that stores telemetry
and delivers it to wherever it’s required. This core is packaged as a single binary,
called a node, that can interact with other nodes, called peers, to coordinate data
storage and transfer. A group of synchronized nodes forms a cluster.

Unlike traditional distributed databases, such as Apache Cassandra, Synnax tailors to
hardware by implementing locality aware data storage. Nodes are deployed in close
physical proximity to the instruments they send and receive data to, increasing write
throughput and reducing network congestion. Synnax is eventually consistent and highly
fault-tolerant. A single node can be completely separated from its peers and still
accept both read and write requests; this makes it ideal for global deployments under
unreliable network conditions, such as fleets of self-driving cars.

Each node also includes a relay that transfers telemetry to other nodes and connected
clients. This relay forms the basis for real-time control and stream data processing.
Multiple entities, called writers, can send telemetry to a node, where the relay then
broadcasts it to multiple readers. These readers then process the information, using it
for analysis

## 4.2 - Analysis Tooling

## 4.3 - Device Interfaces

## 4.4 - Framework for Real-Time Operation

4 - Hypotheses

5 - Ideal Customer Profile
