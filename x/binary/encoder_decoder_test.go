package binary_test

import (
	"bytes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
)

type toEncode struct {
	Value int
}

var _ = Describe("EncoderDecoder", func() {
	DescribeTable("Encode + Decode", func(ecd binary.EncoderDecoder) {
		b, err := ecd.Encode(toEncode{1})
		Expect(err).ToNot(HaveOccurred())
		var d toEncode
		Expect(ecd.Decode(b, &d)).To(Succeed())
		Expect(d.Value).To(Equal(1))
		var d2 toEncode
		Expect(ecd.DecodeStream(bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2.Value).To(Equal(1))
	},
		Entry("Gob", &binary.GobEncoderDecoder{}),
		Entry("JSON", &binary.JSONEncoderDecoder{}),
		Entry("MsgPack", &binary.MsgPackEncoderDecoder{}),
		Entry("PassThrough", &binary.PassThroughEncoderDecoder{EncoderDecoder: &binary.GobEncoderDecoder{}}),
	)
	Describe("PassThrough encoding and decoding", func() {
		It("Should pass through the encoding and decoding when a byte slice is provided", func() {
			ecd := &binary.PassThroughEncoderDecoder{EncoderDecoder: &binary.GobEncoderDecoder{}}
			b, err := ecd.Encode([]byte{1, 2, 3})
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(Equal([]byte{1, 2, 3}))
			var d []byte
			Expect(ecd.Decode(b, &d)).To(Succeed())
			Expect(d).To(Equal([]byte{1, 2, 3}))
		})
	})
})
