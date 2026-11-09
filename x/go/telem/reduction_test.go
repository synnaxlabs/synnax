// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Reduction", func() {
	stride := telem.Reduction{Variant: telem.StrideReduction{Factor: 4}}
	limit := telem.Reduction{Variant: telem.LimitReduction{
		Aggregation: telem.AggregationAverage,
		PointLimit:  512,
	}}

	Describe("MarshalJSON", func() {
		DescribeTable("Should tag the variant",
			func(r telem.Reduction, expected string) {
				Expect(json.Marshal(r)).To(MatchJSON(expected))
			},
			Entry("stride", stride, `{"variant":"stride","factor":4}`),
			Entry(
				"limit",
				limit,
				`{"variant":"limit","aggregation":"average","point_limit":512}`,
			),
			Entry("none", telem.Reduction{}, `null`),
		)
	})

	Describe("UnmarshalJSON", func() {
		DescribeTable("Should round-trip a variant",
			func(r telem.Reduction) {
				var decoded telem.Reduction
				Expect(json.Unmarshal(MustSucceed(json.Marshal(r)), &decoded)).
					To(Succeed())
				Expect(decoded).To(Equal(r))
			},
			Entry("stride", stride),
			Entry("limit", limit),
			Entry("none", telem.Reduction{}),
		)

		It("Should decode a variant when unknown members are rejected", func() {
			var decoded telem.Reduction
			Expect(json.Unmarshal(
				MustSucceed(json.Marshal(limit)),
				&decoded,
				json.RejectUnknownMembers(true),
			)).To(Succeed())
			Expect(decoded).To(Equal(limit))
		})

		It("Should reject an unknown member of a variant when asked", func() {
			var decoded telem.Reduction
			Expect(json.Unmarshal(
				[]byte(`{"variant":"stride","factor":4,"extra":1}`),
				&decoded,
				json.RejectUnknownMembers(true),
			)).To(MatchError(ContainSubstring(`unknown object member name "extra"`)))
		})

		It("Should reject an unknown variant", func() {
			var decoded telem.Reduction
			Expect(json.Unmarshal([]byte(`{"variant":"median"}`), &decoded)).
				To(MatchError(ContainSubstring(`unknown variant "median"`)))
		})

		It("Should reject a variant with mistyped fields", func() {
			var decoded telem.Reduction
			Expect(json.Unmarshal(
				[]byte(`{"variant":"limit","point_limit":"many"}`),
				&decoded,
			)).To(MatchError(ContainSubstring(`within "/point_limit"`)))
		})
	})

	Describe("ApplyDefaults", func() {
		It("Should default the aggregation of a limit to min_max", func() {
			r := telem.Reduction{Variant: telem.LimitReduction{PointLimit: 512}}
			r.ApplyDefaults()
			Expect(r.Variant).To(Equal(telem.LimitReduction{
				Aggregation: telem.AggregationMinMax,
				PointLimit:  512,
			}))
		})

		It("Should leave a stride unchanged", func() {
			r := stride
			r.ApplyDefaults()
			Expect(r).To(Equal(stride))
		})
	})

	Describe("Validate", func() {
		DescribeTable("Should accept a valid reduction",
			func(r telem.Reduction) { Expect(r.Validate()).To(Succeed()) },
			Entry("stride", stride),
			Entry("limit", limit),
			Entry("none", telem.Reduction{}),
		)

		DescribeTable("Should reject an invalid limit",
			func(l telem.LimitReduction, message string) {
				Expect(telem.Reduction{Variant: l}.Validate()).
					To(MatchError(message))
			},
			Entry(
				"unknown aggregation",
				telem.LimitReduction{Aggregation: "median", PointLimit: 512},
				"aggregation: invalid aggregation: median",
			),
			Entry(
				"zero point limit",
				telem.LimitReduction{Aggregation: telem.AggregationMinMax},
				"point_limit: must be greater than or equal to 1",
			),
		)
	})
})
