// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// recordingSink is a driver.Sink that records the values it receives.
type recordingSink struct {
	startErr error
	// writeErr is returned by Write in place of recording the frame.
	writeErr atomic.Pointer[error]
	// health carries the results of Health.
	health chan error
	mu     sync.Mutex
	values []float32
	starts atomic.Int32
	stops  atomic.Int32
}

func (s *recordingSink) Start(context.Context) error {
	s.starts.Add(1)
	return s.startErr
}

func (s *recordingSink) Write(_ context.Context, fr framer.Frame) error {
	if err := s.writeErr.Load(); err != nil {
		return *err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for series := range fr.Series() {
		s.values = append(s.values, series.Unmarshal[float32]()...)
	}
	return nil
}

func (s *recordingSink) Health(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-s.health:
		return err
	}
}

func (s *recordingSink) Stop() error {
	s.stops.Add(1)
	return nil
}

func (s *recordingSink) received() []float32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]float32(nil), s.values...)
}

var _ = Describe("WriteTask", func() {
	var (
		t     task.Task
		sink  *recordingSink
		cmdCh channel.Channel
	)

	BeforeEach(func(ctx SpecContext) {
		t = task.Task{Key: uuid.New(), Name: "write-task-test", Type: "mock"}
		sink = &recordingSink{health: make(chan error, 1)}
		cmdCh = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, &cmdCh)).To(Succeed())
	})

	open := func() *driver.WriteTask {
		GinkgoHelper()
		wt := MustSucceed(driver.NewWriteTask(driver.WriteTaskConfig{
			Sink:     sink,
			Status:   statusSvc,
			Framer:   framerSvc,
			Channels: channel.Keys{cmdCh.Key()},
			Task:     t,
		}))
		DeferCleanup(func() { Expect(wt.Stop(false)).To(Succeed()) })
		return wt
	}

	retrieve := func(ctx context.Context) task.Status {
		var stat task.Status
		_ = statusSvc.NewRetrieve[task.StatusDetails]().
			Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
			Entry(&stat).Exec(ctx, nil)
		return stat
	}

	// command writes value to the command channel until check passes, because a
	// streamer misses the frames written before it connects.
	command := func(ctx context.Context, value float32, check func(g Gomega)) {
		GinkgoHelper()
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{cmdCh.Key()},
			Start: telem.Now(),
		}))
		defer func() { Expect(w.Close()).To(Succeed()) }()
		Eventually(func(g Gomega) {
			g.Expect(w.Write(frame.NewUnary(cmdCh.Key(), telem.NewSeriesV(value)))).
				To(BeTrue())
			check(g)
		}).Should(Succeed())
	}

	Describe("NewWriteTask", func() {
		It("Should reject a configuration with no sink", func() {
			Expect(driver.NewWriteTask(driver.WriteTaskConfig{
				Status:   statusSvc,
				Framer:   framerSvc,
				Channels: channel.Keys{cmdCh.Key()},
				Task:     t,
			})).Error().To(MatchError(ContainSubstring("sink: must be non-nil")))
		})
	})

	Describe("Exec", func() {
		It("Should send streamed frames to the sink between a start and a stop",
			func(ctx SpecContext) {
				wt := open()
				Expect(wt.Exec(ctx, task.Command{Type: "start", Key: "cmd-1"})).
					To(Succeed())
				started := retrieve(ctx)
				Expect(started.Message).To(Equal("Task started successfully"))
				Expect(started.Details.Running).To(BeTrue())
				Expect(started.Details.Cmd).To(Equal("cmd-1"))

				command(ctx, 42, func(g Gomega) {
					g.Expect(sink.received()).To(ContainElement(float32(42)))
				})

				Expect(wt.Exec(ctx, task.Command{Type: "stop", Key: "cmd-2"})).
					To(Succeed())
				stopped := retrieve(ctx)
				Expect(stopped.Message).To(Equal("Task stopped successfully"))
				Expect(stopped.Details.Running).To(BeFalse())
				Expect(stopped.Details.Cmd).To(Equal("cmd-2"))
				Expect(sink.stops.Load()).To(BeEquivalentTo(1))
			},
		)

		It("Should answer the start with the error of a sink that fails to start",
			func(ctx SpecContext) {
				sink.startErr = errors.New("broker refused the connection")
				wt := open()
				Expect(wt.Exec(ctx, task.Command{Type: "start", Key: "cmd-3"})).
					To(MatchError(ContainSubstring("broker refused the connection")))
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Details.Cmd).To(Equal("cmd-3"))
				Expect(stat.Details.Running).To(BeFalse())
			},
		)

		It("Should stop the sink when the streamer fails to open",
			func(ctx SpecContext) {
				wt := MustSucceed(driver.NewWriteTask(driver.WriteTaskConfig{
					Sink:     sink,
					Status:   statusSvc,
					Framer:   framerSvc,
					Channels: channel.Keys{channel.Key(1<<31 - 1)},
					Task:     t,
				}))
				Expect(wt.Start(ctx, "cmd-4")).To(HaveOccurred())
				Expect(sink.stops.Load()).To(BeEquivalentTo(1))
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Details.Running).To(BeFalse())
			},
		)

		It("Should stop without an error every time", func(ctx SpecContext) {
			wt := open()
			for range 50 {
				Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
				Expect(wt.Stop(false)).To(Succeed())
			}
		})

		It("Should reject a command it does not support", func(ctx SpecContext) {
			Expect(open().Exec(ctx, task.Command{Type: "tare"})).
				To(MatchError(driver.ErrUnsupportedCommand))
		})
	})

	Describe("Sink errors", func() {
		It("Should warn on a temporary error and recover on the next write",
			func(ctx SpecContext) {
				wt := open()
				Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
				lost := errors.Wrap(driver.ErrTemporary, "broker connection lost")
				sink.writeErr.Store(&lost)
				command(ctx, 1, func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).
						To(ContainSubstring("broker connection lost"))
					g.Expect(stat.Details.Running).To(BeTrue())
				})
				sink.writeErr.Store(nil)
				command(ctx, 2, func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
					g.Expect(sink.received()).To(ContainElement(float32(2)))
				})
				// A write of 1 still in flight when the error clears may land, but
				// the dropped ones are never replayed after the recovery.
				received := sink.received()
				Expect(received[slices.Index(received, 2):]).
					ToNot(ContainElement(float32(1)))
			},
		)

		It("Should warn while the sink is not healthy, with no frame to send",
			func(ctx SpecContext) {
				wt := open()
				Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
				lost := errors.Wrap(driver.ErrTemporary, "broker connection lost")
				sink.health <- lost
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).
						To(ContainSubstring("broker connection lost"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).Should(Succeed())
				sink.health <- nil
				Eventually(func() status.Variant { return retrieve(ctx).Variant }).
					Should(Equal(status.VariantSuccess))
			},
		)

		It("Should stop on a health error that is not temporary",
			func(ctx SpecContext) {
				wt := open()
				Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
				sink.health <- errors.New("broker rejected the credentials")
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal("broker rejected the credentials"))
					g.Expect(stat.Details.Running).To(BeFalse())
				}).Should(Succeed())
				Eventually(sink.stops.Load).Should(BeEquivalentTo(1))
			},
		)

		It("Should stop on any other error", func(ctx SpecContext) {
			wt := open()
			Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
			failed := errors.New("payload encoder failed")
			sink.writeErr.Store(&failed)
			command(ctx, 1, func(g Gomega) {
				stat := retrieve(ctx)
				g.Expect(stat.Variant).To(Equal(status.VariantError))
				g.Expect(stat.Message).To(Equal("payload encoder failed"))
				g.Expect(stat.Details.Running).To(BeFalse())
			})
			Eventually(sink.stops.Load).Should(BeEquivalentTo(1))
		})
	})
})
