package kv_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/kv"
)

var _ = Describe("KVKey", func() {
	Describe("CompositeKey", func() {
		It("Should generate a composite key from elements", func() {
			key, err := kv.CompositeKey("foo", int8(1))
			Expect(err).ToNot(HaveOccurred())
			Expect(key).To(Equal([]byte{102, 111, 111, 1}))
		})
	})
})
