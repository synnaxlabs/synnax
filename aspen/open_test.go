package aspen_test

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"os"
)

var _ = Describe("RouteStream", func() {
	var (
		db1    aspen.DB
		db2    aspen.DB
		logger *zap.SugaredLogger
		exp    alamos.Experiment
	)
	BeforeEach(func() {
		log := zap.NewNop()
		logger = log.Sugar()
		exp = alamos.New("aspen_join_test")
		var err error
		db1, err = aspen.Open(
			context.TODO(),
			"",
			"localhost:22646",
			[]address.Address{},
			aspen.Bootstrap(),
			aspen.WithLogger(logger),
			aspen.WithExperiment(alamos.Sub(exp, "db1")),
			aspen.MemBacked(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		)
		Expect(err).ToNot(HaveOccurred())
		db2, err = aspen.Open(
			context.TODO(),
			"",
			"localhost:22647",
			[]address.Address{"localhost:22646"},
			aspen.WithLogger(logger),
			aspen.WithExperiment(alamos.Sub(exp, "db2")),
			aspen.MemBacked(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		)
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		Expect(db1.Close()).To(Succeed())
		Expect(db2.Close()).To(Succeed())
		f, err := os.Create("aspen_join_test_report.json")
		defer func() {
			Expect(f.Close()).To(Succeed())
		}()
		Expect(err).ToNot(HaveOccurred())
		Expect(exp.Report().WriteJSON(f)).To(Succeed())
	})
	It("Should be able to join two clusters", func() {
		Eventually(db1.Nodes).Should(HaveLen(2))
		Eventually(db2.Nodes).Should(HaveLen(2))
	})
})
