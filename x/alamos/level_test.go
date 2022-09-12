package alamos_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/alamos"
)

var _ = Describe("Level", func() {
	Describe("LevelFilterSet", func() {
		It("Should filterTest out levels not in the set", func() {
			filter := alamos.LevelFilterSet{alamos.Debug}
			Expect(filter.Test(alamos.Perf)).To(BeFalse())
			Expect(filter.Test(alamos.Debug)).To(BeTrue())
		})
	})
	Describe("LevelFilterThreshold", func() {
		It("Should filter out levels above the threshold", func() {
			filter := alamos.LevelFilterThreshold{Below: true, Level: alamos.Debug}
			Expect(filter.Test(alamos.Perf)).To(BeFalse())
			Expect(filter.Test(alamos.Debug)).To(BeTrue())
		})
		It("Should filter out levels below the threshold", func() {
			filter := alamos.LevelFilterThreshold{Below: false, Level: alamos.Perf}
			Expect(filter.Test(alamos.Production)).To(BeTrue())
			Expect(filter.Test(alamos.Perf)).To(BeTrue())
			Expect(filter.Test(alamos.Debug)).To(BeFalse())
		})
	})
})
