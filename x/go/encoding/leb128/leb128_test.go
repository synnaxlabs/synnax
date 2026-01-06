// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package leb128_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/leb128"
)

var _ = Describe("LEB128", func() {
	Describe("WriteUnsigned", func() {
		It("Should encode small values in a single byte", func() {
			var buf bytes.Buffer
			Expect(leb128.WriteUnsigned(&buf, 0)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x00}))

			buf.Reset()
			Expect(leb128.WriteUnsigned(&buf, 1)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x01}))

			buf.Reset()
			Expect(leb128.WriteUnsigned(&buf, 127)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x7f}))
		})

		It("Should encode values >= 128 in multiple bytes", func() {
			var buf bytes.Buffer
			Expect(leb128.WriteUnsigned(&buf, 128)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x80, 0x01}))

			buf.Reset()
			Expect(leb128.WriteUnsigned(&buf, 255)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0xff, 0x01}))

			buf.Reset()
			Expect(leb128.WriteUnsigned(&buf, 16384)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x80, 0x80, 0x01}))
		})
	})

	Describe("WriteSigned", func() {
		It("Should encode small positive values", func() {
			var buf bytes.Buffer
			Expect(leb128.WriteSigned(&buf, 0)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x00}))

			buf.Reset()
			Expect(leb128.WriteSigned(&buf, 1)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x01}))

			buf.Reset()
			Expect(leb128.WriteSigned(&buf, 42)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x2a}))

			buf.Reset()
			Expect(leb128.WriteSigned(&buf, 63)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x3f}))
		})

		It("Should encode negative values", func() {
			var buf bytes.Buffer
			Expect(leb128.WriteSigned(&buf, -1)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x7f}))

			buf.Reset()
			Expect(leb128.WriteSigned(&buf, -64)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x40}))

			buf.Reset()
			Expect(leb128.WriteSigned(&buf, -65)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0xbf, 0x7f}))
		})

		It("Should NOT use zigzag encoding (unlike Go's binary.PutVarint)", func() {
			var buf bytes.Buffer
			// For value 1: zigzag would produce 0x02, but signed LEB128 produces 0x01
			Expect(leb128.WriteSigned(&buf, 1)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x01}))
			Expect(buf.Bytes()).ToNot(Equal([]byte{0x02})) // Would be zigzag

			// For value -1: zigzag would produce 0x01, but signed LEB128 produces 0x7f
			buf.Reset()
			Expect(leb128.WriteSigned(&buf, -1)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte{0x7f}))
			Expect(buf.Bytes()).ToNot(Equal([]byte{0x01})) // Would be zigzag
		})
	})
})
