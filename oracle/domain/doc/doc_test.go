// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Get", func() {
	DescribeTable("extracts documentation from domains",
		func(domains map[string]resolution.Domain, expected string) {
			Expect(doc.Get(domains)).To(Equal(expected))
		},
		Entry("doc domain with string value",
			map[string]resolution.Domain{
				"doc": {Expressions: []resolution.Expression{{
					Name:   "value",
					Values: []resolution.ExpressionValue{{StringValue: "User represents a system user."}},
				}}},
			}, "User represents a system user."),
		Entry("doc domain with expression name only",
			map[string]resolution.Domain{
				"doc": {Expressions: []resolution.Expression{{Name: "Inline documentation"}}},
			}, "Inline documentation"),
		Entry("missing doc domain",
			map[string]resolution.Domain{
				"other": {Expressions: []resolution.Expression{{Name: "something"}}},
			}, ""),
		Entry("empty domains map", map[string]resolution.Domain{}, ""),
		Entry("nil domains map", nil, ""),
		Entry("doc domain with empty expressions",
			map[string]resolution.Domain{"doc": {Expressions: []resolution.Expression{}}}, ""),
		Entry("doc domain with empty values returns expression name",
			map[string]resolution.Domain{
				"doc": {Expressions: []resolution.Expression{{Name: "fallback", Values: []resolution.ExpressionValue{}}}},
			}, "fallback"),
		Entry("takes first expression when multiple present",
			map[string]resolution.Domain{
				"doc": {Expressions: []resolution.Expression{
					{Name: "first", Values: []resolution.ExpressionValue{{StringValue: "First doc"}}},
					{Name: "second", Values: []resolution.ExpressionValue{{StringValue: "Second doc"}}},
				}},
			}, "First doc"),
	)
})
