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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/tetratelabs/wazero"
)

var _ = Describe("Time", func() {
	Describe("NewModule", func() {
		It("Should create a host", func(ctx SpecContext) {
			factory := MustSucceed(
				time.NewHost(
					ctx,
					wazero.NewRuntimeWithConfig(
						ctx,
						wazero.NewRuntimeConfigInterpreter(),
					),
				),
			)
			Expect(factory).ToNot(BeNil())
		})
	})
	Describe("Interval", func() {
		var factory *time.Host
		var s *node.ProgramState
		var changedOutputs []int
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(
				time.NewHost(
					ctx,
					wazero.NewRuntimeWithConfig(
						ctx,
						wazero.NewRuntimeConfigInterpreter(),
					),
				),
			)
			changedOutputs = nil
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "interval_1"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"interval_1": {"type": "interval", "period": int64(telem.Second)},
				},
				Functions: []ir.Function{{
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
		It("Should create node for interval type", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Inputs: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("interval_1"),
			}
			n := MustSucceed(factory.Create(cfg))
			Expect(n).ToNot(BeNil())
		})
		It(
			"Should create node for qualified time.interval via CompoundFactory",
			func() {
				compound := node.CompoundFactory{factory}
				cfg := node.Config{
					Node: ir.Node{
						Type: "time.interval",
						Inputs: types.Params{
							{
								Name:  "period",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(compound.Create(cfg))
				Expect(n).ToNot(BeNil())
			},
		)
		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("interval_1"),
			}
			_, err := factory.Create(cfg)
			Expect(err).To(Equal(query.ErrNotFound))
		})
		It(
			"Should error at construction when the period input value is invalid",
			func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
							{
								Name:  "period",
								Type:  types.String(),
								Value: "not-a-timespan",
							},
						},
					},
					State: s.Node("interval_1"),
				}
				Expect(factory.Create(cfg)).Error().To(BeAValidationPathError())
			},
		)
		It(
			"Should error at construction when the period is zero",
			func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
							{
								Name:  "period",
								Type:  types.TimeSpan(),
								Value: telem.TimeSpan(0),
							},
						},
					},
					State: s.Node("interval_1"),
				}
				Expect(factory.Create(cfg)).Error().To(SatisfyAll(
					BeAValidationPathError(),
					MatchError(ContainSubstring("period: must be positive, got 0s")),
				))
			},
		)
		It(
			"Should error at construction when the period is negative",
			func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
							{
								Name:  "period",
								Type:  types.TimeSpan(),
								Value: -telem.Second,
							},
						},
					},
					State: s.Node("interval_1"),
				}
				Expect(factory.Create(cfg)).Error().To(SatisfyAll(
					BeAValidationPathError(),
					MatchError(ContainSubstring("period: must be positive, got - 1s")),
				))
			},
		)
		It(
			"Should allow a zero var-bound period at construction",
			func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
							{
								Name:  "period",
								Type:  types.VarRef(types.TimeSpan(), "p"),
								Value: telem.TimeSpan(0),
							},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(factory.Create(cfg))
				Expect(n).ToNot(BeNil())
			},
		)
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
			n := MustSucceed(factory.Create(cfg))
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
			n := MustSucceed(factory.Create(cfg))
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
			n := MustSucceed(factory.Create(cfg))
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
		It("Should stamp the output time from the cycle", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "interval",
					Inputs: types.Params{
						{Name: "period", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("interval_1"),
			}
			n := MustSucceed(factory.Create(cfg))
			intervalNode := s.Node("interval_1")
			*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
			*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			now := telem.SecondTS * 90
			n.Next(node.Context{
				Context:         ctx,
				Now:             now,
				Elapsed:         5 * telem.Second,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(int) {},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})

			outputTime := intervalNode.OutputTime(0)
			Expect(outputTime.Len()).To(Equal(int64(1)))
			Expect(outputTime.ValueAt[telem.TimeStamp](0)).To(Equal(now))
		})
		It(
			"Should not fire on channel input even when period elapsed",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
							{
								Name:  "period",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
			},
		)
		It(
			"Should fire immediately after Reset even if period has not elapsed",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
							{
								Name:  "period",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
				n.Reset(node.Context{})

				// Third tick at 1.5s - should fire because Reset clears started
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
			},
		)
	})
	Describe("Wait", func() {
		var factory *time.Host
		var s *node.ProgramState
		var changedOutputs []int
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(
				time.NewHost(
					ctx,
					wazero.NewRuntimeWithConfig(
						ctx,
						wazero.NewRuntimeConfigInterpreter(),
					),
				),
			)
			changedOutputs = nil
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "wait_1"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"wait_1": {"type": "wait", "duration": int64(telem.Second)},
				},
				Functions: []ir.Function{{
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
		It("Should create node for wait type", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Inputs: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(factory.Create(cfg))
			Expect(n).ToNot(BeNil())
		})
		It(
			"Should create node for qualified time.wait via CompoundFactory",
			func() {
				compound := node.CompoundFactory{factory}
				cfg := node.Config{
					Node: ir.Node{
						Type: "time.wait",
						Inputs: types.Params{
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("wait_1"),
				}
				n := MustSucceed(compound.Create(cfg))
				Expect(n).ToNot(BeNil())
			},
		)
		It(
			"Should error at construction when the duration input value is invalid",
			func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Inputs: types.Params{
							{
								Name:  "duration",
								Type:  types.String(),
								Value: "not-a-timespan",
							},
						},
					},
					State: s.Node("wait_1"),
				}
				Expect(factory.Create(cfg)).Error().To(BeAValidationPathError())
			},
		)
		It(
			"Should error at construction when the duration is zero",
			func() {
				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Inputs: types.Params{
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.TimeSpan(0),
							},
						},
					},
					State: s.Node("wait_1"),
				}
				Expect(factory.Create(cfg)).Error().To(SatisfyAll(
					BeAValidationPathError(),
					MatchError(ContainSubstring("duration: must be positive, got 0s")),
				))
			},
		)
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
			n := MustSucceed(factory.Create(cfg))
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
			n := MustSucceed(factory.Create(cfg))
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
		It("Should stamp the output time from the cycle", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "wait",
					Inputs: types.Params{
						{Name: "duration", Type: types.TimeSpan(), Value: telem.Second},
					},
				},
				State: s.Node("wait_1"),
			}
			n := MustSucceed(factory.Create(cfg))
			waitNode := s.Node("wait_1")
			*waitNode.Output(0) = telem.NewSeriesV[uint8]()
			*waitNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

			n.Next(node.Context{
				Context:         ctx,
				Now:             telem.SecondTS * 10,
				Elapsed:         2 * telem.Second,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(int) {},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})
			now := telem.SecondTS * 11
			n.Next(node.Context{
				Context:         ctx,
				Now:             now,
				Elapsed:         3 * telem.Second,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(int) {},
				MarkSelfChanged: func() {},
				SetDeadline:     func(_ telem.TimeSpan) {},
			})

			outputTime := waitNode.OutputTime(0)
			Expect(outputTime.Len()).To(Equal(int64(1)))
			Expect(outputTime.ValueAt[telem.TimeStamp](0)).To(Equal(now))
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
			n := MustSucceed(factory.Create(cfg))
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
			n := MustSucceed(factory.Create(cfg))
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
			n.Reset(node.Context{})

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
		It(
			"Should start timing from channel input that activates the stage",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Inputs: types.Params{
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("wait_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
			},
		)
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
			n := MustSucceed(factory.Create(cfg))
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
			n.Reset(node.Context{})
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
		It(
			"Should call MarkSelfChanged when active but not yet fired",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Inputs: types.Params{
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("wait_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
			},
		)
		It(
			"Should call MarkSelfChanged on channel input to survive non-tick cycles",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Inputs: types.Params{
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("wait_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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

				// Timer tick at 1s: should fire normally (wasn't starved by channel
				// input)
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
			},
		)
		It(
			"Should not fire on channel input even when duration elapsed",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Inputs: types.Params{
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("wait_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
			},
		)
	})
	Describe("Live span guard", func() {
		var factory *time.Host
		newState := func(
			ctx context.Context,
			nodeKey, typ, param string,
			span int64,
		) *node.ProgramState {
			GinkgoHelper()
			g := graph.Graph{
				Nodes: []graph.Node{{Key: nodeKey}},
				Inputs: map[string]msgpack.EncodedJSON{
					nodeKey: {"type": typ, param: span},
				},
				Functions: []ir.Function{{
					Key: typ,
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.U8()},
					},
					Inputs: types.Params{
						{Name: param, Type: types.I64()},
					},
				}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			return node.New(analyzed)
		}
		newNode := func(
			ctx context.Context,
			s *node.ProgramState,
			nodeKey, typ, param string,
		) node.Node {
			GinkgoHelper()
			cfg := node.Config{
				Node: ir.Node{
					Type: typ,
					Inputs: types.Params{{
						Name:  param,
						Type:  types.VarRef(types.TimeSpan(), "p"),
						Value: telem.TimeSpan(0),
					}},
				},
				State: s.Node(nodeKey),
			}
			n := MustSucceed(factory.Create(cfg))
			ns := s.Node(nodeKey)
			*ns.Output(0) = telem.NewSeriesV[uint8]()
			*ns.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()
			return n
		}
		var (
			reported  []error
			deadlines []telem.TimeSpan
			changed   []int
		)
		tick := func(ctx context.Context, n node.Node, elapsed telem.TimeSpan) {
			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         elapsed,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(i int) { changed = append(changed, i) },
				MarkSelfChanged: func() {},
				SetDeadline: func(d telem.TimeSpan) {
					deadlines = append(deadlines, d)
				},
				ReportError: func(err error) { reported = append(reported, err) },
			})
		}
		BeforeEach(func(ctx SpecContext) {
			factory = MustSucceed(time.NewHost(ctx, nil))
			reported, deadlines, changed = nil, nil, nil
		})
		It("Should park an interval and report the error once", func(ctx SpecContext) {
			s := newState(ctx, "interval_1", "interval", "period", 0)
			n := newNode(ctx, s, "interval_1", "interval", "period")
			tick(ctx, n, 0)
			tick(ctx, n, telem.Second)
			Expect(changed).To(BeEmpty())
			Expect(deadlines).To(BeEmpty())
			Expect(reported).To(HaveLen(1))
			Expect(reported[0]).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"interval period must be positive, got 0s",
				)),
			))
		})
		It(
			"Should report an interval error again after a reset",
			func(ctx SpecContext) {
				s := newState(ctx, "interval_1", "interval", "period", 0)
				n := newNode(ctx, s, "interval_1", "interval", "period")
				tick(ctx, n, 0)
				n.Reset(node.Context{})
				tick(ctx, n, telem.Second)
				Expect(reported).To(HaveLen(2))
			},
		)
		It("Should park a wait and report the error once", func(ctx SpecContext) {
			s := newState(ctx, "wait_1", "wait", "duration", 0)
			n := newNode(ctx, s, "wait_1", "wait", "duration")
			tick(ctx, n, 0)
			tick(ctx, n, telem.Second)
			Expect(changed).To(BeEmpty())
			Expect(deadlines).To(BeEmpty())
			Expect(reported).To(HaveLen(1))
			Expect(reported[0]).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"wait duration must be positive, got 0s",
				)),
			))
		})
		It("Should park a wait with a negative duration", func(ctx SpecContext) {
			s := newState(ctx, "wait_1", "wait", "duration", int64(-telem.Second))
			n := newNode(ctx, s, "wait_1", "wait", "duration")
			tick(ctx, n, 0)
			Expect(changed).To(BeEmpty())
			Expect(deadlines).To(BeEmpty())
			Expect(reported).To(HaveLen(1))
		})
		It(
			"Should fire an interval on its first tick when its live period differs",
			func(ctx SpecContext) {
				s := newState(
					ctx, "interval_1", "interval", "period", int64(telem.Millisecond),
				)
				n := newNode(ctx, s, "interval_1", "interval", "period")
				tick(ctx, n, 0)
				Expect(changed).To(HaveLen(1))
				Expect(deadlines).To(HaveExactElements(telem.Millisecond))
				Expect(reported).To(BeEmpty())
			},
		)
	})
	Describe("Symbols", func() {
		var root *symbol.Symbol
		BeforeEach(func() { root = symbol.NewRoot(nil, time.NewSymbols()) })
		bare := func(ctx context.Context, name string) *symbol.Symbol {
			GinkgoHelper()
			return MustSucceed(root.Resolve(ctx, name, symbol.IncludeInternal))
		}
		timeM := func(ctx context.Context, member string) *symbol.Symbol {
			GinkgoHelper()
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
	Describe("Fire timing", func() {
		var (
			host  *time.Host
			fired int
		)
		BeforeEach(func(ctx SpecContext) {
			host = MustSucceed(time.NewHost(ctx, nil))
			fired = 0
		})
		create := func(key, typ, param string, span telem.TimeSpan) node.Node {
			GinkgoHelper()
			n := ir.Node{
				Key:  key,
				Type: typ,
				Inputs: types.Params{
					{Name: param, Type: types.TimeSpan(), Value: span},
				},
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.U8()},
				},
			}
			state := node.New(ir.IR{Nodes: ir.Nodes{n}})
			created := MustSucceed(host.Create(node.Config{
				Node:  n,
				State: state.Node(key),
			}))
			*state.Node(key).Output(0) = telem.NewSeriesV[uint8]()
			*state.Node(key).OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()
			return created
		}
		tick := func(ctx context.Context, n node.Node, elapsed telem.TimeSpan) {
			n.Next(node.Context{
				Context:         ctx,
				Elapsed:         elapsed,
				Reason:          node.ReasonTimerTick,
				MarkChanged:     func(int) { fired++ },
				MarkSelfChanged: func() {},
				SetDeadline:     func(telem.TimeSpan) {},
			})
		}
		It(
			"Should fire an interval on its deadline and not one nanosecond before it",
			func(ctx SpecContext) {
				n := create("interval_1", "interval", "period", 100*telem.Millisecond)
				tick(ctx, n, 0)
				Expect(fired).To(Equal(1))
				tick(ctx, n, 100*telem.Millisecond-telem.Nanosecond)
				Expect(fired).To(Equal(1))
				tick(ctx, n, 100*telem.Millisecond)
				Expect(fired).To(Equal(2))
			},
		)
		It(
			"Should fire a wait on its deadline and not one nanosecond before it",
			func(ctx SpecContext) {
				n := create("wait_1", "wait", "duration", 100*telem.Millisecond)
				tick(ctx, n, 0)
				tick(ctx, n, 100*telem.Millisecond-telem.Nanosecond)
				Expect(fired).To(BeZero())
				tick(ctx, n, 100*telem.Millisecond)
				Expect(fired).To(Equal(1))
			},
		)
		It(
			"Should fire an interval once per late tick without delaying the schedule",
			func(ctx SpecContext) {
				n := create("interval_1", "interval", "period", 100*telem.Millisecond)
				for _, elapsed := range []telem.TimeSpan{
					0,
					100300 * telem.Microsecond,
					200100 * telem.Microsecond,
					300400 * telem.Microsecond,
					400 * telem.Millisecond,
				} {
					tick(ctx, n, elapsed)
				}
				Expect(fired).To(Equal(5))
			},
		)
		It(
			"Should not fire a wait on the tick of an earlier timer",
			func(ctx SpecContext) {
				n := create("wait_1", "wait", "duration", 13*telem.Millisecond)
				tick(ctx, n, 0)
				tick(ctx, n, 10*telem.Millisecond)
				Expect(fired).To(BeZero())
				tick(ctx, n, 13*telem.Millisecond)
				Expect(fired).To(Equal(1))
			},
		)
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
						"interval_1": {
							"type":   "interval",
							"period": int64(telem.Second),
						},
					},
					Functions: []ir.Function{{
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
							{
								Name:  "period",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
			It("Should count the next fire from the schedule", func(ctx SpecContext) {
				period := telem.Second
				late := 500 * telem.Microsecond
				n := MustSucceed(factory.Create(node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
							{Name: "period", Type: types.TimeSpan(), Value: period},
						},
					},
					State: s.Node("interval_1"),
				}))
				intervalNode := s.Node("interval_1")
				*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
				*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()
				var deadline telem.TimeSpan
				fired := 0
				tick := func(elapsed telem.TimeSpan) {
					n.Next(node.Context{
						Context:         ctx,
						Elapsed:         elapsed,
						Reason:          node.ReasonTimerTick,
						MarkChanged:     func(int) { fired++ },
						MarkSelfChanged: func() {},
						SetDeadline:     func(d telem.TimeSpan) { deadline = d },
					})
				}
				tick(0)
				tick(period + late)
				Expect(fired).To(Equal(2))
				Expect(deadline).To(Equal(2 * period))
				tick(2*period + late)
				Expect(fired).To(Equal(3))
				Expect(deadline).To(Equal(3 * period))
			})
			It("Should skip the missed fires after a pause", func(ctx SpecContext) {
				period := telem.Second
				n := MustSucceed(factory.Create(node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
							{Name: "period", Type: types.TimeSpan(), Value: period},
						},
					},
					State: s.Node("interval_1"),
				}))
				intervalNode := s.Node("interval_1")
				*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
				*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()
				var deadline telem.TimeSpan
				fired := 0
				tick := func(elapsed telem.TimeSpan) {
					n.Next(node.Context{
						Context:         ctx,
						Elapsed:         elapsed,
						Reason:          node.ReasonTimerTick,
						MarkChanged:     func(int) { fired++ },
						MarkSelfChanged: func() {},
						SetDeadline:     func(d telem.TimeSpan) { deadline = d },
					})
				}
				tick(0)
				tick(3500 * telem.Millisecond)
				Expect(fired).To(Equal(2))
				Expect(deadline).To(Equal(4 * period))
				tick(4*period - telem.Nanosecond)
				Expect(fired).To(Equal(2))
				tick(4 * period)
				Expect(fired).To(Equal(3))
			})
			It(
				"Should skip the missed fire exactly one period behind",
				func(ctx SpecContext) {
					period := telem.Second
					n := MustSucceed(factory.Create(node.Config{
						Node: ir.Node{
							Type: "interval",
							Inputs: types.Params{
								{Name: "period", Type: types.TimeSpan(), Value: period},
							},
						},
						State: s.Node("interval_1"),
					}))
					intervalNode := s.Node("interval_1")
					*intervalNode.Output(0) = telem.NewSeriesV[uint8]()
					*intervalNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()
					var deadline telem.TimeSpan
					fired := 0
					tick := func(elapsed telem.TimeSpan) {
						n.Next(node.Context{
							Context:         ctx,
							Elapsed:         elapsed,
							Reason:          node.ReasonTimerTick,
							MarkChanged:     func(int) { fired++ },
							MarkSelfChanged: func() {},
							SetDeadline:     func(d telem.TimeSpan) { deadline = d },
						})
					}
					tick(0)
					tick(2 * period)
					Expect(fired).To(Equal(2))
					Expect(deadline).To(Equal(3 * period))
					tick(2 * period)
					Expect(fired).To(Equal(2))
				},
			)
			It("Should set deadline on channel input", func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "interval",
						Inputs: types.Params{
							{
								Name:  "period",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("interval_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
					Functions: []ir.Function{{
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
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("wait_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("wait_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
			It(
				"Should not set a deadline on the pass that fires",
				func(ctx SpecContext) {
					cfg := node.Config{
						Node: ir.Node{
							Type: "wait",
							Inputs: types.Params{
								{
									Name:  "duration",
									Type:  types.TimeSpan(),
									Value: telem.Second,
								},
							},
						},
						State: s.Node("wait_1"),
					}
					n := MustSucceed(factory.Create(cfg))
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
					deadlineCalled := false
					n.Next(node.Context{
						Context:         ctx,
						Elapsed:         telem.Second,
						Reason:          node.ReasonTimerTick,
						MarkChanged:     func(int) {},
						MarkSelfChanged: func() {},
						SetDeadline:     func(_ telem.TimeSpan) { deadlineCalled = true },
					})
					Expect(waitNode.Output(0).Len()).To(Equal(int64(1)))
					Expect(deadlineCalled).To(BeFalse())
				},
			)
			It("Should set correct deadline after reset", func(ctx SpecContext) {
				cfg := node.Config{
					Node: ir.Node{
						Type: "wait",
						Inputs: types.Params{
							{
								Name:  "duration",
								Type:  types.TimeSpan(),
								Value: telem.Second,
							},
						},
					},
					State: s.Node("wait_1"),
				}
				n := MustSucceed(factory.Create(cfg))
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
				n.Reset(node.Context{})

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
			factory = MustSucceed(
				time.NewHost(
					ctx,
					wazero.NewRuntimeWithConfig(
						ctx,
						wazero.NewRuntimeConfigInterpreter(),
					),
				),
			)
			changedOutputs = nil
			g := graph.Graph{
				Nodes: []graph.Node{{Key: "now_1"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"now_1": {"type": "now"},
				},
				Functions: []ir.Function{
					{
						Key: "now",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.TimeStamp()},
						},
					},
				},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
		})
		It("Should create node for now type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(cfg))
			Expect(n).ToNot(BeNil())
		})
		It(
			"Should output the cycle timestamp when triggered",
			func(ctx SpecContext) {
				cfg := node.Config{
					Node:  ir.Node{Type: "now"},
					State: s.Node("now_1"),
				}
				n := MustSucceed(factory.Create(cfg))
				nowNode := s.Node("now_1")
				*nowNode.Output(0) = telem.NewSeriesV[telem.TimeStamp]()
				*nowNode.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp]()

				now := telem.SecondTS * 77
				n.Next(node.Context{
					Context: ctx,
					Now:     now,
					Elapsed: 5 * telem.Second,
					Reason:  node.ReasonTimerTick,
					MarkChanged: func(i int) {
						changedOutputs = append(changedOutputs, i)
					},
					MarkSelfChanged: func() {},
					SetDeadline:     func(_ telem.TimeSpan) {},
				})

				Expect(changedOutputs).To(HaveLen(1))
				Expect(changedOutputs[0]).To(Equal(0))
				output := nowNode.Output(0)
				Expect(output.Len()).To(Equal(int64(1)))
				Expect(output.ValueAt[telem.TimeStamp](0)).To(Equal(now))
				outputTime := nowNode.OutputTime(0)
				Expect(outputTime.ValueAt[telem.TimeStamp](0)).To(Equal(now))
			},
		)
		It("Should fire on channel input reason", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(cfg))
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
		It(
			"Should create node for qualified time.now via CompoundFactory",
			func() {
				compound := node.CompoundFactory{factory}
				cfg := node.Config{
					Node:  ir.Node{Type: "time.now"},
					State: s.Node("now_1"),
				}
				n := MustSucceed(compound.Create(cfg))
				Expect(n).ToNot(BeNil())
			},
		)
		It("Should set matching output and output time", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(cfg))
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
			ts := output.ValueAt[telem.TimeStamp](0)
			tsTime := outputTime.ValueAt[telem.TimeStamp](0)
			Expect(ts).To(Equal(tsTime))
		})
		It("Should work after reset", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "now"},
				State: s.Node("now_1"),
			}
			n := MustSucceed(factory.Create(cfg))
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

			n.Reset(node.Context{})
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
				ReportError:     func(error) {},
			})
			return r
		}

		Describe("Interval", func() {
			It(
				"Should honor the declared initial before any write",
				func(ctx SpecContext) {
					cfg, _ := varConfig("interval", "period", telem.Second)
					n := MustSucceed(factory.Create(cfg))
					Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeTrue())
					Expect(
						tick(ctx, n, 500*telem.Millisecond, node.ReasonTimerTick).fired,
					).
						To(BeFalse())
					Expect(
						tick(ctx, n, telem.Second, node.ReasonTimerTick).fired,
					).To(BeTrue())
				},
			)
			It(
				"Should fire at once when a zero period turns positive",
				func(ctx SpecContext) {
					cfg, set := varConfig("interval", "period", 0)
					n := MustSucceed(factory.Create(cfg))
					Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeFalse())
					set(telem.Second)
					Expect(
						tick(ctx, n, 100*telem.Millisecond, node.ReasonTimerTick).fired,
					).To(BeTrue())
				},
			)
			It(
				"Should adopt a shortened period at the next evaluation",
				func(ctx SpecContext) {
					cfg, set := varConfig("interval", "period", telem.Second)
					n := MustSucceed(factory.Create(cfg))
					Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeTrue())
					set(100 * telem.Millisecond)
					Expect(
						tick(ctx, n, 100*telem.Millisecond, node.ReasonTimerTick).fired,
					).
						To(BeTrue())
				},
			)
			It(
				"Should adopt a lengthened period without firing early",
				func(ctx SpecContext) {
					cfg, set := varConfig("interval", "period", 100*telem.Millisecond)
					n := MustSucceed(factory.Create(cfg))
					Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeTrue())
					set(telem.Second)
					Expect(
						tick(ctx, n, 100*telem.Millisecond, node.ReasonTimerTick).fired,
					).
						To(BeFalse())
					Expect(
						tick(ctx, n, telem.Second, node.ReasonTimerTick).fired,
					).To(BeTrue())
				},
			)
			It(
				"Should report the deadline from the live period",
				func(ctx SpecContext) {
					cfg, set := varConfig("interval", "period", telem.Second)
					n := MustSucceed(factory.Create(cfg))
					Expect(
						tick(ctx, n, 0, node.ReasonTimerTick).deadline,
					).To(Equal(telem.Second))
					set(2 * telem.Second)
					r := tick(ctx, n, 500*telem.Millisecond, node.ReasonChannelInput)
					Expect(r.fired).To(BeFalse())
					Expect(r.deadline).To(Equal(2 * telem.Second))
				},
			)
			It(
				"Should fire immediately after Reset using the live period",
				func(ctx SpecContext) {
					cfg, set := varConfig("interval", "period", telem.Second)
					n := MustSucceed(factory.Create(cfg))
					Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeTrue())
					Expect(
						tick(ctx, n, telem.Second, node.ReasonTimerTick).fired,
					).To(BeTrue())
					set(5 * telem.Second)
					n.Reset(node.Context{})
					Expect(
						tick(
							ctx,
							n,
							1500*telem.Millisecond,
							node.ReasonTimerTick,
						).fired,
					).
						To(BeTrue())
				},
			)
		})

		Describe("Wait", func() {
			It(
				"Should honor the declared initial before any write",
				func(ctx SpecContext) {
					cfg, _ := varConfig("wait", "duration", telem.Second)
					n := MustSucceed(factory.Create(cfg))
					Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeFalse())
					Expect(
						tick(ctx, n, 500*telem.Millisecond, node.ReasonTimerTick).fired,
					).
						To(BeFalse())
					Expect(
						tick(ctx, n, telem.Second, node.ReasonTimerTick).fired,
					).To(BeTrue())
				},
			)
			It(
				"Should fire earlier when the duration is shortened mid-wait",
				func(ctx SpecContext) {
					cfg, set := varConfig("wait", "duration", 10*telem.Second)
					n := MustSucceed(factory.Create(cfg))
					Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeFalse())
					set(telem.Second)
					Expect(
						tick(ctx, n, telem.Second, node.ReasonTimerTick).fired,
					).To(BeTrue())
				},
			)
			It(
				"Should fire later when the duration is lengthened mid-wait",
				func(ctx SpecContext) {
					cfg, set := varConfig("wait", "duration", telem.Second)
					n := MustSucceed(factory.Create(cfg))
					Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeFalse())
					set(5 * telem.Second)
					Expect(
						tick(ctx, n, telem.Second, node.ReasonTimerTick).fired,
					).To(BeFalse())
					Expect(
						tick(ctx, n, 5*telem.Second, node.ReasonTimerTick).fired,
					).To(BeTrue())
				},
			)
			It(
				"Should report the deadline from the live duration",
				func(ctx SpecContext) {
					cfg, set := varConfig("wait", "duration", telem.Second)
					n := MustSucceed(factory.Create(cfg))
					Expect(
						tick(ctx, n, 0, node.ReasonTimerTick).deadline,
					).To(Equal(telem.Second))
					set(3 * telem.Second)
					r := tick(ctx, n, 500*telem.Millisecond, node.ReasonChannelInput)
					Expect(r.fired).To(BeFalse())
					Expect(r.deadline).To(Equal(3 * telem.Second))
				},
			)
			It("Should stay one-shot after a shortening write", func(ctx SpecContext) {
				cfg, set := varConfig("wait", "duration", telem.Second)
				n := MustSucceed(factory.Create(cfg))
				Expect(tick(ctx, n, 0, node.ReasonTimerTick).fired).To(BeFalse())
				Expect(
					tick(ctx, n, telem.Second, node.ReasonTimerTick).fired,
				).To(BeTrue())
				set(100 * telem.Millisecond)
				Expect(
					tick(ctx, n, 2*telem.Second, node.ReasonTimerTick).fired,
				).To(BeFalse())
			})
		})
	})
})
