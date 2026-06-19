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
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Derived New from @create", func() {
	var (
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		typesPlugin = types.New(types.DefaultOptions())
	})

	It("Should generate a z.input New that omits @output fields", func(ctx SpecContext) {
		source := `
			@ts output "client/ts/src/thing"

			Thing struct {
				key    uuid @key
				name   string
				author uuid @output
				@create
			}
		`
		resp := MustGenerate(ctx, source, "thing", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(
			"export const thingZ",
			`export interface New extends Omit<z.input<typeof thingZ>, "author"> {}`,
		)
		ExpectContent(resp, "types.gen.ts").ToNotContain("export const newZ")
	})

	It("Should derive a generic factory New that partials defaulted fields and keeps non-defaulted keys required", func(ctx SpecContext) {
		source := `
			@ts output "client/ts/src/thing"

			Thing struct<Properties extends record = record> {
				key        string         @key
				name       string
				configured bool = false
				properties Properties
				@create
				@ts concrete_types
			}
		`
		resp := MustGenerate(ctx, source, "thing", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(
			`optional.Optional<Thing<Properties>, "configured">`,
		)
		ExpectContent(resp, "types.gen.ts").ToNotContain("export const newZ")
	})

	It("Should substitute a base field's @create struct type with its New variant", func(ctx SpecContext) {
		loader.Add("schemas/bbase", `
			@ts output "client/ts/src/bbase"

			B struct<Data? = record> {
				task    string
				running bool = false
				@create
				@ts concrete_types
			}
		`)
		source := `
			import "schemas/bbase"

			@ts output "client/ts/src/athing"

			BAlias<Data? = record> = bbase.B<Data> {
			}

			A struct<StatusData? = record> {
				key   string @key
				name  string
				inner BAlias<StatusData>?
				@create
				@ts concrete_types
				@ts coalesce_type_params
			}
		`
		resp := MustGenerate(ctx, source, "athing", loader, typesPlugin)
		// The synthesized New omits the inner field (whose output type is BAlias)
		// and re-extends it with bbase.New so the consumer can omit B's defaulted
		// nested fields when building a creation payload.
		ExpectContent(resp, "types.gen.ts").ToContain(
			`export type New<S extends ASchemas = ASchemas> = Omit<A<S>, "inner"> & {`,
			`inner?: bbase.New<S["statusData"]>;`,
		)
		ExpectContent(resp, "types.gen.ts").ToNotContain("export const newZ")
	})

	It("Should derive New under its own name when the base is renamed", func(ctx SpecContext) {
		source := `
			@ts output "client/ts/src/thing"

			Thing struct {
				key    uuid @key
				name   string
				author uuid @output
				@create
				@ts name "Payload"
			}
		`
		resp := MustGenerate(ctx, source, "thing", loader, typesPlugin)
		// The derived New must emit under its own name (New), referencing the base's
		// renamed payloadZ schema, not re-emit a renamed payloadZ/Payload (which would
		// be a duplicate symbol with an empty body).
		ExpectContent(resp, "types.gen.ts").ToContain(
			"export const payloadZ = z.object(",
			`export interface New extends Omit<z.input<typeof payloadZ>, "author"> {}`,
		)
		ExpectContent(resp, "types.gen.ts").ToNotContain("export const newZ")
	})
})
