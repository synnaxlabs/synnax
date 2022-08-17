package freightfluence_test

import (
	"context"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/freighter/freightfluence"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func clientStreamsToSlice(
	clients map[address.Address]freighter.StreamSenderCloser[int],
) []freighter.StreamSenderCloser[int] {
	slice := make([]freighter.StreamSenderCloser[int], 0, len(clients))
	for _, client := range clients {
		slice = append(slice, client)
	}
	return slice
}

var _ = Describe("Sender", func() {
	var net *fmock.Network[int, int]
	BeforeEach(func() {
		net = fmock.NewNetwork[int, int]()
	})
	Context("Single StreamTransport", func() {
		var (
			streamTransport freighter.Stream[int, int]
			receiverStream  confluence.Stream[int]
			senderStream    confluence.Stream[int]
		)
		BeforeEach(func() {
			streamTransport = net.RouteStream("", 10)
			receiverStream = confluence.NewStream[int](0)
			senderStream = confluence.NewStream[int](0)
			streamTransport.BindHandler(func(ctx context.Context, server freighter.ServerStream[int, int]) error {
				sCtx, cancel := signal.WithCancel(ctx)
				defer cancel()
				receiver := &freightfluence.Receiver[int]{}
				receiver.Receiver = server
				receiver.OutTo(receiverStream)
				receiver.Flow(sCtx, confluence.CloseInletsOnExit())
				return sCtx.Wait()
			})
		})
		Describe("Sender", func() {
			It("Should operate correctly", func() {
				sCtx, cancel := signal.WithCancel(context.TODO())
				client, err := streamTransport.Stream(sCtx, "localhost:0")
				Expect(err).ToNot(HaveOccurred())
				sender := &freightfluence.Sender[int]{Sender: client}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 1
				v := <-receiverStream.Outlet()
				Expect(v).To(Equal(1))
				senderStream.Inlet() <- 2
				v = <-receiverStream.Outlet()
				cancel()
				Expect(sCtx.Wait()).To(Equal(context.Canceled))
				_, ok := <-receiverStream.Outlet()
				Expect(ok).To(BeFalse())
			})
		})
		Describe("TransformSender", func() {
			It("Should transform values before sending them", func() {
				sCtx, cancel := signal.WithCancel(context.TODO())
				client, err := streamTransport.Stream(sCtx, "localhost:0")
				Expect(err).ToNot(HaveOccurred())
				sender := &freightfluence.TransformSender[int, int]{}
				sender.Sender = client
				sender.ApplyTransform = func(ctx context.Context, v int) (int, bool, error) {
					return v * 2, true, nil
				}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 1
				v := <-receiverStream.Outlet()
				Expect(v).To(Equal(2))
				cancel()
				Expect(sCtx.Wait()).To(Equal(context.Canceled))
				_, ok := <-receiverStream.Outlet()
				Expect(ok).To(BeFalse())
			})
			It("Should exit when the transform returns an error", func() {
				sCtx, cancel := signal.WithCancel(context.TODO())
				defer cancel()
				client, err := streamTransport.Stream(sCtx, "localhost:0")
				Expect(err).ToNot(HaveOccurred())
				sender := &freightfluence.TransformSender[int, int]{}
				sender.Sender = client
				sender.ApplyTransform = func(ctx context.Context, v int) (int, bool, error) {
					return v * 2, true, errors.New("error")
				}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 1
				Expect(sCtx.Wait()).To(MatchError("error"))
			})
		})
	})
	Context("Multiple Streams", func() {
		var (
			sCtx            signal.Context
			cancel          context.CancelFunc
			nStreams        = 5
			senderStream    confluence.Stream[int]
			receiverStreams map[address.Address]confluence.Stream[int]
			clientStreams   map[address.Address]freighter.StreamSenderCloser[int]
		)
		BeforeEach(func() {
			sCtx, cancel = signal.WithCancel(context.TODO())
			senderStream = confluence.NewStream[int](nStreams)
			clientTransport := net.RouteStream("", 0)
			clientStreams = make(map[address.Address]freighter.StreamSenderCloser[int], nStreams)
			receiverStreams = make(map[address.Address]confluence.Stream[int], nStreams)
			for i := 0; i < nStreams; i++ {
				stream := net.RouteStream("", 0)
				receiverStream := confluence.NewStream[int](1)
				stream.BindHandler(func(ctx context.Context, server freighter.ServerStream[int, int]) error {
					serverCtx, cancel := signal.WithCancel(ctx)
					defer cancel()
					receiver := &freightfluence.Receiver[int]{}
					receiver.Receiver = server
					receiver.OutTo(receiverStream)
					receiver.Flow(serverCtx, confluence.CloseInletsOnExit())
					return serverCtx.Wait()
				})
				clientStream, err := clientTransport.Stream(sCtx, stream.Address)
				Expect(err).ToNot(HaveOccurred())
				clientStreams[stream.Address] = clientStream
				receiverStreams[stream.Address] = receiverStream
			}
		})
		AfterEach(func() { cancel() })
		Describe("MultiSender", func() {
			It("Should forward values to all streams", func() {
				sender := &freightfluence.MultiSender[int]{}
				sender.Senders = clientStreamsToSlice(clientStreams)
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 2
				for addr := range clientStreams {
					v := <-receiverStreams[addr].Outlet()
					Expect(v).To(Equal(2))
				}
				senderStream.Close()
				Expect(sCtx.Wait()).To(Succeed())
			})
			It("Should exit when the context is canceled", func() {
				sender := &freightfluence.MultiSender[int]{}
				sender.Senders = clientStreamsToSlice(clientStreams)
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 2
				cancel()
				Expect(sCtx.Wait()).To(Equal(context.Canceled))
			})
		})
		Describe("SwitchSender", func() {
			It("Should route values to the correct stream", func() {
				sender := &freightfluence.SwitchSender[int]{}
				sender.Senders = clientStreams
				sender.ApplySwitch = func(ctx context.Context, v int) (address.Address, bool, error) {
					addr := address.Newf("localhost:%v", v)
					return addr, true, nil
				}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				for i := 1; i < nStreams+1; i++ {
					senderStream.Inlet() <- i
					addr := address.Newf("localhost:%v", i)
					v := <-receiverStreams[addr].Outlet()
					Expect(v).To(Equal(i))
				}
				senderStream.Close()
				Expect(sCtx.Wait()).To(Succeed())
			})
			It("Should exit when the context is canceled", func() {
				sender := &freightfluence.SwitchSender[int]{}
				sender.Senders = clientStreams
				sender.ApplySwitch = func(ctx context.Context, v int) (address.Address, bool, error) {
					addr := address.Newf("localhost:%v", v)
					return addr, true, nil
				}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 1
				cancel()
				Expect(sCtx.Wait()).To(Equal(context.Canceled))
			})
			It("Should exit when the switch returns an error", func() {
				sender := &freightfluence.SwitchSender[int]{}
				sender.Senders = clientStreams
				sender.ApplySwitch = func(ctx context.Context, v int) (address.Address, bool, error) {
					return "", false, errors.New("error")
				}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 1
				Expect(sCtx.Wait()).To(HaveOccurred())
			})
		})
		Describe("BatchSwitchSender", func() {
			It("Should route values to the correct stream", func() {
				sender := &freightfluence.BatchSwitchSender[int, int]{}
				sender.Senders = clientStreams
				sender.ApplySwitch = func(ctx context.Context, v int, o map[address.Address]int) error {
					addr := address.Newf("localhost:%v", v)
					o[addr] = v
					addr2 := address.Newf("localhost:%v", v+1)
					o[addr2] = v + 1
					return nil
				}
				sender.InFrom(senderStream)
				sender.Flow(sCtx)
				senderStream.Inlet() <- 2
				Expect(<-receiverStreams["localhost:2"].Outlet()).To(Equal(2))
				Expect(<-receiverStreams["localhost:3"].Outlet()).To(Equal(3))
				senderStream.Close()
				Expect(sCtx.Wait()).To(Succeed())
			})
		})
		It("Should exit when the context is canceled", func() {
			sender := &freightfluence.BatchSwitchSender[int, int]{}
			sender.Senders = clientStreams
			sender.ApplySwitch = func(ctx context.Context, v int, o map[address.Address]int) error {
				addr := address.Newf("localhost:%v", v)
				o[addr] = v
				return nil
			}
			sender.InFrom(senderStream)
			sender.Flow(sCtx)
			senderStream.Inlet() <- 2
			cancel()
			Expect(sCtx.Wait()).To(MatchError(context.Canceled))
		})
		It("Should exit when the switch returns an error", func() {
			sender := &freightfluence.BatchSwitchSender[int, int]{}
			sender.Senders = clientStreams
			sender.ApplySwitch = func(ctx context.Context, v int, o map[address.Address]int) error {
				return errors.New("error")
			}
			sender.InFrom(senderStream)
			sender.Flow(sCtx)
			senderStream.Inlet() <- 2
			Expect(sCtx.Wait()).To(MatchError("error"))
		})
	})
})
