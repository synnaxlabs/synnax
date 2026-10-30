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
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	distmock "github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	servicemock "github.com/synnaxlabs/synnax/pkg/service/mock"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type readResult struct {
	err           error
	frame         framer.Frame
	heldUntilStop bool
}

// queueSource is a driver.Source whose Reader returns the results a spec queues.
type queueSource struct {
	openErr error
	results chan readResult
	opens   atomic.Int32
	closes  atomic.Int32
}

func newQueueSource() *queueSource {
	return &queueSource{results: make(chan readResult, 10)}
}

func (s *queueSource) Open(context.Context) (driver.Reader, error) {
	s.opens.Add(1)
	if s.openErr != nil {
		return nil, s.openErr
	}
	return s, nil
}

func (s *queueSource) Read(ctx context.Context) (framer.Frame, error) {
	select {
	case <-ctx.Done():
		return framer.Frame{}, ctx.Err()
	case r := <-s.results:
		if r.heldUntilStop {
			<-ctx.Done()
		}
		return r.frame, r.err
	}
}

func (s *queueSource) Close() error {
	s.closes.Add(1)
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
			Indexes:  channel.Keys{idxCh.Key()},
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
		GinkgoHelper()
		return retrieveStatus[task.StatusDetails](ctx, t.OntologyID().String())
	}

	// name formats ch as an error message names it.
	name := func(ch channel.Channel) string {
		return fmt.Sprintf("[%s]<%d>", ch.Name, ch.Key())
	}

	// written returns every sample committed to ch.
	written := func(
		ctx context.Context,
		fs *framer.Service,
		ch channel.Channel,
	) telem.Series {
		GinkgoHelper()
		iter := MustSucceed(fs.OpenIterator(ctx, framer.IteratorConfig{
			Keys:   []channel.Key{ch.Key()},
			Bounds: telem.TimeRangeMax,
		}))
		defer func() { Expect(iter.Close()).To(Succeed()) }()
		Expect(iter.SeekFirst()).To(BeTrue())
		all := telem.Series{DataType: ch.DataType}
		for iter.Next(iterator.AutoSpan) {
			all.Data = append(all.Data, iter.Value().Get(ch.Key()).Data()...)
		}
		return all
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
		It("Should reject an index that is not one of its channels", func() {
			Expect(driver.NewReadTask(driver.ReadTaskConfig{
				Source:   src,
				DB:       db,
				Status:   statusSvc,
				Framer:   framerSvc,
				Channels: channel.Keys{dataCh.Key()},
				Indexes:  channel.Keys{idxCh.Key()},
				Task:     t,
			})).Error().To(MatchError(ContainSubstring(
				fmt.Sprintf("indexes: %d is not in channels", idxCh.Key()),
			)))
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
				Expect(started.Message).To(Equal("Started task"))
				Expect(started.Details.Running).To(BeTrue())
				Expect(started.Details.Cmd).To(Equal("cmd-1"))

				src.results <- samples(1, 2)
				src.results <- readResult{}
				src.results <- samples(3)
				Eventually(src.results).Should(BeEmpty())

				Expect(rt.Exec(ctx, task.Command{Type: "stop", Key: "cmd-2"})).
					To(Succeed())
				stopped := retrieve(ctx)
				Expect(stopped.Message).To(Equal("Stopped task"))
				Expect(stopped.Details.Running).To(BeFalse())
				Expect(stopped.Details.Cmd).To(Equal("cmd-2"))
				Expect(src.closes.Load()).To(BeEquivalentTo(1))
				Expect(written(ctx, framerSvc, dataCh)).
					To(telem.MatchWrittenSeries(telem.NewSeriesV[float32](1, 2, 3)))
			},
		)

		It("Should write a frame that the source returns during a stop",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				held := samples(1, 2)
				held.heldUntilStop = true
				src.results <- held
				Eventually(src.results).Should(BeEmpty())
				Expect(rt.Stop(false)).To(Succeed())
				Expect(written(ctx, framerSvc, dataCh)).
					To(telem.MatchWrittenSeries(telem.NewSeriesV[float32](1, 2)))
			},
		)

		It("Should write a frame that another node leads during a stop",
			func(ctx SpecContext) {
				cluster := distmock.NewCluster(ctx, 0)
				local, peer := cluster.Provision(ctx), cluster.Provision(ctx)
				svc := MustOpen(servicemock.OpenLayer(ctx, local))
				w := svc.Channel.NewWriter(nil)
				peerIdx := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Leaseholder: peer.Cluster.HostKey(),
				}
				Expect(w.Create(ctx, &peerIdx)).To(Succeed())
				peerData := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Float32T,
					LocalIndex:  peerIdx.LocalKey,
					Leaseholder: peer.Cluster.HostKey(),
				}
				Expect(w.Create(ctx, &peerData)).To(Succeed())
				keys := channel.Keys{peerIdx.Key(), peerData.Key()}
				rt := MustSucceed(driver.NewReadTask(driver.ReadTaskConfig{
					Source:   src,
					DB:       local.DB,
					Status:   svc.Status,
					Framer:   svc.Framer,
					Channels: keys,
					Indexes:  channel.Keys{peerIdx.Key()},
					Task:     t,
				}))
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				src.results <- readResult{
					frame: frame.NewMulti(keys, []telem.Series{
						telem.NewSeriesSecondsTSV(1, 2),
						telem.NewSeriesV[float32](1, 2),
					}),
					heldUntilStop: true,
				}
				Eventually(src.results).Should(BeEmpty())
				Expect(rt.Stop(false)).To(Succeed())
				Expect(written(ctx, svc.Framer, peerData)).
					To(telem.MatchWrittenSeries(telem.NewSeriesV[float32](1, 2)))
			},
		)

		It("Should write a first frame whose second index starts before its first",
			func(ctx SpecContext) {
				w := channelSvc.NewWriter(nil)
				earlyIdx := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.TimestampT,
					IsIndex:  true,
				}
				Expect(w.Create(ctx, &earlyIdx)).To(Succeed())
				earlyData := channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					LocalIndex: earlyIdx.LocalKey,
				}
				Expect(w.Create(ctx, &earlyData)).To(Succeed())
				keys := channel.Keys{
					idxCh.Key(), dataCh.Key(), earlyIdx.Key(), earlyData.Key(),
				}
				rt := MustSucceed(driver.NewReadTask(driver.ReadTaskConfig{
					Source:   src,
					DB:       db,
					Status:   statusSvc,
					Framer:   framerSvc,
					Channels: keys,
					Indexes:  channel.Keys{idxCh.Key(), earlyIdx.Key()},
					Task:     t,
				}))
				DeferCleanup(func() { Expect(rt.Stop(false)).To(Succeed()) })
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				src.results <- readResult{frame: frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(5, 6),
					telem.NewSeriesV[float32](5, 6),
					telem.NewSeriesSecondsTSV(1, 2),
					telem.NewSeriesV[float32](1, 2),
				})}
				Eventually(src.results).Should(BeEmpty())
				Expect(rt.Stop(false)).To(Succeed())
				Expect(written(ctx, framerSvc, dataCh)).
					To(telem.MatchWrittenSeries(telem.NewSeriesV[float32](5, 6)))
				Expect(written(ctx, framerSvc, earlyData)).
					To(telem.MatchWrittenSeries(telem.NewSeriesV[float32](1, 2)))
			},
		)

		It("Should start the writer at an index rather than at a timestamp value",
			func(ctx SpecContext) {
				w := channelSvc.NewWriter(nil)
				stampCh := channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.TimestampT,
					LocalIndex: idxCh.LocalKey,
				}
				Expect(w.Create(ctx, &stampCh)).To(Succeed())
				keys := channel.Keys{idxCh.Key(), dataCh.Key(), stampCh.Key()}
				rt := MustSucceed(driver.NewReadTask(driver.ReadTaskConfig{
					Source:   src,
					DB:       db,
					Status:   statusSvc,
					Framer:   framerSvc,
					Channels: keys,
					Indexes:  channel.Keys{idxCh.Key()},
					Task:     t,
				}))
				DeferCleanup(func() { Expect(rt.Stop(false)).To(Succeed()) })
				run := func(seconds, stamp telem.TimeStamp) {
					GinkgoHelper()
					Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
					src.results <- readResult{frame: frame.NewMulti(keys, []telem.Series{
						telem.NewSeriesSecondsTSV(seconds, seconds+1),
						telem.NewSeriesV(float32(seconds), float32(seconds+1)),
						telem.NewSeriesSecondsTSV(stamp, stamp),
					})}
					Eventually(src.results).Should(BeEmpty())
					Expect(rt.Stop(false)).To(Succeed())
				}
				run(10, 10)
				// An old timestamp value must not start the writer over the first run.
				run(20, 1)
				Expect(written(ctx, framerSvc, dataCh)).
					To(telem.MatchWrittenSeries(telem.NewSeriesV[float32](10, 11, 20, 21)))
			},
		)

		It("Should answer a start for a running task without a second source open",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				Expect(rt.Exec(ctx, task.Command{Type: "start", Key: "cmd-3"})).
					To(Succeed())
				Expect(src.opens.Load()).To(BeEquivalentTo(1))
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
			Expect(src.closes.Load()).To(BeZero())
		})

		It("Should answer the start with the error of a source that fails to open",
			func(ctx SpecContext) {
				src.openErr = errors.New("broker refused the connection")
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Exec(ctx, task.Command{Type: "start", Key: "cmd-5"})).
					To(MatchError("broker refused the connection"))
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to start task"))
				Expect(stat.Description).To(Equal("broker refused the connection"))
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
			Expect(src.closes.Load()).To(BeEquivalentTo(1))
			stat := retrieve(ctx)
			Expect(stat.Message).To(Equal("Started task"))
		})
	})

	Describe("Source errors", func() {
		It("Should warn on a temporary error and recover on the next read",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				src.results <- readResult{
					err: errors.Mark(
						errors.New("broker connection lost"), driver.ErrTemporary,
					),
				}
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(Equal("Failed to read data"))
					g.Expect(stat.Description).To(Equal("broker connection lost"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).Should(Succeed())
				src.results <- readResult{}
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
					g.Expect(stat.Message).To(Equal("Started task"))
				}).Should(Succeed())
			},
		)

		It("Should warn on a degraded read and still write its frame",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				degraded := samples(1)
				degraded.err = errors.Mark(
					errors.New("dropped 4 messages"), driver.ErrDegraded,
				)
				src.results <- degraded
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(Equal("Failed to read some data"))
					g.Expect(stat.Description).To(Equal("dropped 4 messages"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).Should(Succeed())
				src.results <- samples(2)
				Eventually(func(g Gomega) {
					g.Expect(retrieve(ctx).Variant).To(Equal(status.VariantSuccess))
				}).Should(Succeed())
				Expect(rt.Stop(false)).To(Succeed())
				Expect(written(ctx, framerSvc, dataCh)).
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
					err: errors.Mark(
						errors.New("broker connection lost"), driver.ErrTemporary,
					),
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
				lost := errors.Mark(
					errors.New("broker connection lost"), driver.ErrTemporary,
				)
				src.results <- readResult{err: lost}
				src.results <- readResult{err: lost}
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal("Failed to run task"))
					g.Expect(stat.Description).To(Equal("broker connection lost"))
					g.Expect(stat.Details.Running).To(BeFalse())
				}).Should(Succeed())
				Expect(src.closes.Load()).To(BeEquivalentTo(1))
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
					g.Expect(stat.Message).To(Equal("Failed to run task"))
					g.Expect(stat.Description).To(Equal("payload decoder failed"))
					g.Expect(stat.Details.Running).To(BeFalse())
					g.Expect(stat.Details.Cmd).To(Equal(driver.NoCommand))
				}).Should(Succeed())
				Eventually(src.closes.Load).Should(BeEquivalentTo(1))

				Expect(rt.Exec(ctx, task.Command{Type: "start", Key: "cmd-7"})).
					To(Succeed())
				Expect(src.opens.Load()).To(BeEquivalentTo(2))
				Expect(retrieve(ctx).Details.Running).To(BeTrue())
			},
		)

		It("Should stop while the source waits when the Core rejects a frame",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				// A frame that writes the index without its data channel is invalid.
				src.results <- readResult{frame: frame.NewUnary(
					idxCh.Key(), telem.NewSeriesSecondsTSV(1),
				)}
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal("Failed to run task"))
					g.Expect(stat.Description).To(Equal(fmt.Sprintf(
						"frame must have exactly one series for each data channel "+
							"associated with index %s, but is missing a series for "+
							"channel %s: validation error",
						name(idxCh), name(dataCh),
					)))
					g.Expect(stat.Details.Running).To(BeFalse())
				}).Should(Succeed())
				Eventually(src.closes.Load).Should(BeEquivalentTo(1))
			},
		)

		It("Should stop while the source waits when a frame has the wrong data type",
			func(ctx SpecContext) {
				rt := open(breaker.InfiniteRetries)
				Expect(rt.Start(ctx, driver.NoCommand)).To(Succeed())
				src.results <- readResult{frame: frame.NewMulti(
					[]channel.Key{idxCh.Key(), dataCh.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1),
						telem.NewSeriesV[int32](1),
					},
				)}
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal("Failed to run task"))
					g.Expect(stat.Description).To(Equal(fmt.Sprintf(
						"channel %s: expected data type float32, got int32: "+
							"validation error",
						name(dataCh),
					)))
					g.Expect(stat.Details.Running).To(BeFalse())
				}).Should(Succeed())
				Eventually(src.closes.Load).Should(BeEquivalentTo(1))
			},
		)
	})
})
