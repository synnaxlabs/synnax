// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package streamer_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"

	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Streamer", Ordered, func() {
	var (
		builder     = mock.NewCluster()
		dist        mock.Node
		streamerSvc *streamer.Service
	)
	BeforeAll(func() {
		dist = builder.Provision(ctx)
		calc := MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			Framer:            dist.Framer,
			Channel:           dist.Channel,
			ChannelObservable: dist.Channel.NewObservable(),
		}))
		streamerSvc = MustSucceed(streamer.NewService(streamer.ServiceConfig{
			DistFramer:  dist.Framer,
			Channel:     dist.Channel,
			Calculation: calc,
		}))
	})

	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
	})

	Describe("Happy Path", func() {
		It("Should stream data", func() {
			ch := &channel.Channel{
				Name:     "test",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))
			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{Keys: keys, SendOpenAck: true}))
			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(outlet.Outlet()).Should(Receive())
			time.Sleep(5 * time.Millisecond)
			writtenFr := core.UnaryFrame(ch.Key(), telem.NewSeriesV[float32](1, 2, 3))
			MustSucceed(w.Write(writtenFr))
			var res streamer.Response
			Eventually(outlet.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Frame).To(telem.MatchWrittenFrame(writtenFr.Frame))
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
			Expect(w.Close()).To(Succeed())
		})
	})

	Describe("Calculations", func() {
		var (
			indexCh *channel.Channel
			dataCh1 *channel.Channel
			dataCh2 *channel.Channel
		)
		BeforeEach(func() {
			indexCh = &channel.Channel{
				Name:     "Winston",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())
			dataCh1 = &channel.Channel{
				Name:       "Hobbs",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh1)).To(Succeed())
			dataCh2 = &channel.Channel{
				Name:       "Winston",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh2)).To(Succeed())

		})

		It("Should receive calculated values", func() {
			calculation := &channel.Channel{
				Name:       "Output",
				DataType:   telem.Float32T,
				Expression: "return Hobbs + Winston",
				Requires:   []channel.Key{dataCh1.Key(), dataCh2.Key()},
			}
			Expect(dist.Channel.Create(ctx, calculation)).To(Succeed())
			keys := []channel.Key{indexCh.Key(), dataCh1.Key(), dataCh2.Key()}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  keys,
			}))

			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:        []channel.Key{calculation.Key()},
				SendOpenAck: true,
			}))
			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(outlet.Outlet()).Should(Receive())
			writtenFr := core.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5),
					telem.NewSeriesV[float32](1, 2, 3, 4, 5),
					telem.NewSeriesV[float32](-1, -2, -3, -4, -5),
				},
			)
			MustSucceed(w.Write(writtenFr))
			var res streamer.Response
			Eventually(outlet.Outlet()).Should(Receive(&res))
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
			Expect(w.Close()).To(Succeed())
			Expect(res.Frame.Get(calculation.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](0, 0, 0, 0, 0))
		})

		It("Should allow the user to dynamically update the channels being calculated", func() {
			calculation := &channel.Channel{
				Name:       "Output",
				DataType:   telem.Float32T,
				Expression: "return Hobbs + Winston",
				Requires:   []channel.Key{dataCh1.Key(), dataCh2.Key()},
			}
			Expect(dist.Channel.Create(ctx, calculation)).To(Succeed())
			keys := []channel.Key{indexCh.Key(), dataCh1.Key(), dataCh2.Key()}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  keys,
			}))

			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:        []channel.Key{},
				SendOpenAck: true,
			}))
			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(outlet.Outlet()).Should(Receive())
			inlet.Inlet() <- streamer.Request{Keys: channel.Keys{calculation.Key()}}
			time.Sleep(5 * time.Millisecond)
			writtenFr := core.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5),
					telem.NewSeriesV[float32](1, 2, 3, 4, 5),
					telem.NewSeriesV[float32](-1, -2, -3, -4, -5),
				},
			)
			MustSucceed(w.Write(writtenFr))
			var res streamer.Response
			Eventually(outlet.Outlet()).Should(Receive(&res))
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
			Expect(w.Close()).To(Succeed())
			Expect(res.Frame.Get(calculation.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](0, 0, 0, 0, 0))
		})
	})

	Describe("Downsampling", func() {
		It("Should correctly downsample a factor of 2", func() {
			ch := &channel.Channel{
				Name:     "test",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))
			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:             keys,
				SendOpenAck:      true,
				DownsampleFactor: 2,
			}))
			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(outlet.Outlet()).Should(Receive())
			writtenFr := core.UnaryFrame(ch.Key(), telem.NewSeriesV[float32](1, 2, 3, 4))
			MustSucceed(w.Write(writtenFr))
			var res streamer.Response
			Eventually(outlet.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Get(ch.Key()).Series[0]).To(telem.MatchSeriesData(writtenFr.Get(ch.Key()).Series[0].Downsample(2)))
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
			Expect(w.Close()).To(Succeed())
		})

		It("Should handle invalid downsampling factors", func() {
			ch := &channel.Channel{
				Name:     "test",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}

			_, err := streamerSvc.New(ctx, streamer.Config{
				Keys:             keys,
				SendOpenAck:      true,
				DownsampleFactor: -2,
			})
			Expect(err).To(MatchError(ContainSubstring("downsample_factor: must be greater than or equal to 0")))
		})

		It("Should correctly combine downsampling with calculations", func() {
			indexCh := &channel.Channel{
				Name:     "index",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())

			dataCh1 := &channel.Channel{
				Name:       "data1",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh1)).To(Succeed())

			dataCh2 := &channel.Channel{
				Name:       "data2",
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh2)).To(Succeed())

			calculation := &channel.Channel{
				Name:       "sum",
				DataType:   telem.Float32T,
				Expression: "return data1 + data2",
				Requires:   []channel.Key{dataCh1.Key(), dataCh2.Key()},
			}
			Expect(dist.Channel.Create(ctx, calculation)).To(Succeed())

			keys := []channel.Key{indexCh.Key(), dataCh1.Key(), dataCh2.Key()}
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  keys,
			}))

			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:             []channel.Key{calculation.Key()},
				SendOpenAck:      true,
				DownsampleFactor: 2,
			}))

			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(outlet.Outlet()).Should(Receive())

			writtenFr := core.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8),
					telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8),
					telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6, 7, 8),
				},
			)
			MustSucceed(w.Write(writtenFr))

			var res streamer.Response
			Eventually(outlet.Outlet()).Should(Receive(&res))

			expectedValues := []float32{2, 6, 10, 14}
			Expect(res.Frame.Get(calculation.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](expectedValues...))

			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
			Expect(w.Close()).To(Succeed())
		})
	})
})
