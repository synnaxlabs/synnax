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
	"github.com/synnaxlabs/x/atomic"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type seg struct {
	UnarySink[int]
	AbstractUnarySource[int]
}

func (s seg) Flow(sCtx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(s.Out)
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case v := <-s.In.Outlet():
				if v == 1 {
					panic("got 1")
				}
			}
		}
	}, o.Signal...)
}

var _ = Describe("Confluence", func() {

	Describe("EmptyFlow", func() {

		It("Should do nothing", func() {
			ctx, cancel := signal.Isolated()
			defer cancel()
			Expect(func() {
				NopFlow{}.Flow(ctx)
			}).ToNot(Panic())
		})

	})

	Describe("Options", func() {
		It("Should not close inlet on panic", func() {
			ctx, _ := signal.Isolated()

			var s seg
			i := NewStream[int]()
			o := NewStream[int]()
			s.InFrom(i)
			s.OutTo(o)
			s.Flow(ctx, CloseOutputInletsOnExit(), WithRetryOnPanic(1))

			// this panics
			i.Inlet() <- 1
			// this does not panic
			Expect(func() { i.Inlet() <- 2 }).ToNot(Panic())
			// this panics again, which makes the segment exit with an error
			Expect(func() { i.Inlet() <- 1 }).ToNot(Panic())
			Expect(ctx.Wait()).To(MatchError(ContainSubstring("got 1")))
			_, ok := <-o.Outlet()
			Expect(ok).To(BeFalse())
		})

		It("Should close inlet on a panic-recovered error", func() {
			ctx, _ := signal.Isolated()

			var s seg
			i := NewStream[int]()
			o := NewStream[int]()
			s.InFrom(i)
			s.OutTo(o)
			s.Flow(ctx, CloseOutputInletsOnExit(), RecoverWithErrOnPanic())

			i.Inlet() <- 1
			_, ok := <-o.Outlet()
			Expect(ok).To(BeFalse())
		})

		It("Should still run deferred methods after panic", func() {
			ctx, _ := signal.Isolated()

			var (
				s seg
				a = atomic.Int32Counter{}
			)
			i := NewStream[int]()
			o := NewStream[int]()
			s.InFrom(i)
			s.OutTo(o)
			s.Flow(ctx, CloseOutputInletsOnExit(), RecoverWithErrOnPanic(), Defer(func() {
				a.Add(10)
			}))

			i.Inlet() <- 1
			_, ok := <-o.Outlet()
			Expect(ok).To(BeFalse())
			Expect(a.Value()).To(Equal(int32(10)))
		})
	})
})
