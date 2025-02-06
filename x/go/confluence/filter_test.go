// Copyright 2025 Synnax Labs, Inc.
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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Filter", func() {
	Describe("OutTo", func() {
		It("Should panic if more than two inlets are provided", func() {
			s1, s2, s3 := NewStream[int](3), NewStream[int](3), NewStream[int](3)
			filter := Filter[int]{}
			Expect(func() {
				filter.OutTo(s1, s2, s3)
			}).To(Panic())
		})
		It("Should assign the first inlet to accepted and the second inlet to rejected", func() {
			accepted, rejected := NewStream[int](3), NewStream[int](3)
			filter := Filter[int]{}
			filter.OutTo(accepted, rejected)
			Expect(filter.Rejects).To(Equal(rejected))
		})
	})
	It("Should filter values correctly", func() {
		ctx, cancel := signal.WithCancel(context.Background())
		inlet := NewStream[int](3)
		outlet := NewStream[int](3)
		filter := Filter[int]{
			Filter: func(ctx context.Context, x int) (bool, error) {
				return x%3 == 0, nil
			},
		}
		filter.InFrom(inlet)
		filter.OutTo(outlet)
		filter.Flow(ctx)
		inlet.Inlet() <- 1
		inlet.Inlet() <- 2
		inlet.Inlet() <- 3
		Expect(<-outlet.Outlet()).To(Equal(3))
		cancel()
		Expect(errors.Is(ctx.Wait(), context.Canceled)).To(BeTrue())
	})
	It("Should send to rejects if the filter returns false", func() {
		ctx, cancel := signal.WithCancel(context.Background())
		inlet := NewStream[int](3)
		outlet := NewStream[int](3)
		rejects := NewStream[int](3)
		filter := Filter[int]{
			Filter: func(ctx context.Context, x int) (bool, error) {
				return x%3 == 0, nil
			},
		}
		filter.InFrom(inlet)
		filter.OutTo(outlet, rejects)
		filter.Flow(ctx)
		inlet.Inlet() <- 1
		inlet.Inlet() <- 2
		inlet.Inlet() <- 3
		Expect(<-outlet.Outlet()).To(Equal(3))
		Expect(<-rejects.Outlet()).To(Equal(1))
		Expect(<-rejects.Outlet()).To(Equal(2))
		cancel()
		Expect(errors.Is(ctx.Wait(), context.Canceled)).To(BeTrue())
	})
	It("Should exit if the filter returns an error", func() {
		ctx, cancel := signal.WithCancel(context.Background())
		defer cancel()
		inlet := NewStream[int](3)
		outlet := NewStream[int](3)
		filter := Filter[int]{
			Filter: func(ctx context.Context, x int) (bool, error) {
				return x%3 == 0, errors.New("error")
			},
		}
		filter.InFrom(inlet)
		filter.OutTo(outlet)
		filter.Flow(ctx)
		inlet.Inlet() <- 1
		Expect(ctx.Wait()).To(HaveOccurredAs(errors.New("error")))
	})
})
