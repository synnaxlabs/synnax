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
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Linear", func() {
	It("Should transform values from inlet to outlet", func() {
		i := NewStream[int](1)
		o := NewStream[int](1)
		s := LinearTransform[int, int]{}
		s.Transform = func(ctx context.Context, i int) (int, bool, error) {
			return i * i, true, nil
		}
		s.InFrom(i)
		s.OutTo(o)
		ctx, cancel := signal.Isolated()
		defer cancel()
		s.Flow(ctx)
		i.Inlet() <- 3
		Expect(<-o.Outlet()).To(Equal(9))
	})
	It("Should not send a value if the transform returns false", func() {
		i := NewStream[int](1)
		o := NewStream[int](1)
		s := LinearTransform[int, int]{}
		s.Transform = func(ctx context.Context, i int) (int, bool, error) {
			return 0, false, nil
		}
		s.InFrom(i)
		s.OutTo(o)
		ctx, cancel := signal.Isolated()
		defer cancel()
		s.Flow(ctx, CloseOutputInletsOnExit())
		i.Inlet() <- 3
		i.Close()
		_, ok := <-o.Outlet()
		Expect(ok).To(BeFalse())
	})
})
