// Copyright 2026 Synnax Labs, Inc.
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

// Helper to get StructFullContext from a struct definition
func asStructFull(structDef parser.IStructDefContext) *parser.StructFullContext {
	return structDef.(*parser.StructFullContext)
}

// Helper to get TypeRefNormalContext from a type reference
func asTypeRefNormal(typeRef parser.ITypeRefContext) *parser.TypeRefNormalContext {
	return typeRef.(*parser.TypeRefNormalContext)
}

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

	Describe("File-Level Domains", func() {
		It("Should parse a file-level domain with string output", func() {
			schema, diag := parser.Parse(`@ts output "client/ts/src/rack"`)
			Expect(diag).To(BeNil())
			Expect(schema.AllFileDomain()).To(HaveLen(1))
			fd := schema.FileDomain(0)
			Expect(fd.IDENT().GetText()).To(Equal("ts"))
			content := fd.DomainContent()
			Expect(content).NotTo(BeNil())
			// Single expression (not a block)
			expr := content.Expression()
			Expect(expr).NotTo(BeNil())
			Expect(expr.IDENT().GetText()).To(Equal("output"))
		})

		It("Should parse multiple file-level domains", func() {
			schema, diag := parser.Parse(`
				@ts output "client/ts/src/rack"
				@py output "client/py/synnax/rack"
			`)
			Expect(diag).To(BeNil())
			Expect(schema.AllFileDomain()).To(HaveLen(2))
		})
	})

	Describe("Struct Definitions", func() {
		It("Should parse a simple struct with no fields", func() {
			schema, diag := parser.Parse(`Empty struct {}`)
			Expect(diag).To(BeNil())
			Expect(schema.AllDefinition()).To(HaveLen(1))
			structDef := asStructFull(schema.Definition(0).StructDef())
			Expect(structDef).NotTo(BeNil())
			Expect(structDef.IDENT().GetText()).To(Equal("Empty"))
		})

		It("Should parse a struct with simple fields", func() {
			schema, diag := parser.Parse(`
				Range struct {
					key uuid
					name string
					description string?
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			fields := structDef.StructBody().AllFieldDef()
			Expect(fields).To(HaveLen(3))

			// key uuid
			Expect(fields[0].IDENT().GetText()).To(Equal("key"))
			tr0 := asTypeRefNormal(fields[0].TypeRef())
			Expect(tr0.QualifiedIdent().IDENT(0).GetText()).To(Equal("uuid"))
			Expect(tr0.TypeModifiers()).To(BeNil())

			// name string
			Expect(fields[1].IDENT().GetText()).To(Equal("name"))
			tr1 := asTypeRefNormal(fields[1].TypeRef())
			Expect(tr1.QualifiedIdent().IDENT(0).GetText()).To(Equal("string"))

			// description string?
			Expect(fields[2].IDENT().GetText()).To(Equal("description"))
			tr2 := asTypeRefNormal(fields[2].TypeRef())
			Expect(tr2.TypeModifiers()).NotTo(BeNil())
			Expect(tr2.TypeModifiers().AllQUESTION()).To(HaveLen(1))
		})

		It("Should parse array type fields", func() {
			schema, diag := parser.Parse(`
				Range struct {
					labels uuid[]
					tags string[]?
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			fields := structDef.StructBody().AllFieldDef()
			Expect(fields).To(HaveLen(2))

			// labels uuid[]
			tr0 := asTypeRefNormal(fields[0].TypeRef())
			Expect(tr0.ArrayModifier()).NotTo(BeNil())
			Expect(tr0.TypeModifiers()).To(BeNil())

			// tags string[]?
			tr1 := asTypeRefNormal(fields[1].TypeRef())
			Expect(tr1.ArrayModifier()).NotTo(BeNil())
			Expect(tr1.TypeModifiers()).NotTo(BeNil())
			Expect(tr1.TypeModifiers().AllQUESTION()).To(HaveLen(1))
		})

		It("Should parse qualified type references", func() {
			schema, diag := parser.Parse(`
				Range struct {
					channel channel.Channel
					labels label.Label[]
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			fields := structDef.StructBody().AllFieldDef()

			// channel.Channel
			tr0 := asTypeRefNormal(fields[0].TypeRef())
			qualIdent := tr0.QualifiedIdent()
			Expect(qualIdent.IDENT(0).GetText()).To(Equal("channel"))
			Expect(qualIdent.IDENT(1).GetText()).To(Equal("Channel"))

			// label.Label[]
			tr1 := asTypeRefNormal(fields[1].TypeRef())
			qualIdent2 := tr1.QualifiedIdent()
			Expect(qualIdent2.IDENT(0).GetText()).To(Equal("label"))
			Expect(qualIdent2.IDENT(1).GetText()).To(Equal("Label"))
			Expect(tr1.ArrayModifier()).NotTo(BeNil())
		})

		It("Should parse map type fields", func() {
			schema, diag := parser.Parse(`
				Config struct {
					settings map<string, string>
					counts map<string, uint32>?
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			fields := structDef.StructBody().AllFieldDef()
			Expect(fields).To(HaveLen(2))

			// settings map<string, string>
			tr0, ok := fields[0].TypeRef().(*parser.TypeRefMapContext)
			Expect(ok).To(BeTrue())
			mapType := tr0.MapType()
			Expect(mapType).NotTo(BeNil())
			typeRefs := mapType.AllTypeRef()
			Expect(typeRefs).To(HaveLen(2))
			keyType := asTypeRefNormal(typeRefs[0])
			Expect(keyType.QualifiedIdent().IDENT(0).GetText()).To(Equal("string"))
			valueType := asTypeRefNormal(typeRefs[1])
			Expect(valueType.QualifiedIdent().IDENT(0).GetText()).To(Equal("string"))

			// counts map<string, uint32>?
			tr1, ok := fields[1].TypeRef().(*parser.TypeRefMapContext)
			Expect(ok).To(BeTrue())
			Expect(tr1.TypeModifiers()).NotTo(BeNil())
			Expect(tr1.TypeModifiers().AllQUESTION()).To(HaveLen(1))
		})
	})

	Describe("Inline Field Domains", func() {
		It("Should parse a field with an inline domain", func() {
			schema, diag := parser.Parse(`
				Range struct {
					key uuid @key
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			field := structDef.StructBody().FieldDef(0)
			inlineDomains := field.AllInlineDomain()
			Expect(inlineDomains).To(HaveLen(1))
			Expect(inlineDomains[0].IDENT().GetText()).To(Equal("key"))
			Expect(inlineDomains[0].DomainContent()).To(BeNil())
		})

		It("Should parse a field with inline domain with value", func() {
			schema, diag := parser.Parse(`
				Range struct {
					name string @validate required
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			field := structDef.StructBody().FieldDef(0)
			inlineDomains := field.AllInlineDomain()
			Expect(inlineDomains).To(HaveLen(1))
			Expect(inlineDomains[0].IDENT().GetText()).To(Equal("validate"))
			content := inlineDomains[0].DomainContent()
			Expect(content).NotTo(BeNil())
			// Single expression (not a block)
			expr := content.Expression()
			Expect(expr).NotTo(BeNil())
			Expect(expr.IDENT().GetText()).To(Equal("required"))
		})

		It("Should parse a field with multiple inline domains", func() {
			schema, diag := parser.Parse(`
				Range struct {
					name string @validate required @query eq
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			field := structDef.StructBody().FieldDef(0)
			inlineDomains := field.AllInlineDomain()
			Expect(inlineDomains).To(HaveLen(2))
			Expect(inlineDomains[0].IDENT().GetText()).To(Equal("validate"))
			Expect(inlineDomains[1].IDENT().GetText()).To(Equal("query"))
		})

		It("Should parse an inline domain with a block", func() {
			schema, diag := parser.Parse(`
				Range struct {
					name string @validate {
						required
						max_length 255
						min_length 1
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			field := structDef.StructBody().FieldDef(0)
			inlineDomains := field.AllInlineDomain()
			Expect(inlineDomains).To(HaveLen(1))

			content := inlineDomains[0].DomainContent()
			Expect(content).NotTo(BeNil())
			block := content.DomainBlock()
			Expect(block).NotTo(BeNil())
			exprs := block.AllExpression()
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

		It("Should parse domain expressions with string literals", func() {
			schema, diag := parser.Parse(`
				User struct {
					name string @validate {
						default "untitled"
						pattern "[a-z]+"
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			field := structDef.StructBody().FieldDef(0)
			content := field.InlineDomain(0).DomainContent()
			block := content.DomainBlock()
			exprs := block.AllExpression()

			// default "untitled"
			Expect(exprs[0].IDENT().GetText()).To(Equal("default"))
			Expect(exprs[0].ExpressionValue(0).STRING_LIT().GetText()).To(Equal(`"untitled"`))

			// pattern "[a-z]+"
			Expect(exprs[1].IDENT().GetText()).To(Equal("pattern"))
			Expect(exprs[1].ExpressionValue(0).STRING_LIT().GetText()).To(Equal(`"[a-z]+"`))
		})

		It("Should parse domain expressions with identifier values", func() {
			schema, diag := parser.Parse(`
				Range struct {
					labels uuid[] @relation {
						target label.Label
						cardinality many_to_many
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			field := structDef.StructBody().FieldDef(0)
			content := field.InlineDomain(0).DomainContent()
			block := content.DomainBlock()
			exprs := block.AllExpression()

			// target label.Label
			Expect(exprs[0].IDENT().GetText()).To(Equal("target"))

			// cardinality many_to_many
			Expect(exprs[1].IDENT().GetText()).To(Equal("cardinality"))
			qualIdent := exprs[1].ExpressionValue(0).QualifiedIdent()
			Expect(qualIdent.IDENT(0).GetText()).To(Equal("many_to_many"))
		})
	})

	Describe("Field Body Domains", func() {
		It("Should parse a field with a body containing domains", func() {
			schema, diag := parser.Parse(`
				Range struct {
					name string {
						@validate {
							required
						}
						@query eq
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			field := structDef.StructBody().FieldDef(0)
			fieldBody := field.FieldBody()
			Expect(fieldBody).NotTo(BeNil())
			domains := fieldBody.AllDomain()
			Expect(domains).To(HaveLen(2))
			Expect(domains[0].IDENT().GetText()).To(Equal("validate"))
			Expect(domains[1].IDENT().GetText()).To(Equal("query"))
		})
	})

	Describe("Struct-Level Domains", func() {
		It("Should parse struct-level domains", func() {
			schema, diag := parser.Parse(`
				Range struct {
					key uuid
					name string

					@index {
						composite name created_at sorted
						unique name workspace
					}
				}
			`)
			Expect(diag).To(BeNil())
			structDef := asStructFull(schema.Definition(0).StructDef())
			structBody := structDef.StructBody()

			// Should have 2 fields and 1 domain
			Expect(structBody.AllFieldDef()).To(HaveLen(2))
			Expect(structBody.AllDomain()).To(HaveLen(1))

			domain := structBody.Domain(0)
			Expect(domain.IDENT().GetText()).To(Equal("index"))
			Expect(domain.DomainContent().DomainBlock().AllExpression()).To(HaveLen(2))
		})
	})

	Describe("Enum Definitions", func() {
		It("Should parse an enum with integer values", func() {
			schema, diag := parser.Parse(`
				TaskState enum {
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

			values := enumDef.EnumBody().AllEnumValue()
			Expect(values).To(HaveLen(4))
			Expect(values[0].IDENT().GetText()).To(Equal("pending"))
			Expect(values[0].INT_LIT().GetText()).To(Equal("0"))
			Expect(values[3].IDENT().GetText()).To(Equal("failed"))
			Expect(values[3].INT_LIT().GetText()).To(Equal("3"))
		})

		It("Should parse an enum with string values", func() {
			schema, diag := parser.Parse(`
				DataType enum {
					float32 = "float32"
					float64 = "float64"
					int32 = "int32"
				}
			`)
			Expect(diag).To(BeNil())
			enumDef := schema.Definition(0).EnumDef()
			values := enumDef.EnumBody().AllEnumValue()
			Expect(values).To(HaveLen(3))
			Expect(values[0].IDENT().GetText()).To(Equal("float32"))
			Expect(values[0].STRING_LIT().GetText()).To(Equal(`"float32"`))
		})
	})

	Describe("Struct Aliases", func() {
		It("Should parse a simple struct alias", func() {
			schema, diag := parser.Parse(`Status = status.Status<Details>`)
			Expect(diag).To(BeNil())
			Expect(schema.AllDefinition()).To(HaveLen(1))
			structDef := schema.Definition(0).StructDef()
			alias, ok := structDef.(*parser.StructAliasContext)
			Expect(ok).To(BeTrue())
			Expect(alias.IDENT().GetText()).To(Equal("Status"))
		})
	})

	Describe("Complete Schema", func() {
		It("Should parse a complete schema with new syntax", func() {
			schema, diag := parser.Parse(`
				import "schema/core/label"

				@ts output "client/ts/src/range"

				Range struct {
					key uuid @key

					name string @validate {
						required
						max_length 255
					}

					labels uuid[] @relation {
						target label.Label
						cardinality many_to_many
					}

					time_range time_range @validate required

					created_at timestamp @validate {
						default now
						immutable
					}

					parent uuid? @relation {
						target Range
						self
					}

					@ontology type "range"
				}
			`)
			Expect(diag).To(BeNil())
			Expect(schema.AllImportStmt()).To(HaveLen(1))
			Expect(schema.AllFileDomain()).To(HaveLen(1))
			Expect(schema.AllDefinition()).To(HaveLen(1))

			structDef := asStructFull(schema.Definition(0).StructDef())
			Expect(structDef.IDENT().GetText()).To(Equal("Range"))
			Expect(structDef.StructBody().AllFieldDef()).To(HaveLen(6))
			Expect(structDef.StructBody().AllDomain()).To(HaveLen(1))
		})
	})

	Describe("Error Handling", func() {
		It("Should report syntax errors", func() {
			_, diag := parser.Parse(`struct { }`) // name should be before struct
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
		})

		It("Should report errors for missing braces", func() {
			_, diag := parser.Parse(`Range struct key uuid`)
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
		})

		It("Should report errors for invalid enum value", func() {
			_, diag := parser.Parse(`
				State enum {
					pending
				}
			`) // missing = value
			Expect(diag).NotTo(BeNil())
			Expect(diag.HasErrors()).To(BeTrue())
		})
	})
})
