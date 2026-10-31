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
	"sync"
	"testing"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
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
	ctx := b.Context()
	opts := []cesium.Option{cesium.WithInstrumentation(alamos.New("bench"))}
	dir := b.TempDir()
	if mem {
		dir = ""
		opts = append(opts, cesium.WithFS(fs.NewMem()))
	}
	db, err := cesium.Open(ctx, dir, opts...)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			b.Error(err)
		}
	}()
	keys := []cesium.ChannelKey{1, 2}
	if err := db.CreateChannel(
		ctx,
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
	const samples = 100
	frame := func(start telem.TimeStamp) cesium.Frame {
		ts := make([]telem.TimeStamp, samples)
		for i := range ts {
			ts[i] = start + telem.TimeStamp(i)
		}
		return telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(ts),
			telem.NewSeries(make([]float32, samples)),
		})
	}
	open := func(a control.Authority) *cesium.Writer {
		w, err := db.OpenWriter(ctx, cesium.WriterConfig{
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
	w := open(control.AuthorityAbsolute)
	var (
		wg   sync.WaitGroup
		done = make(chan struct{})
	)
	stop := sync.OnceFunc(func() {
		close(done)
		wg.Wait()
	})
	if contended {
		lw := open(control.AuthorityAbsolute - 1)
		wg.Go(func() {
			f := frame(telem.SecondTS)
			for {
				select {
				case <-done:
					return
				default:
				}
				if _, err := lw.Write(f); err != nil {
					b.Error(err)
					return
				}
			}
		})
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
		if _, err := w.Write(frame(start)); err != nil {
			b.Fatal(err)
		}
		start += samples
	}
	b.StopTimer()
	stop()
	if err := w.Close(); err != nil {
		b.Error(err)
	}
}
