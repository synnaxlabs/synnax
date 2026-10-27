// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package filter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("CompileString", func() {
	DescribeTable(
		"Should match every operator",
		func(s filter.String, v string, expected bool) {
			match := MustSucceed(filter.CompileString(s))
			Expect(match(v)).To(Equal(expected))
		},
		Entry("no operator", filter.String{}, "Tank", true),
		Entry("eq", filter.String{Eq: new("Tank")}, "Tank", true),
		Entry(
			"eq of a different case",
			filter.String{Eq: new("tank")},
			"Tank",
			false,
		),
		Entry("in", filter.String{In: []string{"a", "Tank"}}, "Tank", true),
		Entry("an empty in", filter.String{In: []string{}}, "Tank", false),
		Entry("contains", filter.String{Contains: new("an")}, "Tank", true),
		Entry(
			"contains of a different case",
			filter.String{Contains: new("TA")},
			"Tank",
			false,
		),
		Entry("starts_with", filter.String{StartsWith: new("Ta")}, "Tank", true),
		Entry(
			"starts_with in the middle",
			filter.String{StartsWith: new("an")},
			"Tank",
			false,
		),
		Entry("regex", filter.String{Regex: new("^T.n")}, "Tank", true),
		Entry(
			"regex of a different case",
			filter.String{Regex: new("^t")},
			"Tank",
			false,
		),
		Entry(
			"every operator",
			filter.String{Eq: new("Tank"), Contains: new("x")},
			"Tank",
			false,
		),
	)

	DescribeTable("Should ignore case with CaseInsensitive",
		func(s filter.String, v string) {
			s.CaseInsensitive = true
			match := MustSucceed(filter.CompileString(s))
			Expect(match(v)).To(BeTrue())
		},
		Entry("eq", filter.String{Eq: new("TANK")}, "tank"),
		Entry("in", filter.String{In: []string{"TANK"}}, "tank"),
		Entry("contains", filter.String{Contains: new("AN")}, "tank"),
		Entry("starts_with", filter.String{StartsWith: new("TA")}, "tank"),
		Entry("regex", filter.String{Regex: new("^TA")}, "tank"),
		Entry("a letter outside ASCII", filter.String{Eq: new("ÄRGER")}, "ärger"),
	)

	DescribeTable("Should fold case by Unicode simple case folding",
		func(operand, v string, expected bool) {
			eq := MustSucceed(filter.CompileString(
				filter.String{Eq: new(operand), CaseInsensitive: true},
			))
			Expect(eq(v)).To(Equal(expected))
			re := MustSucceed(filter.CompileString(
				filter.String{
					Regex:           new("^" + operand + "$"),
					CaseInsensitive: true,
				},
			))
			Expect(re(v)).To(Equal(expected))
		},
		Entry("a Greek final sigma", "οδοσ", "ΟΔΟΣ", true),
		Entry("a final and a medial sigma", "σ", "ς", true),
		Entry("a long s", "s", "ſ", true),
		Entry("the Kelvin sign", "k", "K", true),
		Entry("a Turkish dotted capital I", "i", "İ", false),
	)

	It("Should fold contains and starts_with by the same rule", func() {
		match := MustSucceed(filter.CompileString(filter.String{
			Contains:        new("σ"),
			StartsWith:      new("Κ"),
			CaseInsensitive: true,
		}))
		Expect(match("κάτς")).To(BeTrue())
	})

	It("Should reject an invalid regex at its path", func() {
		s := filter.String{Regex: new("(")}
		Expect(filter.CompileString(s)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"regex: error parsing regexp: missing closing ): `(`: validation error",
			),
		))
	})
})
