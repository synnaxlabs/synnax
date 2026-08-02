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
	@go pinned
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

func analyzeFixture(ctx SpecContext, source string) *resolution.Table {
	table, diag := analyzer.AnalyzeSource(
		ctx, source, "test", testutil.NewMockFileLoader(),
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
				Expect(len(rf.Fields)).To(Equal(len(f.Fields)), t.QualifiedName)
				for i := range f.Fields {
					Expect(rf.Fields[i].Name).To(Equal(f.Fields[i].Name))
					Expect(rf.Fields[i].Type.Name).To(Equal(f.Fields[i].Type.Name))
					Expect(rf.Fields[i].Optional).To(Equal(f.Fields[i].Optional))
				}
			case resolution.EnumForm:
				rf := rt.Form.(resolution.EnumForm)
				Expect(len(rf.Values)).To(Equal(len(f.Values)))
			case resolution.UnionForm:
				rf := rt.Form.(resolution.UnionForm)
				Expect(len(rf.Variants)).To(Equal(len(f.Variants)))
				Expect(rf.Discriminator).To(Equal(f.Discriminator))
			}
		}
	})

	It("Should preserve @go pinned and @doc through a round trip", func(
		ctx SpecContext,
	) {
		table := analyzeFixture(ctx, renderFixture)
		rendered := versions.Render(declsOf(table), versions.RenderOptions{})
		again := analyzeFixture(ctx, MustSucceed(formatter.Format(rendered)))
		size := MustBeOk(again.Get("test.Size"))
		_, pinned := size.Domains["go"].Expressions.Find("pinned")
		Expect(pinned).To(BeTrue())
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
})
