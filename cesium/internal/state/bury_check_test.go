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
	"math/rand/v2"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// emptySealed returns the identities of the sealed segments with no pointer in s.
func emptySealed(s *state.Snapshot) []uint64 {
	var ids []uint64
	for _, ch := range s.Channels {
		for _, seg := range ch.Segments {
			if seg.Sealed && len(seg.Pointers) == 0 {
				ids = append(ids, seg.ID)
			}
		}
	}
	return ids
}

var _ = Describe("Check: a sealed segment with no pointer", func() {
	It("Should leave none in the state after each event and after the replay of a "+
		"crash at each point", func() {
		const size = 4 << 20
		keys := []state.WriterKey{{Key: 5, DataType: 4}}
		for runKey := range uint64(8) {
			rnd := rand.New(rand.NewPCG(runKey, 1))
			disk := MustSucceed(fault.New(fault.Config{}))
			Expect(log.Prepare(disk, 1, size)).To(Succeed())
			var (
				mu     sync.Mutex
				images []*fault.FS
				l      *log.Log
				goExe  = func(f func()) { go f() }
				// failed holds the errors of the callbacks and of the runs.
				failed error
			)
			fail := func(err error) {
				mu.Lock()
				defer mu.Unlock()
				failed = errors.Join(failed, err)
			}
			take := func() {
				mu.Lock()
				defer mu.Unlock()
				images = append(images, MustSucceed(disk.Crash(fault.CrashConfig{
					RunKey:            runKey + uint64(len(images)),
					EntrySurvivalRate: 0.5,
					DataSurvivalRate:  0.5,
				})))
			}
			st := MustSucceed(state.Open(state.Config{
				Versions: config(&appended{}).Versions,
				Version:  1,
				Append:   func(e *log.Entry) { l.Append(e) },
				Executor: goExe,
				OnHead:   func(uint64, *state.Snapshot) {},
			}, head(1, 1)))
			l = MustSucceed(log.Start(log.Config{
				FS:       imageFS{FS: disk, take: take},
				Executor: goExe,
				OnDurable: func(seq uint64, records []log.Record) {
					fail(st.Durable(seq, records))
				},
				OnRefused: func(records []log.Record, err error) {
					fail(st.Refused(records, err))
				},
				// No next file, so every entry stays in log 1.
				Prepare: func(next uint64) {
					go l.Prepared(next, errors.New("no next log"))
				},
				Switched:   func(uint64) {},
				DB:         7,
				Generation: 1,
				Sequence:   1,
			}))
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			wg.Go(func() { fail(st.Run(ctx)) })
			wg.Go(func() { fail(l.Run(ctx)) })
			add := func(r log.Record) log.Ack {
				var e log.Entry
				e.Add(r)
				return l.Append(&e)
			}
			// pending holds the appends that no step waited on yet.
			var pending []log.Ack
			settle := func() {
				for _, f := range pending {
					MustSucceed(f.Wait())
				}
				pending = pending[:0]
			}
			expectNone := func(event string) {
				GinkgoHelper()
				Expect(emptySealed(MustSucceed(st.Snapshot()))).To(
					BeEmpty(),
					"run key %d, after %s",
					runKey,
					event,
				)
			}
			// Each writer creates a segment and commits zero to two samples, of which
			// the last may not be durable. It then seals the segment, leaves it open
			// with no writer for the next writer or switch to seal, or holds it through
			// the crash, for the Restart of the replay to seal.
			generation := uint64(1)
			for i := range 6 {
				start := telem.TimeStamp(1000 * (i + 1))
				w := MustSucceed(st.OpenWriter(context.Background(), state.WriterConfig{
					Keys: keys, Clock: 5, Start: start, Bounded: true,
				}))
				// A writer takes over the open segment that a closed writer left.
				if _, held := MustSucceed2(w.Held(5)); held {
					seal := MustSucceed(w.Seal(5))
					pending = append(pending, add(&seal))
				}
				id := MustSucceed(w.NewSegment(5))
				MustSucceed(add(create(id, 5, 5)).Wait())
				n := MustSucceed(w.NewAlignment())
				for c := range rnd.IntN(3) {
					first := int64(start) + int64(c)
					cm := commit(
						1,
						n,
						uint32(c),
						first,
						first,
						slot(id, uint32(8*(c+1))),
					)
					Expect(w.Commit(cm)).To(Succeed())
					pending = append(pending, add(cm))
				}
				expectNone("the commits")
				switch rnd.IntN(3) {
				case 0:
					seal := MustSucceed(w.Seal(5))
					pending = append(pending, add(&seal))
					settle()
					Expect(w.Close()).To(Succeed())
					expectNone("a Seal")
				case 1:
					settle()
					Expect(w.Close()).To(Succeed())
					expectNone("a close")
				}
				if rnd.IntN(2) == 0 {
					settle()
					generation++
					Expect(st.Switch(generation)).To(Succeed())
					expectNone("a switch")
				}
			}
			settle()
			expectNone("the last event")
			Expect(l.Close()).To(Succeed())
			Expect(st.Close()).To(Succeed())
			cancel()
			wg.Wait()
			mu.Lock()
			Expect(failed).ToNot(HaveOccurred())
			Expect(len(images)).To(BeNumerically(">", 6))
			for i, image := range append(images, disk) {
				s := MustSucceed(state.Open(config(&appended{}), head(1, 1)))
				next := MustSucceed(replayInto(s, image, 1))
				Expect(s.Replay(next, []log.Record{&log.Restart{}})).To(Succeed())
				ran := make(chan error, 1)
				runCtx, stop := context.WithCancel(context.Background())
				go func() { ran <- s.Run(runCtx) }()
				snapshot := MustSucceed(s.Snapshot())
				Expect(s.Close()).To(Succeed())
				stop()
				Expect(<-ran).To(Succeed())
				Expect(emptySealed(snapshot)).To(
					BeEmpty(),
					"run key %d, crash image %d of %d",
					runKey,
					i,
					len(images),
				)
				MustSucceed(state.Open(config(&appended{}), snapshot))
			}
			mu.Unlock()
		}
	})
})
