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

var _ = Describe("LogicalOp", func() {
	var (
		ctx context.Context
		cfg std2.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		cfg = std2.Config{
			Node: ir.Node{
				Key: "test_logical_op",
			},
		}
	})

	Describe("AND Operator", func() {
		Context("When both inputs are true", func() {
			It("Should return 1", func() {
				stage, err := std2.AndFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.U8{}}.PutUint8(1)
				v2 := value.Value{Type: ir.U8{}}.PutUint8(1)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})

		Context("When first input is false", func() {
			It("Should return 0", func() {
				stage, err := std2.AndFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.U8{}}.PutUint8(0)
				v2 := value.Value{Type: ir.U8{}}.PutUint8(1)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})

		Context("When second input is false", func() {
			It("Should return 0", func() {
				stage, err := std2.AndFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.U8{}}.PutUint8(1)
				v2 := value.Value{Type: ir.U8{}}.PutUint8(0)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})

		Context("When both inputs are false", func() {
			It("Should return 0", func() {
				stage, err := std2.AndFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.U8{}}.PutUint8(0)
				v2 := value.Value{Type: ir.U8{}}.PutUint8(0)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})
	})

	Describe("OR Operator", func() {
		Context("When both inputs are true", func() {
			It("Should return 1", func() {
				stage, err := std2.OrFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.U8{}}.PutUint8(1)
				v2 := value.Value{Type: ir.U8{}}.PutUint8(1)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})

		Context("When first input is true", func() {
			It("Should return 1", func() {
				stage, err := std2.OrFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.U8{}}.PutUint8(1)
				v2 := value.Value{Type: ir.U8{}}.PutUint8(0)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})

		Context("When second input is true", func() {
			It("Should return 1", func() {
				stage, err := std2.OrFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.U8{}}.PutUint8(0)
				v2 := value.Value{Type: ir.U8{}}.PutUint8(1)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})

		Context("When both inputs are false", func() {
			It("Should return 0", func() {
				stage, err := std2.OrFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v1 := value.Value{Type: ir.U8{}}.PutUint8(0)
				v2 := value.Value{Type: ir.U8{}}.PutUint8(0)

				stage.Load("a", v1)
				stage.Load("b", v2)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})
	})

	Describe("NOT Operator", func() {
		Context("When input is true", func() {
			It("Should return 0", func() {
				stage, err := std2.NotFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v := value.Value{Type: ir.U8{}}.PutUint8(1)

				stage.Load("input", v)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})

		Context("When input is false", func() {
			It("Should return 1", func() {
				stage, err := std2.NotFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v := value.Value{Type: ir.U8{}}.PutUint8(0)

				stage.Load("input", v)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})
		})

		Context("When input is non-zero", func() {
			It("Should return 0", func() {
				stage, err := std2.NotFactory(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				v := value.Value{Type: ir.U8{}}.PutUint8(42)

				stage.Load("input", v)
				stage.Next(ctx)

				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})
	})

	Describe("Combined Operations", func() {
		It("Should chain AND with comparison operators", func() {
			eqStage, err := std2.EQFactory(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			gtStage, err := std2.GTFactory(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			andStage, err := std2.AndFactory(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			var eqOutput, gtOutput, andOutput value.Value

			eqStage.OnOutput(func(_ context.Context, _ string, val value.Value) {
				eqOutput = val
				andStage.Load("a", val)
			})

			gtStage.OnOutput(func(_ context.Context, _ string, val value.Value) {
				gtOutput = val
				andStage.Load("b", val)
			})

			andStage.OnOutput(func(_ context.Context, _ string, val value.Value) {
				andOutput = val
			})

			// Test: 10 == 10 AND 20 > 10 => true AND true => true
			eqStage.Load("a", value.Value{Type: ir.I32{}}.PutInt32(10))
			eqStage.Load("b", value.Value{Type: ir.I32{}}.PutInt32(10))
			eqStage.Next(ctx)

			gtStage.Load("a", value.Value{Type: ir.I32{}}.PutInt32(20))
			gtStage.Load("b", value.Value{Type: ir.I32{}}.PutInt32(10))
			gtStage.Next(ctx)

			andStage.Next(ctx)

			Expect(eqOutput.GetUint8()).To(Equal(uint8(1)))
			Expect(gtOutput.GetUint8()).To(Equal(uint8(1)))
			Expect(andOutput.GetUint8()).To(Equal(uint8(1)))
		})

		It("Should handle NOT with comparison operators", func() {
			eqStage, err := std2.EQFactory(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			notStage, err := std2.NotFactory(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())

			var eqOutput, notOutput value.Value

			eqStage.OnOutput(func(_ context.Context, _ string, val value.Value) {
				eqOutput = val
				notStage.Load("input", val)
			})

			notStage.OnOutput(func(_ context.Context, _ string, val value.Value) {
				notOutput = val
			})

			// Test: NOT(10 == 20) => NOT(false) => true
			eqStage.Load("a", value.Value{Type: ir.I32{}}.PutInt32(10))
			eqStage.Load("b", value.Value{Type: ir.I32{}}.PutInt32(20))
			eqStage.Next(ctx)
			notStage.Next(ctx)

			Expect(eqOutput.GetUint8()).To(Equal(uint8(0)))
			Expect(notOutput.GetUint8()).To(Equal(uint8(1)))
		})
	})
})