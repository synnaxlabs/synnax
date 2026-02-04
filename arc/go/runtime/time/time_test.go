// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package time_test

import (
	"context"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	arctime "github.com/synnaxlabs/arc/runtime/time"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

var _ = Describe("Time", func() {
	Describe("NewFactory", func() {
		It("Should create factory with max timing base", func() {
			factory := arctime.NewFactory()
			Expect(factory).ToNot(BeNil())
		})
	})
	Describe("Interval", func() {
		var factory *arctime.Factory
		var s *state.State
		var changedOutputs []string
		BeforeEach(func() {
			factory = arctime.NewFactory()
			changedOutputs = []string{}
			g := graph.Graph{
				Nodes: []graph.Node{{
					Key:  "interval_1",
					Type: "interval",
					Config: map[string]any{
						"period": int64(telem.Second),
					},
				}},
				Functions: []graph.Function{{
					Key: "interval",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
					Config: types.Params{
						{Name: "period", Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, arctime.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		It("Should create node for interval type", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Config: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("interval_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("interval_1"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(query.ErrNotFound))
		})
		It("Should fire immediately on first tick", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Config: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("interval_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			intervalNode := s.Node("interval_1")
			*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
			*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})

			Expect(changedOutputs).To(HaveLen(1))
			Expect(changedOutputs[0]).To(Equal(ir.DefaultOutputParam))
		})
		It("Should not fire before period elapses", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Config: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("interval_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			intervalNode := s.Node("interval_1")
			*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
			*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			// First tick at 0 - fires
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Second tick at 500ms - should not fire (period is 1s)
			changedOutputs = []string{}
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 500 * telem.Millisecond,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
		It("Should fire after period elapses", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Config: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("interval_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			intervalNode := s.Node("interval_1")
			*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
			*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			// First tick at 0 - fires
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Second tick at 1s - should fire
			changedOutputs = []string{}
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(HaveLen(1))
		})
		It("Should update timing base", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Config: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: 100 * telem.Millisecond},
					},
				},
				State: s.Node("interval_1"),
			}
			_, _ = factory.Create(ctx, cfg)
			Expect(factory.BaseInterval).To(Equal(100 * telem.Millisecond))
		})
		It("Should not fire on channel input even when period elapsed", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Config: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("interval_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			intervalNode := s.Node("interval_1")
			*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
			*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			n.Next(node.Context{
				Context: ctx,
				Elapsed: 2 * telem.Second,
				Reason:  node.ReasonChannelInput,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
	})
	Describe("Wait", func() {
		var factory *arctime.Factory
		var s *state.State
		var changedOutputs []string
		BeforeEach(func() {
			factory = arctime.NewFactory()
			changedOutputs = []string{}
			g := graph.Graph{
				Nodes: []graph.Node{{
					Key:  "wait_1",
					Type: "wait",
					Config: map[string]any{
						"duration": int64(telem.Second),
					},
				}},
				Functions: []graph.Function{{
					Key: "wait",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
					Config: types.Params{
						{Name: "duration", Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, arctime.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		It("Should create node for wait type", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Config: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
		It("Should not fire before duration elapses", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Config: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			waitNode := s.Node("wait_1")
			*waitNode.Output(0) = telem.NewSeriesV[uint8]()
			*waitNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			// Tick at 500ms - should not fire
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 500 * telem.Millisecond,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
		It("Should fire once after duration elapses", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Config: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			waitNode := s.Node("wait_1")
			*waitNode.Output(0) = telem.NewSeriesV[uint8]()
			*waitNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			// First tick at 0 to set start time
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(BeEmpty())

			// Tick at 1s - should fire
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(HaveLen(1))
			Expect(changedOutputs[0]).To(Equal(ir.DefaultOutputParam))
		})
		It("Should only fire once", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Config: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			waitNode := s.Node("wait_1")
			*waitNode.Output(0) = telem.NewSeriesV[uint8]()
			*waitNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			// First tick at 0 to set start time
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})

			// Tick at 1s - fires
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Tick at 2s - should not fire again
			changedOutputs = []string{}
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 2 * telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
		It("Should be resettable", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Config: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			waitNode := s.Node("wait_1")
			*waitNode.Output(0) = telem.NewSeriesV[uint8]()
			*waitNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			// First tick at 0
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})

			// Tick at 1s - fires
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Reset - all nodes now implement Reset() directly
			n.Reset()

			// Tick at 1.5s - should not fire (reset at 1s, duration is 1s)
			changedOutputs = []string{}
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 1500 * telem.Millisecond,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(BeEmpty())

			// Tick at 2.5s - should fire (start was reset at ~1.5s tick, 1s elapsed)
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 2500 * telem.Millisecond,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(HaveLen(1))
		})
		It("Should not fire on channel input even when duration elapsed", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Config: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			waitNode := s.Node("wait_1")
			*waitNode.Output(0) = telem.NewSeriesV[uint8]()
			*waitNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			// First tick at 0 to set start time
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(BeEmpty())

			n.Next(node.Context{
				Context: ctx,
				Elapsed: 2 * telem.Second,
				Reason:  node.ReasonChannelInput,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
	})
	Describe("TimingBase", func() {
		It("Should compute GCD of multiple intervals", func() {
			factory := arctime.NewFactory()
			g := graph.Graph{
				Nodes: []graph.Node{
					{
						Key:  "interval_1",
						Type: "interval",
						Config: map[string]any{
							"period": int64(100 * telem.Millisecond),
						},
					},
					{
						Key:  "interval_2",
						Type: "interval",
						Config: map[string]any{
							"period": int64(150 * telem.Millisecond),
						},
					},
				},
				Functions: []graph.Function{{
					Key: "interval",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
					Config: types.Params{
						{Name: "period", Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, arctime.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})

			// Create first interval with 100ms period
			cfg1 := node.Config{
				Node: ir.Node{
					Type: "interval",
					Config: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: 100 * telem.Millisecond},
					},
				},
				State: s.Node("interval_1"),
			}
			_, _ = factory.Create(ctx, cfg1)
			Expect(factory.BaseInterval).To(Equal(100 * telem.Millisecond))

			// Create second interval with 150ms period
			cfg2 := node.Config{
				Node: ir.Node{
					Type: "interval",
					Config: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: 150 * telem.Millisecond},
					},
				},
				State: s.Node("interval_2"),
			}
			_, _ = factory.Create(ctx, cfg2)
			// GCD(100ms, 150ms) = 50ms
			Expect(factory.BaseInterval).To(Equal(50 * telem.Millisecond))
		})
	})
	Describe("SymbolResolver", func() {
		It("Should resolve interval symbol", func() {
			sym, ok := arctime.SymbolResolver["interval"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("interval"))
		})
		It("Should resolve wait symbol", func() {
			sym, ok := arctime.SymbolResolver["wait"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("wait"))
		})
	})
	Describe("CalculateTolerance", func() {
		It("Should return half of base interval for 100ms", func() {
			tolerance := arctime.CalculateTolerance(100 * telem.Millisecond)
			Expect(tolerance).To(Equal(50 * telem.Millisecond))
		})
		It("Should return MinTolerance when half interval is less than MinTolerance", func() {
			tolerance := arctime.CalculateTolerance(2 * telem.Millisecond)
			Expect(tolerance).To(Equal(arctime.MinTolerance))
		})
		It("Should return MinTolerance for MaxInt64 base interval", func() {
			tolerance := arctime.CalculateTolerance(telem.TimeSpan(math.MaxInt64))
			Expect(tolerance).To(Equal(arctime.MinTolerance))
		})
		It("Should return exactly MinTolerance when half equals MinTolerance", func() {
			tolerance := arctime.CalculateTolerance(2 * arctime.MinTolerance)
			Expect(tolerance).To(Equal(arctime.MinTolerance))
		})
	})
	Describe("Tolerance Behavior", func() {
		var factory *arctime.Factory
		var s *state.State
		var changedOutputs []string
		BeforeEach(func() {
			factory = arctime.NewFactory()
			changedOutputs = []string{}
			g := graph.Graph{
				Nodes: []graph.Node{{
					Key:  "interval_1",
					Type: "interval",
					Config: map[string]any{
						"period": int64(100 * telem.Millisecond),
					},
				}},
				Functions: []graph.Function{{
					Key: "interval",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
					Config: types.Params{
						{Name: "period", Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, arctime.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
		})
		Describe("Interval with tolerance", func() {
			It("Should fire on early tick within tolerance", func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Config: types.Params{
							{Name: "period", Type: types.TimeSpan(), Value: 100 * telem.Millisecond},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(factory.Create(ctx, cfg))
				intervalNode := s.Node("interval_1")
				*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
				*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

				tolerance := telem.TimeSpan(50 * telem.Millisecond)
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   0,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(output string) {
						changedOutputs = append(changedOutputs, output)
					},
				})
				Expect(changedOutputs).To(HaveLen(1))

				changedOutputs = []string{}
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   telem.TimeSpan(99500 * telem.Microsecond),
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(output string) {
						changedOutputs = append(changedOutputs, output)
					},
				})
				Expect(changedOutputs).To(HaveLen(1))
			})
			It("Should not fire too early beyond tolerance", func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Config: types.Params{
							{Name: "period", Type: types.TimeSpan(), Value: 100 * telem.Millisecond},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(factory.Create(ctx, cfg))
				intervalNode := s.Node("interval_1")
				*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
				*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

				tolerance := telem.TimeSpan(50 * telem.Millisecond)
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   0,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(output string) {
						changedOutputs = append(changedOutputs, output)
					},
				})
				Expect(changedOutputs).To(HaveLen(1))

				changedOutputs = []string{}
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   40 * telem.Millisecond,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(output string) {
						changedOutputs = append(changedOutputs, output)
					},
				})
				Expect(changedOutputs).To(BeEmpty())
			})
			It("Should handle jitter simulation with correct firings", func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Config: types.Params{
							{Name: "period", Type: types.TimeSpan(), Value: 100 * telem.Millisecond},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(factory.Create(ctx, cfg))
				intervalNode := s.Node("interval_1")
				*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
				*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

				tolerance := telem.TimeSpan(50 * telem.Millisecond)
				fireCount := 0
				tickTimes := []telem.TimeSpan{
					0,
					telem.TimeSpan(99500 * telem.Microsecond),
					telem.TimeSpan(199800 * telem.Microsecond),
					telem.TimeSpan(300100 * telem.Microsecond),
					telem.TimeSpan(399000 * telem.Microsecond),
				}

				for _, elapsed := range tickTimes {
					n.Next(node.Context{
						Context:   ctx,
						Elapsed:   elapsed,
						Tolerance: tolerance,
						Reason:    node.ReasonTimerTick,
						MarkChanged: func(output string) {
							fireCount++
						},
					})
				}
				Expect(fireCount).To(Equal(5))
			})
			It("Should use MinTolerance floor for OS jitter", func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Config: types.Params{
							{Name: "period", Type: types.TimeSpan(), Value: 100 * telem.Millisecond},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(factory.Create(ctx, cfg))
				intervalNode := s.Node("interval_1")
				*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
				*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

				tolerance := arctime.MinTolerance
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   0,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(output string) {
						changedOutputs = append(changedOutputs, output)
					},
				})
				Expect(changedOutputs).To(HaveLen(1))

				changedOutputs = []string{}
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   96 * telem.Millisecond,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(output string) {
						changedOutputs = append(changedOutputs, output)
					},
				})
				Expect(changedOutputs).To(HaveLen(1))
			})
		})
		Describe("Wait with tolerance", func() {
			It("Should fire early within tolerance", func() {
				g := graph.Graph{
					Nodes: []graph.Node{{
						Key:  "wait_1",
						Type: "wait",
						Config: map[string]any{
							"duration": int64(100 * telem.Millisecond),
						},
					}},
					Functions: []graph.Function{{
						Key: "wait",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
						Config: types.Params{
							{Name: "duration", Type: types.I64()},
						},
					}},
				}
				analyzed, diagnostics := graph.Analyze(ctx, g, arctime.SymbolResolver)
				Expect(diagnostics.Ok()).To(BeTrue())
				waitState := state.New(state.Config{IR: analyzed})
				waitFactory := arctime.NewFactory()

				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Config: types.Params{
							{Name: "duration", Type: types.TimeSpan(), Value: 100 * telem.Millisecond},
						},
					},
					State: waitState.Node("wait_1"),
				}
				n := MustSucceed(waitFactory.Create(ctx, cfg))
				waitNode := waitState.Node("wait_1")
				*waitNode.Output(0) = telem.NewSeriesV[uint8]()
				*waitNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

				tolerance := telem.TimeSpan(50 * telem.Millisecond)
				var waitChangedOutputs []string

				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   0,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(output string) {
						waitChangedOutputs = append(waitChangedOutputs, output)
					},
				})
				Expect(waitChangedOutputs).To(BeEmpty())

				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   telem.TimeSpan(99500 * telem.Microsecond),
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(output string) {
						waitChangedOutputs = append(waitChangedOutputs, output)
					},
				})
				Expect(waitChangedOutputs).To(HaveLen(1))
			})
		})
	})
})
