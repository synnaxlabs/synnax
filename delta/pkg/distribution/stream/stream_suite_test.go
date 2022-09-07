package stream_test

import (
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/stream"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/x/config"
	"go.uber.org/zap"
	"go/types"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStream(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stream Suite")
}

func provisionNServices(n int, logger *zap.Logger) (*mock.CoreBuilder, map[distribcore.NodeID]*stream.Service) {
	builder := mock.NewCoreBuilder(distribcore.Config{Logger: logger, Storage: storage.Config{MemBacked: config.BoolPointer(true)}})
	services := make(map[distribcore.NodeID]*stream.Service)
	readerNet := fmock.NewNetwork[stream.ReadRequest, stream.ReadResponse]()
	writerNet := fmock.NewNetwork[stream.WriteRequest, types.Nil]()
	for i := 0; i < n; i++ {
		_core := builder.New()
		services[_core.Cluster.HostID()] = stream.Open(stream.Config{
			Resolver: _core.Cluster,
			Transport: mockTransport{
				writer: writerNet.RouteStream(_core.Config.AdvertiseAddress, 1),
				reader: readerNet.RouteStream(_core.Config.AdvertiseAddress, 1),
			},
			Logger: logger,
		})
	}
	return builder, services
}
