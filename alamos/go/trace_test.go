// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	. "github.com/synnaxlabs/x/testutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var _ = Describe("Trace", func() {
	Describe("NewTracer", func() {
		It("Should correctly create a new tracer", func() {
			cfg := alamos.TracingConfig{
				Provider:   otel.GetTracerProvider(),
				Propagator: otel.GetTextMapPropagator(),
			}
			tracer := MustSucceed(alamos.NewTracer(cfg))
			Expect(tracer).ToNot(BeNil())
		})
	})
	Describe("No-op", func() {
		It("Should not panic when calling methods on a nil tracer", func() {
			var tracer *alamos.Tracer
			Expect(func() {
				_, sp := tracer.Trace(context.Background(), "test", alamos.InfoLevel)
				sp.End()
			}).ToNot(Panic())
		})
	})
	Describe("Transfer", func() {
		It("Should correctly transfer the span from one context to another", func() {
			tracer := MustSucceed(alamos.NewTracer(alamos.TracingConfig{
				Provider:   otel.GetTracerProvider(),
				Propagator: otel.GetTextMapPropagator(),
			}))
			ctx, sp := tracer.Trace(context.Background(), "test", alamos.InfoLevel)
			sp1 := trace.SpanFromContext(ctx)
			ctx2 := tracer.Transfer(ctx, context.Background())
			sp2 := trace.SpanFromContext(ctx2)
			Expect(sp1).To(BeIdenticalTo(sp2))
			sp.End()
		})
	})
})
