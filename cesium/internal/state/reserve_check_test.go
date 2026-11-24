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
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// hookErrs holds the errors that the hooks of the log owner get from the state.
type hookErrs struct {
	mu   sync.Mutex
	errs []error
}

func (h *hookErrs) add(err error) {
	if err == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errs = append(h.errs, err)
}

func (h *hookErrs) err() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return errors.Join(h.errs...)
}

var _ = Describe("Check: a Reserve on a full log", func() {
	It("Should keep the state when a Reserve fails on a full log and the Create that "+
		"uses its value lands in the next log", func() {
		const size = 4 << 20
		fs := xfs.NewMem()
		Expect(log.Prepare(fs, 1, size)).To(Succeed())
		var (
			l     *log.Log
			room  atomic.Bool
			made  atomic.Int32
			errs  hookErrs
			goExe = func(f func()) { go f() }
		)
		st := MustSucceed(state.Open(state.Config{
			Versions:   set.New[uint16](1),
			Version:    1,
			Append:     func(e *log.Entry) { l.Append(e) },
			Executor:   goExe,
			OnHead:     func(uint64, *state.Snapshot) {},
			Generation: 1,
		}, head(1, 1)))
		l = MustSucceed(log.Start(log.Config{
			FS:       fs,
			Executor: goExe,
			OnDurable: func(seq uint64, records []log.Record) {
				errs.add(st.Durable(seq, records))
			},
			// The disk has no room for the next log until room is set.
			Prepare: func(next uint64) {
				go func() {
					if !room.Load() {
						l.Prepared(next, errors.New("no space left on device"))
						return
					}
					l.Prepared(next, log.Prepare(fs, next, size))
					made.Add(1)
				}()
			},
			OnRefused: func(records []log.Record, err error) {
				errs.add(st.Refused(records, err))
			},
			Switched:   func(g uint64) { errs.add(st.Switch(g)) },
			DB:         7,
			Generation: 1,
			Sequence:   1,
		}))
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Go(func() { errs.add(errors.Skip(st.Run(ctx), context.Canceled)) })
		wg.Go(func() { errs.add(errors.Skip(l.Run(ctx), context.Canceled)) })
		DeferCleanup(func() {
			Expect(l.Close()).To(Succeed())
			Expect(st.Close()).To(Succeed())
			cancel()
			wg.Wait()
			Expect(errs.err()).ToNot(HaveOccurred())
		})

		// Fill log 1 until not even an entry of one record fits.
		key := uint32(1_000_000)
		for _, n := range []int{20_000, 4_000, 800, 160, 32, 8, 1} {
			for {
				var e log.Entry
				for range n {
					e.Add(&log.Intent{Key: key})
					key++
				}
				if _, err := l.Append(&e).Wait(); err != nil {
					Expect(err).To(MatchError(ContainSubstring("log file is full")))
					break
				}
			}
		}
		w := MustSucceed(st.OpenWriter(context.Background(), state.WriterConfig{
			Keys:  []state.WriterKey{{Key: 5, DataType: 4}},
			Clock: 5,
			Start: 100,
		}))
		// The Reserve of this identity reaches a full log, so no value is given out.
		Expect(w.NewSegment(5)).Error().
			To(MatchError(ContainSubstring("log file is full")))
		// Space frees up. The writer asks again, and logs its Create until the log
		// takes it.
		room.Store(true)
		var id uint64
		Eventually(func() error {
			var err error
			id, err = w.NewSegment(5)
			return err
		}).WithTimeout(10 * time.Second).Should(Succeed())
		var e log.Entry
		e.Add(create(id, 5, 5))
		Eventually(func() error {
			_, err := l.Append(&e).Wait()
			return err
		}).Should(Succeed())
		Expect(made.Load()).To(BeNumerically(">=", 1))
		MustSucceed(st.Snapshot())
		Expect(st.Err()).ToNot(HaveOccurred())
	})
})
