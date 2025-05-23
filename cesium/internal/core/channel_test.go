// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"

	"github.com/synnaxlabs/cesium/internal/core"
)

var _ = Describe("Channel", func() {
	Describe("String", func() {
		It("Should return a nicely formatting string with the channel's key and name", func() {
			c := core.Channel{Key: 1, Name: "foo"}
			Expect(c.String()).To(Equal("[foo]<1>"))
		})
		It("Should only return the key if the name is not present", func() {
			c := core.Channel{Key: 1}
			Expect(c.String()).To(Equal("<1>"))
		})
	})

	Describe("ValidateSeries", func() {
		It("Should return an error if the series data type does not match the channel data type", func() {
			c := core.Channel{Key: 1, DataType: telem.Int64T, Name: "cat"}
			s := telem.NewSeriesV[float64](1, 2, 3)
			err := c.ValidateSeries(s)
			Expect(err).To(HaveOccurredAs(validate.Error))
			Expect(err).To(MatchError(ContainSubstring("invalid data type for channel [cat]<1>, expected int64, got float64")))
		})
		It("Should allow int64 series to pass as timestamps", func() {
			c := core.Channel{Key: 1, DataType: telem.TimeStampT, Name: "cat"}
			s := telem.NewSeriesV[int64](1, 2, 3)
			err := c.ValidateSeries(s)
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allow timestamps to pass as int64", func() {
			c := core.Channel{Key: 1, DataType: telem.Int64T, Name: "cat"}
			s := telem.NewSeriesV[telem.TimeStamp](1, 2, 3)
			err := c.ValidateSeries(s)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
