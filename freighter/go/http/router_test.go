// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding/json"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Router", func() {
	Describe("Config", func() {
		Describe("Validate", func() {
			It("should accept the zero value", func() {
				Expect(fhttp.RouterConfig{}.Validate()).To(Succeed())
			})

			It("should accept a fully populated config", func() {
				cfg := fhttp.RouterConfig{
					Instrumentation:     alamos.New("test"),
					StreamWriteDeadline: 100 * time.Millisecond,
				}
				Expect(cfg.Validate()).To(Succeed())
			})
		})

		Describe("Override", func() {
			It("should fill StreamWriteDeadline from the override when receiver is zero", func() {
				base := fhttp.RouterConfig{}
				override := fhttp.RouterConfig{StreamWriteDeadline: 50 * time.Millisecond}
				Expect(base.Override(override).StreamWriteDeadline).
					To(Equal(50 * time.Millisecond))
			})

			It("should preserve receiver's StreamWriteDeadline when the override is zero", func() {
				base := fhttp.RouterConfig{StreamWriteDeadline: 50 * time.Millisecond}
				Expect(base.Override(fhttp.RouterConfig{}).StreamWriteDeadline).
					To(Equal(50 * time.Millisecond))
			})

			It("should let a non-zero override win over a non-zero receiver", func() {
				base := fhttp.RouterConfig{StreamWriteDeadline: 5 * time.Millisecond}
				override := fhttp.RouterConfig{StreamWriteDeadline: 50 * time.Millisecond}
				Expect(base.Override(override).StreamWriteDeadline).
					To(Equal(50 * time.Millisecond))
			})

			It("should fill Instrumentation from the override when receiver is zero", func() {
				ins := alamos.New("test")
				base := fhttp.RouterConfig{}
				override := fhttp.RouterConfig{Instrumentation: ins}
				Expect(base.Override(override).Instrumentation.IsZero()).To(BeFalse())
			})

			It("should preserve receiver's Instrumentation when the override is zero", func() {
				ins := alamos.New("test")
				base := fhttp.RouterConfig{Instrumentation: ins}
				Expect(base.Override(fhttp.RouterConfig{}).Instrumentation.IsZero()).
					To(BeFalse())
			})
		})
	})

	Describe("New", func() {
		It("should construct a router from the zero RouterConfig", func() {
			r := MustSucceed(fhttp.NewRouter())
			Expect(r).ToNot(BeNil())
		})

		It("should accept an explicit RouterConfig", func() {
			Expect(fhttp.NewRouter(fhttp.RouterConfig{
				StreamWriteDeadline: 25 * time.Millisecond,
			})).ToNot(BeNil())
		})

		It("should accept multiple RouterConfigs", func() {
			Expect(fhttp.NewRouter(
				fhttp.RouterConfig{StreamWriteDeadline: 5 * time.Millisecond},
				fhttp.RouterConfig{StreamWriteDeadline: 50 * time.Millisecond},
			)).ToNot(BeNil())
		})
	})

	It("should report an empty Report — the router itself is a passive container", func() {
		r := MustSucceed(fhttp.NewRouter())
		Expect(r.Report()).To(BeEmpty())
	})

	Describe("BindTo", func() {
		It("should register a unary route on the bound fiber app", func(specCtx SpecContext) {
			addr := address.Address("localhost:8095")
			app := fiber.New(fiber.Config{})
			router := MustSucceed(fhttp.NewRouter())
			server := fhttp.NewUnaryServer[test.Request, test.Response](router, "/echo")
			server.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
			router.BindTo(app)
			go func() {
				defer GinkgoRecover()
				Expect(app.Listen(addr.PortString(), fiber.ListenConfig{
					DisableStartupMessage: true,
				})).To(Succeed())
			}()
			DeferCleanup(func() { Expect(app.Shutdown()).To(Succeed()) })

			Eventually(func(g Gomega) {
				_, err := http.Get("http://" + addr.String() + "/echo")
				g.Expect(err).To(Succeed())
			}).WithPolling(time.Millisecond).Should(Succeed())

			client := MustSucceed(fhttp.NewUnaryClient[test.Request, test.Response](
				fhttp.ClientConfig{Codec: json.Codec},
			))
			res := MustSucceed(client.Send(specCtx, addr+"/echo", test.Request{ID: 1, Message: "hi"}))
			Expect(res).To(Equal(test.Response{ID: 1, Message: "hi"}))
		})

		It("should cancel in-flight streams when the bound fiber app shuts down", func(specCtx SpecContext) {
			addr := address.Address("localhost:8096")
			app := fiber.New(fiber.Config{})
			router := MustSucceed(fhttp.NewRouter())

			handlerEntered := make(chan struct{})
			handlerCtxDone := make(chan struct{})
			server := fhttp.NewStreamServer[test.Request, test.Response](router, "/stream")
			server.BindHandler(func(
				ctx context.Context,
				_ freighter.ServerStream[test.Request, test.Response],
			) error {
				close(handlerEntered)
				<-ctx.Done()
				close(handlerCtxDone)
				return ctx.Err()
			})
			router.BindTo(app)

			go func() {
				defer GinkgoRecover()
				Expect(app.Listen(addr.PortString(), fiber.ListenConfig{
					DisableStartupMessage: true,
				})).To(Succeed())
			}()

			Eventually(func(g Gomega) {
				_, err := http.Get("http://" + addr.String() + "/anything")
				g.Expect(err).To(Succeed())
			}).WithPolling(time.Millisecond).Should(Succeed())

			client := MustSucceed(fhttp.NewStreamClient[test.Request, test.Response](
				fhttp.ClientConfig{Codec: json.Codec},
			))
			stream := MustSucceed(client.Stream(specCtx, addr+"/stream"))
			Eventually(handlerEntered).Should(BeClosed())

			Expect(app.Shutdown()).To(Succeed())
			Eventually(handlerCtxDone).Should(BeClosed())
			Expect(stream.CloseSend()).To(Succeed())
		})
	})

	Describe("Use", func() {
		It("should install middleware on every server registered before the call", func(specCtx SpecContext) {
			addr := address.Address("localhost:8097")
			app := fiber.New(fiber.Config{})
			router := MustSucceed(fhttp.NewRouter())

			calls := 0
			server := fhttp.NewUnaryServer[test.Request, test.Response](router, "/echo")
			server.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
			router.Use(freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				calls++
				return next(ctx)
			}))
			router.BindTo(app)

			go func() {
				defer GinkgoRecover()
				Expect(app.Listen(addr.PortString(), fiber.ListenConfig{
					DisableStartupMessage: true,
				})).To(Succeed())
			}()
			DeferCleanup(func() { Expect(app.Shutdown()).To(Succeed()) })
			Eventually(func(g Gomega) {
				_, err := http.Get("http://" + addr.String() + "/anything")
				g.Expect(err).To(Succeed())
			}).WithPolling(time.Millisecond).Should(Succeed())

			client := MustSucceed(fhttp.NewUnaryClient[test.Request, test.Response](
				fhttp.ClientConfig{Codec: json.Codec},
			))
			MustSucceed(client.Send(specCtx, addr+"/echo", test.Request{ID: 1, Message: "hi"}))
			Expect(calls).To(Equal(1))
		})

		It("should not install middleware on servers registered after the call", func(specCtx SpecContext) {
			addr := address.Address("localhost:8098")
			app := fiber.New(fiber.Config{})
			router := MustSucceed(fhttp.NewRouter())

			calls := 0
			router.Use(freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				calls++
				return next(ctx)
			}))
			server := fhttp.NewUnaryServer[test.Request, test.Response](router, "/echo")
			server.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				return test.Response(req), nil
			})
			router.BindTo(app)

			go func() {
				defer GinkgoRecover()
				Expect(app.Listen(addr.PortString(), fiber.ListenConfig{
					DisableStartupMessage: true,
				})).To(Succeed())
			}()
			DeferCleanup(func() { Expect(app.Shutdown()).To(Succeed()) })
			Eventually(func(g Gomega) {
				_, err := http.Get("http://" + addr.String() + "/anything")
				g.Expect(err).To(Succeed())
			}).WithPolling(time.Millisecond).Should(Succeed())

			client := MustSucceed(fhttp.NewUnaryClient[test.Request, test.Response](
				fhttp.ClientConfig{Codec: json.Codec},
			))
			Expect(client.Send(specCtx, addr+"/echo", test.Request{ID: 1, Message: "hi"})).To(Equal(test.Response{
				ID:      1,
				Message: "hi",
			}))
			Expect(calls).To(Equal(0))
		})

		It("should chain multiple middlewares in registration order", func(specCtx SpecContext) {
			addr := address.Address("localhost:8099")
			app := fiber.New(fiber.Config{})
			router := MustSucceed(fhttp.NewRouter())

			var order []string
			server := fhttp.NewUnaryServer[test.Request, test.Response](router, "/echo")
			server.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
				order = append(order, "handler")
				return test.Response(req), nil
			})
			router.Use(
				freighter.MiddlewareFunc(func(
					ctx freighter.Context,
					next freighter.Next,
				) (freighter.Context, error) {
					order = append(order, "first")
					return next(ctx)
				}),
				freighter.MiddlewareFunc(func(
					ctx freighter.Context,
					next freighter.Next,
				) (freighter.Context, error) {
					order = append(order, "second")
					return next(ctx)
				}),
			)
			router.BindTo(app)

			go func() {
				defer GinkgoRecover()
				Expect(app.Listen(addr.PortString(), fiber.ListenConfig{
					DisableStartupMessage: true,
				})).To(Succeed())
			}()
			DeferCleanup(func() { Expect(app.Shutdown()).To(Succeed()) })
			Eventually(func(g Gomega) {
				_, err := http.Get("http://" + addr.String() + "/anything")
				g.Expect(err).To(Succeed())
			}).WithPolling(time.Millisecond).Should(Succeed())

			client := MustSucceed(fhttp.NewUnaryClient[test.Request, test.Response](
				fhttp.ClientConfig{Codec: json.Codec},
			))
			Expect(client.Send(specCtx, addr+"/echo", test.Request{ID: 1, Message: "hi"})).To(Equal(test.Response{
				ID:      1,
				Message: "hi",
			}))
			Expect(order).To(Equal([]string{"first", "second", "handler"}))
		})
	})
})
