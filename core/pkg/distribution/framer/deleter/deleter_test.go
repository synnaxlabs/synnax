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
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Deleter", Ordered, func() {
	scenarios := []func() scenario{
		gatewayOnlyScenario,
	}
	for _, createScenario := range scenarios {
		var (
			s scenario
			d deleter.Deleter
			i *iterator.Iterator
		)
		BeforeAll(func() { s = createScenario() })
		AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })
		Describe("Happy Path", func() {
			Context(fmt.Sprintf("Scenario: %s - Happy Path", s.name), func() {
				BeforeEach(func() {
					writer := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
						Keys:  s.keys,
						Start: 10 * telem.SecondTS,
					}))
					Expect(writer.Write(frame.NewMulti(
						s.keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 11, 12),
							telem.NewSeriesSecondsTSV(10, 11, 12),
							telem.NewSeriesSecondsTSV(10, 11, 12),
						},
					))).To(BeTrue())
					Expect(MustSucceed(writer.Commit())).To(Equal(telem.SecondTS*12 + 1))
					Expect(writer.Close()).To(Succeed())

					d = s.dist.Framer.NewDeleter()
					i = MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
						Keys:   s.keys,
						Bounds: telem.TimeRangeMax,
					}))
				})
				AfterEach(func() {
					Expect(d.DeleteTimeRangeMany(ctx, s.keys, telem.TimeRangeMax))
					Expect(i.Close()).To(Succeed())
				})

				It("Should delete one channel by key", func() {
					Expect(d.DeleteTimeRange(ctx, s.keys[0], (10 * telem.SecondTS).Range(12*telem.SecondTS))).To(Succeed())
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(s.keys[0]).Len()).To(Equal(int64(1)))
					Expect(i.Value().Get(s.keys[0]).TimeRange()).To(Equal((12 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
				})
				It("Should delete one channel by name", func() {
					Expect(d.DeleteTimeRangeByName(ctx, s.names[0], (10 * telem.SecondTS).Range(12*telem.SecondTS))).To(Succeed())
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(s.keys[0]).Len()).To(Equal(int64(1)))
					Expect(i.Value().Get(s.keys[0]).TimeRange()).To(Equal((12 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
				})
				It("Should delete many channels by keys", func() {
					Expect(d.DeleteTimeRangeMany(ctx, s.keys, (10 * telem.SecondTS).Range(12*telem.SecondTS))).To(Succeed())
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(s.keys[1]).Len()).To(Equal(int64(1)))
					Expect(i.Value().Get(s.keys[1]).TimeRange()).To(Equal((12 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
				})
				It("Should delete many channels by names", func() {
					Expect(d.DeleteTimeRangeManyByNames(ctx, s.names, (10 * telem.SecondTS).Range(12*telem.SecondTS))).To(Succeed())
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(s.keys[1]).Len()).To(Equal(int64(1)))
					Expect(i.Value().Get(s.keys[1]).TimeRange()).To(Equal((12 * telem.SecondTS).Range(12*telem.SecondTS + 1)))
				})
			})
		})
		Describe("Channel not found", func() {
			Specify("By name", func() {
				d = s.dist.Framer.NewDeleter()
				Expect(d.DeleteTimeRangeByName(ctx, "kaka", telem.TimeRangeMin)).To(MatchError(ts.ErrChannelNotfound))
			})
			Specify("By key", func() {
				d = s.dist.Framer.NewDeleter()
				Expect(d.DeleteTimeRange(ctx, 10, telem.TimeRangeMax)).To(MatchError(ts.ErrChannelNotfound))
			})
		})
	}
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

func gatewayOnlyScenario() scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 1)
	dist := builder.Nodes[1]
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	names := lo.Map(channels, func(channel channel.Channel, _ int) string { return channel.Name })
	return scenario{
		name:   "Gateway Only",
		keys:   keys,
		names:  names,
		dist:   dist,
		closer: builder,
	}
}
