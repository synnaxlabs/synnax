// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package naming_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
)

var _ = Describe("IsScreamingCase", func() {
	DescribeTable("should return true for all-uppercase identifiers",
		func(input string) {
			Expect(naming.IsScreamingCase(input)).To(BeTrue())
		},
		Entry("short acronym", "IR"),
		Entry("common acronym", "HTTP"),
		Entry("four letter acronym", "WASM"),
		Entry("with underscores", "MY_CONST"),
		Entry("single letter", "A"),
	)

	DescribeTable("should return false for non-screaming identifiers",
		func(input string) {
			Expect(naming.IsScreamingCase(input)).To(BeFalse())
		},
		Entry("pascal case", "UserName"),
		Entry("snake case", "user_name"),
		Entry("camel case", "userName"),
		Entry("lowercase", "http"),
		Entry("mixed with digits", "HTTP2"),
		Entry("empty string", ""),
		Entry("only underscores", "___"),
		Entry("contains space", "MY CONST"),
	)
})

var _ = Describe("DerivePackageName", func() {
	DescribeTable("should extract the last path segment",
		func(path, expected string) {
			Expect(naming.DerivePackageName(path)).To(Equal(expected))
		},
		Entry("nested path", "core/pkg/service/user", "user"),
		Entry("single segment", "user", "user"),
		Entry("deep nesting", "a/b/c/d/e", "e"),
	)
})

var _ = Describe("DerivePackageAlias", func() {
	It("should return the base name when no conflict exists", func() {
		Expect(naming.DerivePackageAlias("core/pkg/user", "channel")).To(Equal("user"))
	})

	It("should prepend parent directory when base name matches current package", func() {
		Expect(naming.DerivePackageAlias("core/pkg/channel", "channel")).To(Equal("pkgchannel"))
	})

	It("should handle single segment paths without conflict", func() {
		Expect(naming.DerivePackageAlias("user", "channel")).To(Equal("user"))
	})
})
