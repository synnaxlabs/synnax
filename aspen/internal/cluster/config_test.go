package cluster_test

import (
	"github.com/arya-analytics/aspen/internal/cluster"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	Describe("Report", func() {
		It("Should return a report of the config", func() {
			cfg := cluster.Config{
				StorageKey: []byte("aspen.cluster"),
			}
			rep := cfg.Report()
			Expect(rep["storageKey"]).To(Equal("aspen.cluster"))
		})

	})

})
