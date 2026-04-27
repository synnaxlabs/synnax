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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

var _ = Describe("Time", func() {
	Describe("NewModule", func() {
		It("Should create module with max timing base", func(ctx SpecContext) {
			factory := MustSucceed(time.NewModule(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			Expect(factory).ToNot(BeNil())
		})
	})
	Describe("Interval", func() {
		var factory *time.Module
		var s *node.ProgramState
		var changedOutputs []int
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewModule(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			changedOutputs = nil
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
			analyzed, diagnostics := graph.Analyze(ctx, g, time.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
		})
		It("Should create node for interval type", func(ctx SpecContext) {
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
		It("Should create node for qualified time.interval via CompoundFactory", func(ctx SpecContext) {
			compound := node.CompoundFactory{factory}
			cfg := node.Config{
				Node: ir.Node{
					Type: "time.interval",
					Config: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("interval_1"),
			}
			n := MustSucceed(compound.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
		It("Should return NotFound for unknown type", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("interval_1"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(query.ErrNotFound))
		})
		It("Should fire immediately on first tick", func(ctx SpecContext) {
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})

			Expect(changedOutputs).To(HaveLen(1))
			Expect(changedOutputs[0]).To(Equal(0))
		})
		It("Should not fire before period elapses", func(ctx SpecContext) {
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Second tick at 500ms - should not fire (period is 1s)
			changedOutputs = nil
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 500 * telem.Millisecond,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
		It("Should fire after period elapses", func(ctx SpecContext) {
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Second tick at 1s - should fire
			changedOutputs = nil
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))
		})
		It("Should update timing base", func(ctx SpecContext) {
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
		It("Should not fire on channel input even when period elapsed", func(ctx SpecContext) {
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
		It("Should fire immediately after Reset even if period has not elapsed", func(ctx SpecContext) {
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

			// First tick at 0 - fires (initial fire)
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Second tick at 1s - fires
			changedOutputs = nil
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Reset the interval (simulates stage re-entry)
			n.Reset()

			// Third tick at 1.5s - should fire because Reset set lastFired = -period
			changedOutputs = nil
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.TimeSpan(1500) * telem.Millisecond,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))
		})
	})
	Describe("Wait", func() {
		var factory *time.Module
		var s *node.ProgramState
		var changedOutputs []int
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewModule(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			changedOutputs = nil
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
			analyzed, diagnostics := graph.Analyze(ctx, g, time.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
		})
		It("Should create node for wait type", func(ctx SpecContext) {
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
		It("Should create node for qualified time.wait via CompoundFactory", func(ctx SpecContext) {
			compound := node.CompoundFactory{factory}
			cfg := node.Config{
				Node: ir.Node{
					Type: "time.wait",
					Config: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(compound.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
		It("Should not fire before duration elapses", func(ctx SpecContext) {
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
		It("Should fire once after duration elapses", func(ctx SpecContext) {
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())

			// Tick at 1s - should fire
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))
			Expect(changedOutputs[0]).To(Equal(0))
		})
		It("Should only fire once", func(ctx SpecContext) {
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})

			// Tick at 1s - fires
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Tick at 2s - should not fire again
			changedOutputs = nil
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 2 * telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
		It("Should be resettable", func(ctx SpecContext) {
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})

			// Tick at 1s - fires
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Reset - all nodes now implement Reset() directly
			n.Reset()

			// Tick at 1.5s - should not fire (reset at 1s, duration is 1s)
			changedOutputs = nil
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 1500 * telem.Millisecond,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())

			// Tick at 2.5s - should fire (start was reset at ~1.5s tick, 1s elapsed)
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 2500 * telem.Millisecond,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))
		})
		It("Should start timing from channel input that activates the stage", func(ctx SpecContext) {
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

			// Simulate stage activation via channel input at elapsed=5s.
			// The wait should record this as its start time even though it
			// does not fire on channel inputs.
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 5 * telem.Second,
				Reason:  node.ReasonChannelInput,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())

			// First timer tick at elapsed=6s (1s after stage activation).
			// The wait duration is 1s, so it should fire here.
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 6 * telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))
		})
		It("Should start timing from channel input after reset", func(ctx SpecContext) {
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

			// Fire once normally
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))

			// Reset simulates re-entering a stage
			n.Reset()
			changedOutputs = nil

			// Channel input at elapsed=2s sets the new start time
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 2 * telem.Second,
				Reason:  node.ReasonChannelInput,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())

			// Timer tick at elapsed=3s (1s after channel input). Should fire.
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 3 * telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))
		})
		It("Should call MarkSelfChanged when active but not yet fired", func(ctx SpecContext) {
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

			selfChangedCalls := 0
			// First tick at 0: starts timer, should call MarkSelfChanged
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {
					selfChangedCalls++
				},
				SetDeadline: func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())
			Expect(selfChangedCalls).To(Equal(1))

			// Tick at 500ms: still timing, should call MarkSelfChanged again
			selfChangedCalls = 0
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 500 * telem.Millisecond,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {
					selfChangedCalls++
				},
				SetDeadline: func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())
			Expect(selfChangedCalls).To(Equal(1))

			// Tick at 1s: fires, should NOT call MarkSelfChanged
			selfChangedCalls = 0
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {
					selfChangedCalls++
				},
				SetDeadline: func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))
			Expect(selfChangedCalls).To(Equal(0))
		})
		It("Should call MarkSelfChanged on channel input to survive non-tick cycles", func(ctx SpecContext) {
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

			selfChangedCalls := 0
			// First tick at 0: starts timer
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {
					selfChangedCalls++
				},
				SetDeadline: func(_ telem.TimeSpan) {},
			})
			Expect(selfChangedCalls).To(Equal(1))

			// Channel input at 200ms: should call MarkSelfChanged to stay alive
			selfChangedCalls = 0
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 200 * telem.Millisecond,
				Reason:  node.ReasonChannelInput,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {
					selfChangedCalls++
				},
				SetDeadline: func(_ telem.TimeSpan) {},
			})
			Expect(selfChangedCalls).To(Equal(1))
			Expect(changedOutputs).To(BeEmpty())

			// Timer tick at 1s: should fire normally (wasn't starved by channel input)
			selfChangedCalls = 0
			n.Next(node.Context{
				Context: ctx,
				Elapsed: telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {
					selfChangedCalls++
				},
				SetDeadline: func(_ telem.TimeSpan) {},
			})
			Expect(selfChangedCalls).To(Equal(0))
			Expect(changedOutputs).To(HaveLen(1))
		})
		It("Should not fire on channel input even when duration elapsed", func(ctx SpecContext) {
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())

			n.Next(node.Context{
				Context: ctx,
				Elapsed: 2 * telem.Second,
				Reason:  node.ReasonChannelInput,
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(BeEmpty())
		})
	})
	Describe("TimingBase", func() {
		It("Should compute GCD of multiple intervals", func(ctx SpecContext) {
			factory := MustSucceed(time.NewModule(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
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
			analyzed, diagnostics := graph.Analyze(ctx, g, time.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)

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
		It("Should resolve interval symbol", func(ctx SpecContext) {
			sym := MustSucceed(time.SymbolResolver.Resolve(ctx, "interval"))
			Expect(sym.Name).To(Equal("interval"))
		})
		It("Should resolve wait symbol", func(ctx SpecContext) {
			sym := MustSucceed(time.SymbolResolver.Resolve(ctx, "wait"))
			Expect(sym.Name).To(Equal("wait"))
		})
		It("Should resolve now symbol", func(ctx SpecContext) {
			sym := MustSucceed(time.SymbolResolver.Resolve(ctx, "now"))
			Expect(sym.Name).To(Equal("now"))
		})
		It("Should resolve now via module-qualified name", func(ctx SpecContext) {
			sym := MustSucceed(time.SymbolResolver.Resolve(ctx, "time.now"))
			Expect(sym.Name).To(Equal("now"))
		})
	})
	Describe("CalculateTolerance", func() {
		It("Should return half of base interval for 100ms", func(ctx SpecContext) {
			tolerance := time.CalculateTolerance(100 * telem.Millisecond)
			Expect(tolerance).To(Equal(50 * telem.Millisecond))
		})
		It("Should return MinTolerance when half interval is less than MinTolerance", func(ctx SpecContext) {
			tolerance := time.CalculateTolerance(2 * telem.Millisecond)
			Expect(tolerance).To(Equal(time.MinTolerance))
		})
		It("Should return MinTolerance for MaxInt64 base interval", func(ctx SpecContext) {
			tolerance := time.CalculateTolerance(telem.TimeSpan(math.MaxInt64))
			Expect(tolerance).To(Equal(time.MinTolerance))
		})
		It("Should return exactly MinTolerance when half equals MinTolerance", func(ctx SpecContext) {
			tolerance := time.CalculateTolerance(2 * time.MinTolerance)
			Expect(tolerance).To(Equal(time.MinTolerance))
		})
	})
	Describe("Tolerance Behavior", func() {
		var factory *time.Module
		var s *node.ProgramState
		var changedOutputs []int
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewModule(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			changedOutputs = nil
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
			analyzed, diagnostics := graph.Analyze(ctx, g, time.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
		})
		Describe("Interval with tolerance", func() {
			It("Should fire on early tick within tolerance", func(ctx SpecContext) {
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
					MarkChanged: func(i int) {
						changedOutputs = append(changedOutputs, i)
					},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				Expect(changedOutputs).To(HaveLen(1))

				changedOutputs = nil
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   telem.TimeSpan(99500 * telem.Microsecond),
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(i int) {
						changedOutputs = append(changedOutputs, i)
					},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				Expect(changedOutputs).To(HaveLen(1))
			})
			It("Should not fire too early beyond tolerance", func(ctx SpecContext) {
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
					MarkChanged: func(i int) {
						changedOutputs = append(changedOutputs, i)
					},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				Expect(changedOutputs).To(HaveLen(1))

				changedOutputs = nil
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   40 * telem.Millisecond,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(i int) {
						changedOutputs = append(changedOutputs, i)
					},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				Expect(changedOutputs).To(BeEmpty())
			})
			It("Should handle jitter simulation with correct firings", func(ctx SpecContext) {
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
						MarkChanged: func(int) {
							fireCount++
						},
						MarkSelfChanged: func() {},
						SetDeadline:     func(_ telem.TimeSpan) {},
					})
				}
				Expect(fireCount).To(Equal(5))
			})
			It("Should use MinTolerance floor for OS jitter", func(ctx SpecContext) {
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

				tolerance := time.MinTolerance
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   0,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(i int) {
						changedOutputs = append(changedOutputs, i)
					},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				Expect(changedOutputs).To(HaveLen(1))

				changedOutputs = nil
				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   96 * telem.Millisecond,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(i int) {
						changedOutputs = append(changedOutputs, i)
					},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				Expect(changedOutputs).To(HaveLen(1))
			})
		})
		Describe("Wait with tolerance", func() {
			It("Should fire early within tolerance", func(ctx SpecContext) {
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
				analyzed, diagnostics := graph.Analyze(ctx, g, time.SymbolResolver)
				Expect(diagnostics.Ok()).To(BeTrue())
				waitState := node.New(analyzed)
				waitFactory := MustSucceed(time.NewModule(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))

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
				var waitChangedOutputs []int

				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   0,
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(i int) {
						waitChangedOutputs = append(waitChangedOutputs, i)
					},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				Expect(waitChangedOutputs).To(BeEmpty())

				n.Next(node.Context{
					Context:   ctx,
					Elapsed:   telem.TimeSpan(99500 * telem.Microsecond),
					Tolerance: tolerance,
					Reason:    node.ReasonTimerTick,
					MarkChanged: func(i int) {
						waitChangedOutputs = append(waitChangedOutputs, i)
					},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				Expect(waitChangedOutputs).To(HaveLen(1))
			})
		})
	})
	Describe("Deadline Reporting", func() {
		Describe("Interval", func() {
			var factory *time.Module
			var s *node.ProgramState
			BeforeEach(func(ctx SpecContext) {
				factory = MustSucceed(time.NewModule(ctx, nil))
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
				analyzed, diagnostics := graph.Analyze(ctx, g, time.SymbolResolver)
				Expect(diagnostics.Ok()).To(BeTrue())
				s = node.New(analyzed)
			})
			It("Should set deadline to lastFired + period", func(ctx SpecContext) {
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

				var deadline telem.TimeSpan
				n.Next(node.Context{
					Context:         ctx,
					Elapsed:         0,
					Reason:          node.ReasonTimerTick,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(d telem.TimeSpan) { deadline = d },
				})
				Expect(deadline).To(Equal(telem.Second))
			})
			It("Should set deadline on channel input", func(ctx SpecContext) {
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
					Context:         ctx,
					Elapsed:         0,
					Reason:          node.ReasonTimerTick,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})

				var deadline telem.TimeSpan
				n.Next(node.Context{
					Context:         ctx,
					Elapsed:         500 * telem.Millisecond,
					Reason:          node.ReasonChannelInput,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(d telem.TimeSpan) { deadline = d },
				})
				Expect(deadline).To(Equal(telem.Second))
			})
		})
		Describe("Wait", func() {
			var factory *time.Module
			var s *node.ProgramState
			BeforeEach(func(ctx SpecContext) {
				factory = MustSucceed(time.NewModule(ctx, nil))
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
				analyzed, diagnostics := graph.Analyze(ctx, g, time.SymbolResolver)
				Expect(diagnostics.Ok()).To(BeTrue())
				s = node.New(analyzed)
			})
			It("Should set deadline to startTime + duration", func(ctx SpecContext) {
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

				var deadline telem.TimeSpan
				n.Next(node.Context{
					Context:         ctx,
					Elapsed:         5 * telem.Second,
					Reason:          node.ReasonTimerTick,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(d telem.TimeSpan) { deadline = d },
				})
				Expect(deadline).To(Equal(6 * telem.Second))
			})
			It("Should not set deadline after firing", func(ctx SpecContext) {
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

				n.Next(node.Context{
					Context:         ctx,
					Elapsed:         0,
					Reason:          node.ReasonTimerTick,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				n.Next(node.Context{
					Context:         ctx,
					Elapsed:         telem.Second,
					Reason:          node.ReasonTimerTick,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})

				deadlineCalled := false
				n.Next(node.Context{
					Context:         ctx,
					Elapsed:         5 * telem.Second,
					Reason:          node.ReasonTimerTick,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(d telem.TimeSpan) { deadlineCalled = true },
				})
				Expect(deadlineCalled).To(BeFalse())
			})
			It("Should set correct deadline after reset", func(ctx SpecContext) {
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

				n.Next(node.Context{
					Context:         ctx,
					Elapsed:         0,
					Reason:          node.ReasonTimerTick,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				n.Next(node.Context{
					Context:         ctx,
					Elapsed:         telem.Second,
					Reason:          node.ReasonTimerTick,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})
				n.Reset()

				var deadline telem.TimeSpan
				n.Next(node.Context{
					Context:         ctx,
					Elapsed:         10 * telem.Second,
					Reason:          node.ReasonTimerTick,
					MarkChanged:     func(int) {},
					MarkSelfChanged: func() {},
					SetDeadline:     func(d telem.TimeSpan) { deadline = d },
				})
				Expect(deadline).To(Equal(11 * telem.Second))
			})
		})
	})
	Describe("Now", func() {
		var factory *time.Module
		var s *node.ProgramState
		var changedOutputs []string
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewModule(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			changedOutputs = []string{}
			g := graph.Graph{
				Nodes: []graph.Node{{
					Key:  "now_1",
					Type: "now",
				}},
				Functions: []graph.Function{{
					Key:     "now",
					Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.TimeStamp()}},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, time.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
		})
		It("Should create node for now type", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
		It("Should output current wall-clock timestamp when triggered", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nowNode := s.Node("now_1")
			*nowNode.Output(0) = telem.NewSeriesV[telem.TimeStamp]()
			*nowNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			before := telem.Now()
			n.Next(node.Context{
				Context: ctx,
				Elapsed: 5 * telem.Second,
				Reason:  node.ReasonTimerTick,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			after := telem.Now()

			Expect(changedOutputs).To(HaveLen(1))
			Expect(changedOutputs[0]).To(Equal(ir.DefaultOutputParam))
			output := nowNode.Output(0)
			Expect(output.Len()).To(Equal(int64(1)))
			ts := telem.ValueAt[telem.TimeStamp](*output, 0)
			Expect(ts).To(BeNumerically(">=", before))
			Expect(ts).To(BeNumerically("<=", after))
		})
		It("Should fire on channel input reason", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nowNode := s.Node("now_1")
			*nowNode.Output(0) = telem.NewSeriesV[telem.TimeStamp]()
			*nowNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			n.Next(node.Context{
				Context: ctx,
				Elapsed: 0,
				Reason:  node.ReasonChannelInput,
				MarkChanged: func(output string) {
					changedOutputs = append(changedOutputs, output)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})

			Expect(changedOutputs).To(HaveLen(1))
			output := nowNode.Output(0)
			Expect(output.Len()).To(Equal(int64(1)))
		})
		It("Should create node for qualified time.now via CompoundFactory", func(ctx SpecContext) {
			compound := node.CompoundFactory{factory}
			cfg := node.Config{
				Node:  ir.Node{Type: "time.now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(compound.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
		It("Should not update base interval", func(ctx SpecContext) {
			Expect(factory.BaseInterval).To(Equal(telem.TimeSpanMax))
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			MustSucceed(factory.Create(ctx, cfg))
			Expect(factory.BaseInterval).To(Equal(telem.TimeSpanMax))
		})
		It("Should set matching output and output time", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nowNode := s.Node("now_1")
			*nowNode.Output(0) = telem.NewSeriesV[telem.TimeStamp]()
			*nowNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         0,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(output string) {},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})

			output := nowNode.Output(0)
			outputTime := nowNode.OutputTime(0)
			Expect(output.Len()).To(Equal(int64(1)))
			Expect(outputTime.Len()).To(Equal(int64(1)))
			ts := telem.ValueAt[telem.TimeStamp](*output, 0)
			tsTime := telem.ValueAt[telem.TimeStamp](*outputTime, 0)
			Expect(ts).To(Equal(tsTime))
		})
		It("Should work after reset", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nowNode := s.Node("now_1")
			*nowNode.Output(0) = telem.NewSeriesV[telem.TimeStamp]()
			*nowNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         0,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(output string) { changedOutputs = append(changedOutputs, output) },
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))

			n.Reset()
			changedOutputs = []string{}

			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         telem.Second,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(output string) { changedOutputs = append(changedOutputs, output) },
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))
			output := nowNode.Output(0)
			Expect(output.Len()).To(Equal(int64(1)))
		})
		It("Should add positive offset to wall-clock timestamp", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "now",
					Config: types.Params{
						{Name: "offset", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nowNode := s.Node("now_1")
			*nowNode.Output(0) = telem.NewSeriesV[telem.TimeStamp]()
			*nowNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			before := telem.Now() + telem.TimeStamp(telem.Second)
			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         0,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(output string) { changedOutputs = append(changedOutputs, output) },
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			after := telem.Now() + telem.TimeStamp(telem.Second)

			Expect(changedOutputs).To(HaveLen(1))
			output := nowNode.Output(0)
			ts := telem.ValueAt[telem.TimeStamp](*output, 0)
			Expect(ts).To(BeNumerically(">=", before))
			Expect(ts).To(BeNumerically("<=", after))
		})
		It("Should subtract negative offset from wall-clock timestamp", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "now",
					Config: types.Params{
						{Name: "offset", Type: types.TimeSpan(), Value: -telem.Second},
					},
				},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nowNode := s.Node("now_1")
			*nowNode.Output(0) = telem.NewSeriesV[telem.TimeStamp]()
			*nowNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			before := telem.Now() - telem.TimeStamp(telem.Second)
			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         0,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(output string) { changedOutputs = append(changedOutputs, output) },
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			after := telem.Now() - telem.TimeStamp(telem.Second)

			Expect(changedOutputs).To(HaveLen(1))
			output := nowNode.Output(0)
			ts := telem.ValueAt[telem.TimeStamp](*output, 0)
			Expect(ts).To(BeNumerically(">=", before))
			Expect(ts).To(BeNumerically("<=", after))
		})
		It("Should default to zero offset when config is absent", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nowNode := s.Node("now_1")
			*nowNode.Output(0) = telem.NewSeriesV[telem.TimeStamp]()
			*nowNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			before := telem.Now()
			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         0,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(output string) {},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			after := telem.Now()

			output := nowNode.Output(0)
			ts := telem.ValueAt[telem.TimeStamp](*output, 0)
			Expect(ts).To(BeNumerically(">=", before))
			Expect(ts).To(BeNumerically("<=", after))
		})
	})
})
