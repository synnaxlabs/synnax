// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	createIndexed := func(ctx SpecContext) (channel.Channel, channel.Channel) {
		idxCh := channel.Channel{
			Name:     testutil.RandomName(),
			DataType: telem.TimeStampT,
			IsIndex:  true,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, &idxCh)).To(Succeed())
		dataCh := channel.Channel{
			Name:       testutil.RandomName(),
			DataType:   telem.Float32T,
			LocalIndex: idxCh.LocalKey,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, &dataCh)).To(Succeed())
		return idxCh, dataCh
	}

	write := func(ctx SpecContext, idxCh, dataCh channel.Channel) {
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Start: telem.SecondTS,
			Keys:  []channel.Key{idxCh.Key(), dataCh.Key()},
		}))
		MustSucceed(w.Write(frame.NewMulti(
			[]channel.Key{idxCh.Key(), dataCh.Key()},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(1, 2, 3),
				telem.NewSeriesV[float32](1, 2, 3),
			},
		)))
		MustSucceed(w.Commit())
		Expect(w.Close()).To(Succeed())
	}

	Describe("ServiceConfig", func() {
		Describe("Validate", func() {
			It("Should return nil for a fully specified configuration", func() {
				Expect(validCfg.Validate()).To(Succeed())
			})
			DescribeTable("Should return an error when a required field is nil",
				func(mutate func(*framer.ServiceConfig), field string) {
					cfg := validCfg
					mutate(&cfg)
					Expect(cfg.Validate()).
						To(MatchError(ContainSubstring(field + ": must be non-nil")))
				},
				Entry("framer", func(c *framer.ServiceConfig) {
					c.Framer = nil
				}, "framer"),
				Entry("channel", func(c *framer.ServiceConfig) {
					c.Channel = nil
				}, "channel"),
				Entry("status", func(c *framer.ServiceConfig) {
					c.Status = nil
				}, "status"),
				Entry("host_resolver", func(c *framer.ServiceConfig) {
					c.HostResolver = nil
				}, "host_resolver"),
			)
		})

		Describe("Override", func() {
			It("Should retain base values when the override is empty", func() {
				res := validCfg.Override(framer.ServiceConfig{})
				Expect(res.Framer).To(Equal(node.Framer))
				Expect(res.Channel).To(Equal(channelSvc))
				Expect(res.Status).To(Equal(statusSvc))
				Expect(res.HostResolver).To(Equal(node.Cluster))
			})
			It("Should replace base values with non-nil overrides", func() {
				res := framer.ServiceConfig{}.Override(validCfg)
				Expect(res.Framer).To(Equal(node.Framer))
				Expect(res.Channel).To(Equal(channelSvc))
				Expect(res.Status).To(Equal(statusSvc))
				Expect(res.HostResolver).To(Equal(node.Cluster))
			})
			It("Should override zero-value instrumentation", func() {
				res := framer.ServiceConfig{}.Override(framer.ServiceConfig{
					Instrumentation: alamos.New("framer-test"),
				})
				Expect(res.Instrumentation.IsZero()).To(BeFalse())
			})
		})
	})

	Describe("OpenService", func() {
		It("Should open and close a service from a valid configuration", func(ctx SpecContext) {
			svc := MustSucceed(framer.OpenService(
				ctx, newFramerConfig(ctx, mock.NewNode(ctx)),
			))
			Expect(svc.Close()).To(Succeed())
		})
		It("Should return an error for an invalid configuration", func(ctx SpecContext) {
			Expect(framer.OpenService(ctx, framer.ServiceConfig{})).
				Error().To(MatchError(ContainSubstring("must be non-nil")))
		})
	})

	Describe("Control update channel configuration", func() {
		controlChannelName := func(n mock.Node) string {
			return fmt.Sprintf("sy_node_%v_control", n.Cluster.HostKey())
		}
		It("Should create the host node's control update channel on open", func(ctx SpecContext) {
			var controlChannels []channel.Channel
			Expect(channelSvc.
				NewRetrieve().
				Where(channel.MatchNames(controlChannelName(node))).
				Entries(&controlChannels).
				Exec(ctx, nil)).To(Succeed())
			Expect(controlChannels).To(HaveLen(1))
			controlCh := controlChannels[0]
			Expect(controlCh.Virtual).To(BeTrue())
			Expect(controlCh.Internal).To(BeTrue())
			Expect(controlCh.DataType).To(Equal(telem.StringT))
			Expect(controlCh.Leaseholder).To(Equal(node.Cluster.HostKey()))
		})
		It("Should reuse an existing control update channel rather than recreating it", func(ctx SpecContext) {
			n := mock.NewNode(ctx)
			cfg := newFramerConfig(ctx, n)
			name := controlChannelName(n)
			existing := channel.Channel{
				Name:        name,
				Leaseholder: n.Cluster.HostKey(),
				Virtual:     true,
				DataType:    telem.StringT,
				Internal:    true,
			}
			Expect(cfg.Channel.NewWriter(nil).Create(ctx, &existing)).To(Succeed())
			DeferClose(MustSucceed(framer.OpenService(ctx, cfg)))
			Expect(cfg.Channel.
				NewRetrieve().
				Where(channel.MatchNames(name)).
				Count(ctx, nil),
			).To(Equal(1))
		})
	})

	Describe("OpenWriter", func() {
		It("Should open a writer and persist a frame", func(ctx SpecContext) {
			idxCh, dataCh := createIndexed(ctx)
			w := MustOpen(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  []channel.Key{idxCh.Key(), dataCh.Key()},
			}))
			MustSucceed(w.Write(frame.NewMulti(
				[]channel.Key{idxCh.Key(), dataCh.Key()},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3),
					telem.NewSeriesV[float32](1, 2, 3),
				},
			)))
			MustSucceed(w.Commit())
		})
	})

	Describe("NewStreamWriter", func() {
		It("Should open a stream writer for resolved channels", func(ctx SpecContext) {
			idxCh, dataCh := createIndexed(ctx)
			s := MustSucceed(framerSvc.NewStreamWriter(ctx, framer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  []channel.Key{idxCh.Key(), dataCh.Key()},
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			inlet, outlet := confluence.Attach(s)
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			inlet.Inlet() <- framer.WriterRequest{
				Command: writer.CommandWrite,
				Frame: frame.NewMulti(
					[]channel.Key{idxCh.Key(), dataCh.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2, 3),
						telem.NewSeriesV[float32](1, 2, 3),
					},
				),
			}
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})
	})

	Describe("OpenIterator", func() {
		It("Should iterate over previously written data", func(ctx SpecContext) {
			idxCh, dataCh := createIndexed(ctx)
			write(ctx, idxCh, dataCh)
			iter := MustSucceed(framerSvc.OpenIterator(ctx, framer.IteratorConfig{
				Keys:   []channel.Key{idxCh.Key(), dataCh.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
			Expect(iter.Value().Get(idxCh.Key()).Series[0]).
				To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(1, 2, 3)))
			Expect(iter.Close()).To(Succeed())
		})
	})

	Describe("NewStreamIterator", func() {
		It("Should stream iteration responses for resolved channels", func(ctx SpecContext) {
			idxCh, dataCh := createIndexed(ctx)
			write(ctx, idxCh, dataCh)
			s := MustSucceed(framerSvc.NewStreamIterator(ctx, framer.IteratorConfig{
				Keys:   []channel.Key{idxCh.Key(), dataCh.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			inlet, outlet := confluence.Attach(s)
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			inlet.Inlet() <- framer.IteratorRequest{Command: iterator.CommandSeekFirst}
			var res framer.IteratorResponse
			Eventually(outlet.Outlet()).Should(Receive(&res))
			Expect(res.Ack).To(BeTrue())
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})
	})

	Describe("NewStreamer", func() {
		It("Should stream live writes to resolved channels", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     testutil.RandomName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			keys := []channel.Key{ch.Key()}
			w := MustOpen(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  keys,
			}))
			s := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
				Keys:        keys,
				SendOpenAck: true,
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			inlet, outlet := confluence.Attach(s)
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			Eventually(outlet.Outlet()).Should(Receive())
			writtenFr := frame.NewUnary(ch.Key(), telem.NewSeriesV[float32](1, 2, 3))
			MustSucceed(w.Write(writtenFr))
			var res framer.StreamerResponse
			Eventually(outlet.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Frame).To(telem.MatchWrittenFrame(writtenFr.Frame))
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})
	})

	Describe("DeleteTimeRange", func() {
		It("Should delete samples within the given range", func(ctx SpecContext) {
			idxCh, dataCh := createIndexed(ctx)
			write(ctx, idxCh, dataCh)
			Expect(framerSvc.DeleteTimeRange(
				ctx,
				channel.Keys{idxCh.Key(), dataCh.Key()},
				telem.TimeRangeMax,
			)).To(Succeed())
			iter := MustSucceed(framerSvc.OpenIterator(ctx, framer.IteratorConfig{
				Keys:   []channel.Key{idxCh.Key(), dataCh.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})
	})
})
