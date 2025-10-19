// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package interval_test

import (
	"context"
	gotime "time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/interval"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	timewheel "github.com/synnaxlabs/arc/runtime/time"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Interval", func() {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		wheel     *timewheel.Wheel
		factory   node.Factory
		progState *state.State
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		// Create time wheel with callback that does nothing
		wheel = timewheel.NewWheel(10*gotime.Millisecond, func(key string) {
			// Callback is called by time wheel, but we'll manually trigger node execution
		})
		wheel.Start(ctx)

		factory = interval.NewFactory(interval.Config{
			TimeWheel: wheel,
		})

		// Create minimal state for testing
		progState = state.New(state.Config{
			Nodes: ir.Nodes{
				{
					Key:  "interval_1",
					Type: "interval",
					Outputs: types.Params{
						Keys:   []string{"tick", "timestamp", "elapsed"},
						Values: []types.Type{types.U64(), types.TimeStamp(), types.TimeSpan()},
					},
				},
			},
		})
	})

	AfterEach(func() {
		wheel.Stop()
		cancel()
	})

	Describe("Factory", func() {
		It("Should create interval node with period", func() {
			n, err := factory.Create(ctx, node.Config{
				Node: ir.Node{
					Key:  "interval_1",
					Type: "interval",
					ConfigValues: map[string]any{
						"period": telem.TimeSpan(100 * gotime.Millisecond),
					},
				},
				State: progState.Node("interval_1"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})

		It("Should create interval node with period and initial delay", func() {
			n, err := factory.Create(ctx, node.Config{
				Node: ir.Node{
					Key:  "interval_1",
					Type: "interval",
					ConfigValues: map[string]any{
						"period":        telem.TimeSpan(100 * gotime.Millisecond),
						"initial_delay": telem.TimeSpan(50 * gotime.Millisecond),
					},
				},
				State: progState.Node("interval_1"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})

		It("Should return NotFound for non-interval nodes", func() {
			_, err := factory.Create(ctx, node.Config{
				Node: ir.Node{
					Key:  "other",
					Type: "other",
				},
			})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Node Execution", func() {
		It("Should output tick, timestamp, and elapsed", func() {
			n, err := factory.Create(ctx, node.Config{
				Node: ir.Node{
					Key:  "interval_1",
					Type: "interval",
					ConfigValues: map[string]any{
						"period": telem.TimeSpan(50 * gotime.Millisecond),
					},
				},
				State: progState.Node("interval_1"),
			})
			Expect(err).ToNot(HaveOccurred())

			// Initialize node
			n.Init(ctx, func(output string) {})

			// Wait for at least one tick
			gotime.Sleep(60 * gotime.Millisecond)

			// Execute node
			var changedOutputs []string
			n.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})

			// Should have marked all outputs as changed
			Expect(changedOutputs).To(ContainElements("tick", "timestamp", "elapsed"))

			// Check outputs are populated
			stateNode := progState.Node("interval_1")
			tickData := stateNode.Output(0)
			timestampData := stateNode.Output(1)
			elapsedData := stateNode.Output(2)

			Expect(tickData.Len()).To(Equal(int64(1)))
			Expect(timestampData.Len()).To(Equal(int64(1)))
			Expect(elapsedData.Len()).To(Equal(int64(1)))

			// Tick should be at least 1
			tick := telem.UnmarshalUint64[uint64](tickData.Data)
			Expect(tick).To(BeNumerically(">=", uint64(1)))
		})

		It("Should increment tick count on each execution", func() {
			n, err := factory.Create(ctx, node.Config{
				Node: ir.Node{
					Key:  "interval_1",
					Type: "interval",
					ConfigValues: map[string]any{
						"period": telem.TimeSpan(30 * gotime.Millisecond),
					},
				},
				State: progState.Node("interval_1"),
			})
			Expect(err).ToNot(HaveOccurred())

			n.Init(ctx, func(output string) {})

			// Wait for first tick
			gotime.Sleep(40 * gotime.Millisecond)
			n.Next(ctx, func(output string) {})

			stateNode := progState.Node("interval_1")
			tick1 := telem.UnmarshalUint64[uint64](stateNode.Output(0).Data)

			// Wait for second tick
			gotime.Sleep(40 * gotime.Millisecond)
			n.Next(ctx, func(output string) {})

			tick2 := telem.UnmarshalUint64[uint64](stateNode.Output(0).Data)

			Expect(tick2).To(BeNumerically(">", tick1))
		})
	})

	Describe("Time Wheel Integration", func() {
		It("Should register interval with time wheel", func() {
			n, err := factory.Create(ctx, node.Config{
				Node: ir.Node{
					Key:  "interval_1",
					Type: "interval",
					ConfigValues: map[string]any{
						"period": telem.TimeSpan(100 * gotime.Millisecond),
					},
				},
				State: progState.Node("interval_1"),
			})
			Expect(err).ToNot(HaveOccurred())

			n.Init(ctx, func(output string) {})

			// Wait for interval to fire
			gotime.Sleep(110 * gotime.Millisecond)

			// Time wheel should have state for this interval
			tick, timestamp, elapsed, ok := wheel.GetState("interval_1")
			Expect(ok).To(BeTrue())
			Expect(tick).To(BeNumerically(">=", uint64(1)))
			Expect(timestamp).To(BeNumerically(">", telem.TimeStamp(0)))
			Expect(elapsed).To(BeNumerically(">=", telem.TimeSpan(100*gotime.Millisecond)))
		})

		It("Should respect initial delay", func() {
			n, err := factory.Create(ctx, node.Config{
				Node: ir.Node{
					Key:  "interval_1",
					Type: "interval",
					ConfigValues: map[string]any{
						"period":        telem.TimeSpan(30 * gotime.Millisecond),
						"initial_delay": telem.TimeSpan(150 * gotime.Millisecond),
					},
				},
				State: progState.Node("interval_1"),
			})
			Expect(err).ToNot(HaveOccurred())

			n.Init(ctx, func(output string) {})

			// Before initial delay, tick should be 0
			gotime.Sleep(80 * gotime.Millisecond)
			tick1, _, _, _ := wheel.GetState("interval_1")
			Expect(tick1).To(Equal(uint64(0)))

			// After initial delay + one period, should have fired
			gotime.Sleep(130 * gotime.Millisecond)
			tick2, _, _, _ := wheel.GetState("interval_1")
			Expect(tick2).To(BeNumerically(">=", uint64(1)))
		})
	})

	Describe("Symbol Resolver", func() {
		It("Should resolve interval symbol", func() {
			sym, err := interval.SymbolResolver.Resolve(ctx, "interval")
			Expect(err).ToNot(HaveOccurred())
			Expect(sym.Name).To(Equal("interval"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
		})

		It("Should not resolve unknown symbols", func() {
			_, err := interval.SymbolResolver.Resolve(ctx, "unknown")
			Expect(err).To(HaveOccurred())
		})
	})
})
