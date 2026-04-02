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
	"github.com/synnaxlabs/oracle/resolution"
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

var _ = Describe("ToPascalCase", func() {
	DescribeTable("should convert snake_case to PascalCase",
		func(input, expected string) {
			Expect(naming.ToPascalCase(input)).To(Equal(expected))
		},
		Entry("simple name", "user_name", "UserName"),
		Entry("single word", "name", "Name"),
		Entry("id acronym", "user_id", "UserID"),
		Entry("xy acronym", "sticky_xy", "StickyXY"),
		Entry("http acronym", "http_server", "HTTPServer"),
		Entry("url acronym", "base_url", "BaseURL"),
		Entry("json acronym", "json_data", "JSONData"),
		Entry("uuid acronym", "my_uuid", "MyUUID"),
		Entry("multiple acronyms", "http_url", "HTTPURL"),
		Entry("acronym at start", "id", "ID"),
		Entry("screaming case passthrough", "WASM", "WASM"),
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

	It("should use grandparent for migration version packages", func() {
		Expect(naming.DerivePackageAlias("arc/go/graph/migrations/v53", "v53")).To(Equal("graphv53"))
	})

	It("should disambiguate migration packages from different source packages", func() {
		Expect(naming.DerivePackageAlias("arc/go/graph/migrations/v53", "other")).To(Equal("graphv53"))
		Expect(naming.DerivePackageAlias("arc/go/ir/migrations/v53", "other")).To(Equal("irv53"))
		Expect(naming.DerivePackageAlias("core/pkg/service/arc/migrations/v53", "other")).To(Equal("arcv53"))
	})

	It("should handle full import paths with migrations pattern", func() {
		Expect(naming.DerivePackageAlias(
			"github.com/synnaxlabs/synnax/arc/go/graph/migrations/v53", "v53",
		)).To(Equal("graphv53"))
	})
})

var _ = Describe("LowerFirst", func() {
	DescribeTable("should lowercase the leading uppercase run",
		func(input, expected string) {
			Expect(naming.LowerFirst(input)).To(Equal(expected))
		},
		Entry("simple pascal", "Key", "key"),
		Entry("acronym prefix", "HTTPClient", "httpClient"),
		Entry("all uppercase", "ID", "id"),
		Entry("single char", "A", "a"),
		Entry("already lowercase", "key", "key"),
		Entry("empty string", "", ""),
		Entry("all uppercase long", "WASM", "wasm"),
	)

	It("should escape Go keywords after lowering", func() {
		Expect(naming.LowerFirst("Type")).To(Equal("typeVal"))
		Expect(naming.LowerFirst("Map")).To(Equal("mapVal"))
		Expect(naming.LowerFirst("String")).To(Equal("stringVal"))
	})
})

var _ = Describe("GetGoName", func() {
	It("should return PascalCase of the type name by default", func() {
		t := resolution.Type{Name: "user_key"}
		Expect(naming.GetGoName(t)).To(Equal("UserKey"))
	})

	It("should use the go name override when present", func() {
		t := resolution.Type{
			Name: "user_key",
			Domains: map[string]resolution.Domain{
				"go": {
					Expressions: []resolution.Expression{
						{Name: "name", Values: []resolution.ExpressionValue{{IdentValue: "CustomName"}}},
					},
				},
			},
		}
		Expect(naming.GetGoName(t)).To(Equal("CustomName"))
	})
})

var _ = Describe("GetFieldName", func() {
	It("should return PascalCase of the field name by default", func() {
		f := resolution.Field{Name: "created_at"}
		Expect(naming.GetFieldName(f)).To(Equal("CreatedAt"))
	})

	It("should use the go name override when present", func() {
		f := resolution.Field{
			Name: "created_at",
			Domains: map[string]resolution.Domain{
				"go": {
					Expressions: []resolution.Expression{
						{Name: "name", Values: []resolution.ExpressionValue{{IdentValue: "Timestamp"}}},
					},
				},
			},
		}
		Expect(naming.GetFieldName(f)).To(Equal("Timestamp"))
	})
})
