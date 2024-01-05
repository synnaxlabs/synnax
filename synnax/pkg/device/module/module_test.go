package module_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/device/module"
	"github.com/synnaxlabs/synnax/pkg/device/rack"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
)

var _ = Describe("Module", func() {
	Describe("Key", func() {
		It("Should construct and deconstruct a key from its components", func() {
			rk := rack.NewKey(core.NodeKey(1), 2)
			k := module.NewKey(rk, 2)
			Expect(k.Rack()).To(Equal(rk))
			Expect(k.LocalKey()).To(Equal(uint32(2)))
		})
	})
})
