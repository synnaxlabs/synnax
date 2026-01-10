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
	"context"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	"github.com/synnaxlabs/oracle/testutil"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TS Types Plugin Suite")
}

var _ = Describe("TS Types Plugin", func() {
	var (
		ctx         context.Context
		loader      *testutil.MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = testutil.NewMockFileLoader()
		typesPlugin = types.New(types.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(typesPlugin.Name()).To(Equal("ts/types"))
		})

		It("Should have no domain filter", func() {
			Expect(typesPlugin.Domains()).To(BeEmpty())
		})
	})

	Describe("Generate", func() {
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
				resp := testutil.MustGenerate(ctx, source, "user", loader, typesPlugin)
				Expect(resp.Files).To(HaveLen(1))

				testutil.ExpectContent(resp, "types.gen.ts").
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Required arrays use nullishToEmpty with element schema (not wrapped in z.array)
			// The array wrapper handles the array semantics
			Expect(content).To(ContainSubstring(`labels: array.nullishToEmpty(z.uuid())`))
			Expect(content).To(ContainSubstring(`parent: z.uuid().optional()`))
			// Optional arrays use nullToUndefined with element schema
			Expect(content).To(ContainSubstring(`tags: array.nullToUndefined(z.string())`))
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`name: z.string().min(1).max(255)`))
			Expect(content).To(ContainSubstring(`email: z.string()`))
			Expect(content).To(ContainSubstring(`age: z.int32().min(0).max(150)`))
		})

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
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Int enums generate a TypeScript enum and z.enum(EnumName)
			Expect(content).To(ContainSubstring(`export enum TaskState`))
			Expect(content).To(ContainSubstring(`pending = 0`))
			Expect(content).To(ContainSubstring(`running = 1`))
			Expect(content).To(ContainSubstring(`completed = 2`))
			Expect(content).To(ContainSubstring(`export const taskStateZ = z.enum(TaskState)`))
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "telem", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// String enums generate a const array and z.enum([...ARRAY])
			Expect(content).To(ContainSubstring(`export const DATA_TYPES = ["float32", "float64", "int32"] as const`))
			Expect(content).To(ContainSubstring(`export const dataTypeZ = z.enum([...DATA_TYPES])`))
		})

		Context("primitive type mappings", func() {
			DescribeTable("should generate correct Zod schema",
				func(oracleType, expectedZodType string) {
					source := `
						@ts output "out"

						Test struct {
							field ` + oracleType + `
						}
					`
					resp := testutil.MustGenerate(ctx, source, "test", loader, typesPlugin)
					testutil.ExpectContent(resp, "types.gen.ts").ToContain("field: " + expectedZodType)
				},
				Entry("uuid", "uuid", "z.uuid()"),
				Entry("string", "string", "z.string()"),
				Entry("bool", "bool", "z.boolean()"),
				Entry("int8", "int8", "zod.int8Z"),
				Entry("int16", "int16", "zod.int16Z"),
				Entry("int32", "int32", "z.int32()"),
				Entry("int64", "int64", "z.int64()"),
				Entry("uint8", "uint8", "zod.uint8Z"),
				Entry("uint16", "uint16", "zod.uint16Z"),
				Entry("uint32", "uint32", "z.uint32()"),
				Entry("uint64", "uint64", "z.uint64()"),
				Entry("float32", "float32", "z.number()"),
				Entry("float64", "float64", "z.number()"),
				Entry("timestamp", "timestamp", "TimeStamp.z"),
				Entry("timespan", "timespan", "TimeSpan.z"),
				Entry("json", "json", "zod.stringifiedJSON()"),
				Entry("bytes", "bytes", "z.instanceof(Uint8Array)"),
			)

			It("Should import required packages for special types", func() {
				source := `
					@ts output "out"

					Test struct {
						a timestamp
						b timespan
						c int8
					}
				`
				resp := testutil.MustGenerate(ctx, source, "test", loader, typesPlugin)
				testutil.ExpectContent(resp, "types.gen.ts").
					ToContain(`import { TimeSpan, TimeStamp, zod } from "@synnaxlabs/x"`)
			})
		})

		Context("@ts to_number directive", func() {
			It("Should generate schema that accepts strings and converts to number with NaN validation", func() {
				source := `
					@ts output "out"

					Key uint32 {
						@ts to_number
					}
				`
				resp := testutil.MustGenerate(ctx, source, "channel", loader, typesPlugin)
				testutil.ExpectContent(resp, "types.gen.ts").
					ToContain(`export const keyZ = z.uint32().or(z.string().refine((v) => !isNaN(Number(v))).transform(Number))`)
			})
		})

		Context("@ts to_string directive", func() {
			It("Should generate schema that accepts numbers and converts to string", func() {
				source := `
					@ts output "out"

					Name string {
						@ts to_string
					}
				`
				resp := testutil.MustGenerate(ctx, source, "user", loader, typesPlugin)
				testutil.ExpectContent(resp, "types.gen.ts").
					ToContain(`export const nameZ = z.string().or(z.number().transform(String))`)
			})
		})

		It("Should convert snake_case to camelCase for field names", func() {
			source := `
				@ts output "out"

				Range struct {
					created_at timestamp
					time_range string
					my_long_field_name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`createdAt:`))
			Expect(content).To(ContainSubstring(`timeRange:`))
			Expect(content).To(ContainSubstring(`myLongFieldName:`))
		})

		It("Should generate create request struct with optional key and password", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const newZ = z.object({`))
			Expect(content).To(ContainSubstring(`key: z.uuid().optional()`))
			Expect(content).To(ContainSubstring(`username: z.string()`))
			Expect(content).To(ContainSubstring(`password: z.string().min(1)`))
			Expect(content).To(ContainSubstring(`firstName: z.string().optional()`))
			Expect(content).To(ContainSubstring(`lastName: z.string().optional()`))
			Expect(content).To(ContainSubstring(`export interface New extends z.infer<typeof newZ> {}`))
		})

		It("Should handle soft optional types (?)", func() {
			source := `
				@ts output "out"

				Device struct {
					key uuid
					name string
					status string?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "device", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Soft optional (?) uses .optional() in TypeScript
			Expect(content).To(ContainSubstring(`status: z.string().optional()`))
		})

		It("Should handle hard optional types (??)", func() {
			source := `
				@ts output "out"

				Task struct {
					key uuid
					name string
					status string??
					description string??
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Hard optional (??) also uses .optional() in TypeScript (no distinction from ?)
			Expect(content).To(ContainSubstring(`status: z.string().optional()`))
			Expect(content).To(ContainSubstring(`description: z.string().optional()`))
		})

		It("Should handle required arrays with array.nullishToEmpty", func() {
			source := `
				@ts output "out"

				Policy struct {
					key uuid
					objects uuid[]
					actions string[]
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "policy", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`import { array } from "@synnaxlabs/x"`))
			// Required arrays use nullishToEmpty with element schema (not double-wrapped with z.array)
			Expect(content).To(ContainSubstring(`objects: array.nullishToEmpty(z.uuid())`))
			Expect(content).To(ContainSubstring(`actions: array.nullishToEmpty(z.string())`))
		})

		It("Should handle optional arrays with array.nullToUndefined", func() {
			source := `
				@ts output "out"

				Channel struct {
					key uuid
					operations string[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "channel", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Optional arrays use nullToUndefined with element schema (not double-wrapped)
			Expect(content).To(ContainSubstring(`operations: array.nullToUndefined(z.string())`))
		})

		It("Should generate error message for required validation", func() {
			source := `
				@ts output "out"

				User struct {
					key uuid
					username string @validate required
					first_name string @validate required
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`username: z.string().min(1, "Username is required")`))
			Expect(content).To(ContainSubstring(`firstName: z.string().min(1, "First Name is required")`))
		})

		It("Should use z.input and jsonStringifier when use_input and stringify are specified", func() {
			source := `
				@ts output "out"

				New struct {
					key uuid?
					name string
					data json @ts stringify

					@ts use_input
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface New extends z.input<typeof newZ> {}`))
			Expect(content).To(ContainSubstring(`data: zod.jsonStringifier`))
		})

		It("Should use stringifiedJSON for json fields without @ts stringify even when use_input is specified", func() {
			source := `
				@ts output "out"

				New struct {
					key uuid?
					name string
					data json

					@ts use_input
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface New extends z.input<typeof newZ> {}`))
			// Without @ts stringify on the field, should use stringifiedJSON
			Expect(content).To(ContainSubstring(`data: zod.stringifiedJSON()`))
		})

		It("Should use jsonStringifier for overridden JSON fields with @ts stringify in child struct", func() {
			source := `
				@ts output "out"

				Parent struct<Properties extends json = json> {
					name string
					properties Properties
				}

				Child struct<Properties extends json = json> extends Parent<Properties> {
					key uuid?
					properties Properties @ts stringify
					@ts use_input
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Child should omit properties from parent and re-extend with jsonStringifier
			Expect(content).To(ContainSubstring(`.omit({ properties: true })`))
			Expect(content).To(ContainSubstring(`properties: zod.jsonStringifier(properties)`))
			// Type should use z.input
			Expect(content).To(ContainSubstring(`z.input<`))
		})

		It("Should NOT auto-stringify inherited JSON fields without @ts stringify", func() {
			source := `
				@ts output "out"

				Parent struct<Properties extends json = json> {
					name string
					properties Properties
				}

				Child struct<Properties extends json = json> extends Parent<Properties> {
					key uuid?
					@ts use_input
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Child should NOT automatically omit and re-extend properties
			// The inherited properties field should use parent's stringifiedJSON
			Expect(content).NotTo(ContainSubstring(`properties: zod.jsonStringifier`))
		})

		It("Should use z.infer by default without use_input", func() {
			source := `
				@ts output "out"

				Workspace struct {
					key uuid
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "workspace", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface Workspace extends z.infer<typeof workspaceZ> {}`))
		})

		It("Should generate getter for direct self-referencing struct", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const typeZ = z.object({`))
			Expect(content).To(ContainSubstring(`kind: kindZ`))
			Expect(content).To(ContainSubstring(`get elem() {`))
			Expect(content).To(ContainSubstring(`return typeZ.optional()`))
			Expect(content).To(ContainSubstring(`export interface Type extends z.infer<typeof typeZ> {}`))
		})

		It("Should generate getter for array self-referencing struct", func() {
			source := `
				@ts output "out"

				Node struct {
					key string
					children Node[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "tree", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const nodeZ = z.object({`))
			Expect(content).To(ContainSubstring(`get children() {`))
			// Optional arrays use array.nullToUndefined with element schema
			Expect(content).To(ContainSubstring(`return array.nullToUndefined(nodeZ)`))
		})

		It("Should generate getter for struct with multiple recursive fields", func() {
			source := `
				@ts output "out"

				MosaicNode struct {
					key int32
					first MosaicNode?
					last MosaicNode?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "mosaic", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const mosaicNodeZ = z.object({`))
			Expect(content).To(ContainSubstring(`get first() {`))
			Expect(content).To(ContainSubstring(`return mosaicNodeZ.optional()`))
			Expect(content).To(ContainSubstring(`get last() {`))
		})

		It("Should generate getter for generic recursive struct with single param", func() {
			source := `
				@ts output "out"

				TreeNode struct<K extends schema = string> {
					key K
					children TreeNode<K>[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "tree", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const treeNodeZ = <K extends z.ZodType = z.ZodString>(k?: K) =>`))
			Expect(content).To(ContainSubstring(`get children() {`))
			// Optional arrays use array.nullToUndefined with element schema
			Expect(content).To(ContainSubstring(`return array.nullToUndefined(treeNodeZ(k))`))
		})

		It("Should generate getter for generic recursive struct with multiple params", func() {
			source := `
				@ts output "out"

				MapNode struct<K extends schema = string, V extends schema = string> {
					key K
					value V
					children MapNode<K, V>[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "tree", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface MapNodeSchemas<K extends z.ZodType = z.ZodString, V extends z.ZodType = z.ZodString>`))
			Expect(content).To(ContainSubstring(`}: MapNodeSchemas<K, V> = {}) =>`))
			Expect(content).To(ContainSubstring(`get children() {`))
			// Optional arrays use array.nullToUndefined with element schema
			Expect(content).To(ContainSubstring(`return array.nullToUndefined(mapNodeZ({k: k, v: v}))`))
		})

		It("Should NOT generate getter for non-recursive struct", func() {
			source := `
				@ts output "out"

				Simple struct {
					key uuid
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "simple", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const simpleZ = z.object({`))
			Expect(content).To(ContainSubstring(`key: z.uuid()`))
			Expect(content).To(ContainSubstring(`name: z.string()`))
			Expect(content).NotTo(ContainSubstring(`get `)) // No getters for non-recursive types
		})

		It("Should generate fallback pattern for type param fields with string constraint", func() {
			source := `
				@ts output "out"

				Task struct<
					Type extends string = string,
					Config extends json = json
				> {
					name   string
					type   Type
					config Config
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// The 'type' field should use: type ?? z.string() since Type extends string
			Expect(content).To(ContainSubstring(`type: type ?? z.string()`), "type field should use type param with fallback")
			// The 'config' field should use stringifiedJSON since Config extends json
			Expect(content).To(ContainSubstring(`config: zod.stringifiedJSON(config)`), "config field should use zod.stringifiedJSON with type param")
			// The 'name' field should just be z.string() (not a type param)
			Expect(content).To(ContainSubstring(`name: z.string()`))
		})

		It("Should generate fallback pattern for type param fields with concrete_types directive", func() {
			source := `
				@ts output "out"

				Task struct<
					Type extends string = string,
					Config extends json = json
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Even with concrete_types, fields using type params should have the fallback pattern
			Expect(content).To(ContainSubstring(`type: type ?? z.string()`), "type field should use type param with fallback even with concrete_types")
			Expect(content).To(ContainSubstring(`config: zod.stringifiedJSON(config)`), "config field should use zod.stringifiedJSON with type param")
		})

		It("Should preserve type params when extending generic parent with pass-through type args", func() {
			source := `
				@ts output "out"

				Task struct<
					Type extends string = string,
					Config extends json = json
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
					Config extends json = json
				> extends Task<Type, Config> {
					key string?

					@ts {
						concrete_types
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// The parent Task should still have the fallback pattern
			Expect(content).To(ContainSubstring(`type: type ?? z.string()`), "parent Task type field should use type param with fallback")
			// The child New should extend the parent properly
			Expect(content).To(ContainSubstring(`newZ`), "should generate newZ schema")
		})

		It("Should generate .extend() for basic struct extension", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Child should use .omit() then .extend()
			Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
			Expect(content).To(ContainSubstring(`.omit({ age: true })`))
			Expect(content).To(ContainSubstring(`.extend({`))
			Expect(content).To(ContainSubstring(`email: z.string()`))
		})

		It("Should generate .omit() for multiple field omissions", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
			Expect(content).To(ContainSubstring(`.omit({`))
			Expect(content).To(ContainSubstring(`a: true`))
			Expect(content).To(ContainSubstring(`c: true`))
		})

		It("Should handle field override to make it optional using .partial()", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Child uses .partial() to make the field optional (not .extend())
			Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
			Expect(content).To(ContainSubstring(`.partial({ name: true })`))
			// Should NOT redefine the field in extend
			Expect(content).NotTo(ContainSubstring(`name: z.string().optional()`))
		})

		It("Should handle extension without new fields (only omissions)", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
			Expect(content).To(ContainSubstring(`.omit({ b: true })`))
		})

		It("Should handle extension of generic struct with type arguments", func() {
			source := `
				@ts output "out"

				Details struct {
					message string
				}

				Status struct<D extends schema> {
					variant int32
					data D
				}

				RackStatus struct extends Status<Details> {
					timestamp timestamp
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// RackStatus should extend Status (generic type args are handled via extends)
			Expect(content).To(ContainSubstring(`export const rackStatusZ = statusZ`))
			Expect(content).To(ContainSubstring(`.extend({`))
			Expect(content).To(ContainSubstring(`timestamp: TimeStamp.z`))
		})

		It("Should generate .merge() chain for multiple extends", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// C should use .extend().shape to combine both parents
			Expect(content).To(ContainSubstring(`export const cZ = aZ.extend(bZ.shape)`))
			Expect(content).To(ContainSubstring(`.extend({`))
			Expect(content).To(ContainSubstring(`c: z.boolean()`))
		})

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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

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
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

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
			table, diag := analyzer.AnalyzeSource(ctx, source, "order", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			zebraIdx := strings.Index(content, "zebra:")
			appleIdx := strings.Index(content, "apple:")
			mangoIdx := strings.Index(content, "mango:")
			Expect(zebraIdx).To(BeNumerically("<", appleIdx))
			Expect(appleIdx).To(BeNumerically("<", mangoIdx))
		})

		It("Should generate type alias for generic struct reference", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// The alias should call the generic struct's factory function with the type arg
			Expect(content).To(ContainSubstring(`export const rackStatusZ = statusZ(statusDetailsZ)`))
			Expect(content).NotTo(ContainSubstring(`rackStatusZ = z.unknown()`))
		})

		It("Should not double-wrap arrays when using array helpers", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Should use operationZ directly, not z.array(operationZ)
			Expect(content).To(ContainSubstring(`operations: array.nullishToEmpty(operationZ)`))
			Expect(content).To(ContainSubstring(`optionalOps: array.nullToUndefined(operationZ)`))
			// Make sure we don't have the double-wrapped version
			Expect(content).NotTo(ContainSubstring(`z.array(operationZ)`))
		})

		Context("map types", func() {
			It("Should handle map with primitive key and value types", func() {
				source := `
					@ts output "out"

					Config struct {
						settings map<string, string>
					}
				`
				resp := testutil.MustGenerate(ctx, source, "config", loader, typesPlugin)
				testutil.ExpectContent(resp, "types.gen.ts").
					ToContain(
						`settings: z.record(z.string(), z.string())`,
					)
			})

			It("Should handle map with different primitive types", func() {
				source := `
					@ts output "out"

					Metrics struct {
						counts map<string, int64>
					}
				`
				resp := testutil.MustGenerate(ctx, source, "metrics", loader, typesPlugin)
				testutil.ExpectContent(resp, "types.gen.ts").
					ToContain(`counts: z.record(z.string(), z.int64())`)
			})

			It("Should handle map with struct value type", func() {
				source := `
					@ts output "out"

					Entry struct {
						value int32
					}

					Store struct {
						entries map<string, Entry>
					}
				`
				resp := testutil.MustGenerate(ctx, source, "store", loader, typesPlugin)
				testutil.ExpectContent(resp, "types.gen.ts").
					ToContain(`entries: z.record(z.string(), entryZ)`)
			})
		})

		Context("@omit directive", func() {
			It("Should skip types with @ts omit directive", func() {
				source := `
					@ts output "out"

					User struct {
						key uuid
						name string
					}

					InternalState struct {
						cache json
						@ts omit
					}
				`
				resp := testutil.MustGenerate(ctx, source, "user", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export const userZ`))
				Expect(content).NotTo(ContainSubstring(`internalStateZ`))
				Expect(content).NotTo(ContainSubstring(`InternalState`))
			})

			It("Should skip enums with @ts omit directive", func() {
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
				resp := testutil.MustGenerate(ctx, source, "status", loader, typesPlugin)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`export enum Status`))
				Expect(content).NotTo(ContainSubstring(`DebugLevel`))
			})
		})
	})
})
