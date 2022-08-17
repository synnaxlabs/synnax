package atomic_test

import (
	"github.com/arya-analytics/x/atomic"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sync"
)

var _ = Describe("Counter", func() {
	Describe("Int32Counter", func() {
		It("Should increment the counter atomically", func() {
			wg := sync.WaitGroup{}
			c := atomic.Int32Counter{}
			wg.Add(10)
			for i := 0; i < 10; i++ {
				go func() {
					defer wg.Done()
					for i := 0; i < 1000; i++ {
						c.Add(1)
					}
				}()
			}
			wg.Wait()
			Expect(c.Value()).To(Equal(int32(10000)))
		})
	})

})
