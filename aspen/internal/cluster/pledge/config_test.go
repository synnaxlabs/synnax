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

	DescribeTable("Validation", func(cfg pledge.Config, expected error) {
		Expect(cfg.Validate()).To(HaveOccurredAs(expected))
	},
		Entry("No Transport",
			pledge.Config{},
			errors.New("[pledge] - transport required"),
		),
		Entry("Zero RequestTimeout",
			pledge.Config{Transport: &fmock.Unary[node.ID, node.ID]{}},
			errors.New("[pledge] - request timeout must be non-zero"),
		),
		Entry("RetryScale < 1",
			pledge.Config{
				Transport:      &fmock.Unary[node.ID, node.ID]{},
				RequestTimeout: time.Second,
			},
			errors.New("[pledge] - retry scale must be >= 1"),
		),
		Entry("MaxProposals == 0",
			pledge.Config{
				Transport:      &fmock.Unary[node.ID, node.ID]{},
				RequestTimeout: time.Second,
				RetryScale:     1,
			},
			errors.New("[pledge] - max proposals must be non-zero"),
		),
		Entry("Candidates == nil",
			pledge.Config{
				Transport:      &fmock.Unary[node.ID, node.ID]{},
				RequestTimeout: time.Second,
				RetryScale:     1,
				MaxProposals:   1,
			},
			errors.New("[pledge] - candidates required"),
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
