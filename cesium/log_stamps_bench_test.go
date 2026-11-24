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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

// partOpens counts the opens of part files.
type partOpens struct {
	xfs.FS
	n atomic.Int64
}

func (f *partOpens) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f *partOpens) Open(name string, flag int) (xfs.File, error) {
	if strings.HasSuffix(name, ".part") {
		f.n.Add(1)
	}
	return f.FS.Open(name, flag)
}

// BenchmarkCommitOldDataPointerLog commits 10 samples per iteration from a writer that
// does not write its clock, on a clock of 50 stretches that 50 parts hold: from the
// oldest stretch, and from the newest. It reports the opens of part files per commit.
// Run it with -benchtime 1000x, so that the writer stays in one stretch.
func BenchmarkCommitOldDataPointerLog(b *testing.B) {
	const stretches, stamps, samples = 50, 10_000, 10
	fs := &partOpens{FS: xfs.NewMem()}
	db, err := cesium.Open(
		b.Context(),
		"",
		cesium.WithFS(fs),
		cesium.WithPointerLog(),
		cesium.WithPartSize(1),
		cesium.WithInstrumentation(alamos.New("bench")),
	)
	if err != nil {
		b.Fatal(err)
	}
	stop := testutil.GoRun(b.Context(), db.Run)
	b.Cleanup(func() {
		if err := errors.Join(stop(), db.Close()); err != nil {
			b.Error(err)
		}
	})
	keys := createIndexedChannels(b, db, 1, 2)
	at := func(stretch, i int) telem.TimeStamp {
		return telem.TimeStamp(stretch*stamps*2+i+1) * telem.SecondTS
	}
	writeStretches(b, db, fs.FS, keys[0], stretches, stamps, at)
	for _, c := range []struct {
		name    string
		key     cesium.ChannelKey
		stretch int
	}{{"oldest", keys[1], 0}, {"newest", keys[2], stretches - 1}} {
		b.Run(c.name, func(b *testing.B) {
			w, err := db.OpenWriter(b.Context(), cesium.WriterConfig{
				Channels: []cesium.ChannelKey{c.key},
				Clock:    keys[0],
				Start:    at(c.stretch, 0),
			})
			if err != nil {
				b.Fatal(err)
			}
			values := telem.NewSeries(make([]float32, samples))
			before := fs.n.Load()
			b.ReportAllocs()
			for b.Loop() {
				writeAndCommit(b, w, telem.UnaryFrame(c.key, values))
			}
			b.ReportMetric(float64(fs.n.Load()-before)/float64(b.N), "part-opens/op")
			if err := w.Close(); err != nil {
				b.Error(err)
			}
		})
	}
}

// writeStretches writes stretches stretches of stamps stamps of clock to db, at the
// stamps that at gives, and switches the log after each, so that a part holds each.
func writeStretches(
	b *testing.B,
	db *cesium.DB,
	fs xfs.FS,
	clock cesium.ChannelKey,
	stretches, stamps int,
	at func(stretch, i int) telem.TimeStamp,
) {
	b.Helper()
	series := make([]telem.TimeStamp, stamps)
	for s := range stretches {
		for i := range series {
			series[i] = at(s, i)
		}
		w, err := db.OpenWriter(b.Context(), cesium.WriterConfig{
			Channels: []cesium.ChannelKey{clock},
			Clock:    clock,
			Start:    series[0],
		})
		if err != nil {
			b.Fatal(err)
		}
		writeAndCommit(b, w, telem.UnaryFrame(clock, telem.NewSeries(series)))
		if err = w.Close(); err != nil {
			b.Fatal(err)
		}
		generation := uint64(s + 2)
		if err = switchLog(db, fs, generation); err != nil {
			b.Fatal(err)
		}
		for deadline := time.Now().Add(time.Minute); ; time.Sleep(time.Millisecond) {
			if _, _, err = checkpoint.Read(fs, generation); err == nil {
				break
			}
			if time.Now().After(deadline) {
				b.Fatal(err)
			}
		}
	}
}
