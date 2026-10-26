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
)

var _ = Describe("MatchesString", func() {
	DescribeTable(
		"Should check every operator but regex",
		func(s filter.String, v string, expected bool) {
			Expect(filter.MatchesString(s, v)).To(Equal(expected))
		},
		Entry("an empty operator", filter.String{}, "Tank", true),
		Entry("eq", filter.String{Eq: new("Tank")}, "Tank", true),
		Entry("eq of a different case", filter.String{Eq: new("tank")}, "Tank", false),
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
		Entry(
			"every operator",
			filter.String{Eq: new("Tank"), Contains: new("x")},
			"Tank",
			false,
		),
		Entry("a regex", filter.String{Regex: new("^x")}, "Tank", true),
	)
	DescribeTable("Should ignore case with CaseInsensitive",
		func(s filter.String, v string) {
			s.CaseInsensitive = true
			Expect(filter.MatchesString(s, v)).To(BeTrue())
		},
		Entry("eq", filter.String{Eq: new("TANK")}, "tank"),
		Entry("in", filter.String{In: []string{"TANK"}}, "tank"),
		Entry("contains", filter.String{Contains: new("AN")}, "tank"),
		Entry("starts_with", filter.String{StartsWith: new("TA")}, "tank"),
		Entry("a letter outside ASCII", filter.String{Eq: new("ÄRGER")}, "ärger"),
	)
})
