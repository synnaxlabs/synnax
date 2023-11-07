package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology", Ordered, func() {
	var (
		services map[core.NodeKey]channel.Service
		builder  *mock.CoreBuilder
	)
	BeforeAll(func() { builder, services = provisionServices() })
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("OpenNexter", func() {
		It("Should correctly iterate over all channels", func() {
			Expect(services[1].Create(ctx, &channel.Channel{Name: "SG01", DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
			Expect(services[1].Create(ctx, &channel.Channel{Name: "SG02", DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
			Expect(services[1].Create(ctx, &channel.Channel{Name: "SG03", DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
			n := testutil.MustSucceed(services[1].OpenNexter())
			v, ok := n.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(v.Name).To(Equal("SG01"))
			v, ok = n.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(v.Name).To(Equal("SG02"))
			v, ok = n.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(v.Name).To(Equal("SG03"))
			Expect(n.Close()).To(Succeed())
		})
	})

})
