// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stat_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/stat"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Stat", func() {

	Describe("avg", func() {

		It("Should compute running average with count-based reset", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "input", Type: "input"},
					{Key: "avg", Type: "avg"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "input",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F64()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			inputNode := s.Node("input")
			n := MustSucceed(stat.Factory.Create(ctx, node.Config{
				Node:  ir.Node{Type: "avg", Config: types.Params{{Name: "count", Type: types.I64(), Value: int64(3)}}},
				State: s.Node("avg"),
			}))
			n.Init(node.Context{Context: ctx, MarkChanged: func(string) {}})
			*inputNode.Output(0) = telem.NewSeriesV(10.0, 20.0, 30.0)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("avg").Output(0)
			resultTime := *s.Node("avg").OutputTime(0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(resultTime.Len()).To(Equal(int64(1)))
			vals := telem.UnmarshalSeries[float64](result)
			Expect(vals[0]).To(BeNumerically("~", 20.0, 0.01))
			timeVals := telem.UnmarshalSeries[telem.TimeStamp](resultTime)
			Expect(timeVals[0]).To(Equal(telem.SecondTS * 3)) // Last input timestamp
			*inputNode.Output(0) = telem.NewSeriesV(40.0, 50.0, 60.0)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result = *s.Node("avg").Output(0)
			resultTime = *s.Node("avg").OutputTime(0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(resultTime.Len()).To(Equal(int64(1)))
			vals = telem.UnmarshalSeries[float64](result)
			Expect(vals[0]).To(BeNumerically("~", 50.0, 0.01))
			timeVals = telem.UnmarshalSeries[telem.TimeStamp](resultTime)
			Expect(timeVals[0]).To(Equal(telem.SecondTS * 6)) // Last input timestamp after reset
		})
	})

	Describe("min", func() {
		It("Should compute running minimum with duration-based reset", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "input", Type: "input"},
					{Key: "min", Type: "min"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "min", Param: ir.DefaultInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "input",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I32()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			inputNode := s.Node("input")
			n := MustSucceed(stat.Factory.Create(ctx, node.Config{
				Node:  ir.Node{Type: "min", Config: types.Params{{Name: "duration", Type: types.TimeSpan(), Value: telem.Second * 5}}},
				State: s.Node("min"),
			}))
			n.Init(node.Context{Context: ctx, MarkChanged: func(string) {}})
			// First batch: timestamps [1s, 2s, 3s] with duration 5s
			// No reset: 3s - 1s = 2s < 5s
			*inputNode.Output(0) = telem.NewSeriesV[int32](50, 10, 70)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("min").Output(0)
			resultTime := *s.Node("min").OutputTime(0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(resultTime.Len()).To(Equal(int64(1)))
			vals := telem.UnmarshalSeries[int32](result)
			Expect(vals[0]).To(Equal(int32(10)))
			timeVals := telem.UnmarshalSeries[telem.TimeStamp](resultTime)
			Expect(timeVals[0]).To(Equal(telem.SecondTS * 3))
			// Second batch: timestamps [6s, 7s, 8s]
			// Reset: 6s - 1s = 5s >= 5s (triggers reset)
			// After reset, min should be 40 (min of second batch only)
			// Without reset, min would still be 10 (min across both batches)
			*inputNode.Output(0) = telem.NewSeriesV[int32](80, 40, 60)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(6, 7, 8)
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result = *s.Node("min").Output(0)
			resultTime = *s.Node("min").OutputTime(0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(resultTime.Len()).To(Equal(int64(1)))
			vals = telem.UnmarshalSeries[int32](result)
			Expect(vals[0]).To(Equal(int32(40)))
			timeVals = telem.UnmarshalSeries[telem.TimeStamp](resultTime)
			Expect(timeVals[0]).To(Equal(telem.SecondTS * 8))
		})
	})

	Describe("max", func() {
		It("Should compute running maximum with signal-based reset", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "input", Type: "input"},
					{Key: "reset_signal", Type: "reset_signal"},
					{Key: "max", Type: "max"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
					{
						Source: ir.Handle{Node: "reset_signal", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "max", Param: "reset"},
					},
				},
				Functions: []graph.Function{
					{
						Key: "input",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U64()},
						},
					},
					{
						Key: "reset_signal",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			inputNode := s.Node("input")
			resetNode := s.Node("reset_signal")
			n := MustSucceed(stat.Factory.Create(ctx, node.Config{
				Node:   ir.Node{Type: "max"},
				State:  s.Node("max"),
				Module: module.Module{IR: analyzed},
			}))
			n.Init(node.Context{Context: ctx, MarkChanged: func(string) {}})
			*inputNode.Output(0) = telem.NewSeriesV[uint64](10, 50, 30)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			*resetNode.Output(0) = telem.NewSeriesV[uint8](0)
			*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("max").Output(0)
			resultTime := *s.Node("max").OutputTime(0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(resultTime.Len()).To(Equal(int64(1)))
			vals := telem.UnmarshalSeries[uint64](result)
			Expect(vals[0]).To(Equal(uint64(50)))
			timeVals := telem.UnmarshalSeries[telem.TimeStamp](resultTime)
			Expect(timeVals[0]).To(Equal(telem.SecondTS * 3)) // Last input timestamp
			*inputNode.Output(0) = telem.NewSeriesV[uint64](25, 15, 70)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
			*resetNode.Output(0) = telem.NewSeriesV[uint8](1)
			*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4)
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result = *s.Node("max").Output(0)
			resultTime = *s.Node("max").OutputTime(0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(resultTime.Len()).To(Equal(int64(1)))
			vals = telem.UnmarshalSeries[uint64](result)
			Expect(vals[0]).To(Equal(uint64(70)))
			timeVals = telem.UnmarshalSeries[telem.TimeStamp](resultTime)
			Expect(timeVals[0]).To(Equal(telem.SecondTS * 6)) // Last input timestamp after reset
		})

		It("Should work without optional reset signal connected", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "input", Type: "input"},
					{Key: "max", Type: "max"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
					// Note: No reset edge connected - testing optional input
				},
				Functions: []graph.Function{
					{
						Key: "input",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U64()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			s := state.New(state.Config{IR: analyzed})
			inputNode := s.Node("input")
			n := MustSucceed(stat.Factory.Create(ctx, node.Config{
				Node:   ir.Node{Type: "max"},
				State:  s.Node("max"),
				Module: module.Module{IR: analyzed},
			}))
			n.Init(node.Context{Context: ctx, MarkChanged: func(string) {}})
			// Should work even without reset signal
			*inputNode.Output(0) = telem.NewSeriesV[uint64](10, 50, 30)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("max").Output(0)
			Expect(result.Len()).To(Equal(int64(1)))
			vals := telem.UnmarshalSeries[uint64](result)
			Expect(vals[0]).To(Equal(uint64(50)))
			// Should continue accumulating without reset
			*inputNode.Output(0) = telem.NewSeriesV[uint64](25, 80, 40)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result = *s.Node("max").Output(0)
			vals = telem.UnmarshalSeries[uint64](result)
			Expect(vals[0]).To(Equal(uint64(80))) // Max across both batches
		})

		It("Should catch fast reset pulses (1->0 transition)", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "input", Type: "input"},
					{Key: "reset_signal", Type: "reset_signal"},
					{Key: "avg", Type: "avg"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
					{
						Source: ir.Handle{Node: "reset_signal", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "avg", Param: "reset"},
					},
				},
				Functions: []graph.Function{
					{
						Key: "input",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
					},
					{
						Key: "reset_signal",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			inputNode := s.Node("input")
			resetNode := s.Node("reset_signal")
			n := MustSucceed(stat.Factory.Create(ctx, node.Config{
				Node:   ir.Node{Type: "avg", Key: "avg"},
				State:  s.Node("avg"),
				Module: module.Module{IR: analyzed},
			}))
			n.Init(node.Context{Context: ctx, MarkChanged: func(string) {}})
			// Accumulate some data
			*inputNode.Output(0) = telem.NewSeriesV[int64](10, 20, 30)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			*resetNode.Output(0) = telem.NewSeriesV[uint8](0)
			*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("avg").Output(0)
			vals := telem.UnmarshalSeries[int64](result)
			Expect(vals[0]).To(Equal(int64(20))) // (10+20+30)/3 = 20
			// Fast pulse: reset goes 1 then immediately back to 0 in same series
			*inputNode.Output(0) = telem.NewSeriesV[int64](40, 50, 60)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
			// Reset pulse: 1 at time 4, 0 at time 5
			*resetNode.Output(0) = telem.NewSeriesV[uint8](1, 0)
			*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5)
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result = *s.Node("avg").Output(0)
			vals = telem.UnmarshalSeries[int64](result)
			// Should have caught the reset pulse and restarted averaging
			// Average of just [40, 50, 60] = 50, not (10+20+30+40+50+60)/6 = 35
			Expect(vals[0]).To(Equal(int64(50)))
		})
	})

	Describe("Alignment Propagation", func() {
		It("Should propagate alignment from input to output", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "input", Type: "input"},
					{Key: "avg", Type: "avg"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
				Functions: []graph.Function{
					{
						Key: "input",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F64()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			inputNode := s.Node("input")
			n := MustSucceed(stat.Factory.Create(ctx, node.Config{
				Node:   ir.Node{Type: "avg"},
				State:  s.Node("avg"),
				Module: module.Module{IR: analyzed},
			}))
			n.Init(node.Context{Context: ctx, MarkChanged: func(string) {}})

			inputSeries := telem.NewSeriesV(10.0, 20.0, 30.0)
			inputSeries.Alignment = 250
			inputSeries.TimeRange = telem.TimeRange{Start: 100 * telem.SecondTS, End: 300 * telem.SecondTS}
			*inputNode.Output(0) = inputSeries
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(100, 200, 300)

			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())

			result := *s.Node("avg").Output(0)
			Expect(result.Alignment).To(Equal(telem.Alignment(250)))
			Expect(result.TimeRange.Start).To(Equal(100 * telem.SecondTS))
			Expect(result.TimeRange.End).To(Equal(300 * telem.SecondTS))

			resultTime := *s.Node("avg").OutputTime(0)
			Expect(resultTime.Alignment).To(Equal(telem.Alignment(250)))
			Expect(resultTime.TimeRange.Start).To(Equal(100 * telem.SecondTS))
			Expect(resultTime.TimeRange.End).To(Equal(300 * telem.SecondTS))
		})

		It("Should sum alignments when reset signal is connected", func() {
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "input", Type: "input"},
					{Key: "reset", Type: "reset"},
					{Key: "avg", Type: "avg"},
				},
				Edges: []graph.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
					{
						Source: ir.Handle{Node: "reset", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "avg", Param: "reset"},
					},
				},
				Functions: []graph.Function{
					{
						Key: "input",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
					},
					{
						Key: "reset",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, stat.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			inputNode := s.Node("input")
			resetNode := s.Node("reset")
			n := MustSucceed(stat.Factory.Create(ctx, node.Config{
				Node:   ir.Node{Key: "avg", Type: "avg"},
				State:  s.Node("avg"),
				Module: module.Module{IR: analyzed},
			}))
			n.Init(node.Context{Context: ctx, MarkChanged: func(string) {}})

			inputSeries := telem.NewSeriesV[int64](10, 20, 30)
			inputSeries.Alignment = 100
			inputSeries.TimeRange = telem.TimeRange{Start: 50 * telem.SecondTS, End: 150 * telem.SecondTS}
			*inputNode.Output(0) = inputSeries
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(50, 100, 150)

			resetSeries := telem.NewSeriesV[uint8](0)
			resetSeries.Alignment = 75
			resetSeries.TimeRange = telem.TimeRange{Start: 25 * telem.SecondTS, End: 175 * telem.SecondTS}
			*resetNode.Output(0) = resetSeries
			*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(25)

			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())

			result := *s.Node("avg").Output(0)
			Expect(result.Alignment).To(Equal(telem.Alignment(175))) // 100 + 75
			Expect(result.TimeRange.Start).To(Equal(25 * telem.SecondTS))
			Expect(result.TimeRange.End).To(Equal(175 * telem.SecondTS))

			resultTime := *s.Node("avg").OutputTime(0)
			Expect(resultTime.Alignment).To(Equal(telem.Alignment(175)))
			Expect(resultTime.TimeRange.Start).To(Equal(25 * telem.SecondTS))
			Expect(resultTime.TimeRange.End).To(Equal(175 * telem.SecondTS))
		})
	})
})
