package compress_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary/compress"
)

var _ = Describe("Compress", func() {
	Describe("Sub 8-bit tests", func() {
		It("Basic", func() {
			bytes := []byte{0, 1, 0, 1, 0, 1}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
		It("Basic II", func() {
			bytes := []byte{1, 0, 1, 0, 1, 0}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
		It("Basic III", func() {
			bytes := []byte{0, 0, 0, 1, 1, 0, 0, 0, 0, 1}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
		It("Basic IV", func() {
			bytes := []byte{0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
		It("Longer I", func() {
			bytes := []byte{1, 1, 1, 0, 0, 1, 1, 1, 1, 0}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
		It("Longer II", func() {
			bytes := []byte{0, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 1, 1, 1, 1}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
		It("Edge", func() {
			bytes := []byte{0}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
		It("Edge II", func() {
			bytes := []byte{1}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
		It("All Zero", func() {
			bytes := []byte{0, 0, 0, 0, 0, 0, 0}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
		It("All One", func() {
			bytes := []byte{1, 1, 1, 1, 1}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
	})
	Describe("Longer Tests", func() {
		It("Longer I", func() {
			bytes := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0,
			}
			compressed, err := compress.Compress(bytes)
			result, err := compress.Decompress(compressed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(bytes))
		})
	})
})
