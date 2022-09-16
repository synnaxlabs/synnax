package stream_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"go.uber.org/zap"
	"go/types"
)

type mockTransport struct {
	reader ReadTransport
	writer WriteTransport
}

func (m mockTransport) Reader() ReadTransport { return m.reader }

func (m mockTransport) Writer() WriteTransport { return m.writer }

var _ = Describe("Local", func() {
	var (
		svc     *Service
		builder *mock.CoreBuilder
	)
	BeforeEach(func() {
		builder = mock.NewCoreBuilder(core.Config{
			Logger: zap.L(),
			Storage: storage.Config{
				MemBacked: config.BoolPointer(true),
			},
		})
		_core := builder.New()

		readerNet := fmock.NewNetwork[ReadRequest, ReadResponse]()
		writeNet := fmock.NewNetwork[WriteRequest, types.Nil]()
		trans := mockTransport{
			reader: readerNet.RouteStream("", 1),
			writer: writeNet.RouteStream("", 1),
		}
		svc = Open(Config{
			Transport: trans,
			Resolver:  _core.Cluster,
			Logger:    zap.L(),
		})
	})
	AfterEach(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
		Expect(svc.Close()).To(Succeed())
	})

	It("Should route written samples to the output", func() {
		w := svc.NewStreamWriter()
		cKey := channel.NewKey(1, 1)
		reader, _ := svc.NewFilteredStreamReader(cKey)
		samples := []Sample{
			{
				ChannelKey: cKey,
				Stamp:      1,
				Value:      []byte{1},
			},
		}
		w.Inlet() <- samples
		Expect(<-reader.Outlet()).To(Equal(samples))
	})
})
