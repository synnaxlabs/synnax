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
			n.Init(ctx, func(string) {})
			*inputNode.Output(0) = telem.NewSeriesV[float64](10.0, 20.0, 30.0)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			changed := make(set.Set[string])
			n.Next(ctx, func(output string) { changed.Add(output) })
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("avg").Output(0)
			Expect(result.Len()).To(Equal(int64(1)))
			vals := telem.UnmarshalSeries[float64](result)
			Expect(vals[0]).To(BeNumerically("~", 20.0, 0.01))
			*inputNode.Output(0) = telem.NewSeriesV[float64](40.0, 50.0, 60.0)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
			changed = make(set.Set[string])
			n.Next(ctx, func(output string) { changed.Add(output) })
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result = *s.Node("avg").Output(0)
			vals = telem.UnmarshalSeries[float64](result)
			Expect(vals[0]).To(BeNumerically("~", 50.0, 0.01))
		})
	})
	Describe("min", func() {
		It("Should compute running minimum with duration-based reset", func() {
			currentTime := telem.TimeStamp(0)
			mockNow := func() telem.TimeStamp {
				return currentTime
			}
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
			factory := stat.NewFactory(stat.Config{Now: mockNow})
			n := MustSucceed(factory.Create(ctx, node.Config{
				Node:  cfg.Nodes[1],
				State: s.Node("min"),
			}))
			n.Init(ctx, func(string) {})
			*inputNode.Output(0) = telem.NewSeriesV[int32](50, 30, 70)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			changed := make(set.Set[string])
			n.Next(ctx, func(output string) { changed.Add(output) })
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("min").Output(0)
			vals := telem.UnmarshalSeries[int32](result)
			Expect(vals[0]).To(Equal(int32(30)))
			currentTime += telem.SecondTS * 6
			*inputNode.Output(0) = telem.NewSeriesV[int32](40, 20, 60)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
			changed = make(set.Set[string])
			n.Next(ctx, func(output string) { changed.Add(output) })
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result = *s.Node("min").Output(0)
			vals = telem.UnmarshalSeries[int32](result)
			Expect(vals[0]).To(Equal(int32(20)))
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
			n.Init(ctx, func(string) {})
			*inputNode.Output(0) = telem.NewSeriesV[uint64](10, 50, 30)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1, 2, 3)
			*resetNode.Output(0) = telem.NewSeriesV[uint8](0)
			*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			changed := make(set.Set[string])
			n.Next(ctx, func(output string) { changed.Add(output) })
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result := *s.Node("max").Output(0)
			vals := telem.UnmarshalSeries[uint64](result)
			Expect(vals[0]).To(Equal(uint64(50)))
			*inputNode.Output(0) = telem.NewSeriesV[uint64](25, 15, 70)
			*inputNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4, 5, 6)
			*resetNode.Output(0) = telem.NewSeriesV[uint8](1)
			*resetNode.OutputTime(0) = telem.NewSeriesSecondsTSV(4)
			changed = make(set.Set[string])
			n.Next(ctx, func(output string) { changed.Add(output) })
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			result = *s.Node("max").Output(0)
			vals = telem.UnmarshalSeries[uint64](result)
			Expect(vals[0]).To(Equal(uint64(70)))
		})
	})
})
