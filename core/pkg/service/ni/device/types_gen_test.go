// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Hardware", func() {
	DescribeTable(
		"Should reject an analog input rate that is not a positive number",
		func(rate telem.Rate) {
			Expect(device.Hardware{AiMinRate: &rate}.Validate()).
				To(MatchError(ContainSubstring("ai_min_rate")))
		},
		Entry("zero", telem.Rate(0)),
		Entry("negative", telem.Rate(-1)),
		Entry("positive infinity", telem.Rate(math.Inf(1))),
		Entry("NaN", telem.Rate(math.NaN())),
	)

	It("Should accept an absent analog input rate", func() {
		Expect(device.Hardware{}.Validate()).To(Succeed())
	})

	reversed := []spatial.ClosedBounds{{Lower: 10, Upper: -10}}
	DescribeTable("Should reject a range whose lower is not below its upper",
		func(hw device.Hardware, field string) {
			Expect(hw.Validate()).To(MatchError(SatisfyAll(
				ContainSubstring(field),
				ContainSubstring("must be less than upper"),
			)))
		},
		Entry("an analog input voltage range",
			device.Hardware{AiVoltageRanges: reversed}, "ai_voltage_ranges"),
		Entry("an analog input current range",
			device.Hardware{AiCurrentRanges: reversed}, "ai_current_ranges"),
		Entry("an analog output voltage range",
			device.Hardware{AoVoltageRanges: reversed}, "ao_voltage_ranges"),
		Entry("an analog output current range",
			device.Hardware{AoCurrentRanges: reversed}, "ao_current_ranges"),
	)

	It("Should accept a range whose lower is below its upper", func() {
		Expect(device.Hardware{
			AiVoltageRanges: []spatial.ClosedBounds{{Lower: -10, Upper: 10}},
		}.Validate()).To(Succeed())
	})
})
