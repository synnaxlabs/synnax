package stable_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/stable"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("StableFor", func() {
	ctx := context.Background()

	Describe("Stability Detection", func() {
		It("should output when value is stable for the duration", func() {
			currentTime := telem.TimeStamp(0)
			f := stable.NewFactory(stable.FactoryConfig{
				Now: func() telem.TimeStamp { return currentTime },
			})
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			inputSourceHandle := ir.Handle{Node: "inputSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "stable", Param: ir.DefaultOutputParam}

			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}
			s.Outputs[outputHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputSourceHandle,
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			}

			irNode := ir.Node{
				Key:  "stable",
				Type: "stable_for",
				Config: map[string]interface{}{
					"duration": int64(1000), // 1000 ns
				},
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			// First tick: value becomes 1
			currentTime = 0
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			changedOutputs := []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty()) // Not stable long enough yet

			// Second tick: still 1, but not enough time
			currentTime = 500
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			changedOutputs = []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty()) // Still not stable long enough

			// Third tick: still 1, now enough time
			currentTime = 1000
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			changedOutputs = []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(ConsistOf(ir.DefaultOutputParam))
			result := s.Outputs[outputHandle].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[uint8](1)))
		})

		It("should not output when value changes before duration", func() {
			currentTime := telem.TimeStamp(0)
			f := stable.NewFactory(stable.FactoryConfig{
				Now: func() telem.TimeStamp { return currentTime },
			})
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			inputSourceHandle := ir.Handle{Node: "inputSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "stable", Param: ir.DefaultOutputParam}

			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}
			s.Outputs[outputHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputSourceHandle,
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			}

			irNode := ir.Node{
				Key:  "stable",
				Type: "stable_for",
				Config: map[string]interface{}{
					"duration": int64(1000),
				},
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			// First tick: value becomes 1
			currentTime = 0
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			changedOutputs := []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty())

			// Second tick: value changes to 2 before duration
			currentTime = 500
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](2)}
			changedOutputs = []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty())

			// Third tick: still 2, not enough time since change
			currentTime = 1000
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](2)}
			changedOutputs = []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty())
		})

		It("should reset timer when value changes in series", func() {
			currentTime := telem.TimeStamp(0)
			f := stable.NewFactory(stable.FactoryConfig{
				Now: func() telem.TimeStamp { return currentTime },
			})
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			inputSourceHandle := ir.Handle{Node: "inputSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "stable", Param: ir.DefaultOutputParam}

			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}
			s.Outputs[outputHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputSourceHandle,
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			}

			irNode := ir.Node{
				Key:  "stable",
				Type: "stable_for",
				Config: map[string]interface{}{
					"duration": int64(1000),
				},
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			// First tick: series with changing values [1, 2, 3]
			currentTime = 0
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1, 2, 3)}
			changedOutputs := []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty()) // Timer reset with each change

			// Second tick: value is now stable at 3
			currentTime = 500
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](3)}
			changedOutputs = []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty())

			// Third tick: still 3, enough time has passed
			currentTime = 1000
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](3)}
			changedOutputs = []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(ConsistOf(ir.DefaultOutputParam))
			result := s.Outputs[outputHandle].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[uint8](3)))
		})

		It("should not output again if value already sent", func() {
			currentTime := telem.TimeStamp(0)
			f := stable.NewFactory(stable.FactoryConfig{
				Now: func() telem.TimeStamp { return currentTime },
			})
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			inputSourceHandle := ir.Handle{Node: "inputSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "stable", Param: ir.DefaultOutputParam}

			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}
			s.Outputs[outputHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputSourceHandle,
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			}

			irNode := ir.Node{
				Key:  "stable",
				Type: "stable_for",
				Config: map[string]interface{}{
					"duration": int64(1000),
				},
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			// Get to stable state
			currentTime = 0
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			runtimeNode.Next(ctx, func(output string) {})

			currentTime = 1000
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			changedOutputs := []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(ConsistOf(ir.DefaultOutputParam))

			// Try again with same value - should not output
			currentTime = 2000
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			changedOutputs = []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty())
		})

		It("should output new value after change and stabilization", func() {
			currentTime := telem.TimeStamp(0)
			f := stable.NewFactory(stable.FactoryConfig{
				Now: func() telem.TimeStamp { return currentTime },
			})
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			inputSourceHandle := ir.Handle{Node: "inputSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "stable", Param: ir.DefaultOutputParam}

			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}
			s.Outputs[outputHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputSourceHandle,
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			}

			irNode := ir.Node{
				Key:  "stable",
				Type: "stable_for",
				Config: map[string]interface{}{
					"duration": int64(1000),
				},
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			// Get to stable state with value 1
			currentTime = 0
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			runtimeNode.Next(ctx, func(output string) {})

			currentTime = 1000
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			runtimeNode.Next(ctx, func(output string) {})

			// Change to value 2
			currentTime = 1500
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](2)}
			changedOutputs := []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty())

			// Wait for stabilization
			currentTime = 2500
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](2)}
			changedOutputs = []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(ConsistOf(ir.DefaultOutputParam))
			result := s.Outputs[outputHandle].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[uint8](2)))
		})

		It("should handle empty input", func() {
			currentTime := telem.TimeStamp(0)
			f := stable.NewFactory(stable.FactoryConfig{
				Now: func() telem.TimeStamp { return currentTime },
			})
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			inputSourceHandle := ir.Handle{Node: "inputSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "stable", Param: ir.DefaultOutputParam}

			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}
			s.Outputs[outputHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputSourceHandle,
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			}

			irNode := ir.Node{
				Key:  "stable",
				Type: "stable_for",
				Config: map[string]interface{}{
					"duration": int64(1000),
				},
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			changedOutputs := []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty())
		})

		It("should handle zero duration", func() {
			currentTime := telem.TimeStamp(0)
			f := stable.NewFactory(stable.FactoryConfig{
				Now: func() telem.TimeStamp { return currentTime },
			})
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}

			inputSourceHandle := ir.Handle{Node: "inputSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "stable", Param: ir.DefaultOutputParam}

			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}
			s.Outputs[outputHandle] = state.Output{Data: telem.NewSeriesV[uint8](), Time: telem.Series{DataType: telem.TimeStampT}}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputSourceHandle,
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			}

			irNode := ir.Node{
				Key:  "stable",
				Type: "stable_for",
				Config: map[string]interface{}{
					"duration": int64(0),
				},
			}

			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			// With zero duration, should output immediately
			currentTime = 0
			s.Outputs[inputSourceHandle] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			changedOutputs := []string{}
			runtimeNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(ConsistOf(ir.DefaultOutputParam))
			result := s.Outputs[outputHandle].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[uint8](1)))
		})
	})
})