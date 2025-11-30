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
	"encoding/binary"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xbinary "github.com/synnaxlabs/x/binary"
)

var _ = Describe("Reader", func() {
	It("Should correctly read values from the buffer", func() {
		// Write data using Writer, then read it back with Reader
		w := xbinary.NewWriter(13, binary.LittleEndian)
		w.Uint8(1)
		w.Uint32(256)
		w.Uint64(1024)

		r := xbinary.NewReader(bytes.NewReader(w.Bytes()), binary.LittleEndian)
		v8, err := r.Uint8()
		Expect(err).To(BeNil())
		Expect(v8).To(Equal(uint8(1)))
		v32, err := r.Uint32()
		Expect(err).To(BeNil())
		Expect(v32).To(Equal(uint32(256)))
		v64, err := r.Uint64()
		Expect(err).To(BeNil())
		Expect(v64).To(Equal(uint64(1024)))
	})

	It("Should return error on EOF", func() {
		r := xbinary.NewReader(bytes.NewReader([]byte{1, 2}), binary.LittleEndian)
		v8, err := r.Uint8()
		Expect(err).To(BeNil())
		Expect(v8).To(Equal(uint8(1)))
		// Try to read uint32 with only 1 byte remaining
		_, err = r.Uint32()
		Expect(err).To(MatchError(io.ErrUnexpectedEOF))
	})

	It("Should return EOF when no data remains", func() {
		r := xbinary.NewReader(bytes.NewReader([]byte{1}), binary.LittleEndian)
		v8, err := r.Uint8()
		Expect(err).To(BeNil())
		Expect(v8).To(Equal(uint8(1)))
		// Now EOF
		_, err = r.Uint8()
		Expect(err).To(MatchError(io.EOF))
	})

	It("Should read arbitrary bytes", func() {
		data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		r := xbinary.NewReader(bytes.NewReader(data), binary.LittleEndian)
		buf := make([]byte, 4)
		n, err := r.Read(buf)
		Expect(err).To(BeNil())
		Expect(n).To(Equal(4))
		Expect(buf).To(Equal([]byte{1, 2, 3, 4}))
		n, err = r.Read(buf)
		Expect(err).To(BeNil())
		Expect(n).To(Equal(4))
		Expect(buf).To(Equal([]byte{5, 6, 7, 8}))
	})

	It("Should work with big-endian data", func() {
		w := xbinary.NewWriter(13, binary.BigEndian)
		w.Uint8(1)
		w.Uint32(256)
		w.Uint64(1024)

		r := xbinary.NewReader(bytes.NewReader(w.Bytes()), binary.BigEndian)
		v8, err := r.Uint8()
		Expect(err).To(BeNil())
		Expect(v8).To(Equal(uint8(1)))
		v32, err := r.Uint32()
		Expect(err).To(BeNil())
		Expect(v32).To(Equal(uint32(256)))
		v64, err := r.Uint64()
		Expect(err).To(BeNil())
		Expect(v64).To(Equal(uint64(1024)))
	})

	Describe("Reset", func() {
		It("Should reset to use a new reader", func() {
			r := xbinary.NewReader(bytes.NewReader([]byte{1}), binary.LittleEndian)
			v8, err := r.Uint8()
			Expect(err).To(BeNil())
			Expect(v8).To(Equal(uint8(1)))
			_, err = r.Uint8() // EOF
			Expect(err).To(MatchError(io.EOF))

			// Reset with new data (little-endian: uint8(42), uint32(1) = {1, 0, 0, 0})
			r.Reset(bytes.NewReader([]byte{42, 1, 0, 0, 0}))
			v8, err = r.Uint8()
			Expect(err).To(BeNil())
			Expect(v8).To(Equal(uint8(42)))
			v32, err := r.Uint32()
			Expect(err).To(BeNil())
			Expect(v32).To(Equal(uint32(1)))
		})
	})

	Describe("Round-trip with Writer", func() {
		It("Should correctly round-trip complex data", func() {
			w := xbinary.NewWriter(100, binary.LittleEndian)
			w.Uint8(255)
			w.Uint32(0xDEADBEEF)
			w.Uint64(0x123456789ABCDEF0)
			w.Write([]byte("hello"))
			w.Uint32(42)

			r := xbinary.NewReader(bytes.NewReader(w.Bytes()), binary.LittleEndian)
			v8, err := r.Uint8()
			Expect(err).To(BeNil())
			Expect(v8).To(Equal(uint8(255)))
			v32, err := r.Uint32()
			Expect(err).To(BeNil())
			Expect(v32).To(Equal(uint32(0xDEADBEEF)))
			v64, err := r.Uint64()
			Expect(err).To(BeNil())
			Expect(v64).To(Equal(uint64(0x123456789ABCDEF0)))
			buf := make([]byte, 5)
			_, err = r.Read(buf)
			Expect(err).To(BeNil())
			Expect(string(buf)).To(Equal("hello"))
			v32, err = r.Uint32()
			Expect(err).To(BeNil())
			Expect(v32).To(Equal(uint32(42)))
		})
	})
})
