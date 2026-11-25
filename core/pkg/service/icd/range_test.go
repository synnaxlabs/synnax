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
		Entry("above a large max, in plain digits",
			icd.BaseField{Scale: 1, Max: new(1e6)}, 2e6,
			"2000000 is above the max 1000000: validation error",
		),
		Entry("below a small min, in plain digits",
			icd.BaseField{Scale: 1, Min: new(5e-5)}, 2e-5,
			"0.00002 is below the min 0.00005: validation error",
		),
	)
})
