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
)

// MockFileLoader is a file loader that serves files from memory.
type MockFileLoader struct {
	Files map[string]string
}

func (m *MockFileLoader) Load(importPath string) (string, string, error) {
	if content, ok := m.Files[importPath]; ok {
		return content, importPath + ".oracle", nil
	}
	if content, ok := m.Files[importPath+".oracle"]; ok {
		return content, importPath + ".oracle", nil
	}
	return "", "", &fileNotFoundError{path: importPath}
}

func (m *MockFileLoader) RepoRoot() string {
	return "/mock/repo"
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

var _ = Describe("Analyzer", func() {
	var (
		ctx    context.Context
		loader *MockFileLoader
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = &MockFileLoader{Files: make(map[string]string)}
	})

	Describe("AnalyzeSource", func() {
		It("Should analyze a simple struct", func() {
			source := `
				struct Range {
					field key uuid {
						domain id
					}
					field name string
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())
			Expect(table).NotTo(BeNil())
			Expect(table.Structs).To(HaveLen(1))

			rangeStruct := table.Structs["ranger.Range"]
			Expect(rangeStruct).NotTo(BeNil())
			Expect(rangeStruct.Name).To(Equal("Range"))
			Expect(rangeStruct.Namespace).To(Equal("ranger"))
			Expect(rangeStruct.HasIDDomain).To(BeTrue())
			Expect(rangeStruct.Fields).To(HaveLen(2))

			// Check key field
			keyField := rangeStruct.Field("key")
			Expect(keyField).NotTo(BeNil())
			Expect(keyField.TypeRef.RawType).To(Equal("uuid"))
			Expect(keyField.TypeRef.Kind).To(Equal(resolution.TypeKindPrimitive))
			Expect(keyField.TypeRef.Primitive).To(Equal("uuid"))
			Expect(keyField.Domains).To(HaveKey("id"))

			// Check name field
			nameField := rangeStruct.Field("name")
			Expect(nameField).NotTo(BeNil())
			Expect(nameField.TypeRef.RawType).To(Equal("string"))
			Expect(nameField.TypeRef.Kind).To(Equal(resolution.TypeKindPrimitive))
		})

		It("Should analyze an enum", func() {
			source := `
				enum TaskState {
					pending = 0
					running = 1
					completed = 2
					failed = 3
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.HasErrors()).To(BeFalse())
			Expect(table.Enums).To(HaveLen(1))

			taskState := table.Enums["task.TaskState"]
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
				enum DataType {
					float32 = "float32"
					float64 = "float64"
					int32 = "int32"
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "telem", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			dataType := table.Enums["telem.DataType"]
			Expect(dataType).NotTo(BeNil())
			Expect(dataType.IsIntEnum).To(BeFalse())
			Expect(dataType.Values[0].StringValue).To(Equal("float32"))
		})

		It("Should collect field domains", func() {
			source := `
				struct User {
					field name string {
						domain validate {
							required
							max_length 255
							min_length 1
						}
						domain query {
							eq
							contains
						}
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "user", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			userStruct := table.Structs["user.User"]
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
				struct Range {
					field key uuid
					field name string

					domain index {
						composite name created_at sorted
					}
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			rangeStruct := table.Structs["ranger.Range"]
			Expect(rangeStruct.Domains).To(HaveLen(1))
			Expect(rangeStruct.Domains).To(HaveKey("index"))

			indexDomain := rangeStruct.Domains["index"]
			Expect(indexDomain.Expressions).To(HaveLen(1))
			Expect(indexDomain.Expressions[0].Name).To(Equal("composite"))
		})

		It("Should handle array types", func() {
			source := `
				struct Range {
					field labels uuid[]
					field tags string[]?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			rangeStruct := table.Structs["ranger.Range"]

			labelsField := rangeStruct.Field("labels")
			Expect(labelsField.TypeRef.IsArray).To(BeTrue())
			Expect(labelsField.TypeRef.IsOptional).To(BeFalse())

			tagsField := rangeStruct.Field("tags")
			Expect(tagsField.TypeRef.IsArray).To(BeTrue())
			Expect(tagsField.TypeRef.IsOptional).To(BeTrue())
		})

		It("Should handle optional types", func() {
			source := `
				struct Range {
					field parent uuid?
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			rangeStruct := table.Structs["ranger.Range"]
			parentField := rangeStruct.Field("parent")
			Expect(parentField.TypeRef.IsOptional).To(BeTrue())
			Expect(parentField.TypeRef.IsArray).To(BeFalse())
		})
	})

	Describe("Import Resolution", func() {
		It("Should resolve imports", func() {
			loader.Files["schema/core/label"] = `
				struct Label {
					field key uuid {
						domain id
					}
					field name string
				}
			`

			source := `
				import "schema/core/label"

				struct Range {
					field key uuid {
						domain id
					}
					field labels uuid[]
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			// Both structs should be in the table
			Expect(table.Structs).To(HaveLen(2))
			Expect(table.Structs).To(HaveKey("ranger.Range"))
			Expect(table.Structs).To(HaveKey("label.Label"))
		})

		It("Should detect circular imports", func() {
			loader.Files["schema/core/a"] = `
				import "schema/core/b"
				struct A {}
			`
			loader.Files["schema/core/b"] = `
				import "schema/core/a"
				struct B {}
			`

			source := `
				import "schema/core/a"
				struct C {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "main", loader)
			// Should not error - circular imports are handled by tracking
			Expect(diag.HasErrors()).To(BeFalse())
			Expect(table.Structs).To(HaveLen(3))
		})

		It("Should report missing imports", func() {
			source := `
				import "schema/core/nonexistent"
				struct Range {}
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
				struct Test {
					field a uuid
					field b string
					field c int32
					field d float64
					field e bool
					field f timestamp
					field g timespan
					field h time_range
					field i json
					field j bytes
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			testStruct := table.Structs["test.Test"]
			for _, field := range testStruct.Fields {
				Expect(field.TypeRef.Kind).To(Equal(resolution.TypeKindPrimitive))
			}
		})

		It("Should resolve struct references in same namespace", func() {
			source := `
				struct Position {
					field x float64
					field y float64
				}

				struct Viewport {
					field position Position
					field zoom float64
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "viz", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			viewportStruct := table.Structs["viz.Viewport"]
			positionField := viewportStruct.Field("position")
			Expect(positionField.TypeRef.Kind).To(Equal(resolution.TypeKindStruct))
			Expect(positionField.TypeRef.StructRef).NotTo(BeNil())
			Expect(positionField.TypeRef.StructRef.Name).To(Equal("Position"))
		})

		It("Should resolve qualified struct references", func() {
			loader.Files["schema/core/label"] = `
				struct Label {
					field key uuid
					field name string
				}
			`

			source := `
				import "schema/core/label"

				struct Range {
					field labels label.Label[]
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			rangeStruct := table.Structs["ranger.Range"]
			labelsField := rangeStruct.Field("labels")
			Expect(labelsField.TypeRef.RawType).To(Equal("label.Label"))
			Expect(labelsField.TypeRef.Kind).To(Equal(resolution.TypeKindStruct))
			Expect(labelsField.TypeRef.StructRef).NotTo(BeNil())
			Expect(labelsField.TypeRef.StructRef.Name).To(Equal("Label"))
		})

		It("Should resolve enum references", func() {
			source := `
				enum TaskState {
					pending = 0
					running = 1
				}

				struct Task {
					field state TaskState
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "task", loader)
			Expect(diag.HasErrors()).To(BeFalse())

			taskStruct := table.Structs["task.Task"]
			stateField := taskStruct.Field("state")
			Expect(stateField.TypeRef.Kind).To(Equal(resolution.TypeKindEnum))
			Expect(stateField.TypeRef.EnumRef).NotTo(BeNil())
			Expect(stateField.TypeRef.EnumRef.Name).To(Equal("TaskState"))
		})
	})

	Describe("Error Handling", func() {
		It("Should report duplicate struct definitions", func() {
			source := `
				struct Range {}
				struct Range {}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
			Expect(table).To(BeNil())
		})

		It("Should report duplicate field definitions", func() {
			source := `
				struct Range {
					field name string
					field name int32
				}
			`
			table, diag := analyzer.AnalyzeSource(ctx, source, "ranger", loader)
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
			Expect(table).To(BeNil())
		})

		It("Should report duplicate enum definitions", func() {
			source := `
				enum State {
					a = 0
				}
				enum State {
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
})
