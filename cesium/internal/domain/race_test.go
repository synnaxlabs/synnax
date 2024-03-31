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
	var db *domain.DB
	BeforeAll(func() {
		db = MustSucceed(domain.Open(domain.Config{FS: fs.NewMem()}))
	})
	Describe("domain/newWriter can fail to find domain overlap errors", func() {
		It("Should be broken", func() {
			By("Creating some new writers and using them to write data")
			i := 10
			var wg sync.WaitGroup
			wg.Add(30)
			for i < 40 {
				go func(i int) {
					defer wg.Done()
					w := MustSucceed(db.NewWriter(ctx, domain.WriterConfig{
						Start: telem.TimeStamp(i) * telem.SecondTS,
						End:   100 * telem.SecondTS,
					}))
					_, err := w.Write([]byte{byte(i + 1), byte(i + 2), byte(i + 3), byte(i + 4)})
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Close()).To(Succeed())
				}(i)
				i++
			}
			wg.Wait()
		})
	})
})
