// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/testutil"
)

var _ = Describe("Testutil", func() {
	var (
		rt  *testutil.Runtime
	)

	BeforeEach(func(ctx SpecContext) {
		rt = testutil.NewRuntime(ctx)
	})

	AfterEach(func(ctx SpecContext) {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	Describe("Passthrough", func() {
		It("Should panic when the host module is not instantiated", func(ctx SpecContext) {
			Expect(func() {
				rt.Passthrough(ctx, "nonexistent")
			}).To(PanicWith(ContainSubstring("not instantiated")))
		})
	})

	Describe("Call", func() {
		It("Should panic when no passthrough has been created", func(ctx SpecContext) {
			Expect(func() {
				rt.Call(ctx, "nonexistent", "fn")
			}).To(PanicWith(ContainSubstring("no passthrough")))
		})
	})

	Describe("Argument Encoding", func() {
		It("Should encode U32 values", func(ctx SpecContext) {
			Expect(testutil.U32(42)).To(Equal(uint64(42)))
		})

		It("Should encode U64 values", func(ctx SpecContext) {
			Expect(testutil.U64(math.MaxUint64)).To(Equal(uint64(math.MaxUint64)))
		})

		It("Should encode I32 values", func(ctx SpecContext) {
			Expect(testutil.I32(-1)).To(Equal(uint64(math.MaxUint32)))
		})

		It("Should encode I64 values", func(ctx SpecContext) {
			Expect(testutil.I64(-1)).To(Equal(uint64(math.MaxUint64)))
		})

		It("Should encode F32 values", func(ctx SpecContext) {
			bits := testutil.F32(1.5)
			Expect(bits).To(Equal(uint64(math.Float32bits(1.5))))
		})

		It("Should encode F64 values", func(ctx SpecContext) {
			bits := testutil.F64(3.14)
			Expect(bits).To(Equal(math.Float64bits(3.14)))
		})
	})

	Describe("Result Decoding", func() {
		It("Should decode U32 values", func(ctx SpecContext) {
			Expect(testutil.AsU32(uint64(42))).To(Equal(uint32(42)))
		})

		It("Should decode U64 values", func(ctx SpecContext) {
			Expect(testutil.AsU64(uint64(99))).To(Equal(uint64(99)))
		})

		It("Should decode F32 values", func(ctx SpecContext) {
			encoded := uint64(math.Float32bits(2.5))
			Expect(testutil.AsF32(encoded)).To(Equal(float32(2.5)))
		})

		It("Should decode F64 values", func(ctx SpecContext) {
			encoded := math.Float64bits(3.14)
			Expect(testutil.AsF64(encoded)).To(Equal(3.14))
		})
	})
})
