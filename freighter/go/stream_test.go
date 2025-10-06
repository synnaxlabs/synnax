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
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/httputil"
	. "github.com/synnaxlabs/x/testutil"
)

type (
	streamServer = freighter.StreamServer[request, response]
	streamClient = freighter.StreamClient[request, response]
	serverStream = freighter.ServerStream[request, response]
)

type streamImplementation interface {
	name() string
	start(host address.Address, ins alamos.Instrumentation) (streamServer, streamClient)
	stop() error
}

const writeDeadline = 20 * time.Millisecond

var streamImplementations = []streamImplementation{
	&httpStreamImplementation{},
	&mockStreamImplementation{},
}

var _ = Describe("Stream", Ordered, Serial, func() {
	for _, impl := range streamImplementations {
		impl := impl
		var (
			addr   address.Address
			server streamServer
			client streamClient
		)
		BeforeAll(func() {
			addr = "localhost:8080"
			server, client = impl.start(addr, alamos.Instrumentation{})
		})
		AfterAll(func() {
			Expect(impl.stop()).ToNot(HaveOccurred())
		})
		Context(fmt.Sprintf("Implementation %s", impl.name()), func() {

			Describe("Normal Operation", func() {

				It("Should exchange messages between a client and a server", func() {
					closed := make(chan struct{})

					server.BindHandler(func(ctx context.Context, server serverStream) error {
						defer GinkgoRecover()
						defer close(closed)
						for {
							req, err := server.Receive()
							if err != nil {
								By("Receiving a transport EOF error from the client")
								Expect(err).To(HaveOccurredAs(freighter.EOF))
								return err
							}
							if err := server.Send(response{ID: req.ID + 1, Message: req.Message}); err != nil {
								return err
							}
						}
					})

					ctx, cancel := context.WithCancel(context.TODO())
					defer cancel()

					By("Opening the stream to the target without error")
					client, err := client.Stream(ctx, addr)
					Expect(err).ToNot(HaveOccurred())

					By("Exchanging ten echo messages")
					for i := range 10 {
						Expect(client.Send(request{ID: i, Message: "Hello"})).To(Succeed())
						msg, err := client.Receive()
						Expect(err).ToNot(HaveOccurred())
						Expect(msg.ID).To(Equal(i + 1))
						Expect(msg.Message).To(Equal("Hello"))
					}

					By("Successfully letting the server know we're done sending messages")
					Expect(client.CloseSend()).To(Succeed())

					By("Receiving a freighter.EOF error from the server")
					_, err = client.Receive()
					Expect(err).To(HaveOccurredAs(freighter.EOF))
					Eventually(closed).Should(BeClosed())
				})

				It("Should allow the server to continue sending messages after CloseSend is called", func() {
					serverClosed := make(chan struct{})
					server.BindHandler(func(ctx context.Context, server serverStream) error {
						defer GinkgoRecover()
						defer close(serverClosed)
						_, err := server.Receive()
						Expect(err).To(HaveOccurredAs(freighter.EOF))
						Expect(server.Send(response{ID: 1, Message: "Hello"})).To(Succeed())
						return nil
					})
					client := MustSucceed(client.Stream(context.TODO(), addr))
					Expect(client.CloseSend()).To(Succeed())
					msg, err := client.Receive()
					Expect(err).ToNot(HaveOccurred())
					Expect(msg.ID).To(Equal(1))
					Expect(msg.Message).To(Equal("Hello"))
					_, err = client.Receive()
					Expect(err).To(HaveOccurredAs(freighter.EOF))
					Eventually(serverClosed).Should(BeClosed())
				})

				It("Should exchange messages in excess of the write deadline", func() {
					serverClosed := make(chan struct{})
					server.BindHandler(func(ctx context.Context, server serverStream) error {
						defer GinkgoRecover()
						defer close(serverClosed)
						for {
							req, err := server.Receive()
							if err != nil {
								return err
							}
							time.Sleep(writeDeadline * 5)
							if err := server.Send(response{ID: req.ID + 1, Message: req.Message}); err != nil {
								return err
							}
						}
					})

					client := MustSucceed(client.Stream(context.TODO(), addr))
					Expect(client.Send(request{ID: 1, Message: "Hello"})).To(Succeed())
					msg, err := client.Receive()
					Expect(err).ToNot(HaveOccurred())
					Expect(msg.ID).To(Equal(2))
					Expect(msg.Message).To(Equal("Hello"))
					time.Sleep(writeDeadline * 2)
					Expect(client.Send(request{ID: 1, Message: "Hello"})).To(Succeed())
					msg, err = client.Receive()
					Expect(err).ToNot(HaveOccurred())
					Expect(msg.ID).To(Equal(2))
					Expect(msg.Message).To(Equal("Hello"))
					Expect(client.CloseSend()).To(Succeed())
					Eventually(serverClosed).Should(BeClosed())
				})

			})
			Describe("Error Handling", func() {

				Describe("Stream returns a non-nil error", func() {
					It("Should send the error to the client", func() {
						serverClosed := make(chan struct{})
						server.BindHandler(func(ctx context.Context, server serverStream) error {
							defer GinkgoRecover()
							defer close(serverClosed)
							_, err := server.Receive()
							Expect(err).ToNot(HaveOccurred())
							return errors.New("zero is not allowed!")
						})
						client, err := client.Stream(context.TODO(), addr)
						Expect(err).ToNot(HaveOccurred())
						Expect(client.Send(request{ID: 0, Message: "Hello"})).To(Succeed())
						_, err = client.Receive()
						Expect(err).To(HaveOccurredAs(errors.New("zero is not allowed!")))
						Eventually(serverClosed).Should(BeClosed())
					})

					Specify("If the client calls Send, if should return an EOF error", func() {
						serverClosed := make(chan struct{})
						server.BindHandler(func(ctx context.Context, server serverStream) error {
							defer GinkgoRecover()
							defer close(serverClosed)
							_, err := server.Receive()
							if err != nil {
								Fail(err.Error())
							}
							return errors.New("zero is not allowed!")
						})
						client, err := client.Stream(context.TODO(), addr)
						Expect(err).ToNot(HaveOccurred())
						Expect(client.Send(request{ID: 0, Message: "Hello"})).To(Succeed())
						_, err = client.Receive()
						Expect(err).To(HaveOccurredAs(errors.New("zero is not allowed!")))
						err = client.Send(request{ID: 0, Message: "Hello"})
						Expect(err).To(HaveOccurredAs(freighter.EOF))
						Eventually(serverClosed).Should(BeClosed())
					})

				})

				Describe("StreamClient cancels the context", func() {
					It("Should propagate the context cancellation to both the server and the client", func() {
						serverClosed := make(chan struct{})
						server.BindHandler(func(ctx context.Context, server serverStream) error {
							defer close(serverClosed)
							defer GinkgoRecover()
							_, err := server.Receive()
							Expect(err).To(HaveOccurredAs(context.Canceled))
							return nil
						})
						ctx, cancel := context.WithCancel(context.TODO())
						client, err := client.Stream(ctx, addr)
						Expect(err).ToNot(HaveOccurred())
						cancel()
						_, err = client.Receive()
						Expect(err).To(HaveOccurredAs(context.Canceled))
						Eventually(serverClosed).Should(BeClosed())
					})
				})

				Describe("StreamClient attempts to send a message after calling close send", func() {
					It("Should return a StreamClosed error", func() {
						serverClosed := make(chan struct{})
						server.BindHandler(func(ctx context.Context, server serverStream) error {
							defer close(serverClosed)
							defer GinkgoRecover()
							_, err := server.Receive()
							Expect(err).To(HaveOccurredAs(freighter.EOF))
							return nil
						})

						ctx, cancel := context.WithCancel(context.TODO())
						defer cancel()

						client, err := client.Stream(ctx, addr)
						Expect(err).ToNot(HaveOccurred())
						Expect(client.CloseSend()).To(Succeed())
						err = client.Send(request{ID: 0, Message: "Hello"})
						Expect(err).To(HaveOccurredAs(freighter.StreamClosed))

						_, err = client.Receive()

						Expect(err).To(HaveOccurredAs(freighter.EOF))
						Eventually(serverClosed).Should(BeClosed())
					})

				})
				Describe("StreamClient attempts to send a message after the server closes", func() {
					It("Should return a EOF error", func() {
						serverClosed := make(chan struct{})
						server.BindHandler(func(ctx context.Context, server serverStream) error {
							defer close(serverClosed)
							for i := range 10 {
								req, err := server.Receive()
								Expect(err).ToNot(HaveOccurred())
								Expect(server.Send(response{
									ID:      req.ID + i,
									Message: req.Message,
								})).To(Succeed())
							}
							return nil
						})
						ctx, cancel := context.WithCancel(context.TODO())
						defer cancel()
						client, err := client.Stream(ctx, addr)
						Expect(err).ToNot(HaveOccurred())
						Eventually(func(g Gomega) {
							g.Expect(client.Send(request{ID: 0, Message: "Hello"})).To(HaveOccurredAs(freighter.EOF))
						}).WithPolling(10 * time.Millisecond).Should(Succeed())
						Eventually(serverClosed).Should(BeClosed())
					})
				})
			})
			Describe("Middleware", func() {
				It("Should correctly execute a middleware in the chain", func() {
					serverClosed := make(chan struct{})
					server.BindHandler(func(ctx context.Context, server serverStream) error {
						defer close(serverClosed)
						defer GinkgoRecover()
						_, err := server.Receive()
						Expect(err).To(HaveOccurredAs(freighter.EOF))
						return nil
					})
					c := 0
					server.Use(freighter.MiddlewareFunc(func(
						ctx freighter.Context,
						next freighter.Next,
					) (freighter.Context, error) {
						c++
						oMd, err := next(ctx)
						c++
						return oMd, err
					}))
					ctx, cancel := context.WithCancel(context.TODO())
					defer cancel()
					client, err := client.Stream(ctx, addr)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.CloseSend()).To(Succeed())
					_, err = client.Receive()
					Expect(err).To(HaveOccurredAs(freighter.EOF))
					Eventually(serverClosed).Should(BeClosed())
					Expect(c).To(Equal(2))
				})
				It("Should correctly propagate an error that arises in a middleware", func() {
					serverClosed := make(chan struct{})
					server.BindHandler(func(ctx context.Context, server serverStream) error {
						defer close(serverClosed)
						defer GinkgoRecover()
						_, err := server.Receive()
						Expect(err).To(HaveOccurredAs(freighter.EOF))
						return nil
					})
					server.Use(freighter.MiddlewareFunc(func(
						ctx freighter.Context,
						next freighter.Next,
					) (freighter.Context, error) {
						return ctx, errors.New("middleware error")
					}))
					ctx, cancel := context.WithCancel(context.TODO())
					defer cancel()
					_, err := client.Stream(ctx, addr)
					Expect(err).To(HaveOccurred())
					Expect(err).To(HaveOccurredAs(errors.New("middleware error")))
				})
			})
		})
	}
	Describe("SenderNopCloser", func() {
		It("Should implement the freighter.StreamSenderCloser interface", func() {
			var closer freighter.StreamSenderCloser[int] = freighter.SenderNopCloser[int]{}
			Expect(closer.CloseSend()).To(Succeed())
		})
	})
})

type httpStreamImplementation struct {
	app *fiber.App
}

func (impl *httpStreamImplementation) name() string { return "HTTP" }

func (impl *httpStreamImplementation) start(
	host address.Address,
	ins alamos.Instrumentation,
) (streamServer, streamClient) {
	impl.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	router := fhttp.NewRouter(fhttp.RouterConfig{
		Instrumentation:     ins,
		StreamWriteDeadline: writeDeadline,
	})
	client := fhttp.NewClientFactory(fhttp.ClientFactoryConfig{Codec: httputil.JSONCodec})
	impl.app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	server := fhttp.StreamServer[request, response](router, "/")
	router.BindTo(impl.app)
	go func() {
		defer GinkgoRecover()
		Expect(impl.app.Listen(host.PortString())).To(Succeed())
	}()
	Eventually(func(g Gomega) {
		_, err := http.Get("http://" + host.String() + "/health")
		g.Expect(err).ToNot(HaveOccurred())
	}).WithPolling(1 * time.Millisecond).Should(Succeed())
	return server, fhttp.StreamClient[request, response](client)
}

func (impl *httpStreamImplementation) stop() error {
	return impl.app.Shutdown()
}

type mockStreamImplementation struct{}

func (impl *mockStreamImplementation) name() string { return "Mock" }

func (impl *mockStreamImplementation) start(
	host address.Address,
	ins alamos.Instrumentation,
) (streamServer, streamClient) {
	return fmock.NewStreamPair[request, response]( /*request buffer */ 11 /* response buffer */, 11)
}

func (impl *mockStreamImplementation) stop() error { return nil }
