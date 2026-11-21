// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func aligned(s telem.Series, sample uint32) telem.Series {
	s.Alignment = telem.NewAlignment(1, sample)
	return s
}

func clocked(s telem.Series, clock uint32) telem.Series {
	s.Clock = clock
	return s
}

var _ = Describe("Series order", func() {
	roundTrip := func(
		ctx SpecContext,
		keys channel.Keys,
		dataTypes []telem.DataType,
		fr framer.Frame,
	) framer.Frame {
		GinkgoHelper()
		c := codec.NewStatic(keys, dataTypes)
		return MustSucceed(c.Decode(MustSucceed(c.Encode(ctx, fr))))
	}

	It("Should keep the producer's order within a key", func(ctx SpecContext) {
		backfill := aligned(telem.NewSeriesV[int32](3, 4), 50)
		earlier := aligned(telem.NewSeriesV[int32](1, 2), 10)
		decoded := roundTrip(
			ctx,
			channel.Keys{1},
			[]telem.DataType{telem.Int32T},
			frame.NewMulti(channel.Keys{1, 1}, []telem.Series{backfill, earlier}),
		)
		Expect(decoded.Get(1).Series).To(HaveExactElements(
			telem.MatchSeries(backfill),
			telem.MatchSeries(earlier),
		))
	})

	It("Should sort the series by key", func(ctx SpecContext) {
		decoded := roundTrip(
			ctx,
			channel.Keys{1, 2},
			[]telem.DataType{telem.Int32T, telem.Int32T},
			frame.NewMulti(
				channel.Keys{2, 1},
				[]telem.Series{
					telem.NewSeriesV[int32](2),
					telem.NewSeriesV[int32](1),
				},
			),
		)
		Expect(decoded.KeysSlice()).To(Equal([]channel.Key{1, 2}))
	})

	Describe("Merges", func() {
		DescribeTable("Should merge adjacent series of a key only when allowed",
			func(ctx SpecContext, first, second telem.Series, count int) {
				decoded := roundTrip(
					ctx,
					channel.Keys{1},
					[]telem.DataType{telem.TimestampT},
					frame.NewMulti(
						channel.Keys{1, 1},
						[]telem.Series{
							aligned(first, 0),
							aligned(second, 2),
						},
					),
				)
				Expect(decoded.Get(1).Series).To(HaveLen(count))
			},
			Entry(
				"contiguous, same clock and type",
				telem.NewSeriesSecondsTSV(1, 2),
				telem.NewSeriesSecondsTSV(3, 4),
				1,
			),
			Entry(
				"contiguous, different clocks",
				clocked(telem.NewSeriesSecondsTSV(1, 2), 5),
				clocked(telem.NewSeriesSecondsTSV(3, 4), 7),
				2,
			),
			Entry(
				"contiguous, different types",
				telem.NewSeriesSecondsTSV(1, 2),
				telem.NewSeriesV[int64](3, 4),
				2,
			),
		)
	})

	Describe("Equal lengths", func() {
		It("Should share one length when no series in the frame is variable", func(
			ctx SpecContext,
		) {
			c := codec.NewStatic(
				channel.Keys{1, 2, 3},
				[]telem.DataType{telem.StringT, telem.Float32T, telem.Float32T},
			)
			fr := frame.NewMulti(
				channel.Keys{2, 3},
				[]telem.Series{
					telem.NewSeriesV[float32](1),
					telem.NewSeriesV[float32](2),
				},
			)
			MustSucceed(c.Decode(MustSucceed(c.Encode(ctx, fr))))
			encoded := MustSucceed(c.Encode(ctx, fr))
			// flags, sequence number, shared length, then a key and 4 data bytes for
			// each series.
			Expect(encoded).To(HaveLen(1 + 4 + 4 + 2*(4+4)))
			decoded := MustSucceed(c.Decode(encoded))
			Expect(decoded.Get(3).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[float32](2)),
			)
		})
	})
})
