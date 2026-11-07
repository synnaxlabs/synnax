// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	"github.com/synnaxlabs/x/defaults"
)

var _ = Describe("AccelChargeSensitivityUnits", func() {
	Describe("IsValid", func() {
		It("Should accept a defined unit", func() {
			unit := v2.AccelChargeSensitivityUnitsPicoCoulombsPerG
			Expect(unit.IsValid()).To(BeTrue())
		})

		It("Should reject an undefined unit", func() {
			Expect(v2.AccelChargeSensitivityUnits("Bogus").IsValid()).To(BeFalse())
		})
	})
})

var _ = Describe("Default groups", func() {
	DescribeTable("Should decode a channel range as one group",
		func(in string, expected v2.MinMaxVal) {
			var ch v2.AIChannel
			Expect(json.Unmarshal([]byte(in), &ch)).To(Succeed())
			Expect(ch.Variant.(v2.AIVoltageChannel).MinMaxVal).To(Equal(expected))
		},
		Entry(
			"an omitted range takes the defaults",
			`{"type":"ai_voltage"}`,
			v2.MinMaxVal{MinVal: 0, MaxVal: 1},
		),
		Entry(
			"a range with one bound keeps the other bound at zero",
			`{"type":"ai_voltage","min_val":-10}`,
			v2.MinMaxVal{MinVal: -10, MaxVal: 0},
		),
		Entry(
			"a full range stays",
			`{"type":"ai_voltage","min_val":-10,"max_val":10}`,
			v2.MinMaxVal{MinVal: -10, MaxVal: 10},
		),
	)

	DescribeTable("Should decode a restated range group as one group",
		func(in string, expected v2.MinMaxVal) {
			var ch v2.CIChannel
			Expect(json.Unmarshal([]byte(in), &ch)).To(Succeed())
			Expect(ch.Variant.(v2.CIFrequencyChannel).MinMaxVal).To(Equal(expected))
		},
		Entry(
			"an omitted range takes the variant's defaults",
			`{"type":"ci_frequency"}`,
			v2.MinMaxVal{MinVal: 2, MaxVal: 100},
		),
		Entry(
			"a range with one bound keeps the other bound at zero",
			`{"type":"ci_frequency","max_val":50}`,
			v2.MinMaxVal{MinVal: 0, MaxVal: 50},
		),
	)

	DescribeTable("Should fill a legacy range as one group",
		func(in, expected v2.MinMaxVal) {
			ch := defaults.Fill(v2.AIVoltageChannel{MinMaxVal: in})
			Expect(ch.MinMaxVal).To(Equal(expected))
		},
		Entry("all zero", v2.MinMaxVal{}, v2.MinMaxVal{MinVal: 0, MaxVal: 1}),
		Entry(
			"one bound zero",
			v2.MinMaxVal{MinVal: -10},
			v2.MinMaxVal{MinVal: -10, MaxVal: 0},
		),
	)
})
