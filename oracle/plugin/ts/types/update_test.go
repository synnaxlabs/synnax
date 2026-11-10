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

var _ = Describe("Update types", func() {
	var (
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/patch", `
			@go output "x/go/patch"
			@ts output "x/ts/src/patch"

			Set struct { path string[] }
			Remove struct { path string[] }
			Insert struct { path string[] }
			Move struct { path string[] }
			Change struct<T> {
				value T?
				@presence
				@ts hand
				@ts include "x/ts/src/patch/field"
			}
			TypedMap struct<K extends comparable, V, U> {
				set    map<K, V>?
				update map<K, U>?
				@ts hand
				@ts include "x/ts/src/patch/field"
			}
		`)
		typesPlugin = types.New(types.DefaultOptions())
	})

	const record = `
		import "schemas/x/patch"

		@go output "core/pkg/service/plot"
		@ts output "client/ts/src/plot"

		Plot struct<Config extends record = record> {
			key    uuid
			name   string              { @patch }
			note   string?             { @patch }
			layout record              { @patch }
			tags   map<string, string> { @patch }
			config Config              { @patch }
			hash   string              { @patch core }

			@actions
			@ts concrete_types
		}
	`

	It("Should take null only in a field that changes an optional value", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, record, "plot", loader, typesPlugin)
		ExpectContent(resp, "client/ts/src/plot/types.gen.ts").
			ToContain(
				"name: z.string().optional(),",
				"note: patch.changeZ(z.string()).optional(),",
				"layout: record.unknownZ().optional(),",
				"tags: patch.typedMapZ({k: z.string(), v: z.string(), u: z.string()})"+
					".optional(),",
				"config: (config == null ? record.unknownZ() : "+
					"patch.structZ(config)).optional(),",
			).
			ToNotContain("nullToUndefined")
	})

	It("Should forbid the variant fields in an update without the type", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, `
			import "schemas/x/patch"

			@go output "core/pkg/service/view"
			@ts output "client/ts/src/view"

			Base struct {
				key  uuid
				name string { @patch }
			}

			View union on type extends Base {
				range {
					where string? { @patch }
				}
				status {
					level int32 { @patch }
				}
				@actions
			}
		`, "view", loader, typesPlugin)
		ExpectContent(resp, "client/ts/src/view/types.gen.ts").ToContain(
			`export const updateZ = z.union([baseUpdateZ.strict(), `+
				`z.discriminatedUnion("type", [`,
			"export type Update = (BaseUpdate & { type?: never; where?: never; "+
				"level?: never; }) | RangeUpdate | StatusUpdate;",
		)
	})

	It("Should derive the input of a client from the update", func(ctx SpecContext) {
		resp := MustGenerate(ctx, record, "plot", loader, typesPlugin)
		ExpectContent(resp, "client/ts/src/plot/types.gen.ts").ToContain(
			"export type UpdateInput<Config extends "+
				"patch.StructSchemas<z.ZodType, z.ZodType<record.Unknown>> = "+
				"patch.StructSchemas<z.ZodType<record.Unknown>, "+
				`z.ZodType<record.Unknown>>> = Omit<Update<Config>, "hash">;`,
		)
	})

	It("Should map each task type to the schemas of its config", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, `
			import "schemas/x/patch"

			@go output "core/pkg/service/ni"
			@ts output "client/ts/src/ni"

			AnalogReadConfig struct {
				key  uuid
				rate float64

				@patch
				@task type "ni_analog_read"
			}
		`, "ni", loader, typesPlugin)
		ExpectContent(resp, "client/ts/src/ni/types.gen.ts").ToContain(
			`export const ANALOG_READ_TASK_TYPE = "ni_analog_read";`,
			"export const CONFIG_SCHEMAS = {\n"+
				"  [ANALOG_READ_TASK_TYPE]: "+
				"{ u: analogReadConfigUpdateZ, t: analogReadConfigZ },\n};",
		)
	})

	It("Should take the struct schemas of a record type parameter", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, record, "plot", loader, typesPlugin)
		ExpectContent(resp, "client/ts/src/plot/types.gen.ts").ToContain(
			"export const updateZ = <Config extends "+
				"patch.StructSchemas<z.ZodType, z.ZodType<record.Unknown>> = "+
				"patch.StructSchemas<z.ZodType<record.Unknown>, "+
				"z.ZodType<record.Unknown>>>(config?: Config) =>",
			`  config?: patch.Struct<Config["u"], Config["t"]>;`,
		)
	})
})
