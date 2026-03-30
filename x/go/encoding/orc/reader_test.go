// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package orc_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
)

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

var _ = Describe("Reader", func() {
	It("Should correctly read primitive values", func() {
		w := orc.NewWriter(13)
		w.Uint8(1)
		w.Uint32(256)
		w.Uint64(1024)
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
		Expect(MustSucceed(r.Uint32())).To(Equal(uint32(256)))
		Expect(MustSucceed(r.Uint64())).To(Equal(uint64(1024)))
	})

	It("Should return error on EOF", func() {
		r := orc.NewReader(bytesReader([]byte{1, 2}))
		Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
		_, err := r.Uint32()
		Expect(err).To(MatchError(io.ErrUnexpectedEOF))
	})

	It("Should return EOF when no data remains", func() {
		r := orc.NewReader(bytesReader([]byte{1}))
		Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
		_, err := r.Uint8()
		Expect(err).To(MatchError(io.EOF))
	})

	It("Should read arbitrary bytes", func() {
		data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		r := orc.NewReader(bytesReader(data))
		buf := make([]byte, 4)
		Expect(MustSucceed(r.Read(buf))).To(Equal(4))
		Expect(buf).To(Equal([]byte{1, 2, 3, 4}))
		Expect(MustSucceed(r.Read(buf))).To(Equal(4))
		Expect(buf).To(Equal([]byte{5, 6, 7, 8}))
	})

	It("Should read signed integers", func() {
		w := orc.NewWriter(0)
		w.Int8(-1)
		w.Int16(-256)
		w.Int32(-65536)
		w.Int64(-1)
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(MustSucceed(r.Int8())).To(Equal(int8(-1)))
		Expect(MustSucceed(r.Int16())).To(Equal(int16(-256)))
		Expect(MustSucceed(r.Int32())).To(Equal(int32(-65536)))
		Expect(MustSucceed(r.Int64())).To(Equal(int64(-1)))
	})

	It("Should read floats", func() {
		w := orc.NewWriter(0)
		w.Float32(3.14)
		w.Float64(2.71828)
		r := orc.NewReader(bytesReader(w.Bytes()))
		f32 := MustSucceed(r.Float32())
		Expect(math.Abs(float64(f32) - 3.14)).To(BeNumerically("<", 0.001))
		f64 := MustSucceed(r.Float64())
		Expect(math.Abs(f64 - 2.71828)).To(BeNumerically("<", 0.00001))
	})

	It("Should read bools", func() {
		w := orc.NewWriter(0)
		w.Bool(true)
		w.Bool(false)
		w.Bool(true)
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(MustSucceed(r.Bool())).To(BeTrue())
		Expect(MustSucceed(r.Bool())).To(BeFalse())
		Expect(MustSucceed(r.Bool())).To(BeTrue())
	})

	It("Should read length-prefixed strings", func() {
		w := orc.NewWriter(0)
		w.String("hello")
		w.String("")
		w.String("world")
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(MustSucceed(r.String())).To(Equal("hello"))
		Expect(MustSucceed(r.String())).To(Equal(""))
		Expect(MustSucceed(r.String())).To(Equal("world"))
	})

	It("Should return error on truncated string", func() {
		buf := make([]byte, 7)
		binary.BigEndian.PutUint32(buf, 100)
		buf[4], buf[5], buf[6] = 'a', 'b', 'c'
		r := orc.NewReader(bytesReader(buf))
		_, err := r.String()
		Expect(err).To(MatchError(io.ErrUnexpectedEOF))
	})

	It("Should reject string exceeding MaxStringLen in io.Reader mode", func() {
		prev := orc.MaxStringLen
		defer func() { orc.MaxStringLen = prev }()
		orc.MaxStringLen = 8
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, 9)
		r := orc.NewReader(bytesReader(buf))
		_, err := r.String()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
	})

	It("Should allow string within MaxStringLen in io.Reader mode", func() {
		prev := orc.MaxStringLen
		defer func() { orc.MaxStringLen = prev }()
		orc.MaxStringLen = 8
		w := orc.NewWriter(0)
		w.String("hello")
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(MustSucceed(r.String())).To(Equal("hello"))
	})

	Describe("Reset", func() {
		It("Should reset to use a new reader", func() {
			r := orc.NewReader(bytesReader([]byte{1}))
			Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
			_, err := r.Uint8()
			Expect(err).To(MatchError(io.EOF))
			r.Reset(bytesReader([]byte{42, 0, 0, 1, 0}))
			Expect(MustSucceed(r.Uint8())).To(Equal(uint8(42)))
			Expect(MustSucceed(r.Uint32())).To(Equal(uint32(256)))
		})
	})

	Describe("Depth tracking", func() {
		It("Should track and limit recursion depth", func() {
			r := orc.NewReader(bytesReader(nil))
			Expect(r.PushDepth(3)).To(Succeed())
			Expect(r.PushDepth(3)).To(Succeed())
			Expect(r.PushDepth(3)).To(Succeed())
			Expect(r.PushDepth(3)).To(MatchError(orc.ErrRecursionDepth))
			r.PopDepth()
			Expect(r.PushDepth(3)).To(Succeed())
		})

		It("Should reset depth on Reset", func() {
			r := orc.NewReader(bytesReader(nil))
			Expect(r.PushDepth(2)).To(Succeed())
			Expect(r.PushDepth(2)).To(Succeed())
			r.Reset(bytesReader(nil))
			Expect(r.PushDepth(2)).To(Succeed())
			Expect(r.PushDepth(2)).To(Succeed())
			Expect(r.PushDepth(2)).To(MatchError(orc.ErrRecursionDepth))
		})
	})

	Describe("Round-trip with Writer", func() {
		It("Should correctly round-trip complex data", func() {
			w := orc.NewWriter(0)
			w.Uint8(255)
			w.Uint32(0xDEADBEEF)
			w.Uint64(0x123456789ABCDEF0)
			w.String("hello")
			w.Bool(true)
			w.Float32(1.5)
			w.Float64(2.5)
			w.Int32(-42)

			r := orc.NewReader(bytesReader(w.Bytes()))
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

	Describe("ResetBytes (direct byte-slice mode)", func() {
		It("Should correctly read primitive values", func() {
			w := orc.NewWriter(0)
			w.Uint8(1)
			w.Uint32(256)
			w.Uint64(1024)
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
			Expect(MustSucceed(r.Uint32())).To(Equal(uint32(256)))
			Expect(MustSucceed(r.Uint64())).To(Equal(uint64(1024)))
		})

		It("Should return EOF when no data remains", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{1})
			Expect(MustSucceed(r.Uint8())).To(Equal(uint8(1)))
			_, err := r.Uint8()
			Expect(err).To(MatchError(io.EOF))
		})

		It("Should return ErrUnexpectedEOF on truncated data", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{1, 2})
			_, err := r.Uint32()
			Expect(err).To(MatchError(io.ErrUnexpectedEOF))
		})

		It("Should read length-prefixed strings", func() {
			w := orc.NewWriter(0)
			w.String("hello")
			w.String("")
			w.String("world")
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(MustSucceed(r.String())).To(Equal("hello"))
			Expect(MustSucceed(r.String())).To(Equal(""))
			Expect(MustSucceed(r.String())).To(Equal("world"))
		})

		It("Should return error on truncated string", func() {
			buf := make([]byte, 7)
			binary.BigEndian.PutUint32(buf, 100)
			buf[4], buf[5], buf[6] = 'a', 'b', 'c'
			r := orc.NewReader(nil)
			r.ResetBytes(buf)
			_, err := r.String()
			Expect(err).To(MatchError(io.ErrUnexpectedEOF))
		})

		It("Should read signed integers and floats", func() {
			w := orc.NewWriter(0)
			w.Int8(-1)
			w.Int16(-256)
			w.Int32(-65536)
			w.Int64(-1)
			w.Float32(1.5)
			w.Float64(2.5)
			w.Bool(true)
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(MustSucceed(r.Int8())).To(Equal(int8(-1)))
			Expect(MustSucceed(r.Int16())).To(Equal(int16(-256)))
			Expect(MustSucceed(r.Int32())).To(Equal(int32(-65536)))
			Expect(MustSucceed(r.Int64())).To(Equal(int64(-1)))
			Expect(MustSucceed(r.Float32())).To(Equal(float32(1.5)))
			Expect(MustSucceed(r.Float64())).To(Equal(float64(2.5)))
			Expect(MustSucceed(r.Bool())).To(BeTrue())
		})

		It("Should read arbitrary bytes", func() {
			data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
			r := orc.NewReader(nil)
			r.ResetBytes(data)
			buf := make([]byte, 4)
			Expect(MustSucceed(r.Read(buf))).To(Equal(4))
			Expect(buf).To(Equal([]byte{1, 2, 3, 4}))
			Expect(MustSucceed(r.Read(buf))).To(Equal(4))
			Expect(buf).To(Equal([]byte{5, 6, 7, 8}))
		})

		It("Should return ErrUnexpectedEOF on partial Read", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{1, 2})
			buf := make([]byte, 4)
			n, err := r.Read(buf)
			Expect(n).To(Equal(2))
			Expect(err).To(MatchError(io.ErrUnexpectedEOF))
		})

		It("Should round-trip complex data", func() {
			w := orc.NewWriter(0)
			w.Uint8(255)
			w.Uint32(0xDEADBEEF)
			w.Uint64(0x123456789ABCDEF0)
			w.String("hello")
			w.Bool(true)
			w.Float32(1.5)
			w.Float64(2.5)
			w.Int32(-42)

			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(MustSucceed(r.Uint8())).To(Equal(uint8(255)))
			Expect(MustSucceed(r.Uint32())).To(Equal(uint32(0xDEADBEEF)))
			Expect(MustSucceed(r.Uint64())).To(Equal(uint64(0x123456789ABCDEF0)))
			Expect(MustSucceed(r.String())).To(Equal("hello"))
			Expect(MustSucceed(r.Bool())).To(BeTrue())
			Expect(MustSucceed(r.Float32())).To(Equal(float32(1.5)))
			Expect(MustSucceed(r.Float64())).To(Equal(float64(2.5)))
			Expect(MustSucceed(r.Int32())).To(Equal(int32(-42)))
		})

		Describe("ResetBytes after Reset", func() {
			It("Should switch modes correctly", func() {
				w := orc.NewWriter(0)
				w.Uint32(42)
				w.String("test")

				r := orc.NewReader(bytesReader(w.Bytes()))
				Expect(MustSucceed(r.Uint32())).To(Equal(uint32(42)))
				Expect(MustSucceed(r.String())).To(Equal("test"))

				r.ResetBytes(w.Bytes())
				Expect(MustSucceed(r.Uint32())).To(Equal(uint32(42)))
				Expect(MustSucceed(r.String())).To(Equal("test"))
			})
		})

		Describe("Depth tracking", func() {
			It("Should reset depth on ResetBytes", func() {
				r := orc.NewReader(nil)
				Expect(r.PushDepth(2)).To(Succeed())
				Expect(r.PushDepth(2)).To(Succeed())
				r.ResetBytes(nil)
				Expect(r.PushDepth(2)).To(Succeed())
				Expect(r.PushDepth(2)).To(Succeed())
				Expect(r.PushDepth(2)).To(MatchError(orc.ErrRecursionDepth))
			})
		})
	})
})
