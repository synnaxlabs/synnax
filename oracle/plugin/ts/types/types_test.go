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
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin TS Types Suite", SuiteConfig())
}

var _ = Describe("TS Types Plugin", func() {
	var (
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		typesPlugin = types.New(types.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(typesPlugin.Name()).To(Equal("ts/types"))
		})

		It("Should have no domain filter", func() {
			Expect(typesPlugin.Domains()).To(BeEmpty())
		})

		It("Should have no dependencies", func() {
			Expect(typesPlugin.Requires()).To(BeNil())
		})
	})

	Describe("Generate", func() {
		It("Should generate the recursive Where of a filter", func() {
			AddFilterSchema(loader)
			resp := MustGenerate(ThingFilter, "thing", loader, typesPlugin)
			ExpectContent(resp, "thing/types.gen.ts").ToContain(
				"export type Where = {",
				"and?: Where[];",
				"or?: Where[];",
				"not?: Where;",
				"export const whereZ: z.ZodType<Where> = z.strictObject({",
				"get and() {\n    return zod.nullToUndefined(whereZ.array());",
				"get not() {\n    return whereZ.optional();",
			)
		})

		It("Should reject unknown keys only in a filter request", func() {
			AddFilterSchema(loader)
			resp := MustGenerate(ThingFilter, "thing", loader, typesPlugin)
			ExpectContent(resp, "thing/types.gen.ts").
				ToContain("export const retrieveRequestZ = z.strictObject({").
				ToNotContain("thingZ = z.strictObject", "retrieveResponseZ = z.strictObject")
		})

		It("Should make every operator type strict, used or not", func() {
			AddFilterSchema(loader)
			resp := MustGenerate(ThingFilter, "thing", loader, typesPlugin)
			ExpectContent(resp, "filter/types.gen.ts").ToContain(
				"export const stringZ = z.strictObject({",
				"export const boolZ = z.strictObject({",
				"export const setZ = <T extends z.ZodType>(t: T) =>\n  z.strictObject({",
				"export const orderedZ = <T extends z.ZodType>(t: T) =>\n  z.strictObject({",
				"export const containsZ = <T extends z.ZodType>(t: T) =>\n  z.strictObject({",
			)
		})

		It("Should leave the defaulted fields of a filter request to the Core", func() {
			AddFilterSchema(loader)
			source := strings.Replace(
				ThingFilter,
				"name string",
				"name string { @search }",
				1,
			)
			resp := MustGenerate(source, "thing", loader, typesPlugin)
			ExpectContent(resp, "thing/types.gen.ts").
				ToContain(
					"searchTerm: z.string().optional(),",
					"export type RetrieveRequest = z.infer<typeof retrieveRequestZ>;",
					"export interface RetrieveResponse extends z.infer<typeof retrieveResponseZ> {}",
				).
				ToNotContain(`.default("")`, "optional.Optional")
		})

		It("Should leave the defaulted fields of an operator type to the Core", func() {
			AddFilterSchema(loader)
			resp := MustGenerate(ThingFilter, "thing", loader, typesPlugin)
			ExpectContent(resp, "filter/types.gen.ts").
				ToContain("caseInsensitive: z.boolean().optional(),").
				ToNotContain(".default(false)")
		})

		Context("field-level omit", func() {
			It("Should leave out an inherited field the parent omits", func() {
				source := `
					@go output "core/pkg/service/auth"
					@ts output "out"

					Parent struct {
						name   string
						secret bytes { @ts omit }
					}

					Child struct extends Parent {
						secret?
						extra  string
					}
				`
				resp := MustGenerate(source, "auth", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("export const childZ", "extra: z.string()").
					ToNotContain("secret")
			})

			It("Should leave out a field omitted in ts", func() {
				source := `
					@go output "core/pkg/service/auth"
					@ts output "out"

					Session struct {
						key     uuid
						hash    bytes { @ts omit }
						subject string
					}
				`
				resp := MustGenerate(source, "auth", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`key: z.uuid()`, `subject: z.string()`).
					ToNotContain("hash")
			})
		})

		Context("basic struct generation", func() {
			It("Should generate schema for simple struct", func() {
				source := `
					@ts output "out"

					User struct {
						key uuid
						name string
						age int32
						active bool
					}
				`
				resp := MustGenerate(source, "user", loader, typesPlugin)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`import { z } from "zod"`,
						`export const userZ = z.object(`,
						`key: z.uuid()`,
						`name: z.string()`,
						`age: z.int32()`,
						`active: z.boolean()`,
						`export interface User extends z.infer<typeof userZ> {}`,
					)
			})
		})

		It(
			"Should generate a typeless override identically to a full restatement",
			func() {
				gen := func(childBody string) string {
					source := `
					@ts output "out"

					Parent struct {
						name  string
						count int32 = 5
						tag   string
					}

					Child struct extends Parent {
						` + childBody + `
					}
				`
					resp := MustGenerate(source, "user", loader, typesPlugin)
					return string(resp.Files[0].Content)
				}
				// A typeless override desugars to the equivalent full restatement, so
				// the generated schema is byte-identical (here, the parent's
				// .partial()/.extend() merge chain rather than a flat object).
				Expect(gen("count = 10")).To(Equal(gen("count int32 = 10")))
				Expect(gen("tag?")).To(Equal(gen("tag string?")))
			},
		)

		It(
			"Should flatten a struct that removes an inherited domain",
			func() {
				source := `
				@ts output "out"

				Parent struct {
					name string @validate { min_length 1 }
				}

				Child struct extends Parent {
					name -@validate
				}
			`
				resp := MustGenerate(source, "user", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`export const childZ = z.object(`, `name: z.string()`)
			},
		)

		Context("parent omitted in TypeScript", func() {
			schema := func(parentDomain, childBody string) string {
				return `
				@go output "core/user"
				@ts output "out"

				Record struct {
					key    uint32
					name   string
					secret string
					` + parentDomain + `
				}

				Shape struct extends Record {
					` + childBody + `
				}
			`
			}

			It(
				"Should declare an inline variant's inherited fields",
				func() {
					source := `
					@go output "core/pkg/service/auth"
					@ts output "out"

					Record struct {
						key  uint32
						name string
						@ts omit
					}

					Chan union on type {
						ai extends Record {
							rate float64
						}
					}
				`
					resp := MustGenerate(source, "auth", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain("key: z.uint32()", "name: z.string()", "rate: z.number()").
						ToNotContain("Record")
				},
			)

			It("Should declare the inherited fields", func() {
				resp := MustGenerate(
					schema("@ts omit", "rate float64"),
					"user",
					loader,
					typesPlugin,
				)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`export const shapeZ = z.object({`,
						`key: z.uint32()`,
						`name: z.string()`,
						`secret: z.string()`,
						`rate: z.number()`,
					).
					ToNotContain(`recordZ`)
			})

			It("Should leave out a removed field", func() {
				resp := MustGenerate(
					schema("@ts omit", "-secret"), "user", loader, typesPlugin,
				)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`export const shapeZ = z.object({`, `name: z.string()`).
					ToNotContain(`secret`, `recordZ`)
			})

			It("Should extend a parent that generates", func() {
				resp := MustGenerate(
					schema("", "-secret"), "user", loader, typesPlugin,
				)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`export const shapeZ = recordZ`, `.omit({ secret: true })`)
			})
		})

		It("Should handle optional and array types", func() {
			source := `
				@ts output "out"

				Range struct {
					key uuid
					labels uuid[]
					parent uuid?
					tags string[]?
				}
			`
			table, diag := Analyze(source, "ranger", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Required arrays default a missing array to [] (the server omits a nil
			// array) while still validating a present array.
			Expect(
				content,
			).To(ContainSubstring(`labels: z.uuid().array().default(() => [])`))
			Expect(content).To(ContainSubstring(`parent: z.uuid().optional()`))
			// Optional arrays use nullToUndefined with array schema
			Expect(
				content,
			).To(ContainSubstring(`tags: zod.nullToUndefined(z.string().array())`))
		})

		It("Should apply validation rules", func() {
			source := `
				@ts output "out"

				User struct {
					name string @validate {
						min_length 1
						max_length 255
					}
					email string
					age int32 @validate {
						min 0
						max 150
					}
				}
			`
			table, diag := Analyze(source, "user", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(
				content,
			).To(ContainSubstring(`name: z.string().min(1).check(zod.maxLength(255))`))
			Expect(content).To(ContainSubstring(`email: z.string()`))
			Expect(content).To(ContainSubstring(`age: z.int32().min(0).max(150)`))
		})

		It(
			"Should classify a distinct numeric type by its primitive base",
			func() {
				source := `
				@ts output "out"

				Key uint32

				Device struct {
					rack Key @validate { min 1 }
				}
			`
				resp := MustGenerate(source, "device", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").ToContain("rack: keyZ.min(1)")
			},
		)

		It(
			"Should apply numeric rules to a field overridden to number",
			func() {
				source := `
				@ts output "out"

				Rate float64 {
					@ts hand
					@validate min 0
				}

				Config struct {
					rate Rate = 10 {
						@ts type "number"
						@validate gt 0
					}
					fallback Rate {
						@ts type "number"
					}
				}
			`
				resp := MustGenerate(source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").ToContain(
					"rate: z.number().gt(0).default(10)",
					"fallback: z.number().min(0)",
				)
			},
		)

		It(
			"Should apply the rules of a hand-written type to a field",
			func() {
				source := `
				@ts output "out"

				Rate float64 {
					@ts hand
					@ts include "out/rate"
					@validate max 100
				}

				Config struct {
					rate Rate
				}
			`
				resp := MustGenerate(source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("rate: rateZ.check(z.lte(100))")
			},
		)

		It(
			"Should apply numeric rules to a type overridden to number",
			func() {
				source := `
				@ts output "out"

				Rate float64 {
					@ts type "number"
					@validate gt 0
				}
			`
				resp := MustGenerate(source, "rate", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").ToContain(
					"export const rateZ = z.number().gt(0);",
				)
			},
		)

		It(
			"Should apply a base chain's rules to a type overridden to number",
			func() {
				source := `
				@ts output "out"

				Rate float64 {
					@validate gt 0
				}

				Hz = Rate {
					@validate max 100
				}

				Fast Hz {
					@ts type "number"
				}
			`
				resp := MustGenerate(source, "rate", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").ToContain(
					"export const fastZ = z.number().gt(0).max(100);",
				)
			},
		)

		It(
			"Should leave a generated type's rules to its own schema",
			func() {
				source := `
				@ts output "out"

				Decimal = float64 {
					@validate min 0
					@validate max 1
				}

				Panel struct {
					size Decimal
				}
			`
				resp := MustGenerate(source, "panel", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").ToContain(
					"export const decimalZ = z.number().min(0).max(1);",
					"size: decimalZ,",
				)
			},
		)

		It(
			"Should treat required on a numeric type as non-zero",
			func() {
				source := `
				@ts output "out"

				Key uint32

				Device struct {
					rack Key @validate { required }
				}
			`
				resp := MustGenerate(source, "device", loader, typesPlugin)
				ExpectContent(
					resp,
					"types.gen.ts",
				).ToContain(`rack: keyZ.refine((v) => v !== 0, "rack is required")`)
			},
		)

		It("Should generate enums", func() {
			source := `
				@ts output "out"

				TaskState enum {
					pending = 0
					running = 1
					completed = 2
				}

				Task struct {
					state TaskState
				}
			`
			table, diag := Analyze(source, "task", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Int enums generate a TypeScript enum and z.enum(EnumName)
			Expect(content).To(ContainSubstring(`export enum TaskState`))
			Expect(content).To(ContainSubstring(`pending = 0`))
			Expect(content).To(ContainSubstring(`running = 1`))
			Expect(content).To(ContainSubstring(`completed = 2`))
			Expect(
				content,
			).To(ContainSubstring(`export const taskStateZ = z.enum(TaskState)`))
			Expect(content).To(ContainSubstring(`state: taskStateZ`))
		})

		It("Should generate string enums", func() {
			source := `
				@ts output "out"

				DataType enum {
					float32 = "float32"
					float64 = "float64"
					int32 = "int32"
				}

				Telem struct {
					data_type DataType
				}
			`
			table, diag := Analyze(source, "telem", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// String enums generate a const array and z.enum(ARRAY)
			Expect(
				content,
			).To(ContainSubstring(`export const DATA_TYPES = ["float32", "float64", "int32"] as const`))
			Expect(
				content,
			).To(ContainSubstring(`export const dataTypeZ = z.enum(DATA_TYPES)`))
		})

		It(
			"Should not double the trailing S on the values const for an enum ending in S",
			func() {
				source := `
				@ts output "out"

				Units enum {
					volts = "Volts"
					amps  = "Amps"
				}
			`
				resp := MustGenerate(source, "telem", loader, typesPlugin)
				content := MustContentOf(resp, "types.gen.ts")
				Expect(
					content,
				).To(ContainSubstring(`export const UNITS = ["Volts", "Amps"] as const`))
				Expect(
					content,
				).To(ContainSubstring(`export const unitsZ = z.enum(UNITS)`))
				Expect(content).ToNot(ContainSubstring(`UNITSS`))
			},
		)

		It(
			"Should generate an extending enum as the union of its parents",
			func() {
				source := `
				@ts output "out"

				XAxisKey enum {
					x1 = "x1"
					x2 = "x2"
				}

				YAxisKey enum {
					y1 = "y1"
					y2 = "y2"
				}

				AxisKey enum extends XAxisKey, YAxisKey {}
			`
				table, diag := Analyze(source, "lineplot", loader)
				Expect(diag.Ok()).To(BeTrue())

				resp := MustSucceed(
					typesPlugin.Generate(&plugin.Request{Resolutions: table}),
				)
				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`export const AXIS_KEYS = ["x1", "x2", "y1", "y2"] as const`))
				Expect(
					content,
				).To(ContainSubstring(`export const axisKeyZ = z.enum(AXIS_KEYS)`))
			},
		)

		Context("primitive type mappings", func() {
			DescribeTable("should generate correct Zod schema",
				func(oracleType, expectedZodType string) {
					source := `
						@ts output "out"

						Test struct {
							field ` + oracleType + `
						}
					`
					resp := MustGenerate(source, "test", loader, typesPlugin)
					ExpectContent(
						resp,
						"types.gen.ts",
					).ToContain("field: " + expectedZodType)
				},
				Entry("uuid", "uuid", "z.uuid()"),
				Entry("string", "string", "z.string()"),
				Entry("bool", "bool", "z.boolean()"),
				Entry("int8", "int8", "zod.int8"),
				Entry("int16", "int16", "zod.int16"),
				Entry("int32", "int32", "z.int32()"),
				Entry("int64", "int64", "z.int64()"),
				Entry("uint8", "uint8", "zod.uint8"),
				Entry("uint16", "uint16", "zod.uint16"),
				Entry("uint32", "uint32", "z.uint32()"),
				Entry("uint64", "uint64", "z.uint64()"),
				Entry("float32", "float32", "z.number()"),
				Entry("float64", "float64", "z.number()"),
				Entry("record", "record", "record.unknownZ().default(() => ({}))"),
				Entry("bytes", "bytes", "z.instanceof(Uint8Array)"),
			)
		})

		Context("@ts to_number directive", func() {
			It(
				"Should generate schema that accepts strings and converts to number with NaN validation",
				func() {
					source := `
					@ts output "out"

					Key uint32 {
						@ts to_number
					}
				`
					resp := MustGenerate(source, "channel", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`export const keyZ = z.uint32().or(z.string().refine((v) => !isNaN(Number(v))).transform(Number))`)
				},
			)
		})

		Context("@ts to_string directive", func() {
			It(
				"Should generate schema that accepts numbers and converts to string",
				func() {
					source := `
					@ts output "out"

					Name string {
						@ts to_string
					}
				`
					resp := MustGenerate(source, "user", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`export const nameZ = z.union([z.string(), z.number()]).transform(String)`)
				},
			)
		})

		It(
			"Should convert snake_case to camelCase for field names",
			func() {
				source := `
				@ts output "out"

				Range struct {
					created_at int64
					time_range string
					my_long_field_name string
				}
			`
				table, diag := Analyze(source, "ranger", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`createdAt:`))
				Expect(content).To(ContainSubstring(`timeRange:`))
				Expect(content).To(ContainSubstring(`myLongFieldName:`))
			},
		)

		It(
			"Should generate create request struct with optional key and password",
			func() {
				source := `
				@ts output "out"

				New struct {
					key uuid?
					username string
					password string @validate min_length 1
					first_name string?
					last_name string?
				}
			`
				table, diag := Analyze(source, "user", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export const newZ = z.object({`))
				Expect(content).To(ContainSubstring(`key: z.uuid().optional()`))
				Expect(content).To(ContainSubstring(`username: z.string()`))
				Expect(content).To(ContainSubstring(`password: z.string().min(1)`))
				Expect(content).To(ContainSubstring(`firstName: z.string().optional()`))
				Expect(content).To(ContainSubstring(`lastName: z.string().optional()`))
				Expect(
					content,
				).To(ContainSubstring(`export interface New extends z.infer<typeof newZ> {}`))
			},
		)

		It("Should handle optional types (?)", func() {
			source := `
				@ts output "out"

				Device struct {
					key uuid
					name string
					status string?
				}
			`
			table, diag := Analyze(source, "device", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// An optional (?) field uses .optional() in TypeScript
			Expect(content).To(ContainSubstring(`status: z.string().optional()`))
		})

		It(
			"Should handle required arrays with a defaulted z.array",
			func() {
				source := `
				@ts output "out"

				Policy struct {
					key uuid
					objects uuid[]
					actions string[]
				}
			`
				table, diag := Analyze(source, "policy", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Required arrays default a missing array to [] without double-wrapping
				// z.array.
				Expect(
					content,
				).To(ContainSubstring(`objects: z.uuid().array().default(() => [])`))
				Expect(
					content,
				).To(ContainSubstring(`actions: z.string().array().default(() => [])`))
			},
		)

		It(
			"Should handle optional arrays with zod.nullToUndefined",
			func() {
				source := `
				@ts output "out"

				Channel struct {
					key uuid
					operations string[]?
				}
			`
				table, diag := Analyze(source, "channel", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Optional arrays use nullToUndefined with array schema
				Expect(
					content,
				).To(ContainSubstring(`operations: zod.nullToUndefined(z.string().array())`))
			},
		)

		It(
			"Should handle required record fields with a defaulted record.unknownZ",
			func() {
				source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record
				}
			`
				table, diag := Analyze(source, "workspace", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Required record fields default a missing record to {} (the server
				// omits a nil record).
				Expect(
					content,
				).To(ContainSubstring(`layout: record.unknownZ().default(() => ({}))`))
			},
		)

		It(
			"Should handle optional record fields with zod.nullToUndefined",
			func() {
				source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record?
				}
			`
				table, diag := Analyze(source, "workspace", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Optional record fields use zod.nullToUndefined to convert null ->
				// undefined
				Expect(
					content,
				).To(ContainSubstring(`layout: zod.nullToUndefined(record.unknownZ())`))
			},
		)

		It(
			"Should handle optional record fields with zod.nullToUndefined",
			func() {
				source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record?
				}
			`
				table, diag := Analyze(source, "workspace", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Optional record fields also use zod.nullToUndefined
				Expect(
					content,
				).To(ContainSubstring(`layout: zod.nullToUndefined(record.unknownZ())`))
			},
		)

		It(
			"Should wrap required record field with caseconv.preserveCase when @ts preserve_case is specified",
			func() {
				source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record {
						@ts preserve_case
					}
				}
			`
				table, diag := Analyze(source, "workspace", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Required record fields with preserve_case wrap with
				// caseconv.preserveCase
				Expect(
					content,
				).To(ContainSubstring(`layout: caseconv.preserveCase(record.unknownZ().default(() => ({})))`))
				Expect(content).To(ContainSubstring(`import { caseconv`))
			},
		)

		It(
			"Should wrap optional record field with caseconv.preserveCase when @ts preserve_case is specified",
			func() {
				source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record? {
						@ts preserve_case
					}
				}
			`
				table, diag := Analyze(source, "workspace", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Optional record fields with preserve_case wrap with
				// caseconv.preserveCase
				Expect(
					content,
				).To(ContainSubstring(`layout: caseconv.preserveCase(zod.nullToUndefined(record.unknownZ()))`))
				Expect(content).To(ContainSubstring(`import { caseconv`))
			},
		)

		It(
			"Should wrap type parameter field with caseconv.preserveCase when @ts preserve_case is specified",
			func() {
				source := `
				@ts output "out"

				Task struct<Config extends record = record> {
					name string
					config Config {
						@ts preserve_case
					}
				}
			`
				table, diag := Analyze(source, "task", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Type parameter fields with preserve_case wrap with
				// caseconv.preserveCase
				Expect(
					content,
				).To(ContainSubstring(`config: caseconv.preserveCase(config ?? record.unknownZ().default(() => ({})))`))
				Expect(content).To(ContainSubstring(`import { caseconv`))
			},
		)

		It(
			"Should not wrap field with caseconv.preserveCase when @ts no_preserve_case overrides inherited directive",
			func() {
				source := `
				@ts output "out"

				Parent struct {
					layout record {
						@ts preserve_case
					}
				}

				Child struct extends Parent {
					layout record {
						@ts no_preserve_case
					}
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Parent should have preserve_case
				Expect(content).To(ContainSubstring(`export const parentZ`))
				// Child should NOT have preserve_case due to no_preserve_case override
				Expect(content).To(ContainSubstring(`export const childZ`))
				// Child's layout should not be wrapped with caseconv.preserveCase
				Expect(content).NotTo(MatchRegexp(`childZ[^}]*caseconv\.preserveCase`))
			},
		)

		It(
			"Should generate error message for required validation",
			func() {
				source := `
				@ts output "out"

				User struct {
					key uuid
					username string @validate required
					first_name string @validate required
				}
			`
				table, diag := Analyze(source, "user", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`username: z.string().min(1, "username is required")`))
				Expect(
					content,
				).To(ContainSubstring(`firstName: z.string().min(1, "first_name is required")`))
			},
		)

		It("Should use z.input when use_input is specified", func() {
			source := `
				@ts output "out"

				New struct {
					key uuid?
					name string
					data record

					@ts use_input
				}
			`
			table, diag := Analyze(source, "workspace", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(
				content,
			).To(ContainSubstring(`export interface New extends z.input<typeof newZ> {}`))
			Expect(
				content,
			).To(ContainSubstring(`data: record.unknownZ().default(() => ({}))`))
		})

		It(
			"Should use z.record for record fields in child struct with type param",
			func() {
				source := `
				@ts output "out"

				Parent struct<Properties extends record = record> {
					name string
					properties Properties
				}

				Child struct<Properties extends record = record> extends Parent<Properties> {
					key uuid?
					@ts use_input
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Type should use z.input
				Expect(content).To(ContainSubstring(`z.input<`))
			},
		)

		It("Should use z.infer by default without use_input", func() {
			source := `
				@ts output "out"

				Workspace struct {
					key uuid
					name string
				}
			`
			table, diag := Analyze(source, "workspace", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(
				content,
			).To(ContainSubstring(`export interface Workspace extends z.infer<typeof workspaceZ> {}`))
		})

		It(
			"Should generate getter for direct self-referencing struct",
			func() {
				source := `
				@ts output "out"

				Kind enum {
					string = 1
					chan = 2
				}

				Type struct {
					kind Kind
					elem Type?
				}
			`
				table, diag := Analyze(source, "arc", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export type Type = {`))
				Expect(
					content,
				).To(ContainSubstring(`export const typeZ: z.ZodType<Type> = z.object({`))
				Expect(content).To(ContainSubstring(`kind: kindZ`))
				Expect(content).To(ContainSubstring(`get elem() {`))
				Expect(content).To(ContainSubstring(`return typeZ.optional()`))
			},
		)

		It(
			"Should generate getter for array self-referencing struct",
			func() {
				source := `
				@ts output "out"

				Node struct {
					key string
					children Node[]?
				}
			`
				table, diag := Analyze(source, "tree", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export type Node = {`))
				Expect(
					content,
				).To(ContainSubstring(`export const nodeZ: z.ZodType<Node> = z.object({`))
				Expect(content).To(ContainSubstring(`get children() {`))
				Expect(
					content,
				).To(ContainSubstring(`return zod.nullToUndefined(nodeZ.array())`))
			},
		)

		It(
			"Should emit Zod v4 recursive pattern for mutually recursive structs",
			func() {
				source := `
				@ts output "out"

				A struct {
					b B?
				}

				B struct {
					a A?
				}
			`
				resp := MustGenerate(source, "cycle", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`export type A = {`,
						`export type B = {`,
						`export const aZ: z.ZodType<A> = z.object({`,
						`export const bZ: z.ZodType<B> = z.object({`,
					).
					ToNotContain(
						`extends z.infer<typeof aZ>`,
						`extends z.infer<typeof bZ>`,
					)
			},
		)

		It(
			"Should emit Zod v4 recursive pattern for cycles through a distinct wrapper",
			func() {
				source := `
				@ts output "out"

				A struct {
					b BWrap?
				}

				B struct {
					a A?
				}

				BWrap B
			`
				resp := MustGenerate(source, "cycle", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`export type A = {`,
						`export type B = {`,
						`export const aZ: z.ZodType<A> = z.object({`,
						`export const bZ: z.ZodType<B> = z.object({`,
					).
					ToNotContain(
						`extends z.infer<typeof aZ>`,
						`extends z.infer<typeof bZ>`,
					)
			},
		)

		It(
			"Should generate getter for struct with multiple recursive fields",
			func() {
				source := `
				@ts output "out"

				MosaicNode struct {
					key int32
					first MosaicNode?
					last MosaicNode?
				}
			`
				table, diag := Analyze(source, "mosaic", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export type MosaicNode = {`))
				Expect(
					content,
				).To(ContainSubstring(`export const mosaicNodeZ: z.ZodType<MosaicNode> = z.object({`))
				Expect(content).To(ContainSubstring(`get first() {`))
				Expect(content).To(ContainSubstring(`return mosaicNodeZ.optional()`))
				Expect(content).To(ContainSubstring(`get last() {`))
			},
		)

		It(
			"Should keep recursive struct-extends bases extendable",
			func() {
				source := `
				@ts output "out"

				FunctionProperties struct {
					inputs Param[]?
				}

				Type struct extends FunctionProperties {
					name string
					elem Type?
				}

				Param struct {
					name string
					type Type
				}
			`
				table, diag := Analyze(source, "arc", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`export const typeZ = functionPropertiesZ`))
				Expect(content).To(ContainSubstring(`.extend({`))
				Expect(
					content,
				).ToNot(ContainSubstring(`functionPropertiesZ: z.ZodType`))
				Expect(
					content,
				).To(ContainSubstring(`export const paramZ: z.ZodType<Param> = z.object({`))
			},
		)

		It(
			"Should keep recursive union variant payloads extendable",
			func() {
				source := `
				@ts output "out"

				Group struct {
					name string
					children Node[]
				}

				Node union on type {
					group Group
				}
			`
				table, diag := Analyze(source, "mosaic", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).ToNot(ContainSubstring(`groupZ: z.ZodType`))
			},
		)

		It(
			"Should generate getter for generic recursive struct with single param",
			func() {
				source := `
				@ts output "out"

				TreeNode struct<K = string> {
					key K
					children TreeNode<K>[]?
				}
			`
				table, diag := Analyze(source, "tree", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`export const treeNodeZ = <K extends z.ZodType = z.ZodString>(k?: K) =>`))
				Expect(content).To(ContainSubstring(`get children():`))
				// Optional arrays use zod.nullToUndefined with array schema
				Expect(
					content,
				).To(ContainSubstring(`return zod.nullToUndefined(treeNodeZ(k).array())`))
			},
		)

		It(
			"Should generate getter for generic recursive struct with multiple params",
			func() {
				source := `
				@ts output "out"

				MapNode struct<K = string, V = string> {
					key K
					value V
					children MapNode<K, V>[]?
				}
			`
				table, diag := Analyze(source, "tree", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`export interface MapNodeSchemas<K extends z.ZodType = z.ZodType, V extends z.ZodType = z.ZodType>`))
				// Schemas use constraint as default, Z function uses specific default
				Expect(
					content,
				).To(ContainSubstring(`export const mapNodeZ = <K extends z.ZodType = z.ZodString, V extends z.ZodType = z.ZodString>`))
				Expect(
					content,
				).To(ContainSubstring(`}: Partial<MapNodeSchemas<K, V>> = {}) =>`))
				Expect(content).To(ContainSubstring(`get children():`))
				// Optional arrays use zod.nullToUndefined with array schema
				Expect(
					content,
				).To(ContainSubstring(`return zod.nullToUndefined(mapNodeZ({k: k, v: v}).array())`))
			},
		)

		It(
			"Should pass Zod types to a generic struct in a recursive interface",
			func() {
				source := `
				@ts output "out"

				Key uint32

				Box struct<T> {
					value T?
				}

				Node struct {
					key      Box<Key>?
					children Node[]?
				}
			`
				resp := MustGenerate(source, "tree", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("key?: Box<typeof keyZ>").
					ToNotContain("Box<Key>")
			},
		)

		It(
			"Should NOT generate getter for non-recursive struct",
			func() {
				source := `
				@ts output "out"

				Simple struct {
					key uuid
					name string
				}
			`
				table, diag := Analyze(source, "simple", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`export const simpleZ = z.object({`))
				Expect(content).To(ContainSubstring(`key: z.uuid()`))
				Expect(content).To(ContainSubstring(`name: z.string()`))
				Expect(
					content,
				).NotTo(ContainSubstring(`get `))
				// No getters for non-recursive types
			},
		)

		It(
			"Should generate fallback pattern for type param fields with string constraint",
			func() {
				source := `
				@ts output "out"

				Task struct<
					Type extends string = string,
					Config extends record = record
				> {
					name   string
					type   Type
					config Config
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// The 'type' field should use: type ?? z.string() since Type extends
				// string
				Expect(
					content,
				).To(ContainSubstring(`type: type ?? z.string()`), "type field should use type param with fallback")
				// The 'config' field should use fallback pattern since Config extends
				// record
				Expect(
					content,
				).To(ContainSubstring(`config: config ?? record.unknownZ().default(() => ({}))`), "config field should use type param with fallback")
				// The 'name' field should just be z.string() (not a type param)
				Expect(content).To(ContainSubstring(`name: z.string()`))
			},
		)

		It(
			"Should preserve trailing acronyms in generated zod schema names",
			func() {
				source := `
				@ts output "out"

				ClientXY struct {
					clientX float64
					clientY float64
				}

				StickyXY struct {
					x float64
					y float64
				}

				EntityID struct {
					value string
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`export const clientXYZ = z.object({`))
				Expect(
					content,
				).To(ContainSubstring(`export const stickyXYZ = z.object({`))
				Expect(
					content,
				).To(ContainSubstring(`export const entityIDZ = z.object({`))
				Expect(content).To(ContainSubstring(`typeof clientXYZ`))
				Expect(content).To(ContainSubstring(`typeof stickyXYZ`))
				Expect(content).NotTo(ContainSubstring(`clientXyZ`))
				Expect(content).NotTo(ContainSubstring(`stickyXyZ`))
				Expect(content).NotTo(ContainSubstring(`entityIdZ`))
			},
		)

		It(
			"Should emit numeric-constrained generic as function with value-typed generic interface",
			func() {
				source := `
				@ts output "out"

				Bounds struct<T extends numeric = float64> {
					lower T
					upper T
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`import { numeric } from "@synnaxlabs/x"`))
				Expect(
					content,
				).To(ContainSubstring(`export const boundsZ = <T extends numeric.Value = number>(t?: z.ZodType<T>) =>`))
				Expect(content).To(ContainSubstring(`lower: t ?? z.number()`))
				Expect(content).To(ContainSubstring(`upper: t ?? z.number()`))
				Expect(
					content,
				).To(ContainSubstring(`export interface Bounds<T extends numeric.Value = number> {`))
				Expect(content).To(ContainSubstring(`lower: T;`))
				Expect(content).To(ContainSubstring(`upper: T;`))
				Expect(content).NotTo(ContainSubstring(`z.infer<T>`))
				Expect(content).NotTo(ContainSubstring(`<T extends z.ZodType`))
			},
		)

		It(
			"Should generate fallback pattern for type param fields with concrete_types directive",
			func() {
				source := `
				@ts output "out"

				Task struct<
					Type extends string = string,
					Config extends record = record
				> {
					name   string
					type   Type
					config Config

					@ts {
						concrete_types
					}
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Even with concrete_types, fields using type params should have the
				// fallback pattern
				Expect(
					content,
				).To(ContainSubstring(`type: type ?? z.string()`), "type field should use type param with fallback even with concrete_types")
				Expect(
					content,
				).To(ContainSubstring(`config: config ?? record.unknownZ().default(() => ({}))`), "config field should use type param with fallback")
			},
		)

		It(
			"Should preserve type params when extending generic parent with pass-through type args",
			func() {
				source := `
				@ts output "out"

				Task struct<
					Type extends string = string,
					Config extends record = record
				> {
					name   string
					type   Type
					config Config

					@ts {
						concrete_types
					}
				}

				New struct<
					Type extends string = string,
					Config extends record = record
				> extends Task<Type, Config> {
					key string?

					@ts {
						concrete_types
					}
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// The parent Task should still have the fallback pattern
				Expect(
					content,
				).To(ContainSubstring(`type: type ?? z.string()`), "parent Task type field should use type param with fallback")
				// The child New should extend the parent properly
				Expect(
					content,
				).To(ContainSubstring(`newZ`), "should generate newZ schema")
			},
		)

		It(
			"Should generate record.Unknown constraint for record-constrained type param",
			func() {
				source := `
				@ts output "out"

				Task struct<Config extends record = record> {
					name   string
					config Config
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`Config extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>`), "record constraint should generate record.Unknown")
				Expect(
					content,
				).To(ContainSubstring(`config: config ?? record.unknownZ().default(() => ({}))`), "record field should use record fallback")
				Expect(
					content,
				).To(ContainSubstring(`import { record`), "should import record")
			},
		)

		It(
			"Should generate ZodNever default and z.unknown() fallback for optional type param",
			func() {
				source := `
				@ts output "out"

				Status struct<Data?> {
					running bool
					data    Data
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`Data extends z.ZodType = z.ZodNever`), "optional param should have ZodNever default")
				Expect(
					content,
				).To(ContainSubstring(`data: data ?? z.unknown().optional()`), "optional param field should use z.unknown().optional() fallback")
			},
		)

		It(
			"Should handle mixed record-constrained and optional type params",
			func() {
				source := `
				@ts output "out"

				Task struct<
					Config extends record = record,
					StatusData?
				> {
					name       string
					config     Config
					statusData StatusData
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`Config extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>`), "constrained record param should use record.Unknown")
				Expect(
					content,
				).To(ContainSubstring(`StatusData extends z.ZodType = z.ZodNever`), "optional param should have ZodNever default")
				Expect(
					content,
				).To(ContainSubstring(`config: config ?? record.unknownZ().default(() => ({}))`), "constrained record field should use record fallback")
				Expect(
					content,
				).To(ContainSubstring(`statusData: statusData ?? z.unknown().optional()`), "optional param field should use z.unknown().optional() fallback")
			},
		)

		It(
			"Should generate .extend() for basic struct extension",
			func() {
				source := `
				@ts output "out"

				Parent struct {
					name string
					age int32
				}

				Child struct extends Parent {
					email string
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Parent should be a regular z.object
				Expect(
					content,
				).To(ContainSubstring(`export const parentZ = z.object({`))
				Expect(content).To(ContainSubstring(`name: z.string()`))
				Expect(content).To(ContainSubstring(`age: z.int32()`))

				// Child should use .extend()
				Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
				Expect(content).To(ContainSubstring(`.extend({`))
				Expect(content).To(ContainSubstring(`email: z.string()`))
			},
		)

		It("Should generate .omit() for field omissions", func() {
			source := `
				@ts output "out"

				Parent struct {
					name string
					age int32
					status string
				}

				Child struct extends Parent {
					-age
					email string
				}
			`
			table, diag := Analyze(source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Child should use .omit() then .extend()
			Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
			Expect(content).To(ContainSubstring(`.omit({ age: true })`))
			Expect(content).To(ContainSubstring(`.extend({`))
			Expect(content).To(ContainSubstring(`email: z.string()`))
		})

		It(
			"Should generate .omit() for multiple field omissions",
			func() {
				source := `
				@ts output "out"

				Parent struct {
					a string
					b string
					c string
					d string
				}

				Child struct extends Parent {
					-a
					-c
					e string
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
				Expect(content).To(ContainSubstring(`.omit({`))
				Expect(content).To(ContainSubstring(`a: true`))
				Expect(content).To(ContainSubstring(`c: true`))
			},
		)

		It(
			"Should handle field override to make it optional using .partial()",
			func() {
				source := `
				@ts output "out"

				Parent struct {
					name string
					age int32
				}

				Child struct extends Parent {
					name string?
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Child uses .partial() to make the field optional (not .extend())
				Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
				Expect(content).To(ContainSubstring(`.partial({ name: true })`))
				// Should NOT redefine the field in extend
				Expect(content).NotTo(ContainSubstring(`name: z.string().optional()`))
			},
		)

		It(
			"Should not emit unused zod import when array/map/record fields only flip optionality",
			func() {
				source := `
				@ts output "out"

				Item struct {
					id string
				}

				Parent struct {
					items Item[]
					tags map<string, Item>
					data record
				}

				Child struct extends Parent {
					items Item[]?
					tags map<string, Item>?
					data record?
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				resp := MustSucceed(
					typesPlugin.Generate(&plugin.Request{Resolutions: table}),
				)

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(
					`.partial({ items: true, tags: true, data: true })`,
				))
				// The partial form does not reference zod.* — the import would be
				// unused and trip the no-unused-vars lint in downstream packages.
				Expect(content).NotTo(ContainSubstring(`zod }`))
				Expect(content).NotTo(ContainSubstring(`zod,`))
				Expect(content).NotTo(ContainSubstring(`, zod `))
			},
		)

		It(
			"Should handle extension without new fields (only omissions)",
			func() {
				source := `
				@ts output "out"

				Parent struct {
					a string
					b string
					c string
				}

				Child struct extends Parent {
					-b
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
				Expect(content).To(ContainSubstring(`.omit({ b: true })`))
			},
		)

		It(
			"Should generate .merge() chain for multiple extends",
			func() {
				source := `
				@ts output "out"

				A struct {
					a string
				}

				B struct {
					b int32
				}

				C struct extends A, B {
					c bool
				}
			`
				table, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// C should use .extend().shape to combine both parents
				Expect(
					content,
				).To(ContainSubstring(`export const cZ = aZ.extend(bZ.shape)`))
				Expect(content).To(ContainSubstring(`.extend({`))
				Expect(content).To(ContainSubstring(`c: z.boolean()`))
			},
		)

		It("Should handle .omit() with multiple extends", func() {
			source := `
				@ts output "out"

				A struct {
					a string
					shared string
				}

				B struct {
					b int32
				}

				C struct extends A, B {
					-shared
					c bool
				}
			`
			table, diag := Analyze(source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// C should extend parents and then omit
			Expect(content).To(ContainSubstring(`aZ.extend(bZ.shape)`))
			Expect(content).To(ContainSubstring(`.omit({ shared: true })`))
			Expect(content).To(ContainSubstring(`c: z.boolean()`))
		})

		It("Should handle three extends with extend chain", func() {
			source := `
				@ts output "out"

				A struct {
					a string
				}

				B struct {
					b int32
				}

				D struct {
					d bool
				}

				C struct extends A, B, D {
					c float32
				}
			`
			table, diag := Analyze(source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// C should chain extend calls for all three parents
			Expect(content).To(ContainSubstring(`aZ.extend(bZ.shape).extend(dZ.shape)`))
			Expect(content).To(ContainSubstring(`.extend({`))
			Expect(content).To(ContainSubstring(`c: z.number()`))
		})

		It("Should preserve field declaration order", func() {
			source := `
				@ts output "out"

				Record struct {
					zebra string
					apple int32
					mango bool
				}
			`
			table, diag := Analyze(source, "order", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			zebraIdx := strings.Index(content, "zebra:")
			appleIdx := strings.Index(content, "apple:")
			mangoIdx := strings.Index(content, "mango:")
			Expect(zebraIdx).To(BeNumerically("<", appleIdx))
			Expect(appleIdx).To(BeNumerically("<", mangoIdx))
		})

		It(
			"Should generate type alias for generic struct reference",
			func() {
				// Regression test: Status = status.Status<StatusDetails> should call
				// the generic struct's factory function with the type argument, not
				// return z.unknown()
				source := `
				@ts output "out"

				StatusDetails struct {
					message string
					code int32
				}

				Status struct<D> {
					variant string
					details D
				}

				RackStatus = Status<StatusDetails>
			`
				table, diag := Analyze(source, "rack", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// The alias should call the generic struct's factory function with the
				// type arg
				Expect(
					content,
				).To(ContainSubstring(`export const rackStatusZ = statusZ(statusDetailsZ)`))
				Expect(content).NotTo(ContainSubstring(`rackStatusZ = z.unknown()`))
			},
		)

		It(
			"Should render array type aliases as defaulted z.array schemas",
			func() {
				// Type aliases that ARE arrays default a missing array to [] since the
				// type itself is fundamentally an array.
				source := `
				@ts output "out"

				Stage struct {
					key string
					name string
				}

				Stages Stage[]

				Stratum = string[]
			`
				table, diag := Analyze(source, "arc", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Array type aliases default a missing array to [] using the element
				// schema.
				Expect(
					content,
				).To(ContainSubstring(`export const stagesZ = stageZ.array().default(() => [])`))
				Expect(
					content,
				).To(ContainSubstring(`export const stratumZ = z.string().array().default(() => [])`))
				// Should NOT be plain z.array()
				Expect(content).NotTo(ContainSubstring(`stagesZ = z.array(stageZ)`))
				Expect(
					content,
				).NotTo(ContainSubstring(`stratumZ = z.array(z.string())`))
			},
		)

		It(
			"Should not double-wrap arrays when using array helpers",
			func() {
				// Regression test: arrays should not be wrapped twice with z.array()
				// The array helpers (nullishToEmpty, nullToUndefined) expect element
				// schemas
				source := `
				@ts output "out"

				Operation struct {
					type string
					duration int64
				}

				Channel struct {
					key uint32
					operations Operation[]
					optional_ops Operation[]?
				}
			`
				table, diag := Analyze(source, "channel", loader)
				Expect(diag.Ok()).To(BeTrue())

				req := &plugin.Request{
					Resolutions: table,
				}

				resp := MustSucceed(typesPlugin.Generate(req))

				content := string(resp.Files[0].Content)
				// Should use operationZ directly, not z.array(operationZ)
				Expect(
					content,
				).To(ContainSubstring(`operations: operationZ.array().default(() => [])`))
				Expect(
					content,
				).To(ContainSubstring(`optionalOps: zod.nullToUndefined(operationZ.array())`))
				// Make sure we don't have the double-wrapped version
				Expect(content).NotTo(ContainSubstring(`z.array(operationZ)`))
			},
		)

		Context("map types", func() {
			It(
				"Should default a required map field to an empty record so a missing map parses cleanly",
				func() {
					source := `
					@ts output "out"

					Config struct {
						settings map<string, string>
					}
				`
					resp := MustGenerate(source, "config", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(
							`settings: z.record(z.string(), z.string()).default(() => ({}))`,
						)
				},
			)

			It(
				"Should preserve typed value schemas when defaulting a required map",
				func() {
					source := `
					@ts output "out"

					Metrics struct {
						counts map<string, int64>
					}
				`
					resp := MustGenerate(source, "metrics", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`counts: z.record(z.string(), z.int64()).default(() => ({}))`)
				},
			)

			It(
				"Should reference struct value schemas inside a defaulted required map",
				func() {
					source := `
					@ts output "out"

					Entry struct {
						value int32
					}

					Store struct {
						entries map<string, Entry>
					}
				`
					resp := MustGenerate(source, "store", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`entries: z.record(z.string(), entryZ).default(() => ({}))`)
				},
			)

			It(
				"Should use z.partialRecord for a map keyed by an enum",
				func() {
					source := `
					@ts output "out"

					Port enum {
						ain0 = "AIN0"
						ain1 = "AIN1"
					}

					Ports struct {
						channels map<Port, uint32>
						labels   map<Port, string>?
						nested   map<string, map<Port, uint32>>
					}
				`
					resp := MustGenerate(source, "ports", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(
							`channels: z.partialRecord(portZ, z.uint32()).default(() => ({}))`,
							`labels: zod.nullToUndefined(z.partialRecord(portZ, z.string()))`,
							`nested: z.record(z.string(), z.partialRecord(portZ, z.uint32())).default(() => ({}))`,
						).
						ToNotContain(`z.record(portZ`)
				},
			)

			It(
				"Should type an enum-keyed map as partial in a recursive struct",
				func() {
					source := `
					@ts output "out"

					Port enum {
						ain0 = "AIN0"
					}

					Node struct {
						child    Node?
						channels map<Port, uint32>
						nested   map<string, map<Port, uint32>>
					}
				`
					resp := MustGenerate(source, "node", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").ToContain(
						`channels: Partial<Record<Port, number>>;`,
						`nested: Record<string, Partial<Record<Port, number>>>;`,
					)
				},
			)

			It(
				"Should use z.partialRecord in the getter of a self-referencing map",
				func() {
					source := `
					@ts output "out"

					Port enum {
						ain0 = "AIN0"
					}

					Holder struct {
						children map<Port, Holder>
						nested   map<string, map<Port, Holder>>
					}
				`
					resp := MustGenerate(source, "holder", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").ToContain(
						`children: Partial<Record<Port, Holder>>;`,
						`nested: Record<string, Partial<Record<Port, Holder>>>;`,
						`return z.partialRecord(portZ, holderZ).default(() => ({}));`,
						`return z.record(z.string(), z.partialRecord(portZ, holderZ)).default(() => ({}));`,
					)
				},
			)

			It(
				"Should type the getter of a generic enum-keyed map as a partial record",
				func() {
					source := `
					@ts output "out"

					Port enum {
						ain0 = "AIN0"
					}

					Holder struct<K = string> {
						key      K
						children map<Port, Holder<K>>
						nested   map<string, map<Port, Holder<K>>>
					}
				`
					resp := MustGenerate(source, "holder", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").ToContain(
						`get children(): z.ZodDefault<z.ZodRecord<typeof portZ & z.core.$partial, ReturnType<typeof holderZ<K>>>> {`,
						`get nested(): z.ZodDefault<z.ZodRecord<z.ZodString, z.ZodRecord<typeof portZ & z.core.$partial, ReturnType<typeof holderZ<K>>>>> {`,
					)
				},
			)

			It(
				"Should wrap optional map fields with zod.nullToUndefined around z.record",
				func() {
					source := `
					@ts output "out"

					Config struct {
						settings map<string, string>?
					}
				`
					resp := MustGenerate(source, "config", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(
							`settings: zod.nullToUndefined(z.record(z.string(), z.string()))`,
						)
				},
			)

			It(
				"Should compose a defaulted required map inside caseconv.preserveCase",
				func() {
					source := `
					@ts output "out"

					Schematic struct {
						configs map<string, record> {
							@ts preserve_case
						}
					}
				`
					resp := MustGenerate(source, "schematic", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(
							`configs: caseconv.preserveCase(z.record(z.string(), record.unknownZ()).default(() => ({})))`,
						)
				},
			)
		})

		Context("@omit directive", func() {
			It("Should skip types with @ts hand directive", func() {
				source := `
					@ts output "out"

					User struct {
						key uuid
						name string
					}

					InternalState struct {
						cache record
						@ts hand
					}
				`
				resp := MustGenerate(source, "user", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export const userZ`))
				Expect(content).NotTo(ContainSubstring(`internalStateZ`))
				Expect(content).NotTo(ContainSubstring(`InternalState`))
			})

			It("Should skip enums with @ts hand directive", func() {
				source := `
					@ts output "out"

					Status enum {
						active = 1
						inactive = 2
					}

					DebugLevel enum {
						verbose = 0
						trace = 1
						@ts hand
					}
				`
				resp := MustGenerate(source, "status", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export enum Status`))
				Expect(content).NotTo(ContainSubstring(`DebugLevel`))
			})
		})

		Context("same-namespace hand-written reference", func() {
			It("Should import a referenced hand-written struct from its module",
				func() {
					source := `
						@ts output "x/ts/src/color"

						Color struct {
							r uint8
							@ts hand
							@ts include "x/ts/src/color/color"
						}

						Band struct {
							threshold float64
							color Color
						}
					`
					resp := MustGenerate(source, "color", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(
							`import { colorZ } from "@/color/color"`,
							`color: colorZ`,
						).
						ToNotContain("export const colorZ")
				})

			It("Should import a hand-written struct that a struct extends",
				func() {
					source := `
						@ts output "x/ts/src/color"

						Color struct {
							r uint8
							@ts hand
							@ts include "x/ts/src/color/color"
						}

						Named struct extends Color {
							name string
						}
					`
					resp := MustGenerate(source, "color", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(
							`import { colorZ } from "@/color/color"`,
							"export const namedZ = colorZ\n",
						)
				})

			It("Should fail when a referenced hand-written struct declares no module",
				func() {
					source := `
						@ts output "x/ts/src/color"

						Color struct {
							r uint8
							@ts hand
						}

						Band struct {
							color Color
						}
					`
					req := MustGenerateRequest(source, "color", loader)
					Expect(typesPlugin.Generate(req)).Error().To(MatchError(
						"failed to generate x/ts/src/color: types with no generated " +
							"TypeScript declaration need a @ts include module: color.Color",
					))
				})
		})

		Context("documentation", func() {
			It("Should generate JSDoc comments from doc domain", func() {
				source := `
					@ts output "out"

					User struct {
						@doc value "A User represents a user in the system."

						key uuid @key {
							@doc value "The unique identifier for the user."
						}

						name string {
							@doc value "The user's display name."
						}

						age int32
					}
				`
				resp := MustGenerate(source, "user", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(
					content,
				).To(ContainSubstring(`/** User A User represents a user in the system. */`))
				Expect(
					content,
				).To(ContainSubstring(`/** key The unique identifier for the user. */`))
				Expect(
					content,
				).To(ContainSubstring(`/** name The user's display name. */`))
			})
		})

		Context("cross-namespace struct reference", func() {
			BeforeEach(func() {
				loader.Add("schemas/common", `
					@ts output "client/ts/src/common"

					Info struct {
						key uuid
						description string
					}
				`)
			})

			It("Should import cross-namespace struct type", func() {
				source := `
					import "schemas/common"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						info common.Info
					}
				`
				resp := MustGenerate(source, "task", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`import { common } from "@/common"`,
						`common.infoZ`,
					)
			})
		})

		Context("schemas with the same file name", func() {
			It("Should alias a name that two modules export", func() {
				loader.Add("schemas/x/color", `
					@ts output "x/ts/src/color"

					Color struct {
						r uint8

						@ts hand
						@ts include "x/ts/src/color/color"
					}
				`)
				loader.Add("schemas/synnax/color", `
					import "schemas/x/color"

					@ts output "client/ts/src/color"

					Named struct {
						light color.Color
					}
				`)
				loader.Add("schemas/synnax/look", `
					import "schemas/synnax/color"
					import "schemas/x/color"

					@ts output "client/ts/src/look"

					Look struct {
						fill color.Named
						raw  color.Color
					}
				`)
				resp := MustGenerateMulti(loader, typesPlugin)
				ExpectContent(resp, "client/ts/src/look/types.gen.ts").
					ToContain(
						`import { color as xcolor } from "@synnaxlabs/x";`,
						`import { color } from "@/color";`,
						"fill: color.namedZ,",
						"raw: xcolor.colorZ,",
					)
			})

			It("Should alias an enum default that two modules export", func() {
				loader.Add("schemas/x/color", `
					@ts output "x/ts/src/color"

					Kind enum {
						solid    = 0
						gradient = 1
					}
				`)
				loader.Add("schemas/synnax/color", `
					@ts output "client/ts/src/color"

					Named struct {
						name string
					}
				`)
				loader.Add("schemas/synnax/look", `
					import "schemas/synnax/color"
					import "schemas/x/color"

					@ts output "client/ts/src/look"

					Look struct {
						fill color.Named
						kind color.Kind = color.KindSolid
					}
				`)
				resp := MustGenerateMulti(loader, typesPlugin)
				ExpectContent(resp, "client/ts/src/look/types.gen.ts").
					ToContain(
						`import { color as xcolor } from "@synnaxlabs/x";`,
						"kind: xcolor.kindZ.default(xcolor.Kind.solid),",
					)
			})

			It("Should import types from a schema with the same file name", func() {
				loader.Add("schemas/x/color", `
					@ts output "x/ts/src/color"

					Color struct {
						r uint8

						@ts hand
						@ts include "x/ts/src/color/color"
					}

					Stop struct {
						position float32
					}
				`)
				loader.Add("schemas/synnax/color", `
					import "schemas/x/color"

					@ts output "client/ts/src/color"

					Named struct {
						light color.Color
						stop  color.Stop
					}
				`)
				resp := MustGenerateMulti(loader, typesPlugin)
				ExpectContent(resp, "client/ts/src/color/types.gen.ts").
					ToContain(
						`import { color } from "@synnaxlabs/x";`,
						"light: color.colorZ,",
						"stop: color.stopZ,",
					)
			})

			It("Should fail when two modules end in the name", func() {
				loader.Add("schemas/a/color", `
					@ts output "client/ts/src/a/color"

					Color struct {
						r uint8
					}
				`)
				loader.Add("schemas/b/color", `
					@ts output "client/ts/src/b/color"

					Named struct {
						name string
					}
				`)
				req := MustGenerateRequest(`
					import "schemas/a/color"
					import "schemas/b/color"

					@ts output "client/ts/src/look"

					Look struct {
						fill color.Named
						raw  color.Color
					}
				`, "look", loader)
				Expect(typesPlugin.Generate(req)).Error().To(MatchError(
					"failed to generate client/ts/src/look: unresolvable import " +
						"name conflicts in client/ts/src/look: " +
						"map[color:[@/a/color @/b/color]]",
				))
			})
		})

		Context("reference into an ancestor namespace", func() {
			BeforeEach(func() {
				loader.Add("schemas/ranger", `
					@ts output "client/ts/src/ranger"

					Key uuid
				`)
			})

			It(
				"Should import the ancestor's types.gen as a namespace",
				func() {
					source := `
					import "schemas/ranger"

					@ts output "client/ts/src/ranger/kv"

					Pair struct {
						range ranger.Key
						key string
					}
				`
					resp := MustGenerate(source, "kv", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(
							`import * as ranger from "@/ranger/types.gen"`,
							`ranger.keyZ`,
						).
						ToNotContain(`from "@/ranger";`)
				},
			)
		})

		Context("snake_case cross-namespace reference", func() {
			BeforeEach(func() {
				loader.Add("schemas/task_config", `
					@ts output "client/ts/src/task/config"

					BaseStart struct {
						auto_start bool = false
					}
				`)
			})

			It("Should camelize the namespace identifier", func() {
				source := `
					import "schemas/task_config"

					@ts output "client/ts/src/http"

					ReadConfig struct extends task_config.BaseStart {
						endpoint string = ""
					}
				`
				resp := MustGenerate(source, "http", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`import { taskConfig } from`,
						`taskConfig.baseStartZ`,
					).
					ToNotContain("task_config")
			})
		})

		Context("cross-namespace enum reference", func() {
			BeforeEach(func() {
				loader.Add("schemas/status", `
					@ts output "client/ts/src/status"

					StatusCode enum {
						ok = 0
						error = 1
					}
				`)
			})

			It("Should import cross-namespace enum type", func() {
				source := `
					import "schemas/status"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						code status.StatusCode
					}
				`
				resp := MustGenerate(source, "task", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`import { status } from`,
						`status.statusCodeZ`,
					)
			})

			It(
				"Should not generate a local copy of cross-namespace enum",
				func() {
					source := `
					import "schemas/status"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						code status.StatusCode
					}
				`
					resp := MustGenerate(source, "task", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(
							`import { status } from`,
							`status.statusCodeZ`,
						).
						ToNotContain(
							`STATUS_CODES`,
							`statusCodeZ = z.enum`,
							`type StatusCode = z.infer`,
						)
				},
			)
		})

		Context("cross-namespace alias reference", func() {
			BeforeEach(func() {
				loader.Add("schemas/common", `
					@ts output "client/ts/src/common"

					Name = string
				`)
			})

			It("Should import the alias's namespace", func() {
				source := `
					import "schemas/common"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						name common.Name
					}
				`
				resp := MustGenerate(source, "task", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`import { common } from "@/common"`, `common.nameZ`)
			})
		})

		Context("same-package cross-namespace reference", func() {
			BeforeEach(func() {
				loader.Add("schemas/common", `
					@ts output "client/ts/src/common"

					Info struct {
						key uuid
						name string
					}
				`)
			})

			It(
				"Should use internal prefix for same-package imports",
				func() {
					source := `
					import "schemas/common"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						info common.Info
					}
				`
					resp := MustGenerate(source, "task", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`import { common } from "@/common"`)
				},
			)
		})

		Context("typedef alias form", func() {
			It("Should generate type alias for alias typedef", func() {
				source := `
					@ts output "out"

					Key = uint32

					User struct {
						key Key
					}
				`
				resp := MustGenerate(source, "user", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("keyZ")
			})

			It("Should generate typedef with array alias", func() {
				source := `
					@ts output "out"

					Tags = string[]
				`
				resp := MustGenerate(source, "tag", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("tagsZ")
			})
		})

		Context("struct with @ts type override on field", func() {
			It("Should use type override for field type", func() {
				source := `
					@ts output "out"

					Key uint32 {
						@ts type "string"
					}

					User struct {
						key Key
						name string
					}
				`
				resp := MustGenerate(source, "user", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("keyZ")
			})

			It(
				"Should use a non-primitive type override referencing another schema type",
				func() {
					source := `
					@ts output "out"

					Bounded struct {
						start int64
						end   int64
					}

					Span struct {
						start int64
					}

					Range struct {
						extent Span {
							@ts type Bounded
						}
					}
				`
					resp := MustGenerate(source, "test", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain("extent: boundedZ").
						ToNotContain("extent: spanZ").
						ToNotContain("extent: z.unknown()")
				},
			)
		})

		Context("struct field with @ts pick", func() {
			It(
				"Should narrow a struct-typed field to the picked fields",
				func() {
					source := `
					@ts output "out"

					Range struct {
						key uuid {
							@key
						}
						name   string
						parent Range?
					}

					New struct extends Range {
						parent Range? {
							@ts pick key
						}
					}
				`
					resp := MustGenerate(source, "test", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(".omit({ parent: true })").
						ToContain("parent: rangeZ.pick({ key: true }).optional()")
				},
			)
		})

		Context("struct with forward reference", func() {
			It(
				"Should handle struct referencing a later-declared struct",
				func() {
					source := `
					@ts output "out"

					Parent struct {
						key uuid
						child Child
					}

					Child struct {
						name string
					}
				`
					resp := MustGenerate(source, "test", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain("parentZ").
						ToContain("childZ")
				},
			)
		})

		Context("self-referencing struct", func() {
			It("Should handle struct with self reference", func() {
				source := `
					@ts output "out"

					Node struct {
						key uuid
						parent Node?
					}
				`
				resp := MustGenerate(source, "tree", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("nodeZ")
			})
		})

		Context("generic struct with type parameter", func() {
			It(
				"Should generate generic struct with zod function",
				func() {
					source := `
					@ts output "out"

					Response struct<T> {
						data T
						status int32
					}
				`
					resp := MustGenerate(source, "api", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain("responseZ")
				},
			)
		})

		Context("struct with map field", func() {
			It("Should generate record type for map field", func() {
				source := `
					@ts output "out"

					Config struct {
						settings map<string, record>
					}
				`
				resp := MustGenerate(source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("configZ").
					ToContain("settings:")
			})
		})

		Context("struct extends another struct", func() {
			It("Should include parent fields in output", func() {
				source := `
					@ts output "out"

					Base struct {
						key uuid
						name string
					}

					Derived struct extends Base {
						extra int32
					}
				`
				resp := MustGenerate(source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("derivedZ").
					ToContain("extra:")
			})
		})

		DescribeTable("Should instantiate a generic parent with its child's arguments",
			func(child, expected string) {
				source := `
					@ts output "out"

					Box struct<T> {
						item T
					}

					Pair struct<K, V> {
						key   K
						value V
					}

					Labeled struct {
						label string
					}
				` + child
				resp := MustGenerate(source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").ToContain(expected)
			},
			Entry("a concrete argument", `
				Panel struct extends Box<string> {
					count int32
				}
			`, "export const panelZ = boxZ(z.string())\n"),
			Entry("a renamed type parameter", `
				Crate struct<U> extends Box<U> {
					count int32
				}
			`, "=>\n  boxZ(u)\n"),
			Entry("a concrete argument in a generic child", `
				Crate struct<T> extends Box<string> {
					extra T
				}
			`, "=>\n  boxZ(z.string())\n"),
			Entry("a non-generic parent of a generic child", `
				Crate struct<T> extends Labeled {
					extra T
				}
			`, "=>\n  labeledZ\n"),
			Entry("a parent with more parameters than its child", `
				Crate struct<T> extends Pair<string, T> {
					extra T
				}
			`, "=>\n  pairZ({k: z.string(), v: t})\n"),
			Entry("reordered type parameters", `
				Crate struct<K, V> extends Pair<V, K> {
					count int32
				}
			`, "  pairZ({k: v, v: k})\n"),
			Entry("a second parent", `
				Panel struct extends Labeled, Box<string> {
					count int32
				}
			`, "export const panelZ = labeledZ.extend(boxZ(z.string()).shape)\n"),
		)

		Context("multiple files from different namespaces", func() {
			BeforeEach(func() {
				loader.Add("schemas/common", `
					@ts output "client/ts/src/common"

					Info struct {
						name string
						description string
					}
				`)
			})

			It(
				"Should generate cross-namespace struct field reference with import",
				func() {
					source := `
					import "schemas/common"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						info common.Info
					}
				`
					resp := MustGenerate(source, "task", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain("taskZ").
						ToContain("common")
				},
			)
		})

		Context("coalesce_type_params", func() {
			It(
				"Should generate single schema param for type interface",
				func() {
					source := `
					@ts output "out"

					Task struct<
						Type extends string = string,
						Config extends record = record,
						StatusData? = record
					> {
						key    uint64
						name   string
						type   Type
						config Config

						@ts {
							concrete_types
							coalesce_type_params
						}
					}
				`
					resp := MustGenerate(source, "test", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`export interface TaskSchemas<`).
						ToContain(`export const taskZ = <`).
						ToContain(`export interface Task<`).
						ToContain(`S extends TaskSchemas = TaskSchemas`).
						ToContain(`z.infer<S["type"]>`).
						ToContain(`z.infer<S["config"]>`)
				},
			)

			It(
				"Should generate coalesced type for extends struct",
				func() {
					source := `
					@ts output "out"

					Task struct<
						Type extends string = string,
						Config extends record = record
					> {
						key    uint64
						name   string
						type   Type
						config Config

						@ts {
							concrete_types
							coalesce_type_params
						}
					}

					New struct<Type extends string = string, Config extends record = record> extends Task<Type, Config> {
						-key
						key uint64?

						@ts {
							concrete_types
							coalesce_type_params
						}
					}
				`
					resp := MustGenerate(source, "test", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`export type New<`).
						ToContain(`S extends NewSchemas = NewSchemas`).
						ToContain(`Task<S>`)
				},
			)

			It("Should coalesce conditional field references", func() {
				source := `
					@ts output "out"

					Task struct<
						Type extends string = string,
						Data? = record
					> {
						name string
						type Type
						data Data?

						@ts {
							concrete_types
							coalesce_type_params
						}
					}
				`
				resp := MustGenerate(source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`S extends TaskSchemas = TaskSchemas`).
					ToContain(`S["data"]`)
			})

			It(
				"Should coalesce extend fields referencing type params",
				func() {
					source := `
					@ts output "out"

					Wrapper struct<Data?> {
						value string
						data  Data?

						@ts concrete_types
					}

					Task struct<
						Type extends string = string,
						Config extends record = record,
						StatusData? = record
					> {
						key    uint64
						name   string
						type   Type
						config Config
						status Wrapper<StatusData>?

						@ts {
							concrete_types
							coalesce_type_params
						}
					}

					NewWrapper struct<Data?> {
						value string
						data  Data

						@ts concrete_types
					}

					New struct<Type extends string = string, Config extends record = record, StatusData? = record> extends Task<Type, Config, StatusData> {
						status NewWrapper<StatusData>?

						@ts {
							use_input
							concrete_types
							coalesce_type_params
						}
					}
				`
					resp := MustGenerate(source, "test", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`export type New<S extends NewSchemas = NewSchemas>`).
						ToContain(`NewWrapper<S["statusData"]>`)
				},
			)
		})

		Context("different-package cross-namespace reference", func() {
			BeforeEach(func() {
				loader.Add("schemas/telem", `
					@ts output "x/ts/src/telem"

					DataType enum {
						float32 = "float32"
						float64 = "float64"
					}
				`)
			})

			It(
				"Should use package name for different package imports",
				func() {
					source := `
					import "schemas/telem"

					@ts output "client/ts/src/channel"

					Channel struct {
						key uint32
						data_type telem.DataType
					}
				`
					resp := MustGenerate(source, "channel", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`import { telem } from "@synnaxlabs/x"`)
				},
			)
		})

		Context("telem numeric defaults", func() {
			BeforeEach(func() {
				loader.Add("schemas/telem", `
					@ts output "x/ts/src/telem"

					TimeSpan int64 {
						@ts hand
					}

					Rate float64 {
						@ts hand
					}
				`)
			})

			It(
				"Should construct telem instances for numeric defaults",
				func() {
					source := `
					import "schemas/telem"

					@ts output "out"

					Config struct {
						sample_rate telem.Rate = 10
						stream_rate telem.Rate = 2.5
						duration    telem.TimeSpan = 0
						window      telem.TimeSpan = 100
					}
				`
					resp := MustGenerate(source, "config", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(
							`sampleRate: telem.rateZ.default(new Rate(10))`,
							`streamRate: telem.rateZ.default(new Rate(2.5))`,
							`duration: telem.timeSpanZ.default(TimeSpan.ZERO)`,
							`window: telem.timeSpanZ.default(new TimeSpan(100))`,
						)
				},
			)

			It(
				"Should emit a plain number default when a telem field is overridden to number",
				func() {
					source := `
					import "schemas/telem"

					@ts output "out"

					Config struct {
						sample_rate telem.Rate = 10 {
							@ts type "number"
						}
					}
				`
					resp := MustGenerate(source, "config", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`sampleRate: z.number().default(10)`)
				},
			)
		})

		Context("enum variant defaults", func() {
			It(
				"Should generate default for same-namespace enum variant",
				func() {
					source := `
					@ts output "out"

					Mode enum {
						automatic = 0
						manual    = 1
					}

					Config struct {
						mode Mode = ModeAutomatic
					}
				`
					resp := MustGenerate(source, "config", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`mode: modeZ.default(Mode.automatic)`)
				},
			)

			It(
				"Should resolve a bare multi-word snake_case variant name as a default",
				func() {
					source := `
					@ts output "out"

					TerminalConfig enum {
						cfg_default = "Cfg_Default"
						rse         = "RSE"
					}

					Config struct {
						terminal TerminalConfig = cfg_default
					}
				`
					resp := MustGenerate(source, "config", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`terminal: terminalConfigZ.default("Cfg_Default")`)
				},
			)

			It(
				"Should generate default for cross-namespace enum variant",
				func() {
					loader.Add("schemas/control", `
					@ts output "x/ts/src/control"

					Concurrency enum {
						exclusive = 0
						shared    = 1
					}
				`)
					source := `
					import "schemas/control"

					@ts output "client/ts/src/channel"

					Channel struct {
						concurrency control.Concurrency = control.ConcurrencyExclusive
					}
				`
					resp := MustGenerate(source, "channel", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`concurrency: control.concurrencyZ.default(control.Concurrency.exclusive)`)
				},
			)

			It(
				"Should emit a string literal default for a string-valued enum variant",
				func() {
					source := `
					@ts output "out"

					Level enum {
						info    = "info"
						warning = "warning"
					}

					Config struct {
						level Level = LevelInfo
					}
				`
					resp := MustGenerate(source, "config", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`level: levelZ.default("info")`)
				},
			)

			It(
				"Should emit a string literal default for a cross-namespace string-valued enum variant",
				func() {
					loader.Add("schemas/text", `
					@ts output "x/ts/src/text"

					Level enum {
						p     = "p"
						small = "small"
					}
				`)
					source := `
					import "schemas/text"

					@ts output "client/ts/src/lineplot"

					Title struct {
						level text.Level = text.LevelP
					}
				`
					resp := MustGenerate(source, "lineplot", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").
						ToContain(`level: text.levelZ.default("p")`)
				},
			)
		})
	})
})

var _ = Describe("TS Union Generation", func() {
	var (
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		typesPlugin = types.New(types.DefaultOptions())
	})

	It(
		"Should generate a discriminated union with per-variant schemas",
		func() {
			source := `
			@ts output "out"

			LinearParams struct {
				slope float64
				yIntercept float64
			}
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none NoneParams
			}
		`
			resp := MustGenerate(source, "ni", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					`export const linearScaleZ = linearParamsZ.extend({`,
					`type: z.literal("linear"),`,
					`export const noneScaleZ = noneParamsZ.extend({`,
					`type: z.literal("none"),`,
					`export const scaleZ = z.discriminatedUnion("type", [`,
					`linearScaleZ,`,
					`noneScaleZ,`,
					`export type Scale = LinearScale | NoneScale;`,
				)
		},
	)

	It("Should key the schema map by a borrowed enum", func() {
		source := `
			@ts output "out"

			Method enum {
				password = "password"
			}
			Credentials struct { username string }

			Proof union on method Method {
				password Credentials
			}
		`
		resp := MustGenerate(source, "auth", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`export const METHODS = ["password"] as const;`,
				`method: z.literal("password"),`,
				`[K in Method]: z.ZodType<Extract<Proof, { method: K }>>;`,
			).
			ToNotContain(`PROOF_TYPES`, `proofTypeZ`, `export type ProofType`)
	})

	It("Should import a borrowed enum from another namespace", func() {
		loader.Add("schemas/auth", `
			@ts output "client/ts/src/auth"

			Method enum {
				password = "password"
			}
		`)
		source := `
			import "schemas/auth"

			@ts output "client/ts/src/login"

			Credentials struct { username string }

			Proof union on method auth.Method {
				password Credentials
			}
		`
		resp := MustGenerate(source, "login", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`import { auth } from "@/auth"`, `[K in auth.Method]`)
	})

	Describe("Lazy variants", func() {
		unionSource := func(count int) string {
			var b strings.Builder
			b.WriteString(
				"@ts output \"out\"\nParams struct { value float64 }\nShape union on variant {\n",
			)
			for i := range count {
				fmt.Fprintf(&b, "v%d Params\n", i)
			}
			b.WriteString("}\n")
			return b.String()
		}

		It(
			"Should wrap each variant in z.lazy when a union has 16 variants",
			func() {
				resp := MustGenerate(unionSource(16), "ni", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`export const shapeZ = z.discriminatedUnion("variant", [`,
						`z.lazy(() => v0ShapeZ),`,
						`z.lazy(() => v15ShapeZ),`,
						`v0: v0ShapeZ,`,
					)
			},
		)

		It(
			"Should reference variants directly when a union has 15 variants",
			func() {
				resp := MustGenerate(unionSource(15), "ni", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`  v0ShapeZ,`).
					ToNotContain(`z.lazy(`)
			},
		)
	})

	It(
		"Should declare inline variant fields directly on the member schema",
		func() {
			source := `
			@ts output "out"

			TabBase struct { key string }
			Labeled struct { label string }

			Tab union on variant extends TabBase {
				resource {
					resource string
				}
				view extends Labeled {
					type string
				}
				empty {}
			}
		`
			resp := MustGenerate(source, "panel", loader, typesPlugin)
			content := ExpectContent(resp, "types.gen.ts")
			content.ToContain(
				`export const resourceTabZ = tabBaseZ.extend({`,
				`variant: z.literal("resource"),`,
				`resource: z.string(),`,
				`export const viewTabZ = tabBaseZ.extend(labeledZ.shape).extend({`,
				`variant: z.literal("view"),`,
				`type: z.string(),`,
				`export const emptyTabZ = tabBaseZ.extend({`,
				`variant: z.literal("empty"),`,
			)
			content.ToNotContain("ViewTabPayload", "viewTabPayloadZ")
		},
	)

	It(
		"Should render a recursive inline variant field as a lazy getter",
		func() {
			source := `
			@ts output "out"

			Node union on variant {
				leaf {
					name string
				}
				split {
					first Node
					last  Node
				}
			}
		`
			resp := MustGenerate(source, "panel", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").ToContain(
				`get first(): z.ZodType<Node> {`,
				`return nodeZ;`,
				`get last(): z.ZodType<Node> {`,
			)
		},
	)

	It("Should carry inline variants through union composition", func() {
		source := `
			@ts output "out"

			NodeConfig union on variant {
				box {
					width float64
				}
			}

			EdgeConfig union on variant {
				pipe {
					length float64
				}
			}

			ElementConfig union on variant extends NodeConfig, EdgeConfig {}
		`
		resp := MustGenerate(source, "schematic", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(
			`export const boxElementConfigZ = z.object({`,
			`width: z.number(),`,
			`export const pipeElementConfigZ = z.object({`,
			`length: z.number(),`,
			`export type ElementConfig = BoxElementConfig | PipeElementConfig;`,
		)
	})

	It(
		"Should generate a bare object schema for an inline variant with no bases",
		func() {
			source := `
			@ts output "out"

			Tab union on variant {
				empty {}
			}
		`
			resp := MustGenerate(source, "panel", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").ToContain(
				`export const emptyTabZ = z.object({`,
				`variant: z.literal("empty"),`,
			)
		},
	)

	It(
		"Should break the inference cycle when a variant struct references the union recursively",
		func() {
			source := `
			@ts output "out"

			Leaf struct { name string }

			Split struct {
				size float64
				first Node
				last Node
			}

			Node union on variant {
				leaf Leaf
				split Split
			}
		`
			resp := MustGenerate(source, "panel", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					`get first(): z.ZodType<Node> {`,
					`return nodeZ;`,
					`export type Node = LeafNode | SplitNode;`,
				).
				ToNotContain(`get first(): typeof nodeZ {`)
		},
	)

	It(
		"Should camelize multi-word discriminators in the generated schemas",
		func() {
			source := `
			@ts output "out"

			Spec struct { type string }

			Source union on value_type {
				boolean Spec
				number Spec
			}
		`
			resp := MustGenerate(source, "telem", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					`valueType: z.literal("boolean"),`,
					`export const sourceZ = z.discriminatedUnion("valueType", [`,
				).
				ToNotContain(`value_type`)
		},
	)

	It(
		"Should generate a flat union for a union extending other unions",
		func() {
			source := `
			@ts output "out"

			TankConfig struct { width float64 }
			PipeConfig struct { length float64 }

			NodeConfig union on variant {
				tank TankConfig
			}

			EdgeConfig union on variant {
				pipe PipeConfig
			}

			ElementConfig union on variant extends NodeConfig, EdgeConfig {}
		`
			resp := MustGenerate(source, "schematic", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					`export const elementConfigZ = z.discriminatedUnion("variant", [`,
					`tankElementConfigZ,`,
					`pipeElementConfigZ,`,
					`export const ELEMENT_CONFIG_TYPES = ["tank", "pipe"] as const;`,
				)
		},
	)

	It(
		"Should wrap preserve_keys map fields with caseconv.preserveKeys",
		func() {
			source := `
			@ts output "out"

			Config struct { variant string }

			Schematic struct {
				configs map<string, Config> {
					@ts preserve_keys
				}
			}
		`
			resp := MustGenerate(source, "schematic", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					`configs: caseconv.preserveKeys(z.record(z.string(), configZ).default(() => ({}))),`,
				)
		},
	)

	It(
		"Should generate a discriminator enum and per-variant interfaces",
		func() {
			source := `
			@ts output "out"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none NoneParams
			}
		`
			resp := MustGenerate(source, "ni", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					`export const SCALE_TYPES = ["linear", "none"] as const;`,
					`export const scaleTypeZ = z.enum(SCALE_TYPES);`,
					`export type ScaleType = z.infer<typeof scaleTypeZ>;`,
					`export interface LinearScale extends z.infer<typeof linearScaleZ> {}`,
				)
		},
	)

	It(
		"Should generate a schema map keyed by discriminator value",
		func() {
			source := `
			@ts output "out"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none NoneParams
			}
		`
			resp := MustGenerate(source, "ni", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					`export const SCALE_SCHEMAS: {`,
					`  [K in ScaleType]: z.ZodType<Extract<Scale, { type: K }>>;`,
					`} = {`,
					`linear: linearScaleZ,`,
					`none: noneScaleZ,`,
				)
		},
	)

	It(
		"Should omit a base field a variant drops",
		func() {
			source := `
			@ts output "out"

			BaseAIChan struct {
				port int32
				enabled bool
			}
			Range struct {
				minVal float64
				maxVal float64
			}

			AIChannel union on type extends BaseAIChan {
				ai_voltage extends Range {}
				ai_temp_builtin extends Range {
					-port
					-maxVal
					units string
				}
			}
		`
			resp := MustGenerate(source, "ni", loader, typesPlugin)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring(
				"export const aiTempBuiltinChannelZ = baseAIChanZ" +
					".omit({ port: true })" +
					".extend(rangeZ.omit({ maxVal: true }).shape)",
			))
			Expect(content).To(ContainSubstring(
				"export const aiVoltageChannelZ = baseAIChanZ.extend(rangeZ.shape)",
			))
		},
	)

	It(
		"Should compose the union base and payload via extend, not flatten",
		func() {
			source := `
			@ts output "out"

			BaseAIChan struct {
				port int32
				enabled bool
			}
			VoltageFields struct { minVal float64 }

			AIChannel union on type extends BaseAIChan {
				ai_voltage VoltageFields
			}
		`
			resp := MustGenerate(source, "ni", loader, typesPlugin)
			content := MustContentOf(resp, "types.gen.ts")
			// The variant composes the base and payload schemas and adds only the
			// discriminator; the shared fields live on baseAIChanZ / voltageFieldsZ.
			Expect(
				content,
			).To(ContainSubstring("export const aiVoltageChannelZ = baseAIChanZ.extend(voltageFieldsZ.shape).extend({\n  type: z.literal(\"ai_voltage\"),\n});"))
			Expect(content).To(ContainSubstring(`port: z.int32(),`))
			Expect(content).To(ContainSubstring(`minVal: z.number(),`))
		},
	)

	It(
		"Should compose a union base schema imported from another namespace",
		func() {
			loader.Add("schemas/common", `
			@ts output "client/ts/src/common"

			BaseAIChan struct {
				port int32
				enabled bool
			}
		`)
			source := `
			import "schemas/common"

			@ts output "client/ts/src/ni"

			VoltageFields struct { minVal float64 }

			AIChannel union on type extends common.BaseAIChan {
				ai_voltage VoltageFields
			}
		`
			resp := MustGenerate(source, "ni", loader, typesPlugin)
			content := MustContentOf(resp, "ni/types.gen.ts")
			Expect(content).To(ContainSubstring(`import { common } from "@/common"`))
			Expect(content).To(ContainSubstring(
				"export const aiVoltageChannelZ = common.baseAIChanZ" +
					".extend(voltageFieldsZ.shape).extend({\n" +
					"  type: z.literal(\"ai_voltage\"),\n});",
			))
		},
	)

	It(
		"Should resolve a union-typed struct field to the union schema",
		func() {
			source := `
			@ts output "out"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none NoneParams
			}

			Channel struct {
				customScale Scale
			}
		`
			resp := MustGenerate(source, "ni", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(`customScale: scaleZ,`)
		},
	)

	It("Should support a variant field that is itself a union", func() {
		source := `
			@ts output "out"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none NoneParams
			}

			VoltageFields struct { customScale Scale }

			AIChannel union on type {
				ai_voltage VoltageFields
			}
		`
		resp := MustGenerate(source, "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`export const aiVoltageChannelZ = voltageFieldsZ.extend({`,
				`customScale: scaleZ,`,
			)
	})

	It("Should render the union doc comment", func() {
		source := `
			@ts output "out"

			LinearParams struct { slope float64 }

			Scale union on type {
				linear LinearParams

				@doc value "determines how raw values are transformed."
			}
		`
		resp := MustGenerate(source, "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`Scale determines how raw values are transformed.`)
	})
})

var _ = Describe("TS Union Field & Variant Coverage", func() {
	var (
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		typesPlugin = types.New(types.DefaultOptions())
	})

	scaleSource := func(channelFields string) string {
		return `
			@ts output "out"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams {
					@doc value "a linear scale."
				}
				none NoneParams

				@doc value "determines how raw values are transformed."
			}

			Channel struct {
` + channelFields + `
			}
		`
	}

	It(
		"Should render the union doc directly above the discriminatedUnion, not the first variant",
		func() {
			resp := MustGenerate(
				scaleSource("\t\t\t\tcustomScale Scale"),
				"ni",
				loader,
				typesPlugin,
			)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(
				content,
			).To(ContainSubstring("/** Scale determines how raw values are transformed. */\nexport const scaleZ = z.discriminatedUnion("))
			Expect(
				content,
			).ToNot(ContainSubstring("/** Scale determines how raw values are transformed. */\nexport const linearScaleZ"))
		},
	)

	It(
		"Should render a per-variant doc comment directly above the variant schema",
		func() {
			resp := MustGenerate(
				scaleSource("\t\t\t\tcustomScale Scale"),
				"ni",
				loader,
				typesPlugin,
			)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					"/** LinearScale a linear scale. */\nexport const linearScaleZ = linearParamsZ.extend({",
					`export interface LinearScale extends z.infer<typeof linearScaleZ> {}`,
				)
		},
	)

	It(
		"Should resolve an optional union-typed field as null-tolerant",
		func() {
			resp := MustGenerate(
				scaleSource("\t\t\t\tcustomScale Scale?"),
				"ni",
				loader,
				typesPlugin,
			)
			ExpectContent(resp, "types.gen.ts").ToContain(
				`customScale: zod.nullToUndefined(scaleZ),`,
			)
		},
	)

	It("Should resolve an array-of-union field", func() {
		resp := MustGenerate(
			scaleSource("\t\t\t\tscales Scale[]"),
			"ni",
			loader,
			typesPlugin,
		)
		ExpectContent(
			resp,
			"types.gen.ts",
		).ToContain(`scales: scaleZ.array().default(() => []),`)
	})

	It(
		"Should import and qualify a cross-namespace union reference",
		func() {
			loader.Add("schemas/scales", `
			@ts output "client/ts/src/scales"

			LinearParams struct { slope float64 }
			NoneParams struct {}

			Scale union on type {
				linear LinearParams
				none NoneParams
			}
		`)
			source := `
			import "schemas/scales"

			@ts output "client/ts/src/task"

			Channel struct {
				customScale scales.Scale
			}
		`
			resp := MustGenerate(source, "task", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").ToContain(`scales.scaleZ`)
		},
	)
})

var _ = ShouldNotLeakGoroutinesPerSpec()
