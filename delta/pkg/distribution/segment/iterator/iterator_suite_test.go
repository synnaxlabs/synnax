package iterator_test

import (
	"context"
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/x/telem"
	"go.uber.org/zap"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	ctx = context.Background()
)

func TestIterator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Iterator Suite")
}

type serviceContainer struct {
	channel   *channel.Service
	transport struct {
		channel channel.CreateTransport
		iter    iterator.Transport
	}
}

func provisionNServices(n int, logger *zap.Logger) (*mock.CoreBuilder, map[core.NodeID]serviceContainer) {
	builder := mock.NewCoreBuilder(core.Config{Logger: logger, Storage: storage.Config{MemBacked: true}})
	services := make(map[core.NodeID]serviceContainer)
	channelNet := fmock.NewNetwork[channel.CreateMessage, channel.CreateMessage]()
	iterNet := fmock.NewNetwork[iterator.Request, iterator.Response]()
	for i := 0; i < n; i++ {
		_core := builder.New()
		var container serviceContainer
		container.transport.channel = channelNet.RouteUnary(_core.Config.AdvertiseAddress)
		container.transport.iter = iterNet.RouteStream(_core.Config.AdvertiseAddress, 0)
		container.channel = channel.New(
			_core.Cluster,
			_core.Storage.Gorpify(),
			_core.Storage.TS,
			container.transport.channel,
		)
		iterator.NewServer(_core.Storage.TS, _core.Cluster.HostID(), container.transport.iter)
		services[_core.Cluster.HostID()] = container
	}
	return builder, services
}

func writeMockData(
	builder *mock.CoreBuilder,
	segmentSize telem.TimeSpan,
	numberOfRequests, numberOfSegmentsPerRequest int,
	channels ...channel.Channel,
) {
	dataFactory := &seg.RandomFloat64Factory{Cache: true}
	for _, ch := range channels {
		req, res, err := builder.Cores[ch.NodeID].Storage.TS.NewCreate().WhereChannels(ch.Key().StorageKey()).Stream(ctx)
		Expect(err).ToNot(HaveOccurred())
		stc := &seg.StreamCreate{
			Req:               req,
			Res:               res,
			SequentialFactory: seg.NewSequentialFactory(dataFactory, segmentSize, ch.Channel),
		}
		stc.CreateCRequestsOfN(numberOfRequests, numberOfSegmentsPerRequest)
		Expect(stc.CloseAndWait()).ToNot(HaveOccurred())
	}
}

func openIter(
	nodeID core.NodeID,
	services map[core.NodeID]serviceContainer,
	builder *mock.CoreBuilder,
	keys channel.Keys,
) iterator.Iterator {
	iter, err := iterator.New(
		ctx,
		builder.Cores[nodeID].Storage.TS,
		services[nodeID].channel,
		builder.Cores[nodeID].Cluster,
		services[nodeID].transport.iter,
		telem.TimeRangeMax,
		keys,
		/* sync */ false,
		/* sendAcknowledge */ false,
		zap.L(),
	)
	Expect(err).ToNot(HaveOccurred())
	return iter
}

// sensibleTimeoutThreshold is the amount of time we allow for an iterator to return the expected number of segments
// before we automatically fail the test.
const sensibleTimeoutThreshold = 20 * time.Millisecond
