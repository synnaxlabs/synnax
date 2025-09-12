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
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Select", func() {
	var (
		ctx  context.Context
		addr address.Address
		cfg  std.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		addr = address.Rand()
		cfg = std.Config{
			Node: ir.Node{
				Key:  "test_select",
				Type: "select",
			},
		}
	})

	Describe("Select Stage", func() {
		Context("Value routing based on condition", func() {
			It("Should output with 'true' param when value is 0", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				// Send a zero value
				v := value.Value{Address: addr, Param: "input", Type: ir.I32{}}.PutInt32(0)
				stage.Next(ctx, v)

				Expect(output.Param).To(Equal("true"))
				Expect(output.GetInt32()).To(Equal(int32(0)))
				Expect(output.Address).To(Equal(addr))
			})

			It("Should output with 'false' param when value is non-zero", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				// Send a non-zero value
				v := value.Value{Address: addr, Param: "input", Type: ir.I32{}}.PutInt32(42)
				stage.Next(ctx, v)

				Expect(output.Param).To(Equal("false"))
				Expect(output.GetInt32()).To(Equal(int32(42)))
				Expect(output.Address).To(Equal(addr))
			})
		})

		Context("Different numeric types", func() {
			It("Should handle float values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				// Test float zero
				v1 := value.Value{Address: addr, Type: ir.F64{}}.PutFloat64(0.0)
				stage.Next(ctx, v1)
				Expect(outputs[0].Param).To(Equal("true"))

				// Test non-zero float
				v2 := value.Value{Address: addr, Type: ir.F64{}}.PutFloat64(3.14)
				stage.Next(ctx, v2)
				Expect(outputs[1].Param).To(Equal("false"))
			})

			It("Should handle unsigned integers", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				// Test uint zero
				v1 := value.Value{Address: addr, Type: ir.U64{}}.PutUint64(0)
				stage.Next(ctx, v1)
				Expect(outputs[0].Param).To(Equal("true"))

				// Test non-zero uint
				v2 := value.Value{Address: addr, Type: ir.U64{}}.PutUint64(100)
				stage.Next(ctx, v2)
				Expect(outputs[1].Param).To(Equal("false"))
			})

			It("Should handle negative values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				// Negative values should result in "false"
				v := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(-10)
				stage.Next(ctx, v)

				Expect(output.Param).To(Equal("false"))
				Expect(output.GetInt32()).To(Equal(int32(-10)))
			})
		})

		Context("Boolean-like behavior", func() {
			It("Should treat boolean true (1) as false output", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				// Boolean true stored as 1
				v := value.Value{Address: addr, Type: ir.U8{}}.PutUint8(1)
				stage.Next(ctx, v)

				Expect(output.Param).To(Equal("false"))
				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})

			It("Should treat boolean false (0) as true output", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				// Boolean false stored as 0
				v := value.Value{Address: addr, Type: ir.U8{}}.PutUint8(0)
				stage.Next(ctx, v)

				Expect(output.Param).To(Equal("true"))
				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})

		Context("Multiple values", func() {
			It("Should handle multiple values in sequence", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					outputs = append(outputs, val)
				})

				// Send multiple values
				values := []struct {
					val      int32
					expected string
				}{
					{0, "true"},
					{1, "false"},
					{0, "true"},
					{-1, "false"},
					{100, "false"},
					{0, "true"},
				}

				for _, test := range values {
					v := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(test.val)
					stage.Next(ctx, v)
				}

				Expect(outputs).To(HaveLen(6))
				for i, test := range values {
					Expect(outputs[i].Param).To(Equal(test.expected))
					Expect(outputs[i].GetInt32()).To(Equal(test.val))
				}
			})
		})

		Context("Integration with other stages", func() {
			It("Should work with comparison operator output", func() {
				// Create an EQ operator
				eqCfg := std.Config{
					Node: ir.Node{
						Key:  "test_eq",
						Type: "eq",
					},
				}
				eqStage, err := std.Create(ctx, eqCfg)
				Expect(err).ToNot(HaveOccurred())

				// Create select stage
				selectStage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				// Wire EQ output to select input
				eqStage.OnOutput(func(ctx context.Context, val value.Value) {
					selectStage.Next(ctx, val)
				})

				var selectOutput value.Value
				selectStage.OnOutput(func(_ context.Context, val value.Value) {
					selectOutput = val
				})

				// Test equal values (should output 1, which select routes to "false")
				v1 := value.Value{Address: addr, Param: "a", Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Address: addr, Param: "b", Type: ir.I32{}}.PutInt32(10)
				eqStage.Next(ctx, v1)
				eqStage.Next(ctx, v2)

				Expect(selectOutput.Param).To(Equal("false")) // Because EQ outputs 1 for true
				Expect(selectOutput.GetUint8()).To(Equal(uint8(1)))

				// Test unequal values (should output 0, which select routes to "true")
				v3 := value.Value{Address: addr, Param: "a", Type: ir.I32{}}.PutInt32(10)
				v4 := value.Value{Address: addr, Param: "b", Type: ir.I32{}}.PutInt32(20)
				eqStage.Next(ctx, v3)
				eqStage.Next(ctx, v4)

				Expect(selectOutput.Param).To(Equal("true")) // Because EQ outputs 0 for false
				Expect(selectOutput.GetUint8()).To(Equal(uint8(0)))
			})

			It("Should work with constant stage", func() {
				// Create a constant stage with value 0
				constCfg := std.Config{
					Node: ir.Node{
						Key:  "test_const",
						Type: "constant",
						Config: map[string]any{
							"value": int32(0),
						},
					},
				}
				constStage, err := std.Create(ctx, constCfg)
				Expect(err).ToNot(HaveOccurred())

				// Create select stage
				selectStage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				// Wire constant output to select input
				constStage.OnOutput(func(ctx context.Context, val value.Value) {
					selectStage.Next(ctx, val)
				})

				var selectOutput value.Value
				selectStage.OnOutput(func(_ context.Context, val value.Value) {
					selectOutput = val
				})

				// Trigger constant
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				constStage.Flow(sCtx)

				// Constant 0 should be routed to "true"
				Expect(selectOutput.Param).To(Equal("true"))
				Expect(selectOutput.GetInt32()).To(Equal(int32(0)))
			})
		})

		Context("Value preservation", func() {
			It("Should preserve the original value type and data", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, val value.Value) {
					output = val
				})

				// Test with a specific float value
				v := value.Value{
					Address: addr,
					Param:   "input",
					Type:    ir.F32{},
				}.PutFloat32(123.456)
				
				stage.Next(ctx, v)

				// Value should be preserved, only param changed
				Expect(output.Address).To(Equal(addr))
				Expect(output.Type).To(Equal(ir.F32{}))
				Expect(output.GetFloat32()).To(BeNumerically("~", float32(123.456), 0.001))
				Expect(output.Param).To(Equal("false")) // Non-zero routes to false
			})
		})
	})
})