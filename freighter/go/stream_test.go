package freighter_test

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/httputil"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type sampleRequest struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

type sampleResponse struct {
	ID      int    `json:"id" msgpack:"id"`
	Message string `json:"message" msgpack:"message"`
}

type (
	streamServer = freighter.StreamServer[sampleRequest, sampleResponse]
	streamClient = freighter.StreamClient[sampleRequest, sampleResponse]
	serverStream = freighter.ServerStream[sampleRequest, sampleResponse]
)

type implementation interface {
	start(host address.Address, logger *zap.SugaredLogger) (streamServer, streamClient)
	stop() error
}

var implementations = []implementation{
	&httpImplementation{},
	&mockImplementation{},
}

var _ = Describe("Stream", Ordered, Serial, func() {
	Describe("Implementation Tests", func() {

		for _, impl := range implementations {
			impl := impl
			var (
				addr   address.Address
				server streamServer
				client streamClient
			)
			BeforeAll(func() {
				addr = "localhost:8080"
				l := zap.NewNop()
				server, client = impl.start(addr, l.Sugar())
			})
			AfterAll(func() {
				Expect(impl.stop()).ToNot(HaveOccurred())
			})
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
							if err := server.Send(sampleResponse{ID: req.ID + 1, Message: req.Message}); err != nil {
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
					for i := 0; i < 10; i++ {
						Expect(client.Send(sampleRequest{ID: i, Message: "Hello"})).To(Succeed())
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
						Expect(server.Send(sampleResponse{ID: 1, Message: "Hello"})).To(Succeed())
						return nil
					})
					client, err := client.Stream(context.TODO(), addr)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.CloseSend()).To(Succeed())
					msg, err := client.Receive()
					Expect(err).ToNot(HaveOccurred())
					Expect(msg.ID).To(Equal(1))
					Expect(msg.Message).To(Equal("Hello"))
					_, err = client.Receive()
					Expect(err).To(HaveOccurredAs(freighter.EOF))
					Eventually(serverClosed).Should(BeClosed())
				})

			})
			Describe("Error Handling", func() {
				Describe("Server returns a non-nil error", func() {
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
						Expect(client.Send(sampleRequest{ID: 0, Message: "Hello"})).To(Succeed())
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
						Expect(client.Send(sampleRequest{ID: 0, Message: "Hello"})).To(Succeed())
						_, err = client.Receive()
						Expect(err).To(HaveOccurredAs(errors.New("zero is not allowed!")))
						err = client.Send(sampleRequest{ID: 0, Message: "Hello"})
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
							Expect(err).To(Equal(context.Canceled))
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
						err = client.Send(sampleRequest{ID: 0, Message: "Hello"})
						Expect(err).To(HaveOccurredAs(freighter.StreamClosed))

						_, err = client.Receive()

						Expect(err).To(HaveOccurredAs(freighter.EOF))
						Eventually(serverClosed).Should(BeClosed())
					})

				})
			})

			Describe("StreamClient attempts to send a message after the server closes", func() {
				It("Should return a EOF error", func() {
					serverClosed := make(chan struct{})
					server.BindHandler(func(ctx context.Context, server serverStream) error {
						defer close(serverClosed)
						for i := 0; i < 10; i++ {
							req, err := server.Receive()
							Expect(err).ToNot(HaveOccurred())
							Expect(server.Send(sampleResponse{
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
						g.Expect(client.Send(sampleRequest{ID: 0, Message: "Hello"})).To(HaveOccurredAs(freighter.EOF))
					}).WithPolling(10 * time.Millisecond).Should(Succeed())
					Eventually(serverClosed).Should(BeClosed())
				})
			})
		}
	})
	Describe("SenderNopCloser", func() {
		It("Should implement the freighter.StreamSenderCloser interface", func() {
			var closer freighter.StreamSenderCloser[int] = freighter.SenderNopCloser[int]{}
			Expect(closer.CloseSend()).To(Succeed())
		})
	})
})

type httpImplementation struct {
	app *fiber.App
}

func (impl *httpImplementation) start(
	host address.Address,
	logger *zap.SugaredLogger,
) (streamServer, streamClient) {
	impl.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	router := fhttp.NewRouter(fhttp.RouterConfig{App: impl.app, Logger: logger})
	client := fhttp.NewClient(fhttp.ClientConfig{EncoderDecoder: httputil.MsgPackEncoderDecoder, Logger: logger})
	impl.app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	server := fhttp.StreamServer[sampleRequest, sampleResponse](router, "/")
	go func() {
		if err := impl.app.Listen(host.PortString()); err != nil {
			logger.Error(err)
		}
	}()
	Eventually(func(g Gomega) {
		_, err := http.Get("http://" + host.String() + "/health")
		g.Expect(err).ToNot(HaveOccurred())
	}).WithPolling(1 * time.Millisecond).Should(Succeed())
	return server, fhttp.NewStreamClient[sampleRequest, sampleResponse](client)
}

func (impl *httpImplementation) stop() error {
	return impl.app.Shutdown()
}

type mockImplementation struct {
	net *fmock.Network[sampleRequest, sampleResponse]
}

func (impl *mockImplementation) start(
	host address.Address,
	logger *zap.SugaredLogger,
) (streamServer, streamClient) {
	return fmock.NewStreamPair[sampleRequest, sampleResponse]( /*request buffer */ 11 /* response buffer */, 11)
}

func (impl *mockImplementation) stop() error { return nil }
