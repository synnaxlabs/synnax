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
	sCtx.Go(func(context.Context) error {
		for v := range s.In.Outlet() {
			if v == 1 {
				panic("got 1")
			}
		}
		return nil
	}, o.Signal...)
}

var _ = Describe("Confluence", func() {
	Describe("EmptyFlow", func() {
		It("Should do nothing", func() {
			ctx, cancel := signal.Isolated()
			defer cancel()
			Expect(func() { NopFlow{}.Flow(ctx) }).ToNot(Panic())
		})
	})
	Describe("Options", func() {
		It("Should not close inlet on panic", func() {
			ctx, _ := signal.Isolated()
			i := NewStream[int]()
			o := NewStream[int]()
			var s seg
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
			i := NewStream[int]()
			o := NewStream[int]()
			var s seg
			s.InFrom(i)
			s.OutTo(o)
			s.Flow(ctx, CloseOutputInletsOnExit(), RecoverWithErrOnPanic())
			i.Inlet() <- 1
			_, ok := <-o.Outlet()
			Expect(ok).To(BeFalse())
		})
		It("Should still run deferred methods after panic", func() {
			ctx, _ := signal.Isolated()
			i := NewStream[int]()
			o := NewStream[int]()
			var s seg
			s.InFrom(i)
			s.OutTo(o)
			var a = atomic.Int32Counter{}
			s.Flow(
				ctx,
				CloseOutputInletsOnExit(),
				RecoverWithErrOnPanic(),
				Defer(func() { a.Add(10) }),
			)
			i.Inlet() <- 1
			_, ok := <-o.Outlet()
			Expect(ok).To(BeFalse())
			Expect(a.Value()).To(BeEquivalentTo(10))
		})
	})
	Describe("Drain", func() {
		It("Should drain an outlet of values until it is closed", func() {
			c := NewStream[int](10)
			go func() {
				for range 10 {
					c.Inlet() <- 1
				}
				c.Close()
			}()
			Drain(c)
			Expect(c.Outlet()).To(BeClosed())
		})
	})
})
