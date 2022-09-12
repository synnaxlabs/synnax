package mock_test

import (
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/arya-analytics/x/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Storage", func() {
	Describe("ProviderBuilder", func() {
		DescribeTable("New", func(cfg ...storage.Config) {
			b := mock.NewBuilder(cfg...)
			store := b.New()
			Expect(store).NotTo(BeNil())
			Expect(store.KV.Set([]byte("foo"), []byte("bar"))).To(Succeed())
			Expect(b.Close()).To(Succeed())
			Expect(b.Cleanup()).To(Succeed())
		},
			Entry("Memory-backed storage implementation"),
			Entry("File-backed storage implementation", storage.Config{MemBacked: config.BoolPointer(false), Dirname: "./tmp"}),
		)
	})
})
