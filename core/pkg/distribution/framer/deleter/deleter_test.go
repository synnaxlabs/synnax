// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deleter_test

import (
	"context"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Deleter", Ordered, func() {
	scenarios := []func(context.Context) scenario{
		gatewayOnlyScenario,
		peerOnlyScenario,
	}
	for _, createScenario := range scenarios {
		var s scenario
		BeforeAll(func() { s = createScenario(context.Background()) })
		AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })
		Describe("Happy Path", func() {
			Context(s.name+" - Happy Path", func() {
				var i *iterator.Iterator
				BeforeEach(func(ctx SpecContext) {
					w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
						Keys:  s.keys,
						Start: 10 * telem.SecondTS,
					}))
					Expect(w.Write(frame.NewMulti(
						s.keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 11, 12),
							telem.NewSeriesSecondsTSV(10, 11, 12),
							telem.NewSeriesSecondsTSV(10, 11, 12),
						},
					))).To(BeTrue())
					Expect(MustSucceed(w.Commit())).To(Equal(telem.SecondTS*12 + 1))
					Expect(w.Close()).To(Succeed())

					// Use context.Background() because the iterator must survive
					// beyond BeforeEach into It/AfterEach. SpecContext is cancelled
					// when BeforeEach exits, which kills peer streams and deadlocks.
					i = MustSucceed(s.dist.Framer.OpenIterator(context.Background(), iterator.Config{
						Keys:   s.keys,
						Bounds: telem.TimeRangeMax,
					}))
				})
				AfterEach(func(ctx SpecContext) {
					Expect(s.dist.Framer.DeleteTimeRange(ctx, s.keys, telem.TimeRangeMax)).To(Succeed())
					Expect(i.Close()).To(Succeed())
				})

				It("Should delete one channel by key", func(ctx SpecContext) {
					Expect(s.dist.Framer.DeleteTimeRange(ctx, s.keys[:1], (10 * telem.SecondTS).Range(12*telem.SecondTS))).To(Succeed())
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(s.keys[0]).Len()).To(Equal(int64(1)))
					Expect(i.Value().Get(s.keys[0]).TimeRange()).To(Equal((12 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
				})
				It("Should delete many channels by keys", func(ctx SpecContext) {
					Expect(s.dist.Framer.DeleteTimeRange(ctx, s.keys, (10 * telem.SecondTS).Range(12*telem.SecondTS))).To(Succeed())
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(s.keys[1]).Len()).To(Equal(int64(1)))
					Expect(i.Value().Get(s.keys[1]).TimeRange()).To(Equal((12 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
				})
				It("Should delete all data in a time range", func(ctx SpecContext) {
					Expect(s.dist.Framer.DeleteTimeRange(ctx, s.keys, telem.TimeRangeMax)).To(Succeed())
					Expect(i.SeekFirst()).To(BeFalse())
				})
				It("Should be idempotent when deleting an empty range", func(ctx SpecContext) {
					emptyRange := (100 * telem.SecondTS).Range(200 * telem.SecondTS)
					Expect(s.dist.Framer.DeleteTimeRange(ctx, s.keys, emptyRange)).To(Succeed())
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(s.keys[0]).Len()).To(Equal(int64(3)))
				})
				It("Should be a no-op when no keys are provided", func(ctx SpecContext) {
					Expect(s.dist.Framer.DeleteTimeRange(ctx, nil, telem.TimeRangeMax)).To(Succeed())
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(s.keys[0]).Len()).To(Equal(int64(3)))
				})
			})
		})
		Describe("Channel not found", func() {
			Specify("By key", func(ctx SpecContext) {
				Expect(s.dist.Framer.DeleteTimeRange(ctx, channel.Keys{10}, telem.TimeRangeMax)).To(MatchError(ts.ErrChannelNotFound))
			})
		})
	}

	Describe("Mixed Gateway and Peer", Ordered, func() {
		var s scenario
		BeforeAll(func(ctx SpecContext) { s = mixedScenario(context.Background()) })
		AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })

		It("Should delete channels across gateway and peer nodes", func(ctx SpecContext) {
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  s.keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(w.Write(frame.NewMulti(
				s.keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesSecondsTSV(10, 11, 12),
				},
			))).To(BeTrue())
			Expect(MustSucceed(w.Commit())).To(Equal(telem.SecondTS*12 + 1))
			Expect(w.Close()).To(Succeed())

			Expect(s.dist.Framer.DeleteTimeRange(ctx, s.keys, (10 * telem.SecondTS).Range(12*telem.SecondTS))).To(Succeed())

			i := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   s.keys,
				Bounds: telem.TimeRangeMax,
			}))
			Expect(i.SeekFirst()).To(BeTrue())
			Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
			for _, key := range s.keys {
				Expect(i.Value().Get(key).Len()).To(Equal(int64(1)))
				Expect(i.Value().Get(key).TimeRange()).To(Equal((12 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
			}
			Expect(i.Close()).To(Succeed())

			Expect(s.dist.Framer.DeleteTimeRange(ctx, s.keys, telem.TimeRangeMax)).To(Succeed())
		})

		It("Should delete all data across gateway and peer nodes", func(ctx SpecContext) {
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  s.keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(w.Write(frame.NewMulti(
				s.keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesSecondsTSV(10, 11, 12),
				},
			))).To(BeTrue())
			Expect(MustSucceed(w.Commit())).To(Equal(telem.SecondTS*12 + 1))
			Expect(w.Close()).To(Succeed())

			Expect(s.dist.Framer.DeleteTimeRange(ctx, s.keys, telem.TimeRangeMax)).To(Succeed())

			i := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   s.keys,
				Bounds: telem.TimeRangeMax,
			}))
			Expect(i.SeekFirst()).To(BeFalse())
			Expect(i.Close()).To(Succeed())
		})
	})
})

type scenario struct {
	dist   mock.Node
	closer io.Closer
	name   string
	keys   channel.Keys
	names  []string
}

func newChannelSet() []channel.Channel {
	return []channel.Channel{
		{
			Name:     "test1",
			IsIndex:  true,
			DataType: telem.TimeStampT,
		},
		{
			Name:     "test2",
			IsIndex:  true,
			DataType: telem.TimeStampT,
		},
		{
			Name:     "test3",
			IsIndex:  true,
			DataType: telem.TimeStampT,
		},
	}
}

func gatewayOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 1)
	dist := builder.Nodes[1]
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	names := lo.Map(channels, func(ch channel.Channel, _ int) string { return ch.Name })
	return scenario{
		name:   "Gateway Only",
		keys:   keys,
		names:  names,
		dist:   dist,
		closer: builder,
	}
}

func peerOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 4)
	dist := builder.Nodes[1]
	for i := range channels {
		channels[i].Leaseholder = node.Key(i + 2)
	}
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(dist.Channel.NewRetrieve().
			Entries(&chs).
			WhereKeys(keys...).
			Exec(ctx, nil)).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	names := lo.Map(channels, func(ch channel.Channel, _ int) string { return ch.Name })
	return scenario{
		name:   "Peer Only",
		keys:   keys,
		names:  names,
		dist:   dist,
		closer: builder,
	}
}

func mixedScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 3)
	dist := builder.Nodes[1]
	for i := range channels {
		channels[i].Leaseholder = node.Key(i + 1)
	}
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(dist.Channel.NewRetrieve().
			Entries(&chs).
			WhereKeys(keys...).
			Exec(ctx, nil)).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	names := lo.Map(channels, func(ch channel.Channel, _ int) string { return ch.Name })
	return scenario{
		name:   "Mixed Gateway and Peer",
		keys:   keys,
		names:  names,
		dist:   dist,
		closer: builder,
	}
}
