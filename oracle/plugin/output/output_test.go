// Copyright 2025 Synnax Labs, Inc.
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
	DescribeTable("extracts output path from struct domains",
		func(domainName string, domains map[string]*resolution.DomainEntry, expected string) {
			entry := &resolution.StructEntry{Domains: domains}
			Expect(output.GetPath(entry, domainName)).To(Equal(expected))
		},
		Entry("go domain with output", "go",
			map[string]*resolution.DomainEntry{
				"go": {Expressions: []*resolution.ExpressionEntry{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "core/pkg/user"}},
				}}},
			}, "core/pkg/user"),
		Entry("ts domain with output", "ts",
			map[string]*resolution.DomainEntry{
				"ts": {Expressions: []*resolution.ExpressionEntry{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "client/ts/user"}},
				}}},
			}, "client/ts/user"),
		Entry("py domain with output", "py",
			map[string]*resolution.DomainEntry{
				"py": {Expressions: []*resolution.ExpressionEntry{{
					Name:   "output",
					Values: []resolution.ExpressionValue{{StringValue: "client/py/user"}},
				}}},
			}, "client/py/user"),
		Entry("missing domain", "go", map[string]*resolution.DomainEntry{}, ""),
		Entry("domain without output expression", "go",
			map[string]*resolution.DomainEntry{
				"go": {Expressions: []*resolution.ExpressionEntry{{Name: "other"}}},
			}, ""),
		Entry("output expression without values", "go",
			map[string]*resolution.DomainEntry{
				"go": {Expressions: []*resolution.ExpressionEntry{{Name: "output"}}},
			}, ""),
		Entry("multiple expressions takes first output", "go",
			map[string]*resolution.DomainEntry{
				"go": {Expressions: []*resolution.ExpressionEntry{
					{Name: "package", Values: []resolution.ExpressionValue{{StringValue: "pkg"}}},
					{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "first"}}},
					{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "second"}}},
				}},
			}, "first"),
		Entry("nil domains map", "go", nil, ""),
	)
})
