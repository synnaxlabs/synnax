// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math_test

import (
	"context"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/testutil"
)

var ctx = context.Background()

var _ = Describe("Math", func() {
	var (
		rt  *testutil.MockHostRuntime
		mod *stlmath.Module
	)

	BeforeEach(func() {
		rt = testutil.NewMockHostRuntime()
		mod = stlmath.NewModule()
		Expect(mod.BindTo(ctx, rt)).To(Succeed())
	})

	Describe("pow", func() {
		It("Should compute i32 power", func() {
			pow := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "math", "pow_i32")
			Expect(pow(ctx, 3, 2)).To(Equal(uint32(9)))
			Expect(pow(ctx, 2, 10)).To(Equal(uint32(1024)))
		})

		It("Should compute u64 power", func() {
			pow := testutil.Get[func(context.Context, uint64, uint64) uint64](rt, "math", "pow_u64")
			Expect(pow(ctx, 2, 10)).To(Equal(uint64(1024)))
		})

		It("Should compute f32 power", func() {
			pow := testutil.Get[func(context.Context, float32, float32) float32](rt, "math", "pow_f32")
			Expect(pow(ctx, 2.0, 3.0)).To(BeNumerically("~", 8.0, 0.001))
		})

		It("Should compute f64 power", func() {
			pow := testutil.Get[func(context.Context, float64, float64) float64](rt, "math", "pow_f64")
			Expect(pow(ctx, 2.0, 0.5)).To(BeNumerically("~", math.Sqrt2, 0.0001))
		})
	})
})
