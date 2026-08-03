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

	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/net"
	. "github.com/synnaxlabs/x/testutil"
)

// serveRouter serves the router on an open port, returning once the app is answering
// requests. All servers must be registered on the router first.
func serveRouter(router *fhttp.Router) (*fiber.App, address.Address) {
	addr := address.Newf("localhost:%d", MustSucceed(net.FindOpenPort()))
	app := newFiberApp(fiber.Config{})
	app.Get("/health", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	router.BindTo(app)
	go func() {
		defer GinkgoRecover()
		Expect(app.Listen(addr.PortString(), fiber.ListenConfig{
			DisableStartupMessage: true,
		})).To(Succeed())
	}()
	Eventually(func(g Gomega) {
		g.Expect(pollHealth("http://" + addr.String() + "/health")).To(Succeed())
	}).WithPolling(time.Millisecond).Should(Succeed())
	return app, addr
}

var _ = Describe("Stream", Ordered, Serial, func() {
	var (
		server freighter.StreamServer[test.Request, test.Response]
		client freighter.StreamClient[test.Request, test.Response]
		addr   address.Address
		app    *fiber.App
	)

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		router := MustSucceed(fhttp.NewRouter(fhttp.RouterConfig{
			StreamWriteDeadline: test.WriteDeadline,
		}))
		server = fhttp.NewStreamServer[test.Request, test.Response](router, "/")
		client = MustSucceed(fhttp.NewStreamClient[test.Request, test.Response](
			fhttp.StreamClientConfig{Codec: json.Codec},
		))
		app, addr = serveRouter(router)
	})

	AfterAll(func() { Expect(app.Shutdown()).To(Succeed()) })

	test.StreamSuite(func() (
		freighter.StreamServer[test.Request, test.Response],
		freighter.StreamClient[test.Request, test.Response],
		address.Address,
	) {
		return server, client, addr
	})

	Describe("Report", func() {
		It(
			"should report the stream server's protocol and the content types it can negotiate at upgrade time",
			func() {
				report := server.Report()
				Expect(report["protocol"]).To(Equal("websocket"))
				Expect(report["encodings"]).To(Equal([]string{
					"application/json", "application/msgpack",
				}))
			},
		)
	})

	Describe("Upgrade Negotiation", func() {
		It(
			"should return 415 Unsupported Media Type when the upgrade request advertises a Content-Type with no registered codec",
			func(ctx context.Context) {
				headers := http.Header{}
				headers.Set(fiber.HeaderContentType, "application/x-no-such-codec")
				_, res, err := (&ws.Dialer{}).DialContext(
					ctx,
					"ws://"+addr.String()+"/",
					headers,
				)
				Expect(err).To(MatchError(ws.ErrBadHandshake))
				Expect(res).ToNot(BeNil())
				DeferCleanup(func() { Expect(res.Body.Close()).To(Succeed()) })
				Expect(res.StatusCode).To(Equal(http.StatusUnsupportedMediaType))
			},
		)

		It(
			"should return 426 Upgrade Required when the request is not a websocket upgrade",
			func() {
				res := MustSucceed(http.Get("http://" + addr.String() + "/"))
				DeferCleanup(func() { Expect(res.Body.Close()).To(Succeed()) })
				Expect(res.StatusCode).To(Equal(http.StatusUpgradeRequired))
			},
		)
	})

	Describe("Shutdown", func() {
		// Serves its own app: shutting down the shared one would strand later specs.
		It(
			"should stop serving a stream whose peer never answers the close message",
			func(ctx SpecContext) {
				router := MustSucceed(fhttp.NewRouter(fhttp.RouterConfig{
					StreamWriteDeadline: test.WriteDeadline,
				}))
				ownServer := fhttp.NewStreamServer[test.Request, test.Response](
					router,
					"/",
				)
				serving := make(chan struct{})
				ownServer.BindHandler(func(
					_ context.Context,
					stream freighter.ServerStream[test.Request, test.Response],
				) error {
					close(serving)
					_, err := stream.Receive()
					return err
				})
				ownApp, ownAddr := serveRouter(router)

				// A raw peer that upgrades and then never reads. Control frames are
				// only processed inside a read, so the close message goes unanswered.
				headers := http.Header{}
				headers.Set(fiber.HeaderContentType, "application/json")
				conn, res := MustSucceed2((&ws.Dialer{}).DialContext(
					ctx, "ws://"+ownAddr.String()+"/", headers,
				))
				DeferCleanup(func() { Expect(conn.Close()).To(Succeed()) })
				Expect(res.Body.Close()).To(Succeed())
				Eventually(serving).Should(BeClosed())

				shutdown := make(chan error, 1)
				go func() { shutdown <- ownApp.Shutdown() }()
				Eventually(shutdown, 5*time.Second).Should(Receive(BeNil()))
			},
		)
	})
})
