package observe_test

import (
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/arya-analytics/x/observe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
)

type simpleFlusher struct {
	value []byte
}

func (f simpleFlusher) Flush(w io.Writer) error {
	_, err := w.Write(f.value)
	return err
}

var _ = Describe("Flush", func() {
	It("Should flush the observable contents", func() {
		o := observe.New[simpleFlusher]()
		kv := memkv.New()
		flush := &observe.FlushSubscriber[simpleFlusher]{
			Key:         []byte("key"),
			Store:       kv,
			MinInterval: 0,
		}
		o.OnChange(flush.FlushSync)
		o.Notify(simpleFlusher{value: []byte("hello")})
		Expect(kv.Get([]byte("key"))).To(Equal([]byte("hello")))
	})
})
