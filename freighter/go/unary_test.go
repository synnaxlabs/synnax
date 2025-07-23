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
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
)

type (
	unaryServer = freighter.UnaryServer[request, response]
	unaryClient = freighter.UnaryClient[request, response]
)

type unaryImplementation interface {
	start(address.Address) (unaryServer, unaryClient)
	stop() error
}

var unaryImplementations = []unaryImplementation{
	&httpUnaryImplementation{},
	&mockUnaryImplementation{},
}

var _ = Describe("Unary", Ordered, Serial, func() {
	Describe("Implementation Tests", func() {
		for _, impl := range unaryImplementations {
			var (
				addr   address.Address
				server unaryServer
				client unaryClient
				req    request
			)
			BeforeAll(func() {
				addr = "localhost:8081"
				server, client = impl.start(addr)
				req = request{ID: 1, Message: "hello"}
			})
			AfterAll(func() {
				Expect(impl.stop()).To(Succeed())
			})
			Describe("Normal Operation", func() {
				It("should send a request", func() {
					server.BindHandler(func(
						_ context.Context,
						req request,
					) (response, error) {
						return req, nil
					})
					Expect(client.Send(context.Background(), addr, req)).To(Equal(req))
				})
			})
			Describe("Details Handling", func() {
				It("Should correctly return a custom error to the client", func() {
					server.BindHandler(func(
						context.Context,
						request,
					) (response, error) {
						return response{}, myCustomError
					})
					Expect(client.Send(context.Background(), addr, req)).
						Error().To(MatchError(myCustomError))
				})
			})
			Describe("Middleware", func() {
				It("Should correctly call the middleware", func() {
					c := 0
					server.Use(freighter.MiddlewareFunc(func(
						ctx freighter.Context,
						next freighter.MiddlewareHandler,
					) (freighter.Context, error) {
						c++
						oMd, err := next(ctx)
						if err != nil {
							return freighter.Context{}, err
						}
						c++
						return oMd, nil
					}))
					server.BindHandler(func(
						context.Context,
						request,
					) (response, error) {
						return response{}, nil
					})
					Expect(client.Send(context.Background(), addr, req)).
						Error().To(Not(HaveOccurred()))
					Expect(c).To(Equal(2))
				})
			})
		}
	})
})

type httpUnaryImplementation struct{ app *fiber.App }

var _ unaryImplementation = &httpUnaryImplementation{}

func (i *httpUnaryImplementation) start(
	host address.Address,
) (unaryServer, unaryClient) {
	i.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	router := fhttp.NewRouter(fhttp.RouterConfig{})
	factory := fhttp.NewClientFactory(fhttp.ClientFactoryConfig{
		Codec: httputil.JSONCodec,
	})
	server := fhttp.UnaryServer[request, response](router, "/")
	client := fhttp.UnaryClient[request, response](factory)
	router.BindTo(i.app)
	i.app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	go func() {
		if err := i.app.Listen(host.PortString()); err != nil {
			log.Fatal(err)
		}
	}()
	Eventually(func(g Gomega) {
		g.Expect(http.Get("http://" + host.String() + "/health")).
			Error().ToNot(HaveOccurred())
	}).WithPolling(1 * time.Millisecond).Should(Succeed())
	return server, client
}

func (i *httpUnaryImplementation) stop() error { return i.app.Shutdown() }

type mockUnaryImplementation struct{}

var _ unaryImplementation = &mockUnaryImplementation{}

func (i *mockUnaryImplementation) start(
	host address.Address,
) (unaryServer, unaryClient) {
	net := fmock.NewNetwork[request, response]()
	server := net.UnaryServer(host)
	client := net.UnaryClient()
	return server, client
}

func (i *mockUnaryImplementation) stop() error { return nil }
