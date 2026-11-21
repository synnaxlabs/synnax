// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Happy Path", Ordered, func() {
		scenarios := []func(context.Context) scenario{
			storageScenario,
			freeWriterScenario,
		}
		for i, sF := range scenarios {
			var s scenario
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				s = DeferClose(sF(ctx))
			})
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func(ctx SpecContext) {
				writer := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				Expect(writer.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				))).To(BeTrue())
				MustCommit(writer)
				Expect(writer.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				))).To(BeTrue())
				MustCommit(writer)
			})
		}
	})

	Describe("Variable Channels", Ordered, func() {
		var (
			s     scenario
			idxCh channel.Channel
			strCh channel.Channel
		)
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			node := testutil.OpenNode(GinkgoT().Context())
			idxCh = channel.Channel{
				Name:     "variable_time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			}
			idxCh = MustSucceed(node.CreateChannels(ctx, []channel.Channel{idxCh}))[0]
			strCh = channel.Channel{
				Name:       "variable_str",
				DataType:   telem.StringT,
				LocalIndex: idxCh.LocalKey,
			}
			strCh = MustSucceed(node.CreateChannels(ctx, []channel.Channel{strCh}))[0]
			s = DeferClose(scenario{
				dist:   node,
				closer: node,
				name:   "Variable",
				keys:   []channel.Key{idxCh.Key(), strCh.Key()},
			})
		})
		It("Should write and read persisted string data", func(ctx SpecContext) {
			w := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  s.keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(w.Write(frame.NewMulti(
				s.keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesV("hello", "world", "foo"),
				},
			))).To(BeTrue())
			MustCommit(w)
			iter := MustOpen(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{strCh.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			Expect(
				iter.Value().SeriesAt(0).Unmarshal[string](),
			).To(Equal([]string{"hello", "world", "foo"}))
		})
		It("Should write mixed fixed and variable channels", func(ctx SpecContext) {
			floatCh := channel.Channel{
				Name:       "variable_float",
				DataType:   telem.Float64T,
				LocalIndex: idxCh.LocalKey,
			}
			floatCh = MustSucceed(
				s.dist.CreateChannels(ctx, []channel.Channel{floatCh}),
			)[0]
			keys := []channel.Key{idxCh.Key(), floatCh.Key(), strCh.Key()}
			w := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 20 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(w.Write(frame.NewMulti(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(20, 21, 22),
					telem.NewSeriesV(1.1, 2.2, 3.3),
					telem.NewSeriesV("a", "b", "c"),
				},
			))).To(BeTrue())
			MustCommit(w)
			iter := MustOpen(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{strCh.Key()},
				Bounds: (20 * telem.SecondTS).Range(23 * telem.SecondTS),
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			Expect(
				iter.Value().SeriesAt(0).Unmarshal[string](),
			).To(Equal([]string{"a", "b", "c"}))
		})
	})

	Describe("Canceled Context", Ordered, func() {
		var s scenario
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			s = DeferClose(storageScenario(ctx))
		})
		DescribeTable("Should return ErrClosed when the pipeline ends first",
			func(ctx SpecContext, exec func(*writer.Writer, frame.Frame)) {
				cfg := writer.Config{
					Keys:              s.keys,
					Start:             10 * telem.SecondTS,
					Authorities:       []control.Authority{control.AuthorityAbsolute},
					ErrOnUnauthorized: new(true),
					Sync:              new(true),
				}
				cCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				w := MustSucceed(s.dist.Framer.OpenWriter(cCtx, cfg))
				cancel()
				Eventually(func() error {
					next, err := s.dist.Framer.OpenWriter(ctx, cfg)
					if err != nil {
						return err
					}
					return next.Close()
				}).Should(Succeed())
				exec(w, frame.NewMulti(s.keys, []telem.Series{
					telem.NewSeriesV[int64](1),
					telem.NewSeriesV[int64](2),
					telem.NewSeriesV[int64](3),
				}))
				Expect(w.Close()).To(Succeed())
			},
			Entry("Write", func(w *writer.Writer, fr frame.Frame) {
				GinkgoHelper()
				Expect(w.Write(fr)).Error().To(MatchError(writer.ErrClosed))
			}),
			Entry("Commit", func(w *writer.Writer, _ frame.Frame) {
				GinkgoHelper()
				Expect(w.Commit()).Error().To(MatchError(writer.ErrClosed))
			}),
		)
	})

	Describe("Open Errors", Ordered, func() {
		var s scenario
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			s = DeferClose(storageScenario(ctx))
		})
		It("Should return an error if no keys are provided", func(ctx SpecContext) {
			Expect(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  []channel.Key{},
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			})).Error().To(MatchError(ContainSubstring("keys: must be non-empty")))
		})
		It(
			"Should return an error if a provided key has no storage channel",
			func(ctx SpecContext) {
				Expect(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  []channel.Key{channel.NewKey(1, 22), s.keys[0]},
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				})).Error().To(SatisfyAll(
					MatchError(query.ErrNotFound),
					MatchError(ContainSubstring("not found")),
				))
			},
		)
	})

	Describe("Requests", func() {
		It("Should reject an unknown command", func(ctx SpecContext) {
			host := testutil.NewNode(ctx)
			keys := channel.KeysFromChannels(
				MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:     "unknown_time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				}})),
			)
			w := MustSucceed(host.Framer.NewStreamWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			in, out := confluence.Attach(w, 1)
			w.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			in.Inlet() <- writer.Request{Command: writer.Command(9)}
			var res writer.Response
			Eventually(out.Outlet()).Should(Receive(&res))
			Expect(res.Err).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("invalid writer command")),
			))
			in.Close()
			Expect(sCtx.Wait()).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Write Group Propagation", func() {
		DescribeTable(
			"Should propagate the writer's group to the streamer response",
			func(ctx SpecContext, open func(context.Context) scenario) {
				s := DeferClose(open(ctx))
				streamer := MustSucceed(s.dist.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        s.keys,
					SendOpenAck: new(true),
				}))
				_, out := confluence.Attach(streamer, 10)
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				streamer.Flow(sCtx)
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				w := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:           s.keys,
					Start:          10 * telem.SecondTS,
					Sync:           new(true),
					ControlSubject: control.Subject{Group: 42},
				}))
				Expect(w.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(res.Group).To(Equal(uint32(42)))
			},
			Entry("Storage", storageScenario),
			Entry("Free", freeWriterScenario),
		)
		DescribeTable(
			"Should set group to zero when the writer has no group",
			func(ctx SpecContext, open func(context.Context) scenario) {
				s := DeferClose(open(ctx))
				streamer := MustSucceed(s.dist.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        s.keys,
					SendOpenAck: new(true),
				}))
				_, out := confluence.Attach(streamer, 10)
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				streamer.Flow(sCtx)
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				w := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				Expect(w.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(res.Group).To(Equal(uint32(0)))
			},
			Entry("Storage", storageScenario),
			Entry("Free", freeWriterScenario),
		)
	})

	Describe("Free Write Group Isolation", func() {
		It(
			"Should propagate distinct groups from different writers",
			func(ctx SpecContext) {
				s := DeferClose(freeWriterScenario(ctx))
				streamer := MustSucceed(s.dist.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        s.keys,
					SendOpenAck: new(true),
				}))
				_, out := confluence.Attach(streamer, 10)
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				streamer.Flow(sCtx)
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				w1 := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:           s.keys,
					Start:          10 * telem.SecondTS,
					Sync:           new(true),
					ControlSubject: control.Subject{Group: 10},
				}))
				w2 := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:           s.keys,
					Start:          10 * telem.SecondTS,
					Sync:           new(true),
					ControlSubject: control.Subject{Group: 20},
				}))
				Expect(w1.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(res.Group).To(Equal(uint32(10)))
				Expect(w2.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](8, 9, 10),
						telem.NewSeriesV[int64](11, 12, 13),
						telem.NewSeriesV[int64](14, 15, 16),
					},
				))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(res.Group).To(Equal(uint32(20)))
			},
		)
	})

	Describe("End Domain", func() {
		It(
			"Should end the domain of an index",
			func(ctx SpecContext) {
				host := testutil.NewNode(ctx)
				idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:     "local_time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				}}))[0]
				data := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:       "local",
					DataType:   telem.Int64T,
					LocalIndex: idx.LocalKey,
				}}))[0]
				keys := channel.Keys{idx.Key(), data.Key()}
				frameAt := func(seconds ...telem.TimeStamp) frame.Frame {
					stamps := make([]telem.TimeStamp, len(seconds))
					values := make([]int64, len(seconds))
					for i, s := range seconds {
						stamps[i] = s * telem.SecondTS
						values[i] = int64(s)
					}
					return frame.NewMulti(keys, []telem.Series{
						telem.NewSeries(stamps),
						telem.NewSeries(values),
					})
				}
				w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				Expect(w.Write(frameAt(10, 11))).To(BeTrue())
				MustCommit(w)
				Expect(w.EndDomain(channel.Keys{idx.Key()})).To(BeTrue())
				Expect(w.Write(frameAt(20, 21))).To(BeTrue())
				MustCommit(w)
				fill := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 12 * telem.SecondTS,
					Mode:  cesium.WriterModePersistOnly,
					Sync:  new(true),
				}))
				Expect(fill.Write(frameAt(12, 13))).To(BeTrue())
				MustCommit(fill)
				Expect(fill.Close()).To(Succeed())
				Expect(w.Close()).To(Succeed())
			},
		)
		It(
			"Should return false when another writer holds control",
			func(ctx SpecContext) {
				host := testutil.NewNode(ctx)
				keys := channel.KeysFromChannels(
					MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
						Name:     "host_time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					}})),
				)
				w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
					ControlSubject: control.Subject{Key: "low"},
					Keys:           keys,
					Start:          10 * telem.SecondTS,
					Authorities:    []control.Authority{1},
					Sync:           new(true),
				}))
				holder := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
					ControlSubject: control.Subject{Key: "high"},
					Keys:           keys,
					Start:          10 * telem.SecondTS,
					Authorities:    []control.Authority{control.AuthorityAbsolute},
					Sync:           new(true),
				}))
				Expect(holder.Write(frame.NewMulti(
					keys,
					[]telem.Series{telem.NewSeriesSecondsTSV(10)},
				))).To(BeTrue())
				Expect(w.EndDomain(keys)).To(BeFalse())
				Expect(holder.Close()).To(Succeed())
				Expect(w.Close()).To(Succeed())
			},
		)
		It("Should return no end from a commit that fails", func(ctx SpecContext) {
			gw := testutil.NewNode(ctx)
			// Commits walk a map of indexes, so most commit before the one that fails.
			channels := make([]channel.Channel, 8)
			series := make([]telem.Series, len(channels))
			for i := range channels {
				channels[i] = channel.Channel{
					Name:        fmt.Sprintf("time_%d", i),
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Leaseholder: 1,
				}
				series[i] = telem.NewSeriesSecondsTSV(1, 30)
			}
			keys := channel.KeysFromChannels(
				MustSucceed(gw.CreateChannels(ctx, channels)),
			)
			live := MustSucceed(gw.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys[:1],
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			fill := MustSucceed(gw.Framer.OpenWriter(ctx, writer.Config{
				Keys:             keys,
				Start:            1 * telem.SecondTS,
				Mode:             cesium.WriterModePersistOnly,
				EnableAutoCommit: new(false),
				Sync:             new(true),
			}))
			Expect(fill.Write(frame.NewMulti(keys, series))).To(BeTrue())
			Expect(fill.Commit()).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("cannot write to time range")),
			))
			Expect(fill.Close()).To(MatchError(validate.ErrValidation))
			Expect(live.Close()).To(Succeed())
		})
		It(
			"Should report a commit without control",
			func(ctx SpecContext) {
				host := testutil.NewNode(ctx)
				keys := channel.KeysFromChannels(
					MustSucceed(host.CreateChannels(ctx, []channel.Channel{
						{Name: "held_time", IsIndex: true, DataType: telem.TimestampT},
						{Name: "free_time", IsIndex: true, DataType: telem.TimestampT},
					})),
				)
				frameAt := func(second telem.TimeStamp) frame.Frame {
					return frame.NewMulti(keys, []telem.Series{
						telem.NewSeriesSecondsTSV(second),
						telem.NewSeriesSecondsTSV(second),
					})
				}
				w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
					ControlSubject:   control.Subject{Key: "low"},
					Keys:             keys,
					Start:            10 * telem.SecondTS,
					Authorities:      []control.Authority{1},
					EnableAutoCommit: new(false),
					Sync:             new(true),
				}))
				Expect(w.Write(frameAt(10))).To(BeTrue())
				holder := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
					ControlSubject: control.Subject{Key: "high"},
					Keys:           channel.Keys{keys[0]},
					Start:          10 * telem.SecondTS,
					Authorities:    []control.Authority{control.AuthorityAbsolute},
					Sync:           new(true),
				}))
				end, authorized := MustSucceed2(w.Commit())
				Expect(authorized).To(BeFalse())
				Expect(end).To(BeZero())
				Expect(holder.Close()).To(Succeed())
				Expect(w.Write(frameAt(11))).To(BeTrue())
				Expect(MustCommit(w)).To(Equal(11*telem.SecondTS + 1))
				Expect(w.Close()).To(Succeed())
			},
		)
		It("Should report a commit that fails", func(ctx SpecContext) {
			host := testutil.NewNode(ctx)
			idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
				Name:        "time",
				IsIndex:     true,
				DataType:    telem.TimestampT,
				Leaseholder: 1,
			}}))[0]
			keys := channel.Keys{idx.Key()}
			live := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			fill := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
				Keys:             keys,
				Start:            1 * telem.SecondTS,
				Mode:             cesium.WriterModePersistOnly,
				EnableAutoCommit: new(false),
				Sync:             new(true),
			}))
			Expect(fill.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(1, 30),
			}))).To(BeTrue())
			Expect(fill.EndDomain(keys)).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("cannot write to time range")),
			))
			Expect(fill.Close()).To(MatchError(validate.ErrValidation))
			Expect(live.Close()).To(Succeed())
		})
		It("Should acknowledge a writer of free channels", func(ctx SpecContext) {
			s := freeWriterScenario(ctx)
			defer func() { Expect(s.Close()).To(Succeed()) }()
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  s.keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(w.EndDomain(s.keys)).To(BeTrue())
			Expect(w.Close()).To(Succeed())
		})
	})

	Describe("Control", func() {
		DescribeTable(
			"Should report a refused write once and keep the end of a commit with control",
			func(ctx SpecContext, sync bool) {
				host := testutil.NewNode(ctx)
				keys := channel.KeysFromChannels(
					MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
						Name:     "refused_time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					}})),
				)
				open := func(key string, authority control.Authority) *writer.Writer {
					GinkgoHelper()
					return MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
						ControlSubject:   control.Subject{Key: key},
						Keys:             keys,
						Start:            10 * telem.SecondTS,
						Authorities:      []control.Authority{authority},
						EnableAutoCommit: new(false),
						Sync:             new(sync),
					}))
				}
				frameAt := func(second telem.TimeStamp) frame.Frame {
					return frame.NewMulti(
						keys,
						[]telem.Series{telem.NewSeriesSecondsTSV(second)},
					)
				}
				low := open("low", 1)
				Expect(low.Write(frameAt(10))).To(BeTrue())
				Expect(MustCommit(low)).To(Equal(10*telem.SecondTS + 1))
				high := open("high", 2)
				Expect(low.Write(frameAt(11))).To(Equal(!sync))
				// A synchronous command orders the write before control returns.
				Expect(low.SetAuthority(writer.Config{
					Authorities: []control.Authority{1},
				})).To(Succeed())
				Expect(high.Close()).To(Succeed())
				Expect(low.Write(frameAt(12))).To(BeTrue())
				end, authorized := MustSucceed2(low.Commit())
				Expect(authorized).To(BeFalse())
				Expect(end).To(Equal(12*telem.SecondTS + 1))
				Expect(low.Write(frameAt(13))).To(BeTrue())
				Expect(MustCommit(low)).To(Equal(13*telem.SecondTS + 1))
				Expect(low.Close()).To(Succeed())
				iter := MustSucceed(host.Framer.OpenIterator(ctx, iterator.Config{
					Keys:   keys,
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
				Expect(iter.Value().SeriesAt(0).Len()).To(Equal(int64(3)))
				Expect(iter.Close()).To(Succeed())
			},
			Entry("sync", true),
			Entry("async", false),
		)
		It(
			"Should hand off control to a waiting writer when a writer's context is canceled",
			func(ctx SpecContext) {
				host := testutil.NewNode(ctx)
				idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:     "handoff_time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				}}))[0]
				data := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:       "handoff",
					DataType:   telem.Int16T,
					LocalIndex: idx.LocalKey,
				}}))[0]
				keys := channel.Keys{idx.Key(), data.Key()}
				frameAt := func(values []int16, seconds ...telem.TimeStamp) frame.Frame {
					return frame.NewMulti(keys, []telem.Series{
						telem.NewSeriesSecondsTSV(seconds...),
						telem.NewSeries(values),
					})
				}
				open := func(
					ctx context.Context,
					key string,
					authority control.Authority,
				) *writer.Writer {
					GinkgoHelper()
					return MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
						ControlSubject: control.Subject{Key: key},
						Keys:           keys,
						Start:          10 * telem.SecondTS,
						Authorities:    []control.Authority{authority},
						Sync:           new(true),
					}))
				}
				firstCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				first := open(firstCtx, "first", control.AuthorityAbsolute-2)
				second := open(ctx, "second", control.AuthorityAbsolute-3)
				Expect(first.Write(frameAt([]int16{1, 2, 3}, 10, 11, 12))).To(BeTrue())
				cancel()
				Expect(first.Close()).To(Succeed())
				Expect(second.Write(frameAt([]int16{4, 5, 6}, 13, 14, 15))).To(BeTrue())
				Expect(MustCommit(second)).To(Equal(15*telem.SecondTS + 1))
				Expect(second.Close()).To(Succeed())
				iter := MustSucceed(host.Framer.OpenIterator(ctx, iterator.Config{
					Keys:   channel.Keys{data.Key()},
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
				Expect(iter.Value().Count()).To(Equal(1))
				Expect(iter.Value().SeriesAt(0).Data).To(Equal(
					telem.NewSeriesV[int16](1, 2, 3, 4, 5, 6).Data,
				))
				Expect(iter.Close()).To(Succeed())
			},
		)
	})

	Describe("Persist Only", func() {
		It(
			"Should commit a writer of stored and free channels that is not sync",
			func(ctx SpecContext) {
				host := testutil.NewNode(ctx)
				stored := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:     "persist_only_stored",
					Virtual:  true,
					DataType: telem.Int64T,
				}}))[0]
				free := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:        "persist_only_free",
					Virtual:     true,
					Leaseholder: node.KeyFree,
					DataType:    telem.Int64T,
				}}))[0]
				keys := channel.Keys{stored.Key(), free.Key()}
				w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
					Keys:             keys,
					Start:            10 * telem.SecondTS,
					Mode:             cesium.WriterModePersistOnly,
					EnableAutoCommit: new(false),
				}))
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
				}))).To(BeTrue())
				committed := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					defer close(committed)
					MustCommit(w)
				}()
				Eventually(committed).Should(BeClosed())
				Expect(w.Close()).To(Succeed())
			},
		)
		It(
			"Should return a storage write error from a writer that is not sync",
			func(ctx SpecContext) {
				host := testutil.NewNode(ctx)
				idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:        "persist_only_error_time",
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Leaseholder: 1,
				}}))[0]
				free := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:        "persist_only_error_free",
					Virtual:     true,
					Leaseholder: node.KeyFree,
					DataType:    telem.Int64T,
				}}))[0]
				keys := channel.Keys{idx.Key(), free.Key()}
				w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
					Keys:             keys,
					Start:            10 * telem.SecondTS,
					Mode:             cesium.WriterModePersistOnly,
					EnableAutoCommit: new(false),
				}))
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(13, 12, 11),
					telem.NewSeriesV[int64](4, 5, 6),
				}))).To(BeTrue())
				committed := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					defer close(committed)
					Expect(w.Commit()).Error().To(
						MatchError(ContainSubstring("index out of order")),
					)
				}()
				Eventually(committed).Should(BeClosed())
				Expect(w.Close()).To(MatchError(ContainSubstring("index out of order")))
			},
		)
		DescribeTable(
			"Should not stream the writes of a persist-only writer",
			func(ctx SpecContext, open func(context.Context) scenario) {
				s := DeferClose(open(ctx))
				streamer := MustSucceed(s.dist.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        s.keys,
					SendOpenAck: new(true),
				}))
				_, out := confluence.Attach(streamer, 10)
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				streamer.Flow(sCtx)
				Eventually(out.Outlet()).Should(Receive())
				w := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Mode:  cesium.WriterModePersistOnly,
					Sync:  new(true),
				}))
				Expect(w.Write(frame.NewMulti(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				))).To(BeTrue())
				Consistently(out.Outlet()).ShouldNot(Receive())
			},
			Entry("Storage", storageScenario),
			Entry("Free", freeWriterScenario),
		)
	})

	Describe("Auto-Index", func() {
		It(
			"Should stamp the index of a data channel",
			func(ctx SpecContext) {
				host := testutil.NewNode(ctx)
				idx := channel.Channel{
					Name:     "auto_index_time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				}
				idx = MustSucceed(host.CreateChannels(ctx, []channel.Channel{idx}))[0]
				data := channel.Channel{
					Name:       "auto_index_data",
					DataType:   telem.Float64T,
					LocalIndex: idx.LocalKey,
				}
				data = MustSucceed(host.CreateChannels(ctx, []channel.Channel{data}))[0]

				before := telem.Now()
				w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
					Keys:      []channel.Key{data.Key()},
					Sync:      new(true),
					AutoIndex: new(true),
					Start:     1 * telem.SecondTS,
				}))
				Expect(w.Write(frame.NewUnary(
					data.Key(),
					telem.NewSeriesV(1.1, 2.2, 3.3),
				))).To(BeTrue())
				MustCommit(w)
				Expect(w.Close()).To(Succeed())
				after := telem.Now()

				iter := MustSucceed(host.Framer.OpenIterator(ctx, iterator.Config{
					Keys:   []channel.Key{idx.Key()},
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
				ts := iter.Value().SeriesAt(0).Unmarshal[telem.TimeStamp]()
				Expect(iter.Close()).To(Succeed())

				Expect(ts).To(HaveLen(3))
				Expect(ts[0]).To(BeNumerically(">=", before))
				Expect(ts[2]).To(BeNumerically("<=", after+telem.TimeStamp(3)))
				Expect(ts[1]).To(Equal(ts[0] + 1))
				Expect(ts[2]).To(Equal(ts[0] + 2))
			},
		)
	})

	Describe("Free Write Alignment", Ordered, func() {
		It(
			"Should correctly set alignments on indexed free channels",
			func(ctx SpecContext) {
				s := DeferClose(storageScenario(ctx))
				var (
					idxCh = channel.Channel{
						Name:        "free_time",
						IsIndex:     true,
						DataType:    telem.TimestampT,
						Leaseholder: node.KeyFree,
						Virtual:     true,
					}
					dataCh = channel.Channel{
						Name:        "free",
						DataType:    telem.Float32T,
						Leaseholder: node.KeyFree,
						Virtual:     true,
					}
				)
				idxCh = MustSucceed(
					s.dist.CreateChannels(ctx, []channel.Channel{idxCh}),
				)[0]
				dataCh.LocalIndex = idxCh.LocalKey
				dataCh = MustSucceed(
					s.dist.CreateChannels(ctx, []channel.Channel{dataCh}),
				)[0]

				keys := []channel.Key{idxCh.Key(), dataCh.Key()}
				streamer := MustSucceed(s.dist.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				_, out := confluence.Attach(streamer, 10)
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				streamer.Flow(sCtx)
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				s.dist.Framer.SetChannelResolver(indexResolver{
					idxCh.Key():  idxCh.Key(),
					dataCh.Key(): idxCh.Key(),
				})
				writer := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				data := telem.NewSeriesV[float32](1, 2)
				idx := telem.NewSeriesSecondsTSV(10*telem.SecondTS, 11*telem.SecondTS)
				Expect(writer.Write(frame.NewMulti(
					keys,
					[]telem.Series{idx, data},
				))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(res.Frame.KeysSlice()).To(Equal(keys))
				writtenData := res.Frame.Get(dataCh.Key()).Series[0]
				Expect(writtenData).To(telem.MatchSeriesData(data))
				writtenIdx := res.Frame.Get(idxCh.Key()).Series[0]
				Expect(writtenIdx).To(telem.MatchSeriesData(idx))
				groupDomain := writtenData.Alignment.DomainIndex()
				Expect(groupDomain).To(BeNumerically(
					">=",
					MustSucceed(
						telem.NewAlignmentCounter(telem.LeadingDomain).Next(),
					).DomainIndex(),
				))
				Expect(writtenData.Alignment.SampleIndex()).To(BeEquivalentTo(0))
				Expect(writtenIdx.Alignment).To(Equal(writtenData.Alignment))
				Expect(writtenData.TimeRange).To(Equal(telem.TimeRangeZero))
				data = telem.NewSeriesV[float32](3, 4)
				idx = telem.NewSeriesSecondsTSV(12*telem.SecondTS, 13*telem.SecondTS)
				Expect(writer.Write(frame.NewMulti(
					keys,
					[]telem.Series{idx, data},
				))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(res.Frame.KeysSlice()).To(Equal(keys))
				writtenData = res.Frame.Get(dataCh.Key()).Series[0]
				Expect(writtenData).To(telem.MatchSeriesData(data))
				writtenIdx = res.Frame.Get(idxCh.Key()).Series[0]
				Expect(writtenIdx).To(telem.MatchSeriesData(idx))
				Expect(writtenData.Alignment.DomainIndex()).To(Equal(groupDomain))
				Expect(writtenData.Alignment.SampleIndex()).To(BeEquivalentTo(2))
				Expect(writtenIdx.Alignment).To(Equal(writtenData.Alignment))
			},
		)

		It(
			"Should stamp index-less free channels with the host's clock",
			func(ctx SpecContext) {
				s := DeferClose(storageScenario(ctx))
				ch := channel.Channel{
					Name:        "free_unindexed",
					DataType:    telem.Int64T,
					Leaseholder: node.KeyFree,
					Virtual:     true,
				}
				ch = MustSucceed(s.dist.CreateChannels(ctx, []channel.Channel{ch}))[0]
				keys := channel.Keys{ch.Key()}
				streamer := MustSucceed(s.dist.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				_, out := confluence.Attach(streamer, 10)
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				streamer.Flow(sCtx)
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				s.dist.Framer.SetChannelResolver(indexResolver{})
				writer := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				data := telem.NewSeriesV[int64](1, 2)
				data.TimeRange = telem.TimeRange{
					Start: 1 * telem.SecondTS,
					End:   2 * telem.SecondTS,
				}
				before := telem.Now()
				Expect(writer.Write(frame.NewUnary(ch.Key(), data))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				after := telem.Now()
				written := res.Frame.Get(ch.Key()).Series[0]
				Expect(written).To(telem.MatchSeriesData(data))
				Expect(written.TimeRange.Span()).To(Equal(2 * telem.Nanosecond))
				Expect(written.TimeRange.Start).To(BeNumerically(">=", before))
				Expect(written.TimeRange.Start).To(BeNumerically("<=", after))
			},
		)

		It(
			"Should give the samples of an index-less free channel distinct alignments",
			func(ctx SpecContext) {
				s := DeferClose(storageScenario(ctx))
				ch := channel.Channel{
					Name:        "free_unindexed_aligned",
					DataType:    telem.Int64T,
					Leaseholder: node.KeyFree,
					Virtual:     true,
				}
				ch = MustSucceed(s.dist.CreateChannels(ctx, []channel.Channel{ch}))[0]
				keys := channel.Keys{ch.Key()}
				streamer := MustSucceed(s.dist.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				_, out := confluence.Attach(streamer, 10)
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				streamer.Flow(sCtx)
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				s.dist.Framer.SetChannelResolver(indexResolver{})
				writer := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
					Sync:  new(true),
				}))
				Expect(writer.Write(frame.NewMulti(
					channel.Keys{ch.Key(), ch.Key()},
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2),
						telem.NewSeriesV[int64](3),
					},
				))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				series := res.Frame.Get(ch.Key()).Series
				Expect(series).To(HaveLen(2))
				Expect(series[1].Alignment).To(Equal(series[0].Alignment.AddSamples(2)))
				Expect(writer.Write(frame.NewUnary(
					ch.Key(),
					telem.NewSeriesV[int64](4),
				))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				next := res.Frame.Get(ch.Key()).Series[0]
				Expect(next.Alignment).To(Equal(series[1].Alignment.AddSamples(1)))
			},
		)

		It(
			"Should return an error when no channel resolver is configured",
			func(ctx SpecContext) {
				s := DeferClose(storageScenario(ctx))
				ch := channel.Channel{
					Name:        "free_unresolved",
					DataType:    telem.Int64T,
					Leaseholder: node.KeyFree,
					Virtual:     true,
				}
				ch = MustSucceed(s.dist.CreateChannels(ctx, []channel.Channel{ch}))[0]
				s.dist.Framer.SetChannelResolver(nil)
				Expect(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  channel.Keys{ch.Key()},
					Start: 10 * telem.SecondTS,
				})).Error().To(MatchError(
					ContainSubstring("no channel resolver configured"),
				))
			},
		)
	})

	Describe("Virtual Write Stamps", func() {
		// writeAndReceive writes the series to the keys in one frame and returns the
		// series streamed for each key with the instants that bound the write.
		writeAndReceive := func(
			ctx SpecContext,
			s scenario,
			keys channel.Keys,
			series ...telem.Series,
		) (map[channel.Key][]telem.Series, telem.TimeStamp, telem.TimeStamp) {
			GinkgoHelper()
			unique := keys.Unique()
			out := confluence.NewStream[framer.StreamerResponse](10)
			sCtx, cancel := signal.WithCancel(ctx)
			DeferCleanup(cancel)
			streamer := MustSucceed(s.dist.Framer.NewStreamer(framer.StreamerConfig{
				Keys:        unique,
				SendOpenAck: new(true),
			}))
			streamer.InFrom(confluence.NewStream[framer.StreamerRequest](1))
			streamer.OutTo(out)
			streamer.Flow(sCtx)
			var res framer.StreamerResponse
			Eventually(out.Outlet()).Should(Receive(&res))
			w := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  unique,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			for i := range series {
				series[i].TimeRange = telem.TimeRange{
					Start: 1 * telem.SecondTS,
					End:   2 * telem.SecondTS,
				}
			}
			before := telem.Now()
			Expect(w.Write(frame.NewMulti(keys, series))).To(BeTrue())
			received := make(map[channel.Key][]telem.Series, len(unique))
			for n := 0; n < len(series); {
				Eventually(out.Outlet()).Should(Receive(&res))
				for _, k := range unique {
					got := res.Frame.Get(k).Series
					received[k] = append(received[k], got...)
					n += len(got)
				}
			}
			return received, before, telem.Now()
		}

		pairs := func(keys channel.Keys) []telem.Series {
			series := make([]telem.Series, len(keys))
			for i := range series {
				series[i] = telem.NewSeriesV[int64](1, 2)
			}
			return series
		}

		DescribeTable(
			"Should stamp each sample of a virtual series 1ns after the one before",
			func(ctx SpecContext, open func(context.Context) scenario) {
				s := DeferClose(open(ctx))
				received, before, after := writeAndReceive(
					ctx,
					s,
					s.keys,
					pairs(s.keys)...,
				)
				for _, k := range s.keys {
					Expect(received[k]).To(HaveLen(1))
					written := received[k][0]
					Expect(
						written,
					).To(telem.MatchSeriesData(telem.NewSeriesV[int64](1, 2)))
					Expect(written.TimeRange.Span()).To(Equal(2 * telem.Nanosecond))
					Expect(written.TimeRange.Start).To(BeNumerically(">=", before))
					Expect(written.TimeRange.Start).To(BeNumerically("<=", after))
				}
			},
			Entry("Storage", storageScenario),
			Entry("Free", freeWriterScenario),
		)

		DescribeTable(
			"Should span a virtual series 1ns for each sample",
			func(ctx SpecContext, length int) {
				s := DeferClose(storageScenario(ctx))
				series := make([]telem.Series, len(s.keys))
				for i := range series {
					series[i] = telem.NewSeries(make([]int64, length))
				}
				received, _, _ := writeAndReceive(ctx, s, s.keys, series...)
				for _, k := range s.keys {
					Expect(
						received[k][0].TimeRange.Span(),
					).To(Equal(telem.TimeSpan(length) * telem.Nanosecond))
				}
			},
			Entry("One sample", 1),
			Entry("Three samples", 3),
			Entry("A thousand samples", 1000),
		)

		It(
			"Should span a variable-length virtual series 1ns for each sample",
			func(ctx SpecContext) {
				s := DeferClose(storageScenario(ctx))
				ch := MustSucceed(s.dist.CreateChannels(ctx, []channel.Channel{{
					Name:     "strings",
					Virtual:  true,
					DataType: telem.StringT,
				}}))[0]
				keys := channel.Keys{ch.Key()}
				received, _, _ := writeAndReceive(
					ctx,
					s,
					keys,
					telem.NewSeriesV("a", "bcdef"),
				)
				Expect(
					received[ch.Key()][0].TimeRange.Span(),
				).To(Equal(2 * telem.Nanosecond))
			},
		)

		It(
			"Should start every virtual series in one write at one instant",
			func(ctx SpecContext) {
				s := DeferClose(storageScenario(ctx))
				received, _, _ := writeAndReceive(
					ctx,
					s,
					s.keys,
					telem.NewSeriesV[int64](1),
					telem.NewSeriesV[int64](1, 2),
					telem.NewSeriesV[int64](1, 2, 3),
				)
				start := received[s.keys[0]][0].TimeRange.Start
				for i, k := range s.keys {
					Expect(received[k][0].TimeRange).To(Equal(
						start.SpanRange(telem.TimeSpan(i+1) * telem.Nanosecond),
					))
				}
			},
		)

		DescribeTable(
			"Should stamp a channel's series in one frame 1ns for each sample, in order",
			func(ctx SpecContext, open func(context.Context) scenario) {
				s := DeferClose(open(ctx))
				k := s.keys[0]
				received, _, _ := writeAndReceive(
					ctx,
					s,
					channel.Keys{k, k},
					telem.NewSeriesV[int64](1, 2),
					telem.NewSeriesV[int64](3),
				)
				Expect(received[k]).To(HaveLen(2))
				first, second := received[k][0].TimeRange, received[k][1].TimeRange
				Expect(first.Span()).To(Equal(2 * telem.Nanosecond))
				Expect(second).To(Equal(first.End.SpanRange(telem.Nanosecond)))
			},
			Entry("Storage", storageScenario),
			Entry("Free", freeWriterScenario),
		)

		It(
			"Should align indexed free channels and stamp index-less ones in one frame",
			func(ctx SpecContext) {
				s := DeferClose(storageScenario(ctx))
				idx := MustSucceed(s.dist.CreateChannels(ctx, []channel.Channel{{
					Name:        "mixed_time",
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Leaseholder: node.KeyFree,
					Virtual:     true,
				}}))[0]
				created := MustSucceed(s.dist.CreateChannels(ctx, []channel.Channel{
					{
						Name:        "mixed_data",
						DataType:    telem.Float32T,
						Leaseholder: node.KeyFree,
						Virtual:     true,
						LocalIndex:  idx.LocalKey,
					},
					{
						Name:        "mixed_unindexed",
						DataType:    telem.Int64T,
						Leaseholder: node.KeyFree,
						Virtual:     true,
					},
				}))
				data, unindexed := created[0], created[1]
				s.dist.Framer.SetChannelResolver(indexResolver{
					idx.Key():  idx.Key(),
					data.Key(): idx.Key(),
				})
				received, _, _ := writeAndReceive(
					ctx,
					s,
					channel.Keys{
						idx.Key(),
						data.Key(),
						unindexed.Key(),
						unindexed.Key(),
					},
					telem.NewSeriesSecondsTSV(10*telem.SecondTS, 11*telem.SecondTS),
					telem.NewSeriesV[float32](1, 2),
					telem.NewSeriesV[int64](1, 2),
					telem.NewSeriesV[int64](3),
				)
				client := telem.TimeRange{
					Start: 1 * telem.SecondTS,
					End:   2 * telem.SecondTS,
				}
				writtenIdx, writtenData := received[idx.Key()][0], received[data.Key()][0]
				Expect(writtenIdx.TimeRange).To(Equal(client))
				Expect(writtenData.TimeRange).To(Equal(client))
				Expect(writtenData.Alignment).To(Equal(writtenIdx.Alignment))
				Expect(received[unindexed.Key()]).To(HaveLen(2))
				first := received[unindexed.Key()][0].TimeRange
				second := received[unindexed.Key()][1].TimeRange
				Expect(first.Span()).To(Equal(2 * telem.Nanosecond))
				Expect(second).To(Equal(first.End.SpanRange(telem.Nanosecond)))
			},
		)
	})
})

// indexResolver resolves the records of free channels from a fixed key-to-index map.
type indexResolver map[channel.Key]channel.Key

func (r indexResolver) Resolve(
	_ context.Context,
	keys channel.Keys,
) (map[channel.Key]writer.Facts, error) {
	facts := make(map[channel.Key]writer.Facts, len(keys))
	for _, k := range keys {
		if idx, ok := r[k]; ok {
			facts[k] = writer.Facts{Index: idx}
		}
	}
	return facts, nil
}

type scenario struct {
	dist   testutil.Node
	closer io.Closer
	name   string
	keys   channel.Keys
}

func (s scenario) Close() error { return s.closer.Close() }

func newChannelSet() []channel.Channel {
	return []channel.Channel{
		{
			Name:     "test1",
			Virtual:  true,
			DataType: telem.Int64T,
		},
		{
			Name:     "test2",
			Virtual:  true,
			DataType: telem.Int64T,
		},
		{
			Name:     "test3",
			Virtual:  true,
			DataType: telem.Int64T,
		},
	}
}

func storageScenario(ctx context.Context) scenario {
	GinkgoHelper()
	channels := newChannelSet()
	node := testutil.OpenNode(GinkgoT().Context())
	channels = MustSucceed(node.CreateChannels(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:   "Storage",
		keys:   keys,
		dist:   node,
		closer: node,
	}
}

func freeWriterScenario(ctx context.Context) scenario {
	GinkgoHelper()
	channels := newChannelSet()
	dist := testutil.OpenNode(GinkgoT().Context())
	for i, ch := range channels {
		ch.Leaseholder = node.KeyFree
		ch.Virtual = true
		channels[i] = ch
	}
	channels = MustSucceed(dist.CreateChannels(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Free Writes", keys: keys, dist: dist, closer: dist}
}
