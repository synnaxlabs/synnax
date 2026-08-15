// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

const renderFixture = `
Key = uuid {
	@key
	@doc value "is the unique identifier."
}

Size uint32 {
	@go marshal flex
}

Kind enum {
	linear = "linear"
	log = "log"
}

Sample struct {
	key Key @key
	kind Kind
	label string?
	weight float64 = 1.5
	tags string[]
	lookup map<string, int64>

	@go marshal
	@doc value """
		is a sample with one of every field shape.
	"""
}

Wrapper struct<D> {
	details D
}

Content union on variant {
	sample Sample
	wrapped Wrapper<Sample>
}
`

// analyzeFixture analyzes source as a version file: marshal tags are only
// legal there.
func analyzeFixture(ctx SpecContext, source string) *resolution.Table {
	GinkgoHelper()
	table := resolution.NewTable()
	diag := analyzer.AnalyzeSeeded(
		ctx, source, "schemas/synnax/versions/test/v0.oracle", "test",
		testutil.NewMockFileLoader(), table,
	)
	Expect(diag.Ok()).To(BeTrue(), func() string { return diag.String() })
	return table
}

func declsOf(table *resolution.Table) []versions.Decl {
	var decls []versions.Decl
	for _, t := range table.TypesInNamespace("test") {
		if t.Synthetic {
			continue
		}
		decls = append(decls, versions.Decl{Type: t})
	}
	return decls
}

var _ = Describe("Render", func() {
	It("Should render declarations that re-analyze to the same shapes", func(
		ctx SpecContext,
	) {
		table := analyzeFixture(ctx, renderFixture)
		rendered := versions.Render(declsOf(table), versions.RenderOptions{})
		formatted := MustSucceed(formatter.Format(rendered))
		Expect(MustSucceed(formatter.Format(formatted))).To(Equal(formatted))

		again := analyzeFixture(ctx, formatted)
		for _, t := range table.TypesInNamespace("test") {
			if t.Synthetic {
				continue
			}
			rt := MustBeOk(again.Get(t.QualifiedName))
			Expect(rt.Form).To(BeAssignableToTypeOf(t.Form), t.QualifiedName)
			switch f := t.Form.(type) {
			case resolution.StructForm:
				rf := rt.Form.(resolution.StructForm)
				Expect(rf.Fields).To(HaveLen(len(f.Fields)), t.QualifiedName)
				for i := range f.Fields {
					Expect(rf.Fields[i].Name).To(Equal(f.Fields[i].Name))
					Expect(rf.Fields[i].Type.Name).To(Equal(f.Fields[i].Type.Name))
					Expect(rf.Fields[i].Optional).To(Equal(f.Fields[i].Optional))
				}
			case resolution.EnumForm:
				rf := rt.Form.(resolution.EnumForm)
				Expect(rf.Values).To(HaveLen(len(f.Values)))
			case resolution.UnionForm:
				rf := rt.Form.(resolution.UnionForm)
				Expect(rf.Variants).To(HaveLen(len(f.Variants)))
				Expect(rf.Discriminator).To(Equal(f.Discriminator))
			}
		}
	})

	It("Should preserve unknown expressions and @doc through a round trip", func(
		ctx SpecContext,
	) {
		table := analyzeFixture(ctx, renderFixture)
		rendered := versions.Render(declsOf(table), versions.RenderOptions{})
		again := analyzeFixture(ctx, MustSucceed(formatter.Format(rendered)))
		size := MustBeOk(again.Get("test.Size"))
		_, marshal := size.Domains["go"].Expressions.Find("marshal")
		Expect(marshal).To(BeTrue())
		sample := MustBeOk(again.Get("test.Sample"))
		doc := sample.Domains["doc"]
		Expect(doc.Expressions[0].Values[0].StringValue).
			To(ContainSubstring("one of every field shape"))
	})

	It("Should render alias lines and sorted imports", func(ctx SpecContext) {
		table := analyzeFixture(ctx, "Key = uuid\n")
		decls := []versions.Decl{
			{Type: MustBeOk(table.Get("test.Key"))},
			{
				Type:  resolution.Type{Name: "Sample"},
				Alias: &versions.Alias{Version: 2, Name: "Sample"},
			},
		}
		rendered := versions.Render(decls, versions.RenderOptions{
			Imports: []string{
				"schemas/x/versions/telem/v1",
				"schemas/synnax/versions/node/v0",
			},
		})
		Expect(rendered).To(ContainSubstring(
			"import \"schemas/synnax/versions/node/v0\"\nimport \"schemas/x/versions/telem/v1\"",
		))
		Expect(rendered).To(ContainSubstring("Sample = v2.Sample"))
	})

	It("Should filter domains and rewrite qualifiers", func(ctx SpecContext) {
		source := `
Entry struct {
	key uuid @key
	name string

	@go marshal
	@go output "core/pkg/service/entry"
	@ts hand
}
`
		table := analyzeFixture(ctx, source)
		rendered := versions.Render(
			[]versions.Decl{{Type: MustBeOk(table.Get("test.Entry"))}},
			versions.RenderOptions{
				KeepExpression: func(domain string, _ resolution.Expression) bool {
					return domain == "go" || domain == "key"
				},
			},
		)
		Expect(rendered).To(ContainSubstring("@go marshal"))
		Expect(rendered).To(ContainSubstring("@go output"))
		Expect(rendered).ToNot(ContainSubstring("@ts"))
	})

	It("Should render domain values of every kind", func() {
		size := int64(16)
		t := resolution.Type{
			Name: "Sample",
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name: "buf",
					Type: resolution.TypeRef{
						Name:      "Array",
						TypeArgs:  []resolution.TypeRef{{Name: "uint8"}},
						ArraySize: &size,
					},
				}},
			},
			Domains: map[string]resolution.Domain{
				"meta": {Name: "meta", Expressions: resolution.Expressions{{
					Name: "mixed",
					Values: []resolution.ExpressionValue{
						{Kind: resolution.ValueKindInt, IntValue: 7},
						{Kind: resolution.ValueKindFloat, FloatValue: 2.5},
						{Kind: resolution.ValueKindBool, BoolValue: true},
						{
							Kind:        resolution.ValueKindIdent,
							IdentValue:  "ns.thing",
							StringValue: "",
						},
						{
							Kind: resolution.ValueKindArray,
							Elements: []resolution.ExpressionValue{
								{Kind: resolution.ValueKindString, StringValue: "a"},
							},
						},
						{
							Kind: resolution.ValueKindStruct,
							Fields: []resolution.StructFieldValue{{
								Name: "x",
								Value: resolution.ExpressionValue{
									Kind:        resolution.ValueKindString,
									StringValue: "line1\nline2",
								},
							}},
						},
						{Kind: resolution.ValueKind(99)},
					},
				}}},
			},
		}
		rendered := versions.Render(
			[]versions.Decl{{Type: t}},
			versions.RenderOptions{
				Qualifier: func(ns string) string { return "other" },
			},
		)
		Expect(rendered).To(ContainSubstring("buf uint8[16]"))
		Expect(rendered).To(ContainSubstring(
			`@meta mixed 7 2.5 true other.thing ["a"] {x = """line1` + "\n" +
				`line2"""}`,
		))
	})

	It("Should render type parameters with modifiers", func() {
		constraint := resolution.TypeRef{Name: "record"}
		def := resolution.TypeRef{Name: "record"}
		t := resolution.Type{
			Name: "Wrapper",
			Form: resolution.AliasForm{
				Target: resolution.TypeRef{
					Name: "Map",
					TypeArgs: []resolution.TypeRef{
						{Name: "string"},
						{TypeParam: &resolution.TypeParam{Name: "D"}},
					},
				},
				TypeParams: []resolution.TypeParam{{
					Name:       "D",
					Optional:   true,
					Constraint: &constraint,
					Default:    &def,
				}},
			},
		}
		rendered := versions.Render(
			[]versions.Decl{{Type: t}}, versions.RenderOptions{},
		)
		Expect(rendered).To(ContainSubstring(
			"Wrapper<D? extends record = record> = map<string, D>",
		))
	})

	It("Should render union variant domain bodies", func() {
		payload := resolution.Type{
			Name:      "payload",
			Synthetic: true,
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name: "depth",
					Type: resolution.TypeRef{Name: "int64"},
				}},
			},
		}
		t := resolution.Type{
			Name: "Node",
			Form: resolution.UnionForm{
				Discriminator: "variant",
				Variants: []resolution.UnionVariant{
					{
						Name: "leaf",
						Type: resolution.TypeRef{Name: "Leaf"},
						Domains: map[string]resolution.Domain{
							"doc": {Name: "doc", Expressions: resolution.Expressions{{
								Name: "value",
								Values: []resolution.ExpressionValue{{
									Kind:        resolution.ValueKindString,
									StringValue: "is a leaf.",
								}},
							}}},
						},
					},
					{
						Name:   "inline",
						Type:   resolution.TypeRef{Name: "payload"},
						Inline: true,
						Domains: map[string]resolution.Domain{
							"doc": {Name: "doc", Expressions: resolution.Expressions{{
								Name: "value",
								Values: []resolution.ExpressionValue{{
									Kind:        resolution.ValueKindString,
									StringValue: "is inline.",
								}},
							}}},
						},
					},
				},
			},
		}
		rendered := versions.Render(
			[]versions.Decl{{Type: t}},
			versions.RenderOptions{
				Resolve: func(name string) (resolution.Type, bool) {
					if name == "payload" {
						return payload, true
					}
					return resolution.Type{}, false
				},
			},
		)
		Expect(rendered).To(ContainSubstring("leaf Leaf {"))
		Expect(rendered).To(ContainSubstring(`@doc value "is a leaf."`))
		Expect(rendered).To(ContainSubstring("inline {"))
		Expect(rendered).To(ContainSubstring("depth int64"))
		Expect(rendered).To(ContainSubstring(`@doc value "is inline."`))
	})

	It("Should append extra lines to single-line declarations", func() {
		t := resolution.Type{
			Name: "Span",
			Form: resolution.DistinctForm{Base: resolution.TypeRef{Name: "int64"}},
		}
		rendered := versions.Render(
			[]versions.Decl{{Type: t}},
			versions.RenderOptions{
				ExtraTypeLines: func(name string) []string {
					return []string{"@ts to_number"}
				},
			},
		)
		Expect(rendered).To(ContainSubstring("Span int64 {"))
		Expect(rendered).To(ContainSubstring("@ts to_number"))
	})
})
