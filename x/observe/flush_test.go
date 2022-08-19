package observe_test

import (
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/arya-analytics/x/observe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type dataStruct struct {
	Value []byte
}

var _ = Describe("Flush", func() {
	It("Should flush the observable contents", func() {
		o := observe.New[dataStruct]()
		kv := memkv.New()
		ecd := &binary.GobEncoderDecoder{}
		flush := &observe.FlushSubscriber[dataStruct]{
			Key:         []byte("key"),
			Store:       kv,
			MinInterval: 0,
			Encoder:     ecd,
		}
		o.OnChange(flush.FlushSync)
		o.Notify(dataStruct{Value: []byte("hello")})
		b, err := kv.Get([]byte("key"))
		Expect(err).ToNot(HaveOccurred())
		var ds dataStruct
		Expect(ecd.Decode(b, &ds)).To(Succeed())
		Expect(ds.Value).To(Equal([]byte("hello")))
	})
})
