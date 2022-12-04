package kv_test

import (
	"encoding/binary"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("IteratorServer", func() {
	var (
		kv kvx.DB
	)
	BeforeEach(func() {
		kv = memkv.New()
	})
	AfterEach(func() {
		Expect(kv.Close()).To(Succeed())
	})
	Describe("PrefixIter", func() {
		It("Should iterate over keys with a given prefix", func() {
			Expect(kv.Set([]byte("a/foo"), []byte("bar"))).To(Succeed())
			Expect(kv.Set([]byte("a/baz"), []byte("qux"))).To(Succeed())
			Expect(kv.Set([]byte("a/qux"), []byte("quux"))).To(Succeed())
			Expect(kv.Set([]byte("b/foo"), []byte("bar"))).To(Succeed())
			Expect(kv.Set([]byte("b/baz"), []byte("qux"))).To(Succeed())

			iter := kv.NewIterator(kvx.PrefixIter([]byte("a")))
			c := 0
			for iter.First(); iter.Valid(); iter.Next() {
				c++
			}
			Expect(c).To(Equal(3))
			Expect(iter.Close()).To(Succeed())
		})
	})
	Describe("Bounds Iter", func() {
		It("Should iterate over keys in a given range", func() {

			for i := 0; i < 10; i++ {
				b := make([]byte, 4)
				binary.LittleEndian.PutUint32(b, uint32(i))
				Expect(kv.Set(b, []byte{1, 2})).To(Succeed())
			}
			lower := make([]byte, 4)
			binary.LittleEndian.PutUint32(lower, uint32(3))
			upper := make([]byte, 4)
			binary.LittleEndian.PutUint32(upper, uint32(7))
			iter := kv.NewIterator(kvx.RangeIter(lower, upper))
			c := 0
			for iter.First(); iter.Valid(); iter.Next() {
				c++
			}
			Expect(c).To(Equal(4))
			Expect(iter.Close()).To(Succeed())
		})
	})
})
