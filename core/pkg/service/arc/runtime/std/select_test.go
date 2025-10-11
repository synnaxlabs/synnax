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
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Select", func() {
	var (
		ctx context.Context
		cfg std.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		cfg = std.Config{
			Node: ir.Node{
				Key:  "test_select",
				Type: "select",
			},
		}
	})

	Describe("Select Stage", func() {
		Context("Value routing based on condition", func() {
			It("Should output with 'false' param when value is 0", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				var outputParam string
				stage.OnOutput(func(_ context.Context, param string, val value.Value) {
					output = val
					outputParam = param
				})

				v := value.Value{Type: ir.I32{}}.PutInt32(0)
				stage.Next(ctx, "input", v)

				Expect(outputParam).To(Equal("false"))
				Expect(output.GetInt32()).To(Equal(int32(0)))

			})

			It("Should output with 'true' param when value is non-zero", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				var outputParam string
				stage.OnOutput(func(_ context.Context, param string, val value.Value) {
					output = val
					outputParam = param
				})

				v := value.Value{Type: ir.I32{}}.PutInt32(42)
				stage.Next(ctx, "input", v)

				Expect(outputParam).To(Equal("true"))
				Expect(output.GetInt32()).To(Equal(int32(42)))

			})
		})

		Context("Different numeric types", func() {
			It("Should handle float values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				var outputParams []string
				stage.OnOutput(func(_ context.Context, param string, val value.Value) {
					outputs = append(outputs, val)
					outputParams = append(outputParams, param)
				})

				v1 := value.Value{Type: ir.F64{}}.PutFloat64(0.0)
				stage.Next(ctx, "input", v1)
				Expect(outputParams[0]).To(Equal("false"))

				v2 := value.Value{Type: ir.F64{}}.PutFloat64(3.14)
				stage.Next(ctx, "input", v2)
				Expect(outputParams[1]).To(Equal("true"))
			})

			It("Should handle unsigned integers", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				var outputParams []string
				stage.OnOutput(func(_ context.Context, param string, val value.Value) {
					outputs = append(outputs, val)
					outputParams = append(outputParams, param)
				})

				v1 := value.Value{Type: ir.U64{}}.PutUint64(0)
				stage.Next(ctx, "input", v1)
				Expect(outputParams[0]).To(Equal("false"))

				v2 := value.Value{Type: ir.U64{}}.PutUint64(100)
				stage.Next(ctx, "input", v2)
				Expect(outputParams[1]).To(Equal("true"))
			})

			It("Should handle negative values", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				var outputParam string
				stage.OnOutput(func(_ context.Context, param string, val value.Value) {
					output = val
					outputParam = param
				})

				v := value.Value{Type: ir.I32{}}.PutInt32(-10)
				stage.Next(ctx, "input", v)

				Expect(outputParam).To(Equal("true"))
				Expect(output.GetInt32()).To(Equal(int32(-10)))
			})
		})

		Context("Boolean-like behavior", func() {
			It("Should treat boolean true (1) as true output", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				var outputParam string
				stage.OnOutput(func(_ context.Context, param string, val value.Value) {
					output = val
					outputParam = param
				})

				v := value.Value{Type: ir.U8{}}.PutUint8(1)
				stage.Next(ctx, "input", v)

				Expect(outputParam).To(Equal("true"))
				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})

			It("Should treat boolean false (0) as false output", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				var outputParam string
				stage.OnOutput(func(_ context.Context, param string, val value.Value) {
					output = val
					outputParam = param
				})

				v := value.Value{Type: ir.U8{}}.PutUint8(0)
				stage.Next(ctx, "input", v)

				Expect(outputParam).To(Equal("false"))
				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})

		Context("Multiple values", func() {
			It("Should handle multiple values in sequence", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var outputs []value.Value
				var outputParams []string
				stage.OnOutput(func(_ context.Context, param string, val value.Value) {
					outputs = append(outputs, val)
					outputParams = append(outputParams, param)
				})

				values := []struct {
					val      int32
					expected string
				}{
					{0, "false"},
					{1, "true"},
					{0, "false"},
					{-1, "true"},
					{100, "true"},
					{0, "false"},
				}

				for _, test := range values {
					v := value.Value{Type: ir.I32{}}.PutInt32(test.val)
					stage.Next(ctx, "input", v)
				}

				Expect(outputs).To(HaveLen(6))
				for i, test := range values {
					Expect(outputParams[i]).To(Equal(test.expected))
					Expect(outputs[i].GetInt32()).To(Equal(test.val))
				}
			})
		})

		Context("Integration with other stages", func() {
			It("Should work with comparison operator output", func() {
				eqCfg := std.Config{
					Node: ir.Node{
						Key:  "test_eq",
						Type: "eq",
					},
				}
				eqStage, err := std.Create(ctx, eqCfg)
				Expect(err).ToNot(HaveOccurred())

				selectStage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				eqStage.OnOutput(func(ctx context.Context, param string, val value.Value) {
					selectStage.Next(ctx, "input", val)
				})

				var selectOutput value.Value
				var selectParam string
				selectStage.OnOutput(func(_ context.Context, param string, val value.Value) {
					selectOutput = val
					selectParam = param
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(10)
				eqStage.Next(ctx, "a", v1)
				eqStage.Next(ctx, "b", v2)

				Expect(selectParam).To(Equal("true"))
				Expect(selectOutput.GetUint8()).To(Equal(uint8(1)))

				v3 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v4 := value.Value{Type: ir.I32{}}.PutInt32(20)
				eqStage.Next(ctx, "a", v3)
				eqStage.Next(ctx, "b", v4)

				Expect(selectParam).To(Equal("false"))
				Expect(selectOutput.GetUint8()).To(Equal(uint8(0)))
			})

			It("Should work with constant stage", func() {
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

				selectStage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				constStage.OnOutput(func(ctx context.Context, param string, val value.Value) {
					selectStage.Next(ctx, "input", val)
				})

				var selectOutput value.Value
				var selectParam string
				selectStage.OnOutput(func(_ context.Context, param string, val value.Value) {
					selectOutput = val
					selectParam = param
				})

				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				constStage.Flow(sCtx)

				Expect(selectParam).To(Equal("false"))
				Expect(selectOutput.GetInt32()).To(Equal(int32(0)))
			})
		})

		Context("Value preservation", func() {
			It("Should preserve the original value type and data", func() {
				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				var outputParam string
				stage.OnOutput(func(_ context.Context, param string, val value.Value) {
					output = val
					outputParam = param
				})

				v := value.Value{

					Type: ir.F32{},
				}.PutFloat32(123.456)

				stage.Next(ctx, "input", v)

				Expect(output.Type).To(Equal(ir.F32{}))
				Expect(output.GetFloat32()).To(BeNumerically("~", float32(123.456), 0.001))
				Expect(outputParam).To(Equal("true"))
			})
		})
	})
})
