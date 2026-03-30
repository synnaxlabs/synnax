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
	"bytes"
	"encoding/binary"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xbinary "github.com/synnaxlabs/x/binary"
	. "github.com/synnaxlabs/x/testutil"
)

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

var _ = Describe("Reader", func() {
	It("Should correctly read primitive values", func() {
		w := xbinary.NewWriter(13, binary.LittleEndian)
		w.Uint8(1)
		w.Uint32(256)
		w.Uint64(1024)
		r := xbinary.NewReader(bytesReader(w.Bytes()), binary.LittleEndian)
		Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
		Expect(MustSucceed(r.Uint32())).To(Equal(uint32(256)))
		Expect(MustSucceed(r.Uint64())).To(Equal(uint64(1024)))
	})

	It("Should return error on EOF", func() {
		r := xbinary.NewReader(bytesReader([]byte{1, 2}), binary.LittleEndian)
		Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
		_, err := r.Uint32()
		Expect(err).To(MatchError(io.ErrUnexpectedEOF))
	})

	It("Should return EOF when no data remains", func() {
		r := xbinary.NewReader(bytesReader([]byte{1}), binary.LittleEndian)
		Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
		_, err := r.Uint8()
		Expect(err).To(MatchError(io.EOF))
	})

	It("Should read arbitrary bytes", func() {
		data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		r := xbinary.NewReader(bytesReader(data), binary.LittleEndian)
		buf := make([]byte, 4)
		Expect(MustSucceed(r.Read(buf))).To(Equal(4))
		Expect(buf).To(Equal([]byte{1, 2, 3, 4}))
		Expect(MustSucceed(r.Read(buf))).To(Equal(4))
		Expect(buf).To(Equal([]byte{5, 6, 7, 8}))
	})

	It("Should work with big-endian byte order", func() {
		w := xbinary.NewWriter(13, binary.BigEndian)
		w.Uint8(1)
		w.Uint32(256)
		w.Uint64(1024)
		r := xbinary.NewReader(bytesReader(w.Bytes()), binary.BigEndian)
		Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
		Expect(MustSucceed(r.Uint32())).To(Equal(uint32(256)))
		Expect(MustSucceed(r.Uint64())).To(Equal(uint64(1024)))
	})

	Describe("Reset", func() {
		It("Should reset to use a new reader", func() {
			r := xbinary.NewReader(bytesReader([]byte{1}), binary.LittleEndian)
			Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
			_, err := r.Uint8()
			Expect(err).To(MatchError(io.EOF))
			r.Reset(bytesReader([]byte{42, 1, 0, 0, 0}))
			Expect(MustSucceed(r.Uint8())).To(Equal(uint8(42)))
			Expect(MustSucceed(r.Uint32())).To(Equal(uint32(1)))
		})
	})
})
