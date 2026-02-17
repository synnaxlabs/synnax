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

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

func StreamSuite(
	deps func() (
		freighter.StreamServer[Request, Response],
		freighter.StreamClient[Request, Response],
		address.Address,
	),
) {
	var ctx context.Context
	ginkgo.BeforeEach(func() {
		ctx = context.Background()
	})
	ginkgo.Describe("Normal Operation", func() {
		ginkgo.It("Should exchange messages between a client and a server", func() {
			server, client, addr := deps()
			closed := make(chan struct{})

			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer ginkgo.GinkgoRecover()
				defer close(closed)
				for {
					req, err := server.Receive()
					if err != nil {
						ginkgo.By("Receiving a transport EOF error from the client")
						gomega.Expect(err).To(gomega.MatchError(freighter.EOF))
						return err
					}
					if err := server.Send(Response{ID: req.ID + 1, Message: req.Message}); err != nil {
						return err
					}
				}
			})

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			ginkgo.By("Opening the stream to the target without error")
			stream := testutil.MustSucceed(client.Stream(ctx, addr))

			ginkgo.By("Exchanging ten echo messages")
			for i := range 10 {
				gomega.Expect(stream.Send(Request{ID: i, Message: "Hello"})).To(gomega.Succeed())
				msg := testutil.MustSucceed(stream.Receive())
				gomega.Expect(msg.ID).To(gomega.Equal(i + 1))
				gomega.Expect(msg.Message).To(gomega.Equal("Hello"))
			}

			ginkgo.By("Successfully letting the server know we're done sending messages")
			gomega.Expect(stream.CloseSend()).To(gomega.Succeed())

			ginkgo.By("Receiving a freighter.EOF error from the server")
			gomega.Expect(stream.Receive()).Error().To(gomega.MatchError(freighter.EOF))
			gomega.Eventually(closed).Should(gomega.BeClosed())
		})

		ginkgo.It("Should allow the server to continue sending messages after CloseSend is called", func() {
			server, client, addr := deps()
			serverClosed := make(chan struct{})
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer ginkgo.GinkgoRecover()
				defer close(serverClosed)
				gomega.Expect(server.Receive()).Error().To(gomega.MatchError(freighter.EOF))
				gomega.Expect(server.Send(Response{ID: 1, Message: "Hello"})).To(gomega.Succeed())
				return nil
			})
			stream := testutil.MustSucceed(client.Stream(ctx, addr))
			gomega.Expect(stream.CloseSend()).To(gomega.Succeed())
			msg := testutil.MustSucceed(stream.Receive())
			gomega.Expect(msg.ID).To(gomega.Equal(1))
			gomega.Expect(msg.Message).To(gomega.Equal("Hello"))
			gomega.Expect(stream.Receive()).Error().To(gomega.MatchError(freighter.EOF))
			gomega.Eventually(serverClosed).Should(gomega.BeClosed())
		})

		ginkgo.It("Should exchange messages in excess of the write deadline", func() {
			server, client, addr := deps()
			serverClosed := make(chan struct{})
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer ginkgo.GinkgoRecover()
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

			stream := testutil.MustSucceed(client.Stream(ctx, addr))
			gomega.Expect(stream.Send(Request{ID: 1, Message: "Hello"})).To(gomega.Succeed())
			msg := testutil.MustSucceed(stream.Receive())
			gomega.Expect(msg.ID).To(gomega.Equal(2))
			gomega.Expect(msg.Message).To(gomega.Equal("Hello"))
			time.Sleep(WriteDeadline * 2)
			gomega.Expect(stream.Send(Request{ID: 1, Message: "Hello"})).To(gomega.Succeed())
			msg = testutil.MustSucceed(stream.Receive())
			gomega.Expect(msg.ID).To(gomega.Equal(2))
			gomega.Expect(msg.Message).To(gomega.Equal("Hello"))
			gomega.Expect(stream.CloseSend()).To(gomega.Succeed())
			gomega.Eventually(serverClosed).Should(gomega.BeClosed())
		})
	})

	ginkgo.Describe("Error Handling", func() {
		ginkgo.Describe("Stream returns a non-nil error", func() {
			ginkgo.It("Should send the error to the client", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
					defer ginkgo.GinkgoRecover()
					defer close(serverClosed)
					gomega.Expect(server.Receive()).Error().ToNot(gomega.HaveOccurred())
					return errors.New("zero is not allowed!")
				})
				stream := testutil.MustSucceed(client.Stream(ctx, addr))
				gomega.Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(gomega.Succeed())
				gomega.Expect(stream.Receive()).Error().To(gomega.MatchError(gomega.ContainSubstring("zero is not allowed!")))
				gomega.Eventually(serverClosed).Should(gomega.BeClosed())
			})

			ginkgo.Specify("If the client calls Send, it should return an EOF error", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
					defer ginkgo.GinkgoRecover()
					defer close(serverClosed)
					gomega.Expect(server.Receive()).Error().ToNot(gomega.HaveOccurred())
					return errors.New("zero is not allowed!")
				})
				stream := testutil.MustSucceed(client.Stream(ctx, addr))
				gomega.Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(gomega.Succeed())
				gomega.Expect(stream.Receive()).Error().To(gomega.MatchError(gomega.ContainSubstring("zero is not allowed!")))
				gomega.Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(gomega.MatchError(freighter.EOF))
				gomega.Eventually(serverClosed).Should(gomega.BeClosed())
			})
		})

		ginkgo.Describe("StreamClient cancels the context", func() {
			ginkgo.It("Should propagate the context cancellation to both the server and the client", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
					defer close(serverClosed)
					defer ginkgo.GinkgoRecover()
					gomega.Expect(server.Receive()).Error().To(gomega.MatchError(context.Canceled))
					return nil
				})
				ctx, cancel := context.WithCancel(ctx)
				stream := testutil.MustSucceed(client.Stream(ctx, addr))
				cancel()
				gomega.Expect(stream.Receive()).Error().To(gomega.MatchError(context.Canceled))
				gomega.Eventually(serverClosed).Should(gomega.BeClosed())
			})
		})

		ginkgo.Describe("StreamClient attempts to send a message after calling close send", func() {
			ginkgo.It("Should return a StreamClosed error", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
					defer close(serverClosed)
					defer ginkgo.GinkgoRecover()
					gomega.Expect(server.Receive()).Error().To(gomega.MatchError(freighter.EOF))
					return nil
				})

				ctx, cancel := context.WithCancel(ctx)
				defer cancel()

				stream := testutil.MustSucceed(client.Stream(ctx, addr))
				gomega.Expect(stream.CloseSend()).To(gomega.Succeed())
				gomega.Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(gomega.MatchError(freighter.ErrStreamClosed))
				gomega.Expect(stream.Receive()).Error().To(gomega.MatchError(freighter.EOF))
				gomega.Eventually(serverClosed).Should(gomega.BeClosed())
			})
		})

		ginkgo.Describe("StreamClient attempts to send a message after the server closes", func() {
			ginkgo.It("Should return a EOF error", func() {
				server, client, addr := deps()
				serverClosed := make(chan struct{})
				server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
					defer close(serverClosed)
					for i := range 10 {
						req := testutil.MustSucceed(server.Receive())
						gomega.Expect(server.Send(Response{
							ID:      req.ID + i,
							Message: req.Message,
						})).To(gomega.Succeed())
					}
					return nil
				})
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				stream := testutil.MustSucceed(client.Stream(ctx, addr))
				gomega.Eventually(func(g gomega.Gomega) {
					g.Expect(stream.Send(Request{ID: 0, Message: "Hello"})).To(gomega.MatchError(freighter.EOF))
				}).WithPolling(10 * time.Millisecond).Should(gomega.Succeed())
				gomega.Eventually(serverClosed).Should(gomega.BeClosed())
			})
		})
	})

	ginkgo.Describe("Middleware", func() {
		ginkgo.It("Should correctly execute a middleware in the chain", func() {
			server, client, addr := deps()
			serverClosed := make(chan struct{})
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer close(serverClosed)
				defer ginkgo.GinkgoRecover()
				gomega.Expect(server.Receive()).Error().To(gomega.MatchError(freighter.EOF))
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
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			stream := testutil.MustSucceed(client.Stream(ctx, addr))
			gomega.Expect(stream.CloseSend()).To(gomega.Succeed())
			gomega.Expect(stream.Receive()).Error().To(gomega.MatchError(freighter.EOF))
			gomega.Eventually(serverClosed).Should(gomega.BeClosed())
			gomega.Expect(c).To(gomega.Equal(2))
		})

		ginkgo.It("Should correctly propagate an error that arises in a middleware", func() {
			server, client, addr := deps()
			serverClosed := make(chan struct{})
			server.BindHandler(func(ctx context.Context, server freighter.ServerStream[Request, Response]) error {
				defer close(serverClosed)
				defer ginkgo.GinkgoRecover()
				gomega.Expect(server.Receive()).Error().To(gomega.MatchError(freighter.EOF))
				return nil
			})
			server.Use(freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				return ctx, errors.New("middleware error")
			}))
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			gomega.Expect(client.Stream(ctx, addr)).Error().To(gomega.MatchError(gomega.ContainSubstring("middleware error")))
		})
	})
}
