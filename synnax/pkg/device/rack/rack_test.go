package rack_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/device/rack"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
)

var _ = Describe("Rack", func() {
	Describe("Key", func() {
		It("Should correctly construct and deconstruct key from its components", func() {
			k := rack.NewKey(1, 2)
			Expect(k.Node()).To(Equal(core.NodeKey(1)))
			Expect(k.LocalKey()).To(Equal(uint16(2)))
		})
	})
})
