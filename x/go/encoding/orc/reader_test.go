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
		Expect(r.Uint8()).To(Equal(uint8(1)))
		Expect(r.Uint32()).To(Equal(uint32(256)))
		Expect(r.Uint64()).To(Equal(uint64(1024)))
	})

	It("Should return error on EOF", func() {
		r := orc.NewReader(bytesReader([]byte{1, 2}))
		Expect(r.Uint8()).To(Equal(uint8(1)))
		Expect(r.Uint32()).Error().To(MatchError(io.ErrUnexpectedEOF))
	})

	It("Should return EOF when no data remains", func() {
		r := orc.NewReader(bytesReader([]byte{1}))
		Expect(r.Uint8()).To(Equal(uint8(1)))
		Expect(r.Uint8()).Error().To(MatchError(io.EOF))
	})

	It("Should read arbitrary bytes", func() {
		data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		r := orc.NewReader(bytesReader(data))
		buf := make([]byte, 4)
		Expect(r.Read(buf)).To(Equal(4))
		Expect(buf).To(Equal([]byte{1, 2, 3, 4}))
		Expect(r.Read(buf)).To(Equal(4))
		Expect(buf).To(Equal([]byte{5, 6, 7, 8}))
	})

	It("Should read signed integers", func() {
		w := orc.NewWriter(0)
		w.Int8(-1)
		w.Int16(-256)
		w.Int32(-65536)
		w.Int64(-1)
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(r.Int8()).To(Equal(int8(-1)))
		Expect(r.Int16()).To(Equal(int16(-256)))
		Expect(r.Int32()).To(Equal(int32(-65536)))
		Expect(r.Int64()).To(Equal(int64(-1)))
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
		Expect(r.Bool()).To(BeTrue())
		Expect(r.Bool()).To(BeFalse())
		Expect(r.Bool()).To(BeTrue())
	})

	It("Should read length-prefixed strings", func() {
		w := orc.NewWriter(0)
		w.String("hello")
		w.String("")
		w.String("world")
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(r.String()).To(Equal("hello"))
		Expect(r.String()).To(Equal(""))
		Expect(r.String()).To(Equal("world"))
	})

	It("Should return error on truncated string", func() {
		buf := make([]byte, 7)
		binary.BigEndian.PutUint32(buf, 100)
		buf[4], buf[5], buf[6] = 'a', 'b', 'c'
		r := orc.NewReader(bytesReader(buf))
		Expect(r.String()).Error().To(MatchError(io.ErrUnexpectedEOF))
	})

	It("Should reject string exceeding MaxStringLen in io.Reader mode", func() {
		prev := orc.MaxStringLen
		defer func() { orc.MaxStringLen = prev }()
		orc.MaxStringLen = 8
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, 9)
		r := orc.NewReader(bytesReader(buf))
		Expect(r.String()).Error().To(MatchError(orc.ErrExceedStringLen))
	})

	It("Should allow string within MaxStringLen in io.Reader mode", func() {
		prev := orc.MaxStringLen
		defer func() { orc.MaxStringLen = prev }()
		orc.MaxStringLen = 8
		w := orc.NewWriter(0)
		w.String("hello")
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(r.String()).To(Equal("hello"))
	})

	It("Should read uint16 in io.Reader mode", func() {
		w := orc.NewWriter(0)
		w.Uint16(1000)
		w.Uint16(65535)
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(r.Uint16()).To(Equal(uint16(1000)))
		Expect(r.Uint16()).To(Equal(uint16(65535)))
	})

	It("Should return error on truncated uint16 in io.Reader mode", func() {
		r := orc.NewReader(bytesReader([]byte{1}))
		Expect(r.Uint16()).Error().To(MatchError(io.ErrUnexpectedEOF))
	})

	It("Should read uint64 in io.Reader mode", func() {
		w := orc.NewWriter(0)
		w.Uint64(0x123456789ABCDEF0)
		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(r.Uint64()).To(Equal(uint64(0x123456789ABCDEF0)))
	})

	It("Should return error on truncated uint64 in io.Reader mode", func() {
		r := orc.NewReader(bytesReader([]byte{1, 2, 3}))
		Expect(r.Uint64()).Error().To(MatchError(io.ErrUnexpectedEOF))
	})

	Describe("CollectionLen", func() {
		It("Should read a valid collection length", func() {
			w := orc.NewWriter(0)
			w.Uint32(100)
			r := orc.NewReader(bytesReader(w.Bytes()))
			Expect(r.CollectionLen()).To(Equal(uint32(100)))
		})

		It("Should reject collection length exceeding MaxCollectionLen", func() {
			prev := orc.MaxCollectionLen
			defer func() { orc.MaxCollectionLen = prev }()
			orc.MaxCollectionLen = 5
			w := orc.NewWriter(0)
			w.Uint32(6)
			r := orc.NewReader(bytesReader(w.Bytes()))
			Expect(r.CollectionLen()).Error().To(MatchError(orc.ErrExceedCollectionLen))
		})

		It("Should read a valid collection length in direct mode", func() {
			w := orc.NewWriter(0)
			w.Uint32(50)
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(r.CollectionLen()).To(Equal(uint32(50)))
		})

		It("Should reject collection length exceeding MaxCollectionLen in direct mode", func() {
			prev := orc.MaxCollectionLen
			defer func() { orc.MaxCollectionLen = prev }()
			orc.MaxCollectionLen = 5
			w := orc.NewWriter(0)
			w.Uint32(6)
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(r.CollectionLen()).Error().To(MatchError(orc.ErrExceedCollectionLen))
		})

		It("Should return error on truncated data", func() {
			r := orc.NewReader(bytesReader([]byte{0, 0}))
			Expect(r.CollectionLen()).Error().To(HaveOccurred())
		})
	})

	Describe("Reset", func() {
		It("Should reset to use a new reader", func() {
			r := orc.NewReader(bytesReader([]byte{1}))
			Expect(r.Uint8()).To(Equal(uint8(1)))
			Expect(r.Uint8()).Error().To(MatchError(io.EOF))
			r.Reset(bytesReader([]byte{42, 0, 0, 1, 0}))
			Expect(r.Uint8()).To(Equal(uint8(42)))
			Expect(r.Uint32()).To(Equal(uint32(256)))
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
			Expect(r.Uint8()).To(Equal(uint8(255)))
			Expect(r.Uint32()).To(Equal(uint32(0xDEADBEEF)))
			Expect(r.Uint64()).To(Equal(uint64(0x123456789ABCDEF0)))
			Expect(r.String()).To(Equal("hello"))
			Expect(r.Bool()).To(BeTrue())
			Expect(r.Float32()).To(Equal(float32(1.5)))
			Expect(r.Float64()).To(Equal(float64(2.5)))
			Expect(r.Int32()).To(Equal(int32(-42)))
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
			Expect(r.Uint8()).To(Equal(uint8(1)))
			Expect(r.Uint32()).To(Equal(uint32(256)))
			Expect(r.Uint64()).To(Equal(uint64(1024)))
		})

		It("Should return EOF when no data remains", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{1})
			Expect(r.Uint8()).To(Equal(uint8(1)))
			Expect(r.Uint8()).Error().To(MatchError(io.EOF))
		})

		It("Should return ErrUnexpectedEOF on truncated data", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{1, 2})
			Expect(r.Uint32()).Error().To(MatchError(io.ErrUnexpectedEOF))
		})

		It("Should return EOF on truncated uint16 when at end of data", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{})
			Expect(r.Uint16()).Error().To(MatchError(io.EOF))
		})

		It("Should return ErrUnexpectedEOF on truncated uint16 with partial data", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{1})
			Expect(r.Uint16()).Error().To(MatchError(io.ErrUnexpectedEOF))
		})

		It("Should return EOF on truncated uint64 when at end of data", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{})
			Expect(r.Uint64()).Error().To(MatchError(io.EOF))
		})

		It("Should return ErrUnexpectedEOF on truncated uint64 with partial data", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{1, 2, 3})
			Expect(r.Uint64()).Error().To(MatchError(io.ErrUnexpectedEOF))
		})

		It("Should return EOF on Read with no data remaining", func() {
			r := orc.NewReader(nil)
			r.ResetBytes([]byte{})
			buf := make([]byte, 4)
			n, err := r.Read(buf)
			Expect(n).To(Equal(0))
			Expect(err).To(MatchError(io.EOF))
		})

		It("Should read length-prefixed strings", func() {
			w := orc.NewWriter(0)
			w.String("hello")
			w.String("")
			w.String("world")
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(r.String()).To(Equal("hello"))
			Expect(r.String()).To(Equal(""))
			Expect(r.String()).To(Equal("world"))
		})

		It("Should return error on truncated string", func() {
			buf := make([]byte, 7)
			binary.BigEndian.PutUint32(buf, 100)
			buf[4], buf[5], buf[6] = 'a', 'b', 'c'
			r := orc.NewReader(nil)
			r.ResetBytes(buf)
			Expect(r.String()).Error().To(MatchError(io.ErrUnexpectedEOF))
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
			Expect(r.Int8()).To(Equal(int8(-1)))
			Expect(r.Int16()).To(Equal(int16(-256)))
			Expect(r.Int32()).To(Equal(int32(-65536)))
			Expect(r.Int64()).To(Equal(int64(-1)))
			Expect(r.Float32()).To(Equal(float32(1.5)))
			Expect(r.Float64()).To(Equal(float64(2.5)))
			Expect(r.Bool()).To(BeTrue())
		})

		It("Should read arbitrary bytes", func() {
			data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
			r := orc.NewReader(nil)
			r.ResetBytes(data)
			buf := make([]byte, 4)
			Expect(r.Read(buf)).To(Equal(4))
			Expect(buf).To(Equal([]byte{1, 2, 3, 4}))
			Expect(r.Read(buf)).To(Equal(4))
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
			Expect(r.Uint8()).To(Equal(uint8(255)))
			Expect(r.Uint32()).To(Equal(uint32(0xDEADBEEF)))
			Expect(r.Uint64()).To(Equal(uint64(0x123456789ABCDEF0)))
			Expect(r.String()).To(Equal("hello"))
			Expect(r.Bool()).To(BeTrue())
			Expect(r.Float32()).To(Equal(float32(1.5)))
			Expect(r.Float64()).To(Equal(float64(2.5)))
			Expect(r.Int32()).To(Equal(int32(-42)))
		})

		Describe("ResetBytes after Reset", func() {
			It("Should switch modes correctly", func() {
				w := orc.NewWriter(0)
				w.Uint32(42)
				w.String("test")

				r := orc.NewReader(bytesReader(w.Bytes()))
				Expect(r.Uint32()).To(Equal(uint32(42)))
				Expect(r.String()).To(Equal("test"))

				r.ResetBytes(w.Bytes())
				Expect(r.Uint32()).To(Equal(uint32(42)))
				Expect(r.String()).To(Equal("test"))
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
