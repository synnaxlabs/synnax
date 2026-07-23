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
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

var _ = Describe("Time", func() {
	Describe("NewModule", func() {
		It("Should create module with max timing base", func(ctx SpecContext) {
			factory := MustSucceed(time.NewHost(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			Expect(factory).ToNot(BeNil())
		})
	})
	Describe("Interval", func() {
		var factory *time.Host
		var s *node.ProgramState
		var changedOutputs []int
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewHost(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			changedOutputs = nil
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "interval_1"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"interval_1": {"type": "interval", "period": int64(telem.Second)},
				},
				Functions: []graph.Function{{
					Key: "interval",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
					Inputs: types.Params{
						{Name: "period", Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
		})
		It("Should create node for interval type", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Inputs: types.Params{
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
					Inputs: types.Params{
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
		It("Should error at construction when the period input value is invalid", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Inputs: types.Params{
						{Name: "period", Type: types.String(), Value: "not-a-timespan"},
					},
				},
				State: s.Node("interval_1"),
			}
			Expect(factory.Create(ctx, cfg)).Error().To(BeAValidationPathError())
		})
		It("Should fire immediately on first tick", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
		var factory *time.Host
		var s *node.ProgramState
		var changedOutputs []int
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewHost(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			changedOutputs = nil
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "wait_1"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"wait_1": {"type": "wait", "duration": int64(telem.Second)},
				},
				Functions: []graph.Function{{
					Key: "wait",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
					Inputs: types.Params{
						{Name: "duration", Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
		})
		It("Should create node for wait type", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Inputs: types.Params{
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
					Inputs: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(compound.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
		It("Should error at construction when the duration input value is invalid", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Inputs: types.Params{
						{Name: "duration", Type: types.String(), Value: "not-a-timespan"},
					},
				},
				State: s.Node("wait_1"),
			}
			Expect(factory.Create(ctx, cfg)).Error().To(BeAValidationPathError())
		})
		It("Should not fire before duration elapses", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
					Inputs: types.Params{
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
			factory := MustSucceed(time.NewHost(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			g := graph.Graph{
				Nodes: []graph.Node{
					{Key: "interval_1"},
					{Key: "interval_2"},
				},
				Inputs: map[string]msgpack.EncodedJSON{
					"interval_1": {"type": "interval", "period": int64(100 * telem.Millisecond)},
					"interval_2": {"type": "interval", "period": int64(150 * telem.Millisecond)},
				},
				Functions: []graph.Function{{
					Key: "interval",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
					Inputs: types.Params{
						{Name: "period", Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			s := node.New(analyzed)

			// Create first interval with 100ms period
			cfg1 := node.Config{
				Node: ir.Node{
					Type: "interval",
					Inputs: types.Params{
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
					Inputs: types.Params{
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
	Describe("Symbols", func() {
		var root *symbol.Symbol
		BeforeEach(func() { root = symbol.NewRoot(nil, time.NewSymbols()) })
		bare := func(ctx context.Context, name string) *symbol.Symbol {
			return MustSucceed(root.Resolve(ctx, name, symbol.IncludeInternal))
		}
		timeM := func(ctx context.Context, member string) *symbol.Symbol {
			mod := MustSucceed(root.Resolve(ctx, "time", symbol.IncludeInternal))
			return MustSucceed(mod.Resolve(ctx, member, symbol.IncludeInternal))
		}
		It("Should expose interval bare symbol", func(ctx SpecContext) {
			Expect(bare(ctx, "interval").Name).To(Equal("interval"))
		})
		It("Should expose wait bare symbol", func(ctx SpecContext) {
			Expect(bare(ctx, "wait").Name).To(Equal("wait"))
		})
		It("Should expose time.now (not deprecated)", func(ctx SpecContext) {
			sym := timeM(ctx, "now")
			Expect(sym.Name).To(Equal("now"))
			Expect(sym.Deprecated).To(BeNil())
		})
		It("Should expose bare now as deprecated", func(ctx SpecContext) {
			sym := bare(ctx, "now")
			Expect(sym.Name).To(Equal("now"))
			Expect(sym.Deprecated).ToNot(BeNil())
			Expect(sym.Deprecated.QualifiedName()).To(Equal("time.now"))
		})
		It("Should mark bare interval as deprecated", func(ctx SpecContext) {
			sym := bare(ctx, "interval")
			Expect(sym.Deprecated).ToNot(BeNil())
			Expect(sym.Deprecated.QualifiedName()).To(Equal("time.interval"))
		})
		It("Should mark bare wait as deprecated", func(ctx SpecContext) {
			sym := bare(ctx, "wait")
			Expect(sym.Deprecated).ToNot(BeNil())
			Expect(sym.Deprecated.QualifiedName()).To(Equal("time.wait"))
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
		var factory *time.Host
		var s *node.ProgramState
		var changedOutputs []int
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewHost(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			changedOutputs = nil
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "interval_1"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"interval_1": {"type": "interval", "period": int64(100 * telem.Millisecond)},
				},
				Functions: []graph.Function{{
					Key: "interval",
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
					Inputs: types.Params{
						{Name: "period", Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
		})
		Describe("Interval with tolerance", func() {
			It("Should fire on early tick within tolerance", func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
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
						Inputs: types.Params{
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
						Inputs: types.Params{
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
						Inputs: types.Params{
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
					Nodes: []graph.Node{{Key: "wait_1"}},
					Inputs: map[string]msgpack.EncodedJSON{
						"wait_1": {"type": "wait", "duration": int64(100 * telem.Millisecond)},
					},
					Functions: []graph.Function{{
						Key: "wait",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
						Inputs: types.Params{
							{Name: "duration", Type: types.I64()},
						},
					}},
				}
				analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue())
				waitState := node.New(analyzed)
				waitFactory := MustSucceed(time.NewHost(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))

				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Inputs: types.Params{
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
			var factory *time.Host
			var s *node.ProgramState
			BeforeEach(func(ctx SpecContext) {
				factory = MustSucceed(time.NewHost(ctx, nil))
				g := graph.Graph{
					Nodes: []graph.Node{{Key: "interval_1"}},
					Inputs: map[string]msgpack.EncodedJSON{
						"interval_1": {"type": "interval", "period": int64(telem.Second)},
					},
					Functions: []graph.Function{{
						Key: "interval",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
						Inputs: types.Params{
							{Name: "period", Type: types.I64()},
						},
					}},
				}
				analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue())
				s = node.New(analyzed)
			})
			It("Should set deadline to lastFired + period", func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
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
						Inputs: types.Params{
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
			var factory *time.Host
			var s *node.ProgramState
			BeforeEach(func(ctx SpecContext) {
				factory = MustSucceed(time.NewHost(ctx, nil))
				g := graph.Graph{
					Nodes: []graph.Node{{Key: "wait_1"}},
					Inputs: map[string]msgpack.EncodedJSON{
						"wait_1": {"type": "wait", "duration": int64(telem.Second)},
					},
					Functions: []graph.Function{{
						Key: "wait",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.U8()},
						},
						Inputs: types.Params{
							{Name: "duration", Type: types.I64()},
						},
					}},
				}
				analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue())
				s = node.New(analyzed)
			})
			It("Should set deadline to startTime + duration", func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Inputs: types.Params{
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
						Inputs: types.Params{
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
						Inputs: types.Params{
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
		var factory *time.Host
		var s *node.ProgramState
		var changedOutputs []int
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewHost(ctx, wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())))
			changedOutputs = nil
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "now_1"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"now_1": {"type": "now"},
				},
				Functions: []graph.Function{{
					Key:     "now",
					Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.TimeStamp()}},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
				},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			after := telem.Now()

			Expect(changedOutputs).To(HaveLen(1))
			Expect(changedOutputs[0]).To(Equal(0))
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
				MarkChanged: func(i int) {
					changedOutputs = append(changedOutputs, i)
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
				MarkChanged:     func(int) {},
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
				MarkChanged:     func(i int) { changedOutputs = append(changedOutputs, i) },
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))

			n.Reset()
			changedOutputs = nil

			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         telem.Second,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(i int) { changedOutputs = append(changedOutputs, i) },
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			Expect(changedOutputs).To(HaveLen(1))
			output := nowNode.Output(0)
			Expect(output.Len()).To(Equal(int64(1)))
		})
	})
	Describe("Variable inputs", func() {
		var factory *time.Host
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewHost(ctx, nil))
		})

		// varConfig builds a config whose span input is var-bound: Value holds
		// the declared initial and set writes the variable's live slot.
		varConfig := func(
			nodeType, param string, initial telem.TimeSpan,
		) (node.Config, func(telem.TimeSpan)) {
			v := ir.Node{
				Key:  "v",
				Type: "variable",
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.I64()},
				},
			}
			n := ir.Node{
				Key:  "n",
				Type: nodeType,
				Inputs: types.Params{
					{Name: param, Type: types.VarRef(types.I64(), "v"), Value: initial},
				},
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.U8()},
				},
			}
			state := node.New(ir.IR{Nodes: ir.Nodes{v, n}})
			set := func(span telem.TimeSpan) {
				*state.Node("v").Output(0) = telem.NewSeriesV(int64(span))
			}
			return node.Config{Node: n, State: state.Node("n")}, set
		}

		type tickResult struct {
			fired    bool
			deadline telem.TimeSpan
		}
		tick := func(
			ctx context.Context,
			n node.Node,
			elapsed telem.TimeSpan,
			reason node.RunReason,
		) tickResult {
			var r tickResult
			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         elapsed,
				Reason:          reason,
				MarkChanged:     func(int) { r.fired = true },
				MarkSelfChanged: func() {},
				SetDeadline:     func(d telem.TimeSpan) { r.deadline = d },
			})
			return r
		}

		Describe("Interval", func() {
			It("Should honor the declared initial before any write", func(ctx SpecContext) {
				cfg, _ := varConfig("interval", "period", telem.Second)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeTrue())
				Expect(tick(ctx, n, 500*telem.Millisecond, node.ReasonTimerTick).fired).
					To(BeFalse())
				Expect(tick(ctx, n, telem.Second, node.ReasonTimerTick).fired).To(BeTrue())
			})
			It("Should adopt a shortened period at the next evaluation", func(ctx SpecContext) {
				cfg, set := varConfig("interval", "period", telem.Second)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeTrue())
				set(100 * telem.Millisecond)
				Expect(tick(ctx, n, 100*telem.Millisecond, node.ReasonTimerTick).fired).
					To(BeTrue())
			})
			It("Should adopt a lengthened period without firing early", func(ctx SpecContext) {
				cfg, set := varConfig("interval", "period", 100*telem.Millisecond)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeTrue())
				set(telem.Second)
				Expect(tick(ctx, n, 100*telem.Millisecond, node.ReasonTimerTick).fired).
					To(BeFalse())
				Expect(tick(ctx, n, telem.Second, node.ReasonTimerTick).fired).To(BeTrue())
			})
			It("Should report the deadline from the live period", func(ctx SpecContext) {
				cfg, set := varConfig("interval", "period", telem.Second)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).deadline).To(Equal(telem.Second))
				set(2 * telem.Second)
				r := tick(ctx, n, 500*telem.Millisecond, node.ReasonChannelInput)
				Expect(r.fired).To(BeFalse())
				Expect(r.deadline).To(Equal(2 * telem.Second))
			})
			It("Should fire immediately after Reset using the live period", func(ctx SpecContext) {
				cfg, set := varConfig("interval", "period", telem.Second)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeTrue())
				Expect(tick(ctx, n, telem.Second, node.ReasonTimerTick).fired).To(BeTrue())
				set(5 * telem.Second)
				n.Reset()
				Expect(tick(ctx, n, 1500*telem.Millisecond, node.ReasonTimerTick).fired).
					To(BeTrue())
			})
			It("Should seed the timing base from the declared value only", func(ctx SpecContext) {
				cfg, set := varConfig("interval", "period", 100*telem.Millisecond)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(factory.BaseInterval).To(Equal(100 * telem.Millisecond))
				set(telem.Millisecond)
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeTrue())
				Expect(factory.BaseInterval).To(Equal(100 * telem.Millisecond))
			})
		})

		Describe("Wait", func() {
			It("Should honor the declared initial before any write", func(ctx SpecContext) {
				cfg, _ := varConfig("wait", "duration", telem.Second)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeFalse())
				Expect(tick(ctx, n, 500*telem.Millisecond, node.ReasonTimerTick).fired).
					To(BeFalse())
				Expect(tick(ctx, n, telem.Second, node.ReasonTimerTick).fired).To(BeTrue())
			})
			It("Should fire earlier when the duration is shortened mid-wait", func(ctx SpecContext) {
				cfg, set := varConfig("wait", "duration", 10*telem.Second)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeFalse())
				set(telem.Second)
				Expect(tick(ctx, n, telem.Second, node.ReasonTimerTick).fired).To(BeTrue())
			})
			It("Should fire later when the duration is lengthened mid-wait", func(ctx SpecContext) {
				cfg, set := varConfig("wait", "duration", telem.Second)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeFalse())
				set(5 * telem.Second)
				Expect(tick(ctx, n, telem.Second, node.ReasonTimerTick).fired).To(BeFalse())
				Expect(tick(ctx, n, 5*telem.Second, node.ReasonTimerTick).fired).To(BeTrue())
			})
			It("Should report the deadline from the live duration", func(ctx SpecContext) {
				cfg, set := varConfig("wait", "duration", telem.Second)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).deadline).To(Equal(telem.Second))
				set(3 * telem.Second)
				r := tick(ctx, n, 500*telem.Millisecond, node.ReasonChannelInput)
				Expect(r.fired).To(BeFalse())
				Expect(r.deadline).To(Equal(3 * telem.Second))
			})
			It("Should stay one-shot after a shortening write", func(ctx SpecContext) {
				cfg, set := varConfig("wait", "duration", telem.Second)
				n := MustSucceed(factory.Create(ctx, cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeFalse())
				Expect(tick(ctx, n, telem.Second, node.ReasonTimerTick).fired).To(BeTrue())
				set(100 * telem.Millisecond)
				Expect(tick(ctx, n, 2*telem.Second, node.ReasonTimerTick).fired).To(BeFalse())
			})
		})
	})
})

var _ = Describe("TimingBase GCD matrix", func() {
	// compileBase compiles source and creates every timer node through a fresh
	// time Host, returning the resulting BaseInterval.
	compileBase := func(ctx context.Context, source string) telem.TimeSpan {
		root := NewRoot(nil,
			symbol.Symbol{Name: "a", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 501},
			symbol.Symbol{Name: "b", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 502},
		)
		prog := MustSucceed(arc.CompileText(ctx, arc.Text{Raw: "import time\n" + source}, root))
		factory := MustSucceed(time.NewHost(ctx, nil))
		s := node.New(prog.IR)
		f := node.CompoundFactory{factory}
		for _, n := range prog.Nodes {
			if _, err := f.Create(ctx, node.Config{
				Node: n, Program: prog, State: s.Node(n.Key),
			}); err != nil && !errors.Is(err, query.ErrNotFound) {
				Fail("create " + n.Key + ": " + err.Error())
			}
		}
		return factory.BaseInterval
	}

	DescribeTable("computes the GCD over declared and literal-reassigned spans",
		func(ctx SpecContext, source string, expected telem.TimeSpan) {
			Expect(compileBase(ctx, source)).To(Equal(expected))
		},
		Entry("two literal intervals", `
time.interval{period=100ms} -> a
time.interval{period=60ms} -> b
`, 20*telem.Millisecond),
		Entry("two intervals fed by vars, never reassigned", `
sequence main {
    p := i64 ns(100ms)
    q := i64 ns(60ms)
    stage run {
        time.interval{period=p} -> a
        time.interval{period=q} -> b
    }
}
`, 20*telem.Millisecond),
		Entry("two intervals fed by vars, each reassigned with a literal", `
sequence main {
    p := i64 ns(100ms)
    q := i64 ns(60ms)
    stage run {
        time.interval{period=p} -> a
        time.interval{period=q} -> b
        1 => faster
    }
    stage faster {
        p = i64 ns(10ms)
        q = i64 ns(45ms)
    }
}
`, 5*telem.Millisecond),
		Entry("two intervals fed by vars, expression reassignments excluded", `
sequence main {
    p := i64 ns(100ms)
    q := i64 ns(60ms)
    stage run {
        time.interval{period=p} -> a
        time.interval{period=q} -> b
        1 => faster
    }
    stage faster {
        p = i64 ns(2 * 25ms)
        q = i64 ns(3 * 20ms)
    }
}
`, 20*telem.Millisecond),
		Entry("two literal waits", `
time.wait{duration=75ms} -> a
time.wait{duration=50ms} -> b
`, 25*telem.Millisecond),
		Entry("two waits fed by vars, each reassigned with a literal", `
sequence main {
    d := i64 ns(80ms)
    e := i64 ns(50ms)
    stage run {
        time.wait{duration=d} -> a
        time.wait{duration=e} -> b
        1 => faster
    }
    stage faster {
        d = i64 ns(30ms)
        e = i64 ns(35ms)
    }
}
`, 5*telem.Millisecond),
		Entry("interval + wait fed by vars, never reassigned", `
sequence main {
    p := i64 ns(100ms)
    d := i64 ns(75ms)
    stage run {
        time.interval{period=p} -> a
        time.wait{duration=d} -> b
    }
}
`, 25*telem.Millisecond),
		Entry("interval + wait fed by vars, each reassigned with a literal", `
sequence main {
    p := i64 ns(100ms)
    d := i64 ns(80ms)
    stage run {
        time.interval{period=p} -> a
        time.wait{duration=d} -> b
        1 => faster
    }
    stage faster {
        p = i64 ns(60ms)
        d = i64 ns(30ms)
    }
}
`, 10*telem.Millisecond),
		Entry("interval + wait fed by vars, expression reassignments excluded", `
sequence main {
    p := i64 ns(100ms)
    d := i64 ns(75ms)
    stage run {
        time.interval{period=p} -> a
        time.wait{duration=d} -> b
        1 => faster
    }
    stage faster {
        p = i64 ns(2 * 25ms)
        d = i64 ns(3 * 15ms)
    }
}
`, 25*telem.Millisecond),
		Entry("literal interval + reassigned var wait", `
sequence main {
    d := i64 ns(60ms)
    stage run {
        time.interval{period=100ms} -> a
        time.wait{duration=d} -> b
        1 => faster
    }
    stage faster {
        d = i64 ns(45ms)
    }
}
`, 5*telem.Millisecond),
		Entry("var interval, two reassignment sites", `
sequence main {
    p := i64 ns(100ms)
    stage run {
        time.interval{period=p} -> a
        1 => mid
    }
    stage mid {
        p = i64 ns(50ms)
        1 => fast
    }
    stage fast {
        p = i64 ns(30ms)
    }
}
`, 10*telem.Millisecond),
		Entry("same var feeding both timer kinds", `
sequence main {
    p := i64 ns(40ms)
    stage run {
        time.interval{period=p} -> a
        time.wait{duration=p} -> b
        1 => faster
    }
    stage faster {
        p = i64 ns(30ms)
    }
}
`, 10*telem.Millisecond),
		Entry("reassignment in an unreached stage still counts", `
sequence main {
    p := i64 ns(100ms)
    stage run {
        time.interval{period=p} -> a
    }
    stage never {
        p = i64 ns(30ms)
    }
}
`, 10*telem.Millisecond),
	)
})
