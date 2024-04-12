package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"sync"
)

var _ = Describe("Race conditions", Ordered, func() {
	var (
		db *domain.DB
		wg sync.WaitGroup
	)
	BeforeAll(func() {
		db = MustSucceed(domain.Open(domain.Config{FS: fs.NewMem()}))
	})
	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("NewWriter can create writers with incorrect bounds", func() {
		It("Should be broken", func() {
			By("Creating some new writers while data is being written")
			i := 40

			wg.Add(30)

			for i > 10 {
				go func(i int) {
					defer wg.Done()
					w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
						Start: telem.TimeStamp(i) * telem.SecondTS,
					}))
					_, err := w.Write([]byte{byte(i), byte(i + 1)})
					Expect(err).ToNot(HaveOccurred())
					err = w.Commit(ctx, telem.TimeStamp(i+1)*telem.SecondTS)
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Close()).To(Succeed())
				}(i)
				i--
			}
			wg.Wait()
		})
	})
})
