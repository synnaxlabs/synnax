// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Down sample", func() {
	It("Should downsample values correctly", func() {
		ctx, cancel := signal.WithCancel(context.Background())
		defer cancel()

		inlet := NewStream[[]int](4)
		outlet := NewStream[[]int](3)

		downsampler := DownSampler[[]int, string]{
			DownSample: func(ctx context.Context, x []int, factors map[string]int) []int {
				// We simulate downsampling by just slicing the array based on a factor.
				// For the sake of this test, we assume "factor1" is the key we care about.
				if factor, ok := factors["factor1"]; ok && factor < len(x) {
					return x[:factor]
				}
				return x
			},
			DownSamplingFactors: map[string]int{
				"factor1": 3, // downsample to the first 3 elements
			},
		}

		downsampler.InFrom(inlet)
		downsampler.OutTo(outlet)
		downsampler.Flow(ctx)

		inlet.Inlet() <- []int{1, 2, 3, 4, 5, 6, 7}
		Expect(<-outlet.Outlet()).To(Equal([]int{1, 2, 3}))
		cancel()
		Expect(errors.Is(ctx.Wait(), context.Canceled)).To(BeTrue())
	})
})
