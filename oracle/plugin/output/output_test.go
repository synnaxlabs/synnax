// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package output_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("GetPath", func() {
	DescribeTable("extracts output path from type domains",
		func(domainName string, domains map[string]resolution.Domain, expected string) {
			entry := resolution.Type{Domains: domains, Form: resolution.StructForm{}}
			Expect(output.GetPath(entry, domainName)).To(Equal(expected))
		},
		Entry("go domain with output", "go",
			map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "core/pkg/user"}},
				}}},
			}, "core/pkg/user"),
		Entry("ts domain with output", "ts",
			map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "client/ts/user"}},
				}}},
			}, "client/ts/user"),
		Entry("py domain with output", "py",
			map[string]resolution.Domain{
				"py": {Expressions: []resolution.Expression{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "client/py/user"}},
				}}},
			}, "client/py/user"),
		Entry("missing domain", "go", map[string]resolution.Domain{}, ""),
		Entry("domain without output expression", "go",
			map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{{Name: "other"}}},
			}, ""),
		Entry("output expression without values", "go",
			map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{{Name: "output"}}},
			}, ""),
		Entry("multiple expressions takes first output", "go",
			map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{
					{Name: "package", Values: []resolution.ExpressionValue{{StringValue: "pkg"}}},
					{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "first"}}},
					{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "second"}}},
				}},
			}, "first"),
		Entry("nil domains map", "go", nil, ""),
	)
})

var _ = Describe("IsOmitted", func() {
	DescribeTable("checks for omit expression in domain",
		func(domainName string, domains map[string]resolution.Domain, expected bool) {
			typ := resolution.Type{Domains: domains}
			Expect(output.IsOmitted(typ, domainName)).To(Equal(expected))
		},
		Entry("has omit expression", "go",
			map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{{Name: "omit"}}},
			}, true),
		Entry("omit with values", "go",
			map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{
					{Name: "omit", Values: []resolution.ExpressionValue{{BoolValue: true}}},
				}},
			}, true),
		Entry("no omit expression", "go",
			map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{{Name: "output"}}},
			}, false),
		Entry("missing domain", "go", map[string]resolution.Domain{}, false),
		Entry("nil domains", "go", nil, false),
		Entry("different domain has omit", "ts",
			map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{{Name: "omit"}}},
			}, false),
	)
})

var _ = Describe("HasPB", func() {
	It("returns true when pb domain exists", func() {
		typ := resolution.Type{
			Domains: map[string]resolution.Domain{
				"pb": {},
			},
		}
		Expect(output.HasPB(typ)).To(BeTrue())
	})

	It("returns true when pb domain has expressions", func() {
		typ := resolution.Type{
			Domains: map[string]resolution.Domain{
				"pb": {Expressions: []resolution.Expression{{Name: "package"}}},
			},
		}
		Expect(output.HasPB(typ)).To(BeTrue())
	})

	It("returns false when no pb domain", func() {
		typ := resolution.Type{
			Domains: map[string]resolution.Domain{
				"go": {},
			},
		}
		Expect(output.HasPB(typ)).To(BeFalse())
	})

	It("returns false when domains is empty", func() {
		typ := resolution.Type{Domains: map[string]resolution.Domain{}}
		Expect(output.HasPB(typ)).To(BeFalse())
	})

	It("returns false when domains is nil", func() {
		typ := resolution.Type{}
		Expect(output.HasPB(typ)).To(BeFalse())
	})
})

var _ = Describe("GetPBPath", func() {
	It("returns go path with /pb suffix when pb domain exists", func() {
		typ := resolution.Type{
			Domains: map[string]resolution.Domain{
				"pb": {},
				"go": {Expressions: []resolution.Expression{
					{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/user"}}},
				}},
			},
		}
		Expect(output.GetPBPath(typ)).To(Equal("core/user/pb"))
	})

	It("returns empty when no pb domain", func() {
		typ := resolution.Type{
			Domains: map[string]resolution.Domain{
				"go": {Expressions: []resolution.Expression{
					{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/user"}}},
				}},
			},
		}
		Expect(output.GetPBPath(typ)).To(BeEmpty())
	})

	It("returns empty when pb exists but no go output path", func() {
		typ := resolution.Type{
			Domains: map[string]resolution.Domain{
				"pb": {},
				"go": {},
			},
		}
		Expect(output.GetPBPath(typ)).To(BeEmpty())
	})

	It("returns empty when pb exists but no go domain", func() {
		typ := resolution.Type{
			Domains: map[string]resolution.Domain{
				"pb": {},
			},
		}
		Expect(output.GetPBPath(typ)).To(BeEmpty())
	})
})
