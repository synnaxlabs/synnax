What we're working on right now is the development of arc-based calculated channels.
One of the key issues we need to solve for is how to properly align telemetry across
channels that are sampled at separate rates.

### Required Pre-Reading

Here's where you can read about index channel structures: https://docs.synnaxlabs.com/reference/concepts/channels
Also good to know about reads: https://docs.synnaxlabs.com/reference/concepts/reads
Writes: https://docs.synnaxlabs.com/reference/concepts/writes
And streams: https://docs.synnaxlabs.com/reference/concepts/streams

The channel source code in core/pkg/distribution/channel
The cesium source code in cesium/
The distribution framer source code in core/pkg/distribution/framer
The arc runtime code in arc/runtime
The arc IR code in arc/ir
The arc stratifier code in arc/stratifier

### The Problem

Our current calculated channel implementation (uses LUA) does not support creating
calculated channels that span multiple indexes or have a combination of indexed and/or
virtual channels.

The fundamental question is as follows: how do we properly 'align' telemetry samples
across channels, especially those that don't share the same time-base (different indexes),
or don't have a time-base at all (virtual)?

Data arrives into the arc runtime in frames, collections of series mapped to the channel
key for that series. Each series has two useful fields. The most useful, called alignment,
represents the unique position of the first sample in the series within the channels
data. The nice property of the alignment field is that it can be used to align the samples
in two data channels that share the same index. This makes it easy to know how to calculate
vectorized operations across channels that use the same index.

The alignment field does not work when channels do not share the same index. This is crucial.
Let's imagine we have an arg program that does the following.

```arc
[sensor_1_idx_1, sensor_2_idx_2] -> sensor_1_idx_1 + sensor_2_idx_2 -> sensor_sum
```

This would create two telemetry source nodes that then feed into the addition node,
then feed the output into a 'write' node that writes to the output channel.

The crucial thing here is that sensor_1_idx_1 values and sensor_2_idx_2 values are
not guaranteed to arrive at the same time or share the same alignment.

This is an example that matches the exact scenario of our fundamental question.

### Ideation on Solutions

Coming up with a solution fundamentally involves two steps:

### Theory

The first is we need a theoretical understanding about strategies to align data. What
makes semantic sense to the caller? What sort of time-base do we calculate for the index
of the output channel? How do we deal with extreme edge cases? Such as a value arriving
once every few days, a value never arriving, a flood of values arriving, strange alignments,
that sort of things.


### Practical

The second thing we need to do is reason about a practical approach to implementing
our theoretical solution. There are a few core areas we should focus on:

#### How does telemetry actually arrive to the calculation?

There are two ways telemetry will arrive: through a streamer in the case of a real-time
calculation, and in an iterator in the case of a historical calculation.

In the real-time scenario, frames containing the data for all of the channels for a
particular index will arrive at the same time. Keep in mind that this means that
if you have idx, ch_1, ch_2, and ch_3, only part of those data channels may be getting
written to, so idx, ch_1, and ch_2 may receive data but ch_3 may not. The crucial note
here is that you want get a frame like idx, ch_1 with alignment 1 to 100 and then idx, ch_2
with alignment 1 to 100. You will get all the available data for a particular alignment
in one frame.

Next thing is the historical side. Iterators do not work that way. They may send data
values independently. so you could get idx 1 to 100, ch_1 1 to 100, then ch_2 1 to 100.

#### Runtime Implementation

The next thing we need to think about is how we actually feed these series through the
flow nodes inside of the runtime. We need to think about how we track which nodes have
been processed, when they get processed, and how we deal with things like reductions
in output data.

One of the crucial pieces to do research on is the arc stratified runtime, and the `state`
pacakge that currently handles movement of data.


#### Other Scenarios Aside from Calculations

Keep in mind taht this spec is calculation focused, but arc can be used in a variety
of ways that don't invalide direct multi-input -> single-output. Notably, arc can be
used for real-time automation. Keep those design principles in mind when trying to solve this problem.

#### Extraneous Ideas

Look for academic papers on how these problems are solved?
High water marks?
Timestamp generation for virtual channels?
Separation between telem and state?
Garbage collecting old data?
