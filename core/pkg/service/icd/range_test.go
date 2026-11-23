// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
)

var _ = Describe("CheckRange", func() {
	DescribeTable("Should accept a value inside the min and max",
		func(b icd.BaseField, v float64) {
			Expect(icd.CheckRange(b, v)).To(Succeed())
		},
		Entry("no limits", icd.BaseField{Scale: 1}, 1e300),
		Entry("the min", icd.BaseField{Scale: 1, Min: new(0.0)}, 0.0),
		Entry("the max", icd.BaseField{Scale: 1, Max: new(100.0)}, 100.0),
		Entry("a value that takes no scale",
			icd.BaseField{Scale: 10, Max: new(100.0)}, 100.0,
		),
	)

	DescribeTable("Should refuse a value outside the min and max",
		func(b icd.BaseField, v float64, expected string) {
			Expect(icd.CheckRange(b, v)).To(MatchError(expected))
		},
		Entry("below the min",
			icd.BaseField{Scale: 1, Min: new(0.0)}, -1.0,
			"-1 is below the min 0: validation error",
		),
		Entry("above the max",
			icd.BaseField{Scale: 1, Max: new(100.0)}, 100.5,
			"100.5 is above the max 100: validation error",
		),
	)
})

var _ = Describe("CheckRawRange", func() {
	DescribeTable("Should accept a raw value that decodes inside the min and max",
		func(b icd.BaseField, raw float64) {
			Expect(icd.CheckRawRange(b, raw)).To(Succeed())
		},
		Entry("the max", icd.BaseField{Scale: 0.5, Max: new(50.0)}, 100.0),
		Entry("an offset", icd.BaseField{Scale: 1, Offset: -10, Max: new(0.0)}, 10.0),
		Entry("a value that float error puts above the max",
			icd.BaseField{Scale: 0.1, Max: new(0.3)}, 3.0,
		),
		Entry("a value that float error puts below the min",
			icd.BaseField{Scale: 0.3, Min: new(0.9)}, 3.0,
		),
	)

	DescribeTable("Should refuse a raw value that decodes outside the min and max",
		func(b icd.BaseField, raw float64, expected string) {
			Expect(icd.CheckRawRange(b, raw)).To(MatchError(expected))
		},
		Entry("an unscaled value",
			icd.BaseField{Scale: 1, Max: new(100.0)}, 120.0,
			"120 is above the max 100: validation error",
		),
		Entry("a scaled value",
			icd.BaseField{Scale: 10, Max: new(100.0)}, 12.0,
			"12 decodes to 120, which is above the max 100: validation error",
		),
		Entry("a value with an offset",
			icd.BaseField{Scale: 1, Offset: 5, Min: new(0.0)}, -6.0,
			"-6 decodes to -1, which is below the min 0: validation error",
		),
		Entry("a value just above the max",
			icd.BaseField{Scale: 0.1, Max: new(0.3)}, 3.000001,
			"3.000001 decodes to 0.30000010000000005, which is above the max 0.3: "+
				"validation error",
		),
	)
})
