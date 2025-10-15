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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/stat"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Avg", func() {
	var (
		ctx       context.Context
		s         *state.State
		avgNode   node.Node
		inputKey  ir.Handle
		outputKey ir.Handle
		nowTime   telem.TimeStamp
		nowFn     func() telem.TimeStamp
	)

	BeforeEach(func() {
		ctx = context.Background()
		s = &state.State{
			Outputs: make(map[ir.Handle]state.Output),
		}

		inputKey = ir.Handle{Node: "input", Param: ir.DefaultOutputParam}
		outputKey = ir.Handle{Node: "avg", Param: ir.DefaultOutputParam}

		nowTime = telem.TimeStamp(0)
		nowFn = func() telem.TimeStamp {
			return nowTime
		}

		s.Outputs[inputKey] = state.Output{Data: telem.Series{DataType: telem.Float64T}}
		s.Outputs[outputKey] = state.Output{Data: telem.Series{DataType: telem.Float64T}}
	})

	Describe("Basic averaging", func() {
		It("should compute average of single batch", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "avg",
				Key:  "avg",
				ConfigValues: map[string]any{
					"duration": telem.Second * 10,
				},
			}
			var err error
			avgNode, err = factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			})
			Expect(err).ToNot(HaveOccurred())

			avgNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0)}

			changed := false
			avgNode.Next(ctx, func(output string) {
				changed = true
			})

			Expect(changed).To(BeTrue())
			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](3.0)))
		})

		It("should accumulate average across multiple calls", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "avg",
				Key:  "avg",
				ConfigValues: map[string]any{
					"duration": telem.Second * 10,
				},
			}
			avgNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			avgNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](1.0, 2.0, 3.0)}
			avgNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](2.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](4.0, 5.0)}
			avgNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](3.0)))
		})
	})

	Describe("Time-based window reset", func() {
		It("should reset after duration expires", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "avg",
				Key:  "avg",
				ConfigValues: map[string]any{
					"duration": telem.Second * 5,
				},
			}
			avgNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			avgNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](10.0, 20.0)}
			avgNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](15.0)))

			nowTime += telem.TimeStamp(6 * telem.Second)

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](100.0, 200.0)}
			avgNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](150.0)))
		})

		It("should accumulate within time window", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "avg",
				Key:  "avg",
				ConfigValues: map[string]any{
					"duration": telem.Second * 10,
				},
			}
			avgNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			avgNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](5.0)}
			avgNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](5.0)))

			nowTime += telem.TimeStamp(3 * telem.Second)

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](15.0)}
			avgNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](10.0)))
		})
	})

	Describe("Empty input handling", func() {
		It("should not change output when input is empty", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "avg",
				Key:  "avg",
				ConfigValues: map[string]any{
					"duration": telem.Second * 10,
				},
			}
			avgNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			avgNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.Series{DataType: telem.Float64T}}

			changed := false
			avgNode.Next(ctx, func(output string) {
				changed = true
			})

			Expect(changed).To(BeFalse())
		})
	})

	Describe("Different data types", func() {
		It("should work with int64", func() {
			s.Outputs[inputKey] = state.Output{Data: telem.Series{DataType: telem.Int64T}}
			s.Outputs[outputKey] = state.Output{Data: telem.Series{DataType: telem.Int64T}}

			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "avg",
				Key:  "avg",
				ConfigValues: map[string]any{
					"duration": telem.Second * 10,
				},
			}
			avgNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			avgNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[int64](10, 20, 30)}
			avgNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[int64](20)))
		})

		It("should work with float32", func() {
			s.Outputs[inputKey] = state.Output{Data: telem.Series{DataType: telem.Float32T}}
			s.Outputs[outputKey] = state.Output{Data: telem.Series{DataType: telem.Float32T}}

			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "avg", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "avg",
				Key:  "avg",
				ConfigValues: map[string]any{
					"duration": telem.Second * 10,
				},
			}
			avgNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			avgNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float32](2.5, 3.5, 4.0)}
			avgNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(telem.ValueAt[float32](result, 0)).To(BeNumerically("~", 3.333, 0.001))
		})
	})
})

var _ = Describe("Min", func() {
	var (
		ctx       context.Context
		s         *state.State
		minNode   node.Node
		inputKey  ir.Handle
		outputKey ir.Handle
		resetKey  ir.Handle
		nowTime   telem.TimeStamp
		nowFn     func() telem.TimeStamp
	)

	BeforeEach(func() {
		ctx = context.Background()
		s = &state.State{
			Outputs: make(map[ir.Handle]state.Output),
		}

		inputKey = ir.Handle{Node: "input", Param: ir.DefaultOutputParam}
		outputKey = ir.Handle{Node: "min", Param: ir.DefaultOutputParam}
		resetKey = ir.Handle{Node: "reset", Param: ir.DefaultOutputParam}

		nowTime = telem.TimeStamp(0)
		nowFn = func() telem.TimeStamp {
			return nowTime
		}

		s.Outputs[inputKey] = state.Output{Data: telem.Series{DataType: telem.Float64T}}
		s.Outputs[outputKey] = state.Output{Data: telem.Series{DataType: telem.Float64T}}
		s.Outputs[resetKey] = state.Output{Data: telem.Series{DataType: telem.Uint8T}}
	})

	Describe("Basic min tracking", func() {
		It("should find minimum of single batch", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "min", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type:         "min",
				Key:          "min",
				ConfigValues: map[string]any{},
			}
			minNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			minNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](5.0, 2.0, 8.0, 1.0, 9.0)}
			minNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](1.0)))
		})

		It("should track minimum across multiple calls", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "min", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type:         "min",
				Key:          "min",
				ConfigValues: map[string]any{},
			}
			minNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			minNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](5.0, 3.0, 7.0)}
			minNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](3.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](4.0, 2.0)}
			minNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](2.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](10.0, 8.0)}
			minNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](2.0)))
		})
	})

	Describe("Reset signal", func() {
		It("should reset when reset signal is 1", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "min", Param: ir.DefaultInputParam},
					},
					{
						Source: resetKey,
						Target: ir.Handle{Node: "min", Param: "reset"},
					},
				},
			}
			irNode := ir.Node{
				Type:         "min",
				Key:          "min",
				ConfigValues: map[string]any{},
			}
			minNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			minNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](5.0, 3.0)}
			s.Outputs[resetKey] = state.Output{Data: telem.NewSeriesV[uint8](0)}
			minNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](3.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](10.0, 8.0, 6.0)}
			s.Outputs[resetKey] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			minNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](6.0)))
		})
	})

	Describe("Duration-based reset", func() {
		It("should reset after duration expires", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "min", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "min",
				Key:  "min",
				ConfigValues: map[string]any{
					"duration": telem.Second * 5,
				},
			}
			minNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			minNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](10.0, 5.0)}
			minNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](5.0)))

			nowTime += telem.TimeStamp(6 * telem.Second)

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](20.0, 15.0)}
			minNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](15.0)))
		})
	})

	Describe("Count-based reset", func() {
		It("should reset after count samples", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "min", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "min",
				Key:  "min",
				ConfigValues: map[string]any{
					"count": int64(5),
				},
			}
			minNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			minNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](10.0, 5.0, 8.0)}
			minNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](5.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](7.0, 3.0)}
			minNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](3.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](20.0, 15.0)}
			minNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](15.0)))
		})
	})

	Describe("Different data types", func() {
		It("should work with int64", func() {
			s.Outputs[inputKey] = state.Output{Data: telem.Series{DataType: telem.Int64T}}
			s.Outputs[outputKey] = state.Output{Data: telem.Series{DataType: telem.Int64T}}

			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "min", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type:         "min",
				Key:          "min",
				ConfigValues: map[string]any{},
			}
			minNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			minNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[int64](30, 10, 20)}
			minNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[int64](10)))
		})
	})
})

var _ = Describe("Max", func() {
	var (
		ctx       context.Context
		s         *state.State
		maxNode   node.Node
		inputKey  ir.Handle
		outputKey ir.Handle
		resetKey  ir.Handle
		nowTime   telem.TimeStamp
		nowFn     func() telem.TimeStamp
	)

	BeforeEach(func() {
		ctx = context.Background()
		s = &state.State{
			Outputs: make(map[ir.Handle]state.Output),
		}

		inputKey = ir.Handle{Node: "input", Param: ir.DefaultOutputParam}
		outputKey = ir.Handle{Node: "max", Param: ir.DefaultOutputParam}
		resetKey = ir.Handle{Node: "reset", Param: ir.DefaultOutputParam}

		nowTime = telem.TimeStamp(0)
		nowFn = func() telem.TimeStamp {
			return nowTime
		}

		s.Outputs[inputKey] = state.Output{Data: telem.Series{DataType: telem.Float64T}}
		s.Outputs[outputKey] = state.Output{Data: telem.Series{DataType: telem.Float64T}}
		s.Outputs[resetKey] = state.Output{Data: telem.Series{DataType: telem.Uint8T}}
	})

	Describe("Basic max tracking", func() {
		It("should find maximum of single batch", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type:         "max",
				Key:          "max",
				ConfigValues: map[string]any{},
			}
			maxNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			maxNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](5.0, 2.0, 8.0, 1.0, 9.0)}
			maxNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](9.0)))
		})

		It("should track maximum across multiple calls", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type:         "max",
				Key:          "max",
				ConfigValues: map[string]any{},
			}
			maxNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			maxNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](5.0, 3.0, 7.0)}
			maxNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](7.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](10.0, 8.0)}
			maxNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](10.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](2.0, 4.0)}
			maxNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](10.0)))
		})
	})

	Describe("Reset signal", func() {
		It("should reset when reset signal is 1", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
					{
						Source: resetKey,
						Target: ir.Handle{Node: "max", Param: "reset"},
					},
				},
			}
			irNode := ir.Node{
				Type:         "max",
				Key:          "max",
				ConfigValues: map[string]any{},
			}
			maxNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			maxNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](5.0, 10.0)}
			s.Outputs[resetKey] = state.Output{Data: telem.NewSeriesV[uint8](0)}
			maxNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](10.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](2.0, 4.0, 6.0)}
			s.Outputs[resetKey] = state.Output{Data: telem.NewSeriesV[uint8](1)}
			maxNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](6.0)))
		})
	})

	Describe("Duration-based reset", func() {
		It("should reset after duration expires", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "max",
				Key:  "max",
				ConfigValues: map[string]any{
					"duration": telem.Second * 5,
				},
			}
			maxNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			maxNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](10.0, 20.0)}
			maxNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](20.0)))

			nowTime += telem.TimeStamp(6 * telem.Second)

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](5.0, 8.0)}
			maxNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](8.0)))
		})
	})

	Describe("Count-based reset", func() {
		It("should reset after count samples", func() {
			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type: "max",
				Key:  "max",
				ConfigValues: map[string]any{
					"count": int64(5),
				},
			}
			maxNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			maxNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](10.0, 20.0, 15.0)}
			maxNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](20.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](18.0, 25.0)}
			maxNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](25.0)))

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[float64](5.0, 10.0)}
			maxNode.Next(ctx, func(output string) {})

			result = s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[float64](10.0)))
		})
	})

	Describe("Different data types", func() {
		It("should work with int64", func() {
			s.Outputs[inputKey] = state.Output{Data: telem.Series{DataType: telem.Int64T}}
			s.Outputs[outputKey] = state.Output{Data: telem.Series{DataType: telem.Int64T}}

			factory := stat.NewFactory(stat.Config{Now: nowFn})
			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: inputKey,
						Target: ir.Handle{Node: "max", Param: ir.DefaultInputParam},
					},
				},
			}
			irNode := ir.Node{
				Type:         "max",
				Key:          "max",
				ConfigValues: map[string]any{},
			}
			maxNode = MustSucceed(factory.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			maxNode.Init(ctx, func(output string) {})

			s.Outputs[inputKey] = state.Output{Data: telem.NewSeriesV[int64](30, 50, 20)}
			maxNode.Next(ctx, func(output string) {})

			result := s.Outputs[outputKey].Data
			Expect(result).To(telem.MatchSeries(telem.NewSeriesV[int64](50)))
		})
	})
})
