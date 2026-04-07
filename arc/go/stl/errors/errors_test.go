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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/errors"
	"github.com/synnaxlabs/arc/stl/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero/experimental/wazerotest"
)

var _ = Describe("errors", func() {
	var (
		rt  *testutil.Runtime
		mod *errors.Module
	)

	BeforeEach(func(ctx SpecContext) {
		rt = testutil.NewRuntime(ctx)
		mod = MustSucceed(errors.NewModule(ctx, nil, rt.Underlying()))
		rt.Passthrough(ctx, "error")
	})

	AfterEach(func(ctx SpecContext) {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	Describe("panic", func() {
		It("Should panic with 'memory not set' when memory is nil", func(ctx SpecContext) {
			Expect(func() {
				rt.Call(ctx, "error", "panic", testutil.U32(0), testutil.U32(5))
			}).To(PanicWith(ContainSubstring("memory not set")))
		})

		It("Should panic with the message read from memory", func(ctx SpecContext) {
			mem := wazerotest.NewMemory(1)
			mem.Write(0, []byte("test error"))
			mod.SetMemory(mem)
			Expect(func() {
				rt.Call(ctx, "error", "panic", testutil.U32(0), testutil.U32(10))
			}).To(PanicWith(ContainSubstring("arc panic: test error")))
		})

		It("Should panic with 'message unreadable' when memory read fails", func(ctx SpecContext) {
			mem := wazerotest.NewMemory(1)
			mod.SetMemory(mem)
			Expect(func() {
				rt.Call(ctx, "error", "panic", testutil.U32(100000), testutil.U32(5))
			}).To(PanicWith(ContainSubstring("arc panic (message unreadable)")))
		})

		It("Should panic with empty message when length is zero", func(ctx SpecContext) {
			mem := wazerotest.NewMemory(1)
			mod.SetMemory(mem)
			Expect(func() {
				rt.Call(ctx, "error", "panic", testutil.U32(0), testutil.U32(0))
			}).To(PanicWith(ContainSubstring("arc panic: ")))
		})
	})
})
