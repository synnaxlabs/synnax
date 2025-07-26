// Copyright 2025 Synnax Labs, Inc.
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
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
	httpIntegration "github.com/synnaxlabs/freighter/integration/http"
	"github.com/synnaxlabs/freighter/integration/payload"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
	. "github.com/synnaxlabs/x/testutil"
)

type message = payload.Message

type (
	unaryClient  = freighter.UnaryClient[message, message]
	streamClient = freighter.StreamClient[message, message]
	readerClient = freighter.UnaryClient[message, io.Reader]
)

var _ = Describe("HTTP Integration", Ordered, Serial, func() {
	var (
		app         *fiber.App
		addr        address.Address
		unaryEcho   unaryClient
		unaryReader readerClient
		streamEcho  streamClient
		ctx         context.Context
		cancel      context.CancelFunc
	)

	BeforeAll(func() {
		addr = "localhost:8082"
		app = fiber.New(fiber.Config{DisableStartupMessage: true})
		httpIntegration.BindTo(app)

		clientConfig := fhttp.ClientConfig{Codec: httputil.JSONCodec}
		unaryEcho = MustSucceed(fhttp.NewUnaryClient[message, message](clientConfig))
		unaryReader = MustSucceed(
			fhttp.NewUnaryClient[message, io.Reader](clientConfig),
		)
		streamEcho = MustSucceed(fhttp.NewStreamClient[message, message](clientConfig))

		app.Get("/health", func(ctx *fiber.Ctx) error {
			return ctx.SendStatus(fiber.StatusOK)
		})

		go func() {
			defer GinkgoRecover()
			Expect(app.Listen(addr.PortString())).To(Succeed())
		}()

		Eventually(func(g Gomega) {
			MustSucceed(http.Get("http://" + addr.String() + "/health"))
		}).WithPolling(1 * time.Millisecond).Should(Succeed())
	})

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() { cancel() })

	AfterAll(func() { Expect(app.Shutdown()).To(Succeed()) })

	Describe("Unary Routes", func() {
		Describe("/unary/echo", func() {
			It("Should echo the message with incremented ID", func() {
				req := message{ID: 5, Message: "hello world"}
				res := MustSucceed(unaryEcho.Send(ctx, addr+"/unary/echo", req))
				Expect(res.ID).To(Equal(6))
				Expect(res.Message).To(Equal("hello world"))
			})

			It("Should handle multiple requests", func() {
				for i := 1; i <= 5; i++ {
					req := message{ID: i, Message: "test"}
					res := MustSucceed(unaryEcho.Send(ctx, addr+"/unary/echo", req))
					Expect(res.ID).To(Equal(i + 1))
					Expect(res.Message).To(Equal("test"))
				}
			})
		})

		Describe("/unary/getReader", func() {
			It("Should return a reader with the message content", func() {
				req := message{ID: 1, Message: "test content"}
				reader := MustSucceed(unaryReader.Send(ctx, addr+"/unary/getReader", req))
				content := MustSucceed(io.ReadAll(reader))
				Expect(string(content)).To(Equal("test content"))
			})

			It("Should handle empty message", func() {
				req := message{ID: 1, Message: ""}
				reader := MustSucceed(unaryReader.Send(ctx, addr+"/unary/getReader", req))
				content := MustSucceed(io.ReadAll(reader))
				Expect(string(content)).To(Equal(""))
			})
		})

		Describe("/unary/middlewareCheck", func() {
			It("Should fail when test param is missing", func() {
				req := message{ID: 10, Message: "middleware test"}
				_, err := unaryEcho.Send(ctx, addr+"/unary/middlewareCheck", req)
				Expect(err).To(HaveOccurred())
				var payloadErr payload.Error
				Expect(errors.As(err, &payloadErr)).To(BeTrue())
				Expect(payloadErr.Message).To(Equal("test param not found"))
				Expect(payloadErr.Code).To(Equal(1))
			})
			It("should pass middleware parameters", func() {
				unaryEcho.Use(freighter.MiddlewareFunc(func(ctx freighter.Context, next freighter.MiddlewareHandler) (freighter.Context, error) {
					ctx.Set("Test", "test")
					return next(ctx)
				}))
				req := message{ID: 10, Message: "middleware test"}
				MustSucceed(unaryEcho.Send(ctx, addr+"/unary/middlewareCheck", req))
			})
		})

		Describe("/unary/slamMessagesTimeoutCheck", func() {
			It("Should return success for non-timeout message", func() {
				req := message{ID: 1, Message: "no-timeout"}
				res := MustSucceed(unaryEcho.Send(ctx, addr+"/unary/slamMessagesTimeoutCheck", req))
				Expect(res.Message).To(Equal("success"))
			})
		})
	})

	Describe("Stream Routes", func() {
		Describe("/stream/echo", func() {
			It("Should echo messages with incremented IDs", func() {
				stream := MustSucceed(streamEcho.Stream(ctx, addr+"/stream/echo"))

				for i := 1; i <= 5; i++ {
					req := message{ID: i, Message: "echo test"}
					Expect(stream.Send(req)).To(Succeed())

					res := MustSucceed(stream.Receive())
					Expect(res.ID).To(Equal(i + 1))
					Expect(res.Message).To(Equal("echo test"))
				}

				Expect(stream.CloseSend()).To(Succeed())
				Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
			})
		})

		Describe("/stream/respondWithTenMessages", func() {
			It("Should receive exactly ten messages", func() {
				stream := MustSucceed(streamEcho.Stream(ctx, addr+"/stream/respondWithTenMessages"))

				receivedMessages := make([]message, 0, 10)
				for range 10 {
					msg := MustSucceed(stream.Receive())
					receivedMessages = append(receivedMessages, msg)
				}

				Expect(len(receivedMessages)).To(Equal(10))
				for i, msg := range receivedMessages {
					Expect(msg.Message).To(Equal("hello"))
					Expect(msg.ID).To(Equal(i))
				}

				// Should receive EOF after ten messages
				Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
			})
		})

		Describe("/stream/immediatelyExitNominally", func() {
			It("Should exit immediately without error", func() {
				stream := MustSucceed(streamEcho.Stream(ctx, addr+"/stream/immediatelyExitNominally"))
				Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
			})
		})

		Describe("/stream/immediatelyExitWithErr", func() {
			It("Should exit immediately with error", func() {
				stream := MustSucceed(streamEcho.Stream(ctx, addr+"/stream/immediatelyExitWithErr"))
				_, err := stream.Receive()
				Expect(err).To(HaveOccurred())
				var payloadErr payload.Error
				Expect(errors.As(err, &payloadErr)).To(BeTrue())
				Expect(payloadErr.Message).To(Equal("unexpected error"))
				Expect(payloadErr.Code).To(Equal(1))
			})
		})

		Describe("/stream/receiveAndExitWithErr", func() {
			It("Should receive one message then exit with error", func() {
				stream := MustSucceed(streamEcho.Stream(ctx, addr+"/stream/receiveAndExitWithErr"))

				req := message{ID: 1, Message: "test"}
				Expect(stream.Send(req)).To(Succeed())

				_, err := stream.Receive()
				Expect(err).To(HaveOccurred())
				var payloadErr payload.Error
				Expect(errors.As(err, &payloadErr)).To(BeTrue(), "error is not a payload error")
				Expect(payloadErr.Message).To(Equal("unexpected error"))
				Expect(payloadErr.Code).To(Equal(1))
			})
		})

		Describe("/stream/sendMessageAfterClientClose", func() {
			It("Should send acknowledgment after client closes", func() {
				stream := MustSucceed(streamEcho.Stream(ctx, addr+"/stream/sendMessageAfterClientClose"))
				Expect(stream.CloseSend()).To(Succeed())
				msg := MustSucceed(stream.Receive())
				Expect(msg.Message).To(Equal("Close Acknowledged"))
			})
		})

		Describe("/stream/eventuallyResponseWithMessage", func() {
			It("Should receive a message after delay", func() {
				stream := MustSucceed(streamEcho.Stream(ctx, addr+"/stream/eventuallyResponseWithMessage"))

				req := message{ID: 1, Message: "delayed response"}
				Expect(stream.Send(req)).To(Succeed())

				start := time.Now()
				msg := MustSucceed(stream.Receive())
				elapsed := time.Since(start)

				Expect(msg.Message).To(Equal("hello"))
				Expect(msg.ID).To(Equal(1))
				Expect(elapsed).To(BeNumerically("~", 250*time.Millisecond, 10*time.Millisecond), "elapsed time is about 250ms")
			})
		})

		Describe("/stream/middlewareCheck", func() {
			It("Should fail when Test param is missing", func() {
				_, err := streamEcho.Stream(ctx, addr+"/stream/middlewareCheck")
				Expect(err).To(HaveOccurred())
				var payloadErr payload.Error
				Expect(errors.As(err, &payloadErr)).To(BeTrue())
				Expect(payloadErr.Message).To(Equal("test param not found"))
				Expect(payloadErr.Code).To(Equal(1))
			})

			It("should pass middleware parameters", func() {
				streamEcho.Use(freighter.MiddlewareFunc(func(ctx freighter.Context, next freighter.MiddlewareHandler) (freighter.Context, error) {
					if ctx.Params == nil {
						ctx.Params = make(freighter.Params)
					}
					ctx.Params["Test"] = "test"
					return next(ctx)
				}))
				stream := MustSucceed(streamEcho.Stream(ctx, addr+"/stream/middlewareCheck"))
				req := message{ID: 1, Message: "hello"}
				Expect(stream.Send(req)).To(Succeed())
				msg := MustSucceed(stream.Receive())
				Expect(msg.Message).To(Equal("hello"))
				Expect(msg.ID).To(Equal(2))
			})
		})

		Describe("/stream/slamMessages", func() {
			It("Should handle high volume message stream", func() {
				stream := MustSucceed(streamEcho.Stream(ctx, addr+"/stream/slamMessages"))
				req := message{ID: 1, Message: "slam-test"}
				Expect(stream.Send(req)).To(Succeed())

				// Receive as many messages as possible until timeout or error
				messageCount := 0
				timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 20*time.Millisecond)
				defer timeoutCancel()

			ReceiveLoop:
				for {
					select {
					case <-timeoutCtx.Done():
						break ReceiveLoop
					default:
						_, err := stream.Receive()
						if err != nil {
							// Expected to eventually get an error due to timeout or
							// stream closure
							break ReceiveLoop
						}
						messageCount++
					}
				}

				// Should have received a significant number of messages
				Expect(messageCount).To(BeNumerically(">", 100))
			})
		})
	})

	Describe("Context Cancellation", func() {
		It("Should handle context cancellation gracefully for streams", func() {
			shortCtx, shortCancel := context.WithCancel(ctx)
			stream := MustSucceed(streamEcho.Stream(shortCtx, addr+"/stream/echo"))
			req := message{ID: 1, Message: "cancel test"}
			Expect(stream.Send(req)).To(Succeed())
			// Cancel context
			shortCancel()
			Eventually(func() error {
				_, err := stream.Receive()
				return err
			}).Should(HaveOccurred())
		})
	})

	Describe("Error Handling", func() {
		It("Should handle non-existent routes", func() {
			req := message{ID: 1, Message: "not found"}
			Expect(unaryEcho.Send(ctx, addr+"/nonexistent", req)).Error().To(HaveOccurred())
		})
	})
})
