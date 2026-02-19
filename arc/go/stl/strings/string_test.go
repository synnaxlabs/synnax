// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package strings_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/tetratelabs/wazero/experimental/wazerotest"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

var _ = Describe("Strings", func() {
	var (
		rt  *testutil.MockHostRuntime
		ss  *state.StringHandleStore
		mod *strings.Module
	)

	BeforeEach(func() {
		rt = testutil.NewMockHostRuntime()
		ss = state.NewStringHandleStore()
		mod = strings.NewModule(ss)
		Expect(mod.BindTo(ctx, rt)).To(Succeed())
	})

	Describe("from_literal", func() {
		It("Should create a handle from WASM memory", func() {
			fromLiteral := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "from_literal")
			mem := wazerotest.NewMemory(1)
			mem.Write(0, []byte("hello"))
			mod.SetMemory(mem)
			h := fromLiteral(ctx, 0, 5)
			Expect(h).ToNot(BeZero())
			Expect(MustBeOk(ss.Get(h))).To(Equal("hello"))
		})

		It("Should return 0 when memory is nil", func() {
			fromLiteral := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "from_literal")
			Expect(fromLiteral(ctx, 0, 5)).To(Equal(uint32(0)))
		})

		It("Should return 0 when memory read fails", func() {
			fromLiteral := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "from_literal")
			mem := wazerotest.NewMemory(1)
			mod.SetMemory(mem)
			Expect(fromLiteral(ctx, 100000, 5)).To(Equal(uint32(0)))
		})

		It("Should handle empty string with length 0", func() {
			fromLiteral := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "from_literal")
			mem := wazerotest.NewMemory(1)
			mod.SetMemory(mem)
			h := fromLiteral(ctx, 0, 0)
			Expect(h).ToNot(BeZero())
			lenFn := testutil.Get[func(context.Context, uint32) uint64](rt, "string", "len")
			Expect(lenFn(ctx, h)).To(Equal(uint64(0)))
		})
	})

	Describe("concat", func() {
		It("Should concatenate two strings", func() {
			concat := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "concat")
			h1 := ss.Create("hello ")
			h2 := ss.Create("world")
			rh := concat(ctx, h1, h2)
			Expect(rh).ToNot(BeZero())
			Expect(MustBeOk(ss.Get(rh))).To(Equal("hello world"))
		})

		It("Should return 0 for invalid handles", func() {
			concat := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "concat")
			Expect(concat(ctx, 9999, 9998)).To(Equal(uint32(0)))
		})
	})

	Describe("equal", func() {
		It("Should return 1 for equal strings", func() {
			equal := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "equal")
			h1 := ss.Create("same")
			h2 := ss.Create("same")
			Expect(equal(ctx, h1, h2)).To(Equal(uint32(1)))
		})

		It("Should return 0 for different strings", func() {
			equal := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "equal")
			h1 := ss.Create("foo")
			h2 := ss.Create("bar")
			Expect(equal(ctx, h1, h2)).To(Equal(uint32(0)))
		})

		It("Should return 0 for invalid handles", func() {
			equal := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "equal")
			Expect(equal(ctx, 9999, 9998)).To(Equal(uint32(0)))
		})
	})

	Describe("len", func() {
		It("Should return byte length", func() {
			lenFn := testutil.Get[func(context.Context, uint32) uint64](rt, "string", "len")
			h := ss.Create("hello")
			Expect(lenFn(ctx, h)).To(Equal(uint64(5)))
		})

		It("Should return 0 for invalid handle", func() {
			lenFn := testutil.Get[func(context.Context, uint32) uint64](rt, "string", "len")
			Expect(lenFn(ctx, 9999)).To(Equal(uint64(0)))
		})
	})

	Describe("cross-function handle reuse", func() {
		It("Should use from_literal result in concat and verify with equal", func() {
			fromLiteral := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "from_literal")
			concat := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "concat")
			equal := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "string", "equal")

			mem := wazerotest.NewMemory(1)
			mem.Write(0, []byte("helloworld"))
			mod.SetMemory(mem)
			h1 := fromLiteral(ctx, 0, 5)
			h2 := fromLiteral(ctx, 5, 5)
			result := concat(ctx, h1, h2)
			expected := ss.Create("helloworld")
			Expect(equal(ctx, result, expected)).To(Equal(uint32(1)))
		})
	})
})
