package lock_test

import (
	"github.com/arya-analytics/x/lock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApplySink", func() {
	It("Should allow the caller to acquire the lock", func() {
		m := lock.NewLock[int]()
		Expect(m.TryLock(1)).To(BeTrue())
	})
	It("Should return an error when the caller tries to acquire a lock that is already held", func() {
		m := lock.NewLock[int]()
		Expect(m.TryLock(1)).To(BeTrue())
		Expect(m.TryLock(1)).To(BeFalse())
	})
	It("Should allow the called to release the lock", func() {
		m := lock.NewLock[int]()
		Expect(m.TryLock(1)).To(BeTrue())
		m.Unlock(1)
		Expect(m.TryLock(1)).To(BeTrue())
	})
	It("Should panic if the caller tries to release an unlocked lock", func() {
		m := lock.NewLock[int]()
		Expect(func() { m.Unlock(1) }).To(Panic())
	})
	It("Should prevent multiple goroutines from acquiring the same key", func() {
		m := lock.NewLock[int]()
		acquisitions := make([]bool, 100)
		for i := 0; i < 100; i++ {
			go func(i int) {
				acquisitions[i] = m.TryLock(1)
			}(i)
		}
		totalTrue := 0
		for _, a := range acquisitions {
			if a {
				totalTrue++
			}
		}
		Expect(totalTrue).To(Equal(1))
	})
})
