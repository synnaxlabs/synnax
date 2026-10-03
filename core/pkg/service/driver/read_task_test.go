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
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type readResult struct {
	err   error
	frame framer.Frame
}

// queueSource is a driver.Source that returns the results a spec queues.
type queueSource struct {
	startErr error
	results  chan readResult
	starts   atomic.Int32
	stops    atomic.Int32
}

func newQueueSource() *queueSource {
	return &queueSource{results: make(chan readResult, 10)}
}

func (s *queueSource) Start(context.Context) error {
	s.starts.Add(1)
	return s.startErr
}

func (s *queueSource) Read(ctx context.Context) (framer.Frame, error) {
	select {
	case <-ctx.Done():
		return framer.Frame{}, ctx.Err()
	case r := <-s.results:
		return r.frame, r.err
	}
}

func (s *queueSource) Stop() error {
	s.stops.Add(1)
	return nil
}

var _ = Describe("ReadTask", func() {
	var (
		t      task.Task
		src    *queueSource
		idxCh  channel.Channel
		dataCh channel.Channel
	)

	BeforeEach(func(ctx SpecContext) {
		t = task.Task{Key: uuid.New(), Name: "read-task-test", Type: "mock"}
		src = newQueueSource()
		w := channelSvc.NewWriter(nil)
		idxCh = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(w.Create(ctx, &idxCh)).To(Succeed())
		dataCh = channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			LocalIndex: idxCh.LocalKey,
		}
		Expect(w.Create(ctx, &dataCh)).To(Succeed())
	})

	openWithBreaker := func(brk breaker.Config) *driver.ReadTask {
		GinkgoHelper()
		rt := MustSucceed(driver.NewReadTask(driver.ReadTaskConfig{
			Source: src,
			DB:     db, Status: statusSvc,
			Framer:   framerSvc,
			Channels: channel.Keys{idxCh.Key(), dataCh.Key()},
			Task:     t,
			Breaker:  brk,
		}))
		DeferCleanup(func() { Expect(rt.Stop(false)).To(Succeed()) })
		return rt
	}

	open := func(maxRetries int) *driver.ReadTask {
		GinkgoHelper()
		return openWithBreaker(breaker.Config{
			BaseInterval: time.Millisecond,
			MaxRetries:   maxRetries,
		})
	}

	retrieve := func(ctx context.Context) task.Status {
		var stat task.Status
		_ = statusSvc.NewRetrieve[task.StatusDetails]().
			Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
			Entry(&stat).Exec(ctx, nil)
		return stat
	}

	samples := func(seconds ...telem.TimeStamp) readResult {
		values := make([]float32, len(seconds))
		for i, s := range seconds {
			values[i] = float32(s)
		}
		return readResult{frame: frame.NewMulti(
			[]channel.Key{idxCh.Key(), dataCh.Key()},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(seconds...),
				telem.NewSeriesV(values...),
			},
		)}
	}

	Describe("NewReadTask", func() {
		It("Should reject a configuration with no source", func() {
			Expect(driver.NewReadTask(driver.ReadTaskConfig{
				DB:       db,
				Status:   statusSvc,
				Framer:   framerSvc,
				Channels: channel.Keys{idxCh.Key()},
				Task:     t,
			})).Error().To(MatchError(ContainSubstring("source: must be non-nil")))
		})
		It("Should reject a task with no key", func() {
			Expect(driver.NewReadTask(driver.ReadTaskConfig{
				Source: src,
				DB:     db, Status: statusSvc,
				Framer:   framerSvc,
				Channels: channel.Keys{idxCh.Key()},
			})).Error().To(MatchError(ContainSubstring("task: must have a key")))
		})
		It("Should reject a breaker whose base interval exceeds its maximum", func() {
			Expect(driver.NewReadTask(driver.ReadTaskConfig{
				Source: src,
				DB:     db, Status: statusSvc,
				Framer:   framerSvc,
				Channels: channel.Keys{idxCh.Key()},
				Task:     t,
				Breaker:  breaker.Config{BaseInterval: time.Hour},
			})).Error().To(MatchError(ContainSubstring(
				"max_interval: must be zero or at least base_interval",
			)))
		})
	})

	Describe("Exec", func() {
		It("Should write the frames of the source between a start and a stop",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Exec(ctx, task.Command{Type: "start", Key: "cmd-1"})).
					To(Succeed())
				started := retrieve(ctx)
				Expect(started.Variant).To(Equal(status.VariantSuccess))
				Expect(started.Message).To(Equal("Task started successfully"))
				Expect(started.Details.Running).To(BeTrue())
				Expect(started.Details.Cmd).To(Equal("cmd-1"))

				src.results <- samples(1, 2)
				src.results <- readResult{}
				src.results <- samples(3)
				Eventually(src.results).Should(BeEmpty())

				Expect(rt.Exec(ctx, task.Command{Type: "stop", Key: "cmd-2"})).
					To(Succeed())
				stopped := retrieve(ctx)
				Expect(stopped.Message).To(Equal("Task stopped successfully"))
				Expect(stopped.Details.Running).To(BeFalse())
				Expect(stopped.Details.Cmd).To(Equal("cmd-2"))
				Expect(src.stops.Load()).To(BeEquivalentTo(1))

				iter := MustOpen(framerSvc.OpenIterator(ctx, framer.IteratorConfig{
					Keys:   []channel.Key{dataCh.Key()},
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
				Expect(iter.Value().Get(dataCh.Key()).Series[0]).
					To(telem.MatchWrittenSeries(telem.NewSeriesV[float32](1, 2, 3)))
			},
		)

		It("Should answer a start for a running task without a second source start",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				Expect(rt.Exec(ctx, task.Command{Type: "start", Key: "cmd-3"})).
					To(Succeed())
				Expect(src.starts.Load()).To(BeEquivalentTo(1))
				stat := retrieve(ctx)
				Expect(stat.Details.Cmd).To(Equal("cmd-3"))
				Expect(stat.Details.Running).To(BeTrue())
			},
		)

		It("Should answer a stop for a stopped task", func(ctx SpecContext) {
			rt := open(breaker.InfiniteRetries)
			Expect(rt.Exec(ctx, task.Command{Type: "stop", Key: "cmd-4"})).
				To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Details.Cmd).To(Equal("cmd-4"))
			Expect(stat.Details.Running).To(BeFalse())
			Expect(src.stops.Load()).To(BeZero())
		})

		It("Should answer the start with the error of a source that fails to start",
			func(ctx SpecContext) {
				src.startErr = errors.New("broker refused the connection")
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Exec(ctx, task.Command{Type: "start", Key: "cmd-5"})).
					To(MatchError(ContainSubstring("broker refused the connection")))
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("broker refused the connection"))
				Expect(stat.Details.Cmd).To(Equal("cmd-5"))
				Expect(stat.Details.Running).To(BeFalse())
			},
		)

		It("Should reject a command it does not support", func(ctx SpecContext) {
			rt := open(breaker.InfiniteRetries)
			Expect(rt.Exec(ctx, task.Command{Type: "tare"})).
				To(MatchError(driver.ErrUnsupportedCommand))
		})
	})

	Describe("Stop", func() {
		It("Should write no status when told not to", func(ctx SpecContext) {
			rt := open(breaker.InfiniteRetries)
			Expect(rt.Start(ctx, "cmd-6")).To(Succeed())
			Expect(rt.Stop(false)).To(Succeed())
			Expect(src.stops.Load()).To(BeEquivalentTo(1))
			stat := retrieve(ctx)
			Expect(stat.Message).To(Equal("Task started successfully"))
		})
	})

	Describe("Source errors", func() {
		It("Should warn on a temporary error and recover on the next read",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				src.results <- readResult{
					err: errors.Wrap(driver.ErrTemporary, "broker connection lost"),
				}
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).
						To(ContainSubstring("broker connection lost"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).Should(Succeed())
				src.results <- readResult{}
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
					g.Expect(stat.Message).To(Equal("Task started successfully"))
				}).Should(Succeed())
			},
		)

		It("Should warn on a degraded read and still write its frame",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				degraded := samples(1)
				degraded.err = errors.Wrap(driver.ErrDegraded, "dropped 4 messages")
				src.results <- degraded
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(ContainSubstring("dropped 4 messages"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).Should(Succeed())
				src.results <- samples(2)
				Eventually(func(g Gomega) {
					g.Expect(retrieve(ctx).Variant).To(Equal(status.VariantSuccess))
				}).Should(Succeed())
				Expect(rt.Stop(false)).To(Succeed())
				iter := MustOpen(framerSvc.OpenIterator(ctx, framer.IteratorConfig{
					Keys:   []channel.Key{dataCh.Key()},
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
				Expect(iter.Value().Get(dataCh.Key()).Series[0]).
					To(telem.MatchWrittenSeries(telem.NewSeriesV[float32](1, 2)))
			},
		)

		It("Should stop without an error while it waits out a temporary error",
			func(ctx SpecContext) {
				rt := openWithBreaker(breaker.Config{
					BaseInterval: time.Hour,
					MaxInterval:  time.Hour,
					MaxRetries:   breaker.InfiniteRetries,
				})
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				src.results <- readResult{
					err: errors.Wrap(driver.ErrTemporary, "broker connection lost"),
				}
				Eventually(func() status.Variant { return retrieve(ctx).Variant }).
					Should(Equal(status.VariantWarning))
				Expect(rt.Stop(true)).To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Details.Running).To(BeFalse())
			},
		)

		It("Should stop when temporary errors exceed the retry limit",
			func(ctx SpecContext) {
				rt := open(1)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				lost := errors.Wrap(driver.ErrTemporary, "broker connection lost")
				src.results <- readResult{err: lost}
				src.results <- readResult{err: lost}
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Details.Running).To(BeFalse())
				}).Should(Succeed())
				Expect(src.stops.Load()).To(BeEquivalentTo(1))
			},
		)

		It("Should stop on any other error and start again on command",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				src.results <- readResult{err: errors.New("payload decoder failed")}
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal("payload decoder failed"))
					g.Expect(stat.Details.Running).To(BeFalse())
					g.Expect(stat.Details.Cmd).To(Equal(driver.NoCommand))
				}).Should(Succeed())
				Eventually(src.stops.Load).Should(BeEquivalentTo(1))

				Expect(rt.Exec(ctx, task.Command{Type: "start", Key: "cmd-7"})).
					To(Succeed())
				Expect(src.starts.Load()).To(BeEquivalentTo(2))
				Expect(retrieve(ctx).Details.Running).To(BeTrue())
			},
		)

		It("Should stop when a frame fails to write", func(ctx SpecContext) {
			rt := open(breaker.InfiniteRetries)
			Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
			// A frame that writes the index without its data channel is invalid.
			src.results <- readResult{frame: frame.NewUnary(
				idxCh.Key(), telem.NewSeriesSecondsTSV(1),
			)}
			Eventually(func(g Gomega) {
				stat := retrieve(ctx)
				g.Expect(stat.Variant).To(Equal(status.VariantError))
				g.Expect(stat.Details.Running).To(BeFalse())
			}).Should(Succeed())
		})
	})
})
