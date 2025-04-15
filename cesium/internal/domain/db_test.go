package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"

	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			Describe("HasDataFor", func() {
				var (
					db      *domain.DB
					fs      xfs.FS
					cleanUp func() error
				)
				BeforeEach(func() {
					fs, cleanUp = makeFS()
					db = MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						Instrumentation: PanicLogger(),
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})
				It("Should return true if the domain DB has data for particular time range", func() {
					tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
					Expect(domain.Write(ctx, db, tr, []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
					Expect(db.HasDataFor(ctx, tr)).To(BeTrue())
				})
				It("Should return false if the domain DB does not have data for particular time range", func() {
					tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
					Expect(domain.Write(ctx, db, tr, []byte{1, 2, 3, 4, 5, 6})).To(Succeed())
					Expect(db.HasDataFor(ctx, (20 * telem.SecondTS).SpanRange(10*telem.Second))).To(BeFalse())
				})
			})
			Describe("Close", func() {
				It("Should return an error if there are open writers on the DB", func() {
					fs, cleanUp := makeFS()
					db := MustSucceed(domain.Open(domain.Config{
						FS:              fs,
						Instrumentation: PanicLogger(),
					}))
					defer func() {
						Expect(cleanUp()).To(Succeed())
					}()
					_ = MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{}))
					Expect(db.Close()).To(MatchError(domain.ErrOpenEntity))
				})
			})
		})
	}
})
