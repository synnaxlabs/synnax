// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

// switchState returns a state of keys clocks, each with four sealed segments and one
// open segment of one pointer each.
func switchState(keys int) *state.Snapshot {
	s := &state.Snapshot{Limits: state.Limits{
		Segments:   uint64(5*keys + 1),
		Alignments: uint64(5*keys + 1),
	}}
	id := uint64(0)
	for k := range keys {
		ch := &state.Channel{Key: uint32(k + 1)}
		for i := range 5 {
			id++
			ch.Segments = append(ch.Segments, state.Segment{
				ID: id, Clock: ch.Key, Version: 1, DataType: 1, Sealed: i < 4,
				Length: 64,
				Pointers: []state.Pointer{{
					First:     telem.TimeStamp(100 * i),
					Last:      telem.TimeStamp(100*i + 7),
					Alignment: telem.NewAlignment(uint32(id), 0),
					Length:    64,
					Samples:   8,
				}},
			})
		}
		s.Channels = append(s.Channels, ch)
	}
	return s
}

// BenchmarkSwitch measures one switch of the checkpointer at a steady state: the diff
// against the parts of the last head, the head, and the clean, which reads and checks
// the older head because its log exists, as at every switch of a running database. The
// first switch, which writes the parts of a state past the part size, runs before the
// timer, and so does the file of each log.
func BenchmarkSwitch(b *testing.B) {
	for _, keys := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("keys=%d", keys), func(b *testing.B) {
			stepped := make(chan struct{}, 1)
			fs := xfs.NewMem()
			c, err := checkpoint.New(checkpoint.Config{
				FS:       fs,
				DB:       1,
				LogSize:  1 << 20,
				Prepared: func(uint64, error) {},
				Executor: func(f func()) {
					f()
					stepped <- struct{}{}
				},
			})
			if err != nil {
				b.Fatal(err)
			}
			ran := make(chan error, 1)
			go func() { ran <- c.Run(b.Context()) }()
			b.Cleanup(func() {
				if err := c.Close(); err != nil {
					b.Error(err)
				}
				if err := <-ran; err != nil && !errors.Is(err, context.Canceled) {
					b.Error(err)
				}
			})
			s, generation := switchState(keys), uint64(1)
			step := func() {
				b.StopTimer()
				f, err := fs.Open(log.Name(generation), os.O_CREATE|os.O_RDWR)
				if err == nil {
					err = f.Close()
				}
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				c.Head(generation, s)
				<-stepped
			}
			step()
			b.ReportAllocs()
			for b.Loop() {
				generation++
				step()
			}
		})
	}
}
