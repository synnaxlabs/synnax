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
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/freighter/recovery"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/net"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Recovery (wire)", func() {
	It("should contain a handler panic and keep serving", func(ctx context.Context) {
		addr := address.Newf("localhost:%d", MustSucceed(net.FindOpenPort()))
		app := fiber.New(fiber.Config{})
		app.Get("/health", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
		router := MustSucceed(fhttp.NewRouter())
		server := fhttp.NewUnaryServer[test.Request, test.Response](router, "/")
		server.Use(recovery.Middleware(alamos.Instrumentation{}))

		panicNext := true
		server.BindHandler(func(_ context.Context, req test.Request) (test.Response, error) {
			if panicNext {
				panic("boom in handler")
			}
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
			_, err := http.Get("http://" + addr.String() + "/health")
			g.Expect(err).To(Succeed())
		}).WithPolling(time.Millisecond).Should(Succeed())

		client := MustSucceed(fhttp.NewUnaryClient[test.Request, test.Response]())

		By("surfacing the panic to the client as a generic error, not a dropped connection")
		Expect(client.Send(ctx, addr, test.Request{ID: 1})).
			Error().To(MatchError(ContainSubstring(recovery.ErrPanic.Error())))

		By("continuing to serve subsequent requests")
		panicNext = false
		Expect(MustSucceed(client.Send(ctx, addr, test.Request{ID: 2, Message: "ok"}))).
			To(Equal(test.Response{ID: 2, Message: "ok"}))
	})
})
