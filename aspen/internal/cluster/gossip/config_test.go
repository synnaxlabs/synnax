package gossip_test

import (
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	"github.com/arya-analytics/freighter/fmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Config", func() {
	Describe("Apply", func() {
		It("Should correctly merge a default configuration", func() {
			def := gossip.Config{
				Interval: 1 * time.Second,
			}
			cfg := gossip.Config{}
			Expect(cfg.Merge(def).Interval).To(Equal(def.Interval))
		})
	})
	Describe("Validate", func() {
		It("Should return an error when no transport is provided", func() {
			cfg := gossip.Config{}
			Expect(cfg.Validate()).To(MatchError("[gossip] - transport required"))
		})
	})
	Describe("Report", func() {
		It("Should generate a valid report", func() {
			cfg := gossip.Config{Interval: 1 * time.Second}
			Expect(cfg.Report()["interval"]).To(Equal(1 * time.Second))
			cfg.Transport = &fmock.Unary[gossip.Message, gossip.Message]{}
			Expect(cfg.Report()["transport"]).ToNot(BeNil())
		})
	})
})
