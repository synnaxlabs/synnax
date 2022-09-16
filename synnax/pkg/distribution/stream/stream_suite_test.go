package stream_test

import (
	"github.com/synnaxlabs/freighter/fmock"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
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

func provisionNServices(n int, logger *zap.Logger) (*mock.CoreBuilder, map[distribcore.NodeID]*Service) {
	builder := mock.NewCoreBuilder(distribcore.Config{Logger: logger, Storage: storage.Config{MemBacked: config.BoolPointer(true)}})
	services := make(map[distribcore.NodeID]*Service)
	readerNet := fmock.NewNetwork[ReadRequest, ReadResponse]()
	writerNet := fmock.NewNetwork[WriteRequest, types.Nil]()
	for i := 0; i < n; i++ {
		_core := builder.New()
		services[_core.Cluster.HostID()] = Open(Config{
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
