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
		Entry("iso acronym", "iso", "ISO"),
		Entry("utc acronym", "utc", "UTC"),
		Entry("word starting with an acronym", "slave", "Slave"),
		Entry("word containing an acronym", "ratio", "Ratio"),
		Entry("plural acronym", "ids", "IDs"),
		Entry("acronym with a digit suffix", "iso8601", "ISO8601"),
		Entry("utf8 acronym", "utf8", "UTF8"),
	)

	DescribeTable("should pass screaming-case identifiers through unchanged",
		func(input string) {
			Expect(naming.ToPascalCase(input)).To(Equal(input))
		},
		Entry("short acronym", "IR"),
		Entry("common acronym", "HTTP"),
		Entry("four letter acronym", "WASM"),
		Entry("with underscores", "MY_CONST"),
		Entry("single letter", "A"),
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

var _ = Describe("AssumedImportName", func() {
	DescribeTable("should return the name goimports binds the path to",
		func(path, expected string) {
			Expect(naming.AssumedImportName(path)).To(Equal(expected))
		},
		Entry("plain path", "github.com/synnaxlabs/x/telem", "telem"),
		Entry("module major version", "encoding/json/v2", "json"),
		Entry("module major version above nine", "github.com/a/b/v12", "b"),
		Entry("version directory", "core/pkg/task/versions/v2", "versions"),
		Entry("single segment", "context", "context"),
	)
})

var _ = Describe("DerivePackageAlias", func() {
	It("should return the base name when no conflict exists", func() {
		Expect(naming.DerivePackageAlias("core/pkg/user", "channel")).To(Equal("user"))
	})

	It(
		"should prepend parent directory when base name matches current package",
		func() {
			Expect(
				naming.DerivePackageAlias("core/pkg/channel", "channel"),
			).To(Equal("pkgchannel"))
		},
	)

	It("should handle single segment paths without conflict", func() {
		Expect(naming.DerivePackageAlias("user", "channel")).To(Equal("user"))
	})

	It("should alias version packages to the resource name", func() {
		Expect(
			naming.DerivePackageAlias("x/go/spatial/versions/v0", "v0"),
		).To(Equal("spatial"))
		Expect(
			naming.DerivePackageAlias("arc/go/graph/migrations/v53", "v53"),
		).To(Equal("graph"))
		Expect(
			naming.DerivePackageAlias("arc/go/ir/versions/v2", "other"),
		).To(Equal("ir"))
	})

	It("should handle full import paths with migrations pattern", func() {
		Expect(naming.DerivePackageAlias(
			"github.com/synnaxlabs/synnax/arc/go/graph/migrations/v53", "v53",
		)).To(Equal("graph"))
	})

	It("should keep the versioned form for a resource named types", func() {
		Expect(
			naming.DerivePackageAlias("arc/go/types/versions/v1", "v2"),
		).To(Equal("types"))
	})
})

var _ = Describe("DeriveVersionedAlias", func() {
	It("should keep the version suffix on version packages", func() {
		Expect(
			naming.DeriveVersionedAlias("x/go/spatial/versions/v0", "v0"),
		).To(Equal("spatialv0"))
		Expect(
			naming.DeriveVersionedAlias("arc/go/graph/migrations/v53", "v53"),
		).To(Equal("graphv53"))
		Expect(
			naming.DeriveVersionedAlias("arc/go/ir/migrations/v53", "other"),
		).To(Equal("irv53"))
		Expect(
			naming.DeriveVersionedAlias("core/pkg/service/arc/migrations/v53", "other"),
		).To(Equal("arcv53"))
	})

	It("should keep the version suffix for a resource named types", func() {
		Expect(
			naming.DeriveVersionedAlias("arc/go/types/versions/v1", "v2"),
		).To(Equal("typesv1"))
	})

	It("should fall back to base-name rules for unversioned paths", func() {
		Expect(
			naming.DeriveVersionedAlias("core/pkg/user", "channel"),
		).To(Equal("user"))
		Expect(
			naming.DeriveVersionedAlias("core/pkg/channel", "channel"),
		).To(Equal("pkgchannel"))
	})
})

var _ = Describe("VersionAliases", func() {
	const current = "github.com/synnaxlabs/arc/graph/versions/v1"

	It("Should give the newest version the bare name and older ones a suffix", func() {
		Expect(naming.VersionAliases([]string{
			"github.com/synnaxlabs/arc/types/versions/v1",
			"github.com/synnaxlabs/arc/ir/versions/v0",
			"github.com/synnaxlabs/arc/types/versions/v0",
			"github.com/synnaxlabs/arc/ir/versions/v1",
		}, current)).To(Equal(map[string]string{
			"github.com/synnaxlabs/arc/ir/versions/v0":    "irv0",
			"github.com/synnaxlabs/arc/ir/versions/v1":    "ir",
			"github.com/synnaxlabs/arc/types/versions/v0": "typesv0",
			"github.com/synnaxlabs/arc/types/versions/v1": "types",
		}))
	})

	It("Should compare versions by number, not text", func() {
		Expect(naming.VersionAliases([]string{
			"github.com/synnaxlabs/x/color/versions/v10",
			"github.com/synnaxlabs/x/color/versions/v9",
		}, current)).To(Equal(map[string]string{
			"github.com/synnaxlabs/x/color/versions/v10": "color",
			"github.com/synnaxlabs/x/color/versions/v9":  "colorv9",
		}))
	})

	It("Should suffix every version when the resource's root is imported", func() {
		Expect(naming.VersionAliases([]string{
			"github.com/synnaxlabs/synnax/core/status",
			"github.com/synnaxlabs/synnax/core/status/versions/v1",
		}, current)).To(Equal(map[string]string{
			"github.com/synnaxlabs/synnax/core/status/versions/v1": "statusv1",
		}))
	})

	It("Should prefix resources that share a name with their parent directory", func() {
		Expect(naming.VersionAliases([]string{
			"github.com/synnaxlabs/synnax/core/arc/task/versions/v0",
			"github.com/synnaxlabs/synnax/core/arc/task/versions/v1",
			"github.com/synnaxlabs/synnax/core/rack/task/versions/v0",
			"github.com/synnaxlabs/synnax/core/rack/task/versions/v1",
		}, current)).To(Equal(map[string]string{
			"github.com/synnaxlabs/synnax/core/arc/task/versions/v0":  "arctaskv0",
			"github.com/synnaxlabs/synnax/core/arc/task/versions/v1":  "arctask",
			"github.com/synnaxlabs/synnax/core/rack/task/versions/v0": "racktaskv0",
			"github.com/synnaxlabs/synnax/core/rack/task/versions/v1": "racktask",
		}))
	})

	It("Should prefix a shared name even when one resource has one version", func() {
		Expect(naming.VersionAliases([]string{
			"github.com/synnaxlabs/synnax/core/arc/task/versions/v0",
			"github.com/synnaxlabs/synnax/core/rack/task/versions/v1",
		}, current)).To(Equal(map[string]string{
			"github.com/synnaxlabs/synnax/core/arc/task/versions/v0":  "arctask",
			"github.com/synnaxlabs/synnax/core/rack/task/versions/v1": "racktask",
		}))
	})

	It("Should suffix every version when another package takes the name", func() {
		Expect(naming.VersionAliases([]string{
			"github.com/synnaxlabs/pluto/color",
			"github.com/synnaxlabs/x/color/versions/v0",
		}, current)).To(Equal(map[string]string{
			"github.com/synnaxlabs/x/color/versions/v0": "colorv0",
		}))
	})

	It("Should leave out a resource imported at one version", func() {
		Expect(naming.VersionAliases([]string{
			"github.com/synnaxlabs/x/color/versions/v0",
			"github.com/synnaxlabs/x/encoding/orc",
		}, current)).To(BeEmpty())
	})

	It("Should leave out the predecessors of the file's own resource", func() {
		Expect(naming.VersionAliases([]string{
			"github.com/synnaxlabs/arc/graph/versions/v0",
			"github.com/synnaxlabs/arc/ir/versions/v0",
		}, current)).To(BeEmpty())
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
	It("should return the declared type name verbatim by default", func() {
		t := resolution.Type{Name: "UserKey"}
		Expect(naming.GetGoName(t)).To(Equal("UserKey"))
	})

	It("should preserve acronyms in the declared name", func() {
		t := resolution.Type{Name: "RTDType"}
		Expect(naming.GetGoName(t)).To(Equal("RTDType"))
	})

	It("should use the go name override when present", func() {
		t := resolution.Type{
			Name: "user_key",
			Domains: map[string]resolution.Domain{
				"go": {
					Expressions: []resolution.Expression{
						{
							Name: "name",
							Values: []resolution.ExpressionValue{
								{IdentValue: "CustomName"},
							},
						},
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
						{
							Name: "name",
							Values: []resolution.ExpressionValue{
								{IdentValue: "Timestamp"},
							},
						},
					},
				},
			},
		}
		Expect(naming.GetFieldName(f)).To(Equal("Timestamp"))
	})
})
