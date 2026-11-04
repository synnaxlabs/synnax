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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Resource Generation", func() {
	var (
		loader *MockFileLoader
		p      *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = types.New(types.DefaultOptions())
	})

	It("Should emit resource metadata with a uuid key", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Policy struct {
				key  uuid @key
				name string

				@resource type "policy"
			}
		`
		resp := MustGenerate(ctx, source, "policy", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`resource.createIDFactory`,
				`"policy"`,
				`resourceID`,
			)
	})

	It("Should emit resource metadata with a string key", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Label struct {
				key  string @key
				name string

				@resource type "label"
			}
		`
		resp := MustGenerate(ctx, source, "label", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`"label"`,
				`resourceID`,
			)
	})

	It("Should respect a @ts type override on the key field", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Item struct {
				key  uint64 @key { @ts type "string" }
				name string

				@resource type "item"
			}
		`
		resp := MustGenerate(ctx, source, "item", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`"item"`)
	})

	It(
		"Should respect a @ts type override on the key's distinct type definition",
		func(ctx SpecContext) {
			source := `
			@ts output "out"

			Key uint64 {
				@ts type "string"
			}

			Item struct {
				key  Key @key
				name string

				@resource type "item"
			}
		`
			resp := MustGenerate(ctx, source, "item", loader, p)
			ExpectContent(resp, "types.gen.ts").
				ToContain(`"item"`)
		},
	)

	It("Should emit resource metadata for a union", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Base struct {
				key uuid @key
			}

			View union on type extends Base {
				first {
					name string
				}
				second {
					size int32
				}

				@resource type "view"
			}
		`
		resp := MustGenerate(ctx, source, "view", loader, p)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`export const resourceID = resource.createIDFactory<Key>("view");`,
				`export const TYPE_RESOURCE_ID = resourceID("");`,
			)
	})

	It(
		"Should emit resource metadata for a union whose base has another output",
		func(ctx SpecContext) {
			source := `
			@ts output "out"

			Key = uuid

			Base struct {
				key Key @key

				@ts output "out/base"
			}

			View union on type extends Base {
				first {
					name string
				}

				@resource type "view"
			}
		`
			resp := MustGenerate(ctx, source, "view", loader, p)
			ExpectContent(resp, "out/types.gen.ts").
				ToContain(`export const resourceID = resource.createIDFactory<Key>("view");`)
		},
	)

	It(
		"Should not emit resource metadata for structs without @resource",
		func(ctx SpecContext) {
			source := `
			@ts output "out"

			Plain struct {
				key  uuid @key
				name string
			}
		`
			resp := MustGenerate(ctx, source, "plain", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).ToNot(ContainSubstring("resourceID"))
		},
	)
})
