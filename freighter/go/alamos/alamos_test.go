// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xalamos "github.com/synnaxlabs/alamos"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/alamos"
	. "github.com/synnaxlabs/x/testutil"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap/zapcore"
)

var finalizer = freighter.FinalizerFunc(
	func(ctx freighter.Context) (freighter.Context, error) { return ctx, nil },
)

var _ = Describe("Falamos", func() {
	Describe("Name", func() {
		It("Should correctly attach tracing metadata", func(ctx SpecContext) {
			ins := Instrumentation("falamos", InstrumentationConfig{Trace: new(true)})
			clientMw := MustSucceed(alamos.Middleware(alamos.Config{
				Instrumentation: ins,
			}))
			oCtx := MustSucceed(clientMw.Exec(
				freighter.Context{
					Context: ctx,
					Role:    freighter.RoleClient,
					Params:  make(freighter.Params),
				},
				finalizer,
			))
			_, ok := oCtx.Get("alamos-traceparent")
			Expect(ok).To(BeTrue())

			serverMw := MustSucceed(alamos.Middleware(alamos.Config{
				Instrumentation: ins,
			}))
			oCtx = MustSucceed(serverMw.Exec(
				freighter.Context{
					Context: ctx,
					Role:    freighter.RoleServer,
					Params:  oCtx.Params,
				},
				finalizer,
			))
			_, ok = oCtx.Get("alamos-traceparent")
			Expect(ok).To(BeTrue())
		})
	})

	Describe("Config", func() {
		It("Should not propagate traces when propagation is disabled", func(
			ctx SpecContext,
		) {
			ins := Instrumentation("falamos", InstrumentationConfig{Trace: new(true)})
			mw := MustSucceed(alamos.Middleware(alamos.Config{
				Instrumentation:   ins,
				EnablePropagation: new(false),
			}))
			oCtx := MustSucceed(mw.Exec(
				freighter.Context{
					Context: ctx,
					Role:    freighter.RoleClient,
					Params:  make(freighter.Params),
				},
				finalizer,
			))
			Expect(oCtx.Params).ToNot(HaveKey("alamos-traceparent"))
		})

		It("Should log requests by default", func(ctx SpecContext) {
			ins, logs := ObservedInstrumentation(zapcore.DebugLevel)
			mw := MustSucceed(alamos.Middleware(alamos.Config{Instrumentation: ins}))
			MustSucceed(mw.Exec(freighter.Context{Context: ctx}, finalizer))
			Expect(logs.Len()).To(Equal(1))
		})

		It("Should log requests when propagation is disabled", func(ctx SpecContext) {
			ins, logs := ObservedInstrumentation(zapcore.DebugLevel)
			mw := MustSucceed(alamos.Middleware(alamos.Config{
				Instrumentation:   ins,
				EnablePropagation: new(false),
			}))
			MustSucceed(mw.Exec(freighter.Context{Context: ctx}, finalizer))
			Expect(logs.Len()).To(Equal(1))
		})

		It("Should not log requests when logging is disabled", func(ctx SpecContext) {
			ins, logs := ObservedInstrumentation(zapcore.DebugLevel)
			mw := MustSucceed(alamos.Middleware(alamos.Config{
				Instrumentation: ins,
				EnableLogging:   new(false),
			}))
			MustSucceed(mw.Exec(freighter.Context{Context: ctx}, finalizer))
			Expect(logs.Len()).To(BeZero())
		})

		It("Should trace requests at the configured level", func(ctx SpecContext) {
			var envs []xalamos.Environment
			tracer := MustSucceed(xalamos.NewTracer(xalamos.TracingConfig{
				OtelProvider:   otel.GetTracerProvider(),
				OtelPropagator: otel.GetTextMapPropagator(),
				Filter: func(env xalamos.Environment, _ string) bool {
					envs = append(envs, env)
					return false
				},
			}))
			mw := MustSucceed(alamos.Middleware(alamos.Config{
				Instrumentation: xalamos.New("falamos", xalamos.WithTracer(tracer)),
				Level:           xalamos.EnvironmentDebug,
			}))
			MustSucceed(mw.Exec(freighter.Context{Context: ctx}, finalizer))
			Expect(envs).To(Equal([]xalamos.Environment{xalamos.EnvironmentDebug}))
		})
	})
})
