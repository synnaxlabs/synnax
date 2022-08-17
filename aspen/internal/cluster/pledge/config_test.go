package pledge_test

import (
	"github.com/arya-analytics/aspen/internal/cluster/pledge"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Config", func() {

	Describe("Validation", func() {

		It("Should return an assertion failed when the transport is nil", func() {
			cfg := &pledge.Config{
				Transport: nil,
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("[pledge] - transport required"))
		})

	})

	Describe("Report", func() {

		It("Should generate a valid configuration report", func() {
			cfg := &pledge.Config{
				RequestTimeout: 5 * time.Millisecond,
			}
			Expect(cfg.Report()["requestTimeout"]).To(Equal(5 * time.Millisecond))
		})

	})

})
