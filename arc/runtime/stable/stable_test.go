package stable_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/stable"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

var ctx = context.Background()

var _ = Describe("StableFor", func() {
	Describe("NewFactory", func() {
		It("Should create factory", func() {
			factory := stable.NewFactory(stable.FactoryConfig{})
			Expect(factory).ToNot(BeNil())
		})
	})

	Describe("Factory.Create", func() {
		var factory node.Factory
		var s *state.State
		BeforeEach(func() {
			factory = stable.NewFactory(stable.FactoryConfig{})
			s = state.New(state.Config{
				Nodes: ir.Nodes{
					{
						Key:  "source",
						Type: "source",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
					{
						Key:  "stable",
						Type: "stable_for",
						ConfigValues: map[string]interface{}{
							"duration": int64(1000000000), // 1 second
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			})
		})

		It("Should create node for stable_for type", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(1000000000),
					},
				},
				State: s.Node("stable"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})

		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "unknown",
					ConfigValues: map[string]interface{}{
						"duration": int64(1000000000),
					},
				},
				State: s.Node("stable"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
		})

		It("Should parse duration configuration", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(5000000000), // 5 seconds
					},
				},
				State: s.Node("stable"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})
	})

	Describe("stableFor.Init", func() {
		It("Should not emit output on Init", func() {
			currentTime := telem.TimeStamp(0)
			factory := stable.NewFactory(stable.FactoryConfig{
				Now: func() telem.TimeStamp { return currentTime },
			})
			s := state.New(state.Config{
				Nodes: ir.Nodes{
					{
						Key:  "source",
						Type: "source",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
					{
						Key:  "stable",
						Type: "stable_for",
						ConfigValues: map[string]interface{}{
							"duration": int64(telem.SecondTS),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			})
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			n, _ := factory.Create(ctx, cfg)
			outputs := []string{}
			n.Init(ctx, func(output string) {
				outputs = append(outputs, output)
			})
			Expect(outputs).To(BeEmpty())
		})
	})

	Describe("stableFor.Next", func() {
		var s *state.State
		var factory node.Factory
		var currentTime telem.TimeStamp

		BeforeEach(func() {
			currentTime = 0
			factory = stable.NewFactory(stable.FactoryConfig{
				Now: func() telem.TimeStamp { return currentTime },
			})
			s = state.New(state.Config{
				Nodes: ir.Nodes{
					{
						Key:  "source",
						Type: "source",
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
					{
						Key:  "stable",
						Type: "stable_for",
						ConfigValues: map[string]interface{}{
							"duration": int64(telem.SecondTS),
						},
						Outputs: types.Params{
							Keys:   []string{ir.DefaultOutputParam},
							Values: []types.Type{types.U8()},
						},
					},
				},
				Edges: ir.Edges{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "stable", Param: ir.DefaultInputParam},
					},
				},
			})
		})

		It("Should handle empty input", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			*source.Output(0) = telem.NewSeriesV[uint8]()
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV()
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeFalse())
		})

		It("Should not emit when value is not stable for duration", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send value 5 at time 0
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeFalse())

			// Advance time but not enough
			currentTime = telem.SecondTS / 2
			outputs = make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeFalse())
		})

		It("Should emit when value is stable for duration", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send value 5 at time 0
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeFalse())

			// Advance time to exactly duration (no new input data)
			currentTime = telem.SecondTS
			*source.Output(0) = telem.NewSeriesV[uint8]()
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV()
			outputs = make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeTrue())

			stableNode := s.Node("stable")
			output := stableNode.Output(0)
			Expect(output.Len()).To(Equal(int64(1)))
			outputVals := telem.UnmarshalSeries[uint8](*output)
			Expect(outputVals).To(Equal([]uint8{5}))
		})

		It("Should reset timer when value changes", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send value 5 at time 0
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(ctx, func(string) {})

			// Advance time partway
			currentTime = telem.SecondTS / 2

			// Send different value 10 at time 1s
			*source.Output(0) = telem.NewSeriesV[uint8](10)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeFalse())

			// Advance to 1.5 seconds from start (0.5s since change at time 1s)
			currentTime = telem.SecondTS + telem.SecondTS/2
			outputs = make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			// Should not emit yet - value changed at 1s, only 0.5s elapsed
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeFalse())

			// Advance to 2 seconds from start (1s since change at time 1s)
			currentTime = telem.SecondTS * 2
			outputs = make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeTrue())
		})

		It("Should not emit same value twice", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send value 5 at time 0
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(ctx, func(string) {})

			// Advance time to emit
			currentTime = telem.SecondTS
			*source.Output(0) = telem.NewSeriesV[uint8]()
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV()
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeTrue())

			// Call again with same value - should not emit
			currentTime = telem.SecondTS * 2
			*source.Output(0) = telem.NewSeriesV[uint8]()
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV()
			outputs = make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeFalse())
		})

		It("Should emit different value after stable period", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send value 5
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(ctx, func(string) {})

			// Emit first value
			currentTime = telem.SecondTS
			n.Next(ctx, func(string) {})

			// Change to value 10
			currentTime = telem.SecondTS * 2
			*source.Output(0) = telem.NewSeriesV[uint8](10)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(2)
			n.Next(ctx, func(string) {})

			// Wait for stability
			currentTime = telem.SecondTS * 3
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeTrue())

			stableNode := s.Node("stable")
			output := stableNode.Output(0)
			outputVals := telem.UnmarshalSeries[uint8](*output)
			Expect(outputVals).To(Equal([]uint8{10}))
		})

		It("Should handle multiple values in single input", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send multiple values, ending with 7 at 0.4s (400ms)
			*source.Output(0) = telem.NewSeriesV[uint8](3, 4, 5, 6, 7)
			*source.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](
				0,
				telem.SecondTS/10,   // 0.1s = 100ms
				telem.SecondTS/5,    // 0.2s = 200ms
				telem.SecondTS*3/10, // 0.3s = 300ms
				telem.SecondTS*2/5,  // 0.4s = 400ms
			)
			n, _ := factory.Create(ctx, cfg)
			n.Next(ctx, func(string) {})

			// Should track last value (7) with time 0.4s, so wait until 1.4s elapsed
			currentTime = telem.SecondTS + telem.SecondTS*2/5 // 1.4s
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeTrue())

			stableNode := s.Node("stable")
			output := stableNode.Output(0)
			outputVals := telem.UnmarshalSeries[uint8](*output)
			Expect(outputVals).To(Equal([]uint8{7}))
		})

		It("Should handle zero duration", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(0),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
			n, _ := factory.Create(ctx, cfg)
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			// Should emit immediately with zero duration
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeTrue())
		})

		It("Should use output timestamp as current time not input time", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			*source.Output(0) = telem.NewSeriesV[uint8](5)
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV(0)
			n, _ := factory.Create(ctx, cfg)
			n.Next(ctx, func(string) {})

			currentTime = telem.SecondTS * 100 // Set current time far in future
			*source.Output(0) = telem.NewSeriesV[uint8]()
			*source.OutputTime(0) = telem.NewSeriesSecondsTSV()
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeTrue())

			stableNode := s.Node("stable")
			outputTime := stableNode.OutputTime(0)
			outputTimes := telem.UnmarshalSeries[telem.TimeStamp](*outputTime)
			Expect(outputTimes).To(Equal([]telem.TimeStamp{telem.SecondTS * 100}))
		})

		It("Should handle same value repeated in input", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "stable_for",
					ConfigValues: map[string]interface{}{
						"duration": int64(telem.SecondTS),
					},
				},
				State: s.Node("stable"),
			}
			source := s.Node("source")
			currentTime = 0
			// Send same value multiple times
			*source.Output(0) = telem.NewSeriesV[uint8](5, 5, 5, 5)
			*source.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](
				0,
				telem.SecondTS/10, // 0.1s = 100ms
				telem.SecondTS/5,  // 0.2s = 200ms
				telem.SecondTS*3/10, // 0.3s = 300ms
			)
			n, _ := factory.Create(ctx, cfg)
			n.Next(ctx, func(string) {})

			// Should use time from first occurrence (0)
			currentTime = telem.SecondTS
			outputs := make(set.Set[string])
			n.Next(ctx, func(output string) { outputs.Add(output) })
			Expect(outputs.Contains(ir.DefaultOutputParam)).To(BeTrue())
		})
	})

	Describe("SymbolResolver", func() {
		It("Should resolve stable_for symbol", func() {
			sym, ok := stable.SymbolResolver["stable_for"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("stable_for"))
		})
	})
})