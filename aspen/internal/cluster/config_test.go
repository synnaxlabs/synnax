package cluster_test

import (
	"github.com/arya-analytics/aspen/internal/cluster"
	"github.com/arya-analytics/x/kv/memkv"
	. "github.com/arya-analytics/x/testutil"
	"github.com/cockroachdb/errors"
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
		It("Should attach a storage report when provided", func() {
			cfg := cluster.Config{
				Storage: memkv.New(),
			}
			rep := cfg.Report()
			Expect(rep["storage"]).To(Equal(cfg.Storage.Report()))
			Expect(cfg.Storage.Close()).To(Succeed())
		})
	})

	DescribeTable("Validate", func(cfg cluster.Config, expected error) {
		Expect(cfg.Validate()).To(HaveOccurredAs(expected))
	},
		Entry("No Host Address", cluster.Config{}, errors.New("[cluster] - HostAddress is required")),
		Entry("No Storage Key", cluster.Config{HostAddress: "localhost:8080"}, errors.New("[cluster] - StorageKey is required")),
		Entry("Zero StorageFlushInterval", cluster.Config{HostAddress: "localhost:8080", StorageKey: []byte("aspen.cluster")}, errors.New("[cluster] - StorageFlushInterval must be FlushOnEvery or positive")),
	)

})
