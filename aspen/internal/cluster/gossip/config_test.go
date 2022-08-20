package gossip_test

import (
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	"github.com/arya-analytics/aspen/internal/cluster/store"
	"github.com/arya-analytics/freighter/fmock"
	. "github.com/arya-analytics/x/testutil"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Config", func() {
	var (
		mockT = &fmock.Unary[gossip.Message, gossip.Message]{}
		s     = store.New()
	)

	DescribeTable("Validation", func(cfg gossip.Config, expected error) {
		Expect(cfg.Validate()).To(HaveOccurredAs(expected))
	},
		Entry("No Transport",
			gossip.Config{},
			errors.New("[gossip] - transport required"),
		),
		Entry("No store",
			gossip.Config{
				Transport: mockT,
			},
			errors.New("[gossip] - store required"),
		),
		Entry("Zero Interval",
			gossip.Config{
				Transport: mockT,
				Store:     s,
			},
			errors.New("[gossip] - interval must be positive"),
		),
		Entry("No Logger",
			gossip.Config{
				Transport: mockT,
				Store:     s,
				Interval:  5 * time.Millisecond,
			},
			errors.New("[gossip] - logger required"),
		),
	)

	Describe("Report", func() {
		It("Should generate a valid report", func() {
			cfg := gossip.Config{Interval: 1 * time.Second, Transport: mockT}
			Expect(cfg.Report()["interval"]).To(Equal(1 * time.Second))
			cfg.Transport = &fmock.Unary[gossip.Message, gossip.Message]{}
			Expect(cfg.Report()["transport"]).ToNot(BeNil())
		})
	})
})
