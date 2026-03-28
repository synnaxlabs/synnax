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
	"math"

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

	It("Should read signed integers", func() {
		w := xbinary.NewWriter(0, binary.BigEndian)
		w.Int8(-1)
		w.Int16(-256)
		w.Int32(-65536)
		w.Int64(-1)
		r := xbinary.NewReader(bytesReader(w.Bytes()), binary.BigEndian)
		Expect(MustSucceed(r.Int8())).To(Equal(int8(-1)))
		Expect(MustSucceed(r.Int16())).To(Equal(int16(-256)))
		Expect(MustSucceed(r.Int32())).To(Equal(int32(-65536)))
		Expect(MustSucceed(r.Int64())).To(Equal(int64(-1)))
	})

	It("Should read floats", func() {
		w := xbinary.NewWriter(0, binary.BigEndian)
		w.Float32(3.14)
		w.Float64(2.71828)
		r := xbinary.NewReader(bytesReader(w.Bytes()), binary.BigEndian)
		f32 := MustSucceed(r.Float32())
		Expect(math.Abs(float64(f32) - 3.14)).To(BeNumerically("<", 0.001))
		f64 := MustSucceed(r.Float64())
		Expect(math.Abs(f64 - 2.71828)).To(BeNumerically("<", 0.00001))
	})

	It("Should read bools", func() {
		w := xbinary.NewWriter(0, binary.BigEndian)
		w.Bool(true)
		w.Bool(false)
		w.Bool(true)
		r := xbinary.NewReader(bytesReader(w.Bytes()), binary.BigEndian)
		Expect(MustSucceed(r.Bool())).To(BeTrue())
		Expect(MustSucceed(r.Bool())).To(BeFalse())
		Expect(MustSucceed(r.Bool())).To(BeTrue())
	})

	It("Should read length-prefixed strings", func() {
		w := xbinary.NewWriter(0, binary.BigEndian)
		w.String("hello")
		w.String("")
		w.String("world")
		r := xbinary.NewReader(bytesReader(w.Bytes()), binary.BigEndian)
		Expect(MustSucceed(r.String())).To(Equal("hello"))
		Expect(MustSucceed(r.String())).To(Equal(""))
		Expect(MustSucceed(r.String())).To(Equal("world"))
	})

	It("Should return error on truncated string", func() {
		buf := make([]byte, 7)
		binary.BigEndian.PutUint32(buf, 100)
		buf[4], buf[5], buf[6] = 'a', 'b', 'c'
		r := xbinary.NewReader(bytesReader(buf), binary.BigEndian)
		_, err := r.String()
		Expect(err).To(MatchError(io.ErrUnexpectedEOF))
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

	Describe("Depth tracking", func() {
		It("Should track and limit recursion depth", func() {
			r := xbinary.NewReader(bytesReader(nil), binary.BigEndian)
			Expect(r.PushDepth(3)).To(Succeed())
			Expect(r.PushDepth(3)).To(Succeed())
			Expect(r.PushDepth(3)).To(Succeed())
			Expect(r.PushDepth(3)).To(MatchError(xbinary.ErrRecursionDepth))
			r.PopDepth()
			Expect(r.PushDepth(3)).To(Succeed())
		})

		It("Should reset depth on Reset", func() {
			r := xbinary.NewReader(bytesReader(nil), binary.BigEndian)
			Expect(r.PushDepth(2)).To(Succeed())
			Expect(r.PushDepth(2)).To(Succeed())
			r.Reset(bytesReader(nil))
			Expect(r.PushDepth(2)).To(Succeed())
			Expect(r.PushDepth(2)).To(Succeed())
			Expect(r.PushDepth(2)).To(MatchError(xbinary.ErrRecursionDepth))
		})
	})

	Describe("Round-trip with Writer", func() {
		It("Should correctly round-trip complex data", func() {
			w := xbinary.NewWriter(0, binary.LittleEndian)
			w.Uint8(255)
			w.Uint32(0xDEADBEEF)
			w.Uint64(0x123456789ABCDEF0)
			w.String("hello")
			w.Bool(true)
			w.Float32(1.5)
			w.Float64(2.5)
			w.Int32(-42)

			r := xbinary.NewReader(bytesReader(w.Bytes()), binary.LittleEndian)
			Expect(MustSucceed(r.Uint8())).To(Equal(uint8(255)))
			Expect(MustSucceed(r.Uint32())).To(Equal(uint32(0xDEADBEEF)))
			Expect(MustSucceed(r.Uint64())).To(Equal(uint64(0x123456789ABCDEF0)))
			Expect(MustSucceed(r.String())).To(Equal("hello"))
			Expect(MustSucceed(r.Bool())).To(BeTrue())
			Expect(MustSucceed(r.Float32())).To(Equal(float32(1.5)))
			Expect(MustSucceed(r.Float64())).To(Equal(float64(2.5)))
			Expect(MustSucceed(r.Int32())).To(Equal(int32(-42)))
		})
	})
})
