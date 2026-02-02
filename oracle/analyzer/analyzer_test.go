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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
)

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

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

		It("Should analyze a string enum", func() {
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

		It("Should collect enum value domains", func() {
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
			Expect(diag.Ok()).To(BeTrue())

			rangeType := table.MustGet("ranger.Range")
			Expect(rangeType.Domains).To(HaveLen(1))
			Expect(rangeType.Domains).To(HaveKey("index"))

			indexDomain := rangeType.Domains["index"]
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
			Expect(diag.Ok()).To(BeTrue())

			rangeType := table.MustGet("ranger.Range")
			form := rangeType.Form.(resolution.StructForm)

			labelsField, _ := form.Field("labels")
			Expect(labelsField.Type.Name).To(Equal("Array"))
			Expect(labelsField.Type.TypeArgs).To(HaveLen(1))
			Expect(labelsField.Type.TypeArgs[0].Name).To(Equal("uuid"))
			Expect(labelsField.IsOptional).To(BeFalse())

			tagsField, _ := form.Field("tags")
			Expect(tagsField.Type.Name).To(Equal("Array"))
			Expect(tagsField.IsOptional).To(BeTrue())
		})

		It("Should handle optional types", func() {
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
			Expect(parentField.IsOptional).To(BeTrue())
			Expect(parentField.Type.Name).To(Equal("uuid"))
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
			Expect(diag.Ok()).To(BeTrue())

			// Both structs should be in the table
			Expect(table.StructTypes()).To(HaveLen(2))
			_, ok := table.Get("ranger.Range")
			Expect(ok).To(BeTrue())
			_, ok = table.Get("label.Label")
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
			Expect(diag.Ok()).To(BeTrue())
			Expect(table.StructTypes()).To(HaveLen(3))
		})

		It("Should report missing imports", func() {
			source := `
				import "schema/core/nonexistent"
				Range struct {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag).NotTo(BeNil())
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})
	})

	Describe("Type Resolution", func() {
		It("Should resolve primitive types", func() {
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
					i json
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
				if contains(primitiveFields, field.Name) {
					Expect(resolution.IsPrimitive(field.Type.Name)).To(BeTrue())
				}
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
		It("Should report duplicate struct definitions", func() {
			source := `
				Range struct {}
				Range struct {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag).NotTo(BeNil())
			Expect(diag.Ok()).To(BeFalse())
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
		It("Should merge multiple file-level domains with the same name", func() {
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

		It("Should merge file-level domains with struct-level domains", func() {
			source := `
				@go output "core/pkg/service/user"

				User struct {
					key uuid @key
					name string

					@go omit
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

			_, found = goDomain.Expressions.Find("omit")
			Expect(found).To(BeTrue())
		})

		It("Should let struct-level domain override file-level domain expression", func() {
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
			Expect(diag.Ok()).To(BeTrue())

			childType := table.MustGet("test.Child")
			form := childType.Form.(resolution.StructForm)
			// Child has its own name field that overrides parent
			Expect(form.Fields).To(HaveLen(1))
			Expect(form.Fields[0].Name).To(Equal("name"))
			Expect(form.Fields[0].IsOptional).To(BeTrue())

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
			Expect(nameField.IsOptional).To(BeTrue())
		})

		It("Should extend generic struct with type arguments", func() {
			source := `
				Status struct<D extends json> {
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
			Expect(diag.Ok()).To(BeTrue())

			childType := table.MustGet("test.Child")
			// UnifiedFields should include fields from all ancestors
			allFields := resolution.UnifiedFields(childType, table)
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
			Expect(diag.Ok()).To(BeTrue())

			extendedType := table.MustGet("test.Extended")
			form := extendedType.Form.(resolution.StructForm)
			Expect(form.Extends).To(HaveLen(1))
			Expect(form.Extends[0].Name).To(Equal("base.Base"))

			allFields := resolution.UnifiedFields(extendedType, table)
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
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})

		It("Should detect self-extension", func() {
			source := `
				Self struct extends Self {
					a string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(table).To(BeNil())
		})

		It("Should error on non-existent parent struct", func() {
			source := `
				Child struct extends NonExistent {
					a string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
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
			Expect(diag.Ok()).To(BeFalse())
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
			Expect(keyField.IsOptional).To(BeTrue())   // Child's type
			Expect(keyField.Domains).To(HaveKey("id")) // Parent's domain inherited
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
			Expect(keyField.IsOptional).To(BeTrue())
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
		It("Should parse multiple extends with comma-separated parents", func() {
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

		It("Should use first parent's field when names conflict", func() {
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

		It("Should handle diamond inheritance", func() {
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

		It("Should detect circular inheritance with multiple parents", func() {
			source := `
				A struct extends C { a string }
				B struct { b string }
				C struct extends A, B { c string }
			`
			_, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
		})

		It("Should handle type parameters with multiple extends", func() {
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

		It("Should allow omitting fields from any parent", func() {
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

		It("Should error when omitting field not in any parent", func() {
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

	Describe("TypeDef", func() {
		It("Should analyze a distinct type (primitive alias)", func() {
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

		It("Should analyze a struct alias", func() {
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

		It("Should analyze an array type definition", func() {
			source := `
				Param struct {
					name string
					value json?
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

		It("Should analyze an alias to an array type", func() {
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
		It("Should parse generic struct with type parameter", func() {
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

		It("Should parse generic struct with constrained type parameter", func() {
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

		It("Should parse generic struct with default type parameter", func() {
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

		It("Should parse struct with generic field type", func() {
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

		It("Should preserve type params on fields with constraints and defaults", func() {
			source := `
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
			Expect(configField.Type.TypeParam.Constraint.Name).To(Equal("json"))
		})

		It("Should preserve type params in UnifiedFields for generic structs", func() {
			source := `
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
		It("Should parse map type", func() {
			source := `
				Config struct {
					settings Map<string, json>
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
			Expect(settingsField.Type.TypeArgs[1].Name).To(Equal("json"))
		})
	})

	Describe("Recursive Types", func() {
		It("Should detect recursive struct", func() {
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

		It("Should detect non-recursive struct", func() {
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
	})
})
