// Copyright 2026 Synnax Labs, Inc.
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

var _ = Describe("transform", func() {
	It("Should transform values correctly", func() {
		inlet := NewStream[int](3)
		outlet := NewStream[int](4)
		square := &LinearTransform[int, int]{}
		square.Transform = func(ctx context.Context, i int) (int, bool, error) {
			return i * i, true, nil
		}
		square.InFrom(inlet)
		square.OutTo(outlet)
		ctx, cancel := signal.WithCancel(context.Background())
		square.Flow(ctx)
		inlet.Inlet() <- 1
		inlet.Inlet() <- 2
		Expect(<-outlet.Outlet()).To(Equal(1))
		Expect(<-outlet.Outlet()).To(Equal(4))
		cancel()
		Expect(errors.Is(ctx.Wait(), context.Canceled)).To(BeTrue())
	})
})
