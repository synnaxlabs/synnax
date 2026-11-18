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
		router := MustSucceed(fhttp.NewRouter(context.Background(), fhttp.RouterConfig{
			Errors:              errorCodec(),
			StreamWriteDeadline: testutil.WriteDeadline,
		}))
		server = router.NewStreamServer[testutil.Request, testutil.Response]("/")
		client = MustSucceed(fhttp.NewStreamClient[testutil.Request, testutil.Response](
			fhttp.StreamClientConfig{Codec: json.Codec, Errors: errorCodec()},
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

	Describe("NewStreamClient", func() {
		It("Should reject a config without an error codec", func() {
			Expect(fhttp.NewStreamClient[testutil.Request, testutil.Response](
				fhttp.StreamClientConfig{Codec: json.Codec},
			)).Error().To(MatchError(ContainSubstring("errors: must be non-nil")))
		})
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
			router := MustSucceed(fhttp.NewRouter(ctx, fhttp.RouterConfig{
				Errors:                 errorCodec(),
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
			"should construct the codec with a live context that ends with the stream",
			func(ctx SpecContext) {
				const contentType = "application/x-test"
				codecCtx := make(chan context.Context, 1)
				router := MustSucceed(fhttp.NewRouter(ctx, fhttp.RouterConfig{
					Errors:              errorCodec(),
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
				_, ownAddr := serveRouter(router)
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
				Eventually(got.Done()).Should(BeClosed())
			},
		)
	})

	Describe("Request context", func() {
		It("should pass request context values to the stream handler", func(
			ctx SpecContext,
		) {
			type key struct{}
			router := MustSucceed(fhttp.NewRouter(ctx, fhttp.RouterConfig{
				Errors:              errorCodec(),
				StreamWriteDeadline: testutil.WriteDeadline,
			}))
			ownServer := router.NewStreamServer[testutil.Request, testutil.Response](
				"/",
			)
			values := make(chan any, 1)
			ownServer.BindHandler(func(
				ctx context.Context,
				stream freighter.ServerStream[testutil.Request, testutil.Response],
			) error {
				values <- ctx.Value(key{})
				_, err := stream.Receive()
				return err
			})
			app := newFiberApp(fiber.Config{})
			app.Use(func(c fiber.Ctx) error {
				c.SetContext(context.WithValue(c.Context(), key{}, "cat"))
				return c.Next()
			})
			router.BindTo(app)
			addr := serveApp(app)
			headers := http.Header{}
			headers.Set(fiber.HeaderContentType, "application/json")
			conn, res := MustSucceed2((&ws.Dialer{}).DialContext(
				ctx, "ws://"+addr.String()+"/", headers,
			))
			DeferCleanup(func() { Expect(conn.Close()).To(Succeed()) })
			Expect(res.Body.Close()).To(Succeed())
			Eventually(values).Should(Receive(Equal("cat")))
		})
	})

	Describe("Subprotocol token", func() {
		var (
			ownAddr address.Address
			params  chan freighter.Params
		)
		BeforeEach(func(ctx SpecContext) {
			router := MustSucceed(fhttp.NewRouter(ctx, fhttp.RouterConfig{
				Errors:              errorCodec(),
				StreamWriteDeadline: testutil.WriteDeadline,
			}))
			ownServer := router.NewStreamServer[testutil.Request, testutil.Response](
				"/",
			)
			params = make(chan freighter.Params, 1)
			ownServer.Use(freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				params <- ctx.Params
				return next(ctx)
			}))
			ownServer.BindHandler(func(
				_ context.Context,
				stream freighter.ServerStream[testutil.Request, testutil.Response],
			) error {
				_, err := stream.Receive()
				return err
			})
			var ownApp *fiber.App
			ownApp, ownAddr = serveRouter(router)
			DeferCleanup(func() { Expect(ownApp.Shutdown()).To(Succeed()) })
		})

		dial := func(
			ctx context.Context,
			query string,
			subprotocols ...string,
		) *http.Response {
			GinkgoHelper()
			headers := http.Header{}
			headers.Set(fiber.HeaderContentType, "application/json")
			dialer := &ws.Dialer{Subprotocols: subprotocols}
			conn, res := MustSucceed2(dialer.DialContext(
				ctx, "ws://"+ownAddr.String()+"/"+query, headers,
			))
			DeferCleanup(func() { Expect(conn.Close()).To(Succeed()) })
			Expect(res.Body.Close()).To(Succeed())
			return res
		}

		It("Should take the token from the subprotocol and reply with only synnax.v1",
			func(ctx SpecContext) {
				res := dial(ctx, "", "synnax.v1", "synnax.token.sys_abc")
				Expect(res.Header.Get(fiber.HeaderSecWebSocketProtocol)).
					To(Equal("synnax.v1"))
				var p freighter.Params
				Eventually(params).Should(Receive(&p))
				Expect(p[fiber.HeaderAuthorization]).To(Equal("Bearer sys_abc"))
				for k := range p {
					Expect(strings.ToLower(k)).ToNot(Equal("sec-websocket-protocol"))
				}
			},
		)

		It("Should keep the query token when the subprotocol holds no token",
			func(ctx SpecContext) {
				query := "?freighterctxAuthorization=Bearer%20sys_q"
				res := dial(ctx, query, "synnax.v1")
				Expect(res.Header.Get(fiber.HeaderSecWebSocketProtocol)).
					To(Equal("synnax.v1"))
				Eventually(params).Should(Receive(
					HaveKeyWithValue(fiber.HeaderAuthorization, "Bearer sys_q"),
				))
			},
		)

		It("Should reply with no subprotocol when the client asks for none",
			func(ctx SpecContext) {
				res := dial(ctx, "")
				Expect(res.Header.Get(fiber.HeaderSecWebSocketProtocol)).To(BeEmpty())
				Eventually(params).Should(Receive(
					Not(HaveKey(fiber.HeaderAuthorization)),
				))
			},
		)
	})

	Describe("Shutdown", func() {
		// Serves its own app: shutting down the shared one would strand later specs.
		It(
			"should stop serving a stream whose peer never answers the close message",
			func(ctx SpecContext) {
				router := MustSucceed(fhttp.NewRouter(ctx, fhttp.RouterConfig{
					Errors:              errorCodec(),
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

		It(
			"should end a stream when the router's lifetime ends",
			func(ctx SpecContext) {
				life, cancel := context.WithCancel(ctx)
				router := MustSucceed(fhttp.NewRouter(life, fhttp.RouterConfig{
					Errors:              errorCodec(),
					StreamWriteDeadline: testutil.WriteDeadline,
				}))
				ownServer := router.NewStreamServer[testutil.Request, testutil.Response](
					"/",
				)
				ended := make(chan struct{})
				ownServer.BindHandler(func(
					ctx context.Context,
					_ freighter.ServerStream[testutil.Request, testutil.Response],
				) error {
					<-ctx.Done()
					close(ended)
					return nil
				})
				_, ownAddr := serveRouter(router)
				headers := http.Header{}
				headers.Set(fiber.HeaderContentType, "application/json")
				conn, res := MustSucceed2((&ws.Dialer{}).DialContext(
					ctx, "ws://"+ownAddr.String()+"/", headers,
				))
				DeferCleanup(func() { Expect(conn.Close()).To(Succeed()) })
				Expect(res.Body.Close()).To(Succeed())
				Consistently(ended, 20*time.Millisecond).ShouldNot(BeClosed())
				cancel()
				Eventually(ended).Should(BeClosed())
			},
		)
	})
})
