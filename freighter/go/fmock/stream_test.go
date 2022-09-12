package fmock_test

import (
	"context"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fmock"
	"sync"
)

var _ = Describe("StreamTransport", func() {
	It("Should correctly exchange messages between a client and server", func() {
		client, server := fmock.NewStreamPair[int, int](context.Background(), 1)
		Expect(client.Send(1)).To(Succeed())
		v, err := server.Receive()
		Expect(err).ToNot(HaveOccurred())
		Expect(v).To(Equal(1))

		Expect(client.Send(2)).To(Succeed())
		v, err = server.Receive()
		Expect(err).ToNot(HaveOccurred())
		Expect(v).To(Equal(2))

		Expect(client.CloseSend()).To(Succeed())

		v, err = server.Receive()
		Expect(err).To(Equal(freighter.EOF))
	})

	It("Should close both client and server streams when the context is cancelled", func() {
		ctx, cancel := context.WithCancel(ctx)
		client, server := fmock.NewStreamPair[int, int](ctx, 1)

		Expect(client.Send(1)).To(Succeed())
		cancel()

		v, err := server.Receive()
		Expect(err).To(Equal(context.Canceled))
		Expect(v).To(BeZero())

		v, err = client.Receive()
		Expect(err).To(Equal(context.Canceled))
		Expect(v).To(BeZero())
	})

	It("Should return a freighter.EOF to the client when the server handler exists normally", func() {
		client, server := fmock.NewStreamPair[int, int](context.TODO(), 1)

		go server.Exec(context.Background(), func(ctx context.Context, srv freighter.ServerStream[int, int]) error {
			return nil
		})

		v, err := client.Receive()
		Expect(err).To(Equal(freighter.EOF))
		Expect(v).To(BeZero())
	})

	It("Should return the server error to the client when the server handler returns an error", func() {
		client, server := fmock.NewStreamPair[int, int](context.TODO())

		srvErr := errors.New("test error")

		go server.Exec(context.Background(), func(ctx context.Context, srv freighter.ServerStream[int, int]) error {
			return srvErr
		})

		v, err := client.Receive()
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, srvErr)).To(BeTrue())
		Expect(v).To(BeZero())

		v, err = client.Receive()
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, srvErr)).To(BeTrue())
		Expect(v).To(BeZero())
	})

	It("Should return a StreamClosed error when the client attempts to send a message after the CloseSend is called", func() {
		client, _ := fmock.NewStreamPair[int, int](context.TODO(), 1)
		Expect(client.CloseSend()).To(Succeed())
		Expect(client.Send(1)).To(MatchError(freighter.StreamClosed))
	})

	It("Should allow the client to continue receiving messages after the CloseSend is called", func() {
		client, server := fmock.NewStreamPair[int, int](context.TODO(), 1)
		Expect(server.Send(1)).To(Succeed())
		Expect(client.CloseSend()).To(Succeed())
		Expect(client.Send(1)).To(MatchError(freighter.StreamClosed))
		v, err := client.Receive()
		Expect(err).ToNot(HaveOccurred())
		Expect(v).To(Equal(1))
	})

	It("Should allow the server to continue sending messages after the CloseSend is called", func() {
		client, server := fmock.NewStreamPair[int, int](context.TODO(), 1)
		Expect(client.CloseSend()).To(Succeed())
		Expect(server.Send(1)).To(Succeed())
		v, err := client.Receive()
		Expect(err).ToNot(HaveOccurred())
		Expect(v).To(Equal(1))
	})

	It("Should return a StreamClosed error when the server attempts to send or receive a message after the handler exits", func() {
		_, server := fmock.NewStreamPair[int, int](context.TODO(), 1)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go server.Exec(context.Background(), func(ctx context.Context, srv freighter.ServerStream[int, int]) error {
			defer wg.Done()
			return nil
		})
		wg.Wait()
		Expect(server.Send(1)).To(MatchError(freighter.StreamClosed))
		res, err := server.Receive()
		Expect(err).To(MatchError(freighter.StreamClosed))
		Expect(res).To(BeZero())
	})

	It("Should return an EOF error when client attempts to send a message after the handler exits", func() {
		client, server := fmock.NewStreamPair[int, int](context.TODO(), 1)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go server.Exec(context.Background(), func(ctx context.Context, srv freighter.ServerStream[int, int]) error {
			defer wg.Done()
			return nil
		})
		wg.Wait()
		// The behavior here mimics the behavior of an actual network transport,
		// where we may be able to put messages into the buffer even after the
		// server exited.
		err := client.Send(1)
		if err == freighter.EOF || err == nil {
			return
		}
		if err = client.Send(1); err == nil {
			return
		}
		Fail("Expected EOF error")
	})

})
