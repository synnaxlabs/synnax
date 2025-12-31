// Copyright 2025 Synnax Labs, Inc.
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
		func(domains map[string]*resolution.DomainEntry, expected string) {
			Expect(doc.Get(domains)).To(Equal(expected))
		},
		Entry("doc domain with string value",
			map[string]*resolution.DomainEntry{
				"doc": {Expressions: []*resolution.ExpressionEntry{{
					Name:   "value",
					Values: []resolution.ExpressionValue{{StringValue: "User represents a system user."}},
				}}},
			}, "User represents a system user."),
		Entry("doc domain with expression name only",
			map[string]*resolution.DomainEntry{
				"doc": {Expressions: []*resolution.ExpressionEntry{{Name: "Inline documentation"}}},
			}, "Inline documentation"),
		Entry("missing doc domain",
			map[string]*resolution.DomainEntry{
				"other": {Expressions: []*resolution.ExpressionEntry{{Name: "something"}}},
			}, ""),
		Entry("empty domains map", map[string]*resolution.DomainEntry{}, ""),
		Entry("nil domains map", nil, ""),
		Entry("doc domain with empty expressions",
			map[string]*resolution.DomainEntry{"doc": {Expressions: []*resolution.ExpressionEntry{}}}, ""),
		Entry("doc domain with empty values returns expression name",
			map[string]*resolution.DomainEntry{
				"doc": {Expressions: []*resolution.ExpressionEntry{{Name: "fallback", Values: []resolution.ExpressionValue{}}}},
			}, "fallback"),
		Entry("takes first expression when multiple present",
			map[string]*resolution.DomainEntry{
				"doc": {Expressions: []*resolution.ExpressionEntry{
					{Name: "first", Values: []resolution.ExpressionValue{{StringValue: "First doc"}}},
					{Name: "second", Values: []resolution.ExpressionValue{{StringValue: "Second doc"}}},
				}},
			}, "First doc"),
	)
})
