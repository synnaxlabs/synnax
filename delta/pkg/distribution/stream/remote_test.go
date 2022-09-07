package stream_test

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/stream"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = FDescribe("Remote", func() {
	var (
		builder  *mock.CoreBuilder
		services map[core.NodeID]*stream.Service
	)
	BeforeEach(func() {
		builder, services = provisionNServices(2, zap.NewNop())
	})
	AfterEach(func() {
		for _, s := range services {
			Expect(s.Close()).To(Succeed())
		}
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	It("Should allow a caller to write to one service and read from another", func() {
		key := channel.NewKey(2, 1)
		w := services[1].NewStreamWriter()
		r, closer := services[2].NewStreamReader(key)
		defer closer.Close()

		inSamples := make([]stream.Sample, 10)
		for i := 0; i < 10; i++ {
			inSamples[i] = stream.Sample{
				ChannelKey: key,
				Stamp:      1,
				Value:      []byte("hello"),
			}
		}
		w.Inlet() <- inSamples
		samples := <-r.Outlet()
		Expect(samples).To(HaveLen(10))

	})
})
