package writer_test

import (
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/telem"
	. "github.com/arya-analytics/x/testutil"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
	"go.uber.org/zap"
	"time"

	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
)

var _ = Describe("Local", Ordered, func() {
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
		log = zap.NewNop()
		builder, services = provisionNServices(1, log)
		dataFactory := &seg.RandomFloat64Factory{Cache: true}
		channels, err := services[1].channel.NewCreate().
			WithName("SG02").
			WithDataRate(25*telem.Hz).
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
			w.Requests() <- writer.Request{Segments: wrapper.Wrap(seg)}
			close(w.Requests())
			for res := range w.Responses() {
				Expect(res.Error).ToNot(HaveOccurred())
			}
			Expect(w.Close()).To(Succeed())
		})
		It("Should write multiple segments to disk", func() {
			seg := factory.NextN(10)
			w.Requests() <- writer.Request{Segments: wrapper.Wrap(seg)}
			close(w.Requests())
			for res := range w.Responses() {
				Expect(res.Error).ToNot(HaveOccurred())
			}
			Expect(w.Close()).To(Succeed())
		})
		It("Should return an error when another writer has a lock on the channel", func() {
			_, err := newWriter()
			Expect(err).To(HaveOccurredAs(errors.New("[cesium] - lock already held")))
		})
	})
})
