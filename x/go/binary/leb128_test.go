// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
)

var _ = Describe("LEB128", func() {
	Describe("WriteLEB128Unsigned", func() {
		It("Should encode small values in a single byte", func() {
			var buf bytes.Buffer
			binary.WriteLEB128Unsigned(&buf, 0)
			Expect(buf.Bytes()).To(Equal([]byte{0x00}))

			buf.Reset()
			binary.WriteLEB128Unsigned(&buf, 1)
			Expect(buf.Bytes()).To(Equal([]byte{0x01}))

			buf.Reset()
			binary.WriteLEB128Unsigned(&buf, 127)
			Expect(buf.Bytes()).To(Equal([]byte{0x7f}))
		})

		It("Should encode values >= 128 in multiple bytes", func() {
			var buf bytes.Buffer
			binary.WriteLEB128Unsigned(&buf, 128)
			Expect(buf.Bytes()).To(Equal([]byte{0x80, 0x01}))

			buf.Reset()
			binary.WriteLEB128Unsigned(&buf, 255)
			Expect(buf.Bytes()).To(Equal([]byte{0xff, 0x01}))

			buf.Reset()
			binary.WriteLEB128Unsigned(&buf, 16384)
			Expect(buf.Bytes()).To(Equal([]byte{0x80, 0x80, 0x01}))
		})
	})

	Describe("WriteLEB128Signed", func() {
		It("Should encode small positive values", func() {
			var buf bytes.Buffer
			binary.WriteLEB128Signed(&buf, 0)
			Expect(buf.Bytes()).To(Equal([]byte{0x00}))

			buf.Reset()
			binary.WriteLEB128Signed(&buf, 1)
			Expect(buf.Bytes()).To(Equal([]byte{0x01}))

			buf.Reset()
			binary.WriteLEB128Signed(&buf, 42)
			Expect(buf.Bytes()).To(Equal([]byte{0x2a}))

			buf.Reset()
			binary.WriteLEB128Signed(&buf, 63)
			Expect(buf.Bytes()).To(Equal([]byte{0x3f}))
		})

		It("Should encode negative values", func() {
			var buf bytes.Buffer
			binary.WriteLEB128Signed(&buf, -1)
			Expect(buf.Bytes()).To(Equal([]byte{0x7f}))

			buf.Reset()
			binary.WriteLEB128Signed(&buf, -64)
			Expect(buf.Bytes()).To(Equal([]byte{0x40}))

			buf.Reset()
			binary.WriteLEB128Signed(&buf, -65)
			Expect(buf.Bytes()).To(Equal([]byte{0xbf, 0x7f}))
		})

		It("Should NOT use zigzag encoding (unlike Go's binary.PutVarint)", func() {
			var buf bytes.Buffer
			// For value 1: zigzag would produce 0x02, but signed LEB128 produces 0x01
			binary.WriteLEB128Signed(&buf, 1)
			Expect(buf.Bytes()).To(Equal([]byte{0x01}))
			Expect(buf.Bytes()).ToNot(Equal([]byte{0x02})) // Would be zigzag

			// For value -1: zigzag would produce 0x01, but signed LEB128 produces 0x7f
			buf.Reset()
			binary.WriteLEB128Signed(&buf, -1)
			Expect(buf.Bytes()).To(Equal([]byte{0x7f}))
			Expect(buf.Bytes()).ToNot(Equal([]byte{0x01})) // Would be zigzag
		})
	})

	Describe("AppendLEB128Unsigned", func() {
		It("Should append encoded bytes to slice", func() {
			dst := []byte{0xaa, 0xbb}
			result := binary.AppendLEB128Unsigned(dst, 127)
			Expect(result).To(Equal([]byte{0xaa, 0xbb, 0x7f}))
		})
	})

	Describe("AppendLEB128Signed", func() {
		It("Should append encoded bytes to slice", func() {
			dst := []byte{0xaa, 0xbb}
			result := binary.AppendLEB128Signed(dst, -1)
			Expect(result).To(Equal([]byte{0xaa, 0xbb, 0x7f}))
		})
	})
})
