// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/parser"
)

var _ = Describe("Parser", func() {
	Describe("Schema Parsing", func() {
		It("Should parse an empty schema", func() {
			schema, diag := parser.Parse(``)
			Expect(diag).To(BeNil())
			Expect(schema).NotTo(BeNil())
			Expect(schema.AllDefinition()).To(BeEmpty())
			Expect(schema.AllImportStmt()).To(BeEmpty())
		})

		It("Should parse a schema with comments only", func() {
			schema, diag := parser.Parse(`
				// This is a comment
				/* This is a
				   multi-line comment */
			`)
			Expect(diag).To(BeNil())
			Expect(schema).NotTo(BeNil())
		})
	})

	Describe("Import Statements", func() {
		It("Should parse a single import", func() {
			schema, diag := parser.Parse(`import "schema/core/label"`)
			Expect(diag).To(BeNil())
			Expect(schema.AllImportStmt()).To(HaveLen(1))
			Expect(schema.ImportStmt(0).STRING_LIT().GetText()).To(Equal(`"schema/core/label"`))
		})

		It("Should parse multiple imports", func() {
			schema, diag := parser.Parse(`
				import "schema/core/label"
				import "schema/core/channel"
				import "schema/visualization/schematic"
			`)
			Expect(diag).To(BeNil())
			Expect(schema.AllImportStmt()).To(HaveLen(3))
		})
	})

	Describe("Struct Definitions", func() {
		It("Should parse a simple struct with no fields", func() {
			schema, diag := parser.Parse(`struct Empty {}`)
			Expect(diag).To(BeNil())
			Expect(schema.AllDefinition()).To(HaveLen(1))
			structDef := schema.Definition(0).StructDef()
			Expect(structDef).NotTo(BeNil())
			Expect(structDef.IDENT().GetText()).To(Equal("Empty"))
		})

		It("Should parse a struct with simple fields", func() {
			schema, diag := parser.Parse(`
				struct Range {
					field key uuid
					field name string
					field description string?
				}
			`)
			Expect(diag).To(BeNil())
			structDef := schema.Definition(0).StructDef()
			fields := structDef.StructBody().AllFieldDef()
			Expect(fields).To(HaveLen(3))

			// key uuid
			Expect(fields[0].IDENT().GetText()).To(Equal("key"))
			Expect(fields[0].TypeRef().QualifiedIdent().IDENT(0).GetText()).To(Equal("uuid"))
			Expect(fields[0].TypeRef().QUESTION()).To(BeNil())

			// name string
			Expect(fields[1].IDENT().GetText()).To(Equal("name"))
			Expect(fields[1].TypeRef().QualifiedIdent().IDENT(0).GetText()).To(Equal("string"))

			// description string?
			Expect(fields[2].IDENT().GetText()).To(Equal("description"))
			Expect(fields[2].TypeRef().QUESTION()).NotTo(BeNil())
		})

		It("Should parse array type fields", func() {
			schema, diag := parser.Parse(`
				struct Range {
					field labels uuid[]
					field tags string[]?
				}
			`)
			Expect(diag).To(BeNil())
			structDef := schema.Definition(0).StructDef()
			fields := structDef.StructBody().AllFieldDef()
			Expect(fields).To(HaveLen(2))

			// labels uuid[]
			Expect(fields[0].TypeRef().LBRACKET()).NotTo(BeNil())
			Expect(fields[0].TypeRef().QUESTION()).To(BeNil())

			// tags string[]?
			Expect(fields[1].TypeRef().LBRACKET()).NotTo(BeNil())
			Expect(fields[1].TypeRef().QUESTION()).NotTo(BeNil())
		})

		It("Should parse qualified type references", func() {
			schema, diag := parser.Parse(`
				struct Range {
					field channel channel.Channel
					field labels label.Label[]
				}
			`)
			Expect(diag).To(BeNil())
			structDef := schema.Definition(0).StructDef()
			fields := structDef.StructBody().AllFieldDef()

			// channel.Channel
			qualIdent := fields[0].TypeRef().QualifiedIdent()
			Expect(qualIdent.IDENT(0).GetText()).To(Equal("channel"))
			Expect(qualIdent.IDENT(1).GetText()).To(Equal("Channel"))

			// label.Label[]
			qualIdent2 := fields[1].TypeRef().QualifiedIdent()
			Expect(qualIdent2.IDENT(0).GetText()).To(Equal("label"))
			Expect(qualIdent2.IDENT(1).GetText()).To(Equal("Label"))
			Expect(fields[1].TypeRef().LBRACKET()).NotTo(BeNil())
		})
	})

	Describe("Field Domains", func() {
		It("Should parse a field with an empty domain", func() {
			schema, diag := parser.Parse(`
				struct Range {
					field key uuid {
						domain id
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := schema.Definition(0).StructDef()
			field := structDef.StructBody().FieldDef(0)
			domains := field.FieldBody().AllDomainDef()
			Expect(domains).To(HaveLen(1))
			Expect(domains[0].IDENT().GetText()).To(Equal("id"))
			Expect(domains[0].DomainBody()).To(BeNil())
		})

		It("Should parse a field with domain expressions", func() {
			schema, diag := parser.Parse(`
				struct Range {
					field name string {
						domain validate {
							required
							max_length 255
							min_length 1
						}
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := schema.Definition(0).StructDef()
			field := structDef.StructBody().FieldDef(0)
			domain := field.FieldBody().DomainDef(0)
			Expect(domain.IDENT().GetText()).To(Equal("validate"))

			exprs := domain.DomainBody().AllExpression()
			Expect(exprs).To(HaveLen(3))

			// required (flag)
			Expect(exprs[0].IDENT().GetText()).To(Equal("required"))
			Expect(exprs[0].AllExpressionValue()).To(BeEmpty())

			// max_length 255
			Expect(exprs[1].IDENT().GetText()).To(Equal("max_length"))
			Expect(exprs[1].ExpressionValue(0).INT_LIT().GetText()).To(Equal("255"))

			// min_length 1
			Expect(exprs[2].IDENT().GetText()).To(Equal("min_length"))
			Expect(exprs[2].ExpressionValue(0).INT_LIT().GetText()).To(Equal("1"))
		})

		It("Should parse a field with multiple domains", func() {
			schema, diag := parser.Parse(`
				struct Range {
					field name string {
						domain validate {
							required
						}
						domain query {
							eq
							contains
							starts_with
						}
						domain index {
							lookup
						}
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := schema.Definition(0).StructDef()
			field := structDef.StructBody().FieldDef(0)
			domains := field.FieldBody().AllDomainDef()
			Expect(domains).To(HaveLen(3))
			Expect(domains[0].IDENT().GetText()).To(Equal("validate"))
			Expect(domains[1].IDENT().GetText()).To(Equal("query"))
			Expect(domains[2].IDENT().GetText()).To(Equal("index"))
		})

		It("Should parse domain expressions with string literals", func() {
			schema, diag := parser.Parse(`
				struct User {
					field name string {
						domain validate {
							default "untitled"
							pattern "[a-z]+"
						}
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := schema.Definition(0).StructDef()
			domain := structDef.StructBody().FieldDef(0).FieldBody().DomainDef(0)
			exprs := domain.DomainBody().AllExpression()

			// default "untitled"
			Expect(exprs[0].IDENT().GetText()).To(Equal("default"))
			Expect(exprs[0].ExpressionValue(0).STRING_LIT().GetText()).To(Equal(`"untitled"`))

			// pattern "[a-z]+"
			Expect(exprs[1].IDENT().GetText()).To(Equal("pattern"))
			Expect(exprs[1].ExpressionValue(0).STRING_LIT().GetText()).To(Equal(`"[a-z]+"`))
		})

		It("Should parse domain expressions with identifier values", func() {
			schema, diag := parser.Parse(`
				struct Range {
					field labels uuid[] {
						domain relation {
							target label.Label
							cardinality many_to_many
						}
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := schema.Definition(0).StructDef()
			domain := structDef.StructBody().FieldDef(0).FieldBody().DomainDef(0)
			exprs := domain.DomainBody().AllExpression()

			// target label.Label
			Expect(exprs[0].IDENT().GetText()).To(Equal("target"))

			// cardinality many_to_many
			Expect(exprs[1].IDENT().GetText()).To(Equal("cardinality"))
			// ExpressionValue now uses QualifiedIdent for identifiers
			qualIdent := exprs[1].ExpressionValue(0).QualifiedIdent()
			Expect(qualIdent.IDENT(0).GetText()).To(Equal("many_to_many"))
		})
	})

	Describe("Struct-Level Domains", func() {
		It("Should parse struct-level domains", func() {
			schema, diag := parser.Parse(`
				struct Range {
					field key uuid
					field name string

					domain index {
						composite name created_at sorted
						unique name workspace
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := schema.Definition(0).StructDef()
			structBody := structDef.StructBody()

			// Should have 2 fields and 1 domain
			Expect(structBody.AllFieldDef()).To(HaveLen(2))
			Expect(structBody.AllDomainDef()).To(HaveLen(1))

			domain := structBody.DomainDef(0)
			Expect(domain.IDENT().GetText()).To(Equal("index"))
			Expect(domain.DomainBody().AllExpression()).To(HaveLen(2))
		})
	})

	Describe("Enum Definitions", func() {
		It("Should parse an enum with integer values", func() {
			schema, diag := parser.Parse(`
				enum TaskState {
					pending = 0
					running = 1
					completed = 2
					failed = 3
				}
			`)
			Expect(diag).To(BeNil())
			Expect(schema.AllDefinition()).To(HaveLen(1))
			enumDef := schema.Definition(0).EnumDef()
			Expect(enumDef).NotTo(BeNil())
			Expect(enumDef.IDENT().GetText()).To(Equal("TaskState"))

			values := enumDef.AllEnumValue()
			Expect(values).To(HaveLen(4))
			Expect(values[0].IDENT().GetText()).To(Equal("pending"))
			Expect(values[0].INT_LIT().GetText()).To(Equal("0"))
			Expect(values[3].IDENT().GetText()).To(Equal("failed"))
			Expect(values[3].INT_LIT().GetText()).To(Equal("3"))
		})

		It("Should parse an enum with string values", func() {
			schema, diag := parser.Parse(`
				enum DataType {
					float32 = "float32"
					float64 = "float64"
					int32 = "int32"
				}
			`)
			Expect(diag).To(BeNil())
			enumDef := schema.Definition(0).EnumDef()
			values := enumDef.AllEnumValue()
			Expect(values).To(HaveLen(3))
			Expect(values[0].IDENT().GetText()).To(Equal("float32"))
			Expect(values[0].STRING_LIT().GetText()).To(Equal(`"float32"`))
		})
	})

	Describe("Complete Schema", func() {
		It("Should parse a complete schema from the RFC", func() {
			schema, diag := parser.Parse(`
				import "schema/core/label"

				struct Range {
					field key uuid {
						domain id
					}

					field name string {
						domain validate {
							required
							max_length 255
						}
						domain query {
							eq
							neq
							contains
							starts_with
						}
						domain index {
							lookup
						}
					}

					field labels uuid[] {
						domain relation {
							target label.Label
							cardinality many_to_many
						}
						domain query {
							has_any
							has_all
							has_none
						}
						domain index {
							lookup
						}
					}

					field time_range time_range {
						domain validate {
							required
						}
						domain query {
							overlaps
							contains_time
						}
						domain index {
							range
						}
					}

					field created_at timestamp {
						domain validate {
							default now
							immutable
						}
						domain query {
							eq
							gt
							gte
							lt
							lte
							between
						}
						domain index {
							sorted
						}
						domain sort
					}

					field parent uuid? {
						domain relation {
							target Range
							self
						}
					}
				}
			`)
			Expect(diag).To(BeNil())
			Expect(schema.AllImportStmt()).To(HaveLen(1))
			Expect(schema.AllDefinition()).To(HaveLen(1))

			structDef := schema.Definition(0).StructDef()
			Expect(structDef.IDENT().GetText()).To(Equal("Range"))
			Expect(structDef.StructBody().AllFieldDef()).To(HaveLen(6))
		})
	})

	Describe("Error Handling", func() {
		It("Should report syntax errors", func() {
			_, diag := parser.Parse(`struct { }`) // missing name
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
		})

		It("Should report errors for missing braces", func() {
			_, diag := parser.Parse(`struct Range field key uuid`)
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
		})

		It("Should report errors for invalid enum value", func() {
			_, diag := parser.Parse(`
				enum State {
					pending
				}
			`) // missing = value
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
		})
	})
})
