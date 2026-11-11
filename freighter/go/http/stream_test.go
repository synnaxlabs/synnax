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
	"strings"
	"time"

	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	. "github.com/synnaxlabs/x/testutil"
)

func serveRouter(router *fhttp.Router) (*fiber.App, address.Address) {
	GinkgoHelper()
	app := newFiberApp(fiber.Config{})
	app.Get("/health", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	router.BindTo(app)
	addr := serveApp(app)
	Eventually(func(g Gomega) {
		g.Expect(pollHealth("http://" + addr.String() + "/health")).To(Succeed())
	}).WithPolling(time.Millisecond).Should(Succeed())
	return app, addr
}

var _ = Describe("Stream", Ordered, Serial, func() {
	var (
		server freighter.StreamServer[testutil.Request, testutil.Response]
		client freighter.StreamClient[testutil.Request, testutil.Response]
		addr   address.Address
	)

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		router := MustSucceed(fhttp.NewRouter(fhttp.RouterConfig{
			StreamWriteDeadline: testutil.WriteDeadline,
		}))
		server = router.NewStreamServer[testutil.Request, testutil.Response]("/")
		client = MustSucceed(fhttp.NewStreamClient[testutil.Request, testutil.Response](
			fhttp.StreamClientConfig{Codec: json.Codec},
		))
		_, addr = serveRouter(router)
	})

	testutil.StreamSuite(func() (
		freighter.StreamServer[testutil.Request, testutil.Response],
		freighter.StreamClient[testutil.Request, testutil.Response],
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
				Expect(report["encodings"]).To(Equal([]string{"application/json"}))
			},
		)
	})

	Describe("Upgrade Negotiation", func() {
		DescribeTable(
			"should return 415 Unsupported Media Type when the upgrade request advertises a Content-Type with no registered codec",
			func(ctx context.Context, contentType string) {
				headers := http.Header{}
				headers.Set(fiber.HeaderContentType, contentType)
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
			Entry("an unknown codec", "application/x-no-such-codec"),
			Entry("msgpack", "application/msgpack"),
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

	Describe("Message size limit", func() {
		It("Should fail a stream whose peer sends a message past the limit", func(
			ctx SpecContext,
		) {
			router := MustSucceed(fhttp.NewRouter(fhttp.RouterConfig{
				StreamWriteDeadline:    testutil.WriteDeadline,
				StreamMessageSizeLimit: 64,
			}))
			ownServer := router.NewStreamServer[testutil.Request, testutil.Response](
				"/",
			)
			received := make(chan error, 1)
			ownServer.BindHandler(func(
				_ context.Context,
				stream freighter.ServerStream[testutil.Request, testutil.Response],
			) error {
				_, err := stream.Receive()
				received <- err
				return err
			})
			ownApp, ownAddr := serveRouter(router)
			DeferCleanup(func() { Expect(ownApp.Shutdown()).To(Succeed()) })
			stream := MustSucceed(client.Stream(ctx, ownAddr))
			Expect(stream.Send(testutil.Request{Message: strings.Repeat("a", 128)})).
				To(Succeed())
			Eventually(received).Should(Receive(MatchError(freighter.ErrStreamClosed)))
			Expect(stream.CloseSend()).To(Succeed())
		})
	})

	Describe("WithAdditionalCodec", func() {
		// Serves its own app: shutting down the shared one would strand later specs.
		It(
			"should construct the codec with a live context that ends when the app stops",
			func(ctx SpecContext) {
				const contentType = "application/x-test"
				codecCtx := make(chan context.Context, 1)
				router := MustSucceed(fhttp.NewRouter(fhttp.RouterConfig{
					StreamWriteDeadline: testutil.WriteDeadline,
				}))
				ownServer := router.NewStreamServer[testutil.Request, testutil.Response](
					"/",
					fhttp.WithAdditionalCodec(
						contentType,
						func(ctx context.Context) encoding.Codec {
							codecCtx <- ctx
							return json.Codec
						},
					),
				)
				ownServer.BindHandler(func(
					_ context.Context,
					stream freighter.ServerStream[testutil.Request, testutil.Response],
				) error {
					_, err := stream.Receive()
					return err
				})
				ownApp, ownAddr := serveRouter(router)
				headers := http.Header{}
				headers.Set(fiber.HeaderContentType, contentType)
				conn, res := MustSucceed2((&ws.Dialer{}).DialContext(
					ctx, "ws://"+ownAddr.String()+"/", headers,
				))
				Expect(res.Body.Close()).To(Succeed())
				var got context.Context
				Eventually(codecCtx).Should(Receive(&got))
				Expect(got.Err()).ToNot(HaveOccurred())
				Expect(conn.Close()).To(Succeed())
				Expect(ownApp.Shutdown()).To(Succeed())
				Eventually(got.Done()).Should(BeClosed())
			},
		)
	})

	Describe("Shutdown", func() {
		// Serves its own app: shutting down the shared one would strand later specs.
		It(
			"should stop serving a stream whose peer never answers the close message",
			func(ctx SpecContext) {
				router := MustSucceed(fhttp.NewRouter(fhttp.RouterConfig{
					StreamWriteDeadline: testutil.WriteDeadline,
				}))
				ownServer := router.NewStreamServer[testutil.Request, testutil.Response](
					"/",
				)
				serving := make(chan struct{})
				ownServer.BindHandler(func(
					_ context.Context,
					stream freighter.ServerStream[testutil.Request, testutil.Response],
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
