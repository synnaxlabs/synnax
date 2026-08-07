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

	It(
		"Should generate a z.input New that omits @output fields",
		func(ctx SpecContext) {
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
		},
	)

	It(
		"Should derive a generic factory New that partials defaulted fields and keeps non-defaulted keys required",
		func(ctx SpecContext) {
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
		},
	)

	It(
		"Should re-project a conditional type-param field as z.input in the New",
		func(ctx SpecContext) {
			source := `
			@ts output "client/ts/src/status"

			Variant enum {
				ok = "ok"
				err = "err"
				@ts literals
			}

			Status struct<Details?, V extends Variant = Variant> {
				key     string = create { @key }
				name    string = ""
				variant V
				message string
				time    int64 = now
				details Details?
				@ts concrete_types
				@create
			}
		`
			resp := MustGenerate(ctx, source, "status", loader, typesPlugin)
			// The output type keeps z.infer for its conditional field.
			ExpectContent(resp, "types.gen.ts").ToContain(
				`} & ([Details] extends [z.ZodNever] ? {} : { details?: z.infer<Details> });`,
			)
			// The New strips the field from its optional.Optional base and re-appends
			// it as a z.input conditional.
			ExpectContent(resp, "types.gen.ts").ToContain(
				`export type New<Details extends z.ZodType = z.ZodNever, V extends z.ZodType<Variant> = typeof variantZ> = Omit<optional.Optional<Status<Details, V>, "key" | "name" | "time">, "details"> & ([Details] extends [z.ZodNever] ? {} : { details: z.input<Details> });`,
			)
			ExpectContent(resp, "types.gen.ts").ToNotContain("export const newZ")
		},
	)

	It(
		"Should project a nested @create field to its New with a non-generic details schema",
		func(ctx SpecContext) {
			loader.Add("schemas/status", `
			@ts output "client/ts/src/status"

			Status struct<Details?> {
				key     string = create { @key }
				name    string = ""
				details Details?
				@ts concrete_types
				@create
			}
		`)
			source := `
			import "schemas/status"

			@ts output "client/ts/src/device"

			DeviceStatus = status.Status<DeviceStatusDetails>

			DeviceStatusDetails struct {
				rack int64
				device string
			}

			Device struct<Properties extends record = record> {
				key        string = create { @key }
				configured bool = false
				properties Properties
				status     DeviceStatus?
				@ts concrete_types
				@create
			}
		`
			resp := MustGenerate(ctx, source, "device", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").ToContain(
				`export type New<Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>> = optional.Optional<Omit<Device<Properties>, "status">, "key" | "configured"> & {`,
				`status?: status.New<typeof deviceStatusDetailsZ>;`,
			)
		},
	)

	It(
		"Should project a nested @create field to its New with a generic details schema",
		func(ctx SpecContext) {
			loader.Add("schemas/status", `
			@ts output "client/ts/src/status"

			Status struct<Details?> {
				key     string = create { @key }
				name    string = ""
				details Details?
				@ts concrete_types
				@create
			}
		`)
			source := `
			import "schemas/status"

			@ts output "client/ts/src/task"

			StatusDetails struct<Data? = record> {
				task    string
				running bool
				data    Data?
				@ts concrete_types
			}

			TaskStatus<Data? = record> = status.Status<StatusDetails<Data>>

			Task struct<Config extends record = record, StatusData? = record> {
				key      string = create { @key }
				name     string
				config   Config
				status   TaskStatus<StatusData>?
				@ts {
					concrete_types
					coalesce_type_params
				}
				@create
			}
		`
			resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").ToContain(
				`export type New<S extends TaskSchemas = TaskSchemas> = optional.Optional<Omit<Task<S>, "status">, "key"> & {`,
				`status?: status.New<ReturnType<typeof statusDetailsZ>>;`,
			)
		},
	)

	It(
		"Should derive a z.input New<Name> alias for a @create union",
		func(ctx SpecContext) {
			source := `
			@ts output "client/ts/src/tab"

			TabBase struct {
				key uuid = create { @key }
			}
			Tab union on variant extends TabBase {
				resource {
					resource string
				}
				view {
					type string
				}
				@create
			}
		`
			resp := MustGenerate(ctx, source, "tab", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").ToContain(
				"export const tabZ = z.discriminatedUnion(",
				"export type NewTab = z.input<typeof tabZ>;",
			)
			ExpectContent(resp, "types.gen.ts").ToNotContain(
				"export const newTabZ",
				"interface NewTab",
			)
		},
	)

	It(
		"Should derive New under its own name when the base is renamed",
		func(ctx SpecContext) {
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
			// The derived New must emit under its own name (New), referencing the
			// base's renamed payloadZ schema, not re-emit a renamed payloadZ/Payload
			// (which would be a duplicate symbol with an empty body).
			ExpectContent(resp, "types.gen.ts").ToContain(
				"export const payloadZ = z.object(",
				`export interface New extends Omit<z.input<typeof payloadZ>, "author"> {}`,
			)
			ExpectContent(resp, "types.gen.ts").ToNotContain("export const newZ")
		},
	)
})
