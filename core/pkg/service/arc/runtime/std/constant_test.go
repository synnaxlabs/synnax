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

var _ = Describe("Constant", func() {
	var (
		ctx context.Context
		cfg std.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		cfg = std.Config{
			Node: ir.Node{
				Key:  "test_constant",
				Type: "constant",
			},
		}
	})

	Describe("Constant Stage", func() {
		Context("Integer constant", func() {
			It("Should output an integer constant value", func() {
				cfg.Node.Config = map[string]any{
					"value": int32(42),
				}

				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				stage.Flow(sCtx)

				Expect(output.GetInt32()).To(Equal(int32(42)))
			})
		})

		Context("Float constant", func() {
			It("Should output a float constant value", func() {
				cfg.Node.Config = map[string]any{
					"value": float64(3.14159),
				}

				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				stage.Flow(sCtx)

				Expect(output.GetFloat64()).To(Equal(3.14159))
			})
		})

		Context("Unsigned integer constant", func() {
			It("Should output an unsigned integer constant", func() {
				cfg.Node.Config = map[string]any{
					"value": uint64(100),
				}

				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				stage.Flow(sCtx)

				Expect(output.GetUint64()).To(Equal(uint64(100)))
			})
		})

		Context("Boolean constant", func() {
			It("Should output true as 1", func() {
				cfg.Node.Config = map[string]any{
					"value": true,
				}

				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				stage.Flow(sCtx)

				Expect(output.GetUint8()).To(Equal(uint8(1)))
			})

			It("Should output false as 0", func() {
				cfg.Node.Config = map[string]any{
					"value": false,
				}

				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
				})

				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				stage.Flow(sCtx)

				Expect(output.GetUint8()).To(Equal(uint8(0)))
			})
		})

		Context("No value configured", func() {
			It("Should output a zero value when no value is configured", func() {
				cfg.Node.Config = map[string]any{}

				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				var output value.Value
				outputCalled := false
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					output = val
					outputCalled = true
				})

				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				stage.Flow(sCtx)

				Expect(outputCalled).To(BeTrue())
				Expect(output.GetUint64()).To(Equal(uint64(0)))
			})
		})

		Context("Multiple calls to Flow", func() {
			It("Should output the same constant value on each Flow call", func() {
				cfg.Node.Config = map[string]any{
					"value": int32(7),
				}

				stage, err := std.Create(ctx, cfg)
				Expect(err).ToNot(HaveOccurred())

				outputs := []value.Value{}
				stage.OnOutput(func(_ context.Context, _ string, val value.Value) {
					outputs = append(outputs, val)
				})

				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()

				stage.Flow(sCtx)
				stage.Flow(sCtx)
				stage.Flow(sCtx)

				Expect(outputs).To(HaveLen(3))
				for _, output := range outputs {
					Expect(output.GetInt32()).To(Equal(int32(7)))
				}
			})
		})
	})

	Describe("Integration with operators", func() {
		It("Should work as input to comparison operators", func() {
			constCfg := cfg
			constCfg.Node.Config = map[string]any{
				"value": int32(10),
			}
			constStage, err := std.Create(ctx, constCfg)
			Expect(err).ToNot(HaveOccurred())

			eqCfg := std.Config{
				Node: ir.Node{
					Key:  "test_eq",
					Type: "eq",
				},
			}
			eqStage, err := std.Create(ctx, eqCfg)
			Expect(err).ToNot(HaveOccurred())

			constStage.OnOutput(func(ctx context.Context, param string, val value.Value) {
				eqStage.Next(ctx, "a", val)
			})

			var eqOutput value.Value
			eqStage.OnOutput(func(_ context.Context, _ string, val value.Value) {
				eqOutput = val
			})

			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			constStage.Flow(sCtx)

			v2 := value.Value{Type: ir.I32{}}.PutInt32(10)
			eqStage.Next(ctx, "b", v2)

			Expect(eqOutput.GetUint8()).To(Equal(uint8(1)))
		})
	})
})
