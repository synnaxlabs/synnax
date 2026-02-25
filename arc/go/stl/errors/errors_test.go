// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/errors"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/tetratelabs/wazero/experimental/wazerotest"
)

var _ = Describe("errors", func() {
	var (
		ctx context.Context
		rt  *testutil.MockHostRuntime
		mod *errors.Module
	)

	BeforeEach(func() {
		ctx = context.Background()
		rt = testutil.NewMockHostRuntime()
		mod = errors.NewModule()
		Expect(mod.BindTo(rt)).To(Succeed())
	})

	Describe("panic", func() {
		It("Should panic with 'memory not set' when memory is nil", func() {
			panicFn := testutil.Get[func(context.Context, uint32, uint32)](rt, "error", "panic")
			Expect(func() { panicFn(ctx, 0, 5) }).To(PanicWith(
				ContainSubstring("memory not set"),
			))
		})

		It("Should panic with the message read from memory", func() {
			panicFn := testutil.Get[func(context.Context, uint32, uint32)](rt, "error", "panic")
			mem := wazerotest.NewMemory(1)
			mem.Write(0, []byte("test error"))
			mod.SetMemory(mem)
			Expect(func() { panicFn(ctx, 0, 10) }).To(PanicWith(
				Equal("arc panic: test error"),
			))
		})

		It("Should panic with 'message unreadable' when memory read fails", func() {
			panicFn := testutil.Get[func(context.Context, uint32, uint32)](rt, "error", "panic")
			mem := wazerotest.NewMemory(1)
			mod.SetMemory(mem)
			Expect(func() { panicFn(ctx, 100000, 5) }).To(PanicWith(
				Equal("arc panic (message unreadable)"),
			))
		})

		It("Should panic with empty message when length is zero", func() {
			panicFn := testutil.Get[func(context.Context, uint32, uint32)](rt, "error", "panic")
			mem := wazerotest.NewMemory(1)
			mod.SetMemory(mem)
			Expect(func() { panicFn(ctx, 0, 0) }).To(PanicWith(
				Equal("arc panic: "),
			))
		})
	})
})
