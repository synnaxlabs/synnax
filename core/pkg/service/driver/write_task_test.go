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
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// recordingSink is a driver.Sink whose Writer records the values it receives. It drops
// a negative value as a bad command.
type recordingSink struct {
	openErr error
	// writeErr is returned by Write in place of recording the frame.
	writeErr atomic.Pointer[error]
	// health carries the results of Health.
	health chan error
	mu     sync.Mutex
	values []float32
	skips  *driver.Skips
	opens  atomic.Int32
	closes atomic.Int32
}

func (s *recordingSink) Open(
	_ context.Context,
	skips *driver.Skips,
) (driver.Writer, error) {
	s.opens.Add(1)
	if s.openErr != nil {
		return nil, s.openErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skips = skips
	return s, nil
}

func (s *recordingSink) Write(_ context.Context, fr framer.Frame) error {
	if err := s.writeErr.Load(); err != nil {
		return *err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for series := range fr.Series() {
		for _, v := range series.Unmarshal[float32]() {
			s.skips.Read()
			if v < 0 {
				s.skips.Skip(time.Now(), fmt.Sprintf("command %v is negative", v))
				continue
			}
			s.values = append(s.values, v)
		}
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

func (s *recordingSink) Close() error {
	s.closes.Add(1)
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
		createTask(ctx, t)
		sink = &recordingSink{health: make(chan error, 1)}
		cmdCh = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &cmdCh)
		})).To(Succeed())
	})

	open := func(cfgs ...driver.WriteTaskConfig) *driver.WriteTask {
		GinkgoHelper()
		wt := MustSucceed(driver.NewWriteTask(append([]driver.WriteTaskConfig{{
			Channel: channelSvc,
			Sink:    sink,
			DB:      db, Status: statusSvc,
			Framer:   framerSvc,
			Channels: channel.Keys{cmdCh.Key()},
			Task:     t,
		}}, cfgs...)...))
		DeferCleanup(func() { Expect(wt.Stop(false)).To(Succeed()) })
		return wt
	}

	retrieve := func(ctx context.Context) task.Status {
		GinkgoHelper()
		return retrieveStatus[task.StatusDetails](ctx, t.ResourceID())
	}

	// command writes value to the command channel once and waits until check passes.
	command := func(ctx context.Context, value float32, check func(g Gomega)) {
		GinkgoHelper()
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{cmdCh.Key()},
			Start: telem.Now(),
		}))
		defer func() { Expect(w.Close()).To(Succeed()) }()
		Expect(w.Write(frame.NewUnary(cmdCh.Key(), telem.NewSeriesV(value)))).
			To(BeTrue())
		Eventually(check).Should(Succeed())
	}

	Describe("NewWriteTask", func() {
		It("Should reject a configuration with no sink", func() {
			Expect(driver.NewWriteTask(driver.WriteTaskConfig{
				Channel:  channelSvc,
				DB:       db,
				Status:   statusSvc,
				Framer:   framerSvc,
				Channels: channel.Keys{cmdCh.Key()},
				Task:     t,
			})).Error().To(MatchError(ContainSubstring("sink: must be non-nil")))
		})

		It("Should reject a skip hold that is not positive", func() {
			Expect(driver.NewWriteTask(driver.WriteTaskConfig{
				Sink:     sink,
				DB:       db,
				Status:   statusSvc,
				Framer:   framerSvc,
				Channels: channel.Keys{cmdCh.Key()},
				Task:     t,
				SkipHold: -time.Second,
			})).Error().To(MatchError(ContainSubstring("skip_hold: must be positive")))
		})
	})

	Describe("Exec", func() {
		It("Should send streamed frames to the sink between a start and a stop",
			func(ctx SpecContext) {
				wt := open()
				Expect(wt.Exec(ctx, task.Command{Type: "start", Key: "cmd-1"})).
					To(Succeed())
				started := retrieve(ctx)
				Expect(started.Message).To(Equal("Started task"))
				Expect(started.Details.Running).To(BeTrue())
				Expect(started.Details.Cmd).To(Equal("cmd-1"))

				command(ctx, 42, func(g Gomega) {
					g.Expect(sink.received()).To(ContainElement(float32(42)))
				})

				Expect(wt.Exec(ctx, task.Command{Type: "stop", Key: "cmd-2"})).
					To(Succeed())
				stopped := retrieve(ctx)
				Expect(stopped.Message).To(Equal("Stopped task"))
				Expect(stopped.Details.Running).To(BeFalse())
				Expect(stopped.Details.Cmd).To(Equal("cmd-2"))
				Expect(sink.closes.Load()).To(BeEquivalentTo(1))
			},
		)

		It("Should answer the start with the error of a sink that fails to open",
			func(ctx SpecContext) {
				sink.openErr = errors.New("broker refused the connection")
				wt := open()
				Expect(wt.Exec(ctx, task.Command{Type: "start", Key: "cmd-3"})).
					To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to start task"))
				Expect(stat.Description).To(Equal("broker refused the connection"))
				Expect(stat.Details.Cmd).To(Equal("cmd-3"))
				Expect(stat.Details.Running).To(BeFalse())
			},
		)

		It("Should close the writer of the sink when the streamer fails to open",
			func(ctx SpecContext) {
				wt := MustSucceed(driver.NewWriteTask(driver.WriteTaskConfig{
					Channel: channelSvc,
					Sink:    sink,
					DB:      db, Status: statusSvc,
					Framer:   framerSvc,
					Channels: channel.Keys{channel.Key(1<<31 - 1)},
					Task:     t,
				}))
				Expect(wt.Start(ctx, "cmd-4")).To(MatchError(query.ErrNotFound))
				Expect(sink.closes.Load()).To(BeEquivalentTo(1))
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

	Describe("Skipped commands", func() {
		It("Should clear the skip warning when the hold passes with no command",
			func(ctx SpecContext) {
				wt := open(driver.WriteTaskConfig{SkipHold: 200 * time.Millisecond})
				Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
				command(ctx, -1, func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(Equal("Skipped 1 of 1 command"))
					g.Expect(stat.Description).To(Equal("command -1 is negative"))
					g.Expect(stat.Details.Running).To(BeTrue())
				})
				Eventually(func() status.Variant { return retrieve(ctx).Variant }).
					Should(Equal(status.VariantSuccess))
				Expect(sink.received()).To(BeEmpty())
			},
		)

		It("Should keep a write error when the skip hold passes",
			func(ctx SpecContext) {
				hold := 200 * time.Millisecond
				wt := open(driver.WriteTaskConfig{SkipHold: hold})
				Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
				command(ctx, -1, func(g Gomega) {
					g.Expect(retrieve(ctx).Message).To(Equal("Skipped 1 of 1 command"))
				})
				timeout := errors.Mark(
					errors.New("publish timed out"), driver.ErrTemporary,
				)
				sink.writeErr.Store(&timeout)
				command(ctx, 1, func(g Gomega) {
					g.Expect(retrieve(ctx).Description).To(Equal("publish timed out"))
				})
				Consistently(func() string { return retrieve(ctx).Description }).
					WithTimeout(3 * hold).
					Should(Equal("publish timed out"))
			},
		)

		It("Should show a sink error over the skip warning", func(ctx SpecContext) {
			wt := open()
			Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
			command(ctx, -1, func(g Gomega) {
				g.Expect(retrieve(ctx).Message).To(Equal("Skipped 1 of 1 command"))
			})
			sink.health <- errors.Mark(
				errors.New("broker connection lost"), driver.ErrTemporary,
			)
			Eventually(func() string { return retrieve(ctx).Description }).
				Should(Equal("broker connection lost"))
		})
	})

	Describe("Sink errors", func() {
		DescribeTable("Should warn on a write error and recover on the next write",
			func(ctx SpecContext, marker error) {
				wt := open()
				Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
				lost := errors.Mark(errors.New("broker connection lost"), marker)
				sink.writeErr.Store(&lost)
				command(ctx, 1, func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(Equal("Failed to write data"))
					g.Expect(stat.Description).To(Equal("broker connection lost"))
					g.Expect(stat.Details.Running).To(BeTrue())
				})
				sink.writeErr.Store(nil)
				command(ctx, 2, func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
					g.Expect(sink.received()).To(ContainElement(float32(2)))
				})
				Expect(sink.received()).To(Equal([]float32{2}))
			},
			Entry("temporary", driver.ErrTemporary),
			Entry("degraded", driver.ErrDegraded),
		)

		It("Should warn while the sink is not healthy, with no frame to send",
			func(ctx SpecContext) {
				wt := open()
				Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
				sink.health <- errors.Mark(
					errors.New("broker connection lost"), driver.ErrTemporary,
				)
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(Equal("Failed to write data"))
					g.Expect(stat.Description).To(Equal("broker connection lost"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).Should(Succeed())
				sink.health <- nil
				Eventually(func() status.Variant { return retrieve(ctx).Variant }).
					Should(Equal(status.VariantSuccess))
			},
		)

		It("Should keep a health warning through a write that succeeds",
			func(ctx SpecContext) {
				wt := open()
				Expect(wt.Start(ctx, driver.NoCommand)).To(Succeed())
				sink.health <- errors.Mark(
					errors.New("command blocked"), driver.ErrTemporary,
				)
				Eventually(func() string { return retrieve(ctx).Description }).
					Should(Equal("command blocked"))
				command(ctx, 1, func(g Gomega) {
					g.Expect(sink.received()).To(ContainElement(float32(1)))
				})
				Consistently(func() string { return retrieve(ctx).Description }).
					Should(Equal("command blocked"))
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
					g.Expect(stat.Message).To(Equal("Failed to run task"))
					g.Expect(stat.Description).
						To(Equal("broker rejected the credentials"))
					g.Expect(stat.Details.Running).To(BeFalse())
				}).Should(Succeed())
				Eventually(sink.closes.Load).Should(BeEquivalentTo(1))
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
				g.Expect(stat.Message).To(Equal("Failed to run task"))
				g.Expect(stat.Description).To(Equal("payload encoder failed"))
				g.Expect(stat.Details.Running).To(BeFalse())
			})
			Eventually(sink.closes.Load).Should(BeEquivalentTo(1))
		})
	})
})
