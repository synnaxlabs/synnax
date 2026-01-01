// Copyright 2025 Synnax Labs, Inc.
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`import { z } from "zod"`))
			Expect(content).To(ContainSubstring(`export const userZ = z.object(`))
			Expect(content).To(ContainSubstring(`key: z.uuid()`))
			Expect(content).To(ContainSubstring(`name: z.string()`))
			Expect(content).To(ContainSubstring(`age: z.int32()`))
			Expect(content).To(ContainSubstring(`active: z.boolean()`))
			Expect(content).To(ContainSubstring(`export interface User extends z.infer<typeof userZ> {}`))
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Required arrays use nullishToEmpty to coerce nullish -> []
			Expect(content).To(ContainSubstring(`labels: array.nullishToEmpty(z.uuid())`))
			Expect(content).To(ContainSubstring(`parent: z.uuid().optional()`))
			// Optional arrays use nullToUndefined to preserve undefined vs []
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
					email string @validate email
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`name: z.string().min(1).max(255)`))
			Expect(content).To(ContainSubstring(`email: z.string().email()`))
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
				OutputDir:   "out",
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// String enums generate a const array and z.enum([...ARRAY])
			Expect(content).To(ContainSubstring(`export const DATA_TYPES = ["float32", "float64", "int32"] as const`))
			Expect(content).To(ContainSubstring(`export const dataTypeZ = z.enum([...DATA_TYPES])`))
		})

		It("Should handle primitive type mappings", func() {
			source := `
				@ts output "out"

				AllTypes struct {
					a uuid
					b string
					c bool
					d int8
					e int16
					f int32
					g int64
					h uint8
					i uint16
					j uint32
					k uint64
					l float32
					m float64
					n timestamp
					o timespan
					p json
					q bytes
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`a: z.uuid()`))
			Expect(content).To(ContainSubstring(`b: z.string()`))
			Expect(content).To(ContainSubstring(`c: z.boolean()`))
			Expect(content).To(ContainSubstring(`d: zod.int8Z`))
			Expect(content).To(ContainSubstring(`e: zod.int16Z`))
			Expect(content).To(ContainSubstring(`f: z.int32()`))
			Expect(content).To(ContainSubstring(`g: zod.int64Z`))
			Expect(content).To(ContainSubstring(`h: zod.uint8Z`))
			Expect(content).To(ContainSubstring(`i: zod.uint16Z`))
			Expect(content).To(ContainSubstring(`j: z.uint32()`))
			Expect(content).To(ContainSubstring(`k: zod.uint64Z`))
			Expect(content).To(ContainSubstring(`l: z.number()`))
			Expect(content).To(ContainSubstring(`m: z.number()`))
			Expect(content).To(ContainSubstring(`n: TimeStamp.z`))
			Expect(content).To(ContainSubstring(`o: TimeSpan.z`))
			Expect(content).To(ContainSubstring(`p: zod.stringifiedJSON`))
			Expect(content).To(ContainSubstring(`q: z.instanceof(Uint8Array)`))
			Expect(content).To(ContainSubstring(`import { TimeSpan, TimeStamp, zod } from "@synnaxlabs/x"`))
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
				OutputDir:   "out",
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
				OutputDir:   "out",
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
				OutputDir:   "out",
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
				OutputDir:   "out",
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`import { array } from "@synnaxlabs/x"`))
			// Required arrays use nullishToEmpty to coerce nullish -> []
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Optional arrays use nullToUndefined to preserve undefined vs []
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`username: z.string().min(1, "Username is required")`))
			Expect(content).To(ContainSubstring(`firstName: z.string().min(1, "First Name is required")`))
		})

		It("Should use z.input and jsonStringifier when use_input is specified in ts domain", func() {
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface New extends z.input<typeof newZ> {}`))
			Expect(content).To(ContainSubstring(`data: zod.jsonStringifier`))
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
				OutputDir:   "out",
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
				OutputDir:   "out",
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const nodeZ = z.object({`))
			Expect(content).To(ContainSubstring(`get children() {`))
			// Optional arrays use array.nullToUndefined
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
				OutputDir:   "out",
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const treeNodeZ = <K extends z.ZodType = z.ZodString>(k?: K) =>`))
			Expect(content).To(ContainSubstring(`get children() {`))
			// Optional arrays use array.nullToUndefined
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export interface MapNodeSchemas<K extends z.ZodType = z.ZodString, V extends z.ZodType = z.ZodString>`))
			Expect(content).To(ContainSubstring(`}: MapNodeSchemas<K, V> = {}) =>`))
			Expect(content).To(ContainSubstring(`get children() {`))
			// Optional arrays use array.nullToUndefined
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const simpleZ = z.object({`))
			Expect(content).To(ContainSubstring(`key: z.uuid()`))
			Expect(content).To(ContainSubstring(`name: z.string()`))
			Expect(content).NotTo(ContainSubstring(`get `)) // No getters for non-recursive types
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
				OutputDir:   "out",
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
				OutputDir:   "out",
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
			Expect(content).To(ContainSubstring(`.omit({`))
			Expect(content).To(ContainSubstring(`a: true`))
			Expect(content).To(ContainSubstring(`c: true`))
		})

		It("Should handle field override to make it optional", func() {
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// Child should use partial for optionality-only changes
			Expect(content).To(ContainSubstring(`export const childZ = parentZ`))
			Expect(content).To(ContainSubstring(`.partial({ name: true })`))
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
				OutputDir:   "out",
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
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			// RackStatus should extend Status (generic type args are handled via extends)
			Expect(content).To(ContainSubstring(`export const rackStatusZ = statusZ`))
			Expect(content).To(ContainSubstring(`.extend({`))
			Expect(content).To(ContainSubstring(`timestamp: TimeStamp.z`))
		})

		It("Should preserve struct declaration order", func() {
			source := `
				@ts output "out"

				Zebra struct {
					name string
				}

				Apple struct {
					color string
				}

				Mango struct {
					ripe bool
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "animals", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			req := &plugin.Request{
				Resolutions: table,
				OutputDir:   "out",
			}

			resp, err := typesPlugin.Generate(req)
			Expect(err).To(BeNil())

			content := string(resp.Files[0].Content)
			zebraIdx := strings.Index(content, "zebraZ")
			appleIdx := strings.Index(content, "appleZ")
			mangoIdx := strings.Index(content, "mangoZ")
			Expect(zebraIdx).To(BeNumerically("<", appleIdx))
			Expect(appleIdx).To(BeNumerically("<", mangoIdx))
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
				OutputDir:   "out",
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
	})
})
