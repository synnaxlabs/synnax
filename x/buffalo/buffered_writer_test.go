package buffalo_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/buffalo"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("BufferedWriter", func() {
	var p *buffalo.Pool
	BeforeEach(func() {
		p = buffalo.NewPool(512)
	})
	Describe("Write", func() {
		Context("Evenly divisible", func() {
			It("Should flush the buffer when full", func() {
				buf := p.Acquire()
				internal := p.Acquire()
				writer := &buffalo.BufferedWriter{
					Buffer: buf,
					Writer: internal,
				}
				b := make([]byte, 512)
				n := MustSucceed(writer.Write(b))
				Expect(n).To(Equal(512))
				Expect(buf.Len()).To(Equal(512))
				Expect(internal.Len()).To(Equal(0))
				n = MustSucceed(writer.Write(b))
				Expect(n).To(Equal(512))
				Expect(buf.Len()).To(Equal(512))
				Expect(internal.Len()).To(Equal(512))
			})
		})
		Context("Not evenly divisible", func() {
			It("Should flush the buffer when full", func() {
				buf := p.Acquire()
				internal := p.Acquire()
				writer := &buffalo.BufferedWriter{
					Buffer: buf,
					Writer: internal,
				}
				b := make([]byte, 256)
				n := MustSucceed(writer.Write(b))
				Expect(n).To(Equal(256))
				Expect(buf.Len()).To(Equal(256))
				Expect(internal.Len()).To(Equal(0))
				b2 := make([]byte, 512)
				n = MustSucceed(writer.Write(b2))
				Expect(n).To(Equal(512))
				Expect(buf.Len()).To(Equal(512))
				Expect(internal.Len()).To(Equal(256))
			})
		})
	})
})
