package constant_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Constant", func() {
	ctx := context.Background()

	Describe("Constant Node", func() {
		DescribeTable("should produce constant values of different types",
			func(
				value any,
				outputDataType telem.DataType,
				expectedOutput telem.Series,
			) {
				f := constant.NewFactory()
				s := &state.State{Outputs: map[ir.Handle]state.Output{}}

				outputHandle := ir.Handle{Node: "constant", Param: ir.DefaultOutputParam}
				s.Outputs[outputHandle] = state.Output{Data: telem.Series{DataType: outputDataType}}

				inter := ir.IR{Edges: []ir.Edge{}}
				irNode := ir.Node{
					Key:    "constant",
					Type:   "constant",
					Config: map[string]any{"value": value},
				}

				runtimeNode := MustSucceed(f.Create(ctx, node.Config{
					State:  s,
					Node:   irNode,
					Module: module.Module{IR: inter},
				}))

				changedOutputs := []string{}
				runtimeNode.Init(ctx, func(output string) {
					changedOutputs = append(changedOutputs, output)
				})

				Expect(changedOutputs).To(ConsistOf([]string{ir.DefaultOutputParam}))

				result := s.Outputs[outputHandle].Data
				Expect(result).To(telem.MatchSeries(expectedOutput))
			},
			Entry("int constant - positive",
				int(42),
				telem.Int64T,
				telem.NewSeriesV[int64](42),
			),
			Entry("int constant - negative",
				int(-100),
				telem.Int64T,
				telem.NewSeriesV[int64](-100),
			),
			Entry("int constant - zero",
				int(0),
				telem.Int64T,
				telem.NewSeriesV[int64](0),
			),
			Entry("float64 constant - positive",
				float64(3.14159),
				telem.Float64T,
				telem.NewSeriesV[float64](3.14159),
			),
			Entry("float64 constant - negative",
				float64(-2.71828),
				telem.Float64T,
				telem.NewSeriesV[float64](-2.71828),
			),
			Entry("float64 constant - zero",
				float64(0.0),
				telem.Float64T,
				telem.NewSeriesV[float64](0.0),
			),
			Entry("float64 constant - very small",
				float64(0.000001),
				telem.Float64T,
				telem.NewSeriesV[float64](0.000001),
			),
			Entry("float64 constant - very large",
				float64(1e10),
				telem.Float64T,
				telem.NewSeriesV[float64](1e10),
			),
			Entry("float32 constant - positive",
				float32(1.23),
				telem.Float32T,
				telem.NewSeriesV[float32](1.23),
			),
			Entry("float32 constant - negative",
				float32(-4.56),
				telem.Float32T,
				telem.NewSeriesV[float32](-4.56),
			),
			Entry("float32 constant - zero",
				float32(0.0),
				telem.Float32T,
				telem.NewSeriesV[float32](0.0),
			),
		)

		It("should not change outputs on Next", func() {
			f := constant.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			outputHandle := ir.Handle{Node: "constant", Param: ir.DefaultOutputParam}
			s.Outputs[outputHandle] = state.Output{Data: telem.Series{DataType: telem.Int64T}}

			inter := ir.IR{Edges: []ir.Edge{}}
			irNode := ir.Node{
				Key:    "constant",
				Type:   "constant",
				Config: map[string]any{"value": int(100)},
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			// Initialize the node
			runtimeNode.Init(ctx, func(output string) {})

			// Call Next multiple times - should not change anything
			changedOutputs := []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})

			Expect(changedOutputs).To(BeEmpty())

			// Verify output remains unchanged
			result := s.Outputs[outputHandle].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[int64](100)))
		})

		It("should cast input value to output type", func() {
			f := constant.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			outputHandle := ir.Handle{Node: "constant", Param: ir.DefaultOutputParam}
			// Output type is float64, but we'll pass an int value
			s.Outputs[outputHandle] = state.Output{Data: telem.Series{DataType: telem.Float64T}}

			inter := ir.IR{Edges: []ir.Edge{}}
			irNode := ir.Node{
				Key:    "constant",
				Type:   "constant",
				Config: map[string]any{"value": int(42)}, // int input
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			runtimeNode.Init(ctx, func(output string) {})

			// Should be cast to float64
			result := s.Outputs[outputHandle].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](42.0)))
		})

		It("should cast float to int type", func() {
			f := constant.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			outputHandle := ir.Handle{Node: "constant", Param: ir.DefaultOutputParam}
			// Output type is int32, but we'll pass a float value
			s.Outputs[outputHandle] = state.Output{Data: telem.Series{DataType: telem.Int32T}}

			inter := ir.IR{Edges: []ir.Edge{}}
			irNode := ir.Node{
				Key:    "constant",
				Type:   "constant",
				Config: map[string]any{"value": float64(3.7)}, // float input
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			runtimeNode.Init(ctx, func(output string) {})

			// Should be cast to int32 (truncated)
			result := s.Outputs[outputHandle].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[int32](3)))
		})

		It("should only produce output during Init, not Next", func() {
			f := constant.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			outputHandle := ir.Handle{Node: "constant", Param: ir.DefaultOutputParam}
			s.Outputs[outputHandle] = state.Output{Data: telem.Series{DataType: telem.Float64T}}

			inter := ir.IR{Edges: []ir.Edge{}}
			irNode := ir.Node{
				Key:    "constant",
				Type:   "constant",
				Config: map[string]any{"value": float64(2.5)},
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			initCalled := false
			runtimeNode.Init(ctx, func(output string) {
				initCalled = true
			})
			Expect(initCalled).To(BeTrue())

			nextCalled := false
			runtimeNode.Next(ctx, func(output string) {
				nextCalled = true
			})
			Expect(nextCalled).To(BeFalse())
		})
	})

	Describe("Constant Factory", func() {
		It("should return NotFound for wrong node type", func() {
			f := constant.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			irNode := ir.Node{
				Key:    "wrong",
				Type:   "not_constant",
				Config: map[string]any{},
			}

			_, err := f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: ir.IR{}},
			})

			Expect(err).To(HaveOccurred())
		})

		It("should create constant node for correct type", func() {
			f := constant.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			irNode := ir.Node{
				Key:    "constant",
				Type:   "constant",
				Config: map[string]any{"value": int(42)},
			}

			n, err := f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: ir.IR{}},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(n).NotTo(BeNil())
		})
	})
})