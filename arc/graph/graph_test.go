package graph_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
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

		It("Should correctly analyze a complete program", func() {
			g := arc.Graph{
				Stages: []ir.Stage{
					{
						Key: "on",
						Config: ir.NamedTypes{
							Keys:   []string{"channel"},
							Values: []ir.Type{ir.Chan{}},
						},
					},
					{
						Key:    "printer",
						Config: ir.NamedTypes{},
					},
				},
				Nodes: []graph.Node{
					{Node: arc.Node{
						Key:    "first",
						Type:   "on",
						Config: map[string]any{"channel": 12},
					}},
					{Node: arc.Node{Key: "printer", Type: "printer"}},
				},
				Edges: []arc.Edge{
					{
						Source: arc.Handle{Node: "first", Param: ""},
						Target: arc.Handle{Node: "printer", Param: ""},
					},
				},
			}
			resolver := ir.MapResolver{
				"12": ir.Symbol{
					Name: "ox_pt_1",
					Type: ir.Chan{ValueType: ir.F32{}},
					Kind: ir.KindChannel,
					ID:   12,
				},
			}
			g = MustSucceed(graph.Parse(g))
			inter, diagnostics := graph.Analyze(ctx, g, resolver)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Stages).To(HaveLen(2))
			Expect(inter.Nodes).To(HaveLen(2))
			Expect(inter.Edges).To(HaveLen(1))

			firstNode := inter.Nodes[0]
			Expect(firstNode.Key).To(Equal("first"))
			Expect(firstNode.Type).To(Equal("on"))
			Expect(firstNode.Config).To(HaveLen(1))
			Expect(firstNode.Channels.Read).To(HaveLen(1))
		})
	})
})
