// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

func findField(fields []resolution.Field, name string) resolution.Field {
	for _, f := range fields {
		if f.Name == name {
			return f
		}
	}
	Fail("field not found: " + name)
	return resolution.Field{}
}

func domainExprNames(d resolution.Domain) []string {
	names := make([]string, len(d.Expressions))
	for i, e := range d.Expressions {
		names[i] = e.Name
	}
	return names
}

var _ = Describe("Analyzer", func() {
	var (
		loader *MockFileLoader
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
	})

	Describe("File-level version", func() {
		It("Should error when @go version is declared file-level", func(ctx SpecContext) {
			source := `
				@go output "out"
				@go version 0
				Entry struct {
					value int32
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("declare it per type"))
		})

		It("Should accept struct-level @go version", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					value int32
					@go version 0
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
		})

		It("Should accept a standalone struct-level @go pinned", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					value int32
					@go pinned
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
		})

		It("Should error when @go pinned is declared file-level", func(ctx SpecContext) {
			source := `
				@go output "out"
				@go pinned
				Entry struct {
					value int32
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("declare it per type"))
		})

		It("Should error when @go pinned carries arguments", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					value int32
					@go pinned 2
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("takes no arguments"))
		})
	})

	Describe("Version arguments", func() {
		It("Should accept a pinned marker", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					value int32
					@go version 0 pinned
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
		})

		It("Should error on an unknown version argument", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					value int32
					@go version 0 pined
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("malformed @go version"))
		})

		It("Should error on extra version arguments", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					value int32
					@go version 0 pinned pinned
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("malformed @go version"))
		})
	})

	Describe("Domain omission", func() {
		It("Should error when a generating type references an omitted type", func(ctx SpecContext) {
			source := `
				@go output "out"
				Inner struct {
					value int32
					@go omit
				}
				Entry struct {
					inner Inner
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("omitted in go"))
		})

		It("Should allow references to hand-written types", func(ctx SpecContext) {
			source := `
				@go output "out"
				Inner struct {
					value int32
					@go hand
				}
				Entry struct {
					inner Inner
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
		})

		It("Should error when every type omits a declared output", func(ctx SpecContext) {
			source := `
				@go output "out"
				@ts output "ts/out"
				Entry struct {
					value int32
					@ts omit
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("remove the @ts output"))
		})

		It("Should keep an output alive through a hand-written type", func(ctx SpecContext) {
			source := `
				@go output "out"
				@ts output "ts/out"
				Entry struct {
					value int32
					@ts hand
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
		})
	})

	Describe("Imports", func() {
		It("Should error on an unused import", func(ctx SpecContext) {
			loader.Add("schemas/dep", `
				Inner struct { value int32 }
			`)
			source := `
				import "schemas/dep"
				Entry struct {
					name string
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring(`unused import "schemas/dep"`))
		})

		It("Should not error when an import is referenced", func(ctx SpecContext) {
			loader.Add("schemas/dep", `
				Inner struct { value int32 }
			`)
			source := `
				import "schemas/dep"
				Entry struct {
					inner dep.Inner
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
		})
	})

	Describe("AnalyzeSource", func() {
		It("Should analyze a simple struct", func(ctx SpecContext) {
			source := `
				Range struct {
					key uuid @key
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.Ok()).To(BeTrue())
			Expect(table).NotTo(BeNil())
			Expect(table.StructTypes()).To(HaveLen(1))

			rangeType := table.MustGet("ranger.Range")
			Expect(rangeType.Name).To(Equal("Range"))
			Expect(rangeType.Namespace).To(Equal("ranger"))

			form, ok := rangeType.Form.(resolution.StructForm)
			Expect(ok).To(BeTrue())
			Expect(form.HasKeyDomain).To(BeTrue())
			Expect(form.Fields).To(HaveLen(2))

			keyField, found := form.Field("key")
			Expect(found).To(BeTrue())
			Expect(keyField.Type.Name).To(Equal("uuid"))
			Expect(resolution.IsPrimitive(keyField.Type.Name)).To(BeTrue())
			Expect(keyField.Domains).To(HaveKey("key"))

			nameField, found := form.Field("name")
			Expect(found).To(BeTrue())
			Expect(nameField.Type.Name).To(Equal("string"))
			Expect(resolution.IsPrimitive(nameField.Type.Name)).To(BeTrue())
		})

		It("Should analyze an enum", func(ctx SpecContext) {
			source := `
				TaskState enum {
					pending = 0
					running = 1
					completed = 2
					failed = 3
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.Ok()).To(BeTrue())
			Expect(table.EnumTypes()).To(HaveLen(1))

			taskStateType := table.MustGet("task.TaskState")
			Expect(taskStateType.Name).To(Equal("TaskState"))

			form, ok := taskStateType.Form.(resolution.EnumForm)
			Expect(ok).To(BeTrue())
			Expect(form.IsIntEnum).To(BeTrue())
			Expect(form.Values).To(HaveLen(4))
			Expect(form.Values[0].Name).To(Equal("pending"))
			Expect(form.Values[0].IntValue()).To(Equal(int64(0)))
		})

		It("Should analyze a string enum", func(ctx SpecContext) {
			source := `
				DataType enum {
					float32 = "float32"
					float64 = "float64"
					int32 = "int32"
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "telem", loader)
			Expect(diag.Ok()).To(BeTrue())

			dataTypeType := table.MustGet("telem.DataType")
			form, ok := dataTypeType.Form.(resolution.EnumForm)
			Expect(ok).To(BeTrue())
			Expect(form.IsIntEnum).To(BeFalse())
			Expect(form.Values[0].StringValue()).To(Equal("float32"))
		})

		It("Should collect enum value domains", func(ctx SpecContext) {
			source := `
				TaskState enum {
					pending = 0 {
						@doc description "The task is waiting to be executed"
					}
					running = 1 {
						@doc description "The task is currently being executed"
						@deprecated reason "Use active instead"
					}
					completed = 2
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.Ok()).To(BeTrue())

			taskStateType := table.MustGet("task.TaskState")
			form, ok := taskStateType.Form.(resolution.EnumForm)
			Expect(ok).To(BeTrue())
			Expect(form.Values).To(HaveLen(3))

			// First value has a doc domain
			Expect(form.Values[0].Name).To(Equal("pending"))
			Expect(form.Values[0].Domains).To(HaveLen(1))
			Expect(form.Values[0].Domains).To(HaveKey("doc"))
			docDomain := form.Values[0].Domains["doc"]
			Expect(docDomain.Expressions).To(HaveLen(1))
			Expect(docDomain.Expressions[0].Name).To(Equal("description"))
			Expect(docDomain.Expressions[0].Values[0].StringValue).To(Equal("The task is waiting to be executed"))

			// Second value has two domains
			Expect(form.Values[1].Name).To(Equal("running"))
			Expect(form.Values[1].Domains).To(HaveLen(2))
			Expect(form.Values[1].Domains).To(HaveKey("doc"))
			Expect(form.Values[1].Domains).To(HaveKey("deprecated"))

			// Third value has no domains
			Expect(form.Values[2].Name).To(Equal("completed"))
			Expect(form.Values[2].Domains).To(BeEmpty())
		})

		It("Should analyze an extending enum as the union of its parents", func(ctx SpecContext) {
			source := `
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

			axis := table.MustGet("lineplot.AxisKey")
			form, ok := axis.Form.(resolution.EnumForm)
			Expect(ok).To(BeTrue())
			Expect(form.IsExtension()).To(BeTrue())
			Expect(form.IsIntEnum).To(BeFalse())
			Expect(form.Values).To(HaveLen(4))
			var names []string
			for _, v := range form.Values {
				names = append(names, v.Name)
			}
			Expect(names).To(Equal([]string{"x1", "x2", "y1", "y2"}))
			Expect(form.Values[2].StringValue()).To(Equal("y1"))
		})

		It("Should let an extending enum add its own members", func(ctx SpecContext) {
			source := `
				Base enum {
					a = "a"
				}

				More enum extends Base {
					b = "b"
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeTrue())
			form := table.MustGet("x.More").Form.(resolution.EnumForm)
			Expect(form.Values).To(HaveLen(2))
			Expect(form.Values[0].Name).To(Equal("a"))
			Expect(form.Values[1].Name).To(Equal("b"))
		})

		It("Should inherit the int kind from int parent enums", func(ctx SpecContext) {
			source := `
				Low  enum { low  = 0 }
				High enum { high = 1 }

				Priority enum extends Low, High {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.Ok()).To(BeTrue())
			form := table.MustGet("task.Priority").Form.(resolution.EnumForm)
			Expect(form.IsIntEnum).To(BeTrue())
			Expect(form.Values[1].IntValue()).To(Equal(int64(1)))
		})

		It("Should reject extending a non-enum type", func(ctx SpecContext) {
			source := `
				Thing struct { name string }

				Bad enum extends Thing {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("which is not an enum"))
		})

		It("Should reject extending an unknown enum", func(ctx SpecContext) {
			source := `
				Bad enum extends Nonexistent {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("extends unknown enum"))
		})

		It("Should reject mixing string and int parent kinds", func(ctx SpecContext) {
			source := `
				Strs enum { a = "a" }
				Ints enum { b = 0 }

				Mixed enum extends Strs, Ints {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("mixed integer and string"))
		})

		It("Should reject parents that contribute conflicting member values", func(ctx SpecContext) {
			source := `
				A enum { shared = "one" }
				B enum { shared = "two" }

				C enum extends A, B {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("conflicting values"))
		})

		It("Should reject an enum that extends itself", func(ctx SpecContext) {
			source := `
				A enum extends A {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("cyclic extends chain"))
		})

		It("Should reject a cyclic extends chain", func(ctx SpecContext) {
			source := `
				A enum extends B {}
				B enum extends A {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("cyclic extends chain"))
		})

		It("Should accept a diamond where two parents share a common ancestor", func(ctx SpecContext) {
			source := `
				Base enum { b = "b" }
				Left  enum extends Base { l = "l" }
				Right enum extends Base { r = "r" }

				Diamond enum extends Left, Right {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeTrue())
			form := table.MustGet("x.Diamond").Form.(resolution.EnumForm)
			var names []string
			for _, v := range form.Values {
				names = append(names, v.Name)
			}
			// Base contributed once through each branch but is de-duplicated.
			Expect(names).To(Equal([]string{"b", "l", "r"}))
		})

		It("Should expand a multi-level extends chain", func(ctx SpecContext) {
			source := `
				C enum { c = "c" }
				B enum extends C { b = "b" }
				A enum extends B { a = "a" }
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			Expect(diag.Ok()).To(BeTrue())
			form := table.MustGet("x.A").Form.(resolution.EnumForm)
			var names []string
			for _, v := range form.Values {
				names = append(names, v.Name)
			}
			Expect(names).To(Equal([]string{"c", "b", "a"}))
		})

		It("Should extend enums imported from another namespace", func(ctx SpecContext) {
			source := `
				import "schemas/spatial"

				AxisKey enum extends spatial.XAxisKey, spatial.YAxisKey {}
			`
			loader.Add("schemas/spatial", `
				XAxisKey enum {
					x1 = "x1"
					x2 = "x2"
				}
				YAxisKey enum {
					y1 = "y1"
					y2 = "y2"
				}
			`)
			table, diag := analyzer.AnalyzeSource(ctx, source, "lineplot", loader)
			Expect(diag.Ok()).To(BeTrue())
			form := table.MustGet("lineplot.AxisKey").Form.(resolution.EnumForm)
			Expect(form.Values).To(HaveLen(4))
			Expect(form.Values[0].Name).To(Equal("x1"))
			Expect(form.Values[3].StringValue()).To(Equal("y2"))
		})

		It("Should collect field domains", func(ctx SpecContext) {
			source := `
				User struct {
					name string {
						@validate {
							required
							max_length 255
							min_length 1
						}
						@query {
							eq
							contains
						}
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.Ok()).To(BeTrue())

			userType := table.MustGet("user.User")
			form := userType.Form.(resolution.StructForm)
			nameField, found := form.Field("name")
			Expect(found).To(BeTrue())
			Expect(nameField.Domains).To(HaveLen(2))
			Expect(nameField.Domains).To(HaveKey("validate"))
			Expect(nameField.Domains).To(HaveKey("query"))

			// Check validate domain expressions
			validateDomain := nameField.Domains["validate"]
			Expect(validateDomain.Expressions).To(HaveLen(3))
			Expect(validateDomain.Expressions[0].Name).To(Equal("required"))
			Expect(validateDomain.Expressions[0].Values).To(BeEmpty())
			Expect(validateDomain.Expressions[1].Name).To(Equal("max_length"))
			Expect(validateDomain.Expressions[1].Values).To(HaveLen(1))
			Expect(validateDomain.Expressions[1].Values[0].Kind).To(Equal(resolution.ValueKindInt))
			Expect(validateDomain.Expressions[1].Values[0].IntValue).To(Equal(int64(255)))
		})

		It("Should collect struct-level domains", func(ctx SpecContext) {
			source := `
				Range struct {
					key uuid
					name string

					@index {
						composite name created_at sorted
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.Ok()).To(BeTrue())

			rangeType := table.MustGet("ranger.Range")
			Expect(rangeType.Domains).To(HaveLen(1))
			Expect(rangeType.Domains).To(HaveKey("index"))

			indexDomain := rangeType.Domains["index"]
			Expect(indexDomain.Expressions).To(HaveLen(1))
			Expect(indexDomain.Expressions[0].Name).To(Equal("composite"))
		})

		It("Should handle array types", func(ctx SpecContext) {
			source := `
				Range struct {
					labels uuid[]
					tags string[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.Ok()).To(BeTrue())

			rangeType := table.MustGet("ranger.Range")
			form := rangeType.Form.(resolution.StructForm)

			labelsField, _ := form.Field("labels")
			Expect(labelsField.Type.Name).To(Equal("Array"))
			Expect(labelsField.Type.TypeArgs).To(HaveLen(1))
			Expect(labelsField.Type.TypeArgs[0].Name).To(Equal("uuid"))
			Expect(labelsField.Optional).To(BeFalse())

			tagsField, _ := form.Field("tags")
			Expect(tagsField.Type.Name).To(Equal("Array"))
			Expect(tagsField.Optional).To(BeTrue())
		})

		It("Should handle optional types", func(ctx SpecContext) {
			source := `
				Range struct {
					parent uuid?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.Ok()).To(BeTrue())

			rangeType := table.MustGet("ranger.Range")
			form := rangeType.Form.(resolution.StructForm)
			parentField, _ := form.Field("parent")
			Expect(parentField.Optional).To(BeTrue())
			Expect(parentField.Type.Name).To(Equal("uuid"))
		})
	})

	Describe("Import Resolution", func() {
		It("Should resolve imports", func(ctx SpecContext) {
			loader.Files["schema/core/label"] = `
				Label struct {
					key uuid @id
					name string
				}
			`

			source := `
				import "schema/core/label"

				Range struct {
					key uuid @id
					labels label.Label[]
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.Ok()).To(BeTrue())

			// Both structs should be in the table
			Expect(table.StructTypes()).To(HaveLen(2))
			_, ok := table.Get("ranger.Range")
			Expect(ok).To(BeTrue())
			_, ok = table.Get("label.Label")
			Expect(ok).To(BeTrue())
		})

		It("Should detect circular imports", func(ctx SpecContext) {
			loader.Files["schema/core/a"] = `
				import "schema/core/b"
				A struct { b b.B? }
			`
			loader.Files["schema/core/b"] = `
				import "schema/core/a"
				B struct { a a.A? }
			`

			source := `
				import "schema/core/a"
				C struct { a a.A? }
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "main", loader)
			// Should not error - circular imports are handled by tracking
			Expect(diag.Ok()).To(BeTrue())
			Expect(table.StructTypes()).To(HaveLen(3))
		})

		It("Should report missing imports", func(ctx SpecContext) {
			source := `
				import "schema/core/nonexistent"
				Range struct {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag).To(HaveOccurred())
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})
	})

	Describe("Type Resolution", func() {
		It("Should resolve primitive types", func(ctx SpecContext) {
			source := `
				import "schemas/telem"

				Test struct {
					a uuid
					b string
					c int32
					d float64
					e bool
					f telem.timestamp
					g telem.timespan
					h telem.time_range
					i record
					j bytes
				}
			`
			loader.Add("schemas/telem", `
				timestamp uint64
				timespan int64
				time_range struct {
					start timestamp
					end timestamp
				}
			`)
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			testType := table.MustGet("test.Test")
			form := testType.Form.(resolution.StructForm)
			primitiveFields := []string{"a", "b", "c", "d", "e", "i", "j"}
			for _, field := range form.Fields {
				if slices.Contains(primitiveFields, field.Name) {
					Expect(resolution.IsPrimitive(field.Type.Name)).To(BeTrue())
				}
			}
		})

		It("Should resolve struct references in same namespace", func(ctx SpecContext) {
			source := `
				Position struct {
					x float64
					y float64
				}

				Viewport struct {
					position Position
					zoom float64
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "viz", loader)
			Expect(diag.Ok()).To(BeTrue())

			viewportType := table.MustGet("viz.Viewport")
			form := viewportType.Form.(resolution.StructForm)
			positionField, _ := form.Field("position")
			Expect(positionField.Type.Name).To(Equal("viz.Position"))

			// Verify it resolves to a struct
			resolved, ok := positionField.Type.Resolve(table)
			Expect(ok).To(BeTrue())
			_, isStruct := resolved.Form.(resolution.StructForm)
			Expect(isStruct).To(BeTrue())
		})

		It("Should resolve qualified struct references", func(ctx SpecContext) {
			loader.Files["schema/core/label"] = `
				Label struct {
					key uuid
					name string
				}
			`

			source := `
				import "schema/core/label"

				Range struct {
					labels label.Label[]
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.Ok()).To(BeTrue())

			rangeType := table.MustGet("ranger.Range")
			form := rangeType.Form.(resolution.StructForm)
			labelsField, _ := form.Field("labels")
			Expect(labelsField.Type.Name).To(Equal("Array"))
			Expect(labelsField.Type.TypeArgs[0].Name).To(Equal("label.Label"))

			// Verify it resolves to a struct
			resolved, ok := labelsField.Type.TypeArgs[0].Resolve(table)
			Expect(ok).To(BeTrue())
			resolvedForm, isStruct := resolved.Form.(resolution.StructForm)
			Expect(isStruct).To(BeTrue())
			Expect(resolvedForm.Fields).To(HaveLen(2))
		})

		It("Should resolve enum references", func(ctx SpecContext) {
			source := `
				TaskState enum {
					pending = 0
					running = 1
				}

				Task struct {
					state TaskState
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.Ok()).To(BeTrue())

			taskType := table.MustGet("task.Task")
			form := taskType.Form.(resolution.StructForm)
			stateField, _ := form.Field("state")
			Expect(stateField.Type.Name).To(Equal("task.TaskState"))

			// Verify it resolves to an enum
			resolved, ok := stateField.Type.Resolve(table)
			Expect(ok).To(BeTrue())
			_, isEnum := resolved.Form.(resolution.EnumForm)
			Expect(isEnum).To(BeTrue())
		})
	})

	Describe("Error Handling", func() {
		It("Should report duplicate struct definitions", func(ctx SpecContext) {
			source := `
				Range struct {}
				Range struct {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag).To(HaveOccurred())
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})

		It("Should report duplicate enum definitions", func(ctx SpecContext) {
			source := `
				State enum {
					a = 0
				}
				State enum {
					b = 0
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag).To(HaveOccurred())
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})
	})

	Describe("DeriveNamespace", func() {
		It("Should extract namespace from file path", func() {
			Expect(analyzer.DeriveNamespace("schema/core/label.oracle")).To(Equal("label"))
			Expect(analyzer.DeriveNamespace("schema/core/label")).To(Equal("label"))
			Expect(analyzer.DeriveNamespace("/path/to/channel.oracle")).To(Equal("channel"))
			Expect(analyzer.DeriveNamespace("ranger")).To(Equal("ranger"))
		})
	})

	Describe("File-level Domain Merging", func() {
		It("Should merge multiple file-level domains with the same name", func(ctx SpecContext) {
			source := `
				@pb output "core/pkg/api/grpc/v1"
				@pb package "api.v1"

				User struct {
					key uuid @key
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.Ok()).To(BeTrue())

			userType := table.MustGet("user.User")
			Expect(userType.Domains).To(HaveKey("pb"))

			pbDomain := userType.Domains["pb"]
			Expect(pbDomain.Expressions).To(HaveLen(2))

			// Both output and package expressions should be present
			outputExpr, found := pbDomain.Expressions.Find("output")
			Expect(found).To(BeTrue())
			Expect(outputExpr.Values[0].StringValue).To(Equal("core/pkg/api/grpc/v1"))

			packageExpr, found := pbDomain.Expressions.Find("package")
			Expect(found).To(BeTrue())
			Expect(packageExpr.Values[0].StringValue).To(Equal("api.v1"))
		})

		It("Should merge file-level domains with struct-level domains", func(ctx SpecContext) {
			source := `
				@go output "core/pkg/service/user"

				User struct {
					key uuid @key
					name string

					@go hand
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.Ok()).To(BeTrue())

			userType := table.MustGet("user.User")
			Expect(userType.Domains).To(HaveKey("go"))

			goDomain := userType.Domains["go"]
			Expect(goDomain.Expressions).To(HaveLen(2))

			// Both file-level output and struct-level omit should be present
			outputExpr, found := goDomain.Expressions.Find("output")
			Expect(found).To(BeTrue())
			Expect(outputExpr.Values[0].StringValue).To(Equal("core/pkg/service/user"))

			_, found = goDomain.Expressions.Find("hand")
			Expect(found).To(BeTrue())
		})

		It("Should let struct-level domain override file-level domain expression", func(ctx SpecContext) {
			source := `
				@ts output "client/ts/src/default"

				User struct {
					key uuid @key
					@ts output "client/ts/src/user"
				}

				Admin struct {
					key uuid @key
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.Ok()).To(BeTrue())

			// User should have overridden output
			userType := table.MustGet("user.User")
			userTsDomain := userType.Domains["ts"]
			userOutput, _ := userTsDomain.Expressions.Find("output")
			Expect(userOutput.Values[0].StringValue).To(Equal("client/ts/src/user"))

			// Admin should have file-level output
			adminType := table.MustGet("user.Admin")
			adminTsDomain := adminType.Domains["ts"]
			adminOutput, _ := adminTsDomain.Expressions.Find("output")
			Expect(adminOutput.Values[0].StringValue).To(Equal("client/ts/src/default"))
		})
	})

	Describe("Struct Extension", func() {
		It("Should parse basic struct extension", func(ctx SpecContext) {
			source := `
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
			Expect(table.StructTypes()).To(HaveLen(2))

			childType := table.MustGet("test.Child")
			form := childType.Form.(resolution.StructForm)
			Expect(form.Extends).To(HaveLen(1))
			Expect(form.Extends[0].Name).To(Equal("test.Parent"))

			// Child should have its own field
			Expect(form.Fields).To(HaveLen(1))
			Expect(form.Fields[0].Name).To(Equal("email"))

			// UnifiedFields should include inherited fields
			allFields := resolution.UnifiedFields(childType, table)
			Expect(allFields).To(HaveLen(3))
			fieldNames := []string{allFields[0].Name, allFields[1].Name, allFields[2].Name}
			Expect(fieldNames).To(ContainElements("name", "age", "email"))
		})

		It("Should parse field omission with -fieldName syntax", func(ctx SpecContext) {
			source := `
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

			childType := table.MustGet("test.Child")
			form := childType.Form.(resolution.StructForm)
			Expect(form.OmittedFields).To(HaveLen(1))
			Expect(form.OmittedFields[0]).To(Equal("age"))

			// UnifiedFields should NOT include omitted field
			allFields := resolution.UnifiedFields(childType, table)
			Expect(allFields).To(HaveLen(3))
			fieldNames := make([]string, len(allFields))
			for i, f := range allFields {
				fieldNames[i] = f.Name
			}
			Expect(fieldNames).To(ContainElements("name", "status", "email"))
			Expect(fieldNames).NotTo(ContainElement("age"))
		})

		It("Should handle field override in child struct", func(ctx SpecContext) {
			source := `
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

			childType := table.MustGet("test.Child")
			form := childType.Form.(resolution.StructForm)
			// Child has its own name field that overrides parent
			Expect(form.Fields).To(HaveLen(1))
			Expect(form.Fields[0].Name).To(Equal("name"))
			Expect(form.Fields[0].Optional).To(BeTrue())

			// UnifiedFields should have child's version of name
			allFields := resolution.UnifiedFields(childType, table)
			Expect(allFields).To(HaveLen(2))

			var nameField *resolution.Field
			for i := range allFields {
				if allFields[i].Name == "name" {
					nameField = &allFields[i]
					break
				}
			}
			Expect(nameField).NotTo(BeNil())
			Expect(nameField.Optional).To(BeTrue())
		})

		It("Should inherit type and optionality when an override omits its type", func(ctx SpecContext) {
			source := `
				Parent struct {
					name string?
					age  int32 = 18
				}

				Child struct extends Parent {
					age = 21
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			child := table.MustGet("test.Child")
			fields := resolution.UnifiedFields(child, table)
			age := findField(fields, "age")
			Expect(age.Type.Name).To(Equal("int32"))
			Expect(age.Default).NotTo(BeNil())
			Expect(age.Default.IntValue).To(Equal(int64(21)))
		})

		It("Should inherit the parent default when an override omits it", func(ctx SpecContext) {
			source := `
				Parent struct {
					count int32 = 5
				}

				Child struct extends Parent {
					count @validate required
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			child := table.MustGet("test.Child")
			fields := resolution.UnifiedFields(child, table)
			count := findField(fields, "count")
			Expect(count.Type.Name).To(Equal("int32"))
			Expect(count.Default).NotTo(BeNil())
			Expect(count.Default.IntValue).To(Equal(int64(5)))
			Expect(count.Domains).To(HaveKey("validate"))
		})

		It("Should merge a domain added by a partial override with inherited domains", func(ctx SpecContext) {
			source := `
				Parent struct {
					name string @validate { min_length 1 }
				}

				Child struct extends Parent {
					name @validate required
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			child := table.MustGet("test.Child")
			fields := resolution.UnifiedFields(child, table)
			name := findField(fields, "name")
			Expect(name.Type.Name).To(Equal("string"))
			Expect(domainExprNames(name.Domains["validate"])).To(Equal([]string{"min_length", "required"}))
		})

		It("Should inherit the parent's domains on a bare typeless override", func(ctx SpecContext) {
			source := `
				Key uint32

				Parent struct {
					key Key {
						@doc value "is the unique identifier for the resource."
					}
				}

				Child struct extends Parent {
					key?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			child := table.MustGet("test.Child")
			form := child.Form.(resolution.StructForm)
			key := findField(form.Fields, "key")
			Expect(key.Domains).To(HaveKey("doc"))
			doc := MustBeOk(key.Domains["doc"].Expressions.Find("value"))
			Expect(doc.Values[0].StringValue).To(Equal("is the unique identifier for the resource."))
		})

		It("Should let a typeless override's own domain win over the inherited one", func(ctx SpecContext) {
			source := `
				Key uint32

				Parent struct {
					key Key {
						@doc value "is the unique identifier for the resource."
					}
				}

				Child struct extends Parent {
					key? {
						@doc value "is an optional key; one is assigned if omitted."
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			child := table.MustGet("test.Child")
			form := child.Form.(resolution.StructForm)
			key := findField(form.Fields, "key")
			doc := MustBeOk(key.Domains["doc"].Expressions.Find("value"))
			Expect(doc.Values[0].StringValue).To(Equal("is an optional key; one is assigned if omitted."))
		})

		It("Should remove an inherited domain with -@domain", func(ctx SpecContext) {
			source := `
				Parent struct {
					name string @validate required
				}

				Child struct extends Parent {
					name -@validate
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			child := table.MustGet("test.Child")
			fields := resolution.UnifiedFields(child, table)
			name := findField(fields, "name")
			Expect(name.Type.Name).To(Equal("string"))
			Expect(name.Domains).NotTo(HaveKey("validate"))
		})

		DescribeTable("Should resolve optionality on a typeless override",
			func(ctx SpecContext, parentField, childField, field, wantType string, wantOptional bool) {
				source := "Key uint32\n\nParent struct {\n  " + parentField +
					"\n}\n\nChild struct extends Parent {\n  " + childField + "\n}\n"
				table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
				Expect(diag.Ok()).To(BeTrue())
				f := findField(resolution.UnifiedFields(table.MustGet("test.Child"), table), field)
				Expect(f.Type.Name).To(Equal(wantType))
				Expect(f.Optional).To(Equal(wantOptional))
			},
			Entry("? makes a required field optional",
				"key Key", "key?", "key", "test.Key", true),
			Entry("? makes a required array field optional",
				"items string[]", "items?", "items", "Array", true),
			Entry("restating the type makes an optional field required",
				"name string?", "name string", "name", "string", false),
		)

		DescribeTable("Should reject the removed ?? optionality marker as a syntax error",
			func(ctx SpecContext, source string) {
				_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
				Expect(diag.Ok()).To(BeFalse())
			},
			Entry("?? on a typed field",
				"S struct {\n  note string??\n}\n"),
			Entry("?? on a typeless override",
				"Parent struct {\n  note string\n}\n\nChild struct extends Parent {\n  note??\n}\n"),
		)

		DescribeTable("Should reject partial-override syntax that cannot resolve",
			func(ctx SpecContext, source, wantErr string) {
				_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
				Expect(diag.Ok()).To(BeFalse())
				Expect(diag.Error()).To(ContainSubstring(wantErr))
			},
			Entry("a typeless field overriding no parent field", `
				Parent struct {
					name string
				}

				Child struct extends Parent {
					age = 21
				}
			`, "declares no type"),
			Entry("a typeless field in a struct that extends nothing", `
				Plain struct {
					name = "x"
				}
			`, "declares no type"),
			Entry("a -@domain removal overriding no parent field", `
				Parent struct {
					name string
				}

				Child struct extends Parent {
					age int32 -@validate
				}
			`, "removes a domain with -@"),
			Entry("a typeless field in an action", `
				Counter struct {
					key uuid

					action SetValue {
						value
					}
				}
			`, "must declare a type"),
		)

		It("Should extend generic struct with type arguments", func(ctx SpecContext) {
			source := `
				Status struct<D extends record> {
					variant int32
					data D
				}

				Details struct {
					message string
				}

				RackStatus struct extends Status<Details> {
					timestamp timestamp
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			rackStatusType := table.MustGet("test.RackStatus")
			form := rackStatusType.Form.(resolution.StructForm)
			Expect(form.Extends).To(HaveLen(1))
			Expect(form.Extends[0].TypeArgs).To(HaveLen(1))
			Expect(form.Extends[0].TypeArgs[0].Name).To(Equal("test.Details"))

			// UnifiedFields should substitute type parameters
			allFields := resolution.UnifiedFields(rackStatusType, table)
			Expect(allFields).To(HaveLen(3))

			var dataField *resolution.Field
			for i := range allFields {
				if allFields[i].Name == "data" {
					dataField = &allFields[i]
					break
				}
			}
			Expect(dataField).NotTo(BeNil())
			Expect(dataField.Type.Name).To(Equal("test.Details"))
		})

		It("Should handle multi-level inheritance", func(ctx SpecContext) {
			source := `
				GrandParent struct {
					a string
				}

				Parent struct extends GrandParent {
					b string
				}

				Child struct extends Parent {
					c string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			childType := table.MustGet("test.Child")
			// UnifiedFields should include fields from all ancestors
			allFields := resolution.UnifiedFields(childType, table)
			Expect(allFields).To(HaveLen(3))
			fieldNames := []string{allFields[0].Name, allFields[1].Name, allFields[2].Name}
			Expect(fieldNames).To(ContainElements("a", "b", "c"))
		})

		It("Should extend struct from imported file", func(ctx SpecContext) {
			loader.Files["schema/core/base"] = `
				Base struct {
					key uuid @id
					name string
				}
			`

			source := `
				import "schema/core/base"

				Extended struct extends base.Base {
					description string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			extendedType := table.MustGet("test.Extended")
			form := extendedType.Form.(resolution.StructForm)
			Expect(form.Extends).To(HaveLen(1))
			Expect(form.Extends[0].Name).To(Equal("base.Base"))

			allFields := resolution.UnifiedFields(extendedType, table)
			Expect(allFields).To(HaveLen(3))
		})

		It("Should detect circular inheritance", func(ctx SpecContext) {
			source := `
				A struct extends B {
					a string
				}

				B struct extends A {
					b string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})

		It("Should detect self-extension", func(ctx SpecContext) {
			source := `
				Self struct extends Self {
					a string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})

		It("Should error on non-existent parent struct", func(ctx SpecContext) {
			source := `
				Child struct extends NonExistent {
					a string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})

		It("Should error on omitting non-existent field", func(ctx SpecContext) {
			source := `
				Parent struct {
					name string
				}

				Child struct extends Parent {
					-nonexistent
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})

		It("Should inherit parent field domains on override", func(ctx SpecContext) {
			source := `
				Parent struct {
					key uuid @id
					name string @validate required
				}

				Child struct extends Parent {
					key uuid?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			childType := table.MustGet("test.Child")
			allFields := resolution.UnifiedFields(childType, table)

			var keyField *resolution.Field
			for i := range allFields {
				if allFields[i].Name == "key" {
					keyField = &allFields[i]
					break
				}
			}
			Expect(keyField).NotTo(BeNil())
			Expect(keyField.Optional).To(BeTrue())     // Child's type
			Expect(keyField.Domains).To(HaveKey("id")) // Parent's domain inherited
		})

		It("Should allow child to override parent domain", func(ctx SpecContext) {
			source := `
				Parent struct {
					name string {
						@validate {
							min_length 1
							max_length 100
						}
					}
				}

				Child struct extends Parent {
					name string {
						@validate {
							min_length 5
							max_length 50
						}
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			childType := table.MustGet("test.Child")
			allFields := resolution.UnifiedFields(childType, table)

			var nameField *resolution.Field
			for i := range allFields {
				if allFields[i].Name == "name" {
					nameField = &allFields[i]
					break
				}
			}
			Expect(nameField).NotTo(BeNil())
			Expect(nameField.Domains).To(HaveKey("validate"))
			validateDomain := nameField.Domains["validate"]
			Expect(validateDomain.Expressions).To(HaveLen(2))
			minLengthExpr, _ := validateDomain.Expressions.Find("min_length")
			Expect(minLengthExpr.Values[0].IntValue).To(Equal(int64(5)))
		})

		It("Should merge domains from parent when child adds new domain", func(ctx SpecContext) {
			source := `
				Parent struct {
					key uuid @id
				}

				Child struct extends Parent {
					key uuid? @validate required
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			childType := table.MustGet("test.Child")
			allFields := resolution.UnifiedFields(childType, table)

			var keyField *resolution.Field
			for i := range allFields {
				if allFields[i].Name == "key" {
					keyField = &allFields[i]
					break
				}
			}
			Expect(keyField).NotTo(BeNil())
			Expect(keyField.Optional).To(BeTrue())
			Expect(keyField.Domains).To(HaveKey("id"))       // Inherited from parent
			Expect(keyField.Domains).To(HaveKey("validate")) // Added by child
		})

		It("Should merge expressions within same domain from parent and child", func(ctx SpecContext) {
			source := `
				Parent struct {
					name string @validate min_length 1
				}

				Child struct extends Parent {
					name string @validate max_length 100
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			childType := table.MustGet("test.Child")
			allFields := resolution.UnifiedFields(childType, table)

			var nameField *resolution.Field
			for i := range allFields {
				if allFields[i].Name == "name" {
					nameField = &allFields[i]
					break
				}
			}
			Expect(nameField).NotTo(BeNil())
			validateDomain := nameField.Domains["validate"]
			Expect(validateDomain.Expressions).To(HaveLen(2)) // Both min_length and max_length
			exprMap := make(map[string]*resolution.Expression)
			for i := range validateDomain.Expressions {
				expr := &validateDomain.Expressions[i]
				exprMap[expr.Name] = expr
			}
			Expect(exprMap).To(HaveKey("min_length")) // From parent
			Expect(exprMap).To(HaveKey("max_length")) // From child
			Expect(exprMap["min_length"].Values[0].IntValue).To(Equal(int64(1)))
			Expect(exprMap["max_length"].Values[0].IntValue).To(Equal(int64(100)))
		})

		// Multiple inheritance tests
		It("Should parse multiple extends with comma-separated parents", func(ctx SpecContext) {
			source := `
				A struct { a string }
				B struct { b string }
				C struct extends A, B { c string }
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			cType := table.MustGet("test.C")
			form := cType.Form.(resolution.StructForm)
			Expect(form.Extends).To(HaveLen(2))
			Expect(form.Extends[0].Name).To(Equal("test.A"))
			Expect(form.Extends[1].Name).To(Equal("test.B"))

			allFields := resolution.UnifiedFields(cType, table)
			Expect(allFields).To(HaveLen(3))
			fieldNames := []string{allFields[0].Name, allFields[1].Name, allFields[2].Name}
			Expect(fieldNames).To(ContainElements("a", "b", "c"))
		})

		It("Should use first parent's field when names conflict", func(ctx SpecContext) {
			source := `
				A struct { shared int32 }
				B struct { shared string }
				C struct extends A, B { }
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			cType := table.MustGet("test.C")
			allFields := resolution.UnifiedFields(cType, table)
			Expect(allFields).To(HaveLen(1))
			Expect(allFields[0].Type.Name).To(Equal("int32")) // From A (first parent)
		})

		It("Should handle diamond inheritance", func(ctx SpecContext) {
			source := `
				Base struct { base string }
				Left struct extends Base { left string }
				Right struct extends Base { right string }
				Diamond struct extends Left, Right { }
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			dType := table.MustGet("test.Diamond")
			allFields := resolution.UnifiedFields(dType, table)
			// base appears once (from Left path), left, right
			Expect(allFields).To(HaveLen(3))
			fieldNames := make([]string, len(allFields))
			for i, f := range allFields {
				fieldNames[i] = f.Name
			}
			Expect(fieldNames).To(ContainElements("base", "left", "right"))
		})

		It("Should detect circular inheritance with multiple parents", func(ctx SpecContext) {
			source := `
				A struct extends C { a string }
				B struct { b string }
				C struct extends A, B { c string }
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
		})

		It("Should handle type parameters with multiple extends", func(ctx SpecContext) {
			source := `
				Generic1 struct<T> { value1 T }
				Generic2 struct<U> { value2 U }
				Combined struct<V> extends Generic1<V>, Generic2<string> {
					combined V
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			cType := table.MustGet("test.Combined")
			allFields := resolution.UnifiedFields(cType, table)
			Expect(allFields).To(HaveLen(3))
		})

		It("Should allow omitting fields from any parent", func(ctx SpecContext) {
			source := `
				A struct {
					a string
					shared string
				}
				B struct { b string }
				C struct extends A, B {
					-shared
					c string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			cType := table.MustGet("test.C")
			allFields := resolution.UnifiedFields(cType, table)
			fieldNames := make([]string, len(allFields))
			for i, f := range allFields {
				fieldNames[i] = f.Name
			}
			Expect(fieldNames).NotTo(ContainElement("shared"))
			Expect(fieldNames).To(ContainElements("a", "b", "c"))
		})

		It("Should error when omitting field not in any parent", func(ctx SpecContext) {
			source := `
				A struct { a string }
				B struct { b string }
				C struct extends A, B {
					-nonexistent
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
		})
	})

	Describe("Action Extension", func() {
		findAction := func(table *resolution.Table, qname, name string) resolution.Action {
			form := table.MustGet(qname).Form.(resolution.StructForm)
			for _, a := range form.Actions {
				if a.Name == name {
					return a
				}
			}
			Fail("action not found: " + name)
			return resolution.Action{}
		}

		It("Should flatten an extended struct's fields into the action payload", func(ctx SpecContext) {
			source := `
				Named struct {
					name string
				}

				Schematic struct {
					key uuid

					action Rename extends Named {
						@doc value "renames the schematic"
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
			action := findAction(table, "test.Schematic", "Rename")
			Expect(action.Fields).To(HaveLen(1))
			Expect(action.Fields[0].Name).To(Equal("name"))
			Expect(action.Fields[0].Type.Name).To(Equal("string"))
		})

		It("Should prepend inherited fields before the action's own fields", func(ctx SpecContext) {
			source := `
				NodeRef struct {
					key      string
					position int32
				}

				Schematic struct {
					id uuid

					action SetNodePosition extends NodeRef {
						animate int32
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
			action := findAction(table, "test.Schematic", "SetNodePosition")
			names := make([]string, len(action.Fields))
			for i, f := range action.Fields {
				names[i] = f.Name
			}
			Expect(names).To(Equal([]string{"key", "position", "animate"}))
		})

		It("Should let an action's own field override an inherited field", func(ctx SpecContext) {
			source := `
				Base struct {
					value int32
				}

				Container struct {
					key uuid

					action SetValue extends Base {
						value string
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
			action := findAction(table, "test.Container", "SetValue")
			Expect(action.Fields).To(HaveLen(1))
			Expect(findField(action.Fields, "value").Type.Name).To(Equal("string"))
		})

		It("Should flatten fields from multiple extended structs (first wins)", func(ctx SpecContext) {
			source := `
				A struct {
					a string
				}
				B struct {
					a int32
					b string
				}

				Container struct {
					key uuid

					action Combine extends A, B {}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
			action := findAction(table, "test.Container", "Combine")
			names := make([]string, len(action.Fields))
			for i, f := range action.Fields {
				names[i] = f.Name
			}
			Expect(names).To(Equal([]string{"a", "b"}))
			Expect(findField(action.Fields, "a").Type.Name).To(Equal("string"))
		})

		It("Should inherit transitively through the extended struct's own parents", func(ctx SpecContext) {
			source := `
				GrandParent struct { a string }
				Parent struct extends GrandParent { b string }

				Container struct {
					key uuid

					action Apply extends Parent {
						c string
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
			action := findAction(table, "test.Container", "Apply")
			names := make([]string, len(action.Fields))
			for i, f := range action.Fields {
				names[i] = f.Name
			}
			Expect(names).To(Equal([]string{"a", "b", "c"}))
		})

		It("Should substitute type arguments when extending a generic struct", func(ctx SpecContext) {
			source := `
				Box struct<T extends record> {
					data T
				}

				Details struct {
					message string
				}

				Container struct {
					key uuid

					action Load extends Box<Details> {}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
			action := findAction(table, "test.Container", "Load")
			Expect(action.Fields).To(HaveLen(1))
			Expect(action.Fields[0].Name).To(Equal("data"))
			Expect(action.Fields[0].Type.Name).To(Equal("test.Details"))
		})

		It("Should error when an action extends an unresolved type", func(ctx SpecContext) {
			source := `
				Container struct {
					key uuid

					action Apply extends Missing {}
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("extends unresolved type"))
		})

		It("Should error when an action extends a non-struct type", func(ctx SpecContext) {
			source := `
				Color enum {
					red = 0
					green = 1
				}

				Container struct {
					key uuid

					action Apply extends Color {}
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("extends non-struct type"))
		})

		It("Should error when extending a generic struct without type arguments", func(ctx SpecContext) {
			source := `
				Box struct<T extends record> {
					data T
				}

				Container struct {
					key uuid

					action Load extends Box {}
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("type arguments"))
		})
	})

	Describe("TypeDef", func() {
		It("Should analyze a distinct type (primitive alias)", func(ctx SpecContext) {
			// Grammar: IDENT qualifiedIdent (no 'type' keyword or '=')
			source := `
				ChannelKey uint32
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "channel", loader)
			Expect(diag.Ok()).To(BeTrue())

			channelKeyType := table.MustGet("channel.ChannelKey")
			form, ok := channelKeyType.Form.(resolution.DistinctForm)
			Expect(ok).To(BeTrue())
			Expect(form.Base.Name).To(Equal("uint32"))
		})

		It("Should analyze a struct alias", func(ctx SpecContext) {
			source := `
				Position struct {
					x float64
					y float64
				}

				Point = Position
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "geo", loader)
			Expect(diag.Ok()).To(BeTrue())

			pointType := table.MustGet("geo.Point")
			form, ok := pointType.Form.(resolution.AliasForm)
			Expect(ok).To(BeTrue())
			Expect(form.Target.Name).To(Equal("geo.Position"))
		})

		It("Should analyze an array type definition", func(ctx SpecContext) {
			source := `
				Param struct {
					name string
					value record?
				}

				Params Param[]
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ir", loader)
			Expect(diag.Ok()).To(BeTrue())

			paramsType := table.MustGet("ir.Params")
			form, ok := paramsType.Form.(resolution.DistinctForm)
			Expect(ok).To(BeTrue())
			Expect(form.Base.Name).To(Equal("Array"))
			Expect(form.Base.TypeArgs).To(HaveLen(1))
			Expect(form.Base.TypeArgs[0].Name).To(Equal("ir.Param"))
		})

		It("Should analyze an alias to an array type", func(ctx SpecContext) {
			source := `
				Stratum = string[]
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ir", loader)
			Expect(diag.Ok()).To(BeTrue())

			stratumType := table.MustGet("ir.Stratum")
			form, ok := stratumType.Form.(resolution.AliasForm)
			Expect(ok).To(BeTrue())
			// The target should be Array with type arg string
			Expect(form.Target.Name).To(Equal("Array"))
			Expect(form.Target.TypeArgs).To(HaveLen(1))
			Expect(form.Target.TypeArgs[0].Name).To(Equal("string"))
		})
	})

	Describe("Generics", func() {
		It("Should parse generic struct with type parameter", func(ctx SpecContext) {
			source := `
				Container struct<T> {
					value T
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			containerType := table.MustGet("test.Container")
			form := containerType.Form.(resolution.StructForm)
			Expect(form.IsGeneric()).To(BeTrue())
			Expect(form.TypeParams).To(HaveLen(1))
			Expect(form.TypeParams[0].Name).To(Equal("T"))

			valueField, _ := form.Field("value")
			Expect(valueField.Type.IsTypeParam()).To(BeTrue())
			Expect(valueField.Type.TypeParam.Name).To(Equal("T"))
		})

		It("Should parse generic struct with constrained type parameter", func(ctx SpecContext) {
			source := `
				NumberContainer struct<T extends int32> {
					value T
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			containerType := table.MustGet("test.NumberContainer")
			form := containerType.Form.(resolution.StructForm)
			Expect(form.TypeParams[0].Constraint).NotTo(BeNil())
			Expect(form.TypeParams[0].Constraint.Name).To(Equal("int32"))
		})

		It("Should resolve comparable constraint without warnings", func(ctx SpecContext) {
			source := `
				State struct<R extends comparable> {
					resource R
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			stateType := table.MustGet("test.State")
			form := stateType.Form.(resolution.StructForm)
			Expect(form.TypeParams[0].Constraint).NotTo(BeNil())
			Expect(form.TypeParams[0].Constraint.Name).To(Equal("comparable"))
		})

		It("Should resolve numeric constraint with numeric default without warnings", func(ctx SpecContext) {
			source := `
				Bounds struct<T extends numeric = float64> {
					lower T
					upper T
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			boundsType := table.MustGet("test.Bounds")
			form := boundsType.Form.(resolution.StructForm)
			Expect(form.TypeParams[0].Constraint).NotTo(BeNil())
			Expect(form.TypeParams[0].Constraint.Name).To(Equal("numeric"))
			Expect(form.TypeParams[0].Default).NotTo(BeNil())
			Expect(form.TypeParams[0].Default.Name).To(Equal("float64"))
		})

		It("Should reject numeric constraint without a default", func(ctx SpecContext) {
			source := `
				Bounds struct<T extends numeric> {
					lower T
					upper T
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("requires a default"))
		})

		It("Should reject numeric constraint with non-numeric default", func(ctx SpecContext) {
			source := `
				Bounds struct<T extends numeric = string> {
					lower T
					upper T
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("non-numeric default"))
		})

		It("Should parse generic struct with default type parameter", func(ctx SpecContext) {
			source := `
				Container struct<T = string> {
					value T
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			containerType := table.MustGet("test.Container")
			form := containerType.Form.(resolution.StructForm)
			Expect(form.TypeParams[0].Default).NotTo(BeNil())
			Expect(form.TypeParams[0].Default.Name).To(Equal("string"))
		})

		It("Should parse struct with generic field type", func(ctx SpecContext) {
			source := `
				Container struct<T> {
					value T
				}

				Wrapper struct {
					container Container<string>
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			wrapperType := table.MustGet("test.Wrapper")
			form := wrapperType.Form.(resolution.StructForm)
			containerField, _ := form.Field("container")
			Expect(containerField.Type.Name).To(Equal("test.Container"))
			Expect(containerField.Type.TypeArgs).To(HaveLen(1))
			Expect(containerField.Type.TypeArgs[0].Name).To(Equal("string"))
		})

		It("Should preserve type params on fields with constraints and defaults", func(ctx SpecContext) {
			source := `
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

			taskType := table.MustGet("test.Task")
			form := taskType.Form.(resolution.StructForm)

			// Verify type params are set up correctly
			Expect(form.IsGeneric()).To(BeTrue())
			Expect(form.TypeParams).To(HaveLen(2))
			Expect(form.TypeParams[0].Name).To(Equal("Type"))
			Expect(form.TypeParams[0].Constraint).NotTo(BeNil())
			Expect(form.TypeParams[0].Constraint.Name).To(Equal("string"))
			Expect(form.TypeParams[0].Default).NotTo(BeNil())
			Expect(form.TypeParams[0].Default.Name).To(Equal("string"))

			// Verify 'name' field is NOT a type param
			nameField, found := form.Field("name")
			Expect(found).To(BeTrue())
			Expect(nameField.Type.IsTypeParam()).To(BeFalse())
			Expect(nameField.Type.Name).To(Equal("string"))

			// Verify 'type' field IS a type param reference
			typeField, found := form.Field("type")
			Expect(found).To(BeTrue())
			Expect(typeField.Type.IsTypeParam()).To(BeTrue(), "type field should be a type param")
			Expect(typeField.Type.TypeParam).NotTo(BeNil(), "type field TypeParam should not be nil")
			Expect(typeField.Type.TypeParam.Name).To(Equal("Type"))
			Expect(typeField.Type.TypeParam.Constraint).NotTo(BeNil())
			Expect(typeField.Type.TypeParam.Constraint.Name).To(Equal("string"))

			// Verify 'config' field IS a type param reference
			configField, found := form.Field("config")
			Expect(found).To(BeTrue())
			Expect(configField.Type.IsTypeParam()).To(BeTrue(), "config field should be a type param")
			Expect(configField.Type.TypeParam).NotTo(BeNil(), "config field TypeParam should not be nil")
			Expect(configField.Type.TypeParam.Name).To(Equal("Config"))
			Expect(configField.Type.TypeParam.Constraint).NotTo(BeNil())
			Expect(configField.Type.TypeParam.Constraint.Name).To(Equal("record"))
		})

		It("Should preserve type params in UnifiedFields for generic structs", func(ctx SpecContext) {
			source := `
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

			taskType := table.MustGet("test.Task")

			// Test UnifiedFields - this is what the TS plugin uses
			unifiedFields := resolution.UnifiedFields(taskType, table)
			Expect(unifiedFields).To(HaveLen(3))

			// Find the type field in unified fields
			var typeField, configField resolution.Field
			for _, f := range unifiedFields {
				if f.Name == "type" {
					typeField = f
				}
				if f.Name == "config" {
					configField = f
				}
			}

			// Verify 'type' field preserves TypeParam through UnifiedFields
			Expect(typeField.Type.IsTypeParam()).To(BeTrue(), "type field should be a type param after UnifiedFields")
			Expect(typeField.Type.TypeParam).NotTo(BeNil(), "type field TypeParam should not be nil after UnifiedFields")
			Expect(typeField.Type.TypeParam.Name).To(Equal("Type"))

			// Verify 'config' field preserves TypeParam through UnifiedFields
			Expect(configField.Type.IsTypeParam()).To(BeTrue(), "config field should be a type param after UnifiedFields")
			Expect(configField.Type.TypeParam).NotTo(BeNil(), "config field TypeParam should not be nil after UnifiedFields")
			Expect(configField.Type.TypeParam.Name).To(Equal("Config"))
		})
	})

	Describe("Map Types", func() {
		It("Should parse map type", func(ctx SpecContext) {
			source := `
				Config struct {
					settings Map<string, record>
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			configType := table.MustGet("test.Config")
			form := configType.Form.(resolution.StructForm)
			settingsField, _ := form.Field("settings")
			Expect(settingsField.Type.Name).To(Equal("Map"))
			Expect(settingsField.Type.TypeArgs).To(HaveLen(2))
			Expect(settingsField.Type.TypeArgs[0].Name).To(Equal("string"))
			Expect(settingsField.Type.TypeArgs[1].Name).To(Equal("record"))
		})
	})

	Describe("Recursive Types", func() {
		It("Should detect recursive struct", func(ctx SpecContext) {
			source := `
				Node struct {
					value string
					children Node[]
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			nodeType := table.MustGet("test.Node")
			form := nodeType.Form.(resolution.StructForm)
			Expect(form.IsRecursive).To(BeTrue())
		})

		It("Should detect non-recursive struct", func(ctx SpecContext) {
			source := `
				Simple struct {
					value string
					count int32
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			simpleType := table.MustGet("test.Simple")
			form := simpleType.Form.(resolution.StructForm)
			Expect(form.IsRecursive).To(BeFalse())
		})

		It("Should detect mutual recursion through struct fields", func(ctx SpecContext) {
			source := `
				A struct {
					b B?
				}
				B struct {
					a A?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			aForm := table.MustGet("test.A").Form.(resolution.StructForm)
			bForm := table.MustGet("test.B").Form.(resolution.StructForm)
			Expect(aForm.IsRecursive).To(BeTrue())
			Expect(bForm.IsRecursive).To(BeTrue())
		})

		It("Should detect mutual recursion through a distinct array wrapper", func(ctx SpecContext) {
			source := `
				A struct {
					bs Bs
				}
				B struct {
					a A?
				}
				Bs B[]
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			aForm := table.MustGet("test.A").Form.(resolution.StructForm)
			bForm := table.MustGet("test.B").Form.(resolution.StructForm)
			Expect(aForm.IsRecursive).To(BeTrue())
			Expect(bForm.IsRecursive).To(BeTrue())
		})

		It("Should detect mutual recursion through an alias wrapper", func(ctx SpecContext) {
			source := `
				A struct {
					bs Bs
				}
				B struct {
					a A?
				}
				Bs = B[]
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			aForm := table.MustGet("test.A").Form.(resolution.StructForm)
			bForm := table.MustGet("test.B").Form.(resolution.StructForm)
			Expect(aForm.IsRecursive).To(BeTrue())
			Expect(bForm.IsRecursive).To(BeTrue())
		})

		It("Should detect mutual recursion through a distinct struct wrapper", func(ctx SpecContext) {
			source := `
				A struct {
					b BWrap?
				}
				B struct {
					a A?
				}
				BWrap B
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			aForm := table.MustGet("test.A").Form.(resolution.StructForm)
			bForm := table.MustGet("test.B").Form.(resolution.StructForm)
			Expect(aForm.IsRecursive).To(BeTrue())
			Expect(bForm.IsRecursive).To(BeTrue())
		})
	})

	Describe("Field Defaults", func() {
		defaultOf := func(ctx SpecContext, fieldDecl string) *resolution.ExpressionValue {
			source := "Item struct {\n\t" + fieldDecl + "\n}\n"
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
			form := table.MustGet("test.Item").Form.(resolution.StructForm)
			Expect(form.Fields).To(HaveLen(1))
			return form.Fields[0].Default
		}

		DescribeTable("Should populate Field.Default from an inline = value",
			func(ctx SpecContext, fieldDecl string, expected resolution.ExpressionValue) {
				def := defaultOf(ctx, fieldDecl)
				Expect(def).NotTo(BeNil())
				Expect(*def).To(Equal(expected))
			},
			Entry("int", "count int32 = 5",
				resolution.ExpressionValue{Kind: resolution.ValueKindInt, IntValue: 5}),
			Entry("float", "ratio float64 = 1.5",
				resolution.ExpressionValue{Kind: resolution.ValueKindFloat, FloatValue: 1.5}),
			Entry("string", "name string = \"untitled\"",
				resolution.ExpressionValue{Kind: resolution.ValueKindString, StringValue: "untitled"}),
			Entry("bool", "active bool = false",
				resolution.ExpressionValue{Kind: resolution.ValueKindBool, BoolValue: false}),
			Entry("ident", "key string = create",
				resolution.ExpressionValue{Kind: resolution.ValueKindIdent, IdentValue: "create"}),
			Entry("qualified ident", "mode string = control.Exclusive",
				resolution.ExpressionValue{Kind: resolution.ValueKindIdent, IdentValue: "control.Exclusive"}),
		)

		It("Should reject a field that is both optional and defaulted", func(ctx SpecContext) {
			source := "Item struct {\n\tname string? = \"\"\n}\n"
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Error()).To(ContainSubstring("both nullable"))
		})

		It("Should leave Default nil when no default is declared", func(ctx SpecContext) {
			Expect(defaultOf(ctx, "count int32")).To(BeNil())
		})

		It("Should accept a default alongside a field body", func(ctx SpecContext) {
			def := defaultOf(ctx, "count int32 = 7 {\n\t\t@doc value \"is a counter.\"\n\t}")
			Expect(def).NotTo(BeNil())
			Expect(def.Kind).To(Equal(resolution.ValueKindInt))
			Expect(def.IntValue).To(Equal(int64(7)))
		})

		It("Should collect an empty array default", func(ctx SpecContext) {
			def := defaultOf(ctx, "vals float64[] = []")
			Expect(def).NotTo(BeNil())
			Expect(def.Kind).To(Equal(resolution.ValueKindArray))
			Expect(def.Elements).To(BeEmpty())
		})

		It("Should collect a populated array default with element values", func(ctx SpecContext) {
			def := defaultOf(ctx, "vals float64[] = [1.5, 2.5]")
			Expect(def).NotTo(BeNil())
			Expect(def.Kind).To(Equal(resolution.ValueKindArray))
			Expect(def.Elements).To(HaveLen(2))
			Expect(def.Elements[0]).To(Equal(resolution.ExpressionValue{Kind: resolution.ValueKindFloat, FloatValue: 1.5}))
			Expect(def.Elements[1]).To(Equal(resolution.ExpressionValue{Kind: resolution.ValueKindFloat, FloatValue: 2.5}))
		})
	})

	Describe("Union Definitions", func() {
		It("Should allow 'on' as a field name since it is not a reserved keyword", func(ctx SpecContext) {
			source := `
				Handle struct { node string }
				Transition struct {
					on     Handle
					target string
				}

				LinearScale struct { slope float64 }
				NoneScale struct {}
				Scale union on type {
					linear LinearScale
					none   NoneScale
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "arc", loader)
			Expect(diag.Ok()).To(BeTrue())
			tr := table.MustGet("arc.Transition")
			form := tr.Form.(resolution.StructForm)
			names := make([]string, len(form.Fields))
			for i, f := range form.Fields {
				names[i] = f.Name
			}
			Expect(names).To(ContainElement("on"))
			scale := table.MustGet("arc.Scale")
			Expect(scale.Form.(resolution.UnionForm).Discriminator).To(Equal("type"))
		})

		It("Should collect a simple union with primitive-only variants", func(ctx SpecContext) {
			source := `
				LinearScale struct {
					slope float64
					yIntercept float64
				}
				MapScale struct {
					preScaledMin float64
					scaledMin float64
				}
				NoneScale struct {}

				Scale union on type {
					linear LinearScale
					map    MapScale
					none   NoneScale
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ni", loader)
			Expect(diag.Ok()).To(BeTrue())
			Expect(table.UnionTypes()).To(HaveLen(1))

			scale := table.MustGet("ni.Scale")
			Expect(scale.Name).To(Equal("Scale"))
			form := scale.Form.(resolution.UnionForm)
			Expect(form.Discriminator).To(Equal("type"))
			Expect(form.Variants).To(HaveLen(3))
			Expect(form.Extends).To(BeEmpty())

			Expect(form.Variants[0].Name).To(Equal("linear"))
			Expect(form.Variants[0].Type.Name).To(Equal("ni.LinearScale"))
			Expect(form.Variants[1].Name).To(Equal("map"))
			Expect(form.Variants[2].Name).To(Equal("none"))
		})

		It("Should expand a union extending other unions into their variant union", func(ctx SpecContext) {
			source := `
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
			table, diag := analyzer.AnalyzeSource(ctx, source, "schematic", loader)
			Expect(diag.Ok()).To(BeTrue())

			el := table.MustGet("schematic.ElementConfig")
			form := el.Form.(resolution.UnionForm)
			Expect(form.Extends).To(BeEmpty())
			Expect(form.Variants).To(HaveLen(2))
			Expect(form.Variants[0].Name).To(Equal("tank"))
			Expect(form.Variants[0].Type.Name).To(Equal("schematic.TankConfig"))
			Expect(form.Variants[1].Name).To(Equal("pipe"))

			node := table.MustGet("schematic.NodeConfig")
			Expect(node.Form.(resolution.UnionForm).Variants).To(HaveLen(1))
		})

		It("Should allow an extending union to declare additional variants", func(ctx SpecContext) {
			source := `
				TankConfig struct {}
				GroupConfig struct {}

				NodeConfig union on variant {
					tank TankConfig
				}

				ElementConfig union on variant extends NodeConfig {
					group GroupConfig
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "schematic", loader)
			Expect(diag.Ok()).To(BeTrue())
			form := table.MustGet("schematic.ElementConfig").Form.(resolution.UnionForm)
			Expect(form.Variants).To(HaveLen(2))
			Expect(form.Variants[1].Name).To(Equal("group"))
		})

		It("Should synthesize suppressed payload types for inline variants", func(ctx SpecContext) {
			source := `
				TabBase struct { key string }
				Labeled struct { label string }

				Tab union on variant extends TabBase {
					resource {
						resource string
					}
					view extends Labeled {
						type string
						args string?

						@doc value "is an inline view tab."
					}
					empty {}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "panel", loader)
			Expect(diag.Ok()).To(BeTrue())

			form := table.MustGet("panel.Tab").Form.(resolution.UnionForm)
			Expect(form.Variants).To(HaveLen(3))
			Expect(form.Variants[0].Inline).To(BeTrue())
			Expect(form.Variants[1].Inline).To(BeTrue())
			Expect(form.Variants[1].Domains).To(HaveKey("doc"))

			view := table.MustGet("panel.TabViewPayload")
			Expect(view.Synthetic).To(BeTrue())
			viewForm := view.Form.(resolution.StructForm)
			Expect(viewForm.Extends).To(HaveLen(1))
			Expect(viewForm.Fields).To(HaveLen(2))
			Expect(viewForm.Fields[0].Name).To(Equal("type"))

			fields := resolution.UnifiedVariantFields(
				table.MustGet("panel.Tab"), form.Variants[1], table)
			names := make([]string, len(fields))
			for i, f := range fields {
				names[i] = f.Name
			}
			Expect(names).To(Equal([]string{"key", "label", "type", "args"}))

			empty := table.MustGet("panel.TabEmptyPayload")
			Expect(empty.Synthetic).To(BeTrue())
			Expect(empty.Form.(resolution.StructForm).Fields).To(BeEmpty())
			Expect(table.StructTypes()).NotTo(ContainElement(
				HaveField("QualifiedName", "panel.TabViewPayload")))
		})

		It("Should reject extended unions with conflicting variant payloads", func(ctx SpecContext) {
			source := `
				TankConfig struct {}
				OtherConfig struct {}

				A union on variant {
					tank TankConfig
				}

				B union on variant {
					tank OtherConfig
				}

				ElementConfig union on variant extends A, B {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "schematic", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(
				ContainSubstring("conflicting payload types from schematic.B (schematic.TankConfig vs schematic.OtherConfig)"),
			)
		})

		It("Should reject extending unions with a different discriminator", func(ctx SpecContext) {
			source := `
				TankConfig struct {}

				A union on kind {
					tank TankConfig
				}

				ElementConfig union on variant extends A {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "schematic", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("different discriminator"))
		})

		It("Should reject mixing struct and union bases in extends", func(ctx SpecContext) {
			source := `
				Base struct { key string }
				TankConfig struct {}

				A union on variant {
					tank TankConfig
				}

				ElementConfig union on variant extends A, Base {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "schematic", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("cannot mix struct and union bases"))
		})

		It("Should reject cyclic union extends chains", func(ctx SpecContext) {
			source := `
				TankConfig struct {}

				A union on variant extends B {
					tank TankConfig
				}

				B union on variant extends A {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "schematic", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("cyclic extends chain"))
		})

		It("Should collect a union with extends (shared base struct)", func(ctx SpecContext) {
			source := `
				BaseAIChannel struct {
					port    int32
					enabled bool
					name    string
				}

				AIVoltageFields struct {
					terminalConfig string
					minVal         float64
					maxVal         float64
				}

				AIAccelFields struct {
					sensitivity float64
				}

				AIChannel union on type extends BaseAIChannel {
					ai_voltage AIVoltageFields
					ai_accel   AIAccelFields
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ni", loader)
			Expect(diag.Ok()).To(BeTrue())

			ch := table.MustGet("ni.AIChannel")
			form := ch.Form.(resolution.UnionForm)
			Expect(form.Extends).To(HaveLen(1))
			Expect(form.Extends[0].Name).To(Equal("ni.BaseAIChannel"))

			voltageFields := resolution.UnifiedVariantFields(ch, form.Variants[0], table)
			fieldNames := make([]string, len(voltageFields))
			for i, f := range voltageFields {
				fieldNames[i] = f.Name
			}
			Expect(fieldNames).To(Equal([]string{
				"port", "enabled", "name",
				"terminalConfig", "minVal", "maxVal",
			}))
		})

		It("Should collect per-variant domains", func(ctx SpecContext) {
			source := `
				LinearScale struct {}
				MapScale struct {}

				Scale union on type {
					linear LinearScale {
						@doc value "linear scaling"
					}
					map MapScale {
						@doc value "piecewise linear map"
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ni", loader)
			Expect(diag.Ok()).To(BeTrue())

			form := table.MustGet("ni.Scale").Form.(resolution.UnionForm)
			Expect(form.Variants[0].Domains).To(HaveKey("doc"))
			Expect(form.Variants[1].Domains).To(HaveKey("doc"))

			linearDoc := form.Variants[0].Domains["doc"]
			Expect(linearDoc.Expressions).To(HaveLen(1))
			Expect(linearDoc.Expressions[0].Name).To(Equal("value"))
			Expect(linearDoc.Expressions[0].Values[0].StringValue).To(Equal("linear scaling"))
		})

		It("Should collect union-level domains", func(ctx SpecContext) {
			source := `
				LinearScale struct {}
				MapScale struct {}

				Scale union on type {
					linear LinearScale
					map MapScale
					@doc value "controls how raw values are transformed"
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ni", loader)
			Expect(diag.Ok()).To(BeTrue())

			scale := table.MustGet("ni.Scale")
			Expect(scale.Domains).To(HaveKey("doc"))
		})

		It("Should support mixin composition in variant structs", func(ctx SpecContext) {
			source := `
				Terminal struct {
					terminalConfig string
				}
				MinMaxVal struct {
					minVal float64
					maxVal float64
				}

				AIVoltageFields struct extends Terminal, MinMaxVal {
					customScale string
				}

				Empty struct {}

				AIChannel union on type {
					ai_voltage AIVoltageFields
					empty      Empty
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ni", loader)
			Expect(diag.Ok()).To(BeTrue())

			ch := table.MustGet("ni.AIChannel")
			form := ch.Form.(resolution.UnionForm)
			fields := resolution.UnifiedVariantFields(ch, form.Variants[0], table)
			names := make([]string, len(fields))
			for i, f := range fields {
				names[i] = f.Name
			}
			Expect(names).To(Equal([]string{
				"terminalConfig", "minVal", "maxVal", "customScale",
			}))
		})

		It("Should support nested unions (variant field typed as a union)", func(ctx SpecContext) {
			source := `
				LinearScale struct { slope float64 }
				NoneScale   struct {}

				Scale union on type {
					linear LinearScale
					none   NoneScale
				}

				AIVoltageFields struct {
					customScale Scale
				}

				Empty struct {}

				AIChannel union on type {
					ai_voltage AIVoltageFields
					empty      Empty
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ni", loader)
			Expect(diag.Ok()).To(BeTrue())

			voltageFields := resolution.UnifiedVariantFields(
				table.MustGet("ni.AIChannel"),
				table.MustGet("ni.AIChannel").Form.(resolution.UnionForm).Variants[0],
				table,
			)
			Expect(voltageFields).To(HaveLen(1))
			Expect(voltageFields[0].Name).To(Equal("customScale"))
			Expect(voltageFields[0].Type.Name).To(Equal("ni.Scale"))
		})

		It("Should error on duplicate variant values", func(ctx SpecContext) {
			source := `
				A struct {}
				B struct {}

				Bad union on type {
					same A
					same B
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring(`duplicate variant value "same"`))
		})

		It("Should error when a variant value collides with the discriminator field name", func(ctx SpecContext) {
			source := `
				A struct {}
				B struct {}

				Bad union on type {
					type A
					linear B
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring(`variant value "type" that collides with the discriminator field name`))
		})

		It("Should error when a base struct declares a field named variant", func(ctx SpecContext) {
			source := `
				Base struct { variant string }
				A struct {}

				Bad union on type extends Base {
					linear A
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring(`declares a field named "variant"`))
		})

		It("Should error when a variant references an unresolved type", func(ctx SpecContext) {
			source := `
				Foo union on type {
					a MissingStruct
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(
				ContainSubstring(`union Foo variant "a" references unresolved type: MissingStruct`),
			)
		})

		It("Should error when a variant references a non-struct type", func(ctx SpecContext) {
			source := `
				Color enum {
					red = "red"
				}

				Foo union on type {
					a Color
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(
				ContainSubstring(`union Foo variant "a" must reference a struct type, got: test.Color`),
			)
		})

		It("Should error when extends targets an unresolved type", func(ctx SpecContext) {
			source := `
				A struct {}

				Foo union on type extends Missing {
					a A
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("union Foo extends unresolved type"))
		})

		It("Should error when extends targets a non-struct type", func(ctx SpecContext) {
			source := `
				Color enum {
					red = "red"
				}

				A struct {}

				Foo union on type extends Color {
					a A
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(
				ContainSubstring("union Foo extends non-struct type at position 1: Color"),
			)
		})

		It("Should error when extending a union that itself extends structs", func(ctx SpecContext) {
			source := `
				Base struct {
					name string
				}

				TankConfig struct {}

				A union on variant extends Base {
					tank TankConfig
				}

				ElementConfig union on variant extends A {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(
				ContainSubstring("union test.ElementConfig cannot extend union test.A, which extends base structs"),
			)
		})

		It("Should error on empty union", func(ctx SpecContext) {
			source := `
				Empty union on type {}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("union Empty has no variants"))
		})

		It("Should error when a variant struct declares the discriminator field", func(ctx SpecContext) {
			source := `
				BadVariant struct {
					kind string
					payload int32
				}
				Other struct {}

				Bad union on kind {
					bad   BadVariant
					other Other
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring(`variant "bad" (test.BadVariant) declares the discriminator field "kind"`))
		})

		It("Should error when a base struct declares the discriminator field", func(ctx SpecContext) {
			source := `
				BadBase struct {
					type string
					port int32
				}
				A struct {}
				B struct {}

				Bad union on type extends BadBase {
					a A
					b B
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring(`base struct declares the discriminator field "type"`))
		})

		It("Should error when a variant references a non-struct type", func(ctx SpecContext) {
			source := `
				Color enum {
					red   = "red"
					green = "green"
				}
				A struct {}

				Bad union on type {
					a     A
					color Color
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring(`variant "color" must reference a struct type`))
		})

		It("Should error when extends targets a non-struct type", func(ctx SpecContext) {
			source := `
				Color enum {
					red   = "red"
					green = "green"
				}
				A struct {}
				B struct {}

				Bad union on type extends Color {
					a A
					b B
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("extends non-struct type"))
		})

		It("Should error on duplicate type definition for a union name", func(ctx SpecContext) {
			source := `
				A struct {}
				Foo struct {}
				Foo union on type {
					a A
				}
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.Error()).To(ContainSubstring("duplicate type definition"))
		})

		It("Should sort unions topologically after their variant and base types", func(ctx SpecContext) {
			source := `
				BaseAIChannel struct { port int32 }
				AIVoltageFields struct { minVal float64 }
				AIAccelFields struct { sensitivity float64 }

				AIChannel union on type extends BaseAIChannel {
					ai_voltage AIVoltageFields
					ai_accel   AIAccelFields
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ni", loader)
			Expect(diag.Ok()).To(BeTrue())

			all := table.TypesInNamespace("ni")
			sorted := table.TopologicalSort(all)
			indexOf := func(qname string) int {
				for i, t := range sorted {
					if t.QualifiedName == qname {
						return i
					}
				}
				return -1
			}
			Expect(indexOf("ni.BaseAIChannel")).To(BeNumerically("<", indexOf("ni.AIChannel")))
			Expect(indexOf("ni.AIVoltageFields")).To(BeNumerically("<", indexOf("ni.AIChannel")))
			Expect(indexOf("ni.AIAccelFields")).To(BeNumerically("<", indexOf("ni.AIChannel")))
		})

		It("Should support unions extending multiple bases", func(ctx SpecContext) {
			source := `
				Ident struct {
					key string
				}
				Audited struct {
					createdAt int64
					updatedAt int64
				}

				A struct {
					aField string
				}
				B struct {
					bField string
				}

				Mixed union on type extends Ident, Audited {
					a A
					b B
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			mixed := table.MustGet("test.Mixed")
			form := mixed.Form.(resolution.UnionForm)
			Expect(form.Extends).To(HaveLen(2))

			aFields := resolution.UnifiedVariantFields(mixed, form.Variants[0], table)
			names := make([]string, len(aFields))
			for i, f := range aFields {
				names[i] = f.Name
			}
			Expect(names).To(Equal([]string{"key", "createdAt", "updatedAt", "aField"}))
		})

		It("UnifiedVariantFields should return nil for a non-union type", func(ctx SpecContext) {
			source := `
				Foo struct { x int32 }
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())

			foo := table.MustGet("test.Foo")
			fakeVariant := resolution.UnionVariant{Name: "x", Type: resolution.TypeRef{Name: "test.Foo"}}
			Expect(resolution.UnifiedVariantFields(foo, fakeVariant, table)).To(BeNil())
		})
	})

	Describe("Struct Defaults", func() {
		structDefaultOf := func(ctx SpecContext, source string) *resolution.ExpressionValue {
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue())
			form := table.MustGet("test.Outer").Form.(resolution.StructForm)
			return form.Fields[0].Default
		}

		It("Should collect an empty struct default", func(ctx SpecContext) {
			def := structDefaultOf(ctx,
				"Point struct {\n\tx int32\n\ty int32\n}\n"+
					"Outer struct {\n\tp Point = {}\n}\n")
			Expect(def).NotTo(BeNil())
			Expect(def.Kind).To(Equal(resolution.ValueKindStruct))
			Expect(def.Fields).To(BeEmpty())
		})

		It("Should collect a populated struct default with field values", func(ctx SpecContext) {
			def := structDefaultOf(ctx,
				"Point struct {\n\tx int32\n\ty int32\n}\n"+
					"Outer struct {\n\tp Point = { x = 1, y = 2 }\n}\n")
			Expect(def).NotTo(BeNil())
			Expect(def.Kind).To(Equal(resolution.ValueKindStruct))
			Expect(def.Fields).To(HaveLen(2))
			Expect(def.Fields[0]).To(Equal(resolution.StructFieldValue{
				Name:  "x",
				Value: resolution.ExpressionValue{Kind: resolution.ValueKindInt, IntValue: 1},
			}))
			Expect(def.Fields[1]).To(Equal(resolution.StructFieldValue{
				Name:  "y",
				Value: resolution.ExpressionValue{Kind: resolution.ValueKindInt, IntValue: 2},
			}))
		})

		It("Should collect nested struct and array values", func(ctx SpecContext) {
			def := structDefaultOf(ctx,
				"Inner struct {\n\ttags string[]\n}\n"+
					"Mid struct {\n\tinner Inner\n}\n"+
					"Outer struct {\n\tm Mid = { inner = { tags = [\"a\", \"b\"] } }\n}\n")
			Expect(def).NotTo(BeNil())
			Expect(def.Kind).To(Equal(resolution.ValueKindStruct))
			Expect(def.Fields).To(HaveLen(1))
			inner := def.Fields[0].Value
			Expect(inner.Kind).To(Equal(resolution.ValueKindStruct))
			Expect(inner.Fields).To(HaveLen(1))
			tags := inner.Fields[0].Value
			Expect(tags.Kind).To(Equal(resolution.ValueKindArray))
			Expect(tags.Elements).To(HaveLen(2))
			Expect(tags.Elements[0].StringValue).To(Equal("a"))
			Expect(tags.Elements[1].StringValue).To(Equal("b"))
		})
	})
})

var _ = Describe("Analyze", func() {
	var loader *MockFileLoader

	BeforeEach(func() { loader = NewMockFileLoader() })

	It("Should analyze multiple files into one table", func(ctx SpecContext) {
		loader.Add("label", `
			Label struct {
				key uuid @key
				name string
			}
		`).Add("ranger", `
			Range struct {
				key uuid @key
				name string
			}
		`)
		table, diag := analyzer.Analyze(ctx, []string{"label", "ranger"}, loader)
		Expect(diag.Ok()).To(BeTrue())
		Expect(table.MustGet("label.Label").Name).To(Equal("Label"))
		Expect(table.MustGet("ranger.Range").Name).To(Equal("Range"))
	})

	It("Should analyze a file only once when listed twice", func(ctx SpecContext) {
		loader.Add("label", `
			Label struct {
				key uuid @key
			}
		`)
		table, diag := analyzer.Analyze(ctx, []string{"label", "label.oracle"}, loader)
		Expect(diag.Ok()).To(BeTrue())
		Expect(table.MustGet("label.Label").Name).To(Equal("Label"))
	})

	It("Should report a load failure under the requested file", func(ctx SpecContext) {
		Expect(analyzer.Analyze(ctx, []string{"missing"}, loader)).Error().
			To(MatchError(ContainSubstring("failed to load file")))
	})

	It("Should report parse errors under the loaded file path", func(ctx SpecContext) {
		loader.Add("broken", "Label struct {{{{")
		table, diag := analyzer.Analyze(ctx, []string{"broken"}, loader)
		Expect(table).To(BeNil())
		Expect(diag.Ok()).To(BeFalse())
		var files []string
		diag.Each(func(file string, _ diagnostics.Diagnostic) {
			files = append(files, file)
		})
		Expect(files).To(ContainElement("broken.oracle"))
	})
})
