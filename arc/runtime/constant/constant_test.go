package constant_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var ctx = context.Background()

var _ = Describe("Constant", func() {
	Describe("NewFactory", func() {
		It("Should create factory", func() {
			factory := constant.NewFactory()
			Expect(factory).ToNot(BeNil())
		})
	})
	Describe("Factory.Create", func() {
		var factory node.Factory
		var s *state.State
		BeforeEach(func() {
			factory = constant.NewFactory()
			s = state.New(state.Config{
				Edges: ir.Edges{
					{Source: ir.Handle{
						Node:  "const",
						Param: ir.DefaultOutputParam,
					}, Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam}},
				},
				Nodes: ir.Nodes{
					{
						Key:  "const",
						Type: "constant",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
				},
			})
		})
		It("Should create node for constant type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": 42}},
				State: s.Node("const"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})
		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("const"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
		})
		It("Should handle float64 value", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": 3.14}},
				State: s.Node("const"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})
		It("Should handle int value", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": 100}},
				State: s.Node("const"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})
		It("Should handle uint8 value", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": uint8(255)}},
				State: s.Node("const"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})
	})
	Describe("constant.Init", func() {
		var s *state.State
		var factory node.Factory
		var outputs []string
		BeforeEach(func() {
			factory = constant.NewFactory()
			s = state.New(state.Config{
				Edges: ir.Edges{
					{Source: ir.Handle{Node: "const", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam}},
				},
				Nodes: ir.Nodes{
					{
						Key:  "const",
						Type: "constant",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
				},
			})
			outputs = []string{}
		})
		It("Should emit output on Init with int value", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": 42}},
				State: s.Node("const"),
			}
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(output string) {
				outputs = append(outputs, output)
			})
			Expect(outputs).To(HaveLen(1))
			Expect(outputs[0]).To(Equal(ir.DefaultOutputParam))
		})
		It("Should set output data on Init", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": int64(100)}},
				State: s.Node("const"),
			}
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(string) {})
			out := s.Node("const").Output(0)
			Expect(out.Len()).To(Equal(int64(1)))
		})
		It("Should set output time on Init", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type:         "constant",
					ConfigValues: map[string]interface{}{"value": 3.14},
				},
				State: s.Node("const"),
			}
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(string) {})
			outTime := s.Node("const").OutputTime(0)
			Expect(outTime.Len()).To(Equal(int64(1)))
			times := telem.UnmarshalSeries[telem.TimeStamp](*outTime)
			Expect(times[0]).To(BeNumerically(">", int64(0)))
		})
		It("Should handle float64 constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": 2.718}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[float64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(string) {})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[float64](*out)
			Expect(vals[0]).To(Equal(2.718))
		})
		It("Should handle int32 constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": int32(42)}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int32](0)
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(string) {})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[int32](*out)
			Expect(vals[0]).To(Equal(int32(42)))
		})
		It("Should handle uint8 constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": uint8(255)}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[uint8](0)
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(string) {})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[uint8](*out)
			Expect(vals[0]).To(Equal(uint8(255)))
		})
		It("Should allow downstream nodes to read constant", func() {
			s = state.New(state.Config{
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "const", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
					},
				},
				Nodes: ir.Nodes{
					{
						Key:  "const",
						Type: "constant",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{Key: "sink"},
				},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": int64(999)}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(string) {})
			sink := s.Node("sink")
			recalc := sink.RefreshInputs()
			Expect(recalc).To(BeTrue())
			input := sink.Input(0)
			vals := telem.UnmarshalSeries[int64](input)
			Expect(vals[0]).To(Equal(int64(999)))
		})
		It("Should handle zero value constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": 0}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(string) {})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[int64](*out)
			Expect(vals[0]).To(Equal(int64(0)))
		})
		It("Should handle negative value constant", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": -42}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(string) {})
			out := constNode.Output(0)
			vals := telem.UnmarshalSeries[int64](*out)
			Expect(vals[0]).To(Equal(int64(-42)))
		})
	})
	Describe("constant.Next", func() {
		It("Should do nothing on Next", func() {
			s := state.New(state.Config{
				Edges: ir.Edges{
					{Source: ir.Handle{Node: "const", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam}},
				},
				Nodes: ir.Nodes{
					{
						Key:  "const",
						Type: "constant",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
				},
			})
			factory := constant.NewFactory()
			cfg := node.Config{
				Node:  ir.Node{Type: "constant", ConfigValues: map[string]interface{}{"value": 42}},
				State: s.Node("const"),
			}
			constNode := s.Node("const")
			*constNode.Output(0) = telem.NewSeriesV[int64](0)
			n, _ := factory.Create(ctx, cfg)
			n.Init(ctx, func(string) {})
			out1 := constNode.Output(0)
			len1 := out1.Len()
			outputs := []string{}
			n.Next(ctx, func(output string) {
				outputs = append(outputs, output)
			})
			Expect(outputs).To(BeEmpty())
			out2 := constNode.Output(0)
			Expect(out2.Len()).To(Equal(len1))
		})
	})
	Describe("SymbolResolver", func() {
		It("Should resolve constant symbol", func() {
			sym, ok := constant.SymbolResolver["constant"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("constant"))
		})
	})
})
