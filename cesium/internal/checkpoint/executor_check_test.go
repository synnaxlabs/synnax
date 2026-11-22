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
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// serial is an executor that runs one job at a time, in the order of the calls, as the
// harness of a run key does.
type serial struct {
	jobs chan func()
	done chan struct{}
}

func (s *serial) run(f func()) {
	select {
	case s.jobs <- f:
	case <-s.done:
	}
}

// executor returns the executor of a spec: one that runs one job at a time, or one that
// runs each job on a new goroutine, as production does.
func executor(oneAtATime bool) func(func()) {
	if !oneAtATime {
		return goExec
	}
	s := &serial{jobs: make(chan func(), 1024), done: make(chan struct{})}
	go func() {
		for {
			select {
			case f := <-s.jobs:
				f()
			case <-s.done:
				return
			}
		}
	}()
	DeferCleanup(func() { close(s.done) })
	return s.run
}

// ownerErrs holds the errors of owners and their hooks until the cleanup of the spec
// stops them. An error after that point is the stop itself.
type ownerErrs struct {
	mu       sync.Mutex
	errs     []error
	stopping atomic.Bool
}

func (o *ownerErrs) add(err error) {
	if err == nil || o.stopping.Load() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.errs = append(o.errs, err)
}

func (o *ownerErrs) err() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return errors.Join(o.errs...)
}

// runOwners runs each owner until the spec ends, and waits for them all to return at
// its cleanup, before the cleanup of the executor that they share. The cleanup fails
// on an error of errs.
func runOwners(errs *ownerErrs, runs ...func(context.Context) error) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, run := range runs {
		wg.Go(func() { errs.add(errors.Skip(run(ctx), context.Canceled)) })
	}
	DeferCleanup(func() {
		errs.stopping.Store(true)
		cancel()
		wg.Wait()
		Expect(errs.err()).ToNot(HaveOccurred())
	})
}

var _ = Describe("Check: executor", func() {
	DescribeTable("Should write the head of a switched log when the state owner and "+
		"the checkpointer share an executor", func(oneAtATime bool) {
		fs, errs := xfs.NewMem(), &ownerErrs{}
		ex := executor(oneAtATime)
		Expect(log.Prepare(fs, 2, logSize)).To(Succeed())
		var cp *checkpoint.Checkpointer
		st := MustSucceed(state.Open(state.Config{
			Versions:   set.New[uint16](1),
			Version:    1,
			Threshold:  0.2,
			Append:     func(*log.Entry) {},
			Executor:   ex,
			OnHead:     func(g uint64, s *state.Snapshot) { cp.Head(g, s) },
			Generation: 1,
		}, &state.Snapshot{Limits: state.Limits{Segments: 1, Alignments: 1}}))
		cp = MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  logSize,
			Prepared: func(uint64, error) {},
			Executor: ex,
		}))
		runOwners(errs, st.Run, cp.Run)
		Expect(st.Switch(2)).To(Succeed())
		Eventually(func() error { return readErr(fs, 2) }).
			WithTimeout(5 * time.Second).
			Should(Succeed())
	},
		Entry("that runs each job on a goroutine", false),
		Entry("that runs one job at a time", true),
	)

	DescribeTable("Should switch to the next log when the log owner and the state "+
		"owner share an executor", func(oneAtATime bool) {
		fs, errs := xfs.NewMem(), &ownerErrs{}
		ex := executor(oneAtATime)
		Expect(log.Prepare(fs, 1, logSize)).To(Succeed())
		Expect(log.Prepare(fs, 2, logSize)).To(Succeed())
		var (
			l        *log.Log
			switched = make(chan uint64, 1)
		)
		st := MustSucceed(state.Open(state.Config{
			Versions:   set.New[uint16](1),
			Version:    1,
			Threshold:  0.2,
			Append:     func(e *log.Entry) { l.Append(e) },
			Executor:   ex,
			OnHead:     func(uint64, *state.Snapshot) {},
			Generation: 1,
		}, &state.Snapshot{Limits: state.Limits{Segments: 1, Alignments: 1}}))
		l = MustSucceed(log.Start(log.Config{
			FS:       fs,
			Executor: ex,
			OnDurable: func(seq uint64, records []log.Record) {
				errs.add(st.Durable(seq, records))
			},
			Prepare: func(next uint64) {
				if next == 2 {
					l.Prepared(next, nil)
				}
			},
			OnRefused: func(records []log.Record, err error) {
				errs.add(st.Refused(records, err))
			},
			Switched: func(g uint64) {
				errs.add(st.Switch(g))
				switched <- g
			},
			DB:         db,
			Generation: 1,
			Sequence:   1,
		}))
		runOwners(errs, st.Run, l.Run)
		Eventually(fill(l)).WithTimeout(10 * time.Second).Should(Receive(BeNil()))
		Expect(switched).To(Receive(Equal(uint64(2))))
	},
		Entry("that runs each job on a goroutine", false),
		Entry("that runs one job at a time", true),
	)

	DescribeTable("Should switch to the next log when the log owner and the "+
		"checkpointer share an executor", func(oneAtATime bool) {
		fs, errs := xfs.NewMem(), &ownerErrs{}
		ex := executor(oneAtATime)
		Expect(log.Prepare(fs, 1, logSize)).To(Succeed())
		var (
			l        *log.Log
			ready    = make(chan uint64, 1)
			switched = make(chan uint64, 1)
		)
		cp := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:      fs,
			DB:      db,
			LogSize: logSize,
			Prepared: func(next uint64, err error) {
				l.Prepared(next, err)
				if err == nil {
					ready <- next
				}
			},
			Executor: ex,
		}))
		l = MustSucceed(log.Start(log.Config{
			FS:        fs,
			Executor:  ex,
			OnDurable: func(uint64, []log.Record) {},
			Prepare:   cp.Prepare,
			OnRefused: func([]log.Record, error) {},
			Switched: func(g uint64) {
				cp.Head(g, &state.Snapshot{})
				switched <- g
			},
			DB:         db,
			Generation: 1,
			Sequence:   1,
		}))
		// The log passes three quarters before the checkpointer's first step is on the
		// executor.
		runOwners(errs, l.Run)
		Eventually(fill(l)).WithTimeout(10 * time.Second).Should(Receive(BeNil()))
		runOwners(errs, cp.Run)
		Eventually(ready).WithTimeout(10 * time.Second).
			Should(Receive(Equal(uint64(2))))
		var e log.Entry
		e.Add(&log.Intent{Key: 1})
		MustSucceed(l.Append(&e).Wait())
		Eventually(switched).WithTimeout(10 * time.Second).
			Should(Receive(Equal(uint64(2))))
		Eventually(func() error { return readErr(fs, 2) }).
			WithTimeout(5 * time.Second).
			Should(Succeed())
	},
		Entry("that runs each job on a goroutine", false),
		Entry("that runs one job at a time", true),
	)
})
