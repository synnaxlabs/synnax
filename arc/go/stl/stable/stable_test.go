// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stable_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("StableFor", func() {
	var (
		module      *stable.Host
		s           *node.ProgramState
		irNode      ir.Node
		currentTime telem.TimeStamp
	)
	BeforeEach(func(ctx SpecContext) {
		module = stable.NewHost()
		irNode = ir.Node{
			Key:  "stable",
			Type: "stable_for",
			Inputs: types.Params{
				{Name: "duration", Type: types.TimeSpan(), Value: telem.Second * 1},
				{Name: ir.DefaultInputParam, Type: types.U8()},
			},
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.U8()},
			},
		}
		g := graph.Graph{
			Nodes: []graph.Node{
				{Key: "source"},
				{Key: "stable"},
			},
			Inputs: map[string]msgpack.EncodedJSON{
				"source": {"type": "source"},
				"stable": {"type": "stable_for"},
			},
			Edges: graph.Edges{
				{
					Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
				},
			},
			Functions: []ir.Function{
				{
					Key: "source",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
				},
				{
					Key: "stable_for",
					Inputs: types.Params{
						{Name: ir.DefaultInputParam, Type: types.U8()},
					},
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
				},
			},
		}
		analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
		Expect(diagnostics.Ok()).To(BeTrue())
		s = node.New(analyzed)
	})

	Describe("Factory.Create", func() {
		It("Should create node for stable_for type", func() {
			n := MustSucceed(module.Create(node.Config{
				Node: irNode, State: s.Node(irNode.Key),
			}))
			Expect(n).ToNot(BeNil())
		})

		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("stable"),
			}
			Expect(module.Create(cfg)).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Next", func() {
		It("Should handle empty input", func(ctx SpecContext) {
			cfg := node.Config{Node: irNode, State: s.Node("stable")}
			source := s.Node("source")
			*source.Output(0) = telem.NewSeriesV[uint8]()
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV()
			source.MarkFresh(0)
			n := MustSucceed(module.Create(cfg))
			outputs := make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeFalse())
		})

		It(
			"Should not emit when value is not stable for duration",
			func(ctx SpecContext) {
				cfg := node.Config{Node: irNode, State: s.Node("stable")}
				source := s.Node("source")
				currentTime = 0
				*source.Output(0) = telem.NewSeriesV[uint8](5)
				*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
				source.MarkFresh(0)
				n := MustSucceed(module.Create(cfg))
				outputs := make(set.Set[int])
				n.Next(
					node.Context{
						Context:     ctx,
						Now:         currentTime,
						MarkChanged: func(i int) { outputs.Add(i) },
					},
				)
				Expect(outputs.Contains(0)).To(BeFalse())
				currentTime = telem.SecondTS / 2
				outputs = make(set.Set[int])
				n.Next(
					node.Context{
						Context:     ctx,
						Now:         currentTime,
						MarkChanged: func(i int) { outputs.Add(i) },
					},
				)
				Expect(outputs.Contains(0)).To(BeFalse())
			},
		)

		It("Should emit when value is stable for duration", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					Inputs: types.Params{
						{
							Name:  "duration",
							Type:  types.TimeSpan(),
							Value: telem.SecondTS,
						},
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send value 5 at time 1s
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			source.MarkFresh(0)
			n := MustSucceed(module.Create(cfg))
			outputs := make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeFalse())

			// A sample of the same value stamped exactly duration later closes
			// the window.
			currentTime = telem.SecondTS * 2
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
			source.MarkFresh(0)
			outputs = make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeTrue())

			stableNode := s.Node("stable")
			output := stableNode.Output(0)
			Expect(output.Len()).To(Equal(int64(1)))
			outputVals := output.Unmarshal[uint8]()
			Expect(outputVals).To(Equal([]uint8{5}))
		})

		It("Should reset timer when value changes", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					Inputs: types.Params{
						{
							Name:  "duration",
							Type:  types.TimeSpan(),
							Value: telem.SecondTS,
						},
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send value 5 at time 0
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
			source.MarkFresh(0)
			n := MustSucceed(module.Create(cfg))
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(int) {},
				},
			)

			// Advance time partway
			currentTime = telem.SecondTS / 2

			// Send different value 10 at time 1s
			*source.Output(0) = telem.NewSeriesV[uint8](10)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			source.MarkFresh(0)
			outputs := make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeFalse())

			// Same value at 1.5s: only 0.5s since the change at 1s.
			currentTime = telem.SecondTS + telem.SecondTS/2
			*source.Output(0) = telem.NewSeriesV[uint8](10)
			*source.OutputTime(0) = telem.NewSeriesV(currentTime)
			source.MarkFresh(0)
			outputs = make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			// Should not emit yet - value changed at 1s, only 0.5s elapsed
			Expect(outputs.Contains(0)).To(BeFalse())

			// Same value at 2s: 1s since the change at 1s.
			currentTime = telem.SecondTS * 2
			*source.Output(0) = telem.NewSeriesV[uint8](10)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
			source.MarkFresh(0)
			outputs = make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeTrue())
		})

		It("Should not emit same value twice", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					Inputs: types.Params{
						{
							Name:  "duration",
							Type:  types.TimeSpan(),
							Value: telem.SecondTS,
						},
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send value 5 at time 1
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			source.MarkFresh(0)
			n := MustSucceed(module.Create(cfg))
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(int) {},
				},
			)

			currentTime = telem.SecondTS * 2
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
			source.MarkFresh(0)
			outputs := make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeTrue())

			// Call again with same value - should not emit
			currentTime = telem.SecondTS * 3
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(3)
			source.MarkFresh(0)
			outputs = make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeFalse())
		})

		It("Should emit different value after stable period", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					Inputs: types.Params{
						{
							Name:  "duration",
							Type:  types.TimeSpan(),
							Value: telem.SecondTS,
						},
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send value 5
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
			source.MarkFresh(0)
			n := MustSucceed(module.Create(cfg))
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(int) {},
				},
			)

			// Emit first value
			currentTime = telem.SecondTS
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			source.MarkFresh(0)
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(int) {},
				},
			)

			// Change to value 10
			currentTime = telem.SecondTS * 2
			*source.Output(0) = telem.NewSeriesV[uint8](10)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
			source.MarkFresh(0)
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(int) {},
				},
			)

			// Wait for stability
			currentTime = telem.SecondTS * 3
			*source.Output(0) = telem.NewSeriesV[uint8](10)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(3)
			source.MarkFresh(0)
			outputs := make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeTrue())

			stableNode := s.Node("stable")
			output := stableNode.Output(0)
			outputVals := output.Unmarshal[uint8]()
			Expect(outputVals).To(Equal([]uint8{10}))
		})

		It("Should handle multiple values in single input", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					Inputs: types.Params{
						{
							Name:  "duration",
							Type:  types.TimeSpan(),
							Value: telem.SecondTS,
						},
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send multiple values, ending with 7 at 0.4s (400ms)
			*source.Output(0) = telem.NewSeriesV[uint8](3, 4, 5, 6, 7)
			*source.OutputTime(0) = telem.NewSeriesV(
				0,
				telem.SecondTS/10,   // 0.1s = 100ms
				telem.SecondTS/5,    // 0.2s = 200ms
				telem.SecondTS*3/10, // 0.3s = 300ms
				telem.SecondTS*2/5,  // 0.4s = 400ms
			)
			source.MarkFresh(0)
			n := MustSucceed(module.Create(cfg))
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(int) {},
				},
			)

			// Should track last value (7) with time 0.4s, so a sample at 1.4s closes
			currentTime = telem.SecondTS + telem.SecondTS*2/5 // 1.4s
			*source.Output(0) = telem.NewSeriesV[uint8](7)
			*source.OutputTime(0) = telem.NewSeriesV(currentTime)
			source.MarkFresh(0)
			outputs := make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeTrue())

			stableNode := s.Node("stable")
			output := stableNode.Output(0)
			outputVals := output.Unmarshal[uint8]()
			Expect(outputVals).To(Equal([]uint8{7}))
		})

		It(
			"Should stamp the output with the newest input sample, not the cycle",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "stable_for",
						Inputs: types.Params{
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.SecondTS,
							},
						},
					},
					State: s.Node("stable"),
				}
				source := s.Node("source")
				currentTime = 0
				*source.Output(0) = telem.NewSeriesV[uint8](5)
				*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
				source.MarkFresh(0)
				n := MustSucceed(module.Create(cfg))
				n.Next(
					node.Context{
						Context:     ctx,
						Now:         currentTime,
						MarkChanged: func(int) {},
					},
				)

				currentTime = telem.SecondTS * 100 // Set current time far in future
				*source.Output(0) = telem.NewSeriesV[uint8](5)
				*source.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
				source.MarkFresh(0)
				outputs := make(set.Set[int])
				n.Next(
					node.Context{
						Context:     ctx,
						Now:         currentTime,
						MarkChanged: func(i int) { outputs.Add(i) },
					},
				)
				Expect(outputs.Contains(0)).To(BeTrue())

				stableNode := s.Node("stable")
				outputTime := stableNode.OutputTime(0)
				outputTimes := outputTime.Unmarshal[telem.TimeStamp]()
				Expect(outputTimes).To(Equal([]telem.TimeStamp{telem.SecondTS * 2}))
			},
		)

		It("Should handle same value repeated in input", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					Inputs: types.Params{
						{
							Name:  "duration",
							Type:  types.TimeSpan(),
							Value: telem.SecondTS,
						},
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send same value multiple times
			*source.Output(0) = telem.NewSeriesV[uint8](5, 5, 5, 5)
			*source.OutputTime(0) = telem.NewSeriesV(
				0,
				telem.SecondTS/10,   // 0.1s = 100ms
				telem.SecondTS/5,    // 0.2s = 200ms
				telem.SecondTS*3/10, // 0.3s = 300ms
			)
			source.MarkFresh(0)
			n := MustSucceed(module.Create(cfg))
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(int) {},
				},
			)

			// Should use time from first occurrence (0), so a sample at 1s closes
			currentTime = telem.SecondTS
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			source.MarkFresh(0)
			outputs := make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         currentTime,
					MarkChanged: func(i int) { outputs.Add(i) },
				},
			)
			Expect(outputs.Contains(0)).To(BeTrue())
		})
	})

	Describe("Symbols", func() {
		var root *symbol.Symbol
		BeforeEach(func() { root = symbol.NewRoot(nil, stable.NewSymbols()) })
		It("Should expose bare stable_for symbol", func(ctx SpecContext) {
			sym := MustSucceed(root.Resolve(ctx, "stable_for", symbol.IncludeInternal))
			Expect(sym.Name).To(Equal("stable_for"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
		})
		It("Should expose qualified stable.for symbol", func(ctx SpecContext) {
			mod := MustSucceed(root.Resolve(ctx, "stable", symbol.IncludeInternal))
			sym := MustSucceed(mod.Resolve(ctx, "for", symbol.IncludeInternal))
			Expect(sym.Name).To(Equal("for"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
		})
	})
	Describe("Factory", func() {
		It(
			"Should create node for stable.for via CompoundFactory",
			func(ctx SpecContext) {
				g := graph.Graph{
					Nodes: []graph.Node{
						{Key: "source"},
						{Key: "stable"},
					},
					Inputs: map[string]msgpack.EncodedJSON{
						"source": {"type": "source"},
						"stable": {
							"type":     "stable_for",
							"duration": int(telem.Second),
						},
					},
					Edges: graph.Edges{
						{
							Source: ir.Handle{
								Node:  "source",
								Param: ir.DefaultOutputParam,
							},
							Target: ir.Handle{
								Node:  "stable",
								Param: ir.DefaultInputParam,
							},
						},
					},
					Functions: []ir.Function{
						{
							Key: "source",
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.U8()},
							},
						},
						{
							Key: "stable_for",
							Inputs: types.Params{
								{Name: "duration", Type: types.TimeSpan()},
								{Name: ir.DefaultInputParam, Type: types.U8()},
							},
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.U8()},
							},
						},
					},
				}
				analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				s := node.New(analyzed)
				compound := node.CompoundFactory{stable.NewHost()}
				irNode := analyzed.Nodes[1]
				irNode.Type = "stable.for"
				n := MustSucceed(compound.Create(node.Config{
					Node:  irNode,
					State: s.Node("stable"),
				}))
				Expect(n).ToNot(BeNil())
			},
		)
	})

	Describe("Reset", func() {
		It(
			"Should not emit after reset when no new data arrives",
			func(ctx SpecContext) {
				cfg := node.Config{Node: irNode, State: s.Node("stable")}
				source := s.Node("source")
				currentTime = 0
				*source.Output(0) = telem.NewSeriesV[uint8](5)
				*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
				source.MarkFresh(0)
				n := MustSucceed(module.Create(cfg))
				outputs := make(set.Set[int])
				n.Next(
					node.Context{
						Context:     ctx,
						Now:         currentTime,
						MarkChanged: func(i int) { outputs.Add(i) },
					},
				)
				Expect(outputs.Contains(0)).To(BeFalse())

				currentTime = telem.SecondTS * 2
				*source.Output(0) = telem.NewSeriesV[uint8](5)
				*source.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
				source.MarkFresh(0)
				outputs = make(set.Set[int])
				n.Next(
					node.Context{
						Context:     ctx,
						Now:         currentTime,
						MarkChanged: func(i int) { outputs.Add(i) },
					},
				)
				Expect(outputs.Contains(0)).To(BeTrue())

				n.Reset(node.Context{})
				currentTime = telem.SecondTS * 5
				outputs = make(set.Set[int])
				n.Next(
					node.Context{
						Context:     ctx,
						Now:         currentTime,
						MarkChanged: func(i int) { outputs.Add(i) },
					},
				)
				Expect(outputs.Contains(0)).To(BeFalse())
			},
		)

		It(
			"Should ignore values written while the stage was inactive",
			func(ctx SpecContext) {
				cfg := node.Config{Node: irNode, State: s.Node("stable")}
				source := s.Node("source")
				currentTime = 0
				*source.Output(0) = telem.NewSeriesV[uint8](5)
				*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
				source.MarkFresh(0)
				n := MustSucceed(module.Create(cfg))
				n.Next(
					node.Context{
						Context:     ctx,
						Now:         currentTime,
						MarkChanged: func(int) {},
					},
				)

				*source.Output(0) = telem.NewSeriesV[uint8](9)
				*source.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
				source.MarkFresh(0)
				n.Reset(node.Context{})

				currentTime = telem.SecondTS * 10
				outputs := make(set.Set[int])
				n.Next(
					node.Context{
						Context:     ctx,
						Now:         currentTime,
						MarkChanged: func(i int) { outputs.Add(i) },
					},
				)
				Expect(outputs.Contains(0)).To(BeFalse())
			},
		)
	})
})

var _ = Describe("Construction validation", func() {
	It(
		"Should error at construction when the input param is missing",
		func() {
			prog := ir.IR{Nodes: ir.Nodes{{
				Key:  "stable",
				Type: "stable_for",
				Inputs: types.Params{
					{
						Name:  "duration",
						Type:  types.TimeSpan(),
						Value: telem.TimeSpanZero,
					},
				},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			}}}
			s := node.New(prog)
			cfg := node.Config{Node: prog.Nodes[0], State: s.Node("stable")}
			Expect(stable.NewHost().Create(cfg)).Error().
				To(MatchError(node.ErrInputNotFound))
		},
	)
	It(
		"Should error at construction when the duration input value is invalid",
		func() {
			prog := ir.IR{Nodes: ir.Nodes{{
				Key:  "stable",
				Type: "stable_for",
				Inputs: types.Params{
					{Name: "duration", Type: types.String(), Value: []any{1}},
					{Name: ir.DefaultInputParam, Type: types.F32(), Value: float32(0)},
				},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			}}}
			s := node.New(prog)
			cfg := node.Config{Node: prog.Nodes[0], State: s.Node("stable")}
			Expect(
				stable.NewHost().Create(cfg),
			).Error().
				To(BeAValidationPathError())
		},
	)
})

var _ = Describe("Variable duration", func() {
	var (
		module      *stable.Host
		s           *node.ProgramState
		cfg         node.Config
		setDuration func(telem.TimeSpan)
		currentTime telem.TimeStamp
	)
	BeforeEach(func() {
		currentTime = 0
		module = stable.NewHost()
	})
	// buildUnits wires source -> stable with a var-bound duration: Value holds the
	// declared initial and setDuration writes the variable's live slot.
	buildUnits := func(initial telem.TimeSpan, from, to *types.Unit) {
		elem := types.I64()
		elem.Unit = from
		ref := types.VarRef(elem, "v")
		ref.Unit = to
		prog := ir.IR{
			Nodes: ir.Nodes{
				{
					Key:  "source",
					Type: "source",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
				},
				{
					Key:  "v",
					Type: "variable",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: elem},
					},
				},
				{
					Key:  "stable",
					Type: "stable_for",
					Inputs: types.Params{
						{Name: "duration", Type: ref, Value: initial},
						{Name: ir.DefaultInputParam, Type: types.U8()},
					},
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
				},
			},
			Edges: ir.Edges{{
				Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
			}},
		}
		s = node.New(prog)
		cfg = node.Config{Node: prog.Nodes.Get("stable"), State: s.Node("stable")}
		setDuration = func(d telem.TimeSpan) {
			*s.Node("v").Output(0) = telem.NewSeriesV(int64(d))
		}
	}
	build := func(initial telem.TimeSpan) { buildUnits(initial, nil, nil) }
	ingest := func(v uint8, at telem.TimeStamp) {
		*s.Node("source").Output(0) = telem.NewSeriesV(v)
		*s.Node("source").OutputTime(0) = telem.NewSeriesV(at)
		s.Node("source").MarkFresh(0)
	}
	next := func(ctx SpecContext, n node.Node) bool {
		fired := false
		n.Next(
			node.Context{
				Context:     ctx,
				Now:         currentTime,
				MarkChanged: func(int) { fired = true },
			},
		)
		return fired
	}

	It(
		"Should convert a live duration to the unit of the input",
		func(ctx SpecContext) {
			buildUnits(
				telem.Second,
				&types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"},
				&types.Unit{Dimensions: types.DimTime, Scale: 1, Name: "ns"},
			)
			n := MustSucceed(module.Create(cfg))
			// The variable holds 2 in seconds.
			setDuration(2)
			ingest(5, telem.SecondTS)
			currentTime = telem.SecondTS
			Expect(next(ctx, n)).To(BeFalse())
			currentTime = 2*telem.SecondTS + telem.SecondTS/2
			ingest(5, currentTime)
			Expect(next(ctx, n)).To(BeFalse())
			currentTime = 3 * telem.SecondTS
			ingest(5, currentTime)
			Expect(next(ctx, n)).To(BeTrue())
		},
	)

	It("Should honor the declared initial before any write", func(ctx SpecContext) {
		build(telem.Second)
		n := MustSucceed(module.Create(cfg))
		ingest(5, telem.SecondTS)
		currentTime = telem.SecondTS
		Expect(next(ctx, n)).To(BeFalse())
		currentTime = telem.SecondTS + telem.SecondTS/2
		ingest(5, currentTime)
		Expect(next(ctx, n)).To(BeFalse())
		currentTime = 2 * telem.SecondTS
		ingest(5, currentTime)
		Expect(next(ctx, n)).To(BeTrue())
	})

	It(
		"Should adopt a shortening write at the next window, not mid-window",
		func(ctx SpecContext) {
			build(10 * telem.Second)
			n := MustSucceed(module.Create(cfg))
			ingest(5, telem.SecondTS)
			currentTime = telem.SecondTS
			Expect(next(ctx, n)).To(BeFalse())
			setDuration(telem.Second)
			// The in-flight window completes under its starting 10s.
			currentTime = 2 * telem.SecondTS
			ingest(5, currentTime)
			Expect(next(ctx, n)).To(BeFalse())
			currentTime = 11 * telem.SecondTS
			ingest(5, currentTime)
			Expect(next(ctx, n)).To(BeTrue())
			// The next window latches the shortened 1s.
			ingest(7, 12*telem.SecondTS)
			currentTime = 12 * telem.SecondTS
			Expect(next(ctx, n)).To(BeFalse())
			currentTime = 13 * telem.SecondTS
			ingest(7, currentTime)
			Expect(next(ctx, n)).To(BeTrue())
		},
	)

	It(
		"Should adopt a lengthening write at the next window, not mid-window",
		func(ctx SpecContext) {
			build(telem.Second)
			n := MustSucceed(module.Create(cfg))
			ingest(5, telem.SecondTS)
			currentTime = telem.SecondTS
			Expect(next(ctx, n)).To(BeFalse())
			setDuration(10 * telem.Second)
			// The in-flight window completes under its starting 1s.
			currentTime = 2 * telem.SecondTS
			ingest(5, currentTime)
			Expect(next(ctx, n)).To(BeTrue())
			// The next window latches the lengthened 10s.
			ingest(7, 3*telem.SecondTS)
			currentTime = 3 * telem.SecondTS
			Expect(next(ctx, n)).To(BeFalse())
			currentTime = 4 * telem.SecondTS
			ingest(7, currentTime)
			Expect(next(ctx, n)).To(BeFalse())
			currentTime = 13 * telem.SecondTS
			ingest(7, currentTime)
			Expect(next(ctx, n)).To(BeTrue())
			output := s.Node("stable").Output(0)
			Expect(output.Unmarshal[uint8]()).To(Equal([]uint8{7}))
		},
	)
})

// newTypedState builds a source -> stable_for graph where every param is t,
// so the emitted output must preserve t's full-width value.
func newTypedState(ctx context.Context, t types.Type) *node.ProgramState {
	GinkgoHelper()
	g := graph.Graph{
		Nodes: []graph.Node{{Key: "source"}, {Key: "stable"}},
		Inputs: map[string]msgpack.EncodedJSON{
			"source": {"type": "source"},
			"stable": {"type": "stable_for"},
		},
		Edges: graph.Edges{{
			Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
		}},
		Functions: []ir.Function{
			{Key: "source", Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: t},
			}},
			{
				Key:     "stable_for",
				Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: t}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: t}},
			},
		},
	}
	analyzed, diags := graph.Analyze(ctx, g, NewGraphRoot(nil))
	Expect(diags.Ok()).To(BeTrue())
	return node.New(analyzed)
}

var _ = Describe("StableFor type preservation", func() {
	DescribeTable(
		"Should debounce and emit the full-width value for any input type",
		func(ctx SpecContext, t types.Type, input telem.Series) {
			var now telem.TimeStamp
			module := stable.NewHost()
			state := newTypedState(ctx, t)
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					Inputs: types.Params{
						{
							Name:  "duration",
							Type:  types.TimeSpan(),
							Value: telem.SecondTS,
						},
					},
				},
				State: state.Node("stable"),
			}
			source := state.Node("source")
			n := MustSucceed(module.Create(cfg))

			now = 0
			*source.Output(0) = input
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			source.MarkFresh(0)
			fired := make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         now,
					MarkChanged: func(i int) { fired.Add(i) },
				},
			)
			Expect(fired.Contains(0)).To(BeFalse())

			now = telem.SecondTS * 2
			*source.Output(0) = input
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
			source.MarkFresh(0)
			fired = make(set.Set[int])
			n.Next(
				node.Context{
					Context:     ctx,
					Now:         now,
					MarkChanged: func(i int) { fired.Add(i) },
				},
			)
			Expect(fired.Contains(0)).To(BeTrue())

			out := state.Node("stable").Output(0)
			Expect(out.DataType).To(Equal(input.DataType))
			Expect(out.Data).To(Equal(input.Data))
		},
		Entry("u8", types.U8(), telem.NewSeriesV[uint8](42)),
		Entry("u16", types.U16(), telem.NewSeriesV[uint16](50_000)),
		Entry("u32", types.U32(), telem.NewSeriesV[uint32](3_000_000_000)),
		Entry("u64", types.U64(), telem.NewSeriesV[uint64](18_000_000_000_000_000_000)),
		Entry("i8", types.I8(), telem.NewSeriesV[int8](-42)),
		Entry("i16", types.I16(), telem.NewSeriesV[int16](-1_000)),
		Entry("i32", types.I32(), telem.NewSeriesV[int32](-100_000)),
		Entry("i64", types.I64(), telem.NewSeriesV[int64](-5_000_000_000)),
		Entry("f32", types.F32(), telem.NewSeriesV[float32](4.321)),
		Entry("f64", types.F64(), telem.NewSeriesV(4.321)),
		Entry("bool", types.Bool(), telem.NewSeriesV(true)),
	)
})

// The window is measured in sample time at both ends: a batch that is older than
// the window but spans less than it has not shown the value held.
var _ = Describe("Sample-time window", func() {
	var (
		s           *node.ProgramState
		n           node.Node
		currentTime telem.TimeStamp
	)
	build := func(duration telem.TimeSpan) {
		GinkgoHelper()
		prog := ir.IR{
			Nodes: ir.Nodes{
				{
					Key:  "source",
					Type: "source",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
				},
				{
					Key:  "stable",
					Type: "stable_for",
					Inputs: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: duration},
						{Name: ir.DefaultInputParam, Type: types.U8()},
					},
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
				},
			},
			Edges: ir.Edges{{
				Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
			}},
		}
		s = node.New(prog)
		n = MustSucceed(stable.NewHost().Create(node.Config{
			Node: prog.Nodes.Get("stable"), State: s.Node("stable"),
		}))
	}
	BeforeEach(func() {
		currentTime = telem.SecondTS
		build(10 * telem.Millisecond)
	})
	ingest := func(values []uint8, at []telem.TimeStamp) {
		*s.Node("source").Output(0) = telem.NewSeriesV(values...)
		*s.Node("source").OutputTime(0) = telem.NewSeriesV(at...)
		s.Node("source").MarkFresh(0)
	}
	next := func(ctx SpecContext) bool {
		fired := false
		n.Next(node.Context{
			Context:     ctx,
			Now:         currentTime,
			MarkChanged: func(int) { fired = true },
		})
		return fired
	}
	ms := func(v int) telem.TimeStamp { return telem.TimeStamp(v) * telem.MillisecondTS }
	outputStamp := func() telem.TimeStamp {
		return s.Node("stable").OutputTime(0).ValueAt[telem.TimeStamp](0)
	}
	outputStamps := func() []telem.TimeStamp {
		return s.Node("stable").OutputTime(0).Unmarshal[telem.TimeStamp]()
	}
	outputs := func() []uint8 { return s.Node("stable").Output(0).Unmarshal[uint8]() }

	It("Should not close on a batch older than the window but shorter than it",
		func(ctx SpecContext) {
			// At cycle stamp T, a batch stamped T-18ms to T-16ms shows a 2ms hold.
			ingest([]uint8{1, 1, 1}, []telem.TimeStamp{
				currentTime - ms(18), currentTime - ms(17), currentTime - ms(16),
			})
			Expect(next(ctx)).To(BeFalse())
		},
	)

	It("Should close once on the sample that shows the hold, stamped at that sample",
		func(ctx SpecContext) {
			ingest([]uint8{1, 1, 1}, []telem.TimeStamp{
				currentTime - ms(18), currentTime - ms(17), currentTime - ms(16),
			})
			Expect(next(ctx)).To(BeFalse())
			ingest([]uint8{1, 1}, []telem.TimeStamp{
				currentTime - ms(8), currentTime - ms(7),
			})
			Expect(next(ctx)).To(BeTrue())
			Expect(s.Node("stable").Output(0).Unmarshal[uint8]()).To(Equal([]uint8{1}))
			Expect(outputStamp()).To(Equal(currentTime - ms(8)))
			ingest([]uint8{1}, []telem.TimeStamp{currentTime - ms(5)})
			Expect(next(ctx)).To(BeFalse())
		},
	)

	It("Should not close when the value changes back inside the window",
		func(ctx SpecContext) {
			ingest([]uint8{1}, []telem.TimeStamp{ms(0)})
			Expect(next(ctx)).To(BeFalse())
			ingest([]uint8{0, 1}, []telem.TimeStamp{ms(5), ms(8)})
			Expect(next(ctx)).To(BeFalse())
			ingest([]uint8{1}, []telem.TimeStamp{ms(15)})
			Expect(next(ctx)).To(BeFalse())
			ingest([]uint8{1}, []telem.TimeStamp{ms(18)})
			Expect(next(ctx)).To(BeTrue())
			Expect(outputStamp()).To(Equal(ms(18)))
		},
	)

	It("Should not close on a cycle with no new input", func(ctx SpecContext) {
		ingest([]uint8{1}, []telem.TimeStamp{ms(0)})
		Expect(next(ctx)).To(BeFalse())
		currentTime = 100 * telem.SecondTS
		Expect(next(ctx)).To(BeFalse())
	})

	It("Should emit a value confirmed inside a batch that then changes",
		func(ctx SpecContext) {
			ingest([]uint8{1, 1, 1}, []telem.TimeStamp{ms(0), ms(1), ms(2)})
			Expect(next(ctx)).To(BeFalse())
			ingest([]uint8{1, 2, 2}, []telem.TimeStamp{ms(5000), ms(5001), ms(5002)})
			Expect(next(ctx)).To(BeTrue())
			Expect(outputs()).To(Equal([]uint8{1}))
			Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(5000)}))
		},
	)

	It("Should emit every value confirmed inside one batch, each at its sample",
		func(ctx SpecContext) {
			ingest(
				[]uint8{1, 1, 2, 2},
				[]telem.TimeStamp{ms(0), ms(20), ms(30), ms(50)},
			)
			Expect(next(ctx)).To(BeTrue())
			Expect(outputs()).To(Equal([]uint8{1, 2}))
			Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(20), ms(50)}))
		},
	)

	It("Should emit a value at most once across batches", func(ctx SpecContext) {
		ingest([]uint8{1}, []telem.TimeStamp{ms(0)})
		Expect(next(ctx)).To(BeFalse())
		ingest([]uint8{1}, []telem.TimeStamp{ms(20)})
		Expect(next(ctx)).To(BeTrue())
		ingest([]uint8{1, 1}, []telem.TimeStamp{ms(40), ms(60)})
		Expect(next(ctx)).To(BeFalse())
	})

	It("Should emit a value again after a different value was sent",
		func(ctx SpecContext) {
			ingest([]uint8{1, 1}, []telem.TimeStamp{ms(0), ms(20)})
			Expect(next(ctx)).To(BeTrue())
			Expect(outputs()).To(Equal([]uint8{1}))
			ingest([]uint8{2, 2}, []telem.TimeStamp{ms(30), ms(50)})
			Expect(next(ctx)).To(BeTrue())
			Expect(outputs()).To(Equal([]uint8{2}))
			ingest([]uint8{1, 1}, []telem.TimeStamp{ms(60), ms(80)})
			Expect(next(ctx)).To(BeTrue())
			Expect(outputs()).To(Equal([]uint8{1}))
			Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(80)}))
		},
	)

	It("Should not emit a value when a change precedes its confirmation in the batch",
		func(ctx SpecContext) {
			ingest([]uint8{1, 1, 1}, []telem.TimeStamp{ms(0), ms(1), ms(2)})
			Expect(next(ctx)).To(BeFalse())
			ingest([]uint8{2, 1}, []telem.TimeStamp{ms(5000), ms(5001)})
			Expect(next(ctx)).To(BeFalse())
		},
	)

	// A virtual channel has no index, so the runtime stamps each batch at the cycle
	// that reads it, 1ns apart inside the batch.
	Describe("Cycle-stamped batches", func() {
		ns := func(v int) telem.TimeStamp { return telem.TimeStamp(v) }

		It("Should not close inside one batch", func(ctx SpecContext) {
			ingest([]uint8{1, 1, 1}, []telem.TimeStamp{ms(0), ns(1), ns(2)})
			Expect(next(ctx)).To(BeFalse())
		})

		It("Should close on the first batch stamped a window later",
			func(ctx SpecContext) {
				ingest([]uint8{1, 1, 1}, []telem.TimeStamp{ms(0), ns(1), ns(2)})
				Expect(next(ctx)).To(BeFalse())
				ingest([]uint8{1, 1}, []telem.TimeStamp{ms(5), ms(5) + ns(1)})
				Expect(next(ctx)).To(BeFalse())
				ingest([]uint8{1, 1}, []telem.TimeStamp{ms(10), ms(10) + ns(1)})
				Expect(next(ctx)).To(BeTrue())
				Expect(outputs()).To(Equal([]uint8{1}))
				Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(10)}))
				ingest([]uint8{1}, []telem.TimeStamp{ms(15)})
				Expect(next(ctx)).To(BeFalse())
			},
		)

		It("Should emit each value its own batch confirms across many batches",
			func(ctx SpecContext) {
				ingest([]uint8{1, 1}, []telem.TimeStamp{ms(0), ns(1)})
				Expect(next(ctx)).To(BeFalse())
				ingest([]uint8{1, 2, 2}, []telem.TimeStamp{
					ms(10), ms(10) + ns(1), ms(10) + ns(2),
				})
				Expect(next(ctx)).To(BeTrue())
				Expect(outputs()).To(Equal([]uint8{1}))
				Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(10)}))
				ingest([]uint8{2}, []telem.TimeStamp{ms(15)})
				Expect(next(ctx)).To(BeFalse())
				ingest([]uint8{2, 1}, []telem.TimeStamp{ms(21), ms(21) + ns(1)})
				Expect(next(ctx)).To(BeTrue())
				Expect(outputs()).To(Equal([]uint8{2}))
				Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(21)}))
				ingest([]uint8{1, 1}, []telem.TimeStamp{ms(32), ms(32) + ns(1)})
				Expect(next(ctx)).To(BeTrue())
				Expect(outputs()).To(Equal([]uint8{1}))
				Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(32)}))
			},
		)

		It("Should not close across a change inside an earlier batch",
			func(ctx SpecContext) {
				ingest([]uint8{1}, []telem.TimeStamp{ms(0)})
				Expect(next(ctx)).To(BeFalse())
				ingest([]uint8{2, 1}, []telem.TimeStamp{ms(5), ms(5) + ns(1)})
				Expect(next(ctx)).To(BeFalse())
				ingest([]uint8{1}, []telem.TimeStamp{ms(12)})
				Expect(next(ctx)).To(BeFalse())
				ingest([]uint8{1, 1}, []telem.TimeStamp{ms(16), ms(16) + ns(1)})
				Expect(next(ctx)).To(BeTrue())
				Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(16)}))
			},
		)
	})

	It("Should emit on the changing sample when the duration is zero",
		func(ctx SpecContext) {
			build(0)
			ingest([]uint8{1}, []telem.TimeStamp{ms(0)})
			Expect(next(ctx)).To(BeTrue())
			Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(0)}))
			ingest([]uint8{1, 2}, []telem.TimeStamp{ms(1), ms(2)})
			Expect(next(ctx)).To(BeTrue())
			Expect(outputs()).To(Equal([]uint8{2}))
			Expect(outputStamps()).To(Equal([]telem.TimeStamp{ms(2)}))
		},
	)
})
