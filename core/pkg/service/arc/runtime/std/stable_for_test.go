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
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("StableFor", func() {
	var (
		ctx      context.Context
		cfg      std.Config
		mockTime telem.TimeStamp
		mockNow  func() telem.TimeStamp
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockTime = telem.TimeStamp(0)
		mockNow = func() telem.TimeStamp {
			return mockTime
		}
		cfg = std.Config{
			Node: ir.Node{
				Key:  "test_stable_for",
				Type: "stable_for",
				Config: map[string]any{
					"duration": int(telem.Second * 5),
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
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					outputs = append(outputs, val)
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				stage.Next(ctx, "input", v1)
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(3 * telem.Second)
				stage.Next(ctx, "input", v1)
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, "input", v1)
				Expect(outputs).To(HaveLen(1))
				Expect(outputs[0].GetInt32()).To(Equal(int32(10)))
			})

			It("Should reset timer when value changes", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					outputs = append(outputs, val)
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				stage.Next(ctx, "input", v1)

				mockTime = telem.TimeStamp(4 * telem.Second)
				stage.Next(ctx, "input", v1)
				Expect(outputs).To(HaveLen(0))

				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)
				stage.Next(ctx, "input", v2)
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(8 * telem.Second)
				stage.Next(ctx, "input", v2)
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(9 * telem.Second)
				stage.Next(ctx, "input", v2)
				Expect(outputs).To(HaveLen(1))
				Expect(outputs[0].GetInt32()).To(Equal(int32(20)))
			})
		})

		Context("Multiple value changes", func() {
			It("Should handle rapid value changes", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					outputs = append(outputs, val)
				})

				for i := 0; i < 10; i++ {
					mockTime = telem.TimeStamp(telem.TimeSpan(i) * telem.Second)
					v := value.Value{Type: ir.I32{}}.PutInt32(int32(i))
					stage.Next(ctx, "input", v)
				}
				Expect(outputs).To(HaveLen(0))

				v := value.Value{Type: ir.I32{}}.PutInt32(9)

				mockTime = telem.TimeStamp(12 * telem.Second)
				stage.Next(ctx, "input", v)
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(14 * telem.Second)
				stage.Next(ctx, "input", v)
				Expect(outputs).To(HaveLen(1))
				Expect(outputs[0].GetInt32()).To(Equal(int32(9)))
			})

			It("Should output each stable value only once", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					outputs = append(outputs, val)
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				stage.Next(ctx, "input", v1)

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, "input", v1)
				Expect(outputs).To(HaveLen(1))

				mockTime = telem.TimeStamp(10 * telem.Second)
				stage.Next(ctx, "input", v1)
				Expect(outputs).To(HaveLen(1))

				mockTime = telem.TimeStamp(11 * telem.Second)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)
				stage.Next(ctx, "input", v2)
				Expect(outputs).To(HaveLen(1))

				mockTime = telem.TimeStamp(16 * telem.Second)
				stage.Next(ctx, "input", v2)
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
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v := value.Value{Type: ir.F64{}}.PutFloat64(3.14)
				stage.Next(ctx, "input", v)

				mockTime = telem.TimeStamp(100 * telem.Millisecond)
				stage.Next(ctx, "input", v)
				Expect(output.GetFloat64()).To(Equal(3.14))
			})

			It("Should work with very long duration", func() {
				cfg.Node.Config["duration"] = int(telem.Hour)
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					outputs = append(outputs, val)
				})

				v := value.Value{Type: ir.I32{}}.PutInt32(42)
				stage.Next(ctx, "input", v)

				mockTime = telem.TimeStamp(59 * telem.Minute)
				stage.Next(ctx, "input", v)
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(telem.Hour)
				stage.Next(ctx, "input", v)
				Expect(outputs).To(HaveLen(1))
			})
		})

		Context("Edge cases", func() {
			It("Should handle zero value", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v := value.Value{Type: ir.I32{}}.PutInt32(0)
				stage.Next(ctx, "input", v)

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, "input", v)
				Expect(output.GetInt32()).To(Equal(int32(0)))
			})

			It("Should handle negative values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v := value.Value{Type: ir.I32{}}.PutInt32(-100)
				stage.Next(ctx, "input", v)

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, "input", v)
				Expect(output.GetInt32()).To(Equal(int32(-100)))
			})

			It("Should handle value toggling", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					outputs = append(outputs, val)
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(1)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(2)

				for i := 0; i < 10; i++ {
					mockTime = telem.TimeStamp(telem.TimeSpan(i) * telem.Second)
					if i%2 == 0 {
						stage.Next(ctx, "input", v1)
					} else {
						stage.Next(ctx, "input", v2)
					}
				}
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(10 * telem.Second)
				stage.Next(ctx, "input", v1)

				mockTime = telem.TimeStamp(15 * telem.Second)
				stage.Next(ctx, "input", v1)
				Expect(outputs).To(HaveLen(1))
				Expect(outputs[0].GetInt32()).To(Equal(int32(1)))
			})
		})

		Context("Integration with other stages", func() {
			It("Should work with comparison operator output", func() {
				gtCfg := std.Config{
					Node: ir.Node{
						Key:  "test_gt",
						Type: "gt",
					},
				}
				gtStage, err := std.Create(ctx, gtCfg)
				Expect(err).ToNot(HaveOccurred())

				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				gtStage.OnOutput(func(ctx context.Context, param string, val value.Value) {
					stage.Next(ctx, "input", val)
				})

				var stableOutput value.Value
				outputCount := 0
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					stableOutput = val
					outputCount++
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(20)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(10)
				gtStage.Next(ctx, "a", v1)
				gtStage.Next(ctx, "b", v2)

				mockTime = telem.TimeStamp(2 * telem.Second)
				gtStage.Next(ctx, "a", v1)
				gtStage.Next(ctx, "b", v2)

				mockTime = telem.TimeStamp(5 * telem.Second)
				gtStage.Next(ctx, "a", v1)
				gtStage.Next(ctx, "b", v2)

				Expect(outputCount).To(Equal(1))
				Expect(stableOutput.GetUint8()).To(Equal(uint8(1)))

				mockTime = telem.TimeStamp(6 * telem.Second)
				v3 := value.Value{Type: ir.I32{}}.PutInt32(5)
				v4 := value.Value{Type: ir.I32{}}.PutInt32(10)
				gtStage.Next(ctx, "a", v3)
				gtStage.Next(ctx, "b", v4)

				mockTime = telem.TimeStamp(11 * telem.Second)
				gtStage.Next(ctx, "a", v3)
				gtStage.Next(ctx, "b", v4)

				Expect(outputCount).To(Equal(2))
				Expect(stableOutput.GetUint8()).To(Equal(uint8(0)))
			})
		})

		Context("Value types", func() {
			It("Should handle float values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v := value.Value{Type: ir.F64{}}.PutFloat64(3.14159)
				stage.Next(ctx, "input", v)

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, "input", v)
				Expect(output.GetFloat64()).To(BeNumerically("~", 3.14159, 0.00001))
			})

			It("Should handle uint values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v := value.Value{Type: ir.U64{}}.PutUint64(uint64(1 << 40))
				stage.Next(ctx, "input", v)

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, "input", v)
				Expect(output.GetUint64()).To(Equal(uint64(1 << 40)))
			})
		})

		Context("Time boundary conditions", func() {
			It("Should output exactly at duration boundary", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					outputs = append(outputs, val)
				})

				v := value.Value{Type: ir.I32{}}.PutInt32(100)
				stage.Next(ctx, "input", v)

				mockTime = telem.TimeStamp(5*telem.Second - 1)
				stage.Next(ctx, "input", v)
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(5 * telem.Second)
				stage.Next(ctx, "input", v)
				Expect(outputs).To(HaveLen(1))
			})

			It("Should handle time going backwards gracefully", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					outputs = append(outputs, val)
				})

				v := value.Value{Type: ir.I32{}}.PutInt32(50)
				mockTime = telem.TimeStamp(2 * telem.Second)
				stage.Next(ctx, "input", v)

				mockTime = telem.TimeStamp(1 * telem.Second)
				stage.Next(ctx, "input", v)
				Expect(outputs).To(HaveLen(0))

				mockTime = telem.TimeStamp(7 * telem.Second)
				stage.Next(ctx, "input", v)
				Expect(outputs).To(HaveLen(1))
			})
		})
	})
})
