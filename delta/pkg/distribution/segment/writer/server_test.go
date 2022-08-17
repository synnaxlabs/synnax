package writer_test

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
	"go.uber.org/zap"
	"time"
)

func openClient(ctx context.Context, id distribution.NodeID, services map[distribution.NodeID]serviceContainer) writer.Client {
	client, err := services[id].transport.writer.Stream(ctx, "localhost:0")
	Expect(err).ToNot(HaveOccurred())
	return client
}

func openRequest(client writer.Client, keys channel.Keys) (writer.Response, error) {
	Expect(client.Send(writer.Request{OpenKeys: keys})).To(Succeed())
	Expect(client.CloseSend()).To(Succeed())
	return client.Receive()
}

var _ = Describe("Server", func() {
	var (
		log      *zap.Logger
		services map[distribcore.NodeID]serviceContainer
		builder  *mock.CoreBuilder
	)
	BeforeEach(func() {
		log = zap.NewNop()
		builder, services = provisionNServices(1, log)
		_, err := services[1].channel.NewCreate().
			WithName("SG02").
			WithDataRate(1*telem.Hz).
			WithDataType(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
	})
	BeforeEach(func() {
		routines := gleak.Goroutines()
		DeferCleanup(func() {
			Eventually(gleak.Goroutines).WithTimeout(time.Second).ShouldNot(gleak.HaveLeaked(routines))
		})
	})
	AfterEach(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	DescribeTable("Open Request", func(keys channel.Keys, expectedResError, expectedTransportError error) {
		res, err := openRequest(openClient(ctx, 1, services), keys)
		Expect(errors.Is(res.Error, expectedResError)).To(BeTrue())
		Expect(errors.Is(err, expectedTransportError)).To(BeTrue())
	},
		Entry("Open the writer properly when the keys exist", channel.Keys{channel.NewKey(1, 1)}, nil, freighter.EOF),
		Entry("Return an error when no keys are provided", channel.Keys{}, nil, errors.New("[segment.w] - server expected OpenKeys to be defined")),
		Entry("Return an error when invalid keys are provided", channel.Keys{channel.NewKey(1, 2)}, nil, query.NotFound),
	)
	Describe("Write Request", func() {
		It("Should immediately abort all operations when the context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.TODO())
			client := openClient(ctx, 1, services)
			Expect(client.Send(writer.Request{OpenKeys: channel.Keys{channel.NewKey(1, 1)}})).To(Succeed())
			var s core.Segment
			s.Data = []byte{1, 2, 3}
			s.Start = telem.TimeStamp(25)
			Expect(client.Send(writer.Request{Segments: []core.Segment{s}})).To(Succeed())
			cancel()
			res, err := client.Receive()
			Expect(res.Error).To(BeNil())
			Expect(errors.Is(err, context.Canceled)).To(BeTrue())
		})
		Describe("No Cancellation", func() {
			var client writer.Client
			BeforeEach(func() {
				client = openClient(ctx, 1, services)
				Expect(client.Send(writer.Request{OpenKeys: channel.Keys{channel.NewKey(1, 1)}})).To(Succeed())
			})
			It("Should execute a valid write request", func() {
				var s core.Segment
				s.ChannelKey = channel.NewKey(1, 1)
				s.Data = []byte{1, 2, 3}
				s.Start = telem.TimeStamp(25)
				Expect(client.Send(writer.Request{Segments: []core.Segment{s}})).To(Succeed())
				Expect(client.CloseSend()).To(Succeed())
				res, err := client.Receive()
				Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
				Expect(res.Error).ToNot(HaveOccurred())
			})
			It("Should return an error when the write request has no channel key", func() {
				var s core.Segment
				s.Data = []byte{1, 2, 3}
				s.Start = telem.TimeStamp(25)
				Expect(client.Send(writer.Request{Segments: []core.Segment{s}})).To(Succeed())
				Expect(client.CloseSend()).To(Succeed())
				res, err := client.Receive()
				Expect(errors.Is(res.Error, query.NotFound)).To(BeTrue())
				Expect(err).To(BeNil())
				res, err = client.Receive()
				Expect(res.Error).To(BeNil())
				Expect(errors.Is(err, freighter.EOF)).To(BeTrue())
			})
		})

	})
})
