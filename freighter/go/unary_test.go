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
	"go/types"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
	. "github.com/synnaxlabs/x/testutil"
)

type (
	unaryServer     = freighter.UnaryServer[request, response]
	unaryClient     = freighter.UnaryClient[request, response]
	streamingClient = freighter.UnaryClient[types.Nil, io.Reader]
	streamingServer = freighter.UnaryServer[types.Nil, io.Reader]
)

type unaryImplementation interface {
	start(address.Address) (unaryServer, unaryClient, streamingServer, streamingClient)
	stop() error
}

var unaryImplementations = []unaryImplementation{&httpUnaryImplementation{}}

var _ = Describe("Unary", Ordered, Serial, func() {
	Describe("Implementation Tests", func() {
		for _, impl := range unaryImplementations {
			var (
				addr            address.Address
				server          unaryServer
				client          unaryClient
				streamingServer streamingServer
				streamingClient streamingClient
			)
			BeforeAll(func() {
				addr = "localhost:8081"
				server, client, streamingServer, streamingClient = impl.start(addr)
			})
			AfterAll(func() {
				Expect(impl.stop()).To(Succeed())
			})
			Describe("Normal Operation", func() {
				It("should send a request", func() {
					server.BindHandler(func(_ context.Context, req request) (response, error) {
						return response{
							ID:      req.ID,
							Message: req.Message,
						}, nil
					})
					req := request{ID: 1, Message: "hello"}
					res, err := client.Send(context.Background(), addr, req)
					Expect(err).To(Succeed())
					Expect(res).To(Equal(response{ID: 1, Message: "hello"}))
				})
			})
			Describe("streaming", Focus, func() {
				It("should allow for stream processing", func() {
					streamingServer.BindHandler(func(context.Context, types.Nil) (io.Reader, error) {
						r, w := io.Pipe()
						go func() {
							for i := range 5 {
								w.Write([]byte{byte(i)})
							}
							w.Close()
						}()
						return r, nil
					})
					reader := MustSucceed(streamingClient.Send(context.Background(), addr+"/streaming", types.Nil{}))
					data := MustSucceed(io.ReadAll(reader))
					Expect(data).To(Equal([]byte{0, 1, 2, 3, 4}))
				})

			})
			Describe("Details Handling", func() {
				It("Should correctly return a custom error to the client", func() {
					server.BindHandler(func(context.Context, request) (response, error) {
						return response{}, myCustomError
					})
					req := request{ID: 1, Message: "hello"}
					_, err := client.Send(context.Background(), addr, req)
					Expect(err).To(Equal(myCustomError))
				})
			})
			Describe("Middleware", func() {
				It("Should correctly call the middleware", func() {
					c := 0
					server.Use(freighter.MiddlewareFunc(func(ctx freighter.Context, next freighter.Next) (freighter.Context, error) {
						c++
						oMd, err := next(ctx)
						c++
						return oMd, err
					}))
					server.BindHandler(func(context.Context, request) (response, error) {
						return response{}, nil
					})
					req := request{ID: 1, Message: "hello"}
					_, err := client.Send(context.Background(), addr, req)
					Expect(err).To(Succeed())
					Expect(c).To(Equal(2))
				})
			})
		}
	})
})

type httpUnaryImplementation struct{ app *fiber.App }

var _ unaryImplementation = &httpUnaryImplementation{}

func (h *httpUnaryImplementation) start(host address.Address) (unaryServer, unaryClient, streamingServer, streamingClient) {
	h.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	router := fhttp.NewRouter(fhttp.RouterConfig{})
	factory := fhttp.NewClientFactory(fhttp.ClientFactoryConfig{
		Codec: httputil.JSONCodec,
	})
	server := fhttp.UnaryServer[request, response](router, "/")
	client := fhttp.UnaryClient[request, response](factory)
	streamingClient := fhttp.UnaryClient[types.Nil, io.Reader](factory)
	streamingServer := fhttp.UnaryServer[types.Nil, io.Reader](router, "/streaming")
	router.BindTo(h.app)
	h.app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	go func() {
		if err := h.app.Listen(host.PortString()); err != nil {
			log.Fatal(err)
		}
	}()
	Eventually(func(g Gomega) {
		_, err := http.Get("http://" + host.String() + "/health")
		g.Expect(err).ToNot(HaveOccurred())
	}).WithPolling(1 * time.Millisecond).Should(Succeed())
	return server, client, streamingServer, streamingClient
}

func (h *httpUnaryImplementation) stop() error { return h.app.Shutdown() }
