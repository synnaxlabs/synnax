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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/internal/telem"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Telem", func() {
	var table *resolution.Table

	BeforeEach(func() {
		table = resolution.NewTable()
		ref := func(name string) resolution.TypeRef {
			return resolution.TypeRef{Name: name}
		}
		hand := map[string]resolution.Domain{
			"ts": {Expressions: resolution.Expressions{{Name: "hand"}}},
		}
		for _, typ := range []resolution.Type{
			{
				Name:          "TimeStamp",
				QualifiedName: "telem.TimeStamp",
				Form:          resolution.DistinctForm{Base: ref("int64")},
			},
			{
				Name:          "TimeSpan",
				QualifiedName: "telem.TimeSpan",
				Form:          resolution.DistinctForm{Base: ref("int64")},
			},
			{
				Name:          "Stamp",
				QualifiedName: "test.Stamp",
				Form:          resolution.AliasForm{Target: ref("telem.TimeStamp")},
			},
			{
				Name:          "Created",
				QualifiedName: "test.Created",
				Form:          resolution.DistinctForm{Base: ref("test.Stamp")},
			},
			{
				Name:          "Moment",
				QualifiedName: "test.Moment",
				Form:          resolution.AliasForm{Target: ref("test.Created")},
			},
			{
				Name:          "Rate",
				QualifiedName: "telem.Rate",
				Form:          resolution.DistinctForm{Base: ref("float64")},
			},
			{
				Name:          "NumericTimeSpan",
				QualifiedName: "telem.NumericTimeSpan",
				Domains:       hand,
				Form:          resolution.AliasForm{Target: ref("telem.TimeSpan")},
			},
			{
				Name:          "Count",
				QualifiedName: "test.Count",
				Form:          resolution.DistinctForm{Base: ref("int64")},
			},
		} {
			Expect(table.Add(typ)).To(Succeed())
		}
	})

	DescribeTable("IsTimeStamp",
		func(name string, expected bool) {
			Expect(telem.IsTimeStamp(resolution.TypeRef{Name: name}, table, "ts")).
				To(Equal(expected))
		},
		Entry("the TimeStamp itself", "telem.TimeStamp", true),
		Entry("an alias of it", "test.Stamp", true),
		Entry("a distinct type over an alias of it", "test.Created", true),
		Entry("an alias of a distinct type over it", "test.Moment", true),
		Entry("the TimeSpan", "telem.TimeSpan", false),
		Entry("another distinct type over the same primitive", "test.Count", false),
		Entry("the primitive", "int64", false),
		Entry("an unresolved reference", "test.Missing", false),
	)

	DescribeTable("TimeStampRef",
		func(name string, expected resolution.TypeRef, expectedOk bool) {
			ref, ok := telem.TimeStampRef(resolution.TypeRef{Name: name}, table, "ts")
			Expect(ok).To(Equal(expectedOk))
			Expect(ref).To(Equal(expected))
		},
		Entry("the TimeStamp itself", "telem.TimeStamp",
			resolution.TypeRef{Name: "telem.TimeStamp"}, true),
		Entry("an alias of a distinct type over it", "test.Moment",
			resolution.TypeRef{Name: "telem.TimeStamp"}, true),
		Entry("the TimeSpan", "telem.TimeSpan", resolution.TypeRef{}, false),
	)

	DescribeTable("IsTimeSpan",
		func(name, domainName string, expected bool) {
			Expect(telem.IsTimeSpan(resolution.TypeRef{Name: name}, table, domainName)).
				To(Equal(expected))
		},
		Entry("the TimeSpan itself", "telem.TimeSpan", "ts", true),
		Entry("the TimeStamp", "telem.TimeStamp", "ts", false),
		Entry("an alias of the TimeStamp", "test.Stamp", "ts", false),
		Entry(
			"an alias hand-written in the language",
			"telem.NumericTimeSpan", "ts", false,
		),
		Entry(
			"an alias hand-written in another language",
			"telem.NumericTimeSpan", "py", true,
		),
	)

	DescribeTable("IsRate",
		func(name string, expected bool) {
			Expect(telem.IsRate(resolution.TypeRef{Name: name}, table, "ts")).
				To(Equal(expected))
		},
		Entry("the Rate itself", "telem.Rate", true),
		Entry("the TimeSpan", "telem.TimeSpan", false),
	)
})
