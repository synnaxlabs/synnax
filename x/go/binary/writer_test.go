// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary_test

import (
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xbinary "github.com/synnaxlabs/x/binary"
)

var _ = Describe("Writer", func() {
	It("Should correctly write values to the buffer", func() {
		b := xbinary.NewWriter(13, binary.LittleEndian)
		Expect(b.Uint8(1)).To(Equal(1))
		Expect(b.Uint32(1)).To(Equal(4))
		Expect(b.Uint64(1)).To(Equal(8))
		Expect(b.Uint64(1)).To(Equal(0))
		Expect(b.Uint8(1)).To(Equal(0))
		Expect(b.Uint32(1)).To(Equal(0))
		Expect(b.Bytes()).To(Equal([]byte{1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}))
	})

	It("Should write arbitrary bytes to the writer", func() {
		b := xbinary.NewWriter(10, binary.LittleEndian)
		Expect(b.Write([]byte{1, 2, 3, 4})).To(Equal(4))
		Expect(b.Bytes()).To(Equal([]byte{1, 2, 3, 4}))
		Expect(b.Write([]byte{5, 6, 7, 8})).To(Equal(4))
		Expect(b.Bytes()).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
		Expect(b.Write([]byte{9, 10, 11, 12})).To(Equal(2))
		Expect(b.Bytes()).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
	})

	It("Should work with big-endian data", func() {
		b := xbinary.NewWriter(13, binary.BigEndian)
		Expect(b.Uint8(1)).To(Equal(1))
		Expect(b.Uint32(1)).To(Equal(4))
		Expect(b.Uint64(1)).To(Equal(8))
		Expect(b.Uint64(1)).To(Equal(0))
		Expect(b.Uint8(1)).To(Equal(0))
		Expect(b.Uint32(1)).To(Equal(0))
		Expect(b.Bytes()).To(Equal([]byte{1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1}))
	})

	Describe("Reset", func() {
		It("Should reset the underlying buffer and allow new writes to occur", func() {
			b := xbinary.NewWriter(10, binary.LittleEndian)
			b.Write([]byte{1, 2, 3, 4})
			b.Reset()
			Expect(b.Write([]byte{5, 6, 7, 8})).To(Equal(4))
			Expect(b.Bytes()).To(Equal([]byte{5, 6, 7, 8}))
		})
	})

	Describe("Resize", func() {
		It("Should resize the underlying buffer", func() {
			b := xbinary.NewWriter(5, binary.LittleEndian)
			b.Write([]byte{1, 2, 3, 4, 5})
			b.Resize(10)
			Expect(b.Write([]byte{6, 7, 8, 9, 10})).To(Equal(5))
			Expect(b.Bytes()).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))

			b.Reset()
			b.Write([]byte{1, 2, 3})
			b.Resize(2)
			Expect(b.Bytes()).To(Equal([]byte{1, 2}))
		})
	})

})
