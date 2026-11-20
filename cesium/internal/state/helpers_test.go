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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// appended holds the entries that a state appends to its log.
type appended struct {
	mu      sync.Mutex
	entries [][]byte
}

func (a *appended) append(e *log.Entry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, append([]byte(nil), e.Bytes()...))
}

// reserves returns the Reserve records of the appended entries, in order.
func (a *appended) reserves() []log.Reserve {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []log.Reserve
	for _, b := range a.entries {
		var d log.Decoder
		MustSucceed(d.Reset(b))
		for {
			r, ok := MustSucceed2(d.Next())
			if !ok {
				break
			}
			if v, ok := r.(*log.Reserve); ok {
				out = append(out, *v)
			}
		}
	}
	return out
}

func config(a *appended) state.Config {
	return state.Config{
		Versions: set.New[uint16](1),
		Version:  1,
		Append:   a.append,
		Executor: func(f func()) { go f() },
	}
}

// head returns an empty durable state with limits.
func head(segments, alignments uint64) *state.Snapshot {
	return &state.Snapshot{
		Limits: state.Limits{Segments: segments, Alignments: alignments},
	}
}

// run runs s until the spec ends.
func run(s *state.State) {
	GinkgoHelper()
	ctx, cancel := context.WithCancel(context.Background())
	ran := make(chan error, 1)
	go func() { ran <- s.Run(ctx) }()
	DeferCleanup(func() {
		if err := s.Err(); err != nil {
			Expect(s.Close()).To(MatchError(err))
		} else {
			Expect(s.Close()).To(Succeed())
		}
		Eventually(ran).Should(Receive(BeNil()))
		cancel()
	})
}

func create(id uint64, key, clock uint32) *log.Create {
	return &log.Create{Segment: id, Key: key, Clock: clock, Version: 1, DataType: 4}
}

func commit(
	samples uint32,
	number, index uint32,
	first, last int64,
	slots ...log.Slot,
) *log.Commit {
	return &log.Commit{
		Samples:   samples,
		Alignment: uint64(telem.NewAlignment(number, index)),
		First:     first,
		Last:      last,
		Slots:     slots,
	}
}

func slot(id uint64, length uint32) log.Slot {
	return log.Slot{Segment: id, Length: length, TailCRC: length * 3}
}

func pointer(
	number, index uint32,
	first, last int64,
	off, length, samples uint32,
) state.Pointer {
	return state.Pointer{
		First:     telem.TimeStamp(first),
		Last:      telem.TimeStamp(last),
		Alignment: telem.NewAlignment(number, index),
		Offset:    off,
		Length:    length,
		Samples:   samples,
	}
}
