// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/std"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("StableFor", func() {
	var (
		ctx      context.Context
		addr     address.Address
		cfg      std.Config
		mockTime telem.TimeStamp
		mockNow  func() telem.TimeStamp
	)

	BeforeEach(func() {
		ctx = context.Background()
		addr = address.Rand()
		mockTime = telem.TimeStamp(0)
		mockNow = func() telem.TimeStamp {
			return mockTime
		}
		cfg = std.Config{
			Node: ir.Node{
				Key:  "test_stable_for",
				Type: "stable_for",
				Config: map[string]any{
					"duration": int(telem.Second * 5), // 5 seconds stability required
				},
			},
			Now: mockNow,
		}
	})

	Describe("Stable For Stage", func() {
		Context("Basic stability detection", func() {
			It("Should not output until value is stable for duration", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				// Send initial value at time 0
				v1 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)
				stage.Next(ctx, v1)
				Expect(outputs).To(HaveLen(0)) // No output yet

				// Advance time by 3 seconds (less than duration)
				mockTime = telem.TimeStamp(3 * telem.Second)
				stage.Next(ctx, v1)
				Expect(outputs).To(HaveLen(0)) // Still no output

				// Advance time to 5 seconds (exactly at duration)
				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, v1)
				Expect(outputs).To(HaveLen(1)) // Should output now
				Expect(outputs[0].GetInt32()).To(Equal(int32(10)))
			})

			It("Should reset timer when value changes", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				// Send initial value at time 0
				v1 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)
				stage.Next(ctx, v1)

				// Advance time by 4 seconds
				mockTime = telem.TimeStamp(4 * telem.Second)
				stage.Next(ctx, v1)
				Expect(outputs).To(HaveLen(0))

				// Change value (resets timer)
				v2 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(20)
				stage.Next(ctx, v2)
				Expect(outputs).To(HaveLen(0))

				// Advance time by 4 more seconds (total 8, but only 4 since value change)
				mockTime = telem.TimeStamp(8 * telem.Second)
				stage.Next(ctx, v2)
				Expect(outputs).To(HaveLen(0)) // Still not stable long enough

				// Advance to 9 seconds (5 seconds since value change)
				mockTime = telem.TimeStamp(9 * telem.Second)
				stage.Next(ctx, v2)
				Expect(outputs).To(HaveLen(1))
				Expect(outputs[0].GetInt32()).To(Equal(int32(20)))
			})
		})

		Context("Multiple value changes", func() {
			It("Should handle rapid value changes", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				// Rapid value changes
				for i := 0; i < 10; i++ {
					mockTime = telem.TimeStamp(telem.TimeSpan(i) * telem.Second)
					v := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(int32(i))
					stage.Next(ctx, v)
				}
				Expect(outputs).To(HaveLen(0)) // No stable value

				// Keep last value stable
				v := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(9)
				
				// Advance time and keep sending same value
				mockTime = telem.TimeStamp(12 * telem.Second)
				stage.Next(ctx, v)
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(14 * telem.Second)
				stage.Next(ctx, v)
				Expect(outputs).To(HaveLen(1))
				Expect(outputs[0].GetInt32()).To(Equal(int32(9)))
			})

			It("Should output each stable value only once", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				// First stable value
				v1 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)
				stage.Next(ctx, v1)
				
				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, v1)
				Expect(outputs).To(HaveLen(1))

				// Keep sending same value - should not output again
				mockTime = telem.TimeStamp(10 * telem.Second)
				stage.Next(ctx, v1)
				Expect(outputs).To(HaveLen(1))

				// Change to new value
				mockTime = telem.TimeStamp(11 * telem.Second)
				v2 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(20)
				stage.Next(ctx, v2)
				Expect(outputs).To(HaveLen(1))

				// Wait for stability
				mockTime = telem.TimeStamp(16 * telem.Second)
				stage.Next(ctx, v2)
				Expect(outputs).To(HaveLen(2))
				Expect(outputs[1].GetInt32()).To(Equal(int32(20)))
			})
		})

		Context("Different durations", func() {
			It("Should work with very short duration", func() {
				cfg.Node.Config["duration"] = int(100 * telem.Millisecond)
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				v := value.Value{Address: addr, Type: ir.F64{}}.PutFloat64(3.14)
				stage.Next(ctx, v)

				mockTime = telem.TimeStamp(100 * telem.Millisecond)
				stage.Next(ctx, v)
				Expect(output.GetFloat64()).To(Equal(3.14))
			})

			It("Should work with very long duration", func() {
				cfg.Node.Config["duration"] = int(telem.Hour)
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				v := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(42)
				stage.Next(ctx, v)

				// Not stable after 59 minutes
				mockTime = telem.TimeStamp(59 * telem.Minute)
				stage.Next(ctx, v)
				Expect(outputs).To(HaveLen(0))

				// Stable after 1 hour
				mockTime = telem.TimeStamp(telem.Hour)
				stage.Next(ctx, v)
				Expect(outputs).To(HaveLen(1))
			})
		})

		Context("Edge cases", func() {
			It("Should handle zero value", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				v := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(0)
				stage.Next(ctx, v)

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, v)
				Expect(output.GetInt32()).To(Equal(int32(0)))
			})

			It("Should handle negative values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				v := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(-100)
				stage.Next(ctx, v)

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, v)
				Expect(output.GetInt32()).To(Equal(int32(-100)))
			})

			It("Should handle value toggling", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				v1 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(1)
				v2 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(2)

				// Toggle between values
				for i := 0; i < 10; i++ {
					mockTime = telem.TimeStamp(telem.TimeSpan(i) * telem.Second)
					if i%2 == 0 {
						stage.Next(ctx, v1)
					} else {
						stage.Next(ctx, v2)
					}
				}
				Expect(outputs).To(HaveLen(0)) // Never stable

				// Stabilize on v1
				mockTime = telem.TimeStamp(10 * telem.Second)
				stage.Next(ctx, v1)
				
				mockTime = telem.TimeStamp(15 * telem.Second)
				stage.Next(ctx, v1)
				Expect(outputs).To(HaveLen(1))
				Expect(outputs[0].GetInt32()).To(Equal(int32(1)))
			})
		})

		Context("Integration with other stages", func() {
			It("Should work with comparison operator output", func() {
				// Create a GT operator
				gtCfg := std.Config{
					Node: ir.Node{
						Key:  "test_gt",
						Type: "gt",
					},
				}
				gtStage, err := std.Create(ctx, gtCfg)
				Expect(err).ToNot(HaveOccurred())

				// Create stable_for stage
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				// Wire GT output to stable_for input
				gtStage.OnOutput(func(ctx context.Context, val value.Value) {
					stage.Next(ctx, val)
				})

				var stableOutput value.Value
				outputCount := 0
				stage.OnOutput(func(_ context.Context, val value.Value) {
					stableOutput = val
					outputCount++
				})

				// Send values that result in GT = 1 (true)
				v1 := value.Value{Address: addr, Param: "a", Type: ir.I32{}}.PutInt32(20)
				v2 := value.Value{Address: addr, Param: "b", Type: ir.I32{}}.PutInt32(10)
				gtStage.Next(ctx, v1)
				gtStage.Next(ctx, v2)

				// Keep sending same comparison
				mockTime = telem.TimeStamp(2 * telem.Second)
				gtStage.Next(ctx, v1)
				gtStage.Next(ctx, v2)

				mockTime = telem.TimeStamp(5 * telem.Second)
				gtStage.Next(ctx, v1)
				gtStage.Next(ctx, v2)

				Expect(outputCount).To(Equal(1))
				Expect(stableOutput.GetUint8()).To(Equal(uint8(1)))

				// Change comparison result
				mockTime = telem.TimeStamp(6 * telem.Second)
				v3 := value.Value{Address: addr, Param: "a", Type: ir.I32{}}.PutInt32(5)
				v4 := value.Value{Address: addr, Param: "b", Type: ir.I32{}}.PutInt32(10)
				gtStage.Next(ctx, v3)
				gtStage.Next(ctx, v4) // GT = 0 (false)

				// Wait for stability
				mockTime = telem.TimeStamp(11 * telem.Second)
				gtStage.Next(ctx, v3)
				gtStage.Next(ctx, v4)

				Expect(outputCount).To(Equal(2))
				Expect(stableOutput.GetUint8()).To(Equal(uint8(0)))
			})
		})

		Context("Value types", func() {
			It("Should handle float values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				v := value.Value{Address: addr, Type: ir.F64{}}.PutFloat64(3.14159)
				stage.Next(ctx, v)

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, v)
				Expect(output.GetFloat64()).To(BeNumerically("~", 3.14159, 0.00001))
			})

			It("Should handle uint values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				v := value.Value{Address: addr, Type: ir.U64{}}.PutUint64(uint64(1<<40))
				stage.Next(ctx, v)

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, v)
				Expect(output.GetUint64()).To(Equal(uint64(1 << 40)))
			})
		})

		Context("Time boundary conditions", func() {
			It("Should output exactly at duration boundary", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				v := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(100)
				stage.Next(ctx, v)

				// Just before duration
				mockTime = telem.TimeStamp(5*telem.Second - 1)
				stage.Next(ctx, v)
				Expect(outputs).To(HaveLen(0))

				// Exactly at duration
				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, v)
				Expect(outputs).To(HaveLen(1))
			})

			It("Should handle time going backwards gracefully", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				v := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(50)
				mockTime = telem.TimeStamp(2 * telem.Second)
				stage.Next(ctx, v)

				// Time goes backwards
				mockTime = telem.TimeStamp(1 * telem.Second)
				stage.Next(ctx, v)
				Expect(outputs).To(HaveLen(0))

				// Advance forward again
				mockTime = telem.TimeStamp(7 * telem.Second)
				stage.Next(ctx, v)
				Expect(outputs).To(HaveLen(1))
			})
		})
	})
})