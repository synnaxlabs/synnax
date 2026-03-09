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
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero/experimental/wazerotest"
)

var ctx = context.Background()

var _ = Describe("Strings", func() {
	var (
		rt  *testutil.Runtime
		ss  *strings.State
		mem *wazerotest.Memory
	)

	call := func(fn string, args ...uint64) []uint64 {
		return rt.Call(ctx, "string", fn, args...)
	}
	callU32 := func(fn string, args ...uint64) uint32 {
		return testutil.AsU32(call(fn, args...)[0])
	}
	callU64 := func(fn string, args ...uint64) uint64 {
		return call(fn, args...)[0]
	}

	BeforeEach(func() {
		ctx = context.Background()
		rt = testutil.NewRuntime(ctx)
		ss = strings.NewState()
		mem = wazerotest.NewMemory(1)
		_, err := strings.NewModule(ctx, ss, rt.Underlying(), mem)
		Expect(err).ToNot(HaveOccurred())
		rt.Passthrough(ctx, "string")
	})

	AfterEach(func() {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	Describe("from_literal", func() {
		It("Should create a handle from WASM memory", func() {
			mem.Write(0, []byte("hello"))
			h := callU32("from_literal", testutil.U32(0), testutil.U32(5))
			Expect(h).ToNot(BeZero())
			Expect(MustBeOk(ss.Get(h))).To(Equal("hello"))
		})

		It("Should return 0 when memory read fails", func() {
			Expect(callU32("from_literal", testutil.U32(100000), testutil.U32(5))).To(Equal(uint32(0)))
		})

		It("Should handle empty string with length 0", func() {
			h := callU32("from_literal", testutil.U32(0), testutil.U32(0))
			Expect(h).ToNot(BeZero())
			Expect(callU64("len", testutil.U32(h))).To(Equal(uint64(0)))
		})
	})

	Describe("from_literal with nil memory", func() {
		It("Should return 0 when memory is nil", func() {
			rt2 := testutil.NewRuntime(ctx)
			defer rt2.Close(ctx)
			ss2 := strings.NewState()
			_, err := strings.NewModule(ctx, ss2, rt2.Underlying(), nil)
			Expect(err).ToNot(HaveOccurred())
			rt2.Passthrough(ctx, "string")
			res := rt2.Call(ctx, "string", "from_literal", testutil.U32(0), testutil.U32(5))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(0)))
		})
	})

	Describe("concat", func() {
		It("Should concatenate two strings", func() {
			h1 := ss.Create("hello ")
			h2 := ss.Create("world")
			rh := callU32("concat", testutil.U32(h1), testutil.U32(h2))
			Expect(rh).ToNot(BeZero())
			Expect(MustBeOk(ss.Get(rh))).To(Equal("hello world"))
		})

		It("Should return 0 for invalid handles", func() {
			Expect(callU32("concat", testutil.U32(9999), testutil.U32(9998))).To(Equal(uint32(0)))
		})
	})

	Describe("equal", func() {
		It("Should return 1 for equal strings", func() {
			h1 := ss.Create("same")
			h2 := ss.Create("same")
			Expect(callU32("equal", testutil.U32(h1), testutil.U32(h2))).To(Equal(uint32(1)))
		})

		It("Should return 0 for different strings", func() {
			h1 := ss.Create("foo")
			h2 := ss.Create("bar")
			Expect(callU32("equal", testutil.U32(h1), testutil.U32(h2))).To(Equal(uint32(0)))
		})

		It("Should return 0 for invalid handles", func() {
			Expect(callU32("equal", testutil.U32(9999), testutil.U32(9998))).To(Equal(uint32(0)))
		})
	})

	Describe("len", func() {
		It("Should return byte length", func() {
			h := ss.Create("hello")
			Expect(callU64("len", testutil.U32(h))).To(Equal(uint64(5)))
		})

		It("Should return 0 for invalid handle", func() {
			Expect(callU64("len", testutil.U32(9999))).To(Equal(uint64(0)))
		})
	})

	Describe("cross-function handle reuse", func() {
		It("Should use from_literal result in concat and verify with equal", func() {
			mem.Write(0, []byte("helloworld"))
			h1 := callU32("from_literal", testutil.U32(0), testutil.U32(5))
			h2 := callU32("from_literal", testutil.U32(5), testutil.U32(5))
			result := callU32("concat", testutil.U32(h1), testutil.U32(h2))
			expected := ss.Create("helloworld")
			Expect(callU32("equal", testutil.U32(result), testutil.U32(expected))).To(Equal(uint32(1)))
		})
	})
})
