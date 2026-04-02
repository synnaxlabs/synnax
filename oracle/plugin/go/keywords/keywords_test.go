// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package keywords_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/keywords"
)

var _ = Describe("Reserved", func() {
	DescribeTable("should contain Go keywords",
		func(keyword string) {
			Expect(keywords.Reserved.Contains(keyword)).To(BeTrue())
		},
		Entry("break", "break"),
		Entry("func", "func"),
		Entry("return", "return"),
		Entry("select", "select"),
		Entry("struct", "struct"),
		Entry("type", "type"),
		Entry("var", "var"),
		Entry("map", "map"),
		Entry("range", "range"),
		Entry("chan", "chan"),
	)

	DescribeTable("should contain predeclared types",
		func(name string) {
			Expect(keywords.Reserved.Contains(name)).To(BeTrue())
		},
		Entry("bool", "bool"),
		Entry("string", "string"),
		Entry("int", "int"),
		Entry("error", "error"),
		Entry("any", "any"),
		Entry("comparable", "comparable"),
	)

	DescribeTable("should contain predeclared constants",
		func(name string) {
			Expect(keywords.Reserved.Contains(name)).To(BeTrue())
		},
		Entry("true", "true"),
		Entry("false", "false"),
		Entry("nil", "nil"),
		Entry("iota", "iota"),
	)

	DescribeTable("should contain predeclared functions",
		func(name string) {
			Expect(keywords.Reserved.Contains(name)).To(BeTrue())
		},
		Entry("make", "make"),
		Entry("len", "len"),
		Entry("append", "append"),
		Entry("delete", "delete"),
		Entry("panic", "panic"),
		Entry("recover", "recover"),
	)

	It("should not contain non-reserved names", func() {
		Expect(keywords.Reserved.Contains("User")).To(BeFalse())
		Expect(keywords.Reserved.Contains("myVar")).To(BeFalse())
		Expect(keywords.Reserved.Contains("foo")).To(BeFalse())
	})
})

var _ = Describe("Escape", func() {
	DescribeTable("should append Val to reserved words",
		func(input, expected string) {
			Expect(keywords.Escape(input)).To(Equal(expected))
		},
		Entry("type keyword", "type", "typeVal"),
		Entry("map keyword", "map", "mapVal"),
		Entry("string type", "string", "stringVal"),
		Entry("error type", "error", "errorVal"),
		Entry("nil constant", "nil", "nilVal"),
		Entry("make function", "make", "makeVal"),
	)

	DescribeTable("should not modify non-reserved names",
		func(input string) {
			Expect(keywords.Escape(input)).To(Equal(input))
		},
		Entry("regular name", "user"),
		Entry("pascal case", "UserName"),
		Entry("camel case", "userName"),
		Entry("empty string", ""),
	)
})
