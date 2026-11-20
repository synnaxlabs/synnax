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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
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

func sealsOf(st *state.State) func() ([]log.Record, error) {
	return func() ([]log.Record, error) {
		seals, err := st.SealUnheld()
		records := make([]log.Record, len(seals))
		for i := range seals {
			records[i] = &seals[i]
		}
		return records, err
	}
}

var _ = Describe("Check: executor", func() {
	DescribeTable("Should write the head of a switched log when the state owner and "+
		"the checkpointer share an executor", func(oneAtATime bool) {
		fs := xfs.NewMem()
		ex := executor(oneAtATime)
		Expect(log.Prepare(fs, 2, logSize)).To(Succeed())
		st := MustSucceed(state.Open(state.Config{
			Versions: set.New[uint16](1),
			Version:  1,
			Append:   func(*log.Entry) {},
			Executor: ex,
		}, &state.Snapshot{Limits: state.Limits{Segments: 1, Alignments: 1}}))
		cp := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  logSize,
			Snapshot: st.Snapshot,
			Prepared: func(uint64, error) {},
			Executor: ex,
		}))
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		go func() { _ = st.Run(ctx) }()
		go func() { _ = cp.Run(ctx) }()
		cp.Switched(2)
		Eventually(func() error { return readErr(fs, 2) }).
			WithTimeout(5 * time.Second).
			Should(Succeed())
	},
		Entry("that runs each job on a goroutine", false),
		Entry("that runs one job at a time", true),
	)

	DescribeTable("Should switch to the next log when the log owner and the state "+
		"owner share an executor", func(oneAtATime bool) {
		fs := xfs.NewMem()
		ex := executor(oneAtATime)
		Expect(log.Prepare(fs, 1, logSize)).To(Succeed())
		Expect(log.Prepare(fs, 2, logSize)).To(Succeed())
		var l *log.Log
		st := MustSucceed(state.Open(state.Config{
			Versions: set.New[uint16](1),
			Version:  1,
			Append:   func(e *log.Entry) { l.Append(e) },
			Executor: ex,
		}, &state.Snapshot{Limits: state.Limits{Segments: 1, Alignments: 1}}))
		l = MustSucceed(log.Start(log.Config{
			FS:       fs,
			Executor: ex,
			OnDurable: func(seq uint64, records []log.Record) {
				_ = st.Durable(seq, records)
			},
			Prepare: func(next uint64) {
				if next == 2 {
					l.Prepared(next, nil)
				}
			},
			Seals:      sealsOf(st),
			Switched:   func(uint64) {},
			DB:         db,
			Generation: 1,
			Sequence:   1,
		}))
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		go func() { _ = st.Run(ctx) }()
		go func() { _ = l.Run(ctx) }()
		done := make(chan error, 1)
		go func() {
			for i := range uint32(12) {
				var filler log.Entry
				for k := range uint32(10_000) {
					filler.Add(&log.Intent{Key: 1_000_000 + i*10_000 + k})
				}
				if _, err := l.Append(&filler).Wait(); err != nil {
					done <- err
					return
				}
			}
			done <- nil
		}()
		Eventually(done).WithTimeout(5 * time.Second).Should(Receive(BeNil()))
	},
		Entry("that runs each job on a goroutine", false),
		Entry("that runs one job at a time", true),
	)

	DescribeTable("Should switch to the next log when the log owner and the "+
		"checkpointer share an executor", func(oneAtATime bool) {
		fs := xfs.NewMem()
		ex := executor(oneAtATime)
		Expect(log.Prepare(fs, 1, logSize)).To(Succeed())
		var l *log.Log
		cp := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:      fs,
			DB:      db,
			LogSize: logSize,
			Snapshot: func() (*state.Snapshot, error) {
				return &state.Snapshot{}, nil
			},
			Prepared: func(next uint64, err error) { l.Prepared(next, err) },
			Executor: ex,
		}))
		l = MustSucceed(log.Start(log.Config{
			FS:         fs,
			Executor:   ex,
			OnDurable:  func(uint64, []log.Record) {},
			Prepare:    cp.Prepare,
			Seals:      func() ([]log.Record, error) { return nil, nil },
			Switched:   cp.Switched,
			DB:         db,
			Generation: 1,
			Sequence:   1,
		}))
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		go func() { _ = l.Run(ctx) }()
		// One entry past three quarters of the log: the log owner's step reaches the
		// switch before the checkpointer's first step is on the executor.
		var big log.Entry
		for k := range uint32(100_000) {
			big.Add(&log.Intent{Key: k + 1})
		}
		ack := make(chan error, 1)
		go func() {
			_, err := l.Append(&big).Wait()
			ack <- err
		}()
		time.Sleep(200 * time.Millisecond)
		go func() { _ = cp.Run(ctx) }()
		Eventually(ack).WithTimeout(5 * time.Second).Should(Receive(BeNil()))
	},
		Entry("that runs each job on a goroutine", false),
		Entry("that runs one job at a time", true),
	)
})
