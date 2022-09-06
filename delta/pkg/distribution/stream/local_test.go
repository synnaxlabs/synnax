package stream_test

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/stream"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/x/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go/types"
)

type mockTransport struct {
	reader stream.ReadTransport
	writer stream.WriteTransport
}

func (m mockTransport) Reader() stream.ReadTransport { return m.reader }

func (m mockTransport) Writer() stream.WriteTransport { return m.writer }

var _ = Describe("Local", func() {
	var (
		svc     *stream.Service
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

		readerNet := fmock.NewNetwork[stream.ReadRequest, stream.ReadResponse]()
		writeNet := fmock.NewNetwork[stream.WriteRequest, types.Nil]()
		trans := mockTransport{
			reader: readerNet.RouteStream("", 1),
			writer: writeNet.RouteStream("", 1),
		}
		svc = stream.Open(stream.Config{
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
		reader, _ := svc.NewStreamReader(cKey)
		samples := []stream.Sample{{
			ChannelKey: cKey,
			Stamp:      1,
			Value:      []byte{1},
		}}
		w.Inlet() <- samples
		Eventually(reader.Outlet()).Should(Receive(Equal(samples)))
	})
})
