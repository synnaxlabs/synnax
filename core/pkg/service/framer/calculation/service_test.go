// Copyright 2025 Synnax Labs, Inc.
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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation", Ordered, func() {
	var (
		c    *calculation.Service
		dist mock.Node
	)
	open := func(
		indexChannels,
		baseChannels,
		calculations *[]channel.Channel,

	) (*framer.Writer, confluence.Outlet[streamer.Response], context.CancelFunc) {
		if indexChannels != nil {
			Expect(dist.Channel.CreateMany(ctx, indexChannels)).To(Succeed())
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
		Expect(dist.Channel.CreateMany(ctx, baseChannels)).To(Succeed())
		Expect(dist.Channel.CreateMany(ctx, calculations)).To(Succeed())
		for _, calc := range *calculations {
			if !calc.IsIndex {
				MustSucceed(c.Request(ctx, calc.Key()))
			}
		}
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
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{
					Keys:        []channel.Key{(*calculations)[0].Key()},
					SendOpenAck: config.True(),
				},
			),
		)
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet()).Should(Receive())
		return w, sOutlet, func() {
			Expect(w.Close()).To(Succeed())
			cancel()
		}
	}

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		labelSvc := MustSucceed(label.OpenService(ctx, label.Config{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		}))
		statusSvc := MustSucceed(svcstatus.OpenService(ctx, svcstatus.ServiceConfig{
			DB:       dist.DB,
			Label:    labelSvc,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		}))
		arcSvc := MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
			Channel:  dist.Channel,
			Ontology: dist.Ontology,
			DB:       dist.DB,
			Framer:   dist.Framer,
			Status:   statusSvc,
			Signals:  dist.Signals,
		}))
		c = MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			Framer:            dist.Framer,
			Channel:           dist.Channel,
			ChannelObservable: dist.Channel.NewObservable(),
			Arc:               arcSvc,
		}))
	})

	AfterAll(func() {
		Expect(c.Close()).To(Succeed())
		Expect(dist.Close()).To(Succeed())
	})

	Describe("Changing Input Profiles", func() {
		Specify("Single Virtual Channel as Base", func() {
			bases := []channel.Channel{{
				Name:     "base",
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			calcs := []channel.Channel{{
				Name:        "calculated",
				DataType:    telem.Int64T,
				Virtual:     true,
				Leaseholder: cluster.Free,
				Expression:  "return base * 2",
			}}
			w, sOutlet, cancel := open(nil, &bases, &calcs)
			defer cancel()
			baseCh := bases[0]
			calcCh := calcs[0]
			MustSucceed(w.Write(core.UnaryFrame(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
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
			BeforeEach(func() {
				bases = []channel.Channel{
					{
						Name:     "base_1",
						DataType: telem.Int64T,
						Virtual:  true,
					},
					{
						Name:     "base_2",
						DataType: telem.Int64T,
						Virtual:  true,
					},
				}
				calcs = []channel.Channel{{
					Name:        "calculated",
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: cluster.Free,
					Expression:  "return base_1 * base_2",
				}}
			})
			Specify("Single Write with Data for Both Channels", func() {
				w, sOutlet, cancel := open(nil, &bases, &calcs)
				defer cancel()
				baseCH1 := bases[0]
				baseCh2 := bases[1]
				calcCh := calcs[0]
				MustSucceed(w.Write(core.MultiFrame(
					[]channel.Key{baseCH1.Key(), baseCh2.Key()},
					[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](2, 4)},
				)))
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
				Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
				Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](2, 8))
				Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
			})

			Specify("Two Writes with Data for Individual Channels", func() {
				w, sOutlet, cancel := open(nil, &bases, &calcs)
				defer cancel()
				baseCH1 := bases[0]
				baseCh2 := bases[1]
				calcCh := calcs[0]
				MustSucceed(w.Write(core.UnaryFrame(baseCH1.Key(), telem.NewSeriesV[int64](1, 2))))
				MustSucceed(w.Write(core.UnaryFrame(baseCh2.Key(), telem.NewSeriesV[int64](2, 4))))
				var res framer.StreamerResponse
				Eventually(sOutlet.Outlet()).Should(Receive(&res))
				Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calcCh.Key()}))
				Expect(res.Frame.Get(calcCh.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](2, 8))
				Consistently(sOutlet.Outlet(), 10*time.Millisecond).ShouldNot(Receive())
			})
		})

		Specify("Single Data Channel as Base", func() {
			var (
				indexes = []channel.Channel{{
					Name:     "base_time",
					DataType: telem.TimeStampT,
					IsIndex:  true,
				}}
				bases = []channel.Channel{{
					Name:     "base",
					DataType: telem.Int64T,
				}}
				calcs = []channel.Channel{{
					Name:        "calculated",
					DataType:    telem.Int64T,
					Virtual:     true,
					Leaseholder: cluster.Free,
					Expression:  "return base * 2",
				}}
			)
			w, sOutlet, cancel := open(&indexes, &bases, &calcs)
			defer cancel()
			idxCh := indexes[0]
			baseCh := bases[0]
			calcCh := calcs[0]
			MustSucceed(w.Write(core.MultiFrame(
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
			Specify("Shared Index", func() {
				var (
					indexes = []channel.Channel{{
						Name:     "base_time",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					}}
					bases = []channel.Channel{
						{
							Name:     "base_1",
							DataType: telem.Float32T,
						},
						{
							Name:     "base_2",
							DataType: telem.Float32T,
						},
					}
					calcs = []channel.Channel{{
						Name:        "calculated",
						DataType:    telem.Float32T,
						Virtual:     true,
						Leaseholder: cluster.Free,
						Expression:  "return base_1 * base_2",
					}}
				)
				w, sOutlet, cancel := open(&indexes, &bases, &calcs)
				defer cancel()
				idxCh := indexes[0]
				baseCh1 := bases[0]
				baseCh2 := bases[1]
				calcCh := calcs[0]
				MustSucceed(w.Write(core.MultiFrame(
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

			Specify("Unique Indexes", func() {
				var (
					indexes = []channel.Channel{
						{
							Name:     "base_1_time",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						{
							Name:     "base_2_time",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}
					bases = []channel.Channel{
						{
							Name:     "base_1",
							DataType: telem.Float32T,
						},
						{
							Name:     "base_2",
							DataType: telem.Float32T,
						},
					}
					calcs = []channel.Channel{{
						Name:        "calculated",
						DataType:    telem.Float32T,
						Virtual:     true,
						Leaseholder: cluster.Free,
						Expression:  "return base_1 * base_2",
					}}
				)
				w, sOutlet, cancel := open(&indexes, &bases, &calcs)
				defer cancel()
				var (
					idxCh1  = indexes[0]
					idxCh2  = indexes[1]
					baseCh1 = bases[0]
					baseCh2 = bases[1]
					calcCh  = calcs[0]
				)
				MustSucceed(w.Write(core.MultiFrame(
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

			Specify("Unique Indexes, Separate Writes", func() {
				var (
					indexes = []channel.Channel{
						{
							Name:     "base_1_time",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						{
							Name:     "base_2_time",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}
					bases = []channel.Channel{
						{
							Name:     "base_1",
							DataType: telem.Float32T,
						},
						{
							Name:     "base_2",
							DataType: telem.Float32T,
						},
					}
					calcs = []channel.Channel{{
						Name:        "calculated",
						DataType:    telem.Float32T,
						Virtual:     true,
						Leaseholder: cluster.Free,
						Expression:  "return base_1 * base_2",
					}}
				)
				w, sOutlet, cancel := open(&indexes, &bases, &calcs)
				defer cancel()
				var (
					idxCh1  = indexes[0]
					idxCh2  = indexes[1]
					baseCh1 = bases[0]
					baseCh2 = bases[1]
					calcCh  = calcs[0]
				)
				MustSucceed(w.Write(core.MultiFrame(
					[]channel.Key{idxCh1.Key(), baseCh1.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(3, 4),
						telem.NewSeriesV[float32](2, 4),
					},
				)))
				MustSucceed(w.Write(core.MultiFrame(
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
	})
})
