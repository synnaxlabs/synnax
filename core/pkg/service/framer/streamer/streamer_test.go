// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package streamer_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Streamer", Ordered, func() {
	var (
		node          mock.Node
		channelSvc    *channel.Service
		channelWriter channel.Writer
		streamerSvc   *streamer.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node = mock.NewNode(ctx)
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       node.DB,
			Group:    groupSvc,
			Ontology: otg,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      node.Channel,
			DB:           node.DB,
			HostProvider: node.Cluster,
			Ontology:     otg,
			Group:        groupSvc,
			Search:       searchIdx,
			Status:       statusSvc,
		}))
		channelWriter = channelSvc.NewWriter(nil)
		writerSvc := MustSucceed(writer.NewService(writer.ServiceConfig{
			Framer:  node.Framer,
			Channel: channelSvc,
		}))
		calc := MustOpen(calculation.OpenService(ctx, calculation.ServiceConfig{
			Framer:  node.Framer,
			Writer:  writerSvc,
			Channel: channelSvc,
			Status:  statusSvc,
		}))
		streamerSvc = MustSucceed(streamer.NewService(streamer.ServiceConfig{
			Framer:      node.Framer,
			Channel:     channelSvc,
			Calculation: calc,
		}))
	})

	Describe("Happy Path", func() {
		It("Should stream data", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     "test",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))
			s := MustSucceed(
				streamerSvc.New(ctx, streamer.Config{Keys: keys, SendOpenAck: true}),
			)
			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(outlet.Outlet()).Should(Receive())
			writtenFr := frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](1, 2, 3))
			MustSucceed(w.Write(writtenFr))
			var res streamer.Response
			Eventually(outlet.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Frame).To(telem.MatchWrittenFrame(writtenFr.Frame))
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})
	})

	Describe("Calculations", func() {
		var (
			indexCh *channel.Channel
			dataCh1 *channel.Channel
			dataCh2 *channel.Channel
		)
		BeforeEach(func(ctx SpecContext) {
			indexCh = &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(channelWriter.Create(ctx, indexCh)).To(Succeed())
			dataCh1 = &channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(channelWriter.Create(ctx, dataCh1)).To(Succeed())
			dataCh2 = &channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(channelWriter.Create(ctx, dataCh2)).To(Succeed())
		})

		It("Should receive calculated values", func(ctx SpecContext) {
			calculation := &channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Expression: fmt.Sprintf("return %s + %s", dataCh1.Name, dataCh2.Name),
			}
			Expect(channelWriter.Create(ctx, calculation)).To(Succeed())
			keys := []channel.Key{indexCh.Key(), dataCh1.Key(), dataCh2.Key()}
			w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
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
			writtenFr := frame.NewMulti(
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
			Expect(
				res.Frame.Get(calculation.Key()).Series[0],
			).To(telem.MatchSeriesDataV[float32](0, 0, 0, 0, 0))
		})

		It(
			"Should allow the user to dynamically update the channels being calculated",
			func(ctx SpecContext) {
				calculation := &channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Float32T,
					Expression: fmt.Sprintf(
						"return %s + %s",
						dataCh1.Name,
						dataCh2.Name,
					),
				}
				Expect(channelWriter.Create(ctx, calculation)).To(Succeed())
				keys := []channel.Key{indexCh.Key(), dataCh1.Key(), dataCh2.Key()}
				w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
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
				time.Sleep(500 * time.Millisecond)
				writtenFr := frame.NewMulti(
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
				Expect(
					res.Frame.Get(calculation.Key()).Series[0],
				).To(telem.MatchSeriesDataV[float32](0, 0, 0, 0, 0))
			},
		)

		It(
			"Should compute when inputs with different indexes arrive in separate frames",
			func(ctx SpecContext) {
				idxA := &channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.TimeStampT,
					IsIndex:  true,
				}
				Expect(channelWriter.Create(ctx, idxA)).To(Succeed())
				idxB := &channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.TimeStampT,
					IsIndex:  true,
				}
				Expect(channelWriter.Create(ctx, idxB)).To(Succeed())
				dataA := &channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					LocalIndex: idxA.LocalKey,
				}
				Expect(channelWriter.Create(ctx, dataA)).To(Succeed())
				dataB := &channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					LocalIndex: idxB.LocalKey,
				}
				Expect(channelWriter.Create(ctx, dataB)).To(Succeed())

				calculation := &channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					Expression: fmt.Sprintf("return %s + %s", dataA.Name, dataB.Name),
				}
				Expect(channelWriter.Create(ctx, calculation)).To(Succeed())

				keysA := []channel.Key{idxA.Key(), dataA.Key()}
				wA := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
					Start: telem.SecondTS,
					Keys:  keysA,
				}))
				keysB := []channel.Key{idxB.Key(), dataB.Key()}
				wB := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
					Start: telem.SecondTS,
					Keys:  keysB,
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

				// Writer A sends [idxA, dataA] — not enough inputs to compute
				MustSucceed(wA.Write(frame.NewMulti(
					keysA,
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2, 3),
						telem.NewSeriesV[float32](10, 20, 30),
					},
				)))

				// Writer B sends [idxB, dataB] — now both inputs available
				MustSucceed(wB.Write(frame.NewMulti(
					keysB,
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2, 3),
						telem.NewSeriesV[float32](1, 2, 3),
					},
				)))

				var res streamer.Response
				Eventually(outlet.Outlet()).Should(Receive(&res))
				Expect(res.Frame.Get(calculation.Key()).Series[0]).To(
					telem.MatchSeriesDataV[float32](11, 22, 33),
				)

				inlet.Close()
				Eventually(outlet.Outlet()).Should(BeClosed())
			},
		)
	})

	Describe("Downsampling", func() {
		It("Should correctly downsample a factor of 2", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
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
			writtenFr := frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](1, 2, 3, 4))
			MustSucceed(w.Write(writtenFr))
			var res streamer.Response
			Eventually(outlet.Outlet()).Should(Receive(&res))
			Expect(
				res.Frame.Get(ch.Key()).Series[0],
			).To(telem.MatchSeriesData(writtenFr.Get(ch.Key()).Series[0].Downsample(2)))
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})

		It("Should handle invalid downsampling factors", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}

			_, err := streamerSvc.New(ctx, streamer.Config{
				Keys:             keys,
				SendOpenAck:      true,
				DownsampleFactor: -2,
			})
			Expect(
				err,
			).To(MatchError(ContainSubstring("downsample_factor: must be greater than or equal to 0")))
		})

		It(
			"Should correctly combine downsampling with calculations",
			func(ctx SpecContext) {
				indexCh := &channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.TimeStampT,
					IsIndex:  true,
				}
				Expect(channelWriter.Create(ctx, indexCh)).To(Succeed())

				dataCh1 := &channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					LocalIndex: indexCh.LocalKey,
				}
				Expect(channelWriter.Create(ctx, dataCh1)).To(Succeed())

				dataCh2 := &channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					LocalIndex: indexCh.LocalKey,
				}
				Expect(channelWriter.Create(ctx, dataCh2)).To(Succeed())

				calculation := &channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Float32T,
					Expression: fmt.Sprintf(
						"return %s + %s",
						dataCh1.Name,
						dataCh2.Name,
					),
				}
				Expect(channelWriter.Create(ctx, calculation)).To(Succeed())

				keys := []channel.Key{indexCh.Key(), dataCh1.Key(), dataCh2.Key()}
				w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
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

				writtenFr := frame.NewMulti(
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
				Expect(
					res.Frame.Get(calculation.Key()).Series[0],
				).To(telem.MatchSeriesDataV(expectedValues...))

				inlet.Close()
				Eventually(outlet.Outlet()).Should(BeClosed())
			},
		)
	})
	Describe("Throttling", func() {
		It("Should accumulate and throttle frames", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))

			throttleRate := 5 * telem.Hertz
			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:         keys,
				SendOpenAck:  true,
				ThrottleRate: throttleRate,
			}))

			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())

			Eventually(outlet.Outlet()).Should(Receive())

			writtenFr := frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](1, 2, 3))
			MustSucceed(w.Write(writtenFr))

			var res streamer.Response
			Eventually(outlet.Outlet(), 500*time.Millisecond).Should(Receive(&res))
			Expect(res.Frame.Len()).To(BeNumerically(">", 0))

			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})

		It("Should not throttle when rate is 0", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))

			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:         keys,
				SendOpenAck:  true,
				ThrottleRate: 0,
			}))

			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())

			Eventually(outlet.Outlet()).Should(Receive())

			writtenFr := frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](1, 2, 3))
			MustSucceed(w.Write(writtenFr))

			var res streamer.Response
			Eventually(outlet.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Frame).To(telem.MatchWrittenFrame(writtenFr.Frame))

			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})

		It("Should combine throttling and downsampling", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))

			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:             keys,
				SendOpenAck:      true,
				DownsampleFactor: 2,
				ThrottleRate:     5 * telem.Hertz,
			}))

			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())

			Eventually(outlet.Outlet()).Should(Receive())

			writtenFr := frame.NewUnary(
				ch.Key(),
				telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6),
			)
			MustSucceed(w.Write(writtenFr))

			var res streamer.Response
			Eventually(outlet.Outlet(), 500*time.Millisecond).Should(Receive(&res))
			Expect(res.Frame.Len()).To(BeNumerically(">", 0))

			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})
	})

	Describe("Throttling", func() {
		It("Should accumulate and throttle frames", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))

			throttleRate := 5 * telem.Hertz
			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:         keys,
				SendOpenAck:  true,
				ThrottleRate: throttleRate,
			}))

			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())

			Eventually(outlet.Outlet()).Should(Receive())

			writtenFr := frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](1, 2, 3))
			MustSucceed(w.Write(writtenFr))

			var res streamer.Response
			Eventually(outlet.Outlet(), 500*time.Millisecond).Should(Receive(&res))
			Expect(res.Frame.Len()).To(BeNumerically(">", 0))

			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})

		It("Should not throttle when rate is 0", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))

			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:         keys,
				SendOpenAck:  true,
				ThrottleRate: 0,
			}))

			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())

			Eventually(outlet.Outlet()).Should(Receive())

			writtenFr := frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](1, 2, 3))
			MustSucceed(w.Write(writtenFr))

			var res streamer.Response
			Eventually(outlet.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Frame).To(telem.MatchWrittenFrame(writtenFr.Frame))

			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})

		It("Should combine throttling and downsampling", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))

			s := MustSucceed(streamerSvc.New(ctx, streamer.Config{
				Keys:             keys,
				SendOpenAck:      true,
				DownsampleFactor: 2,
				ThrottleRate:     5 * telem.Hertz,
			}))

			sCtx, cancel := signal.Isolated()
			inlet, outlet := confluence.Attach(s)
			defer cancel()
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())

			Eventually(outlet.Outlet()).Should(Receive())

			writtenFr := frame.NewUnary(
				ch.Key(),
				telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6),
			)
			MustSucceed(w.Write(writtenFr))

			var res streamer.Response
			Eventually(outlet.Outlet(), 500*time.Millisecond).Should(Receive(&res))
			Expect(res.Frame.Len()).To(BeNumerically(">", 0))

			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})
	})
})
