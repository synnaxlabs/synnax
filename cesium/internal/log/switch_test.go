// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"context"
	"encoding/binary"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// switches records the calls that a log makes to its switch hooks.
type switches struct {
	mu       sync.Mutex
	prepared []uint64
	switched []uint64
}

func (s *switches) prepare(next uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prepared = append(s.prepared, next)
}

func (s *switches) onSwitched(gen uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.switched = append(s.switched, gen)
}

func (s *switches) get() ([]uint64, []uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]uint64(nil), s.prepared...), append([]uint64(nil), s.switched...)
}

var _ = Describe("Switch", func() {
	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var (
				fs    *opFS
				l     *log.Log
				ran   chan error
				hook  *durables
				calls *switches
			)

			// open starts log 3 of logSize bytes, and answers each Prepare with answer.
			open := func(answer func(next uint64)) {
				GinkgoHelper()
				fs, hook, calls = newOpFS(makeFS()), &durables{}, &switches{}
				Expect(log.Prepare(fs, generation, logSize)).To(Succeed())
				l = MustSucceed(log.Start(log.Config{
					FS:        fs,
					Executor:  goExec,
					OnDurable: hook.on,
					OnRefused: hook.onRefused,
					Prepare: func(next uint64) {
						calls.prepare(next)
						go answer(next)
					},
					Switched:   calls.onSwitched,
					DB:         db,
					Generation: generation,
					Sequence:   5,
				}))
				ran = make(chan error, 1)
				ctx, cancel := context.WithCancel(context.Background())
				go func() { ran <- l.Run(ctx) }()
				DeferCleanup(func() {
					Expect(l.Close()).To(Succeed())
					Eventually(ran).Should(Receive(BeNil()))
					cancel()
				})
			}

			ready := func(next uint64) {
				l.Prepared(next, log.Prepare(fs, next, logSize))
			}

			// fill appends entries until the log has frames past half of its size.
			fill := func() uint64 {
				GinkgoHelper()
				var seq uint64
				for range 3 {
					seq = MustSucceed(l.Append(bigEntry(logSize / 4)).Wait())
				}
				return seq
			}

			It(
				"Should switch to the next log before a frame passes three quarters, "+
					"and link it to the last frame",
				func() {
					done := make(chan uint64, 4)
					open(func(next uint64) {
						ready(next)
						done <- next
					})
					Eventually(done).Should(Receive(Equal(uint64(generation + 1))))
					last := fill()
					Expect(last).To(Equal(uint64(7)))
					old := walkLog(logBytes(fs, generation), generation)
					next := logBytes(fs, generation+1)
					h := MustSucceed(log.DecodeHeader(next, db, generation+1, 0))
					Expect(h.Sequence).To(Equal(uint64(7)))
					Expect(h.PrevCRC).To(Equal(old[len(old)-1].CRC))
					frames := walkLog(next, generation+1)
					Expect(frames[0].Sequence).To(Equal(uint64(7)))
					Expect(frames[0].PrevCRC).
						To(Equal(binary.LittleEndian.Uint32(next[60:])))
					Eventually(func() []uint64 {
						p, _ := calls.get()
						return p
					}).Should(Equal([]uint64{4, 5}))
					_, switched := calls.get()
					Expect(switched).To(Equal([]uint64{4}))
				},
			)

			It(
				"Should go on in the live log past three quarters until the next file is "+
					"ready, then switch",
				func() {
					asked := make(chan uint64, 4)
					open(func(next uint64) { asked <- next })
					Eventually(asked).Should(Receive(Equal(uint64(generation + 1))))
					Expect(fill()).To(Equal(uint64(7)))
					Expect(
						MustSucceed(fs.Exists(log.Name(generation + 1))),
					).To(BeFalse())
					ready(generation + 1)
					Expect(l.Append(bigEntry(logSize / 8)).Wait()).To(Equal(uint64(8)))
					frames := walkLog(logBytes(fs, generation+1), generation+1)
					Expect(frames[0].Sequence).To(Equal(uint64(8)))
					_, switched := calls.get()
					Expect(switched).To(Equal([]uint64{4}))
				},
			)

			It("Should stay in the live log when the next file fails, and ask again",
				func() {
					open(func(next uint64) { l.Prepared(next, errors.New("no space")) })
					Expect(fill()).To(Equal(uint64(7)))
					var entries uint32
					for _, f := range walk(fileBytes(fs)) {
						entries += f.Count
					}
					Expect(entries).To(Equal(uint32(3)))
					Eventually(func() []uint64 {
						p, _ := calls.get()
						return p
					}).Should(Equal([]uint64{4, 4}))
					_, switched := calls.get()
					Expect(switched).To(BeEmpty())
				})
		})
	}
})
