package writer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/testutil/seg"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"time"

	"github.com/synnaxlabs/synnax/pkg/distribution/segment/writer"
)

var _ = FDescribe("Local", Ordered, func() {
	var (
		log       *zap.Logger
		w         writer.Writer
		services  map[distribcore.NodeID]serviceContainer
		builder   *mock.CoreBuilder
		factory   seg.SequentialFactory
		wrapper   *core.StorageWrapper
		keys      channel.Keys
		newWriter func() (writer.Writer, error)
	)
	BeforeAll(func() {
		log = lo.Must(zap.NewDevelopment())
		builder, services = provisionNServices(1, log)
		dataFactory := &seg.RandomFloat64Factory{Cache: true}
		channels, err := services[1].channel.NewCreate().
			WithName("SG02").
			WithRate(25*telem.Hz).
			WithDataType(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		factory = seg.NewSequentialFactory(dataFactory, 10*telem.Second, channels[0].Channel)
		wrapper = &core.StorageWrapper{Host: 1}
		keys = channel.Keys{channels[0].Key()}
		newWriter = func() (writer.Writer, error) { return openWriter(1, services, builder, keys, log) }
	})
	BeforeEach(func() {
		var err error
		w, err = newWriter()
		Expect(err).ToNot(HaveOccurred())
		routines := gleak.Goroutines()
		DeferCleanup(func() {
			Eventually(gleak.Goroutines).WithTimeout(time.Second).ShouldNot(gleak.HaveLeaked(routines))
		})
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Context("Behavioral Accuracy", func() {
		It("Should write a segment to disk", func() {
			seg := factory.NextN(1)
			Expect(w.Write(wrapper.Wrap(seg))).To(BeTrue())
			Expect(w.Commit()).To(BeTrue())
			Expect(w.Close()).To(Succeed())
		})
		It("Should write multiple segments to disk", func() {
			seg := factory.NextN(10)
			Expect(w.Write(wrapper.Wrap(seg))).To(BeTrue())
			Expect(w.Commit()).To(BeTrue())
			Expect(w.Close()).To(Succeed())
		})
		It("Should return an error when another writerClient has a lock on the channelClient", func() {
			_, err := newWriter()
			Expect(err).To(HaveOccurredAs(cesium.ErrWriteLock))
		})
	})
})
