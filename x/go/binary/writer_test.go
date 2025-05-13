package binary_test

import (
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xbinary "github.com/synnaxlabs/x/binary"
)

var _ = Describe("Writer", func() {
	It("Should correctly write values to the buffer", func() {
		b := xbinary.NewWriter(13, 0, binary.LittleEndian)
		Expect(b.Uint8(1)).To(Equal(1))
		Expect(b.Uint32(1)).To(Equal(4))
		Expect(b.Uint64(1)).To(Equal(8))
		Expect(b.Uint64(1)).To(Equal(0))
		Expect(b.Uint8(1)).To(Equal(0))
		Expect(b.Uint32(1)).To(Equal(0))
		Expect(b.Bytes()).To(Equal([]byte{1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}))
	})

	It("Should write arbitrary bytes to the writer", func() {
		b := xbinary.NewWriter(10, 0, binary.LittleEndian)
		Expect(b.Write([]byte{1, 2, 3, 4})).To(Equal(4))
		Expect(b.Bytes()).To(Equal([]byte{1, 2, 3, 4}))
		Expect(b.Write([]byte{5, 6, 7, 8})).To(Equal(4))
		Expect(b.Bytes()).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
		Expect(b.Write([]byte{9, 10, 11, 12})).To(Equal(2))
		Expect(b.Bytes()).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))

	})

	It("Should panic when the starting offset is greater than the size", func() {
		Expect(func() {
			xbinary.NewWriter(13, 14, binary.LittleEndian)
		}).To(Panic())

	})
})
