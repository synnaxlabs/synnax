// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

type (
	StreamServer = freighter.StreamServer[Request, Response]
	StreamClient = freighter.StreamClient[Request, Response]
	ServerStream = freighter.ServerStream[Request, Response]
)

type StreamImplementation interface {
	Start(address.Address, alamos.Instrumentation) (StreamServer, StreamClient)
	Stop() error
}

const StreamWriteDeadline = 20 * time.Millisecond

func AssertStream(impl StreamImplementation) {
	Describe("Fulfills StreamImplementation", Ordered, Serial, func() {
		var (
			addr   address.Address
			server StreamServer
			client StreamClient
		)
		BeforeAll(func() {
			addr = "localhost:8080"
			server, client = impl.Start(addr, alamos.Instrumentation{})
		})
		AfterAll(func() {
			Expect(impl.Stop()).To(Succeed())
		})

		Describe("Normal Operation", func() {

			It("Should exchange messages between a client and a server", func() {
				closed := make(chan struct{})

				server.BindHandler(func(ctx context.Context, server ServerStream) error {
					defer GinkgoRecover()
					defer close(closed)
					for {
						req, err := server.Receive()
						if err != nil {
							By("Receiving a transport EOF error from the client")
							Expect(err).To(HaveOccurredAs(freighter.EOF))
							return err
						}
						if err := server.Send(Response{ID: req.ID + 1, Message: req.Message}); err != nil {
							return err
						}
					}
				})

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				By("Opening the stream to the target without error")
				client, err := client.Stream(ctx, addr)
				Expect(err).ToNot(HaveOccurred())

				By("Exchanging ten echo messages")
				for i := range 10 {
					Expect(client.Send(Request{ID: i, Message: "Hello"})).To(Succeed())
					msg, err := client.Receive()
					Expect(err).ToNot(HaveOccurred())
					Expect(msg.ID).To(Equal(i + 1))
					Expect(msg.Message).To(Equal("Hello"))
				}

				By("Successfully letting the server know we're done sending messages")
				Expect(client.CloseSend()).To(Succeed())

				By("Receiving a freighter.ErrEOF error from the server")
				_, err = client.Receive()
				Expect(err).To(HaveOccurredAs(freighter.EOF))
				Eventually(closed).Should(BeClosed())
			})

			It("Should allow the server to continue sending messages after CloseSend is called", func() {
				serverClosed := make(chan struct{})
				server.BindHandler(func(_ context.Context, server ServerStream) error {
					defer GinkgoRecover()
					defer close(serverClosed)
					_, err := server.Receive()
					Expect(err).To(HaveOccurredAs(freighter.EOF))
					Expect(server.Send(Response{ID: 1, Message: "Hello"})).To(Succeed())
					return nil
				})
				client := MustSucceed(client.Stream(context.Background(), addr))
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
				server.BindHandler(func(_ context.Context, server ServerStream) error {
					defer GinkgoRecover()
					defer close(serverClosed)
					for {
						req, err := server.Receive()
						if err != nil {
							return err
						}
						time.Sleep(StreamWriteDeadline * 5)
						if err := server.Send(Response{ID: req.ID + 1, Message: req.Message}); err != nil {
							return err
						}
					}
				})

				client := MustSucceed(client.Stream(context.Background(), addr))
				Expect(client.Send(Request{ID: 1, Message: "Hello"})).To(Succeed())
				msg, err := client.Receive()
				Expect(err).ToNot(HaveOccurred())
				Expect(msg.ID).To(Equal(2))
				Expect(msg.Message).To(Equal("Hello"))
				time.Sleep(StreamWriteDeadline * 2)
				Expect(client.Send(Request{ID: 1, Message: "Hello"})).To(Succeed())
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
					server.BindHandler(func(_ context.Context, server ServerStream) error {
						defer GinkgoRecover()
						defer close(serverClosed)
						_, err := server.Receive()
						Expect(err).ToNot(HaveOccurred())
						return errors.New("zero is not allowed!")
					})
					client, err := client.Stream(context.Background(), addr)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.Send(Request{ID: 0, Message: "Hello"})).To(Succeed())
					_, err = client.Receive()
					Expect(err).To(HaveOccurredAs(errors.New("zero is not allowed!")))
					Eventually(serverClosed).Should(BeClosed())
				})

				Specify("If the client calls Send, if should return an EOF error", func() {
					serverClosed := make(chan struct{})
					server.BindHandler(func(_ context.Context, server ServerStream) error {
						defer GinkgoRecover()
						defer close(serverClosed)
						_, err := server.Receive()
						if err != nil {
							Fail(err.Error())
						}
						return errors.New("zero is not allowed!")
					})
					client, err := client.Stream(context.Background(), addr)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.Send(Request{ID: 0, Message: "Hello"})).To(Succeed())
					_, err = client.Receive()
					Expect(err).To(HaveOccurredAs(errors.New("zero is not allowed!")))
					err = client.Send(Request{ID: 0, Message: "Hello"})
					Expect(err).To(HaveOccurredAs(freighter.EOF))
					Eventually(serverClosed).Should(BeClosed())
				})

			})

			Describe("StreamClient cancels the context", func() {
				It("Should propagate the context cancellation to both the server and the client", func() {
					serverClosed := make(chan struct{})
					server.BindHandler(func(_ context.Context, server ServerStream) error {
						defer close(serverClosed)
						defer GinkgoRecover()
						_, err := server.Receive()
						Expect(err).To(Equal(context.Canceled))
						return nil
					})
					ctx, cancel := context.WithCancel(context.Background())
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
					server.BindHandler(func(_ context.Context, server ServerStream) error {
						defer close(serverClosed)
						defer GinkgoRecover()
						_, err := server.Receive()
						Expect(err).To(HaveOccurredAs(freighter.EOF))
						return nil
					})

					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					client, err := client.Stream(ctx, addr)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.CloseSend()).To(Succeed())
					err = client.Send(Request{ID: 0, Message: "Hello"})
					Expect(err).To(HaveOccurredAs(freighter.ErrStreamClosed))

					_, err = client.Receive()

					Expect(err).To(HaveOccurredAs(freighter.EOF))
					Eventually(serverClosed).Should(BeClosed())
				})

			})
			Describe("StreamClient attempts to send a message after the server closes", func() {
				It("Should return a EOF error", func() {
					serverClosed := make(chan struct{})
					server.BindHandler(func(_ context.Context, server ServerStream) error {
						defer close(serverClosed)
						for i := range 10 {
							req, err := server.Receive()
							Expect(err).ToNot(HaveOccurred())
							Expect(server.Send(Response{
								ID:      req.ID + i,
								Message: req.Message,
							})).To(Succeed())
						}
						return nil
					})
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					client, err := client.Stream(ctx, addr)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func(g Gomega) {
						g.Expect(client.Send(Request{ID: 0, Message: "Hello"})).To(HaveOccurredAs(freighter.EOF))
					}).WithPolling(10 * time.Millisecond).Should(Succeed())
					Eventually(serverClosed).Should(BeClosed())
				})
			})
		})
		Describe("Middleware", func() {
			It("Should correctly execute a middleware in the chain", func() {
				serverClosed := make(chan struct{})
				server.BindHandler(func(_ context.Context, server ServerStream) error {
					defer close(serverClosed)
					defer GinkgoRecover()
					_, err := server.Receive()
					Expect(err).To(HaveOccurredAs(freighter.EOF))
					return nil
				})
				c := 0
				server.Use(freighter.MiddlewareFunc(func(
					ctx freighter.Context,
					next freighter.MiddlewareHandler,
				) (freighter.Context, error) {
					c++
					oMd, err := next(ctx)
					c++
					return oMd, err
				}))
				ctx, cancel := context.WithCancel(context.Background())
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
				server.BindHandler(func(_ context.Context, server ServerStream) error {
					defer close(serverClosed)
					defer GinkgoRecover()
					_, err := server.Receive()
					Expect(err).To(HaveOccurredAs(freighter.EOF))
					return nil
				})
				server.Use(freighter.MiddlewareFunc(func(
					ctx freighter.Context,
					_ freighter.MiddlewareHandler,
				) (freighter.Context, error) {
					return ctx, errors.New("middleware error")
				}))
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				_, err := client.Stream(ctx, addr)
				Expect(err).To(HaveOccurred())
				Expect(err).To(HaveOccurredAs(errors.New("middleware error")))
			})

		})
	})
}
