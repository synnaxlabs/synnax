// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

func StreamSuite(
	deps func() (
		freighter.StreamServer[Request, Response],
		freighter.StreamClient[Request, Response],
		address.Address,
	),
) {
	Describe("Normal Operation", func() {
		It("Should exchange messages between a client and a server", func() {
			server, client, addr := deps()
			closed := make(chan struct{})

			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer GinkgoRecover()
				defer close(closed)
				for {
					req, err := server.Receive()
					if err != nil {
						By("Receiving a transport EOF error from the client")
						Expect(err).To(MatchError(freighter.EOF))
						return err
					}
					if err := server.Send(Response{ID: req.ID + 1, Message: req.Message}); err != nil {
						return err
					}
				}
			})

			ctx, cancel := context.WithCancel(context.TODO())
			defer cancel()

			By("Opening the stream to the target without error")
			stream := MustSucceed(client.Stream(ctx, addr))

			By("Exchanging ten echo messages")
			for i := range 10 {
				Expect(stream.Send(Request{ID: i, Message: "Hello"})).To(Succeed())
				msg := MustSucceed(stream.Receive())
				Expect(msg.ID).To(Equal(i + 1))
				Expect(msg.Message).To(Equal("Hello"))
			}

			By("Successfully letting the server know we're done sending messages")
			Expect(stream.CloseSend()).To(Succeed())

			By("Receiving a freighter.EOF error from the server")
			Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
			Eventually(closed).Should(BeClosed())
		})

		It("Should allow the server to continue sending messages after CloseSend is called", func() {
			server, client, addr := deps()
			serverClosed := make(chan struct{})
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer GinkgoRecover()
				defer close(serverClosed)
				Expect(server.Receive()).Error().To(MatchError(freighter.EOF))
				Expect(server.Send(Response{ID: 1, Message: "Hello"})).To(Succeed())
				return nil
			})
			stream := MustSucceed(client.Stream(context.TODO(), addr))
			Expect(stream.CloseSend()).To(Succeed())
			msg := MustSucceed(stream.Receive())
			Expect(msg.ID).To(Equal(1))
			Expect(msg.Message).To(Equal("Hello"))
			Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
			Eventually(serverClosed).Should(BeClosed())
		})

		It("Should exchange messages in excess of the write deadline", func() {
			server, client, addr := deps()
			serverClosed := make(chan struct{})
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer GinkgoRecover()
				defer close(serverClosed)
				for {
					req, err := server.Receive()
					if err != nil {
						return err
					}
					time.Sleep(WriteDeadline * 5)
					if err := server.Send(Response{ID: req.ID + 1, Message: req.Message}); err != nil {
						return err
					}
				}
			})

			stream := MustSucceed(client.Stream(context.TODO(), addr))
			Expect(stream.Send(Request{ID: 1, Message: "Hello"})).To(Succeed())
			msg := MustSucceed(stream.Receive())
			Expect(msg.ID).To(Equal(2))
			Expect(msg.Message).To(Equal("Hello"))
			time.Sleep(WriteDeadline * 2)
			Expect(stream.Send(Request{ID: 1, Message: "Hello"})).To(Succeed())
			msg = MustSucceed(stream.Receive())
			Expect(msg.ID).To(Equal(2))
			Expect(msg.Message).To(Equal("Hello"))
			Expect(stream.CloseSend()).To(Succeed())
			Eventually(serverClosed).Should(BeClosed())
		})
	})

	Describe("Error Handling", func() {
		Describe("Stream returns a non-nil error", func() {
			It("Should send the error to the client", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
					defer GinkgoRecover()
					defer close(serverClosed)
					Expect(server.Receive()).Error().ToNot(HaveOccurred())
					return errors.New("zero is not allowed!")
				})
				stream := MustSucceed(client.Stream(context.TODO(), addr))
				Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(Succeed())
				Expect(stream.Receive()).Error().To(MatchError(ContainSubstring("zero is not allowed!")))
				Eventually(serverClosed).Should(BeClosed())
			})

			Specify("If the client calls Send, it should return an EOF error", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
					defer GinkgoRecover()
					defer close(serverClosed)
					_, err := server.Receive()
					if err != nil {
						Fail(err.Error())
					}
					return errors.New("zero is not allowed!")
				})
				stream := MustSucceed(client.Stream(context.TODO(), addr))
				Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(Succeed())
				Expect(stream.Receive()).Error().To(MatchError(ContainSubstring("zero is not allowed!")))
				Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(MatchError(freighter.EOF))
				Eventually(serverClosed).Should(BeClosed())
			})
		})

		Describe("StreamClient cancels the context", func() {
			It("Should propagate the context cancellation to both the server and the client", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
					defer close(serverClosed)
					defer GinkgoRecover()
					Expect(server.Receive()).Error().To(MatchError(context.Canceled))
					return nil
				})
				ctx, cancel := context.WithCancel(context.TODO())
				stream := MustSucceed(client.Stream(ctx, addr))
				cancel()
				Expect(stream.Receive()).Error().To(MatchError(context.Canceled))
				Eventually(serverClosed).Should(BeClosed())
			})
		})

		Describe("StreamClient attempts to send a message after calling close send", func() {
			It("Should return a StreamClosed error", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
					defer close(serverClosed)
					defer GinkgoRecover()
					Expect(server.Receive()).Error().To(MatchError(freighter.EOF))
					return nil
				})

				ctx, cancel := context.WithCancel(context.TODO())
				defer cancel()

				stream := MustSucceed(client.Stream(ctx, addr))
				Expect(stream.CloseSend()).To(Succeed())
				Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(MatchError(freighter.ErrStreamClosed))
				Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
				Eventually(serverClosed).Should(BeClosed())
			})
		})

		Describe("StreamClient attempts to send a message after the server closes", func() {
			It("Should return a EOF error", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
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
				ctx, cancel := context.WithCancel(context.TODO())
				defer cancel()
				stream := MustSucceed(client.Stream(ctx, addr))
				Eventually(func(g Gomega) {
					g.Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(MatchError(freighter.EOF))
				}).WithPolling(10 * time.Millisecond).Should(Succeed())
				Eventually(serverClosed).Should(BeClosed())
			})
		})
	})

	Describe("Middleware", func() {
		It("Should correctly execute a middleware in the chain", func() {
			server, client, addr := deps()
			serverClosed := make(chan struct{})
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer close(serverClosed)
				defer GinkgoRecover()
				Expect(server.Receive()).Error().To(MatchError(freighter.EOF))
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
			stream := MustSucceed(client.Stream(ctx, addr))
			Expect(stream.CloseSend()).To(Succeed())
			Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
			Eventually(serverClosed).Should(BeClosed())
			Expect(c).To(Equal(2))
		})

		It("Should correctly propagate an error that arises in a middleware", func() {
			server, client, addr := deps()
			serverClosed := make(chan struct{})
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer close(serverClosed)
				defer GinkgoRecover()
				Expect(server.Receive()).Error().To(MatchError(freighter.EOF))
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
			Expect(err).To(MatchError(ContainSubstring("middleware error")))
		})
	})
}
