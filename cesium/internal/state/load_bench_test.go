// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkLoad loads a clock and a key that it times while one writer commits a sample
// of both in a loop. Each case runs its loads on that many goroutines.
func BenchmarkLoad(b *testing.B) {
	for _, readers := range []int{1, 8} {
		b.Run(fmt.Sprintf("readers=%d", readers), func(b *testing.B) {
			s := runBenchState(b)
			stop := commitLoop(b, s)
			keys := []uint32{1, 2}
			b.ReportAllocs()
			b.ResetTimer()
			var wg sync.WaitGroup
			for r := range readers {
				wg.Go(func() {
					chs := make([]*state.Channel, len(keys))
					for i := r; i < b.N; i += readers {
						s.Load(keys, chs)
					}
				})
			}
			wg.Wait()
			b.StopTimer()
			if err := stop(); err != nil {
				b.Fatal(err)
			}
		})
	}
}

// benchLog hands each entry that a state appends back to it as durable at once.
type benchLog struct {
	b   *testing.B
	s   *state.State
	seq uint64
}

// append runs on a step of the owner, so the entries come in log order.
func (l *benchLog) append(e *log.Entry) {
	var (
		d       log.Decoder
		records []log.Record
	)
	if _, err := d.Reset(append([]byte(nil), e.Bytes()...)); err != nil {
		l.b.Error(err)
		return
	}
	for {
		r, ok, err := d.Next()
		if err != nil {
			l.b.Error(err)
			return
		}
		if !ok {
			break
		}
		records = append(records, r)
	}
	l.seq++
	if err := l.s.Durable(l.seq, records); err != nil {
		l.b.Error(err)
	}
}

// runBenchState runs a state whose appended entries are durable at once.
func runBenchState(b *testing.B) *state.State {
	b.Helper()
	l := &benchLog{b: b}
	s, err := state.Open(state.Config{
		Versions: set.New[uint16](1),
		Version:  1,
		Append:   l.append,
		Executor: func(f func()) { go f() },
		OnHead:   func(uint64, *state.Snapshot) {},
	}, &state.Snapshot{})
	if err != nil {
		b.Fatal(err)
	}
	l.s = s
	ctx, cancel := context.WithCancel(context.Background())
	ran := make(chan error, 1)
	go func() { ran <- s.Run(ctx) }()
	b.Cleanup(func() {
		if err := errors.Join(s.Close(), <-ran); err != nil {
			b.Error(err)
		}
		cancel()
	})
	return s
}

// commitLoop commits one sample of clock 1 and key 2 in a loop until the returned
// function stops it.
func commitLoop(b *testing.B, s *state.State) (stop func() error) {
	b.Helper()
	w, err := s.OpenWriter(context.Background(), state.WriterConfig{
		Keys:  []state.WriterKey{{Key: 1, DataType: 4}, {Key: 2, DataType: 4}},
		Clock: 1,
		Start: 1,
	})
	if err != nil {
		b.Fatal(err)
	}
	clock, err := w.NewSegment(1)
	if err != nil {
		b.Fatal(err)
	}
	data, err := w.NewSegment(2)
	if err != nil {
		b.Fatal(err)
	}
	number, err := w.NewAlignment()
	if err != nil {
		b.Fatal(err)
	}
	done, committed := make(chan struct{}), make(chan error, 1)
	go func() {
		for i := uint32(0); ; i++ {
			select {
			case <-done:
				committed <- w.Close()
				return
			default:
			}
			if err := w.Commit(&log.Commit{
				Samples:   1,
				Alignment: uint64(telem.NewAlignment(number, i)),
				First:     int64(i + 1),
				Last:      int64(i + 1),
				Slots: []log.Slot{
					{Segment: clock, Length: 8 * (i + 1)},
					{Segment: data, Length: 4 * (i + 1)},
				},
			}); err != nil {
				committed <- err
				return
			}
		}
	}()
	return func() error {
		close(done)
		return <-committed
	}
}
