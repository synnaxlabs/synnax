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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin TS Types Suite")
}

var _ = Describe("TSFormatter", func() {
	f := types.TSFormatter

	Describe("FormatQualified", func() {
		It("Should format qualified names with dot separator", func() {
			Expect(f.FormatQualified("pkg", "Type")).To(Equal("pkg.Type"))
		})

		It("Should return type name when qualifier is empty", func() {
			Expect(f.FormatQualified("", "Type")).To(Equal("Type"))
		})
	})

	Describe("FormatGeneric", func() {
		It("Should format generic types with angle brackets", func() {
			Expect(f.FormatGeneric("Container", []string{"T", "U"})).To(Equal("Container<T, U>"))
		})

		It("Should return base name when no type args", func() {
			Expect(f.FormatGeneric("Container", nil)).To(Equal("Container"))
		})
	})

	Describe("FormatArray", func() {
		It("Should format as TypeScript array syntax", func() {
			Expect(f.FormatArray("string")).To(Equal("string[]"))
		})
	})

	Describe("FormatMap", func() {
		It("Should format as Record type", func() {
			Expect(f.FormatMap("string", "number")).To(Equal("Record<string, number>"))
		})
	})

	Describe("FallbackType", func() {
		It("Should return unknown", func() {
			Expect(f.FallbackType()).To(Equal("unknown"))
		})
	})
})

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

		It("Should pass check", func() {
			Expect(typesPlugin.Check(&plugin.Request{})).To(Succeed())
		})
	})

	Describe("Generate", func() {
		Context("basic struct generation", func() {
			It("Should generate schema for simple struct", func(ctx SpecContext) {
				source := `
					@ts output "out"

					User struct {
						key uuid
						name string
						age int32
						active bool
					}
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
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

		It("Should generate a typeless override identically to a full restatement", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				return string(resp.Files[0].Content)
			}
			// A typeless override desugars to the equivalent full restatement, so
			// the generated schema is byte-identical (here, the parent's
			// .partial()/.extend() merge chain rather than a flat object).
			Expect(gen("count = 10")).To(Equal(gen("count int32 = 10")))
			Expect(gen("tag?")).To(Equal(gen("tag string?")))
		})

		It("Should flatten a struct that removes an inherited domain", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Parent struct {
					name string @validate { min_length 1 }
				}

				Child struct extends Parent {
					name -@validate
				}
			`
			resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(`export const childZ = z.object(`, `name: z.string()`)
		})

		It("Should handle optional and array types", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Range struct {
					key uuid
					labels uuid[]
					parent uuid?
					tags string[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Required arrays default a missing array to [] (the server omits a nil
			// array) while still validating a present array.
			Expect(content).To(ContainSubstring(`labels: z.uuid().array().default(() => [])`))
			Expect(content).To(ContainSubstring(`parent: z.uuid().optional()`))
			// Optional arrays use nullToUndefined with array schema
			Expect(content).To(ContainSubstring(`tags: zod.nullToUndefined(z.string().array())`))
		})

		It("Should apply validation rules", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`name: z.string().min(1).max(255)`))
			Expect(content).To(ContainSubstring(`email: z.string()`))
			Expect(content).To(ContainSubstring(`age: z.int32().min(0).max(150)`))
		})

		It("Should classify a distinct numeric type by its primitive base", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Key uint32

				Device struct {
					rack Key @validate { min 1 }
				}
			`
			resp := MustGenerate(ctx, source, "device", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").ToContain("rack: keyZ.min(1)")
		})

		It("Should treat required on a numeric type as non-zero", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Key uint32

				Device struct {
					rack Key @validate { required }
				}
			`
			resp := MustGenerate(ctx, source, "device", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").ToContain(`rack: keyZ.refine((v) => v !== 0, "rack is required")`)
		})

		It("Should generate enums", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
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
			Expect(content).To(ContainSubstring(`export const taskStateZ = z.enum(TaskState)`))
			Expect(content).To(ContainSubstring(`state: taskStateZ`))
		})

		It("Should generate string enums", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "telem", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// String enums generate a const array and z.enum(ARRAY)
			Expect(content).To(ContainSubstring(`export const DATA_TYPES = ["float32", "float64", "int32"] as const`))
			Expect(content).To(ContainSubstring(`export const dataTypeZ = z.enum(DATA_TYPES)`))
		})

		It("Should not double the trailing S on the values const for an enum ending in S", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Units enum {
					volts = "Volts"
					amps  = "Amps"
				}
			`
			resp := MustGenerate(ctx, source, "telem", loader, typesPlugin)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring(`export const UNITS = ["Volts", "Amps"] as const`))
			Expect(content).To(ContainSubstring(`export const unitsZ = z.enum(UNITS)`))
			Expect(content).ToNot(ContainSubstring(`UNITSS`))
		})

		It("Should generate an extending enum as the union of its parents", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "lineplot", loader)
			Expect(diag.Ok()).To(BeTrue())

			resp := MustSucceed(typesPlugin.Generate(&plugin.Request{Resolutions: table}))
			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const AXIS_KEYS = ["x1", "x2", "y1", "y2"] as const`))
			Expect(content).To(ContainSubstring(`export const axisKeyZ = z.enum(AXIS_KEYS)`))
		})

		Context("primitive type mappings", func() {
			DescribeTable("should generate correct Zod schema",
				func(ctx SpecContext, oracleType, expectedZodType string) {
					source := `
						@ts output "out"

						Test struct {
							field ` + oracleType + `
						}
					`
					resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
					ExpectContent(resp, "types.gen.ts").ToContain("field: " + expectedZodType)
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
			It("Should generate schema that accepts strings and converts to number with NaN validation", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Key uint32 {
						@ts to_number
					}
				`
				resp := MustGenerate(ctx, source, "channel", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`export const keyZ = z.uint32().or(z.string().refine((v) => !isNaN(Number(v))).transform(Number))`)
			})
		})

		Context("@ts to_string directive", func() {
			It("Should generate schema that accepts numbers and converts to string", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Name string {
						@ts to_string
					}
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`export const nameZ = z.string().or(z.number().transform(String))`)
			})
		})

		It("Should convert snake_case to camelCase for field names", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Range struct {
					created_at timestamp
					time_range string
					my_long_field_name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`createdAt:`))
			Expect(content).To(ContainSubstring(`timeRange:`))
			Expect(content).To(ContainSubstring(`myLongFieldName:`))
		})

		It("Should generate create request struct with optional key and password", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
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
			Expect(content).To(ContainSubstring(`export interface New extends z.infer<typeof newZ> {}`))
		})

		It("Should handle optional types (?)", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Device struct {
					key uuid
					name string
					status string?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "device", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// An optional (?) field uses .optional() in TypeScript
			Expect(content).To(ContainSubstring(`status: z.string().optional()`))
		})

		It("Should handle required arrays with a defaulted z.array", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Policy struct {
					key uuid
					objects uuid[]
					actions string[]
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "policy", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Required arrays default a missing array to [] without double-wrapping z.array.
			Expect(content).To(ContainSubstring(`objects: z.uuid().array().default(() => [])`))
			Expect(content).To(ContainSubstring(`actions: z.string().array().default(() => [])`))
		})

		It("Should handle optional arrays with zod.nullToUndefined", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Channel struct {
					key uuid
					operations string[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "channel", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Optional arrays use nullToUndefined with array schema
			Expect(content).To(ContainSubstring(`operations: zod.nullToUndefined(z.string().array())`))
		})

		It("Should handle required record fields with a defaulted record.unknownZ", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Required record fields default a missing record to {} (the server omits
			// a nil record).
			Expect(content).To(ContainSubstring(`layout: record.unknownZ().default(() => ({}))`))
		})

		It("Should handle optional record fields with zod.nullToUndefined", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Optional record fields use zod.nullToUndefined to convert null -> undefined
			Expect(content).To(ContainSubstring(`layout: zod.nullToUndefined(record.unknownZ())`))
		})

		It("Should handle optional record fields with zod.nullToUndefined", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Optional record fields also use zod.nullToUndefined
			Expect(content).To(ContainSubstring(`layout: zod.nullToUndefined(record.unknownZ())`))
		})

		It("Should wrap required record field with caseconv.preserveCase when @ts preserve_case is specified", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record {
						@ts preserve_case
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Required record fields with preserve_case wrap with caseconv.preserveCase
			Expect(content).To(ContainSubstring(`layout: caseconv.preserveCase(record.unknownZ().default(() => ({})))`))
			Expect(content).To(ContainSubstring(`import { caseconv`))
		})

		It("Should wrap optional record field with caseconv.preserveCase when @ts preserve_case is specified", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Workspace struct {
					key uuid
					layout record? {
						@ts preserve_case
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Optional record fields with preserve_case wrap with caseconv.preserveCase
			Expect(content).To(ContainSubstring(`layout: caseconv.preserveCase(zod.nullToUndefined(record.unknownZ()))`))
			Expect(content).To(ContainSubstring(`import { caseconv`))
		})

		It("Should wrap type parameter field with caseconv.preserveCase when @ts preserve_case is specified", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Task struct<Config extends record = record> {
					name string
					config Config {
						@ts preserve_case
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Type parameter fields with preserve_case wrap with caseconv.preserveCase
			Expect(content).To(ContainSubstring(`config: caseconv.preserveCase(config ?? record.unknownZ().default(() => ({})))`))
			Expect(content).To(ContainSubstring(`import { caseconv`))
		})

		It("Should not wrap field with caseconv.preserveCase when @ts no_preserve_case overrides inherited directive", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
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
		})

		It("Should generate error message for required validation", func(ctx SpecContext) {
			source := `
				@ts output "out"

				User struct {
					key uuid
					username string @validate required
					first_name string @validate required
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`username: z.string().min(1, "username is required")`))
			Expect(content).To(ContainSubstring(`firstName: z.string().min(1, "first_name is required")`))
		})

		It("Should use z.input when use_input is specified", func(ctx SpecContext) {
			source := `
				@ts output "out"

				New struct {
					key uuid?
					name string
					data record

					@ts use_input
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface New extends z.input<typeof newZ> {}`))
			Expect(content).To(ContainSubstring(`data: record.unknownZ().default(() => ({}))`))
		})

		It("Should use z.record for record fields in child struct with type param", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Type should use z.input
			Expect(content).To(ContainSubstring(`z.input<`))
		})

		It("Should use z.infer by default without use_input", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Workspace struct {
					key uuid
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface Workspace extends z.infer<typeof workspaceZ> {}`))
		})

		It("Should generate getter for direct self-referencing struct", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "arc", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface Type {`))
			Expect(content).To(ContainSubstring(`export const typeZ: z.ZodType<Type> = z.object({`))
			Expect(content).To(ContainSubstring(`kind: kindZ`))
			Expect(content).To(ContainSubstring(`get elem() {`))
			Expect(content).To(ContainSubstring(`return typeZ.optional()`))
		})

		It("Should generate getter for array self-referencing struct", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Node struct {
					key string
					children Node[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "tree", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface Node {`))
			Expect(content).To(ContainSubstring(`export const nodeZ: z.ZodType<Node> = z.object({`))
			Expect(content).To(ContainSubstring(`get children() {`))
			Expect(content).To(ContainSubstring(`return zod.nullToUndefined(nodeZ.array())`))
		})

		It("Should emit Zod v4 recursive pattern for mutually recursive structs", func(ctx SpecContext) {
			source := `
				@ts output "out"

				A struct {
					b B?
				}

				B struct {
					a A?
				}
			`
			resp := MustGenerate(ctx, source, "cycle", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					`export interface A {`,
					`export interface B {`,
					`export const aZ: z.ZodType<A> = z.object({`,
					`export const bZ: z.ZodType<B> = z.object({`,
				).
				ToNotContain(
					`extends z.infer<typeof aZ>`,
					`extends z.infer<typeof bZ>`,
				)
		})

		It("Should emit Zod v4 recursive pattern for cycles through a distinct wrapper", func(ctx SpecContext) {
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
			resp := MustGenerate(ctx, source, "cycle", loader, typesPlugin)
			ExpectContent(resp, "types.gen.ts").
				ToContain(
					`export interface A {`,
					`export interface B {`,
					`export const aZ: z.ZodType<A> = z.object({`,
					`export const bZ: z.ZodType<B> = z.object({`,
				).
				ToNotContain(
					`extends z.infer<typeof aZ>`,
					`extends z.infer<typeof bZ>`,
				)
		})

		It("Should generate getter for struct with multiple recursive fields", func(ctx SpecContext) {
			source := `
				@ts output "out"

				MosaicNode struct {
					key int32
					first MosaicNode?
					last MosaicNode?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "mosaic", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface MosaicNode {`))
			Expect(content).To(ContainSubstring(`export const mosaicNodeZ: z.ZodType<MosaicNode> = z.object({`))
			Expect(content).To(ContainSubstring(`get first() {`))
			Expect(content).To(ContainSubstring(`return mosaicNodeZ.optional()`))
			Expect(content).To(ContainSubstring(`get last() {`))
		})

		It("Should keep recursive struct-extends bases extendable", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "arc", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const typeZ = functionPropertiesZ`))
			Expect(content).To(ContainSubstring(`.extend({`))
			Expect(content).ToNot(ContainSubstring(`functionPropertiesZ: z.ZodType`))
			Expect(content).To(ContainSubstring(`export const paramZ: z.ZodType<Param> = z.object({`))
		})

		It("Should keep recursive union variant payloads extendable", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "mosaic", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).ToNot(ContainSubstring(`groupZ: z.ZodType`))
		})

		It("Should generate getter for generic recursive struct with single param", func(ctx SpecContext) {
			source := `
				@ts output "out"

				TreeNode struct<K extends schema = string> {
					key K
					children TreeNode<K>[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "tree", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const treeNodeZ = <K extends z.ZodType = z.ZodString>(k?: K) =>`))
			Expect(content).To(ContainSubstring(`get children():`))
			// Optional arrays use zod.nullToUndefined with array schema
			Expect(content).To(ContainSubstring(`return zod.nullToUndefined(treeNodeZ(k).array())`))
		})

		It("Should generate getter for generic recursive struct with multiple params", func(ctx SpecContext) {
			source := `
				@ts output "out"

				MapNode struct<K extends schema = string, V extends schema = string> {
					key K
					value V
					children MapNode<K, V>[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "tree", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface MapNodeSchemas<K extends z.ZodType = z.ZodType, V extends z.ZodType = z.ZodType>`))
			// Schemas use constraint as default, Z function uses specific default
			Expect(content).To(ContainSubstring(`export const mapNodeZ = <K extends z.ZodType = z.ZodString, V extends z.ZodType = z.ZodString>`))
			Expect(content).To(ContainSubstring(`}: Partial<MapNodeSchemas<K, V>> = {}) =>`))
			Expect(content).To(ContainSubstring(`get children():`))
			// Optional arrays use zod.nullToUndefined with array schema
			Expect(content).To(ContainSubstring(`return zod.nullToUndefined(mapNodeZ({k: k, v: v}).array())`))
		})

		It("Should NOT generate getter for non-recursive struct", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Simple struct {
					key uuid
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "simple", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const simpleZ = z.object({`))
			Expect(content).To(ContainSubstring(`key: z.uuid()`))
			Expect(content).To(ContainSubstring(`name: z.string()`))
			Expect(content).NotTo(ContainSubstring(`get `)) // No getters for non-recursive types
		})

		It("Should generate fallback pattern for type param fields with string constraint", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// The 'type' field should use: type ?? z.string() since Type extends string
			Expect(content).To(ContainSubstring(`type: type ?? z.string()`), "type field should use type param with fallback")
			// The 'config' field should use fallback pattern since Config extends record
			Expect(content).To(ContainSubstring(`config: config ?? record.unknownZ().default(() => ({}))`), "config field should use type param with fallback")
			// The 'name' field should just be z.string() (not a type param)
			Expect(content).To(ContainSubstring(`name: z.string()`))
		})

		It("Should preserve trailing acronyms in generated zod schema names", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const clientXYZ = z.object({`))
			Expect(content).To(ContainSubstring(`export const stickyXYZ = z.object({`))
			Expect(content).To(ContainSubstring(`export const entityIDZ = z.object({`))
			Expect(content).To(ContainSubstring(`typeof clientXYZ`))
			Expect(content).To(ContainSubstring(`typeof stickyXYZ`))
			Expect(content).NotTo(ContainSubstring(`clientXyZ`))
			Expect(content).NotTo(ContainSubstring(`stickyXyZ`))
			Expect(content).NotTo(ContainSubstring(`entityIdZ`))
		})

		It("Should emit numeric-constrained generic as function with value-typed generic interface", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Bounds struct<T extends numeric = float64> {
					lower T
					upper T
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`import { numeric } from "@synnaxlabs/x"`))
			Expect(content).To(ContainSubstring(`export const boundsZ = <T extends numeric.Value = number>(t?: z.ZodType<T>) =>`))
			Expect(content).To(ContainSubstring(`lower: t ?? z.number()`))
			Expect(content).To(ContainSubstring(`upper: t ?? z.number()`))
			Expect(content).To(ContainSubstring(`export interface Bounds<T extends numeric.Value = number> {`))
			Expect(content).To(ContainSubstring(`lower: T;`))
			Expect(content).To(ContainSubstring(`upper: T;`))
			Expect(content).NotTo(ContainSubstring(`z.infer<T>`))
			Expect(content).NotTo(ContainSubstring(`<T extends z.ZodType`))
		})

		It("Should generate fallback pattern for type param fields with concrete_types directive", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Even with concrete_types, fields using type params should have the fallback pattern
			Expect(content).To(ContainSubstring(`type: type ?? z.string()`), "type field should use type param with fallback even with concrete_types")
			Expect(content).To(ContainSubstring(`config: config ?? record.unknownZ().default(() => ({}))`), "config field should use type param with fallback")
		})

		It("Should preserve type params when extending generic parent with pass-through type args", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// The parent Task should still have the fallback pattern
			Expect(content).To(ContainSubstring(`type: type ?? z.string()`), "parent Task type field should use type param with fallback")
			// The child New should extend the parent properly
			Expect(content).To(ContainSubstring(`newZ`), "should generate newZ schema")
		})

		It("Should generate record.Unknown constraint for record-constrained type param", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Task struct<Config extends record = record> {
					name   string
					config Config
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`Config extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>`), "record constraint should generate record.Unknown")
			Expect(content).To(ContainSubstring(`config: config ?? record.unknownZ().default(() => ({}))`), "record field should use record fallback")
			Expect(content).To(ContainSubstring(`import { record`), "should import record")
		})

		It("Should generate ZodNever default and z.unknown() fallback for optional type param", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Status struct<Data?> {
					running bool
					data    Data
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`Data extends z.ZodType = z.ZodNever`), "optional param should have ZodNever default")
			Expect(content).To(ContainSubstring(`data: data ?? z.unknown().optional()`), "optional param field should use z.unknown().optional() fallback")
		})

		It("Should handle mixed record-constrained and optional type params", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`Config extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>`), "constrained record param should use record.Unknown")
			Expect(content).To(ContainSubstring(`StatusData extends z.ZodType = z.ZodNever`), "optional param should have ZodNever default")
			Expect(content).To(ContainSubstring(`config: config ?? record.unknownZ().default(() => ({}))`), "constrained record field should use record fallback")
			Expect(content).To(ContainSubstring(`statusData: statusData ?? z.unknown().optional()`), "optional param field should use z.unknown().optional() fallback")
		})

		It("Should generate .extend() for basic struct extension", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Parent should be a regular z.object
			Expect(content).To(ContainSubstring(`export const parentZ = z.object({`))
			Expect(content).To(ContainSubstring(`name: z.string()`))
			Expect(content).To(ContainSubstring(`age: z.int32()`))

			// Child should use .extend()
			Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
			Expect(content).To(ContainSubstring(`.extend({`))
			Expect(content).To(ContainSubstring(`email: z.string()`))
		})

		It("Should generate .omit() for field omissions", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
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

		It("Should generate .omit() for multiple field omissions", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
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
		})

		It("Should handle field override to make it optional using .partial()", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
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
		})

		It("Should not emit unused zod import when array/map/record fields only flip optionality", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			resp := MustSucceed(typesPlugin.Generate(&plugin.Request{Resolutions: table}))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(
				`.partial({ items: true, tags: true, data: true })`,
			))
			// The partial form does not reference zod.* — the import would be
			// unused and trip the no-unused-vars lint in downstream packages.
			Expect(content).NotTo(ContainSubstring(`zod }`))
			Expect(content).NotTo(ContainSubstring(`zod,`))
			Expect(content).NotTo(ContainSubstring(`, zod `))
		})

		It("Should handle extension without new fields (only omissions)", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
			Expect(content).To(ContainSubstring(`.omit({ b: true })`))
		})

		It("Should generate .merge() chain for multiple extends", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// C should use .extend().shape to combine both parents
			Expect(content).To(ContainSubstring(`export const cZ = aZ.extend(bZ.shape)`))
			Expect(content).To(ContainSubstring(`.extend({`))
			Expect(content).To(ContainSubstring(`c: z.boolean()`))
		})

		It("Should handle .omit() with multiple extends", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
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

		It("Should handle three extends with extend chain", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
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

		It("Should preserve field declaration order", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Record struct {
					zebra string
					apple int32
					mango bool
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "order", loader)
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

		It("Should generate type alias for generic struct reference", func(ctx SpecContext) {
			// Regression test: Status = status.Status<StatusDetails> should call the
			// generic struct's factory function with the type argument, not return z.unknown()
			source := `
				@ts output "out"

				StatusDetails struct {
					message string
					code int32
				}

				Status struct<D extends schema> {
					variant string
					details D
				}

				RackStatus = Status<StatusDetails>
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "rack", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// The alias should call the generic struct's factory function with the type arg
			Expect(content).To(ContainSubstring(`export const rackStatusZ = statusZ(statusDetailsZ)`))
			Expect(content).NotTo(ContainSubstring(`rackStatusZ = z.unknown()`))
		})

		It("Should render array type aliases as defaulted z.array schemas", func(ctx SpecContext) {
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "arc", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Array type aliases default a missing array to [] using the element schema.
			Expect(content).To(ContainSubstring(`export const stagesZ = stageZ.array().default(() => [])`))
			Expect(content).To(ContainSubstring(`export const stratumZ = z.string().array().default(() => [])`))
			// Should NOT be plain z.array()
			Expect(content).NotTo(ContainSubstring(`stagesZ = z.array(stageZ)`))
			Expect(content).NotTo(ContainSubstring(`stratumZ = z.array(z.string())`))
		})

		It("Should not double-wrap arrays when using array helpers", func(ctx SpecContext) {
			// Regression test: arrays should not be wrapped twice with z.array()
			// The array helpers (nullishToEmpty, nullToUndefined) expect element schemas
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "channel", loader)
			Expect(diag.Ok()).To(BeTrue())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp := MustSucceed(typesPlugin.Generate(req))

			content := string(resp.Files[0].Content)
			// Should use operationZ directly, not z.array(operationZ)
			Expect(content).To(ContainSubstring(`operations: operationZ.array().default(() => [])`))
			Expect(content).To(ContainSubstring(`optionalOps: zod.nullToUndefined(operationZ.array())`))
			// Make sure we don't have the double-wrapped version
			Expect(content).NotTo(ContainSubstring(`z.array(operationZ)`))
		})

		Context("map types", func() {
			It("Should default a required map field to an empty record so a missing map parses cleanly", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Config struct {
						settings map<string, string>
					}
				`
				resp := MustGenerate(ctx, source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`settings: z.record(z.string(), z.string()).default(() => ({}))`,
					)
			})

			It("Should preserve typed value schemas when defaulting a required map", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Metrics struct {
						counts map<string, int64>
					}
				`
				resp := MustGenerate(ctx, source, "metrics", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`counts: z.record(z.string(), z.int64()).default(() => ({}))`)
			})

			It("Should reference struct value schemas inside a defaulted required map", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Entry struct {
						value int32
					}

					Store struct {
						entries map<string, Entry>
					}
				`
				resp := MustGenerate(ctx, source, "store", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`entries: z.record(z.string(), entryZ).default(() => ({}))`)
			})

			It("Should wrap optional map fields with zod.nullToUndefined around z.record", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Config struct {
						settings map<string, string>?
					}
				`
				resp := MustGenerate(ctx, source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`settings: zod.nullToUndefined(z.record(z.string(), z.string()))`,
					)
			})

			It("Should compose a defaulted required map inside caseconv.preserveCase", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Schematic struct {
						configs map<string, record> {
							@ts preserve_case
						}
					}
				`
				resp := MustGenerate(ctx, source, "schematic", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`configs: caseconv.preserveCase(z.record(z.string(), record.unknownZ()).default(() => ({})))`,
					)
			})
		})

		Context("@omit directive", func() {
			It("Should skip types with @ts omit directive", func(ctx SpecContext) {
				source := `
					@ts output "out"

					User struct {
						key uuid
						name string
					}

					InternalState struct {
						cache record
						@ts omit
					}
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export const userZ`))
				Expect(content).NotTo(ContainSubstring(`internalStateZ`))
				Expect(content).NotTo(ContainSubstring(`InternalState`))
			})

			It("Should skip enums with @ts omit directive", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Status enum {
						active = 1
						inactive = 2
					}

					DebugLevel enum {
						verbose = 0
						trace = 1
						@ts omit
					}
				`
				resp := MustGenerate(ctx, source, "status", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export enum Status`))
				Expect(content).NotTo(ContainSubstring(`DebugLevel`))
			})
		})

		Context("documentation", func() {
			It("Should generate JSDoc comments from doc domain", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`/** User A User represents a user in the system. */`))
				Expect(content).To(ContainSubstring(`/** key The unique identifier for the user. */`))
				Expect(content).To(ContainSubstring(`/** name The user's display name. */`))
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

			It("Should import cross-namespace struct type", func(ctx SpecContext) {
				source := `
					import "schemas/common"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						info common.Info
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`import { common } from "@/common"`,
						`common.infoZ`,
					)
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

			It("Should import cross-namespace enum type", func(ctx SpecContext) {
				source := `
					import "schemas/status"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						code status.StatusCode
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`import { status } from`,
						`status.statusCodeZ`,
					)
			})

			It("Should not generate a local copy of cross-namespace enum", func(ctx SpecContext) {
				source := `
					import "schemas/status"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						code status.StatusCode
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
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

			It("Should use internal prefix for same-package imports", func(ctx SpecContext) {
				source := `
					import "schemas/common"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						info common.Info
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`import { common } from "@/common"`)
			})
		})

		Context("typedef alias form", func() {
			It("Should generate type alias for alias typedef", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Key = uint32

					User struct {
						key Key
					}
				`
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("keyZ")
			})

			It("Should generate typedef with array alias", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Tags = string[]
				`
				resp := MustGenerate(ctx, source, "tag", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("tagsZ")
			})
		})

		Context("struct with @ts type override on field", func() {
			It("Should use type override for field type", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "user", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("keyZ")
			})

			It("Should use a non-primitive type override referencing another schema type", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("extent: boundedZ").
					ToNotContain("extent: spanZ").
					ToNotContain("extent: z.unknown()")
			})
		})

		Context("struct field with @ts pick", func() {
			It("Should narrow a struct-typed field to the picked fields", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(".omit({ parent: true })").
					ToContain("parent: rangeZ.pick({ key: true }).optional()")
			})
		})

		Context("struct with forward reference", func() {
			It("Should handle struct referencing a later-declared struct", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("parentZ").
					ToContain("childZ")
			})
		})

		Context("self-referencing struct", func() {
			It("Should handle struct with self reference", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Node struct {
						key uuid
						parent Node?
					}
				`
				resp := MustGenerate(ctx, source, "tree", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("nodeZ")
			})
		})

		Context("generic struct with type parameter", func() {
			It("Should generate generic struct with zod function", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Response struct<T> {
						data T
						status int32
					}
				`
				resp := MustGenerate(ctx, source, "api", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("responseZ")
			})
		})

		Context("struct with map field", func() {
			It("Should generate record type for map field", func(ctx SpecContext) {
				source := `
					@ts output "out"

					Config struct {
						settings map<string, record>
					}
				`
				resp := MustGenerate(ctx, source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("configZ").
					ToContain("settings:")
			})
		})

		Context("struct extends another struct", func() {
			It("Should include parent fields in output", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("derivedZ").
					ToContain("extra:")
			})
		})

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

			It("Should generate cross-namespace struct field reference with import", func(ctx SpecContext) {
				source := `
					import "schemas/common"

					@ts output "client/ts/src/task"

					Task struct {
						key uuid
						info common.Info
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain("taskZ").
					ToContain("common")
			})
		})

		Context("coalesce_type_params", func() {
			It("Should generate single schema param for type interface", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`export interface TaskSchemas<`).
					ToContain(`export const taskZ = <`).
					ToContain(`export interface Task<`).
					ToContain(`S extends TaskSchemas = TaskSchemas`).
					ToContain(`z.infer<S["type"]>`).
					ToContain(`z.infer<S["config"]>`)
			})

			It("Should generate coalesced type for extends struct", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`export type New<`).
					ToContain(`S extends NewSchemas = NewSchemas`).
					ToContain(`Task<S>`)
			})

			It("Should coalesce conditional field references", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`S extends TaskSchemas = TaskSchemas`).
					ToContain(`S["data"]`)
			})

			It("Should coalesce extend fields referencing type params", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "test", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`export type New<S extends NewSchemas = NewSchemas>`).
					ToContain(`NewWrapper<S["statusData"]>`)
			})
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

			It("Should use package name for different package imports", func(ctx SpecContext) {
				source := `
					import "schemas/telem"

					@ts output "client/ts/src/channel"

					Channel struct {
						key uint32
						data_type telem.DataType
					}
				`
				resp := MustGenerate(ctx, source, "channel", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`import { telem } from "@synnaxlabs/x"`)
			})
		})

		Context("telem numeric defaults", func() {
			BeforeEach(func() {
				loader.Add("schemas/telem", `
					@ts output "x/ts/src/telem"

					TimeSpan int64 {
						@ts omit
					}

					Rate float64 {
						@ts omit
					}
				`)
			})

			It("Should construct telem instances for numeric defaults", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(
						`sampleRate: telem.rateZ.default(new Rate(10))`,
						`streamRate: telem.rateZ.default(new Rate(2.5))`,
						`duration: telem.timeSpanZ.default(TimeSpan.ZERO)`,
						`window: telem.timeSpanZ.default(new TimeSpan(100))`,
					)
			})

			It("Should emit a plain number default when a telem field is overridden to number", func(ctx SpecContext) {
				source := `
					import "schemas/telem"

					@ts output "out"

					Config struct {
						sample_rate telem.Rate = 10 {
							@ts type "number"
						}
					}
				`
				resp := MustGenerate(ctx, source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`sampleRate: z.number().default(10)`)
			})
		})

		Context("enum variant defaults", func() {
			It("Should generate default for same-namespace enum variant", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`mode: modeZ.default(Mode.automatic)`)
			})

			It("Should resolve a bare multi-word snake_case variant name as a default", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`terminal: terminalConfigZ.default("Cfg_Default")`)
			})

			It("Should generate default for cross-namespace enum variant", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "channel", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`concurrency: control.concurrencyZ.default(control.Concurrency.exclusive)`)
			})

			It("Should emit a string literal default for a string-valued enum variant", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "config", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`level: levelZ.default("info")`)
			})

			It("Should emit a string literal default for a cross-namespace string-valued enum variant", func(ctx SpecContext) {
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
				resp := MustGenerate(ctx, source, "lineplot", loader, typesPlugin)
				ExpectContent(resp, "types.gen.ts").
					ToContain(`level: text.levelZ.default("p")`)
			})
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

	It("Should generate a discriminated union with per-variant schemas", func(ctx SpecContext) {
		source := `
			@ts output "out"

			LinearScale struct {
				slope float64
				yIntercept float64
			}
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`export const scaleLinearZ = linearScaleZ.extend({`,
				`type: z.literal("linear"),`,
				`export const scaleNoneZ = noneScaleZ.extend({`,
				`type: z.literal("none"),`,
				`export const scaleZ = z.discriminatedUnion("type", [`,
				`scaleLinearZ,`,
				`scaleNoneZ,`,
				`export type Scale = ScaleLinear | ScaleNone;`,
			)
	})

	It("Should declare inline variant fields directly on the member schema", func(ctx SpecContext) {
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
		resp := MustGenerate(ctx, source, "panel", loader, typesPlugin)
		content := ExpectContent(resp, "types.gen.ts")
		content.ToContain(
			`export const tabResourceZ = tabBaseZ.extend({`,
			`variant: z.literal("resource"),`,
			`resource: z.string(),`,
			`export const tabViewZ = tabBaseZ.extend(labeledZ.shape).extend({`,
			`variant: z.literal("view"),`,
			`type: z.string(),`,
			`export const tabEmptyZ = tabBaseZ.extend({`,
			`variant: z.literal("empty"),`,
		)
		content.ToNotContain("TabViewPayload", "tabViewPayloadZ")
	})

	It("Should carry inline variants through union composition", func(ctx SpecContext) {
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
		resp := MustGenerate(ctx, source, "schematic", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(
			`export const elementConfigBoxZ = z.object({`,
			`width: z.number(),`,
			`export const elementConfigPipeZ = z.object({`,
			`length: z.number(),`,
			`export type ElementConfig = ElementConfigBox | ElementConfigPipe;`,
		)
	})

	It("Should generate a bare object schema for an inline variant with no bases", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Tab union on variant {
				empty {}
			}
		`
		resp := MustGenerate(ctx, source, "panel", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(
			`export const tabEmptyZ = z.object({`,
			`variant: z.literal("empty"),`,
		)
	})

	It("Should break the inference cycle when a variant struct references the union recursively", func(ctx SpecContext) {
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
		resp := MustGenerate(ctx, source, "panel", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`get first(): z.ZodType<Node> {`,
				`return nodeZ;`,
				`export type Node = NodeLeaf | NodeSplit;`,
			).
			ToNotContain(`get first(): typeof nodeZ {`)
	})

	It("Should camelize multi-word discriminators in the generated schemas", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Spec struct { type string }

			Source union on value_type {
				boolean Spec
				number Spec
			}
		`
		resp := MustGenerate(ctx, source, "telem", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`valueType: z.literal("boolean"),`,
				`export const sourceZ = z.discriminatedUnion("valueType", [`,
			).
			ToNotContain(`value_type`)
	})

	It("Should generate a flat union for a union extending other unions", func(ctx SpecContext) {
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
		resp := MustGenerate(ctx, source, "schematic", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`export const elementConfigZ = z.discriminatedUnion("variant", [`,
				`elementConfigTankZ,`,
				`elementConfigPipeZ,`,
				`export const ELEMENT_CONFIG_TYPES = ["tank", "pipe"] as const;`,
			)
	})

	It("Should wrap preserve_keys map fields with caseconv.preserveKeys", func(ctx SpecContext) {
		source := `
			@ts output "out"

			Config struct { variant string }

			Schematic struct {
				configs map<string, Config> {
					@ts preserve_keys
				}
			}
		`
		resp := MustGenerate(ctx, source, "schematic", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`configs: caseconv.preserveKeys(z.record(z.string(), configZ).default(() => ({}))),`,
			)
	})

	It("Should generate a discriminator enum and per-variant interfaces", func(ctx SpecContext) {
		source := `
			@ts output "out"

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`export const SCALE_TYPES = ["linear", "none"] as const;`,
				`export const scaleTypeZ = z.enum(SCALE_TYPES);`,
				`export type ScaleType = z.infer<typeof scaleTypeZ>;`,
				`export interface ScaleLinear extends z.infer<typeof scaleLinearZ> {}`,
			)
	})

	It("Should generate a schema map keyed by discriminator value", func(ctx SpecContext) {
		source := `
			@ts output "out"

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`export const SCALE_SCHEMAS: {`,
				`  [K in ScaleType]: z.ZodType<Extract<Scale, { type: K }>>;`,
				`} = {`,
				`linear: scaleLinearZ,`,
				`none: scaleNoneZ,`,
			)
	})

	It("Should compose the union base and payload via extend, not flatten", func(ctx SpecContext) {
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
		resp := MustGenerate(ctx, source, "ni", loader, typesPlugin)
		content := MustContentOf(resp, "types.gen.ts")
		// The variant composes the base and payload schemas and adds only the
		// discriminator; the shared fields live on baseAIChanZ / voltageFieldsZ.
		Expect(content).To(ContainSubstring("export const aiVoltageChannelZ = baseAIChanZ.extend(voltageFieldsZ.shape).extend({\n  type: z.literal(\"ai_voltage\"),\n});"))
		Expect(content).To(ContainSubstring(`port: z.int32(),`))
		Expect(content).To(ContainSubstring(`minVal: z.number(),`))
	})

	It("Should resolve a union-typed struct field to the union schema", func(ctx SpecContext) {
		source := `
			@ts output "out"

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}

			Channel struct {
				customScale Scale
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(`customScale: scaleZ,`)
	})

	It("Should support a variant field that is itself a union", func(ctx SpecContext) {
		source := `
			@ts output "out"

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}

			VoltageFields struct { customScale Scale }

			AIChannel union on type {
				ai_voltage VoltageFields
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				`export const aiVoltageChannelZ = voltageFieldsZ.extend({`,
				`customScale: scaleZ,`,
			)
	})

	It("Should render the union doc comment", func(ctx SpecContext) {
		source := `
			@ts output "out"

			LinearScale struct { slope float64 }

			Scale union on type {
				linear LinearScale

				@doc value "determines how raw values are transformed."
			}
		`
		resp := MustGenerate(ctx, source, "ni", loader, typesPlugin)
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

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale {
					@doc value "a linear scale."
				}
				none NoneScale

				@doc value "determines how raw values are transformed."
			}

			Channel struct {
` + channelFields + `
			}
		`
	}

	It("Should render the union doc directly above the discriminatedUnion, not the first variant", func(ctx SpecContext) {
		resp := MustGenerate(ctx, scaleSource("\t\t\t\tcustomScale Scale"), "ni", loader, typesPlugin)
		content := MustContentOf(resp, "types.gen.ts")
		Expect(content).To(ContainSubstring("/** Scale determines how raw values are transformed. */\nexport const scaleZ = z.discriminatedUnion("))
		Expect(content).ToNot(ContainSubstring("/** Scale determines how raw values are transformed. */\nexport const scaleLinearZ"))
	})

	It("Should render a per-variant doc comment directly above the variant schema", func(ctx SpecContext) {
		resp := MustGenerate(ctx, scaleSource("\t\t\t\tcustomScale Scale"), "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				"/** ScaleLinear a linear scale. */\nexport const scaleLinearZ = linearScaleZ.extend({",
				`export interface ScaleLinear extends z.infer<typeof scaleLinearZ> {}`,
			)
	})

	It("Should resolve an optional union-typed field as null-tolerant", func(ctx SpecContext) {
		resp := MustGenerate(ctx, scaleSource("\t\t\t\tcustomScale Scale?"), "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(
			`customScale: zod.nullToUndefined(scaleZ),`,
		)
	})

	It("Should resolve an array-of-union field", func(ctx SpecContext) {
		resp := MustGenerate(ctx, scaleSource("\t\t\t\tscales Scale[]"), "ni", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(`scales: scaleZ.array().default(() => []),`)
	})

	It("Should import and qualify a cross-namespace union reference", func(ctx SpecContext) {
		loader.Add("schemas/scales", `
			@ts output "client/ts/src/scales"

			LinearScale struct { slope float64 }
			NoneScale struct {}

			Scale union on type {
				linear LinearScale
				none NoneScale
			}
		`)
		source := `
			import "schemas/scales"

			@ts output "client/ts/src/task"

			Channel struct {
				customScale scales.Scale
			}
		`
		resp := MustGenerate(ctx, source, "task", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(`scales.scaleZ`)
	})
})
