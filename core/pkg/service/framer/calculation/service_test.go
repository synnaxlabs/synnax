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
	"go/types"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation", Ordered, func() {
	var (
		c             *calculation.Service
		dist          mock.Node
		statusSvc     *status.Service
		channelSvc    *channel.Service
		channelWriter channel.Writer
	)
	open := func(
		ctx context.Context,
		indexChannels,
		baseChannels,
		calculations *[]channel.Channel,
		streamKeys func([]channel.Channel) channel.Keys,
	) (*framer.Writer, confluence.Outlet[streamer.Response], context.CancelFunc) {
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
		filtered := lo.Filter(*calculations, func(item channel.Channel, index int) bool {
			return !item.IsIndex
		})
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{Keys: streamKeys(filtered), SendOpenAck: new(true)},
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
		dist = mock.NewNode(ctx)
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: dist.DB}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       dist.DB,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       dist.DB,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       dist.DB,
			Group:    groupSvc,
			Ontology: otg,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      dist.Channel,
			DB:           dist.DB,
			HostResolver: dist.Cluster,
			Ontology:     otg,
			Group:        groupSvc,
			Search:       searchIdx,
			Status:       statusSvc,
		}))
		channelWriter = channelSvc.NewWriter(nil)
		c = MustOpen(calculation.OpenService(ctx, calculation.ServiceConfig{
			Framer:  dist.Framer,
			Channel: channelSvc,
			Status:  statusSvc,
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
			w, sOutlet, cancel := open(ctx, nil, &bases, &calcs, channel.KeysFromChannels)
			defer cancel()
			baseCh := bases[0]
			calcCh := calcs[0]
			MustSucceed(w.Write(frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
			var res framer.StreamerResponse
			Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
			Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](2, 4))
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
				calcs = []channel.Channel{{
					Name:        UniqueChannelName(),
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: node.KeyFree,
					Expression:  fmt.Sprintf("return %s * %s", bases[0].Name, bases[1].Name),
				}}
			})
			Specify("Single Write with Data for Both Channels", func(ctx SpecContext) {
				w, sOutlet, cancel := open(ctx, nil, &bases, &calcs, channel.KeysFromChannels)
				defer cancel()
				baseCh1 := bases[0]
				baseCh2 := bases[1]
				calcCh := calcs[0]
				MustSucceed(w.Write(frame.NewMulti(
					[]channel.Key{baseCh1.Key(), baseCh2.Key()},
					[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](2, 4)},
				)))
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
				Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](2, 8))
				Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
			})

			Specify("Two Writes with Data for Individual Channels", func(ctx SpecContext) {
				w, sOutlet, cancel := open(ctx, nil, &bases, &calcs, channel.KeysFromChannels)
				defer cancel()
				baseCh1 := bases[0]
				baseCh2 := bases[1]
				calcCh := calcs[0]
				MustSucceed(w.Write(frame.NewUnary(baseCh1.Key(), telem.NewSeriesV[int64](1, 2))))
				MustSucceed(w.Write(frame.NewUnary(baseCh2.Key(), telem.NewSeriesV[int64](2, 4))))
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet()).Should(Receive(&res))
				Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
				Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](2, 8))
				Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
			})
		})

		Specify("Single Data Channel as Base", func(ctx SpecContext) {
			var (
				indexes = []channel.Channel{{
					Name:     UniqueChannelName(),
					DataType: telem.TimeStampT,
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
			w, sOutlet, cancel := open(ctx, &indexes, &bases, &calcs, channel.KeysFromChannels)
			defer cancel()
			idxCh := indexes[0]
			baseCh := bases[0]
			calcCh := calcs[0]
			MustSucceed(w.Write(frame.NewMulti(
				[]channel.Key{idxCh.Key(), baseCh.Key()},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2),
					telem.NewSeriesV[int64](1, 2),
				},
			)))
			var res framer.StreamerResponse
			Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
			Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](2, 4))
			Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
		})

		Describe("Multiple Data Channels as Base", func() {
			Specify("Shared Index", func(ctx SpecContext) {
				var (
					indexes = []channel.Channel{{
						Name:     UniqueChannelName(),
						DataType: telem.TimeStampT,
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
					calcs = []channel.Channel{{
						Name:        UniqueChannelName(),
						DataType:    telem.Float32T,
						Virtual:     true,
						Leaseholder: node.KeyFree,
						Expression:  fmt.Sprintf("return %s * %s", bases[0].Name, bases[1].Name),
					}}
				)
				w, sOutlet, cancel := open(ctx, &indexes, &bases, &calcs, channel.KeysFromChannels)
				defer cancel()
				idxCh := indexes[0]
				baseCh1 := bases[0]
				baseCh2 := bases[1]
				calcCh := calcs[0]
				MustSucceed(w.Write(frame.NewMulti(
					[]channel.Key{idxCh.Key(), baseCh1.Key(), baseCh2.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2),
						telem.NewSeriesV[float32](1, 2),
						telem.NewSeriesV[float32](2, 4),
					},
				)))
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](2, 8))
			})

			Specify("Unique Indexes", func(ctx SpecContext) {
				var (
					indexes = []channel.Channel{
						{
							Name:     UniqueChannelName(),
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						{
							Name:     UniqueChannelName(),
							DataType: telem.TimeStampT,
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
					calcs = []channel.Channel{{
						Name:        UniqueChannelName(),
						DataType:    telem.Float32T,
						Virtual:     true,
						Leaseholder: node.KeyFree,
						Expression:  fmt.Sprintf("return %s * %s", bases[0].Name, bases[1].Name),
					}}
				)
				w, sOutlet, cancel := open(ctx, &indexes, &bases, &calcs, channel.KeysFromChannels)
				defer cancel()
				var (
					idxCh1  = indexes[0]
					idxCh2  = indexes[1]
					baseCh1 = bases[0]
					baseCh2 = bases[1]
					calcCh  = calcs[0]
				)
				MustSucceed(w.Write(frame.NewMulti(
					[]channel.Key{idxCh1.Key(), idxCh2.Key(), baseCh1.Key(), baseCh2.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2),
						telem.NewSeriesSecondsTSV(3, 4),
						telem.NewSeriesV[float32](1, 2),
						telem.NewSeriesV[float32](2, 4),
					},
				)))
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](2, 8))
			})

			Specify("Unique Indexes, Separate Writes", func(ctx SpecContext) {
				var (
					indexes = []channel.Channel{
						{
							Name:     UniqueChannelName(),
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						{
							Name:     UniqueChannelName(),
							DataType: telem.TimeStampT,
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
					calcs = []channel.Channel{{
						Name:        UniqueChannelName(),
						DataType:    telem.Float32T,
						Virtual:     true,
						Leaseholder: node.KeyFree,
						Expression:  fmt.Sprintf("return %s * %s", bases[0].Name, bases[1].Name),
					}}
				)
				w, sOutlet, cancel := open(ctx, &indexes, &bases, &calcs, channel.KeysFromChannels)
				defer cancel()
				var (
					idxCh1  = indexes[0]
					idxCh2  = indexes[1]
					baseCh1 = bases[0]
					baseCh2 = bases[1]
					calcCh  = calcs[0]
				)
				MustSucceed(w.Write(frame.NewMulti(
					[]channel.Key{idxCh1.Key(), baseCh1.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(3, 4),
						telem.NewSeriesV[float32](2, 4),
					},
				)))
				MustSucceed(w.Write(frame.NewMulti(
					[]channel.Key{idxCh2.Key(), baseCh2.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2),
						telem.NewSeriesV[float32](2, 4),
					},
				)))
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](4, 16))
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
				w, sOutlet, cancel := open(ctx, nil, &bases, &calcs, channel.KeysFromChannels)
				defer cancel()
				baseCh := bases[0]
				calcCh := calcs[0]
				calc2Ch := calcs[1]
				MustSucceed(w.Write(frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))

				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key(), calc2Ch.Key()}))
				Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](2, 4))
				Expect(res.Frame.Get(calc2Ch.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](4, 8))
			})

			Specify("Calculations of Calculations, Base Not Requested", func(ctx SpecContext) {
				w, sOutlet, cancel := open(ctx, nil, &bases, &calcs, func(calcs []channel.Channel) channel.Keys {
					return []channel.Key{calcs[1].Key()}
				})
				defer cancel()
				baseCh := bases[0]
				calc2Ch := calcs[1]
				MustSucceed(w.Write(frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))

				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calc2Ch.Key()}))
				Expect(res.Frame.Get(calc2Ch.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](4, 8))
			})
		})
	})

	Describe("Calculation Status", func() {
		Specify("Should persist error status on invalid expression request", func(ctx SpecContext) {
			base := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true}
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
				ctx, base.Key(), false),
			).To(Succeed())
			rm := c.OpenRequestManager()
			Expect(
				rm.Set(ctx, channel.KeysFromChannels([]channel.Channel{calc})),
			).To(Succeed())
			var st calculation.Status
			statusKey := calc.OntologyID().String()
			Expect(status.NewRetrieve[types.Nil](statusSvc).
				Where(status.MatchKeys[types.Nil](statusKey)).
				Entry(&st).
				Exec(ctx, nil)).To(Succeed())
			Expect(st.Variant).To(Equal(status.VariantError))
			Expect(rm.Close(ctx)).To(Succeed())
		})
		Specify("Should persist error status on calculation update failure", func(ctx SpecContext) {
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
			// record. The runtime reacts to writes of the calc channel by recompiling
			// it, which now fails because the dependency no longer resolves. Rename
			// (unlike Create) persists the record without re-analyzing the expression,
			// so the failure surfaces asynchronously as an error status rather than
			// synchronously.
			Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())
			Expect(channelWriter.Rename(
				ctx, calc.Key(), UniqueChannelName(), false),
			).To(Succeed())
			var st calculation.Status
			statusKey := calc.OntologyID().String()
			Eventually(func(g Gomega) {
				g.Expect(status.NewRetrieve[types.Nil](statusSvc).
					Where(status.MatchKeys[types.Nil](statusKey)).
					Entry(&st).
					Exec(ctx, nil)).To(Succeed())
				g.Expect(st.Variant).To(Equal(status.VariantError))
			}).Should(Succeed())
			Expect(rm.Close(ctx)).To(Succeed())
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
			Expect(status.NewRetrieve[types.Nil](statusSvc).
				Where(status.MatchKeys[types.Nil](expectedKey)).
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
			w, sOutlet, cancel := open(ctx, nil, &bases, &calcs, channel.KeysFromChannels)
			defer cancel()
			baseCh := bases[0]
			calcCh := calcs[0]
			MustSucceed(w.Write(frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
			var res framer.StreamerResponse
			Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
			Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](2, 4))

			calcs[0].Expression = fmt.Sprintf("return %s * 3", bases[0].Name)
			Expect(channelWriter.Create(ctx, &calcs[0])).To(Succeed())

			Eventually(func(g Gomega) {
				MustSucceed(w.Write(frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
				g.Eventually(sOutlet.Outlet(), 200*time.Millisecond).Should(Receive(&res))
				g.Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
				g.Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](3, 6))
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
			w, sOutlet, cancel := open(ctx, nil, &bases, &calcs, channel.KeysFromChannels)
			defer cancel()
			baseCh := bases[0]
			baseCh2 := bases[1]
			calcCh := calcs[0]
			MustSucceed(w.Write(frame.NewUnary(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
			var res framer.StreamerResponse
			Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
			Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](2, 4))

			calcs[0].Expression = fmt.Sprintf("return %s * 3", baseCh2.Name)
			Expect(channelWriter.Create(ctx, &calcs[0])).To(Succeed())

			Eventually(func(g Gomega) {
				MustSucceed(w.Write(frame.NewUnary(baseCh2.Key(), telem.NewSeriesV[int64](1, 2))))
				g.Eventually(sOutlet.Outlet(), 200*time.Millisecond).Should(Receive(&res))
				g.Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
				g.Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](3, 6))
			}, 5*time.Second).Should(Succeed())

			Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
		})
	})
})
