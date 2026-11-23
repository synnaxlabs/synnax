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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/payload"
)

var _ = Describe("CheckDefault", func() {
	DescribeTable("Should accept a default that its field can hold",
		func(v icd.JSONFieldVariant) {
			Expect(icd.CheckDefault(icd.JSONField{Variant: v})).To(Succeed())
		},
		Entry("no default", icd.FloatJSONField{Scale: 1, Max: new(0.0)}),
		Entry("a scaled default at the max", icd.FloatJSONField{
			Scale: 0.5, Max: new(50.0), Default: new(payload.Float(100)),
		}),
		Entry("a default with an offset", icd.IntegerJSONField{
			Scale: 1, Offset: -10, Max: new(0.0), Default: new(payload.Integer(10)),
		}),
		Entry("a default that float error puts above the max", icd.FloatJSONField{
			Scale: 0.1, Max: new(0.3), Default: new(payload.Float(3)),
		}),
		Entry("a default that float error puts below the min", icd.UnsignedJSONField{
			Scale: 0.3, Min: new(0.9), Default: new(payload.Unsigned(3)),
		}),
		Entry("base64 text", icd.BytesJSONField{Default: new("AQI=")}),
		Entry("valid JSON", icd.JSONJSONField{Default: new(`{"a":1}`)}),
	)

	DescribeTable("Should refuse a default that its field cannot hold",
		func(v icd.JSONFieldVariant, expected string) {
			Expect(icd.CheckDefault(icd.JSONField{Variant: v})).To(MatchError(expected))
		},
		Entry("a NaN float",
			icd.FloatJSONField{Scale: 1, Default: new(payload.Float(math.NaN()))},
			"must be finite: validation error",
		),
		Entry("an infinite float",
			icd.FloatJSONField{Scale: 1, Default: new(payload.Float(math.Inf(1)))},
			"must be finite: validation error",
		),
		Entry("text that is not base64",
			icd.BytesJSONField{Default: new("a*b=")},
			"must be base64: validation error",
		),
		Entry("text that is not valid JSON",
			icd.JSONJSONField{Default: new("{")},
			"must be valid JSON: validation error",
		),
		Entry("an unscaled default above the max",
			icd.FloatJSONField{
				Scale: 1, Max: new(100.0), Default: new(payload.Float(120)),
			},
			"120 is above the max 100: validation error",
		),
		Entry("a scaled default above the max",
			icd.IntegerJSONField{
				Scale: 10, Max: new(100.0), Default: new(payload.Integer(12)),
			},
			"12 decodes to 120, which is above the max 100: validation error",
		),
		Entry("a default with an offset below the min",
			icd.FloatJSONField{
				Scale: 1, Offset: 5, Min: new(0.0), Default: new(payload.Float(-6)),
			},
			"-6 decodes to -1, which is below the min 0: validation error",
		),
		Entry("a default just above the max",
			icd.FloatJSONField{
				Scale: 0.1, Max: new(0.3), Default: new(payload.Float(3.000001)),
			},
			"3.000001 decodes to 0.30000010000000005, which is above the max 0.3: "+
				"validation error",
		),
	)
})
