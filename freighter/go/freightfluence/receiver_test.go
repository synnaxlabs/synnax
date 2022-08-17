package freightfluence_test

import (
	"context"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/freighter/freightfluence"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Receiver", func() {
	var (
		net    *fmock.Network[int, int]
		stream freighter.Stream[int, int]
	)
	BeforeEach(func() {
		net = fmock.NewNetwork[int, int]()
		stream = net.RouteStream("", 10)
	})
	Describe("Receiver", func() {
		It("Should operate correctly", func() {
			var receivedValues []int
			receiverStream := confluence.NewStream[int](10)
			stream.BindHandler(func(ctx context.Context, server freighter.ServerStream[int, int]) error {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				receiver := &freightfluence.Receiver[int]{Receiver: server}
				receiver.OutTo(receiverStream)
				receiver.Flow(sCtx, confluence.CloseInletsOnExit())
				By("Receiving values from the input stream")
				receivedValues = append(receivedValues, <-receiverStream.Outlet())
				return sCtx.Wait()
			})
			client, err := stream.Stream(context.TODO(), "localhost:0")
			Expect(err).ToNot(HaveOccurred())
			Expect(client.Send(1)).To(Succeed())
			Expect(client.CloseSend()).To(Succeed())
			By("Closing the network pipe on return")
			_, err = client.Receive()
			Expect(err).To(Equal(freighter.EOF))
			Expect(receivedValues).To(Equal([]int{1}))
			By("Closing the receive stream on exit")
			_, ok := <-receiverStream.Outlet()
			Expect(ok).To(BeFalse())
		})
		It("Should exit the receiver on context cancellation", func() {
			receiverStream := confluence.NewStream[int](10)
			stream.BindHandler(func(ctx context.Context, server freighter.ServerStream[int, int]) error {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				receiver := &freightfluence.Receiver[int]{Receiver: server}
				receiver.OutTo(receiverStream)
				receiver.Flow(sCtx, confluence.CloseInletsOnExit())
				By("Receiving values from the input stream")
				return sCtx.Wait()
			})
			ctx, cancel := context.WithCancel(context.TODO())
			client, err := stream.Stream(ctx, "localhost:0")
			Expect(err).ToNot(HaveOccurred())
			Expect(client.Send(1)).To(Succeed())
			By("Closing the network pipe")
			v := <-receiverStream.Outlet()
			Expect(v).To(Equal(1))
			cancel()
			_, err = client.Receive()
			Expect(err).To(Equal(context.Canceled))
			By("Closing the receive stream on exit")
			_, ok := <-receiverStream.Outlet()
			Expect(ok).To(BeFalse())
		})
	})
	Describe("TransformReceiver", func() {
		It("It should transform values before sending them through the channel", func() {
			var receivedValues []int
			receiverStream := confluence.NewStream[int](10)
			stream.BindHandler(func(ctx context.Context, server freighter.ServerStream[int, int]) error {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				receiver := &freightfluence.TransformReceiver[int, int]{}
				receiver.Receiver = server
				receiver.OutTo(receiverStream)
				receiver.ApplyTransform = func(ctx context.Context, v int) (int, bool, error) {
					return v * 2, true, nil
				}
				receiver.Flow(sCtx, confluence.CloseInletsOnExit())
				By("Receiving values from the input stream")
				receivedValues = append(receivedValues, <-receiverStream.Outlet())
				return sCtx.Wait()
			})
			client, err := stream.Stream(context.TODO(), "localhost:0")
			Expect(err).ToNot(HaveOccurred())
			Expect(client.Send(1)).To(Succeed())
			Expect(client.CloseSend()).To(Succeed())
			By("Closing the network pipe on return")
			_, err = client.Receive()
			Expect(err).To(Equal(freighter.EOF))
			Expect(receivedValues).To(Equal([]int{2}))
			By("Closing the receive stream on exit")
			_, ok := <-receiverStream.Outlet()
			Expect(ok).To(BeFalse())
		})
	})
})
