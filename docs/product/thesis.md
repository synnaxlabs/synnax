# Synnax: Product Thesis

# 0 - Positioning Statement

Synnax is for real-time hardware operations teams dissatisfied with the long, inefficient cycles between acquiring data and using it to make actionable decisions.Unlike traditional systems that disregard data handling beyond writing to a file, Synnax considers the entire data lifecycle; its modular, open architecture delivers locality aware distributed data storage and transport, extensible interfaces for integrating analysis tools, and a performant pipeline for manual and programmatic control at any scale.

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
decision. The faster they can acquire, analyze, and act on this data, the faster they
can improve.

Existing data acquisition and control (DAQ) systems pay little attention to the
importance of this cycle. **They focus on recording data and controlling hardware, but
disregard the analysis process**. Leaving out this critical step is akin to writing
a turkey recipe without instructions for cooking it: it's well-dressed, but nobody
can eat it.

Larger, well funded teams bridge this gap with a patchwork of tools, while smaller ones
end up under-analyzing their data. Both of these approaches lead to slower, error-prone
development cycles.

## 2.0 - Local File Systems

Existing systems export recorded data to a local file system. These files use
proprietary formats (like `.tdms`) and are **only accessible by the small number of
users** with access to the machine. To share data, engineers must manually convert it
to a more accessible format and upload it to a network drive. The files on these drives
can have obscure names, making it difficult for an engineer to locate specific data
sets.

Analysts typically query a small subset of the data stored in a file. When accessing a
file with 1000 channels, they may only examine 3, yet still need to download the
entire file to their local machine and load it into their analysis tool of choice.

**Accessing data at scale is extremely challenging**. To understand the evolution of a
system over time, an engineer must manually search for dozens or even hundreds of files,
post-process each one, and aggregate the results.

The process of taking recorded data from a file and extracting meaningful insights is
slow and error-prone. In consequence, engineers often under-analyze the systems they
operate, leading to less effective, slower, and more expensive development cycles.

## 2.1 - Closed, Proprietary Software

Current DAQ platforms are iterations on software developed decades ago. These systems
are **proprietary and restrict users to their internal ecosystem**. Modern, cloud-native
data processing and aggregation tools such as Apache Kafka and Spark have shifted the
landscape of data engineering and analysis, and big data and machine learning are
becoming one of the most important assets for the modern enterprise. Existing DAQ
platforms provide limited or no support for integration with these tools.

LabVIEW, the clear frontrunner in data acquisition, was initially released in
the 1980s and implements a domain specific, graphical programming language. Finding
engineers with experience is difficult, and training new hires involves paying
the vendor for expensive courses.

## 2.2 - Archaic User Experiences

Enterprise software is notorious for its poor user experience. Data acquisition software
is no stranger to this issue; **existing user experiences are difficult to navigate and
require extensive training to use effectively**.

Programmatic interfaces are outdated and pay little attention to the developer
experience. LabVIEW requires users to learn an entire programming language to get started.
Training is expensive, and the pool of certified developers is limited.

User interfaces are no better. They focus on implementing a feature set, not delivering
a cohesive solution to the problem at hand. Large, nested context menus and antiquated
designs are commonplace, and make it difficult to find the functionality you need.
Research into user experience design has made great strides in recent years, and these
improvements have yet to find themselves in the world of industrial control and data
acquisition.

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
