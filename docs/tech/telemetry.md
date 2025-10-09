# Telemetry

# 0 - Summary

The term "telemetry" is thrown around throughout Synnax's product and software design,
and lies at the core of Synnax's vision for data acquisition and analysis. The purpose
of this page is to provide a clear definition of what telemetry is in the context of the
Synnax ecosystem. All concepts discussed here are fundamental to understanding how
Synnax works and delivers value.

# 1 - General Definition

While the term "telemetry" is used in countless different contexts throughout hardware
and software, the general definition actually provides a good starting point for
understanding Synnax's use of the term:

> The process of recording and transmitting the readings of an instrument

At the highest level of abstraction, this is all Synnax does: move the
readings/emissions of an instrument from one place to another. In many ways, all of
Synnax's code lies around implementing or supporting this process. Whenever you see a
particular feature set, ask how it relates to this concept.

# 2 - Telemetry in Synnax

As humanity continues to develop advanced technologies and deeply connected systems, the
technical scope telemetry encompasses continues to widen, and the definition above
becomes less meaningful. The following are a few of the common contexts in which
telemetry is used:

- In **agriculture**, telemetry is characterized by low-frequency readings across vast
  areas to study the effects of various factors on crop growth.

- In **software**, telemetry takes a variety of forms from reading hardware metrics such
  as CPU usage to tracking user behavior and interactions. For example, Synnax uses a
  tool called [opentelemetry](https://opentelemetry.io/) to collect and transmit logs,
  metrics, and traces so that we can better understand the state of the software we're
  developing.

- In **aerospace**, telemetry typically represents high-rate data collection from
  sensors and actuators all over a vehicle. This data is used to monitor the health of
  the vehicle and control it in real-time. Perhaps one of the best documented examples
  is the [Space Shuttle's RS-25 engine controller](https://en.wikipedia.org/wiki/RS-25),
  which was responsible for controlling the Space Shuttle's main engines during flight.

Telemetry in Synnax most closely relates to the third definition. While it can be used
in the first two scenarios, Synnax resembles a test and measurement system that is
designed to acquire telemetry from sensors and send commands to actuators in real-time.

# 3 - Fundamental Properties

## 3.0 - Samples

A sample is the most basic unit of telemetry, and is simply a time-associated value.
Time-association is done with a timestamp, or number that represents a particular
duration of time since a fixed epoch. Values, on the other hand, can be almost anything
that can be represented as binary, whether it be a single number, JSON file, or a video
frame.

## 3.1 - Read and Write Patterns

The read and write patterns of time-series data are perhaps some of the most predictable
and constrained in the database world. These patterns allow us to make various
optimizations to increase database performance.

### 3.1.0 - Writes

**Writes are typically append only**, meaning that older data is rarely inserted before
newer data. High volume writes occur when ingesting pre-existing data or recording data
from high rate sensors; in both cases, data can be batched to reduce the number of
network calls, memory allocations, and storage operations. It's also important to note
that point writes from a single source are less common than a long-lived stream of data
arriving over an extended period of time (most commonly a sensor).

### 3.1.1 - Reads

Read patterns typically come in one of two flavors. The most common, and simplest is a
time-range based lookup. These reads often contain a large number of consecutive
samples. Range based reads are far more common than point lookups; so much so that the
Synnax storage engine, Cesium, intentionally sacrifices point lookup performance for
range lookup performance.

Value filters are the second read flavor. This pattern returns samples that match a set
of criteria, such as a threshold. While more expensive to execute, this pattern is also
less common than range based lookups. Despite their relative rarity, these reads play an
especially important role in queries that look for specific events over long periods of
time.

### 3.1.2 - Frequency and Volume

The number of concurrent readers is typically much higher than the number of concurrent
writers. Many users and applications are typically reading data from the same source,
while only one source is emitting that data.

# 4 - Telemetry Variants

The following are the essential categories of telemetry that Synnax is designed to
handle. They're organized from highest to lowest level of predictability. The earlier
categories are easy to optimize for, while the latter categories are more challenging.
Some of these categories share characteristics; in those cases, I'll note the
similarities but won't repeat the details.

## 4.1 - Sensor Data

Sensor data is the most common variant of telemetry processed by Synnax, and represents
the bulk volume of data moved throughout a cluster: probably around 90% in a naive
estimate. Sensor data generated by hardware typically has a few important
characteristics that we leverage throughout Synnax's design.

### 4.1.0 - Single Source

One of the hardest problems in database design is concurrent transaction handling, with
increasingly complex solutions being developed to handle the problem of writing to the
same value at the same time. When collecting sensor data, this issue becomes far less
relevant. It doesn't make physical sense to have two different entities writing values
for the same sensor at the same time. We leverage this property to simplify the way we
handle transaction control within Synnax's storage engine.

### 4.1.1 - Tightly Constrained Types

Unlike software events, sensor data is typically constrained to a small subset of highly
predictable data types: numbers. You'd be hard-pressed to find a physical sensor that
doesn't produce a number or collection of numbers. Numeric data types have a fixed
length and can be encoded/decoded very efficiently. Fixed data types mean we can predict
memory allocations and perform more efficient storage lookups.

### 4.1.1 - Regularity

Unlike software telemetry such as user events, sensor data is typically sampled at a
fixed or almost-fixed rate. The consistency in time-spacing between samples means we can
make assumptions about the volumes of data we need to sample, as well as make relatively
precise time-based lookups without expensive operations such as binary searches.

### 4.1.1 - Batching

A hardware device typically has a fixed number of sensors. A particular data acquisition
device can have many thousands of sensors. Data from these sensors is collected in
batches that are all sampled at, for all necessary purposes, identical times. This means
that we can leverage batching to reduce the number of network calls, computations, and
storage operations we need to perform. A batch in Synnax is referred to as a `frame`,
which contains a collection of time-aligned samples for a number of channels. This also
means that we can associate many samples with a single timestamp, reducing data storage
and indexing costs, and do fast correlations between samples from different sensors.

### 4.1.2 - Noise

One of the challenges with working with sensor data is that there is large variability
in the amount of noise present in the data. For boolean and state related channels,
noise is extremely low, and they often remain in the same state for long periods of
time. On the other hand, sensors such as accelerometers or thermocouples are extremely
noisy, and, aside from long-term trends, are almost completely random. This poses unique
challenges for data compression and storage, as well as analysis.

## 3.2 - Real-Time Commands

Real-time commands are used to control the precise positions of actuators at discrete
points in time. This telemetry variant is similar to sensor data in that it has
predictable, fixed length data types, but differs in that emission rates can have a high
degree of variance. This is especially true when a human operator is involved, as they
can issue commands at arbitrary times.

Perhaps the most demanding example of real-time command would be something along the
lines of over the network Diagram control. Sensor data must be processed and analyzed in
real-time, often at rates exceeding several hundred hertz.

### 3.2.0 - Multi-Source

Commands can be issued by multiple sources concurrently. For example, an operator can
issue a manual command while an auto-sequence (programmatic control) is running. In
order to deal with multi-source writes, we need some way of clearly defining who has the
authority to issue commands, and how we handle ordering from multiple sources for both
reads and writes.

### 3.2.1 - Highly-Variable Emission Rates

Real-time commands can be issued at arbitrary intervals. In the Diagram control example,
this isn't the case, and the commands are probably issued at predictable rates. However,
in the case of a human operator, commands can be issued at any time, and at any rate.
For writes, this variability means it's more difficult to pre-allocate buffers and size
caches, and, for reads, lookups become far more expensive.

### 3.2.3 - Small Batch Sizes

In most cases, commands for different channels are issued independently or in relatively
small batches of less than 10 actuators. As a result, timestamp cardinality across
multiple command channels is very high, and we need to store more timestamps per channel
than with sensor data.

### 3.2.4 - Low Latency and Jitter

When doing over the network (OTN) real-time control, the two most important metrics are
latency and jitter. Latency measures the time it takes for an issued command to reach
its target actuator, while jitter measures the variation in this latency over time. Low
latency is critical for reducing the effects of phase lag ('tail wagging the dog') and
jitter is fundamental in timing critical scenarios where two commands must be executed
in a particular order, separated by a tightly constrained amount of time (think draining
a precise volume of propellant from a tank).

Real-time commands require both low latency and jitter. Reducing latency and writing
anti-jitter algorithms is a complex task, and consumes a large amount of resources.

### 3.2.5 - Execution Acknowledgement

One of the unique challenges in command telemetry is execution acknowledgement. It's not
sufficient to simply send a command to an actuator and assume it was received and
executed, some sort of acknowledgement must be sent back to the source of the command.
This execute-acknowledge cycle adds complexity and places additional importance on low
latency.

## 3.3 - Supervisory Commands

Supervisory commands are issued to set the state or parameters of another device; this
device then uses these parameters to perform real-time control much closer to the
hardware. Diagram control is also a good example of this. In a supervisory control
system, a command would be sent over the network containing the P,I, and D parameters
for operation. Another command is issued to start an embedded Diagram controller, which
then uses the parameters to perform real-time control independent of Synnax. Another
similar example involve pushing a configuration file to a flight computer or a
calibration sheet to a data acquisition device. Supervisory control is as important as
its real-time counterpart, yet has very different requirements.

### 3.3.0 - Low, Highly Variables Frequencies

Supervisory commands are issued at a much lower, far more variable frequencies than
real-time commands, and these channels have much lower sample counts than other
telemetry variants. Although we can't make the transportation pipeline as efficient for
these channels, we don't really need to.

### 3.3.1 - Large, Variably Sized Data Types

Supervisory commands are often complex, containing many different parameters with
variable size data types such as strings. These commands can also be very large; it
wouldn't be surprising to see a supervisory command that is hundreds of kilobytes in
size, several orders of magnitude larger than a typical sensor value. Commands can also
vary in size for the same channel, and we can't make assumptions about the size of a
command based on the channel it's sent to.

### 3.3.2 - Latency and Jitter Tolerance

Latency and jitter are essentially irrelevant for supervisory commands. One of the main
reasons for supervisory control is to execute commands closer to the hardware to
minimize the risk of network failures. As long as we can guarantee that the command will
be sent and received within a leisurely time frame, we can safely ignore latency and
jitter.

## 3.4 - Software Signals

Software signals are used to track and listen to the state of the Synnax cluster. An
example of such an event would be the creation of a new channel, the commitment of a new
range, or the deletion of a channel. By allowing our users to listen to these events as
pseudo software-sensors, we enable a truly dynamic system for interacting with a Synnax
cluster.

For example, if we have a new channel that emits a value whenever a new range is labeled
as "Tank Pressurization", we can notify automated post-processing systems that perform
analysis on the data without any human interaction. It's easy how to see how this
pattern can be extended to other use cases, making for a very powerful system that can
automate many manual tasks.

It's important to note that not all telemetry generated by a cluster is considered a
software signal. For example, the CPU usage of the node falls under the sensor data
category.

### 3.4.1 - Highly Variable Frequencies

Software events and signals have highly variable frequencies. They can be emitted in
bursts, widely spaced intervals that are weeks or even months apart. In a data
acquisition context, they do peak out at relatively low frequencies; I find it hard to
imagine a useful software signal that is emitted more than ten times per second.

### 3.4.2 - Variable Payload Sizes

As with supervisory commands, software signals can have arbitrary payloads.

### 3.4.3 - Persistence Requirements

One of the simplifying properties of software signals is that they are emergent, and are
typically triggered by some command or sensor value. These emergent events aren't useful
to query in hindsight, and, as a result, most software signals are completely virtual
and don't need to be persisted to disk.

## 3.5 - Summary

The main differences these variants can be broken down into:

- Emission Rates
- Emission Regularity
- Payload Sizes and Variation
- Persistence Requirements
- Latency and Jitter Tolerance

Events vs. Metrics
