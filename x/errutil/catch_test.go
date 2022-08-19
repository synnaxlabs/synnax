package errutil_test

import (
	"fmt"
	"github.com/arya-analytics/x/errutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Catch", func() {
	Describe("Catch", func() {
		Context("No error encountered", func() {
			var (
				counter int
				catcher *errutil.Catch
			)
			BeforeEach(func() {
				counter = 1
				catcher = errutil.NewCatch()
				for i := 0; i < 4; i++ {
					catcher.Exec(func() error {
						counter++
						return nil
					})
				}
			})
			It("Should continue to execute functions", func() {

				Expect(counter).To(Equal(5))
			})
			It("Should contain a nil error", func() {
				Expect(catcher.Error()).To(BeNil())
			})
		})
		Context("Errors encountered", func() {
			var (
				counter int
				catcher *errutil.Catch
			)
			BeforeEach(func() {
				counter = 1
				catcher = errutil.NewCatch()
				for i := 0; i < 4; i++ {
					catcher.Exec(func() error {
						if i == 2 {
							return fmt.Errorf("encountered unknown error")
						}
						counter++
						return nil
					})
				}
			})
			It("Should stop execution", func() {
				Expect(counter).To(Equal(3))
			})
			It("Should contain a non-nil error", func() {
				Expect(catcher.Error()).ToNot(BeNil())
			})
			Describe("Reset", func() {
				It("Should reset the catcher", func() {
					catcher.Reset()
					Expect(catcher.Error()).To(BeNil())
				})
			})

		})
		Context("Aggregation", func() {
			var catcher = errutil.NewCatch(errutil.WithAggregation())
			It("Should aggregate the errors", func() {
				counter := 1
				for i := 0; i < 4; i++ {
					catcher.Exec(func() error {
						counter++
						return fmt.Errorf("error encountered")
					})
				}
				Expect(counter).To(Equal(5))
				Expect(catcher.Errors()).To(HaveLen(4))
			})
		})
	})

})
