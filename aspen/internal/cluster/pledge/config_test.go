package pledge_test

import (
	"github.com/arya-analytics/aspen/internal/cluster/pledge"
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/freighter/fmock"
	. "github.com/arya-analytics/x/testutil"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Config", func() {
	var mockT = &fmock.Unary[node.ID, node.ID]{}

	DescribeTable("Validation", func(cfg pledge.Config, expected error) {
		Expect(cfg.Validate()).To(HaveOccurredAs(expected))
	},
		Entry("No Transport",
			pledge.Config{},
			errors.New("[pledge] - transport required"),
		),
		Entry("Zero RequestTimeout",
			pledge.Config{Transport: mockT},
			errors.New("[pledge] - request timeout must be non-zero"),
		),
		Entry("RetryScale < 1",
			pledge.Config{
				Transport:      mockT,
				RequestTimeout: time.Second,
			},
			errors.New("[pledge] - retry scale must be >= 1"),
		),
		Entry("MaxProposals == 0",
			pledge.Config{
				Transport:      mockT,
				RequestTimeout: time.Second,
				RetryScale:     1,
			},
			errors.New("[pledge] - max proposals must be non-zero"),
		),
		Entry("Candidates == nil",
			pledge.Config{
				Transport:      mockT,
				RequestTimeout: time.Second,
				RetryScale:     1,
				MaxProposals:   1,
			},
			errors.New("[pledge] - candidates required"),
		),
		Entry("Logger == nil",
			pledge.Config{
				Transport:      mockT,
				RequestTimeout: time.Second,
				RetryScale:     1,
				MaxProposals:   1,
				Candidates:     func() node.Group { return node.Group{} },
			},
			errors.New("[pledge] - logger required"),
		),
	)

	Describe("Report", func() {

		It("Should generate a valid configuration report", func() {
			cfg := pledge.DefaultConfig.Override(pledge.Config{
				RequestTimeout: 5 * time.Millisecond,
				Transport:      &fmock.Unary[node.ID, node.ID]{},
			})
			Expect(cfg.Report()["requestTimeout"]).To(Equal(5 * time.Millisecond))
		})

	})

})
