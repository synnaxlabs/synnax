package writer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"time"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
)

var _ = FDescribe("Local", Ordered, func() {
	var (
		log       *zap.Logger
		w         writer.Writer
		services  map[distribcore.NodeID]serviceContainer
		builder   *mock.CoreBuilder
		key       channel.Key
		newWriter func() (writer.Writer, error)
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder, services = provisionNServices(1, log)
		ch := channel.Channel{Name: "SG02", Rate: 25 * telem.Hz, DataType: telem.Float64T, NodeID: 1}
		key = ch.Key()
		Expect(services[1].channel.Create(&ch)).To(Succeed())
		newWriter = func() (writer.Writer, error) { return openWriter(1, services, builder, []channel.Key{ch.Key()}, log) }
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
			Expect(w.Write(core.UnaryFrame(key, telem.NewArrayV[float64](1, 2, 3, 4, 5)))).To(BeTrue())
			Expect(w.Commit()).To(BeTrue())
			Expect(w.Close()).To(Succeed())
		})
	})
})
