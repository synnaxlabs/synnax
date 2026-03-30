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
	It("Should correctly write primitive values", func() {
		w := xbinary.NewWriter(0, binary.LittleEndian)
		w.Uint8(1)
		w.Uint32(256)
		w.Uint64(1024)
		Expect(w.Len()).To(Equal(13))
		expected := make([]byte, 13)
		expected[0] = 1
		binary.LittleEndian.PutUint32(expected[1:], 256)
		binary.LittleEndian.PutUint64(expected[5:], 1024)
		Expect(w.Bytes()).To(Equal(expected))
	})

	It("Should auto-grow beyond initial capacity", func() {
		w := xbinary.NewWriter(4, binary.LittleEndian)
		w.Uint32(1)
		w.Uint32(2)
		w.Uint32(3)
		Expect(w.Len()).To(Equal(12))
	})

	It("Should write arbitrary bytes", func() {
		w := xbinary.NewWriter(0, binary.LittleEndian)
		w.Write([]byte{1, 2, 3, 4})
		Expect(w.Bytes()).To(Equal([]byte{1, 2, 3, 4}))
		w.Write([]byte{5, 6, 7, 8})
		Expect(w.Bytes()).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	})

	It("Should work with big-endian byte order", func() {
		w := xbinary.NewWriter(0, binary.BigEndian)
		w.Uint8(1)
		w.Uint32(1)
		w.Uint64(1)
		Expect(w.Bytes()).To(Equal([]byte{1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1}))
	})

	Describe("Reset", func() {
		It("Should clear the buffer for reuse", func() {
			w := xbinary.NewWriter(10, binary.LittleEndian)
			w.Write([]byte{1, 2, 3, 4})
			w.Reset()
			Expect(w.Len()).To(Equal(0))
			w.Write([]byte{5, 6, 7, 8})
			Expect(w.Bytes()).To(Equal([]byte{5, 6, 7, 8}))
		})
	})

	Describe("Resize", func() {
		It("Should ensure capacity without losing data", func() {
			w := xbinary.NewWriter(5, binary.LittleEndian)
			w.Write([]byte{1, 2, 3, 4, 5})
			w.Resize(20)
			w.Write([]byte{6, 7, 8, 9, 10})
			Expect(w.Bytes()).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
		})

		It("Should be a no-op when capacity is sufficient", func() {
			w := xbinary.NewWriter(100, binary.LittleEndian)
			w.Write([]byte{1, 2, 3})
			w.Resize(50)
			Expect(w.Bytes()).To(Equal([]byte{1, 2, 3}))
		})
	})
})
