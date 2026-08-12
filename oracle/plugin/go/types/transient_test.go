// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/types"
)

var _ = Describe("Transient types", func() {
	var goPlugin *types.Plugin

	BeforeEach(func() { goPlugin = types.New(types.DefaultOptions()) })

	It(
		"Should generate unversioned typedefs at the package root",
		func(ctx SpecContext) {
			resp := chainGenerate(ctx, goPlugin, map[string]string{
				"schemas/synnax/thing.oracle": `
@go output "out"
Key uint32 {
	@ts type string
}
LocalKey uint20 {
}
Entry struct {
	key   string   {@key}
	local LocalKey
	@go marshal
}
APIEntry struct {
	@go output "api/out"
	key Key
	local LocalKey
}
`,
				"schemas/synnax/versions/thing/v0.oracle": `
Entry struct {
	key   string   {@key}
	local LocalKey
	@go marshal
}
`,
			}, "schemas/synnax/thing.oracle", "thing")
			var root string
			var count int
			for _, f := range resp.Files {
				if f.Path == "out/types.gen.go" {
					root = string(f.Content)
					count++
				}
			}
			Expect(count).To(Equal(1))
			Expect(root).To(ContainSubstring("type Key uint32"))
			Expect(root).To(ContainSubstring("type LocalKey types.Uint20"))
			Expect(root).To(ContainSubstring("type Entry = versions.Entry"))
			Expect(strings.Count(root, "package out")).To(Equal(1))
			Expect(strings.Count(root, "import")).To(Equal(1))
		},
	)

	It(
		"Should reference versioned types locally, never through versions/vN",
		func(ctx SpecContext) {
			resp := chainGenerate(ctx, goPlugin, map[string]string{
				"schemas/synnax/dep.oracle": `
@go output "dep"
Item struct {
	key string {@key}
	@go marshal
}
`,
				"schemas/synnax/versions/dep/v1.oracle": `
Item struct {
	key string {@key}
	@go marshal
}
`,
				"schemas/synnax/thing.oracle": `
import "schemas/synnax/dep"
@go output "out"
Entry struct {
	key string {@key}
	@go marshal
}
View struct {
	entry Entry
	item  dep.Item
}
`,
				"schemas/synnax/versions/thing/v0.oracle": `
Entry struct {
	key string {@key}
	@go marshal
}
`,
			}, "schemas/synnax/thing.oracle", "thing")
			var root string
			for _, f := range resp.Files {
				if f.Path == "out/types.gen.go" {
					root = string(f.Content)
				}
			}
			Expect(root).To(ContainSubstring("Entry Entry"))
			Expect(root).To(ContainSubstring("dep.Item"))
			Expect(root).ToNot(ContainSubstring("versions/v0"))
			Expect(root).ToNot(ContainSubstring("dep/versions/v1"))
		},
	)
})
