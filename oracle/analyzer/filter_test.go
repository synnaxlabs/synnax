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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func fieldTypes(form resolution.StructForm) map[string]resolution.TypeRef {
	return lo.SliceToMap(form.Fields, func(f resolution.Field) (string, resolution.TypeRef) {
		return f.Name, f.Type
	})
}

func fieldNames(form resolution.StructForm) []string {
	return lo.Map(form.Fields, func(f resolution.Field, _ int) string { return f.Name })
}

var _ = Describe("Filter definitions", func() {
	var loader *MockFileLoader

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/filter", FilterSchema)
	})

	Describe("Expansion", func() {
		const source = `
			import "schemas/x/filter"

			Key = uuid
			Kind enum {
				a = "a"
				b = "b"
			}
			Span struct {
				start int64
			}
			Thing struct {
				key    Key      { @key }
				name   string
				active bool
				count  int32
				kind   Kind
				tags   string[]
				span   Span
			}
			Where filter Thing {
				key
				name
				overlaps Span {
					@doc value "matches things whose span overlaps the given span."
				}
				active
				count
				kind
				tags
			}
		`

		It("Should expand the tree fields and lines in declared order", func(
			ctx SpecContext,
		) {
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			form := table.MustGet("test.Where").Form.(resolution.StructForm)
			Expect(fieldNames(form)).To(Equal([]string{
				"and", "or", "not", "key", "name", "overlaps", "active", "count",
				"kind", "tags",
			}))
			Expect(lo.EveryBy(form.Fields, func(f resolution.Field) bool {
				return f.Optional
			})).To(BeTrue())
		})

		It("Should pick each bare line's operator from its field type", func(
			ctx SpecContext,
		) {
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			types := fieldTypes(table.MustGet("test.Where").Form.(resolution.StructForm))
			Expect(types["and"]).To(Equal(resolution.TypeRef{
				Name:     "Array",
				TypeArgs: []resolution.TypeRef{{Name: "test.Where"}},
			}))
			Expect(types["not"]).To(Equal(resolution.TypeRef{Name: "test.Where"}))
			Expect(types["key"]).To(Equal(resolution.TypeRef{
				Name:     "filter.Set",
				TypeArgs: []resolution.TypeRef{{Name: "test.Key"}},
			}))
			Expect(types["name"]).To(Equal(resolution.TypeRef{Name: "filter.String"}))
			Expect(types["overlaps"]).To(Equal(resolution.TypeRef{Name: "test.Span"}))
			Expect(types["active"]).To(Equal(resolution.TypeRef{Name: "filter.Bool"}))
			Expect(types["count"]).To(Equal(resolution.TypeRef{
				Name:     "filter.Ordered",
				TypeArgs: []resolution.TypeRef{{Name: "int32"}},
			}))
			Expect(types["kind"]).To(Equal(resolution.TypeRef{
				Name:     "filter.Set",
				TypeArgs: []resolution.TypeRef{{Name: "test.Kind"}},
			}))
			Expect(types["tags"]).To(Equal(resolution.TypeRef{
				Name:     "filter.Contains",
				TypeArgs: []resolution.TypeRef{{Name: "string"}},
			}))
		})

		It("Should record the target and lines", func(ctx SpecContext) {
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			where := table.MustGet("test.Where")
			Expect(where.Filter).ToNot(BeNil())
			Expect(where.Filter.Target).To(Equal(resolution.TypeRef{Name: "test.Thing"}))
			overlaps := MustBeOk(where.Filter.Line("overlaps"))
			Expect(overlaps.Bare()).To(BeFalse())
			Expect(overlaps.Type).To(Equal(resolution.TypeRef{Name: "test.Span"}))
			Expect(MustBeOk(where.Filter.Line("name")).Bare()).To(BeTrue())
		})

		It("Should keep a line's doc and give a bare line one", func(ctx SpecContext) {
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			form := table.MustGet("test.Where").Form.(resolution.StructForm)
			doc := func(name string) string {
				f := MustBeOk(form.Field(name))
				return f.Domains["doc"].Expressions[0].Values[0].StringValue
			}
			Expect(doc("overlaps")).To(Equal(
				"matches things whose span overlaps the given span.",
			))
			Expect(doc("name")).To(Equal("selects by name."))
			Expect(doc("and")).To(Equal("matches when every child matches."))
		})

		It("Should mark the expanded struct recursive", func(ctx SpecContext) {
			table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			form := table.MustGet("test.Where").Form.(resolution.StructForm)
			Expect(form.IsRecursive).To(BeTrue())
		})

		It("Should expand over the base fields of a union", func(ctx SpecContext) {
			table, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				Base struct {
					key  uuid { @key }
					name string
				}
				Thing union on variant extends Base {
					a {}
					b {}
				}
				Where filter Thing {
					key
					name
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			types := fieldTypes(table.MustGet("test.Where").Form.(resolution.StructForm))
			Expect(types["name"]).To(Equal(resolution.TypeRef{Name: "filter.String"}))
		})

		It("Should type a bare line by its type parameter's default", func(
			ctx SpecContext,
		) {
			table, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				Kind enum {
					a = "a"
				}
				Thing struct<D?, K extends Kind = Kind> {
					key     uuid { @key }
					kind    K
					details D
				}
				Where filter Thing {
					kind
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			types := fieldTypes(table.MustGet("test.Where").Form.(resolution.StructForm))
			Expect(types["kind"]).To(Equal(resolution.TypeRef{
				Name:     "filter.Set",
				TypeArgs: []resolution.TypeRef{{Name: "test.Kind"}},
			}))
		})

		It("Should type a key line by a computed retrieve key", func(ctx SpecContext) {
			table, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				Key uint32
				Thing struct {
					name string

					@retrieve key "Key"
				}
				Where filter Thing {
					key
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			types := fieldTypes(table.MustGet("test.Where").Form.(resolution.StructForm))
			Expect(types["key"]).To(Equal(resolution.TypeRef{
				Name:     "filter.Set",
				TypeArgs: []resolution.TypeRef{{Name: "test.Key"}},
			}))
		})

		It("Should count the filter import as used by a filter", func(
			ctx SpecContext,
		) {
			_, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				Thing struct {
					key  uuid { @key }
				}
				Where filter Thing {
					key
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})

		It("Should not need the filter import for typed lines only", func(
			ctx SpecContext,
		) {
			_, diag := analyzer.AnalyzeSource(ctx, `
				Thing struct {
					key uuid { @key }
				}
				Where filter Thing {
					labels uuid[]
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})
	})

	Describe("Errors", func() {
		DescribeTable("Should reject an invalid definition",
			func(ctx SpecContext, body, message string) {
				_, diag := analyzer.AnalyzeSource(ctx, `
					import "schemas/x/filter"

					Span struct {
						start int64
					}
					Thing struct<D> {
						key     uuid { @key }
						name    string
						color   string?
						span    Span
						details D
					}
					Other enum {
						a = "a"
					}
				`+body, "test", loader)
				Expect(diag.Ok()).To(BeFalse())
				Expect(diag.String()).To(ContainSubstring(message))
			},
			Entry("a keyword other than filter",
				"Where filters Thing {\n key\n}",
				`Where: expected 'filter' before the target, got "filters"`),
			Entry("a line naming no field",
				"Where filter Thing {\n missing\n}",
				"filter Where: Thing has no field missing; give the line a type"),
			Entry("an optional line",
				"Where filter Thing {\n name?\n}",
				"filter Where: line name is always optional; remove the ?"),
			Entry("a line with a default",
				"Where filter Thing {\n flag bool = true\n}",
				"filter Where: line flag cannot have a default"),
			Entry("a reserved line",
				"Where filter Thing {\n and\n}",
				`filter Where: "and" is reserved for the tree`),
			Entry("a duplicate line",
				"Where filter Thing {\n name\n name\n}",
				`filter Where: duplicate line "name"`),
			Entry("a bare line over a field without operators",
				"Where filter Thing {\n span\n}",
				"filter Where: Thing.span has no operators for its type"),
			Entry("a bare line over an optional field",
				"Where filter Thing {\n color\n}",
				"filter Where: Thing.color is optional, which bare lines do not"),
			Entry("a bare line over a field typed by a parameter without a default",
				"Where filter Thing {\n details\n}",
				"filter Where: Thing.details has the type parameter D, which has no "+
					"default"),
			Entry("a target that is not a struct or a union",
				"Where filter Other {\n key\n}",
				"filter Where: the target Other must be a struct or a union"),
			Entry("an unknown target",
				"Where filter Missing {\n key\n}",
				"filter Where: unknown target Missing"),
			Entry("a generic target reference",
				"Where filter Thing<string> {\n key\n}",
				"filter Where: the target must be a plain type reference"),
			Entry("two filters in one schema",
				"Where filter Thing {\n key\n}\nOther2 filter Thing {\n key\n}",
				"declares filters Where and Other2; a schema declares at most one"),
		)

		It("Should reject bare lines without the filter import", func(
			ctx SpecContext,
		) {
			_, diag := analyzer.AnalyzeSource(ctx, `
				Thing struct {
					key uuid { @key }
				}
				Where filter Thing {
					key
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring(
				"filter Where: bare lines need an import of the filter schema",
			))
		})

		It("Should reject an order without the filter import", func(ctx SpecContext) {
			_, diag := analyzer.AnalyzeSource(ctx, `
				Thing struct {
					key  uuid  { @key }
					time int64 { @index sorted }
				}
				Where filter Thing {
					labels uuid[]
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring(
				"filter Where: order_by needs an import of the filter schema",
			))
		})

		It("Should reject a target in another schema", func(ctx SpecContext) {
			loader.Add("schemas/dep", "Thing struct {\n key uuid { @key }\n}\n")
			_, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/dep"
				import "schemas/x/filter"

				Where filter dep.Thing {
					key
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring(
				"filter Where: the target dep.Thing must be declared in the same schema",
			))
		})
	})

	Describe("Retrieve request", func() {
		It("Should derive the request and response", func(ctx SpecContext) {
			table, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				@go output "out"

				Label struct {
					key uuid { @key }
				}
				Thing struct {
					key    uuid     { @key }
					name   string   { @search }
					labels Label[]? { @include }
				}
				Where filter Thing {
					key
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			req := table.MustGet("test.RetrieveRequest")
			form := req.Form.(resolution.StructForm)
			Expect(fieldNames(form)).To(Equal([]string{
				"where", "search_term", "limit", "offset", "include_labels",
			}))
			types := fieldTypes(form)
			Expect(types["where"]).To(Equal(resolution.TypeRef{Name: "test.Where"}))
			Expect(types["limit"]).To(Equal(resolution.TypeRef{Name: "uint32"}))
			Expect(types["include_labels"]).To(Equal(resolution.TypeRef{Name: "bool"}))
			Expect(req.Domains).To(HaveKey("go"))
			res := table.MustGet("test.RetrieveResponse").Form.(resolution.StructForm)
			Expect(fieldTypes(res)).To(Equal(map[string]resolution.TypeRef{
				"things": {
					Name:     "Array",
					TypeArgs: []resolution.TypeRef{{Name: "test.Thing"}},
				},
			}))
			Expect(res.Fields[0].Optional).To(BeFalse())
		})

		It("Should omit the request wherever the filter is omitted", func(
			ctx SpecContext,
		) {
			table, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				@go output "out"
				@cpp output "out"

				Thing struct {
					key  uuid  { @key }
					time int64 { @index sorted }
				}
				Where filter Thing {
					key

					@cpp omit
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			for _, name := range []string{"RetrieveRequest", "Order", "OrderField"} {
				typ := table.MustGet("test." + name)
				_, omitted := typ.Domains["cpp"].Expressions.Find("omit")
				Expect(omitted).To(BeTrue(), name)
			}
			res := table.MustGet("test.RetrieveResponse")
			_, omitted := res.Domains["cpp"].Expressions.Find("omit")
			Expect(omitted).To(BeFalse())
		})

		It("Should leave out search_term when no field has @search", func(
			ctx SpecContext,
		) {
			table, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				Thing struct {
					key uuid { @key }
				}
				Where filter Thing {
					key
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			form := table.MustGet("test.RetrieveRequest").Form.(resolution.StructForm)
			Expect(fieldNames(form)).To(Equal([]string{"where", "limit", "offset"}))
		})

		It("Should derive an order over sorted fields", func(ctx SpecContext) {
			table, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				Thing struct {
					key  uuid   { @key }
					name string { @index sorted }
					time int64  { @index sorted }
				}
				Where filter Thing {
					key
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			form := table.MustGet("test.RetrieveRequest").Form.(resolution.StructForm)
			Expect(fieldTypes(form)["order_by"]).To(Equal(
				resolution.TypeRef{Name: "test.Order"},
			))
			order := table.MustGet("test.Order").Form.(resolution.StructForm)
			Expect(fieldTypes(order)).To(Equal(map[string]resolution.TypeRef{
				"field":     {Name: "test.OrderField"},
				"direction": {Name: "filter.Direction"},
			}))
			field := table.MustGet("test.OrderField").Form.(resolution.EnumForm)
			Expect(lo.Map(field.Values, func(v resolution.EnumValue, _ int) string {
				return v.StringValue()
			})).To(Equal([]string{"name", "time"}))
		})

		It("Should hold a generic target over any", func(ctx SpecContext) {
			table, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				Kind enum {
					a = "a"
				}
				Thing struct<D?, K extends Kind = Kind> {
					key     uuid { @key }
					details D
					kind    K
				}
				Where filter Thing {
					key
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			res := table.MustGet("test.RetrieveResponse").Form.(resolution.StructForm)
			Expect(fieldTypes(res)["things"]).To(Equal(resolution.TypeRef{
				Name: "Array",
				TypeArgs: []resolution.TypeRef{{
					Name:     "test.Thing",
					TypeArgs: []resolution.TypeRef{{Name: "any"}},
				}},
			}))
		})

		It("Should take the response type and its outputs from @response", func(
			ctx SpecContext,
		) {
			table, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				@go output "service"

				Thing struct {
					key uuid { @key }
				}
				Wire struct {
					key uuid

					@go output "api"
				}
				Where filter Thing {
					key

					@response type Wire
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			res := table.MustGet("test.RetrieveResponse")
			Expect(fieldTypes(res.Form.(resolution.StructForm))["things"]).To(Equal(
				resolution.TypeRef{
					Name:     "Array",
					TypeArgs: []resolution.TypeRef{{Name: "test.Wire"}},
				},
			))
			output := MustBeOk(res.Domains["go"].Expressions.Find("output"))
			Expect(output.Values[0].StringValue).To(Equal("api"))
			req := table.MustGet("test.RetrieveRequest")
			output = MustBeOk(req.Domains["go"].Expressions.Find("output"))
			Expect(output.Values[0].StringValue).To(Equal("service"))
		})

		It("Should reject a derived name the schema already declares", func(
			ctx SpecContext,
		) {
			_, diag := analyzer.AnalyzeSource(ctx, `
				import "schemas/x/filter"

				Thing struct {
					key uuid { @key }
				}
				RetrieveRequest struct {
					keys uuid[]
				}
				Where filter Thing {
					key
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring(
				"filter Where: Where derives test.RetrieveRequest, which the schema " +
					"already declares",
			))
		})

		It("Should not derive a request in a version file", func(ctx SpecContext) {
			table := resolution.NewTable()
			diag := analyzer.AnalyzeSeeded(ctx, `
import "schemas/x/filter"

Thing struct {
	key uuid {
		@key
	}
}

Where filter Thing {
	key
}
`, "schemas/synnax/versions/thing/v0.oracle", "v0", loader, table)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			MustBeOk(table.Get("v0.Where"))
			_, ok := table.Get("v0.RetrieveRequest")
			Expect(ok).To(BeFalse())
		})
	})
})
