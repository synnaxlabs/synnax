// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Channel", func() {
	Describe("String", func() {
		It("Should return a nicely formatting string with the channel's key and name", func() {
			c := channel.Channel{Key: 1, Name: "foo"}
			Expect(c.String()).To(Equal("[foo]<1>"))
		})
		It("Should only return the key if the name is not present", func() {
			c := channel.Channel{Key: 1}
			Expect(c.String()).To(Equal("<1>"))
		})
	})

	Describe("ValidateSeries", func() {
		It("Should allow int64 series to pass as timestamps", func() {
			c := channel.Channel{Key: 1, DataType: telem.TimeStampT, Name: "cat"}
			s := telem.NewSeriesV[int64](1, 2, 3)
			Expect(c.ValidateSeries(s)).To(Succeed())
		})
		It("Should allow timestamps to pass as int64", func() {
			c := channel.Channel{Key: 1, DataType: telem.Int64T, Name: "cat"}
			s := telem.NewSeriesV[telem.TimeStamp](1, 2, 3)
			Expect(c.ValidateSeries(s)).To(Succeed())
		})
		It("Should return an error if the series data type does not match the channel data type", func() {
			c := channel.Channel{Key: 1, DataType: telem.Int64T, Name: "cat"}
			s := telem.NewSeriesV[float64](1, 2, 3)
			Expect(c.ValidateSeries(s)).To(
				And(
					MatchError(ContainSubstring("invalid data type for channel [cat]<1>, expected int64, got float64")),
					MatchError(validate.ErrValidation),
				))
		})

	})
	Describe("Validate", func() {
		DescribeTable("Valid channels", func(ch channel.Channel) {
			Expect(ch.Validate()).To(Succeed())
		},
			Entry("Valid channel", channel.Channel{
				Key:      1,
				DataType: telem.Int64T,
				Name:     "cat",
				Index:    2,
			}),
			Entry("Valid index channel", channel.Channel{
				Key:      1,
				DataType: telem.TimeStampT,
				Name:     "cat",
				IsIndex:  true,
			}),
			Entry("Valid virtual channel", channel.Channel{
				Key:      1,
				DataType: telem.Int64T,
				Name:     "cat",
				Virtual:  true,
			}),
			Entry("Valid virtual channel with an index", channel.Channel{
				Key:      1,
				DataType: telem.Int64T,
				Name:     "cat",
				Virtual:  true,
				Index:    2,
			}),
			Entry("Valid virtual index channel", channel.Channel{
				Key:      1,
				DataType: telem.TimeStampT,
				Name:     "cat",
				IsIndex:  true,
				Virtual:  true,
			}),
		)
		DescribeTable("Invalid channels", func(substring string, ch channel.Channel) {
			Expect(ch.Validate()).To(MatchError(ContainSubstring(substring)))
		},
			Entry("ChannelKey has no datatype",
				"data_type: required",
				cesium.Channel{Name: "cat", Key: 9990, IsIndex: true},
			),
			Entry("ChannelKey IsIndex - Non Int64 Series Variant",
				"data_type: index channel must be of type timestamp",
				cesium.Channel{Name: "Richard", Key: 9993, IsIndex: true, DataType: telem.Int64T},
			),
			Entry("ChannelKey IsIndex - LocalIndex non-zero",
				"index: index channel cannot be indexed by another channel",
				cesium.Channel{Name: "Feynman", Key: 9995, IsIndex: true, Index: 500, DataType: telem.TimeStampT},
			),
			Entry("ChannelKey has no index",
				"index: non-indexed channel must have an index",
				cesium.Channel{Name: "Steinbeck", Key: 9998, DataType: telem.Float32T},
			),
			Entry("Virtual index channel with a non-timestamp data type",
				"data_type: index channel must be of type timestamp",
				cesium.Channel{Name: "Woolf", Key: 9999, Virtual: true, IsIndex: true, DataType: telem.Float32T},
			),
		)
	})
})
