package kv_test

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
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
				Expect(c.Add()).To(Equal(int64(1)))
			})
			It("Should increment the number by a set value", func() {
				Expect(c.Add(10)).To(Equal(int64(11)))
			})
		})
		Context("Existing Counter", func() {
			It("Should load the value of the existing counter", func() {
				c, err := kv.NewPersistedCounter(kve, []byte("test-two"))
				Expect(err).NotTo(HaveOccurred())
				Expect(c.Value()).To(Equal(int64(0)))
				_, err = c.Add(10)
				Expect(err).NotTo(HaveOccurred())
				_, err = c.Add(10)
				Expect(err).NotTo(HaveOccurred())
				cTwo, err := kv.NewPersistedCounter(kve, []byte("test-two"))
				Expect(err).NotTo(HaveOccurred())
				Expect(cTwo.Value()).To(Equal(int64(20)))
			})
		})
	})
})
