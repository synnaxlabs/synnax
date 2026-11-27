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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/testutil"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Attach", func() {
	It("Should feed the segment from the inlet and its output to the outlet", func(
		ctx SpecContext,
	) {
		double := &confluence.LinearTransform[int, int]{}
		double.Transform = func(_ context.Context, v int) (int, bool, error) {
			return 2 * v, true, nil
		}
		in, out := testutil.Attach[int, int](double, 1)
		sCtx, cancel := signal.WithCancel(ctx)
		double.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		in.Inlet() <- 3
		Eventually(out.Outlet()).Should(Receive(Equal(6)))
		in.Close()
		Eventually(out.Outlet()).Should(BeClosed())
		cancel()
		Expect(sCtx.Wait()).To(Succeed())
	})

	DescribeTable("Should give both streams the buffer", func(
		buffer []int,
		want int,
	) {
		in, out := testutil.Attach[int, int](
			&confluence.LinearTransform[int, int]{},
			buffer...,
		)
		Expect(cap(in.Inlet())).To(Equal(want))
		Expect(cap(out.Outlet())).To(Equal(want))
	},
		Entry("of the default size", nil, cap(confluence.NewStream[int]().Inlet())),
		Entry("of a given size", []int{7}, 7),
	)
})
