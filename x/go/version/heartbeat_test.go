package version_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/version"
)

var _ = Describe("Heartbeat", func() {
	Describe("Increment", func() {
		It("should increment version", func() {
			h := version.Heartbeat{Generation: 1, Version: 1}
			h = h.Increment()
			Expect(h.Version).To(Equal(uint32(2)))
			Expect(h.Generation).To(Equal(uint32(1)))
		})
	})

	Describe("Decrement", func() {
		It("should decrement version", func() {
			h := version.Heartbeat{Generation: 1, Version: 2}
			h = h.Decrement()
			Expect(h.Version).To(Equal(uint32(1)))
			Expect(h.Generation).To(Equal(uint32(1)))
		})
	})

	Describe("Restart", func() {
		It("should increment generation and reset version", func() {
			h := version.Heartbeat{Generation: 1, Version: 5}
			h = h.Restart()
			Expect(h.Generation).To(Equal(uint32(2)))
			Expect(h.Version).To(Equal(uint32(0)))
		})
	})

	Describe("OlderThan", func() {
		It("should compare generations correctly", func() {
			h1 := version.Heartbeat{Generation: 2, Version: 1}
			h2 := version.Heartbeat{Generation: 1, Version: 5}
			Expect(h1.OlderThan(h2)).To(BeTrue())
		})

		It("should compare versions when generations are equal", func() {
			h1 := version.Heartbeat{Generation: 1, Version: 5}
			h2 := version.Heartbeat{Generation: 1, Version: 3}
			Expect(h1.OlderThan(h2)).To(BeTrue())
		})
	})

	Describe("YoungerThan", func() {
		It("should compare generations correctly", func() {
			h1 := version.Heartbeat{Generation: 1, Version: 5}
			h2 := version.Heartbeat{Generation: 2, Version: 1}
			Expect(h1.YoungerThan(h2)).To(BeTrue())
		})

		It("should compare versions when generations are equal", func() {
			h1 := version.Heartbeat{Generation: 1, Version: 3}
			h2 := version.Heartbeat{Generation: 1, Version: 5}
			Expect(h1.YoungerThan(h2)).To(BeTrue())
		})
	})
})
