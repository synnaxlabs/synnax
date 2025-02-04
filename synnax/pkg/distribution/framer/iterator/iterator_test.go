// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"io"
)

var _ = Describe("Iterator", func() {
	Describe("Happy Path", Ordered, func() {
		scenarios := []func() scenario{
			gatewayOnlyScenario,
			peerOnlyScenario,
		}
		for i, sF := range scenarios {
			_sF := sF
			var s scenario
			BeforeAll(func() {
				s = _sF()
				writer := MustSucceed(s.writerService.New(context.TODO(), writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
				}))
				Expect(writer.Write(core.Frame{
					Keys: s.keys,
					Series: []telem.Series{
						telem.NewSeriesV[int64](10, 11, 12),
					}},
				)).To(BeTrue())
				Expect(writer.Write(core.Frame{
					Keys: s.keys,
					Series: []telem.Series{
						telem.NewSeriesV[int64](13, 14, 15, 16, 17),
					}},
				)).To(BeTrue())
				Expect(writer.Write(core.Frame{
					Keys: s.keys,
					Series: []telem.Series{
						telem.NewSeriesV[int64](18, 19, 20, 21, 22),
					},
				})).To(BeTrue())
				Expect(writer.Commit()).To(BeTrue())
				Expect(writer.Error()).To(Succeed())
				Expect(writer.Close()).To(Succeed())
			})
			AfterAll(func() { Expect(s.close.Close()).To(Succeed()) })
			Specify(fmt.Sprintf("Scenario: %v - Iteration", i), func() {
				iter := MustSucceed(s.iteratorService.New(context.TODO(), iterator.Config{
					Keys:   s.keys,
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(4 * telem.Second)).To(BeTrue())
				Expect(iter.Value().Series[0].Data).To(EqualUnmarshal([]int64{10, 11, 12, 13}))
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev(6 * telem.Second)).To(BeTrue())
				Expect(iter.Value().Series[0].Data).To(EqualUnmarshal([]int64{17, 18, 19, 20, 21, 22}))

				Expect(iter.SeekGE(100 * telem.SecondTS)).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())
				Expect(iter.SeekLE(22*telem.SecondTS + 1)).To(BeTrue())
				Expect(iter.Prev(2 * telem.Second)).To(BeTrue())
				Expect(iter.Value().Series[0].Data).To(EqualUnmarshal([]int64{21, 22}))

				Expect(iter.SeekLE(0 * telem.SecondTS)).To(BeFalse())
				Expect(iter.Valid()).To(BeFalse())
				Expect(iter.SeekGE(13 * telem.SecondTS)).To(BeTrue())
				Expect(iter.Next(20 * telem.Second)).To(BeTrue())
				Expect(iter.Value().Series[0].Data).To(EqualUnmarshal([]int64{13, 14, 15, 16, 17, 18, 19, 20, 21, 22}))

				Expect(iter.Close()).To(Succeed())
			})

			Specify(fmt.Sprintf("Scenario: %v - Auto chunk", i), func() {
				iter := MustSucceed(s.iteratorService.New(context.TODO(), iterator.Config{
					Keys:      s.keys,
					Bounds:    telem.TimeRangeMax,
					ChunkSize: 3,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
				Expect(iter.Value().Series[0].Data).To(EqualUnmarshal([]int64{10, 11, 12}))
				Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
				Expect(iter.Value().Series[0].Data).To(EqualUnmarshal([]int64{13, 14, 15}))
				Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
				Expect(iter.Value().Series[0].Data).To(EqualUnmarshal([]int64{16, 17, 18}))

				Expect(iter.Close()).To(Succeed())
			})
		}
	})
})

type scenario struct {
	name            string
	keys            channel.Keys
	writerService   *writer.Service
	iteratorService *iterator.Service
	channel         channel.Service
	close           io.Closer
}

func newChannelSet() []channel.Channel {
	return []channel.Channel{
		{
			Name:     "test1",
			Rate:     1 * telem.Hz,
			DataType: telem.Int64T,
		},
	}
}

func gatewayOnlyScenario() scenario {
	channels := newChannelSet()
	builder, services := provision(1)
	svc := services[1]
	Expect(svc.channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:            "gatewayOnly",
		keys:            keys,
		writerService:   svc.writer,
		iteratorService: svc.iter,
		close:           builder,
		channel:         svc.channel,
	}
}

func peerOnlyScenario() scenario {
	channels := newChannelSet()
	builder, services := provision(4)
	svc := services[1]
	for i, ch := range channels {
		ch.Leaseholder = dcore.NodeKey(i + 2)
		channels[i] = ch
	}
	Expect(svc.channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := svc.channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:            "peerOnly",
		keys:            keys,
		writerService:   svc.writer,
		iteratorService: svc.iter,
		close:           builder,
		channel:         svc.channel,
	}
}
