package freighter_test

import (
	"context"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/freighter/fws"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/httputil"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
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
	sampleTransport    = freighter.StreamTransport[sampleRequest, sampleResponse]
	sampleServerStream = freighter.ServerStream[sampleRequest, sampleResponse]
)

type implementation interface {
	start(host address.Address, logger *zap.SugaredLogger) sampleTransport
	stop() error
}

var implementations = []implementation{
	&wsImplementation{},
	&mockImplementation{},
}

var _ = Describe("StreamTransport", Ordered, Serial, func() {
	for _, impl := range implementations {
		impl := impl
		var (
			addr address.Address
			t    sampleTransport
		)
		BeforeAll(func() {
			addr = "localhost:8080"
			l := zap.NewNop()
			t = impl.start(addr, l.Sugar())
		})
		AfterAll(func() {
			Expect(impl.stop()).ToNot(HaveOccurred())
		})
		Describe("Normal Operation", func() {

			It("Should exchange messages between a client and a server", func() {
				closed := make(chan struct{})

				t.BindHandler(func(ctx context.Context, server sampleServerStream) error {
					defer GinkgoRecover()
					defer close(closed)
					for {
						req, err := server.Receive()
						if err != nil {
							By("Receiving a transport EOF error from the client")
							Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
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
				client, err := t.Stream(ctx, addr)
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
				Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
				Eventually(closed).Should(BeClosed())
			})

			It("Should allow the server to continue sending messages after CloseSend is called", func() {
				serverClosed := make(chan struct{})
				t.BindHandler(func(ctx context.Context, server sampleServerStream) error {
					defer GinkgoRecover()
					defer close(serverClosed)
					_, err := server.Receive()
					Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
					Expect(server.Send(sampleResponse{ID: 1, Message: "Hello"})).To(Succeed())
					return nil
				})
				client, err := t.Stream(context.TODO(), addr)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.CloseSend()).To(Succeed())
				msg, err := client.Receive()
				Expect(err).ToNot(HaveOccurred())
				Expect(msg.ID).To(Equal(1))
				Expect(msg.Message).To(Equal("Hello"))
				_, err = client.Receive()
				Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
				Eventually(serverClosed).Should(BeClosed())
			})

		})
		Describe("Provider Handling", func() {
			Describe("Server returns a non-nil error", func() {
				It("Should send the error to the client", func() {
					serverClosed := make(chan struct{})
					t.BindHandler(func(ctx context.Context, server sampleServerStream) error {
						defer GinkgoRecover()
						defer close(serverClosed)
						_, err := server.Receive()
						if err != nil {
							Fail(err.Error())
						}
						return errors.New("zero is not allowed!")
					})
					client, err := t.Stream(context.TODO(), addr)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.Send(sampleRequest{ID: 0, Message: "Hello"})).To(Succeed())
					_, err = client.Receive()
					Expect(errors.Is(err, errors.New("zero is not allowed!"))).To(BeTrue())
					Eventually(serverClosed).Should(BeClosed())
				})

				Specify("If the client calls Send, if should return an EOF error", func() {
					serverClosed := make(chan struct{})
					t.BindHandler(func(ctx context.Context, server sampleServerStream) error {
						defer GinkgoRecover()
						defer close(serverClosed)
						_, err := server.Receive()
						if err != nil {
							Fail(err.Error())
						}
						return errors.New("zero is not allowed!")
					})
					client, err := t.Stream(context.TODO(), addr)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.Send(sampleRequest{ID: 0, Message: "Hello"})).To(Succeed())
					_, err = client.Receive()
					Expect(errors.Is(err, errors.New("zero is not allowed!"))).To(BeTrue())
					err = client.Send(sampleRequest{ID: 0, Message: "Hello"})
					Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
					Eventually(serverClosed).Should(BeClosed())
				})

			})

			Describe("Client cancels the context", func() {
				It("Should propagate the context cancellation to both the server and the client", func() {
					serverClosed := make(chan struct{})
					t.BindHandler(func(ctx context.Context, server sampleServerStream) error {
						defer close(serverClosed)
						defer GinkgoRecover()
						_, err := server.Receive()
						Expect(err).To(Equal(context.Canceled))
						return nil
					})
					ctx, cancel := context.WithCancel(context.TODO())
					client, err := t.Stream(ctx, addr)
					Expect(err).ToNot(HaveOccurred())
					cancel()
					_, err = client.Receive()
					Expect(errors.Is(err, context.Canceled)).To(BeTrue())
					Eventually(serverClosed).Should(BeClosed())
				})
			})

			Describe("Client attempts to send a message after calling close send", func() {
				It("Should return a StreamClosed error", func() {
					serverClosed := make(chan struct{})
					t.BindHandler(func(ctx context.Context, server sampleServerStream) error {
						defer close(serverClosed)
						defer GinkgoRecover()
						_, err := server.Receive()
						Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
						return nil
					})

					ctx, cancel := context.WithCancel(context.TODO())
					defer cancel()

					client, err := t.Stream(ctx, addr)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.CloseSend()).To(Succeed())
					err = client.Send(sampleRequest{ID: 0, Message: "Hello"})
					Expect(errors.Is(err, freighter.StreamClosed)).To(BeTrue())

					_, err = client.Receive()

					Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
					Eventually(serverClosed).Should(BeClosed())
				})

			})
		})

		Describe("Client attempts to send a message after the server closes", func() {
			It("Should return a EOF error", func() {
				serverClosed := make(chan struct{})
				t.BindHandler(func(ctx context.Context, server sampleServerStream) error {
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
				client, err := t.Stream(ctx, addr)
				Expect(err).ToNot(HaveOccurred())
				i := 0
				for {
					err = client.Send(sampleRequest{ID: 0, Message: "Hello"})
					if err != nil {
						Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
						break
					}
					Expect(err).ToNot(HaveOccurred())
					i++
				}
				Eventually(serverClosed).Should(BeClosed())
			})
		})
	}
})

type wsImplementation struct {
	app *fiber.App
}

func (impl *wsImplementation) start(
	host address.Address,
	logger *zap.SugaredLogger,
) sampleTransport {
	impl.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	t := fws.New[sampleRequest, sampleResponse](
		&httputil.MsgPackEncoderDecoder{},
		logger,
	)
	t.BindTo(impl.app, "/")
	go func() {
		if err := impl.app.Listen(host.PortString()); err != nil {
			logger.Error(err)
		}
	}()
	return t
}

func (impl *wsImplementation) stop() error {
	return impl.app.Shutdown()
}

type mockImplementation struct {
	net *fmock.Network[sampleRequest, sampleResponse]
}

func (impl *mockImplementation) start(
	host address.Address,
	logger *zap.SugaredLogger,
) sampleTransport {
	impl.net = fmock.NewNetwork[sampleRequest, sampleResponse]()
	return impl.net.RouteStream(host, 10)
}

func (impl *mockImplementation) stop() error { return nil }
