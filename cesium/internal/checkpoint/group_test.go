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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Group commit", func() {
	It("Should commit from writers that sync their segments while the log switches",
		func() {
			fs := xfs.NewMem()
			Expect(log.Prepare(fs, 1, logSize)).To(Succeed())
			Expect(
				checkpoint.Write(fs, 1, checkpoint.Encode(db, 1, &state.Snapshot{})),
			).
				To(Succeed())
			var (
				l     *log.Log
				cp    *checkpoint.Checkpointer
				ready = make(chan uint64, 8)
			)
			st := MustSucceed(state.Open(state.Config{
				Versions: set.New[uint16](1),
				Version:  1,
				Append:   func(e *log.Entry) { l.Append(e) },
				Executor: goExec,
				OnHead:   func(g uint64, s *state.Snapshot) { cp.Head(g, s) },
			}, &state.Snapshot{}))
			cp = MustSucceed(checkpoint.New(checkpoint.Config{
				FS:      fs,
				DB:      db,
				LogSize: logSize,
				Prepared: func(next uint64, err error) {
					l.Prepared(next, err)
					ready <- next
				},
				Executor: goExec,
			}))
			l = MustSucceed(log.Start(log.Config{
				FS:       fs,
				Executor: goExec,
				OnDurable: func(seq uint64, records []log.Record) {
					Expect(st.Durable(seq, records)).To(Succeed())
				},
				OnRefused: func(records []log.Record, err error) {
					Expect(st.Refused(records, err)).To(Succeed())
				},
				Prepare:    cp.Prepare,
				Switched:   func(g uint64) { Expect(st.Switch(g)).To(Succeed()) },
				DB:         db,
				Generation: 1,
				Sequence:   1,
			}))
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)
			var owners sync.WaitGroup
			for _, run := range []func(context.Context) error{st.Run, l.Run, cp.Run} {
				owners.Go(func() { Expect(run(ctx)).To(Succeed()) })
			}
			DeferCleanup(func() {
				Expect(l.Close()).To(Succeed())
				Expect(cp.Close()).To(Succeed())
				Expect(st.Close()).To(Succeed())
				owners.Wait()
			})
			Eventually(ready).Should(Receive(Equal(uint64(2))))
			budget := MustSucceed(segment.NewBudget(2, 2, segment.GoExecutor))
			const commits = 40
			var writers sync.WaitGroup
			for k := range uint32(4) {
				key := 10 + k
				writers.Go(func() {
					defer GinkgoRecover()
					w := MustSucceed(st.OpenWriter(state.WriterConfig{
						Keys:  []state.WriterKey{{Key: key, DataType: 4}},
						Clock: key,
						Start: 1,
					}))
					id := MustSucceed(w.NewSegment(key))
					h := MustSucceed(budget.Create(fs, fmt.Sprintf("%016x.seg", id)))
					sw := segment.NewWriter(h, segment.Config{
						DB: db, ID: id, DataType: 4, Clock: key,
					})
					n := MustSucceed(w.NewAlignment())
					for i := range uint32(commits) {
						Expect(sw.Write(make([]byte, 8))).To(Succeed())
						length, tail := sw.Commit()
						Expect(budget.Sync([]*segment.Handle{h})).To(Succeed())
						at := int64(i+1) * 10
						c := &log.Commit{
							Samples:   1,
							Alignment: uint64(telem.NewAlignment(n, i)),
							First:     at,
							Last:      at,
							Slots: []log.Slot{
								{Segment: id, Length: length, TailCRC: tail},
							},
						}
						Expect(w.Commit(c)).To(Succeed())
						var e log.Entry
						if i == 0 {
							e.Add(&log.Create{
								Segment:  id,
								Key:      key,
								Clock:    key,
								Version:  1,
								DataType: 4,
							})
						}
						e.Add(c)
						MustSucceed(l.Append(&e).Wait())
					}
					Expect(w.Close()).To(Succeed())
					Expect(h.Close()).To(Succeed())
				})
			}
			for i := range uint32(fillers) {
				var filler log.Entry
				for k := range uint32(10_000) {
					filler.Add(&log.Intent{Key: 1_000_000 + i*10_000 + k})
				}
				MustSucceed(l.Append(&filler).Wait())
			}
			writers.Wait()
			Eventually(func() error { return readErr(fs, 2) }).Should(Succeed())
			Eventually(cp.Err).Should(Succeed())
			snap := MustSucceed(st.Snapshot())
			Expect(snap.Channels).To(HaveLen(4))
			for _, ch := range snap.Channels {
				Expect(ch.Segments).To(HaveLen(1))
				Expect(ch.Segments[0].Pointers).To(HaveLen(1))
				Expect(ch.Segments[0].Pointers[0].Samples).To(Equal(uint32(commits)))
			}
		})
})
