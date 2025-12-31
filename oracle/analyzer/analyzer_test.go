// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Analyzer", func() {
	var (
		ctx    context.Context
		loader *testutil.MockFileLoader
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = testutil.NewMockFileLoader()
	})

	Describe("AnalyzeSource", func() {
		It("Should analyze a simple struct", func() {
			source := `
				Range struct {
					key uuid @key
					name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())
			Expect(table).NotTo(BeNil())
			Expect(table.Structs).To(HaveLen(1))

			rangeStruct := table.MustGetStruct("ranger.Range")
			Expect(rangeStruct).NotTo(BeNil())
			Expect(rangeStruct.Name).To(Equal("Range"))
			Expect(rangeStruct.Namespace).To(Equal("ranger"))
			Expect(rangeStruct.HasKeyDomain).To(BeTrue())
			Expect(rangeStruct.Fields).To(HaveLen(2))

			// Check key field
			keyField := rangeStruct.Field("key")
			Expect(keyField).NotTo(BeNil())
			Expect(keyField.TypeRef.RawType).To(Equal("uuid"))
			Expect(keyField.TypeRef.Kind).To(Equal(resolution.TypeKindPrimitive))
			Expect(keyField.TypeRef.Primitive).To(Equal("uuid"))
			Expect(keyField.Domains).To(HaveKey("key"))

			// Check name field
			nameField := rangeStruct.Field("name")
			Expect(nameField).NotTo(BeNil())
			Expect(nameField.TypeRef.RawType).To(Equal("string"))
			Expect(nameField.TypeRef.Kind).To(Equal(resolution.TypeKindPrimitive))
		})

		It("Should analyze an enum", func() {
			source := `
				TaskState enum {
					pending = 0
					running = 1
					completed = 2
					failed = 3
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.HasErrors()).To(BeFalse())
			Expect(table.Enums).To(HaveLen(1))

			taskState := table.MustGetEnum("task.TaskState")
			Expect(taskState).NotTo(BeNil())
			Expect(taskState.Name).To(Equal("TaskState"))
			Expect(taskState.IsIntEnum).To(BeTrue())
			Expect(taskState.Values).To(HaveLen(4))
			Expect(taskState.Values[0].Name).To(Equal("pending"))
			Expect(taskState.Values[0].IntValue).To(Equal(int64(0)))
			Expect(taskState.ValuesByName["failed"].IntValue).To(Equal(int64(3)))
		})

		It("Should analyze a string enum", func() {
			source := `
				DataType enum {
					float32 = "float32"
					float64 = "float64"
					int32 = "int32"
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "telem", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			dataType := table.MustGetEnum("telem.DataType")
			Expect(dataType).NotTo(BeNil())
			Expect(dataType.IsIntEnum).To(BeFalse())
			Expect(dataType.Values[0].StringValue).To(Equal("float32"))
		})

		It("Should collect field domains", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			userStruct := table.MustGetStruct("user.User")
			nameField := userStruct.Field("name")
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

		It("Should collect struct-level domains", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			rangeStruct := table.MustGetStruct("ranger.Range")
			Expect(rangeStruct.Domains).To(HaveLen(1))
			Expect(rangeStruct.Domains).To(HaveKey("index"))

			indexDomain := rangeStruct.Domains["index"]
			Expect(indexDomain.Expressions).To(HaveLen(1))
			Expect(indexDomain.Expressions[0].Name).To(Equal("composite"))
		})

		It("Should handle array types", func() {
			source := `
				Range struct {
					labels uuid[]
					tags string[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			rangeStruct := table.MustGetStruct("ranger.Range")

			labelsField := rangeStruct.Field("labels")
			Expect(labelsField.TypeRef.IsArray).To(BeTrue())
			Expect(labelsField.TypeRef.IsOptional).To(BeFalse())

			tagsField := rangeStruct.Field("tags")
			Expect(tagsField.TypeRef.IsArray).To(BeTrue())
			Expect(tagsField.TypeRef.IsOptional).To(BeTrue())
		})

		It("Should handle optional types", func() {
			source := `
				Range struct {
					parent uuid?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			rangeStruct := table.MustGetStruct("ranger.Range")
			parentField := rangeStruct.Field("parent")
			Expect(parentField.TypeRef.IsOptional).To(BeTrue())
			Expect(parentField.TypeRef.IsArray).To(BeFalse())
		})
	})

	Describe("Import Resolution", func() {
		It("Should resolve imports", func() {
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
					labels uuid[]
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			// Both structs should be in the table
			Expect(table.Structs).To(HaveLen(2))
			_, ok := table.GetStruct("ranger.Range")
			Expect(ok).To(BeTrue())
			_, ok = table.GetStruct("label.Label")
			Expect(ok).To(BeTrue())
		})

		It("Should detect circular imports", func() {
			loader.Files["schema/core/a"] = `
				import "schema/core/b"
				A struct {}
			`
			loader.Files["schema/core/b"] = `
				import "schema/core/a"
				B struct {}
			`

			source := `
				import "schema/core/a"
				C struct {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "main", loader)
			// Should not error - circular imports are handled by tracking
			Expect(diag.HasErrors()).To(BeFalse())
			Expect(table.Structs).To(HaveLen(3))
		})

		It("Should report missing imports", func() {
			source := `
				import "schema/core/nonexistent"
				Range struct {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
			Expect(table).To(BeNil())
		})
	})

	Describe("Type Resolution", func() {
		It("Should resolve primitive types", func() {
			source := `
				Test struct {
					a uuid
					b string
					c int32
					d float64
					e bool
					f timestamp
					g timespan
					h time_range
					i json
					j bytes
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			testStruct := table.MustGetStruct("test.Test")
			for _, field := range testStruct.Fields {
				Expect(field.TypeRef.Kind).To(Equal(resolution.TypeKindPrimitive))
			}
		})

		It("Should resolve struct references in same namespace", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			viewportStruct := table.MustGetStruct("viz.Viewport")
			positionField := viewportStruct.Field("position")
			Expect(positionField.TypeRef.Kind).To(Equal(resolution.TypeKindStruct))
			Expect(positionField.TypeRef.StructRef).NotTo(BeNil())
			Expect(positionField.TypeRef.StructRef.Name).To(Equal("Position"))
		})

		It("Should resolve qualified struct references", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			rangeStruct := table.MustGetStruct("ranger.Range")
			labelsField := rangeStruct.Field("labels")
			Expect(labelsField.TypeRef.RawType).To(Equal("label.Label"))
			Expect(labelsField.TypeRef.Kind).To(Equal(resolution.TypeKindStruct))
			Expect(labelsField.TypeRef.StructRef).NotTo(BeNil())
			Expect(labelsField.TypeRef.StructRef.Name).To(Equal("Label"))
		})

		It("Should resolve enum references", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			taskStruct := table.MustGetStruct("task.Task")
			stateField := taskStruct.Field("state")
			Expect(stateField.TypeRef.Kind).To(Equal(resolution.TypeKindEnum))
			Expect(stateField.TypeRef.EnumRef).NotTo(BeNil())
			Expect(stateField.TypeRef.EnumRef.Name).To(Equal("TaskState"))
		})
	})

	Describe("Error Handling", func() {
		It("Should report duplicate struct definitions", func() {
			source := `
				Range struct {}
				Range struct {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
			Expect(table).To(BeNil())
		})

		It("Should report duplicate field definitions", func() {
			source := `
				Range struct {
					name string
					name int32
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
			Expect(table).To(BeNil())
		})

		It("Should report duplicate enum definitions", func() {
			source := `
				State enum {
					a = 0
				}
				State enum {
					b = 0
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
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

	Describe("Struct Extension", func() {
		It("Should parse basic struct extension", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())
			Expect(table.Structs).To(HaveLen(2))

			child := table.MustGetStruct("test.Child")
			Expect(child.HasExtends()).To(BeTrue())
			Expect(child.Extends).NotTo(BeNil())
			Expect(child.Extends.StructRef).NotTo(BeNil())
			Expect(child.Extends.StructRef.Name).To(Equal("Parent"))

			// Child should have its own field
			Expect(child.Fields).To(HaveLen(1))
			Expect(child.Fields[0].Name).To(Equal("email"))

			// AllFields should include inherited fields
			allFields := child.AllFields()
			Expect(allFields).To(HaveLen(3))
			fieldNames := []string{allFields[0].Name, allFields[1].Name, allFields[2].Name}
			Expect(fieldNames).To(ContainElements("name", "age", "email"))
		})

		It("Should parse field omission with -fieldName syntax", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			child := table.MustGetStruct("test.Child")
			Expect(child.OmittedFields).To(HaveLen(1))
			Expect(child.OmittedFields[0]).To(Equal("age"))

			// AllFields should NOT include omitted field
			allFields := child.AllFields()
			Expect(allFields).To(HaveLen(3))
			fieldNames := []string{allFields[0].Name, allFields[1].Name, allFields[2].Name}
			Expect(fieldNames).To(ContainElements("name", "status", "email"))
			Expect(fieldNames).NotTo(ContainElement("age"))
		})

		It("Should handle field override in child struct", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			child := table.MustGetStruct("test.Child")
			// Child has its own name field that overrides parent
			Expect(child.Fields).To(HaveLen(1))
			Expect(child.Fields[0].Name).To(Equal("name"))
			Expect(child.Fields[0].TypeRef.IsOptional).To(BeTrue())

			// AllFields should have child's version of name
			allFields := child.AllFields()
			Expect(allFields).To(HaveLen(2))

			var nameField *resolution.FieldEntry
			for _, f := range allFields {
				if f.Name == "name" {
					nameField = f
					break
				}
			}
			Expect(nameField).NotTo(BeNil())
			Expect(nameField.TypeRef.IsOptional).To(BeTrue())
		})

		It("Should extend generic struct with type arguments", func() {
			source := `
				Status struct<D extends schema> {
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
			Expect(diag.HasErrors()).To(BeFalse())

			rackStatus := table.MustGetStruct("test.RackStatus")
			Expect(rackStatus.HasExtends()).To(BeTrue())
			Expect(rackStatus.Extends.TypeArgs).To(HaveLen(1))
			Expect(rackStatus.Extends.TypeArgs[0].StructRef.Name).To(Equal("Details"))

			// AllFields should substitute type parameters
			allFields := rackStatus.AllFields()
			Expect(allFields).To(HaveLen(3))

			var dataField *resolution.FieldEntry
			for _, f := range allFields {
				if f.Name == "data" {
					dataField = f
					break
				}
			}
			Expect(dataField).NotTo(BeNil())
			Expect(dataField.TypeRef.Kind).To(Equal(resolution.TypeKindStruct))
			Expect(dataField.TypeRef.StructRef.Name).To(Equal("Details"))
		})

		It("Should handle multi-level inheritance", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			child := table.MustGetStruct("test.Child")
			// AllFields should include fields from all ancestors
			allFields := child.AllFields()
			Expect(allFields).To(HaveLen(3))
			fieldNames := []string{allFields[0].Name, allFields[1].Name, allFields[2].Name}
			Expect(fieldNames).To(ContainElements("a", "b", "c"))
		})

		It("Should extend struct from imported file", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			extended := table.MustGetStruct("test.Extended")
			Expect(extended.HasExtends()).To(BeTrue())
			Expect(extended.Extends.StructRef.Name).To(Equal("Base"))
			Expect(extended.Extends.StructRef.Namespace).To(Equal("base"))

			allFields := extended.AllFields()
			Expect(allFields).To(HaveLen(3))
		})

		It("Should detect circular inheritance", func() {
			source := `
				A struct extends B {
					a string
				}

				B struct extends A {
					b string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeTrue())
			Expect(table).To(BeNil())
		})

		It("Should detect self-extension", func() {
			source := `
				Self struct extends Self {
					a string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeTrue())
			Expect(table).To(BeNil())
		})

		It("Should error on non-existent parent struct", func() {
			source := `
				Child struct extends NonExistent {
					a string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeTrue())
			Expect(table).To(BeNil())
		})

		It("Should error on omitting non-existent field", func() {
			source := `
				Parent struct {
					name string
				}

				Child struct extends Parent {
					-nonexistent
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeTrue())
			Expect(table).To(BeNil())
		})

		It("Should inherit parent field domains on override", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			child := table.MustGetStruct("test.Child")
			allFields := child.AllFields()

			// Find the key field
			var keyField *resolution.FieldEntry
			for _, f := range allFields {
				if f.Name == "key" {
					keyField = f
					break
				}
			}

			Expect(keyField).NotTo(BeNil())
			Expect(keyField.TypeRef.IsOptional).To(BeTrue())  // Child's type
			Expect(keyField.Domains).To(HaveKey("id"))        // Parent's domain inherited
		})

		It("Should allow child to override parent domain", func() {
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
			Expect(diag.HasErrors()).To(BeFalse())

			child := table.MustGetStruct("test.Child")
			allFields := child.AllFields()

			// Find the name field
			var nameField *resolution.FieldEntry
			for _, f := range allFields {
				if f.Name == "name" {
					nameField = f
					break
				}
			}

			Expect(nameField).NotTo(BeNil())
			Expect(nameField.Domains).To(HaveKey("validate"))

			// Child's @validate should win - check min_length is 5, not 1
			validateDomain := nameField.Domains["validate"]
			Expect(validateDomain.Expressions).To(HaveLen(2))

			// Find min_length expression
			var minLengthExpr *resolution.ExpressionEntry
			for _, expr := range validateDomain.Expressions {
				if expr.Name == "min_length" {
					minLengthExpr = expr
					break
				}
			}
			Expect(minLengthExpr).NotTo(BeNil())
			Expect(minLengthExpr.Values[0].IntValue).To(Equal(int64(5))) // Child's value, not parent's 1
		})

		It("Should merge domains from parent when child adds new domain", func() {
			source := `
				Parent struct {
					key uuid @id
				}

				Child struct extends Parent {
					key uuid? @validate required
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			child := table.MustGetStruct("test.Child")
			allFields := child.AllFields()

			var keyField *resolution.FieldEntry
			for _, f := range allFields {
				if f.Name == "key" {
					keyField = f
					break
				}
			}

			Expect(keyField).NotTo(BeNil())
			Expect(keyField.TypeRef.IsOptional).To(BeTrue())
			Expect(keyField.Domains).To(HaveKey("id"))       // Inherited from parent
			Expect(keyField.Domains).To(HaveKey("validate")) // Added by child
		})

		It("Should merge expressions within same domain from parent and child", func() {
			source := `
				Parent struct {
					name string @validate min_length 1
				}

				Child struct extends Parent {
					name string @validate max_length 100
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			child := table.MustGetStruct("test.Child")
			allFields := child.AllFields()

			var nameField *resolution.FieldEntry
			for _, f := range allFields {
				if f.Name == "name" {
					nameField = f
					break
				}
			}

			Expect(nameField).NotTo(BeNil())
			validateDomain := nameField.Domains["validate"]
			Expect(validateDomain.Expressions).To(HaveLen(2)) // Both min_length and max_length

			// Build map for easy lookup
			exprMap := make(map[string]*resolution.ExpressionEntry)
			for _, expr := range validateDomain.Expressions {
				exprMap[expr.Name] = expr
			}

			Expect(exprMap).To(HaveKey("min_length")) // From parent
			Expect(exprMap).To(HaveKey("max_length")) // From child
			Expect(exprMap["min_length"].Values[0].IntValue).To(Equal(int64(1)))
			Expect(exprMap["max_length"].Values[0].IntValue).To(Equal(int64(100)))
		})
	})
})
