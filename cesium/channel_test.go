// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ChannelKey", Ordered, func() {
	var db cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Create", func() {
		Describe("Happy Path", func() {
			It("Should assign an auto-incremented key if a key is not present", func() {
				ch := cesium.Channel{Key: "chOne", Rate: 10 * telem.Hz, DataType: telem.Float64T}
				Expect(db.CreateChannel(ctx, ch)).To(Succeed())
				Expect(ch.Key).To(Equal("chOne"))
			})
		})
		DescribeTable("Validation", func(expected error, channels ...cesium.Channel) {
			Expect(db.CreateChannel(ctx, channels...)).To(HaveOccurredAs(expected))
		},
			Entry("ChannelKey has no datatype",
				errors.Wrap(validate.Error, "[cesium] - data type must be set"),
				cesium.Channel{Key: "10", Rate: 10 * telem.Hz},
			),
			Entry("ChannelKey key already exists",
				errors.Wrap(validate.Error, "[cesium] - channel 11 already exists"),
				cesium.Channel{Key: "11", DataType: telem.Float32T, Rate: 10 * telem.Hz},
				cesium.Channel{Key: "11", Rate: 10 * telem.Hz, DataType: telem.Float64T},
			),
			Entry("ChannelKey IsIndex - Non Int64 Array Variant",
				errors.Wrap(validate.Error, "[cesium] - index channel must be of type timestamp"),
				cesium.Channel{Key: "12", IsIndex: true, DataType: telem.Float32T},
			),
			Entry("ChannelKey IsIndex - LocalIndex non-zero",
				errors.Wrap(validate.Error, "[cesium] - index channel cannot be indexed by another channel"),
				cesium.Channel{Key: "45", IsIndex: true, DataType: telem.TimeStampT},
				cesium.Channel{Key: "46", IsIndex: true, Index: "45", DataType: telem.TimeStampT},
			),
			Entry("ChannelKey has index - LocalIndex does not exist",
				errors.Wrapf(validate.Error, "[cesium] - index %s does not exist", "40000"),
				cesium.Channel{Key: "47", Index: "40000", DataType: telem.Float64T},
			),
			Entry("ChannelKey has no index - fixed rate not provided",
				errors.Wrap(validate.Error, "[cesium] - rate must be positive"),
				cesium.Channel{Key: "48", DataType: telem.Float32T},
			),
			Entry("ChannelKey has index - provided index key is not an indexed channel",
				errors.Wrap(validate.Error, "[cesium] - channel 60 is not an index"),
				cesium.Channel{Key: "60", DataType: telem.Float32T, Rate: 1 * telem.Hz},
				cesium.Channel{Key: "61", Index: "60", DataType: telem.Float32T},
			),
		)
	})
})
