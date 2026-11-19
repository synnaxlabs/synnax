// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package selector_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/selector"
)

var _ = Describe("IsName", func() {
	DescribeTable("Should accept a selector that names shapes",
		func(sel string) { Expect(selector.IsName(sel)).To(BeTrue()) },
		Entry("an id", "#body"),
		Entry("a class", ".a"),
		Entry("classes", ".a.b"),
		Entry("a data-region-id", `[data-region-id="region-1"]`),
		Entry("a data-region-id with an escaped quote", `[data-region-id="a\"b"]`),
		Entry("a hex escape", `#\31 23`),
		Entry("a literal escape", `#a\.b`),
		Entry("two hyphens", "#--x"),
		Entry("a hyphen and a letter", "#-a"),
		Entry("an underscore", "#_a"),
		Entry("a non-ASCII letter", "#é"),
	)

	DescribeTable("Should refuse a selector that does not name shapes",
		func(sel string) { Expect(selector.IsName(sel)).To(BeFalse()) },
		Entry("an empty selector", ""),
		Entry("an empty id", "#"),
		Entry("a leading digit", "#1a"),
		Entry("a class with a leading digit", ".a.1a"),
		Entry("a hyphen and a digit", "#-1"),
		Entry("a combinator", "g > #a"),
		Entry("a tag", "rect"),
		Entry("a position", "rect:nth-of-type(1)"),
		Entry("a comma", "#a,b"),
		Entry("a data-region-id with a bare quote", `[data-region-id="a"b"]`),
	)
})
