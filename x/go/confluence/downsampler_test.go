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
	FIt("Should downsample values correctly", func() { // TODO: remove F for focus
		ctx, cancel := signal.WithCancel(context.Background())
		inlet := NewStream[[]int](4)
		outlet := NewStream[[]int](3)
		downsampler := DownSampler[[]int]{
			DownSample: func(ctx context.Context, x []int) []int {
				return x[:3]
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
