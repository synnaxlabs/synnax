// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package recovery_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/recovery"
	. "github.com/synnaxlabs/x/testutil"
)

func newCollector() freighter.MiddlewareCollector {
	var mc freighter.MiddlewareCollector
	mc.Use(recovery.Middleware(alamos.Instrumentation{}))
	return mc
}

func newContext() freighter.Context {
	return freighter.Context{
		Context:  context.Background(),
		Protocol: "test",
		Target:   "test-target",
		Params:   make(freighter.Params),
	}
}

var _ = Describe("Recovery", func() {
	Describe("Middleware", func() {
		It("should recover a panic in the finalizer and return ErrPanic", func() {
			mc := newCollector()
			oCtx, err := mc.Exec(newContext(), freighter.FinalizerFunc(
				func(freighter.Context) (freighter.Context, error) {
					panic("boom in handler")
				},
			))
			Expect(err).To(MatchError(recovery.ErrPanic))
			Expect(oCtx.Protocol).To(Equal("test"))
		})

		It("should recover a panic raised by a downstream middleware", func() {
			mc := newCollector()
			mc.Use(freighter.MiddlewareFunc(func(
				freighter.Context,
				freighter.Next,
			) (freighter.Context, error) {
				panic("boom in middleware")
			}))
			_, err := mc.Exec(newContext(), freighter.NopFinalizer)
			Expect(err).To(MatchError(recovery.ErrPanic))
		})

		It("should not leak the panic value into the returned error", func() {
			mc := newCollector()
			_, err := mc.Exec(newContext(), freighter.FinalizerFunc(
				func(freighter.Context) (freighter.Context, error) {
					panic("super-secret internal detail")
				},
			))
			Expect(err).ToNot(MatchError(ContainSubstring("super-secret")))
		})

		It("should pass through unmodified when no panic occurs", func() {
			mc := newCollector()
			oCtx := MustSucceed(mc.Exec(newContext(), freighter.FinalizerFunc(
				func(c freighter.Context) (freighter.Context, error) {
					c.Set("handled", true)
					return c, nil
				},
			)))
			Expect(oCtx.GetDefault("handled", false)).To(BeTrue())
		})
	})

	Describe("LogPanic", func() {
		It("should be a no-op when the instrumentation has no logger", func() {
			Expect(func() {
				recovery.LogPanic(
					alamos.Instrumentation{},
					"target",
					"boom",
					[]byte("stack"),
				)
			}).ToNot(Panic())
		})
	})
})
