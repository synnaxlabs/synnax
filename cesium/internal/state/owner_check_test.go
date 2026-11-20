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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("State owner check", func() {
	cfg := state.Config{
		Versions: set.New[uint16](1),
		Version:  1,
		Append:   func(*log.Entry) {},
		Executor: func(f func()) { go f() },
	}
	keys := []state.WriterKey{{Key: 5, DataType: 4}}
	create := func(id uint64) *log.Create {
		return &log.Create{Segment: id, Key: 5, Clock: 9, Version: 1, DataType: 4}
	}
	run := func(s *state.State) {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		go func() { _ = s.Run(ctx) }()
	}
	open := func() *state.State {
		GinkgoHelper()
		s := MustSucceed(state.Open(cfg, &state.Snapshot{
			Limits: state.Limits{Segments: 1, Alignments: 1},
		}))
		run(s)
		return s
	}

	It("Should not seal at a switch a segment that no durable Create made", func() {
		s := open()
		w := MustSucceed(s.OpenWriter(state.WriterConfig{
			Keys: []state.WriterKey{{Key: 5, DataType: 4}}, Clock: 5, Start: 10,
		}))
		MustSucceed(w.NewSegment(5))
		Expect(s.Durable(1, []log.Record{
			&log.Reserve{Segments: 17, Alignments: 1},
		})).To(Succeed())
		Expect(w.Close()).To(Succeed())
		seals := MustSucceed(s.SealUnheld())
		records := make([]log.Record, len(seals))
		for i := range seals {
			records[i] = &seals[i]
		}
		Expect(s.Durable(2, records)).To(Succeed())
		MustSucceed(s.Snapshot())
		Expect(s.Err()).ToNot(HaveOccurred())
	})

	It("Should seal every open segment of a key after a takeover and a close", func() {
		s := open()
		a := MustSucceed(s.OpenWriter(state.WriterConfig{
			Keys: keys, Clock: 9, Start: 100,
		}))
		b := MustSucceed(s.OpenWriter(state.WriterConfig{
			Keys: keys, Clock: 9, Start: 10, Bounded: true,
		}))
		ia, ib := MustSucceed(a.NewSegment(5)), MustSucceed(b.NewSegment(5))
		Expect(s.Durable(1, []log.Record{
			&log.Reserve{Segments: 17, Alignments: 1}, create(ia), create(ib),
		})).To(Succeed())
		Expect(a.Close()).To(Succeed())
		Expect(b.Close()).To(Succeed())
		c := MustSucceed(s.OpenWriter(state.WriterConfig{
			Keys: keys, Clock: 9, Start: 200,
		}))
		Expect(c.Close()).To(Succeed())
		seals := MustSucceed(s.SealUnheld())
		ids := make([]uint64, len(seals))
		for i, seal := range seals {
			ids[i] = seal.Segment
		}
		Expect(ids).To(ConsistOf(ia, ib))
	})

	It("Should never give out segment 0", func() {
		s, err := state.Open(cfg, &state.Snapshot{})
		if err != nil {
			Expect(err).To(MatchError(state.ErrRefused))
			return
		}
		run(s)
		w := MustSucceed(s.OpenWriter(state.WriterConfig{
			Keys: keys, Clock: 5, Start: 10,
		}))
		Expect(MustSucceed(w.NewSegment(5))).ToNot(BeZero())
	})
})
