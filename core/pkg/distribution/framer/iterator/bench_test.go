// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/telem"
	xtestutil "github.com/synnaxlabs/x/testutil"
)

// streamChunkSize is the chunk size of the Python client.
const streamChunkSize = 100_000

// openStreamData opens a node with an index and data float32 channels on it, writes
// samples stamps a millisecond apart with zeros in each data channel, and returns the
// node, the keys (the index first), and the time range written.
func openStreamData(
	b *testing.B,
	data, samples int,
) (testutil.Node, channel.Keys, telem.TimeRange) {
	b.Helper()
	gomega.RegisterTestingT(b)
	dist := testutil.OpenNode(b.Context())
	b.Cleanup(func() {
		if err := dist.Close(); err != nil {
			b.Error(err)
		}
	})
	idx := xtestutil.MustSucceed(dist.CreateChannels(b.Context(), []channel.Channel{{
		Name:     "stream_time",
		IsIndex:  true,
		DataType: telem.TimestampT,
	}}))[0]
	channels := []channel.Channel{idx}
	for i := range data {
		channels = append(channels, channel.Channel{
			Name:       fmt.Sprintf("stream_data_%d", i),
			LocalIndex: idx.LocalKey,
			DataType:   telem.Float32T,
		})
	}
	channels = append(
		channels[:1],
		xtestutil.MustSucceed(dist.CreateChannels(b.Context(), channels[1:]))...,
	)
	keys := channel.KeysFromChannels(channels)
	start := telem.SecondTS
	w := xtestutil.MustSucceed(dist.Framer.OpenWriter(b.Context(), writer.Config{
		Keys:  keys,
		Start: start,
		Sync:  new(true),
	}))
	const batch = 100_000
	for written := 0; written < samples; written += batch {
		n := min(batch, samples-written)
		stamps := make([]telem.TimeStamp, n)
		for i := range stamps {
			stamps[i] = start + telem.TimeStamp(written+i)*telem.MillisecondTS
		}
		series := []telem.Series{telem.NewSeries(stamps)}
		for range data {
			series = append(series, telem.NewSeries(make([]float32, n)))
		}
		if authorized, err := w.Write(frame.NewMulti(keys, series)); err != nil ||
			!authorized {
			b.Fatalf("write failed: authorized=%v, err=%v", authorized, err)
		}
	}
	if _, authorized, err := w.Commit(); err != nil || !authorized {
		b.Fatalf("commit failed: authorized=%v, err=%v", authorized, err)
	}
	if err := w.Close(); err != nil {
		b.Fatal(err)
	}
	end := start + telem.TimeStamp(samples)*telem.MillisecondTS
	return dist, keys, start.Range(end)
}

// readAll reads every sample of keys in tr through one iterator and returns the count.
func readAll(
	b *testing.B,
	dist testutil.Node,
	keys channel.Keys,
	tr telem.TimeRange,
) int64 {
	iter, err := dist.Framer.OpenIterator(b.Context(), iterator.Config{
		Keys:      keys,
		Bounds:    tr,
		ChunkSize: streamChunkSize,
	})
	if err != nil {
		b.Error(err)
		return 0
	}
	var n int64
	if iter.SeekFirst() && iter.Next(tr.Span()) {
		for s := range iter.Value().Series() {
			n += s.Len()
		}
	}
	if err = iter.Close(); err != nil {
		b.Error(err)
	}
	return n
}

// BenchmarkStream reads an index and its data channels through the framer's iterator,
// in chunks of the Python client's chunk size, with one reader or eight at once.
func BenchmarkStream(b *testing.B) {
	for _, samples := range []int{1_000_000, 4_000_000} {
		for _, data := range []int{1, 8} {
			for _, readers := range []int{1, 8} {
				name := fmt.Sprintf(
					"samples=%d/channels=%d/readers=%d",
					samples,
					data,
					readers,
				)
				b.Run(name, func(b *testing.B) {
					dist, keys, tr := openStreamData(b, data, samples)
					want := int64(samples * len(keys))
					b.ReportAllocs()
					b.ResetTimer()
					for range b.N {
						var wg sync.WaitGroup
						for range readers {
							wg.Go(func() {
								if n := readAll(b, dist, keys, tr); n != want {
									b.Errorf("read %d samples, want %d", n, want)
								}
							})
						}
						wg.Wait()
					}
				})
			}
		}
	}
}
