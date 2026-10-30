// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation", Ordered, func() {
	var (
		c             *calculation.Service
		dist          testutil.Node
		statusSvc     *status.Service
		channelSvc    *channel.Service
		channelWriter channel.Writer
		channelGraph  *calcgraph.Graph
	)
	open := func(
		ctx context.Context,
		indexChannels,
		baseChannels,
		calculations *[]channel.Channel,
		streamKeys func([]channel.Channel) channel.Keys,
	) (*framer.Writer, confluence.Outlet[streamer.Response], context.CancelFunc) {
		GinkgoHelper()
		if indexChannels != nil {
			Expect(channelWriter.CreateMany(ctx, indexChannels)).To(Succeed())
		}
		for i, channel := range *baseChannels {
			if channel.Virtual {
				continue
			}
			toGet := i
			if len(*indexChannels) == 1 {
				toGet = 0
			}
			channel.LocalIndex = (*indexChannels)[toGet].LocalKey
			(*baseChannels)[i] = channel
		}
		Expect(channelWriter.CreateMany(ctx, baseChannels)).To(Succeed())
		Expect(channelWriter.CreateMany(ctx, calculations)).To(Succeed())
		rm := c.OpenRequestManager()
		Expect(rm.Set(ctx, channel.KeysFromChannels(*calculations))).To(Succeed())
		writerKeys := channel.KeysFromChannels(*baseChannels)
		if indexChannels != nil {
			writerKeys = append(writerKeys, channel.KeysFromChannels(*indexChannels)...)
		}
		sCtx, cancel := signal.Isolated()
		w := MustSucceed(dist.Framer.OpenWriter(
			ctx,
			framer.WriterConfig{
				Start: 1 * telem.SecondTS,
				Keys:  writerKeys,
			},
		))
		filtered := lo.Filter(
			*calculations,
			func(item channel.Channel, index int) bool {
				return !item.IsIndex
			},
		)
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				framer.StreamerConfig{
					Keys:        streamKeys(filtered),
					SendOpenAck: new(true),
				},
			),
		)
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet()).Should(Receive())
		return w, sOutlet, func() {
			Expect(rm.Close(ctx)).To(Succeed())
			Expect(w.Close()).To(Succeed())
			cancel()
		}
	}

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		dist = testutil.NewNode(ctx)
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: dist.DB}))
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       dist.DB,
			Ontology: otg,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       dist.DB,
			Ontology: otg,
			Group:    groupSvc,
		}))
		statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       dist.DB,
			Group:    groupSvc,
			Ontology: otg,
			Label:    labelSvc,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      dist.Channel,
			DB:           dist.DB,
			HostProvider: dist.Cluster,
			Ontology:     otg,
			Group:        groupSvc,
			Status:       statusSvc,
		}))
		channelWriter = channelSvc.NewWriter(nil)
		writerSvc := MustSucceed(writer.NewService(writer.ServiceConfig{
			Framer:  dist.Framer,
			Channel: channelSvc,
		}))
		channelGraph = MustOpen(calcgraph.Open(ctx, calcgraph.Config{
			DB:      dist.DB,
			Channel: channelSvc,
			Status:  statusSvc,
		}))
		c = MustOpen(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:           dist.DB,
			Framer:       dist.Framer,
			Writer:       writerSvc,
			Channel:      channelSvc,
			ChannelGraph: channelGraph,
		}))
	})

	Describe("Calculation Patterns", func() {
		Specify("Single Virtual Channel as Base", func(ctx SpecContext) {
			bases := []channel.Channel{{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			calcs := []channel.Channel{{
				Name:        UniqueChannelName(),
				DataType:    telem.Int64T,
				Virtual:     true,
				Leaseholder: node.KeyFree,
				Expression:  fmt.Sprintf("return %s * 2", bases[0].Name),
			}}
			w, sOutlet, cancel := open(
				ctx,
				nil,
				&bases,
				&calcs,
				channel.KeysFromChannels,
			)
			defer cancel()
			baseCh := bases[0]
			calcCh := calcs[0]
			Expect(
				w.Write(frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2))),
			).To(BeTrue())
			var res framer.StreamerResponse
			Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
			Expect(
				res.Frame.Get(calcCh.Key()).Series[0],
			).To(telem.MatchSeriesDataV[int64](2, 4))
			Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
		})

		Describe("Two Virtual Channels as Bases", func() {
			var (
				bases []channel.Channel
				calcs []channel.Channel
			)
			BeforeEach(func(ctx SpecContext) {
				bases = []channel.Channel{
					{
						Name:     UniqueChannelName(),
						DataType: telem.Int64T,
						Virtual:  true,
					},
					{
						Name:     UniqueChannelName(),
						DataType: telem.Int64T,
						Virtual:  true,
					},
				}
				calcs = []channel.Channel{
					{
						Name:        UniqueChannelName(),
						DataType:    telem.Int64T,
						Virtual:     true,
						Leaseholder: node.KeyFree,
						Expression: fmt.Sprintf(
							"return %s * %s",
							bases[0].Name,
							bases[1].Name,
						),
					},
				}
			})
			Specify("Single Write with Data for Both Channels", func(ctx SpecContext) {
				w, sOutlet, cancel := open(
					ctx,
					nil,
					&bases,
					&calcs,
					channel.KeysFromChannels,
				)
				defer cancel()
				baseCh1 := bases[0]
				baseCh2 := bases[1]
				calcCh := calcs[0]
				Expect(w.Write(frame.NewMulti(
					[]channel.Key{baseCh1.Key(), baseCh2.Key()},
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2),
						telem.NewSeriesV[int64](2, 4),
					},
				))).To(BeTrue())
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
				Expect(
					res.Frame.Get(calcCh.Key()).Series[0],
				).To(telem.MatchSeriesDataV[int64](2, 8))
				Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
			})

			Specify(
				"Two Writes with Data for Individual Channels",
				func(ctx SpecContext) {
					w, sOutlet, cancel := open(
						ctx,
						nil,
						&bases,
						&calcs,
						channel.KeysFromChannels,
					)
					defer cancel()
					baseCh1 := bases[0]
					baseCh2 := bases[1]
					calcCh := calcs[0]
					Expect(
						w.Write(
							frame.NewUnary(
								baseCh1.Key(),
								telem.NewSeriesV[int64](1, 2),
							),
						),
					).To(BeTrue())
					Expect(
						w.Write(
							frame.NewUnary(
								baseCh2.Key(),
								telem.NewSeriesV[int64](2, 4),
							),
						),
					).To(BeTrue())
					var res framer.StreamerResponse
					Eventually(sOutlet.Outlet()).Should(Receive(&res))
					Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
					Expect(
						res.Frame.Get(calcCh.Key()).Series[0],
					).To(telem.MatchSeriesDataV[int64](2, 8))
					Consistently(
						sOutlet.Outlet(),
						10*time.Millisecond,
					).ShouldNot(Receive())
				},
			)
		})

		Specify("Single Data Channel as Base", func(ctx SpecContext) {
			var (
				indexes = []channel.Channel{{
					Name:     UniqueChannelName(),
					DataType: telem.TimestampT,
					IsIndex:  true,
				}}
				bases = []channel.Channel{{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
				}}
				calcs = []channel.Channel{{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s * 2", bases[0].Name),
				}}
			)
			w, sOutlet, cancel := open(
				ctx,
				&indexes,
				&bases,
				&calcs,
				channel.KeysFromChannels,
			)
			defer cancel()
			idxCh := indexes[0]
			baseCh := bases[0]
			calcCh := calcs[0]
			Expect(w.Write(frame.NewMulti(
				[]channel.Key{idxCh.Key(), baseCh.Key()},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2),
					telem.NewSeriesV[int64](1, 2),
				},
			))).To(BeTrue())
			var res framer.StreamerResponse
			Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
			Expect(
				res.Frame.Get(calcCh.Key()).Series[0],
			).To(telem.MatchSeriesDataV[int64](2, 4))
			Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
		})

		Describe("Multiple Data Channels as Base", func() {
			Specify("Shared Index", func(ctx SpecContext) {
				var (
					indexes = []channel.Channel{{
						Name:     UniqueChannelName(),
						DataType: telem.TimestampT,
						IsIndex:  true,
					}}
					bases = []channel.Channel{
						{
							Name:     UniqueChannelName(),
							DataType: telem.Float32T,
						},
						{
							Name:     UniqueChannelName(),
							DataType: telem.Float32T,
						},
					}
					calcs = []channel.Channel{
						{
							Name:        UniqueChannelName(),
							DataType:    telem.Float32T,
							Virtual:     true,
							Leaseholder: node.KeyFree,
							Expression: fmt.Sprintf(
								"return %s * %s",
								bases[0].Name,
								bases[1].Name,
							),
						},
					}
				)
				w, sOutlet, cancel := open(
					ctx,
					&indexes,
					&bases,
					&calcs,
					channel.KeysFromChannels,
				)
				defer cancel()
				idxCh := indexes[0]
				baseCh1 := bases[0]
				baseCh2 := bases[1]
				calcCh := calcs[0]
				Expect(w.Write(frame.NewMulti(
					[]channel.Key{idxCh.Key(), baseCh1.Key(), baseCh2.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2),
						telem.NewSeriesV[float32](1, 2),
						telem.NewSeriesV[float32](2, 4),
					},
				))).To(BeTrue())
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(
					res.Frame.Get(calcCh.Key()).Series[0],
				).To(telem.MatchSeriesDataV[float32](2, 8))
			})

			Specify("Unique Indexes", func(ctx SpecContext) {
				var (
					indexes = []channel.Channel{
						{
							Name:     UniqueChannelName(),
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
						{
							Name:     UniqueChannelName(),
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
					}
					bases = []channel.Channel{
						{
							Name:     UniqueChannelName(),
							DataType: telem.Float32T,
						},
						{
							Name:     UniqueChannelName(),
							DataType: telem.Float32T,
						},
					}
					calcs = []channel.Channel{
						{
							Name:        UniqueChannelName(),
							DataType:    telem.Float32T,
							Virtual:     true,
							Leaseholder: node.KeyFree,
							Expression: fmt.Sprintf(
								"return %s * %s",
								bases[0].Name,
								bases[1].Name,
							),
						},
					}
				)
				w, sOutlet, cancel := open(
					ctx,
					&indexes,
					&bases,
					&calcs,
					channel.KeysFromChannels,
				)
				defer cancel()
				var (
					idxCh1  = indexes[0]
					idxCh2  = indexes[1]
					baseCh1 = bases[0]
					baseCh2 = bases[1]
					calcCh  = calcs[0]
				)
				Expect(w.Write(frame.NewMulti(
					[]channel.Key{
						idxCh1.Key(),
						idxCh2.Key(),
						baseCh1.Key(),
						baseCh2.Key(),
					},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2),
						telem.NewSeriesSecondsTSV(3, 4),
						telem.NewSeriesV[float32](1, 2),
						telem.NewSeriesV[float32](2, 4),
					},
				))).To(BeTrue())
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(
					res.Frame.Get(calcCh.Key()).Series[0],
				).To(telem.MatchSeriesDataV[float32](2, 8))
			})

			Specify("Unique Indexes, Separate Writes", func(ctx SpecContext) {
				var (
					indexes = []channel.Channel{
						{
							Name:     UniqueChannelName(),
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
						{
							Name:     UniqueChannelName(),
							DataType: telem.TimestampT,
							IsIndex:  true,
						},
					}
					bases = []channel.Channel{
						{
							Name:     UniqueChannelName(),
							DataType: telem.Float32T,
						},
						{
							Name:     UniqueChannelName(),
							DataType: telem.Float32T,
						},
					}
					calcs = []channel.Channel{
						{
							Name:        UniqueChannelName(),
							DataType:    telem.Float32T,
							Virtual:     true,
							Leaseholder: node.KeyFree,
							Expression: fmt.Sprintf(
								"return %s * %s",
								bases[0].Name,
								bases[1].Name,
							),
						},
					}
				)
				w, sOutlet, cancel := open(
					ctx,
					&indexes,
					&bases,
					&calcs,
					channel.KeysFromChannels,
				)
				defer cancel()
				var (
					idxCh1  = indexes[0]
					idxCh2  = indexes[1]
					baseCh1 = bases[0]
					baseCh2 = bases[1]
					calcCh  = calcs[0]
				)
				Expect(w.Write(frame.NewMulti(
					[]channel.Key{idxCh1.Key(), baseCh1.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(3, 4),
						telem.NewSeriesV[float32](2, 4),
					},
				))).To(BeTrue())
				Expect(w.Write(frame.NewMulti(
					[]channel.Key{idxCh2.Key(), baseCh2.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2),
						telem.NewSeriesV[float32](2, 4),
					},
				))).To(BeTrue())
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(
					res.Frame.Get(calcCh.Key()).Series[0],
				).To(telem.MatchSeriesDataV[float32](4, 16))
			})
		})

		Describe("Nested Calculations", func() {
			var (
				bases []channel.Channel
				calcs []channel.Channel
			)
			BeforeEach(func(ctx SpecContext) {
				calc1Name := UniqueChannelName()
				bases = []channel.Channel{{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
					Virtual:  true,
				}}
				calcs = []channel.Channel{{
					Name:        calc1Name,
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s * 2", bases[0].Name),
				}, {
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s * 2", calc1Name),
				}}
			})
			Specify("Base and Derived Requested", func(ctx SpecContext) {
				w, sOutlet, cancel := open(
					ctx,
					nil,
					&bases,
					&calcs,
					channel.KeysFromChannels,
				)
				defer cancel()
				baseCh := bases[0]
				calcCh := calcs[0]
				calc2Ch := calcs[1]
				Expect(
					w.Write(
						frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2)),
					),
				).To(BeTrue())

				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(
					res.Frame.KeysSlice(),
				).To(Equal([]channel.Key{calcCh.Key(), calc2Ch.Key()}))
				Expect(
					res.Frame.Get(calcCh.Key()).Series[0],
				).To(telem.MatchSeriesDataV[int64](2, 4))
				Expect(
					res.Frame.Get(calc2Ch.Key()).Series[0],
				).To(telem.MatchSeriesDataV[int64](4, 8))
			})

			Specify(
				"Calculations of Calculations, Base Not Requested",
				func(ctx SpecContext) {
					w, sOutlet, cancel := open(
						ctx,
						nil,
						&bases,
						&calcs,
						func(calcs []channel.Channel) channel.Keys {
							return []channel.Key{calcs[1].Key()}
						},
					)
					defer cancel()
					baseCh := bases[0]
					calc2Ch := calcs[1]
					Expect(
						w.Write(
							frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2)),
						),
					).To(BeTrue())

					var res framer.StreamerResponse
					Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
					Expect(
						res.Frame.KeysSlice(),
					).To(Equal([]channel.Key{calc2Ch.Key()}))
					Expect(
						res.Frame.Get(calc2Ch.Key()).Series[0],
					).To(telem.MatchSeriesDataV[int64](4, 8))
				},
			)
		})

		Describe("Reset Channels", func() {
			var base, sig channel.Channel
			// Requests only calc, writes [10, 20, 30] with the signal off, then [5, 7]
			// with the signal on for the first sample, and expects the max to clear.
			expectReset := func(
				ctx context.Context,
				calc channel.Channel,
				off, on telem.Series,
			) {
				GinkgoHelper()
				rm := c.OpenRequestManager()
				Expect(rm.Set(ctx, channel.Keys{calc.Key()})).To(Succeed())
				sCtx, cancel := signal.Isolated()
				w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
					Start: 1 * telem.SecondTS,
					Keys:  channel.Keys{base.Key(), sig.Key()},
				}))
				defer func() {
					Expect(rm.Close(ctx)).To(Succeed())
					Expect(w.Close()).To(Succeed())
					cancel()
				}()
				strm := MustSucceed(dist.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        channel.Keys{calc.Key()},
					SendOpenAck: new(true),
				}))
				_, sOutlet := confluence.Attach(strm, 1, 1)
				strm.Flow(sCtx)
				Eventually(sOutlet.Outlet()).Should(Receive())

				keys := []channel.Key{base.Key(), sig.Key()}
				Expect(w.Write(frame.NewMulti(
					keys,
					[]telem.Series{telem.NewSeriesV[int64](10, 20, 30), off},
				))).To(BeTrue())
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.Get(calc.Key()).Series[0]).
					To(telem.MatchSeriesDataV[int64](30))

				Expect(w.Write(frame.NewMulti(
					keys,
					[]telem.Series{telem.NewSeriesV[int64](5, 7), on},
				))).To(BeTrue())
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.Get(calc.Key()).Series[0]).
					To(telem.MatchSeriesDataV[int64](7))
			}

			BeforeEach(func(ctx SpecContext) {
				base = channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
					Virtual:  true,
				}
			})

			It("Should reset from a virtual boolean channel", func(ctx SpecContext) {
				sig = channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.BooleanT,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				Expect(channelWriter.Create(ctx, &sig)).To(Succeed())
				calc := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s", base.Name),
					Operations: []channel.Operation{{
						Type:         channel.OperationTypeMax,
						ResetChannel: sig.Key(),
					}},
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				expectReset(
					ctx,
					calc,
					telem.NewSeriesV(false, false, false),
					telem.NewSeriesV(true, false),
				)
			})

			It("Should reset from a calculated boolean channel", func(ctx SpecContext) {
				sig = channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Uint8T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				Expect(channelWriter.Create(ctx, &sig)).To(Succeed())
				resetCalc := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.BooleanT,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s != 0", sig.Name),
				}
				Expect(channelWriter.Create(ctx, &resetCalc)).To(Succeed())
				calc := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s", base.Name),
					Operations: []channel.Operation{{
						Type:         channel.OperationTypeMax,
						ResetChannel: resetCalc.Key(),
					}},
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				expectReset(
					ctx,
					calc,
					telem.NewSeriesV[uint8](0, 0, 0),
					telem.NewSeriesV[uint8](1, 0),
				)
			})
		})
	})

	Describe("Requests", func() {
		It(
			"Should calculate a source write issued right after the request",
			func(ctx SpecContext) {
				base := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s * 2", base.Name),
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
					Start: telem.Now(),
					Keys:  channel.Keys{base.Key()},
				}))
				defer func() { Expect(w.Close()).To(Succeed()) }()
				s := MustSucceed(dist.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        channel.Keys{calc.Key()},
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				_, outlet := confluence.Attach(s, 1, 1)
				s.Flow(sCtx)
				Eventually(outlet.Outlet()).Should(Receive())
				rm := c.OpenRequestManager()
				Expect(rm.Set(ctx, channel.Keys{calc.Key()})).To(Succeed())
				defer func() { Expect(rm.Close(ctx)).To(Succeed()) }()
				Expect(
					w.Write(frame.NewUnary(base.Key(), telem.NewSeriesV[int64](1, 2))),
				).To(BeTrue())
				var res framer.StreamerResponse
				Eventually(outlet.Outlet()).Should(Receive(&res))
				Expect(
					res.Frame.Get(calc.Key()).Series[0],
				).To(telem.MatchSeriesDataV[int64](2, 4))
			},
		)
	})

	Describe("Calculation Status", func() {
		Specify(
			"Should persist error status on invalid expression request",
			func(ctx SpecContext) {
				base := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s * 2", base.Name),
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				Expect(channelWriter.Delete(
					ctx, base.Key(), false,
				),
				).To(Succeed())
				stored := func() (calculation.Status, error) {
					var st calculation.Status
					err := statusSvc.NewRetrieve[struct{}]().
						Where(status.MatchKeys[struct{}](calc.OntologyID().String())).
						Entry(&st).
						Exec(ctx, nil)
					return st, err
				}
				Eventually(stored).
					Should(HaveField("Message", "Failed to compile calculation"))
				rm := c.OpenRequestManager()
				Expect(
					rm.Set(ctx, channel.KeysFromChannels([]channel.Channel{calc})),
				).To(Succeed())
				st := MustSucceed(stored())
				Expect(st.Variant).To(Equal(status.VariantError))
				Expect(st.Name).To(Equal(calc.Name))
				Expect(st.Message).To(Equal("Failed to request calculation"))
				Expect(st.Description).To(ContainSubstring(base.Name))
				Expect(rm.Close(ctx)).To(Succeed())
			},
		)
		Specify(
			"Should persist error status on calculation update failure",
			func(ctx SpecContext) {
				base := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Float64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s + 1", base.Name),
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				rm := c.OpenRequestManager()
				Expect(rm.Set(ctx, channel.Keys{calc.Key()})).To(Succeed())
				// Delete the dependency, then rename the calc to force a write of its
				// record. The runtime reacts to writes of the calc channel by
				// recompiling it, which now fails because the dependency no longer
				// resolves. Rename (unlike Create) persists the record without
				// re-analyzing the expression, so the failure surfaces asynchronously
				// as an error status rather than synchronously.
				Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())
				Expect(channelWriter.Rename(
					ctx, calc.Key(), UniqueChannelName(), false,
				),
				).To(Succeed())
				var st calculation.Status
				statusKey := calc.OntologyID().String()
				Eventually(func(g Gomega) {
					g.Expect(statusSvc.NewRetrieve[struct{}]().
						Where(status.MatchKeys[struct{}](statusKey)).
						Entry(&st).
						Exec(ctx, nil)).To(Succeed())
					g.Expect(st.Variant).To(Equal(status.VariantError))
					g.Expect(st.Message).To(Equal("Failed to compile calculation"))
					g.Expect(st.Description).To(ContainSubstring(base.Name))
				}).Should(Succeed())
				Expect(rm.Close(ctx)).To(Succeed())
			},
		)
		Specify(
			"Should write runtime failures of a dependency after a rejected request",
			func(ctx SpecContext) {
				base := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				dep := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return 100 / %s", base.Name),
				}
				Expect(channelWriter.Create(ctx, &dep)).To(Succeed())
				stored := func() (calculation.Status, error) {
					var st calculation.Status
					err := statusSvc.NewRetrieve[struct{}]().
						Where(status.MatchKeys[struct{}](dep.OntologyID().String())).
						Entry(&st).
						Exec(ctx, nil)
					return st, err
				}
				name := base.Name
				Expect(channelWriter.Rename(
					ctx, base.Key(), UniqueChannelName(), false,
				)).To(Succeed())
				Eventually(stored).
					Should(HaveField("Message", "Failed to compile calculation"))
				depRM := c.OpenRequestManager()
				DeferCleanup(func(ctx SpecContext) {
					Expect(depRM.Close(ctx)).To(Succeed())
				})
				Expect(depRM.Set(ctx, channel.Keys{dep.Key()})).To(Succeed())
				Expect(stored()).
					To(HaveField("Message", "Failed to request calculation"))
				Expect(channelWriter.Rename(ctx, base.Key(), name, false)).
					To(Succeed())
				Eventually(func() error {
					_, err := stored()
					return err
				}).Should(MatchError(query.ErrNotFound))
				reader := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s * 2", dep.Name),
				}
				Expect(channelWriter.Create(ctx, &reader)).To(Succeed())
				readerRM := c.OpenRequestManager()
				DeferCleanup(func(ctx SpecContext) {
					Expect(readerRM.Close(ctx)).To(Succeed())
				})
				Expect(readerRM.Set(ctx, channel.Keys{reader.Key()})).To(Succeed())
				w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
					Start: 1 * telem.SecondTS,
					Keys:  channel.Keys{base.Key()},
				}))
				DeferCleanup(func() { Expect(w.Close()).To(Succeed()) })
				Eventually(func(g Gomega) {
					g.Expect(w.Write(frame.NewUnary(
						base.Key(),
						telem.NewSeriesV[int64](0),
					))).To(BeTrue())
					g.Expect(stored()).To(SatisfyAll(
						HaveField("Message", "Failed to run calculation"),
						HaveField(
							"Description",
							ContainSubstring("integer divide by zero"),
						),
					))
				}).Should(Succeed())
			},
		)

		Describe("Runtime", func() {
			var (
				w      *framer.Writer
				out    confluence.Outlet[streamer.Response]
				base   channel.Channel
				calc   channel.Channel
				mu     sync.Mutex
				writes []calculation.Status
			)
			// openCalc must run in the spec body: the writer closes with its context.
			openCalc := func(ctx context.Context) {
				GinkgoHelper()
				bases := []channel.Channel{{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
					Virtual:  true,
				}}
				calcs := []channel.Channel{{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return 100 / %s", bases[0].Name),
				}}
				var cancel context.CancelFunc
				w, out, cancel = open(
					ctx,
					nil,
					&bases,
					&calcs,
					channel.KeysFromChannels,
				)
				DeferCleanup(cancel)
				base, calc = bases[0], calcs[0]
				mu.Lock()
				writes = nil
				mu.Unlock()
				DeferCleanup(statusSvc.Observe().OnChange(func(
					_ context.Context,
					r gorp.TxReader[status.Key, status.Status[any]],
				) {
					mu.Lock()
					defer mu.Unlock()
					for ch := range r {
						if ch.Key == calc.OntologyID().String() &&
							ch.Variant == change.VariantSet {
							writes = append(writes, calculation.Status{
								Variant:     ch.Value.Variant,
								Message:     ch.Value.Message,
								Description: ch.Value.Description,
							})
						}
					}
				}))
			}
			written := func() []calculation.Status {
				mu.Lock()
				defer mu.Unlock()
				return slices.Clone(writes)
			}
			write := func(values ...int64) {
				GinkgoHelper()
				Expect(w.Write(frame.NewUnary(
					base.Key(),
					telem.NewSeriesV(values...),
				))).To(BeTrue())
			}
			// A good frame flushes the frames before it, since one goroutine runs the
			// calculation of every frame in order.
			writeGood := func() {
				GinkgoHelper()
				write(5)
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(res.Frame.Get(calc.Key()).Series[0]).
					To(telem.MatchSeriesDataV[int64](20))
			}
			failure := func() types.GomegaMatcher {
				return SatisfyAll(
					HaveField("Variant", status.VariantError),
					HaveField("Message", "Failed to run calculation"),
					HaveField(
						"Description",
						ContainSubstring("integer divide by zero"),
					),
				)
			}
			success := calculation.Status{
				Variant: status.VariantSuccess,
				Message: "Calculation running",
			}

			It(
				"Should write a failure once until its text changes",
				func(ctx SpecContext) {
					openCalc(ctx)
					write(0)
					write(0)
					write(0)
					Eventually(written).Should(ConsistOf(failure()))
					write(0, 0)
					Eventually(written).Should(HaveLen(2))
					got := written()
					Expect(got).To(HaveEach(failure()))
					Expect(got[1].Description).ToNot(Equal(got[0].Description))
					writeGood()
					Expect(written()).To(HaveLen(3))
				},
			)

			It(
				"Should write a success on the first good frame after a failure",
				func(ctx SpecContext) {
					openCalc(ctx)
					writeGood()
					Expect(written()).To(BeEmpty())
					write(0)
					Eventually(written).Should(ConsistOf(failure()))
					writeGood()
					writeGood()
					Expect(written()).To(HaveExactElements(failure(), success))
				},
			)

			It(
				"Should write a success after a rebuild during a failure",
				func(ctx SpecContext) {
					openCalc(ctx)
					write(0)
					Eventually(written).Should(ConsistOf(failure()))
					sibling := channel.Channel{
						Name:        UniqueChannelName(),
						DataType:    telem.Int64T,
						Virtual:     true,
						Leaseholder: node.KeyFree,
						Expression:  fmt.Sprintf("return %s * 2", base.Name),
					}
					Expect(channelWriter.Create(ctx, &sibling)).To(Succeed())
					rm := c.OpenRequestManager()
					Expect(rm.Set(ctx, channel.Keys{sibling.Key()})).To(Succeed())
					DeferCleanup(func(ctx SpecContext) {
						Expect(rm.Close(ctx)).To(Succeed())
					})
					writeGood()
					Eventually(written).
						Should(HaveExactElements(failure(), success))
				},
			)

			// flushChanges returns after the Service handles each channel change made
			// before the call: the channel graph notifies one batch at a time.
			flushChanges := func(ctx context.Context) {
				GinkgoHelper()
				name := UniqueChannelName()
				seen := make(chan struct{})
				disconnect := channelGraph.Observe().OnChange(func(
					_ context.Context,
					r calcgraph.Changes,
				) {
					for ch := range r {
						if ch.Value.Name == name {
							close(seen)
						}
					}
				})
				defer disconnect()
				Expect(channelWriter.Create(ctx, &channel.Channel{
					Name:     name,
					DataType: telem.Int64T,
					Virtual:  true,
				})).To(Succeed())
				Eventually(seen).Should(BeClosed())
			}

			It(
				"Should keep a failure after a rename of the channel",
				func(ctx SpecContext) {
					openCalc(ctx)
					write(0)
					Eventually(written).Should(ConsistOf(failure()))
					Expect(channelWriter.Rename(
						ctx, calc.Key(), UniqueChannelName(), false,
					)).To(Succeed())
					flushChanges(ctx)
					writeGood()
					Eventually(written).
						Should(HaveExactElements(failure(), success))
				},
			)

			It(
				"Should keep a failure after a second request of the channel",
				func(ctx SpecContext) {
					openCalc(ctx)
					write(0)
					Eventually(written).Should(ConsistOf(failure()))
					rm := c.OpenRequestManager()
					Expect(rm.Set(ctx, channel.Keys{calc.Key()})).To(Succeed())
					DeferCleanup(func(ctx SpecContext) {
						Expect(rm.Close(ctx)).To(Succeed())
					})
					writeGood()
					Eventually(written).
						Should(HaveExactElements(failure(), success))
				},
			)
			// failAndRebuild sends a failing and a good frame, rebuilds the group with
			// a sibling calculation of baseName, and sends a good frame.
			failAndRebuild := func(ctx context.Context, baseName string) {
				GinkgoHelper()
				write(0)
				writeGood()
				sibling := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s * 2", baseName),
				}
				Expect(channelWriter.Create(ctx, &sibling)).To(Succeed())
				rm := c.OpenRequestManager()
				Expect(rm.Set(ctx, channel.Keys{sibling.Key()})).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(rm.Close(ctx)).To(Succeed())
				})
				writeGood()
			}
			compileFailure := func() types.GomegaMatcher {
				return HaveField("Message", "Failed to compile calculation")
			}

			It(
				"Should keep a compile failure through runtime failures and rebuilds",
				func(ctx SpecContext) {
					openCalc(ctx)
					dependent := channel.Channel{
						Name:        UniqueChannelName(),
						DataType:    telem.Int64T,
						Virtual:     true,
						Leaseholder: node.KeyFree,
						Expression:  fmt.Sprintf("return %s * 2", calc.Name),
					}
					Expect(channelWriter.Create(ctx, &dependent)).To(Succeed())
					calc.Expression = fmt.Sprintf(
						"return %s + 100 / %s", dependent.Name, base.Name,
					)
					Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
					Eventually(written).Should(ContainElement(SatisfyAll(
						compileFailure(),
						HaveField("Description", ContainSubstring("circular")),
					)))
					failAndRebuild(ctx, base.Name)
					Expect(written()).To(HaveEach(compileFailure()))
				},
			)

			It(
				"Should keep a compile failure after a dependency rename without a save",
				func(ctx SpecContext) {
					openCalc(ctx)
					baseName := UniqueChannelName()
					Expect(channelWriter.Rename(ctx, base.Key(), baseName, false)).
						To(Succeed())
					Eventually(written).Should(ContainElement(compileFailure()))
					failAndRebuild(ctx, baseName)
					Expect(written()).To(HaveEach(compileFailure()))
				},
			)

			It(
				"Should write a failure again after a dependency is renamed away and back",
				func(ctx SpecContext) {
					openCalc(ctx)
					write(0)
					Eventually(written).Should(ConsistOf(failure()))
					name := base.Name
					Expect(channelWriter.Rename(
						ctx, base.Key(), UniqueChannelName(), false,
					)).To(Succeed())
					Eventually(written).Should(ContainElement(compileFailure()))
					Expect(channelWriter.Rename(ctx, base.Key(), name, false)).
						To(Succeed())
					stored := func() (calculation.Status, error) {
						var st calculation.Status
						err := statusSvc.NewRetrieve[struct{}]().
							Where(status.MatchKeys[struct{}](calc.OntologyID().String())).
							Entry(&st).
							Exec(ctx, nil)
						return st, err
					}
					Eventually(func() error {
						_, err := stored()
						return err
					}).Should(MatchError(query.ErrNotFound))
					write(0)
					Eventually(stored).Should(failure())
				},
			)
		})

		Specify("Should use channel ontology ID as status key", func(ctx SpecContext) {
			base := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.Int64T,
				Virtual:     true,
				Leaseholder: node.KeyFree,
				Expression:  fmt.Sprintf("return %s * 2", base.Name),
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())
			rm := c.OpenRequestManager()
			Expect(rm.Set(ctx, channel.Keys{calc.Key()})).To(Succeed())
			var st calculation.Status
			expectedKey := calc.OntologyID().String()
			Expect(statusSvc.NewRetrieve[struct{}]().
				Where(status.MatchKeys[struct{}](expectedKey)).
				Entry(&st).
				Exec(ctx, nil)).To(Succeed())
			Expect(st.Key).To(Equal(expectedKey))
			Expect(rm.Close(ctx)).To(Succeed())
		})
	})

	Describe("Calculation Updates", func() {
		Specify("Modified Expression, No New Dependencies", func(ctx SpecContext) {
			bases := []channel.Channel{{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			calcs := []channel.Channel{{
				Name:        UniqueChannelName(),
				DataType:    telem.Int64T,
				Virtual:     true,
				Leaseholder: node.KeyFree,
				Expression:  fmt.Sprintf("return %s * 2", bases[0].Name),
			}}
			w, sOutlet, cancel := open(
				ctx,
				nil,
				&bases,
				&calcs,
				channel.KeysFromChannels,
			)
			defer cancel()
			baseCh := bases[0]
			calcCh := calcs[0]
			Expect(
				w.Write(frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2))),
			).To(BeTrue())
			var res framer.StreamerResponse
			Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
			Expect(
				res.Frame.Get(calcCh.Key()).Series[0],
			).To(telem.MatchSeriesDataV[int64](2, 4))

			calcs[0].Expression = fmt.Sprintf("return %s * 3", bases[0].Name)
			Expect(channelWriter.Create(ctx, &calcs[0])).To(Succeed())

			Eventually(func(g Gomega) {
				Expect(
					w.Write(
						frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2)),
					),
				).To(BeTrue())
				g.Eventually(sOutlet.Outlet(), 200*time.Millisecond).
					Should(Receive(&res))
				g.Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
				g.Expect(res.Frame.Get(calcCh.Key()).Series[0]).
					To(telem.MatchSeriesDataV[int64](3, 6))
			}, 5*time.Second).Should(Succeed())

			Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
		})

		Specify("Modified Expression, New Dependencies", func(ctx SpecContext) {
			bases := []channel.Channel{{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}, {
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			calcs := []channel.Channel{{
				Name:        UniqueChannelName(),
				DataType:    telem.Int64T,
				Virtual:     true,
				Leaseholder: node.KeyFree,
				Expression:  fmt.Sprintf("return %s * 2", bases[0].Name),
			}}
			w, sOutlet, cancel := open(
				ctx,
				nil,
				&bases,
				&calcs,
				channel.KeysFromChannels,
			)
			defer cancel()
			baseCh := bases[0]
			baseCh2 := bases[1]
			calcCh := calcs[0]
			Expect(
				w.Write(frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2))),
			).To(BeTrue())
			var res framer.StreamerResponse
			Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
			Expect(
				res.Frame.Get(calcCh.Key()).Series[0],
			).To(telem.MatchSeriesDataV[int64](2, 4))

			calcs[0].Expression = fmt.Sprintf("return %s * 3", baseCh2.Name)
			Expect(channelWriter.Create(ctx, &calcs[0])).To(Succeed())

			Eventually(func(g Gomega) {
				Expect(
					w.Write(
						frame.NewUnary(baseCh2.Key(), telem.NewSeriesV[int64](1, 2)),
					),
				).To(BeTrue())
				g.Eventually(sOutlet.Outlet(), 200*time.Millisecond).
					Should(Receive(&res))
				g.Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
				g.Expect(res.Frame.Get(calcCh.Key()).Series[0]).
					To(telem.MatchSeriesDataV[int64](3, 6))
			}, 5*time.Second).Should(Succeed())

			Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
		})
	})
})
