// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Chunked time", func() {
	var (
		node        testutil.Node
		iteratorSvc *iterator.Service
		channelSvc  *channel.Service
		idx, d1, d2 *channel.Channel
	)
	BeforeEach(func(ctx SpecContext) {
		node = testutil.NewNode(ctx)
		iteratorSvc, channelSvc = openServices(ctx, node)
		idx = &channel.Channel{
			Name:     "breaker_index",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, idx)
		})).To(Succeed())
		d1 = &channel.Channel{
			Name:       "breaker_d1",
			DataType:   telem.Float32T,
			LocalIndex: idx.LocalKey,
		}
		d2 = &channel.Channel{
			Name:       "breaker_d2",
			DataType:   telem.Float32T,
			LocalIndex: idx.LocalKey,
		}
		for _, ch := range []*channel.Channel{d1, d2} {
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, ch)
			})).To(Succeed())
		}
		// d1 holds samples at 1 to 3 seconds, and d2 at 10 to 12 seconds. The index
		// holds all six stamps.
		write := func(data channel.Key, start telem.TimeStamp) {
			keys := channel.Keys{idx.Key(), data}
			w := MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: start,
				Keys:  keys,
			}))
			s := start / telem.SecondTS
			Expect(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(s, s+1, s+2),
				telem.NewSeriesV[float32](1, 2, 3),
			}))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
		}
		write(d1.Key(), telem.SecondTS)
		write(d2.Key(), 10*telem.SecondTS)
	})

	It("Should read the time of the latest samples of a data channel", func(
		ctx SpecContext,
	) {
		iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
			Keys:      channel.Keys{d1.Key()},
			Bounds:    telem.TimeRangeMax,
			ChunkSize: 3,
		}))
		Expect(iter.SeekLast()).To(BeTrue())
		Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
		fr := iter.Value()
		Expect(iter.Close()).To(Succeed())
		values := fr.Get(d1.Key()).Series
		Expect(values).To(HaveLen(1))
		stamps := fr.Get(idx.Key()).Series
		Expect(stamps).ToNot(BeEmpty())
		bounds := make([]telem.AlignmentBounds, 0, len(stamps))
		for _, s := range stamps {
			bounds = append(bounds, s.AlignmentBounds())
		}
		Expect(bounds).To(ContainElement(values[0].AlignmentBounds()))
	})

	It("Should read the time of the earliest samples of a data channel", func(
		ctx SpecContext,
	) {
		iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
			Keys:      channel.Keys{d2.Key()},
			Bounds:    telem.TimeRangeMax,
			ChunkSize: 3,
		}))
		Expect(iter.SeekFirst()).To(BeTrue())
		Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
		fr := iter.Value()
		Expect(iter.Close()).To(Succeed())
		values := fr.Get(d2.Key()).Series
		Expect(values).To(HaveLen(1))
		stamps := fr.Get(idx.Key()).Series
		Expect(stamps).ToNot(BeEmpty())
		bounds := make([]telem.AlignmentBounds, 0, len(stamps))
		for _, s := range stamps {
			bounds = append(bounds, s.AlignmentBounds())
		}
		Expect(bounds).To(ContainElement(values[0].AlignmentBounds()))
	})
	It("Should read the time of the latest samples when the read requests the index",
		func(ctx SpecContext) {
			iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
				Keys:      channel.Keys{idx.Key(), d1.Key()},
				Bounds:    telem.TimeRangeMax,
				ChunkSize: 3,
			}))
			Expect(iter.SeekLast()).To(BeTrue())
			Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
			fr := iter.Value()
			Expect(iter.Close()).To(Succeed())
			values := fr.Get(d1.Key()).Series
			Expect(values).To(HaveLen(1))
			stamps := fr.Get(idx.Key()).Series
			bounds := make([]telem.AlignmentBounds, 0, len(stamps))
			for _, s := range stamps {
				bounds = append(bounds, s.AlignmentBounds())
			}
			Expect(bounds).To(ContainElement(values[0].AlignmentBounds()))
		},
	)

	It("Should read the time of two clocks whose series share alignment bounds",
		func(ctx SpecContext) {
			other := &channel.Channel{
				Name:     "other_index",
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, other)
			})).To(Succeed())
			data := &channel.Channel{
				Name:       "other_data",
				DataType:   telem.Float32T,
				LocalIndex: other.LocalKey,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, data)
			})).To(Succeed())
			keys := channel.Keys{other.Key(), data.Key()}
			w := MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: 20 * telem.SecondTS,
				Keys:  keys,
			}))
			Expect(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(20, 21, 22),
				telem.NewSeriesV[float32](4, 5, 6),
			}))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			iter := MustSucceed(iteratorSvc.Open(ctx, iterator.Config{
				Keys:      channel.Keys{d1.Key(), data.Key()},
				Bounds:    telem.TimeRangeMax,
				ChunkSize: 3,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
			fr := iter.Value()
			Expect(iter.Close()).To(Succeed())
			for clock, key := range map[channel.Key]channel.Key{
				idx.Key():   d1.Key(),
				other.Key(): data.Key(),
			} {
				values := fr.Get(key).Series
				Expect(values).To(HaveLen(1))
				stamps := fr.Get(clock).Series
				bounds := make([]telem.AlignmentBounds, 0, len(stamps))
				for _, s := range stamps {
					bounds = append(bounds, s.AlignmentBounds())
				}
				Expect(bounds).To(ContainElement(values[0].AlignmentBounds()))
			}
		},
	)
})
