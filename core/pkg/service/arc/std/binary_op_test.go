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
	std2 "github.com/synnaxlabs/synnax/pkg/service/arc/std"
	"github.com/synnaxlabs/synnax/pkg/service/arc/value"
)

var _ = Describe("BinaryOp", func() {
	var (
		ctx context.Context
		cfg std2.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		cfg = std2.Config{
			Node: ir.Node{
				Key: "test_op",
			},
		}
	})

	Describe("Comparison Operators", func() {
		Context("EQ operator", func() {
			It("Should return 1 when values are equal", func() {
				stage, err := std2.EQFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(10)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})

			It("Should return 0 when values are not equal", func() {
				stage, err := std2.EQFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})

			It("Should handle mixed types with coercion", func() {
				stage, err := std2.EQFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.F32{}}.PutFloat32(10.0)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(10)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})

		Context("NE operator", func() {
			It("Should return 1 when values are not equal", func() {
				stage, err := std2.NEFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})

		Context("GT operator", func() {
			It("Should return 1 when a > b", func() {
				stage, err := std2.GTFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(20)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(10)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})

			It("Should return 0 when a <= b", func() {
				stage, err := std2.GTFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})

		Context("GE operator", func() {
			It("Should return 1 when a >= b", func() {
				stage, err := std2.GEFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(10)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})

		Context("LT operator", func() {
			It("Should return 1 when a < b", func() {
				stage, err := std2.LTFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})

		Context("LE operator", func() {
			It("Should return 1 when a <= b", func() {
				stage, err := std2.LEFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})
	})

	Describe("Arithmetic Operators", func() {
		Context("Add operator", func() {
			It("Should add two integers", func() {
				stage, err := std2.AddFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetInt32()).To(Equal(int32(30)))
				Expect(output.Type).To(Equal(ir.I32{}))
			})

			It("Should add two floats", func() {
				stage, err := std2.AddFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.F64{}}.PutFloat64(10.5)
				v2 := value.Value{Type: ir.F64{}}.PutFloat64(20.5)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetFloat64()).To(Equal(31.0))
				Expect(output.Type).To(Equal(ir.F64{}))
			})

			It("Should handle mixed types", func() {
				stage, err := std2.AddFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.F32{}}.PutFloat32(10.5)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetFloat32()).To(Equal(float32(30.5)))
				Expect(output.Type).To(Equal(ir.F32{}))
			})
		})

		Context("Sub operator", func() {
			It("Should subtract two integers", func() {
				stage, err := std2.SubFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(30)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetInt32()).To(Equal(int32(10)))
			})
		})

		Context("Mul operator", func() {
			It("Should multiply two integers", func() {
				stage, err := std2.MulFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(3)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetInt32()).To(Equal(int32(30)))
			})
		})

		Context("Div operator", func() {
			It("Should divide two integers", func() {
				stage, err := std2.DivFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(30)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(10)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetInt32()).To(Equal(int32(3)))
			})

			It("Should handle division by zero", func() {
				stage, err := std2.DivFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(30)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(0)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetInt32()).To(Equal(int32(0)))
			})
		})

		Context("Mod operator", func() {
			It("Should calculate modulo", func() {
				stage, err := std2.ModFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(3)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetInt32()).To(Equal(int32(1)))
			})
		})
	})

	Describe("State Management", func() {
		It("Should not reset state after outputting", func() {
			stage, err := std2.EQFactory(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			outputCount := 0
			stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
				outputCount++
			})

			v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
			v2 := value.Value{Type: ir.I32{}}.PutInt32(10)
			stage.Load("a", v1)
			stage.Load("b", v2)
			stage.Next(ctx)

			Expect(outputCount).To(Equal(1))

			v3 := value.Value{Type: ir.I32{}}.PutInt32(20)
			v4 := value.Value{Type: ir.I32{}}.PutInt32(20)
			stage.Load("a", v3)
			stage.Load("b", v4)
			stage.Next(ctx)

			Expect(outputCount).To(Equal(2))
		})

		It("Should only output when both values are present", func() {
			stage, err := std2.EQFactory(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			outputCalled := false
			stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
				outputCalled = true
			})

			v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
			stage.Load("a", v1)

			// Should not output yet
			Expect(outputCalled).To(BeFalse())

			v2 := value.Value{Type: ir.I32{}}.PutInt32(10)
			stage.Load("b", v2)
				stage.Next(ctx)

			// Now should output
			Expect(outputCalled).To(BeTrue())
		})
	})
})
