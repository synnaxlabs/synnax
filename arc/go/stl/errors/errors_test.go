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
)

var ctx = context.Background()

var _ = Describe("errors", func() {
	var (
		rt  *testutil.MockHostRuntime
		mod *errors.Module
	)

	BeforeEach(func() {
		rt = testutil.NewMockHostRuntime()
		mod = errors.NewModule()
		Expect(mod.BindTo(ctx, rt)).To(Succeed())
	})

	Describe("panic", func() {
		It("Should panic with 'memory not set' when memory is nil", func() {
			panicFn := testutil.Get[func(context.Context, uint32, uint32)](rt, "error", "panic")
			Expect(func() { panicFn(ctx, 0, 5) }).To(PanicWith(
				ContainSubstring("memory not set"),
			))
		})
	})
})
