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
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "input",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F64()},
						},
					},
					{
						Key:  "avg",
						Type: "avg",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.F64()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.F64()},
						},
						ConfigValues: map[string]interface{}{
							"count": int64(3),
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
			}
			s := state.New(cfg)
			inputNode := s.Node("input")
			factory := stat.NewFactory(stat.Config{})
			n := MustSucceed(factory.Create(ctx, node.Config{
				Node:  cfg.Nodes[1],
				State: s.Node("avg"),
			}))
			n.Init(node.Context{Context: ctx, MarkChanged: func(string) {}})
			*inputNode.Output(0) = telem.NewSeriesV[float64](10.0, 20.0, 30.0)
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
			*inputNode.Output(0) = telem.NewSeriesV[float64](40.0, 50.0, 60.0)
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
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "input",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
					},
					{
						Key:  "min",
						Type: "min",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam},
							Values: []types.Type{types.I32()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I32()},
						},
						ConfigValues: map[string]interface{}{
							"duration": telem.Second * 5,
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "min", Param: ir.DefaultInputParam},
					},
				},
			}
			s := state.New(cfg)
			inputNode := s.Node("input")
			factory := stat.NewFactory(stat.Config{})
			n := MustSucceed(factory.Create(ctx, node.Config{
				Node:  cfg.Nodes[1],
				State: s.Node("min"),
			}))
			n.Init(node.Context{Context: ctx, MarkChanged: func(string) {}})
			// First batch: timestamps [1s, 2s, 3s] with duration 5s
			// No reset: 3s - 1s = 2s < 5s
			*inputNode.Output(0) = telem.NewSeriesV[int32](50, 30, 70)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("min").Output(0)
			resultTime := *s.Node("min").OutputTime(0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(resultTime.Len()).To(Equal(int64(1)))
			vals := telem.UnmarshalSeries[int32](result)
			Expect(vals[0]).To(Equal(int32(30)))
			timeVals := telem.UnmarshalSeries[telem.TimeStamp](resultTime)
			Expect(timeVals[0]).To(Equal(telem.SecondTS * 3)) // Last input timestamp
			// Second batch: timestamps [6s, 7s, 8s]
			// Reset: 6s - 1s = 5s >= 5s (triggers reset)
			*inputNode.Output(0) = telem.NewSeriesV[int32](40, 20, 60)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(6, 7, 8)
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result = *s.Node("min").Output(0)
			resultTime = *s.Node("min").OutputTime(0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(resultTime.Len()).To(Equal(int64(1)))
			vals = telem.UnmarshalSeries[int32](result)
			Expect(vals[0]).To(Equal(int32(20)))
			timeVals = telem.UnmarshalSeries[telem.TimeStamp](resultTime)
			Expect(timeVals[0]).To(Equal(telem.SecondTS * 8)) // Last input timestamp after reset
		})
	})
	Describe("max", func() {
		It("Should compute running maximum with signal-based reset", func() {
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "input",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U64()},
						},
					},
					{
						Key: "reset_signal",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
					{
						Key:  "max",
						Type: "max",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam, "reset"},
							Values: []types.Type{types.U64(), types.U8()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U64()},
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
					{
						Source: ir.Handle{Node: "reset_signal", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "max", Param: "reset"},
					},
				},
			}
			s := state.New(cfg)
			inputNode := s.Node("input")
			resetNode := s.Node("reset_signal")
			factory := stat.NewFactory(stat.Config{})
			inter := ir.IR{Edges: cfg.Edges}
			n := MustSucceed(factory.Create(ctx, node.Config{
				Node:   cfg.Nodes[2],
				State:  s.Node("max"),
				Module: module.Module{IR: inter},
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
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "input",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U64()},
						},
					},
					{
						Key:  "max",
						Type: "max",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam, "reset"},
							Values: []types.Type{types.U64(), types.U8()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U64()},
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
					// Note: No reset edge connected - testing optional input
				},
			}
			s := state.New(cfg)
			inputNode := s.Node("input")
			factory := stat.NewFactory(stat.Config{})
			inter := ir.IR{Edges: cfg.Edges}
			n := MustSucceed(factory.Create(ctx, node.Config{
				Node:   cfg.Nodes[1],
				State:  s.Node("max"),
				Module: module.Module{IR: inter},
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
			cfg := state.Config{
				Nodes: []ir.Node{
					{
						Key: "input",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
					{
						Key: "reset_signal",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
					{
						Key:  "avg",
						Type: "avg",
						Inputs: types.Params{
							Keys:   []string{ir.DefaultInputParam, "reset"},
							Values: []types.Type{types.I64(), types.U8()},
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.I64()},
						},
					},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
					{
						Source: ir.Handle{Node: "reset_signal", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "avg", Param: "reset"},
					},
				},
			}
			s := state.New(cfg)
			inputNode := s.Node("input")
			resetNode := s.Node("reset_signal")
			factory := stat.NewFactory(stat.Config{})
			inter := ir.IR{Edges: cfg.Edges}
			n := MustSucceed(factory.Create(ctx, node.Config{
				Node:   cfg.Nodes[2],
				State:  s.Node("avg"),
				Module: module.Module{IR: inter},
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
})
