// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
)

type transport struct {
	freighter.Reporter
	middleware []freighter.Middleware
}

var _ freighter.Transport = (*transport)(nil)

func newTransport() *transport {
	return &transport{middleware: make([]freighter.Middleware, 0)}
}

func (t *transport) Use(middleware ...freighter.Middleware) {
	for _, m := range middleware {
		t.middleware = append(t.middleware, m)
	}
}

var _ = Describe("Middleware", func() {
	Describe("ExtractContext", func() {
		It("should panic if the context is not a freighter context", func() {
			ctx := context.Background()
			Expect(func() { freighter.ExtractContext(ctx) }).To(Panic())
		})
		It("should return the context if it is a freighter context", func() {
			ctx := freighter.Context{}
			Expect(freighter.ExtractContext(ctx)).To(Equal(ctx))
		})
	})
	Describe("UseOnAll", func() {
		It("should use the middleware on all the transports", func() {
			transport1 := newTransport()
			transport2 := newTransport()
			transports := []*transport{transport1, transport2}
			var (
				middleware1 *freighter.MiddlewareCollector
				middleware2 *freighter.MiddlewareFunc
			)
			freighter.UseOnAll(
				[]freighter.Middleware{middleware1, middleware2},
				transport1,
				transport2,
			)
			for _, t := range transports {
				Expect(t.middleware).To(HaveLen(2))
				Expect(t.middleware).To(ContainElement(middleware1))
				Expect(t.middleware).To(ContainElement(middleware2))
			}
		})
		It("should work if no transports are provided", func() {
			var middleware freighter.Middleware
			freighter.UseOnAll([]freighter.Middleware{middleware})
		})
		It("should work if no middleware is provided", func() {
			transport := newTransport()
			freighter.UseOnAll([]freighter.Middleware{}, transport)
			Expect(transport.middleware).To(HaveLen(0))
		})
	})
	Describe("NoopMiddlewareHandler", func() {
		It("should return the context unchanged", func() {
			ctx := freighter.Context{Target: "test"}
			Expect(freighter.NoopMiddlewareHandler(ctx)).To(Equal(ctx))
		})
	})
})
