We're focused on improving the Synnax telemetry infrastructure's performance. For the scope
of this project, we'll focus exclusively inside the core `core` and maybe the `arc` directory.

Most of our energy is centered around the calculation engine, specifically with streaming
calculations. The problem right now is that each calculation gets run in a separate go-routine.
This is great because it means we can distribute CPU load across all cores. At the same
time it's not great for a few reasons:

1. With 1 calc per gr, large numbers of calculations will result in excessive context
switching, which will hurt performance.
2. Each calculation outputs to a writer with a frame that has two series, the data for
the calculated channel and the data for the calculated channel's index. This means
that if a streamer is receiving data for 10 calculated channel, it will receive ten
separate frames containing data for each channel (and index). This is still the
case even if these channels depend on the same set of source channels or the source
channels are arriving at the same rate. Sending 10 small frames is dramatically less
performance for both the console and the framing infrastructure.
3. Nested calculations i.e. one calc that depends on another calc results in excessive
passing of data between goroutines. Because we go from source writer -> relay -> based calc ->
calc writer -> relay -> nested calc -> nested calc writer -> relay -> out to caller. This
results in a lot of channel communication and, as a result, a lot of excessive CPU
syncrhonization.
4. MOre goroutines, more writers = more cahnnels = more synchronization = lower efficiency.

So the question is, how do we improve the calculated channel architecture to improve it's
performance.
