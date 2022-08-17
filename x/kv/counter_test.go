package kv_test

import (
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/pebblekv"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Counter", Ordered, func() {
	var (
		kve kv.DB
	)
	BeforeAll(func() {
		db, err := pebble.Open("", &pebble.Options{FS: vfs.NewMem()})
		Expect(err).NotTo(HaveOccurred())
		kve = pebblekv.Wrap(db)
	})
	AfterAll(func() {
		Expect(kve.Close()).To(Succeed())
	})
	Describe("PersistedCounter", func() {
		Context("Requests Counter", Ordered, func() {
			var c *kv.PersistedCounter
			BeforeAll(func() {
				var err error
				c, err = kv.NewPersistedCounter(kve, []byte("test"))
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create a counter with a starting value of 0", func() {
				Expect(c.Value()).To(Equal(int64(0)))
			})
			It("Should increment the counter correctly", func() {
				Expect(c.Increment()).To(Equal(int64(1)))
			})
			It("Should decrement the counter correctly", func() {
				Expect(c.Decrement()).To(Equal(int64(0)))
			})
			It("Should increment the number by a set value", func() {
				Expect(c.Increment(10)).To(Equal(int64(10)))
			})
			It("Should decrement the number by a set value", func() {
				Expect(c.Decrement(10)).To(Equal(int64(0)))
			})
		})
		Context("Existing Counter", func() {
			It("Should load the value of the existing counter", func() {
				c, err := kv.NewPersistedCounter(kve, []byte("test-two"))
				Expect(err).NotTo(HaveOccurred())
				Expect(c.Value()).To(Equal(int64(0)))
				_, err = c.Increment()
				Expect(err).NotTo(HaveOccurred())
				cTwo, err := kv.NewPersistedCounter(kve, []byte("test-two"))
				Expect(err).NotTo(HaveOccurred())
				Expect(cTwo.Value()).To(Equal(int64(1)))
			})
		})
	})
})
