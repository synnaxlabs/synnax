// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

// BenchmarkWrite writes and commits 100 samples to an index and a data channel per
// iteration. With contended set, a lower-authority writer retries writes to the same
// channels for the whole run.
func BenchmarkWrite(b *testing.B) {
	for _, fsName := range []string{"mem", "os"} {
		for _, contended := range []bool{false, true} {
			name := "fs=" + fsName + "/single"
			if contended {
				name = "fs=" + fsName + "/contended"
			}
			b.Run(name, func(b *testing.B) {
				benchmarkWrite(b, fsName == "mem", contended)
			})
		}
	}
}

func benchmarkWrite(b *testing.B, mem, contended bool) {
	b.Helper()
	db := openBenchDB(b, mem)
	keys := []cesium.ChannelKey{1, 2}
	createBenchChannels(b, db, keys)
	w := openBenchWriter(b, db, keys, control.AuthorityAbsolute)
	stop := func() {}
	if contended {
		lw := openBenchWriter(b, db, keys, control.AuthorityAbsolute-1)
		stop = retryWrites(b, lw, keys)
		defer func() {
			if err := lw.Close(); err != nil {
				b.Error(err)
			}
		}()
	}
	// Runs before the deferred close above, so a failed iteration stops the retry
	// goroutine before its writer closes.
	defer stop()
	b.ReportAllocs()
	start := telem.SecondTS
	for b.Loop() {
		if _, err := w.Write(b.Context(), benchFrame(keys, start)); err != nil {
			b.Fatal(err)
		}
		start += benchFrameSamples
	}
	b.StopTimer()
	stop()
	if err := w.Close(); err != nil {
		b.Error(err)
	}
}

// BenchmarkCommit writes one frame of float32 samples and commits it to the index files
// on the OS file system, per iteration.
func BenchmarkCommit(b *testing.B) {
	for _, c := range []struct{ indexes, channels, samples int }{
		{indexes: 1, channels: 32, samples: 125},
		{indexes: 10, channels: 990, samples: 100},
	} {
		name := fmt.Sprintf(
			"indexes=%d/channels=%d/samples=%d",
			c.indexes,
			c.channels,
			c.samples,
		)
		b.Run(name, func(b *testing.B) {
			perIndex := c.channels / c.indexes
			db := openBenchDB(b, false)
			keys := createIndexedChannels(b, db, c.indexes, perIndex)
			w, err := db.OpenWriter(b.Context(), cesium.WriterConfig{
				Channels:                 keys,
				Start:                    telem.SecondTS,
				Sync:                     new(true),
				AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
			})
			if err != nil {
				b.Fatal(err)
			}
			data := telem.NewSeries(make([]float32, c.samples))
			stamps := make([]telem.TimeStamp, c.samples)
			start := telem.SecondTS
			b.ReportAllocs()
			for b.Loop() {
				for i := range stamps {
					stamps[i] = start + telem.TimeStamp(i)
				}
				// The relay can hold a frame after Write returns, so each write gets
				// its own series slice and stamps.
				series := make([]telem.Series, len(keys))
				for i := range series {
					series[i] = data
				}
				time := telem.NewSeries(stamps)
				for i := 0; i < len(series); i += perIndex + 1 {
					series[i] = time
				}
				if _, err := w.Write(
					b.Context(),
					telem.MultiFrame(keys, series),
				); err != nil {
					b.Fatal(err)
				}
				start += telem.TimeStamp(c.samples)
			}
			b.StopTimer()
			if err := w.Close(); err != nil {
				b.Error(err)
			}
		})
	}
}

// openBenchDB opens a DB on an in-memory file system when mem is true, and in a
// temporary directory otherwise. The DB runs until the benchmark ends, then closes.
func openBenchDB(b *testing.B, mem bool) *cesium.DB {
	b.Helper()
	opts := []cesium.Option{cesium.WithInstrumentation(alamos.New("bench"))}
	dir := b.TempDir()
	if mem {
		dir = ""
		opts = append(opts, cesium.WithFS(fs.NewMem()))
	}
	db, err := cesium.Open(b.Context(), dir, opts...)
	if err != nil {
		b.Fatal(err)
	}
	stop := testutil.GoRun(b.Context(), db.Run)
	b.Cleanup(func() {
		if err := errors.Join(stop(), db.Close()); err != nil {
			b.Error(err)
		}
	})
	return db
}

// createBenchChannels creates an index channel keys[0] and a float32 data channel
// keys[1] that it indexes.
func createBenchChannels(b *testing.B, db *cesium.DB, keys []cesium.ChannelKey) {
	b.Helper()
	if err := db.CreateChannel(
		b.Context(),
		cesium.Channel{
			Key:      keys[0],
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		},
		cesium.Channel{
			Key:      keys[1],
			Name:     "data",
			Index:    keys[0],
			DataType: telem.Float32T,
		},
	); err != nil {
		b.Fatal(err)
	}
}

const benchFrameSamples = 100

// benchFrame returns a frame of benchFrameSamples samples for the channels
// createBenchChannels creates, with timestamps from start.
func benchFrame(keys []cesium.ChannelKey, start telem.TimeStamp) cesium.Frame {
	ts := make([]telem.TimeStamp, benchFrameSamples)
	for i := range ts {
		ts[i] = start + telem.TimeStamp(i)
	}
	return telem.MultiFrame(keys, []telem.Series{
		telem.NewSeries(ts),
		telem.NewSeries(make([]float32, benchFrameSamples)),
	})
}

func openBenchWriter(
	b *testing.B,
	db *cesium.DB,
	keys []cesium.ChannelKey,
	a control.Authority,
) *cesium.Writer {
	b.Helper()
	w, err := db.OpenWriter(b.Context(), cesium.WriterConfig{
		Channels:    keys,
		Start:       telem.SecondTS,
		Authorities: []control.Authority{a},
		Sync:        new(true),
	})
	if err != nil {
		b.Fatal(err)
	}
	return w
}

// retryWrites writes the same frame to w in a loop until the returned function runs.
func retryWrites(b *testing.B, w *cesium.Writer, keys []cesium.ChannelKey) func() {
	b.Helper()
	var (
		wg   sync.WaitGroup
		done = make(chan struct{})
	)
	wg.Go(func() {
		f := benchFrame(keys, telem.SecondTS)
		for {
			select {
			case <-done:
				return
			default:
			}
			if _, err := w.Write(
				b.Context(),
				f,
			); err != nil && !errors.Is(err, control.ErrUnauthorized) {
				b.Error(err)
				return
			}
		}
	})
	return sync.OnceFunc(func() {
		close(done)
		wg.Wait()
	})
}
