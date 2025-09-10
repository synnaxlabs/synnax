package graph_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Graph", func() {
	Describe("Parse", func() {
		It("Should correctly parse a single stage", func() {
			g := graph.Graph{
				Stages: []ir.Stage{
					{
						Key: "add",
						Params: ir.NamedTypes{
							Keys:   []string{"a", "b"},
							Values: []ir.Type{ir.I64{}, ir.I64{}},
						},
						Return: ir.I64{},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			Expect(g.Stages[0].Body.AST).ToNot(BeNil())
		})

		It("Should correctly parse a single function", func() {
			g := graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Params: ir.NamedTypes{
							Keys:   []string{"a", "b"},
							Values: []ir.Type{ir.I64{}, ir.I64{}},
						},
						Return: ir.I64{},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			Expect(g.Functions[0].Body.AST).ToNot(BeNil())
		})
	})

	Describe("Analyze", func() {
		It("Should correctly analyze a single stage", func() {
			g := graph.Graph{
				Stages: []ir.Stage{
					{
						Key: "add",
						Params: ir.NamedTypes{
							Keys:   []string{"a", "b"},
							Values: []ir.Type{ir.I64{}, ir.I64{}},
						},
						Return: ir.I64{},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Stages).To(HaveLen(1))
			stageScope := MustSucceed(inter.Symbols.Resolve(ctx, "add"))
			Expect(stageScope.Children).To(HaveLen(3))
			params := stageScope.FilterChildrenByKind(ir.KindParam)
			Expect(params).To(HaveLen(2))
			Expect(params[0].Name).To(Equal("a"))
			Expect(params[0].Type).To(Equal(ir.I64{}))
			Expect(params[1].Name).To(Equal("b"))
			Expect(params[1].Type).To(Equal(ir.I64{}))
		})

		It("Should correctly analyze a single function", func() {
			g := graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Params: ir.NamedTypes{
							Keys:   []string{"a", "b"},
							Values: []ir.Type{ir.I64{}, ir.I64{}},
						},
						Return: ir.I64{},
						Body: ir.Body{Raw: `{
							return a + b
						}`},
					},
				},
			}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Functions).To(HaveLen(1))
			funcScope := MustSucceed(inter.Symbols.Resolve(ctx, "add"))
			Expect(funcScope.Children).To(HaveLen(3))
			params := funcScope.FilterChildrenByKind(ir.KindParam)
			Expect(params).To(HaveLen(2))
			Expect(params[0].Name).To(Equal("a"))
			Expect(params[0].Type).To(Equal(ir.I64{}))
			Expect(params[1].Name).To(Equal("b"))
			Expect(params[1].Type).To(Equal(ir.I64{}))
		})
	})
})
