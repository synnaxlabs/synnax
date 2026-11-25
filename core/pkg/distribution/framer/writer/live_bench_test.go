// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	xtestutil "github.com/synnaxlabs/x/testutil"
)

// liveShape is the stream of writes of one driver task.
type liveShape struct {
	name string
	// channels is the number of data channels, which share one index.
	channels int
	dataType telem.DataType
	rate     telem.Rate
	// samples is the number of samples of each channel in one write.
	samples int
}

var liveShapes = []liveShape{
	{
		name:     "1khz",
		channels: 8,
		dataType: telem.Float64T,
		rate:     telem.Kilohertz,
		samples:  40,
	},
	{
		name:     "25khz-card",
		channels: 16,
		dataType: telem.Float32T,
		rate:     25 * telem.Kilohertz,
		samples:  1000,
	},
	{
		name:     "wide",
		channels: 1000,
		dataType: telem.Float64T,
		rate:     25,
		samples:  1,
	},
}

const (
	// liveWriteRate is the rate at which a driver task writes its frames.
	liveWriteRate telem.Rate = 25
	// liveWarmUp is the number of writes before the measure, so that the first write
	// to each file does not set the largest latency.
	liveWarmUp = 5
)

// liveRig is an auto-commit writer and a streamer on the stored channels of a shape,
// on a node that keeps its data on the OS file system.
type liveRig struct {
	shape  liveShape
	keys   channel.Keys
	writer *writer.Writer
	// data holds the series of each data channel, which every write shares.
	data      []telem.Series
	next      telem.TimeStamp
	responses confluence.Outlet[relay.Response]
	closer    func() error
}

func openLive(b *testing.B, shape liveShape) *liveRig {
	b.Helper()
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	store := xtestutil.MustSucceed(storage.OpenLayer(ctx, storage.LayerConfig{
		InMemory: new(false),
		Dirname:  b.TempDir(),
		Version:  "0.0.0",
	}))
	stopStore := xtestutil.GoRun(ctx, store.Run)
	dist := testutil.OpenNode(ctx, distribution.LayerConfig{Storage: store})
	keys := createLiveChannels(ctx, dist, shape)
	data := make([]telem.Series, shape.channels)
	for i := range data {
		data[i] = telem.Series{
			DataType: shape.dataType,
			Data:     make([]byte, shape.samples*int(shape.dataType.Density())),
		}
	}
	streamer := xtestutil.MustSucceed(dist.Framer.NewStreamer(relay.StreamerConfig{
		Keys:            keys,
		SendOpenAck:     new(true),
		MissedRequested: new(true),
	}))
	sCtx, cancel := signal.Isolated()
	// Unbuffered, as the API's streamer is, so no buffer hides a stall.
	requests, responses := confluence.Attach(streamer)
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	<-responses.Outlet()
	start := 10 * telem.SecondTS
	w := xtestutil.MustSucceed(dist.Framer.OpenWriter(
		ctx,
		writer.Config{Keys: keys, Start: start},
	))
	return &liveRig{
		shape:     shape,
		keys:      keys,
		writer:    w,
		data:      data,
		next:      start,
		responses: responses,
		closer: func() error {
			err := w.Close()
			requests.Close()
			cancel()
			confluence.Drain(responses)
			return errors.Join(err, dist.Close(), stopStore(), store.Close())
		},
	}
}

// createLiveChannels creates the index and the data channels of shape, and returns
// their keys, the index first.
func createLiveChannels(
	ctx context.Context,
	dist testutil.Node,
	shape liveShape,
) channel.Keys {
	index := xtestutil.MustSucceed(dist.CreateChannels(ctx, []channel.Channel{{
		Name:     "time",
		DataType: telem.TimestampT,
		IsIndex:  true,
	}}))[0]
	channels := make([]channel.Channel, shape.channels)
	for i := range channels {
		channels[i] = channel.Channel{
			Name:       fmt.Sprintf("data_%d", i),
			DataType:   shape.dataType,
			LocalIndex: index.LocalKey,
		}
	}
	channels = xtestutil.MustSucceed(dist.CreateChannels(ctx, channels))
	return append(channel.Keys{index.Key()}, channel.KeysFromChannels(channels)...)
}

// frame returns the next frame and the last stamp of its index.
func (r *liveRig) frame() (frame.Frame, telem.TimeStamp) {
	period := r.shape.rate.Period()
	stamps := make([]telem.TimeStamp, r.shape.samples)
	for i := range stamps {
		stamps[i] = r.next
		r.next = r.next.Add(period)
	}
	series := append([]telem.Series{telem.NewSeries(stamps)}, r.data...)
	return frame.NewMulti(r.keys, series), stamps[len(stamps)-1]
}

func (r *liveRig) write(fr frame.Frame) error {
	authorized, err := r.writer.Write(fr)
	if err == nil && !authorized {
		err = errors.New("the writer has no authority")
	}
	return err
}

// await reads the streamer until it delivers the frame whose index ends at last.
func (r *liveRig) await(last telem.TimeStamp) error {
	for res := range r.responses.Outlet() {
		if res.Missed {
			return errors.New("the relay dropped frames for the streamer")
		}
		if index := res.Frame.Get(r.keys[0]); index.Len() > 0 &&
			index.Series[len(index.Series)-1].ValueAt[telem.TimeStamp](-1) == last {
			return nil
		}
	}
	return errors.New("the streamer closed")
}

// writeAndAwait writes fr, whose index ends at last, and waits until the streamer
// delivers it.
func (r *liveRig) writeAndAwait(fr frame.Frame, last telem.TimeStamp) error {
	if err := r.write(fr); err != nil {
		return err
	}
	return r.await(last)
}

func (r *liveRig) warmUp(b *testing.B) {
	b.Helper()
	for range liveWarmUp {
		if err := r.writeAndAwait(r.frame()); err != nil {
			b.Fatal(err)
		}
	}
}

func (r *liveRig) close(b *testing.B) {
	b.Helper()
	if err := r.closer(); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkLiveWriteLatency measures the time from a call to Write on an auto-commit
// writer to the delivery of its frame by a streamer. It counts the write, with any
// commit or sync that it waits for, the relay, and the delivery. Writes come at the
// rate of a driver task, and the time between them does not count, so sec/op is the
// mean latency. It reports the median, the 99th percentile, and the largest latency.
func BenchmarkLiveWriteLatency(b *testing.B) {
	for _, shape := range liveShapes {
		b.Run("shape="+shape.name, func(b *testing.B) {
			r := openLive(b, shape)
			r.warmUp(b)
			pace := time.NewTicker(liveWriteRate.Period().Duration())
			latencies := make([]time.Duration, b.N)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				b.StopTimer()
				<-pace.C
				fr, last := r.frame()
				b.StartTimer()
				start := time.Now()
				if err := r.writeAndAwait(fr, last); err != nil {
					b.Fatal(err)
				}
				latencies[i] = time.Since(start)
			}
			b.StopTimer()
			pace.Stop()
			reportLatencies(b, latencies)
			r.close(b)
		})
	}
}

// BenchmarkLiveWriteThroughput measures how fast an auto-commit writer delivers frames
// to a streamer when it writes them back to back. The timer stops when the streamer
// delivers the last frame. It reports the data samples delivered per second.
func BenchmarkLiveWriteThroughput(b *testing.B) {
	for _, shape := range liveShapes {
		b.Run("shape="+shape.name, func(b *testing.B) {
			r := openLive(b, shape)
			r.warmUp(b)
			samples := b.N * shape.samples
			last := r.next.Add(shape.rate.Period() * telem.TimeSpan(samples-1))
			delivered := make(chan error, 1)
			b.ReportAllocs()
			b.ResetTimer()
			go func() { delivered <- r.await(last) }()
			for range b.N {
				fr, _ := r.frame()
				if err := r.write(fr); err != nil {
					b.Fatal(err)
				}
			}
			err := <-delivered
			b.StopTimer()
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(
				float64(samples*shape.channels)/b.Elapsed().Seconds(),
				"samples/s",
			)
			r.close(b)
		})
	}
}

// reportLatencies reports the median, the 99th percentile, and the largest of
// latencies, by nearest rank.
func reportLatencies(b *testing.B, latencies []time.Duration) {
	b.Helper()
	slices.Sort(latencies)
	at := func(q float64) float64 {
		i := int(math.Ceil(q*float64(len(latencies)))) - 1
		return float64(latencies[max(i, 0)].Nanoseconds())
	}
	b.ReportMetric(at(0.5), "p50-ns")
	b.ReportMetric(at(0.99), "p99-ns")
	b.ReportMetric(at(1), "max-ns")
}
