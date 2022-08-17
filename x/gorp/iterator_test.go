package gorp_test

import (
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/gorp"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/memkv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Iterator", func() {
	var (
		kv   kvx.DB
		ecdc binary.EncoderDecoder
	)
	BeforeEach(func() {
		kv = memkv.New()
		ecdc = &binary.GobEncoderDecoder{}
		val, err := ecdc.Encode(map[string]string{"key1": "value1", "key2": "value2"})
		Expect(err).To(BeNil())
		Expect(kv.Set([]byte("key1"), val)).To(Succeed())
		Expect(kv.Set([]byte("key2"), val)).To(Succeed())

	})
	AfterEach(func() {
		Expect(kv.Close()).To(Succeed())
	})
	It("Should decode values before returning them to the caller", func() {
		iter := gorp.WrapKVIter[map[string]string](
			kv.NewIterator(kvx.PrefixIter([]byte("key"))),
			gorp.WithEncoderDecoder(ecdc),
		)
		for iter.First(); iter.Valid(); iter.Next() {
			Expect(iter.Value()).To(Equal(map[string]string{"key1": "value1", "key2": "value2"}))
		}
		Expect(iter.Error()).To(BeNil())
		Expect(iter.Close()).To(Succeed())
	})
})
