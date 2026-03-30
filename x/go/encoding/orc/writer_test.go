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
	"encoding/binary"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	It("Should correctly write primitive values", func() {
		w := orc.NewWriter(0)
		w.Uint8(1)
		w.Uint32(256)
		w.Uint64(1024)
		Expect(w.Len()).To(Equal(13))
		expected := make([]byte, 13)
		expected[0] = 1
		binary.BigEndian.PutUint32(expected[1:], 256)
		binary.BigEndian.PutUint64(expected[5:], 1024)
		Expect(w.Bytes()).To(Equal(expected))
	})

	It("Should auto-grow beyond initial capacity", func() {
		w := orc.NewWriter(4)
		w.Uint32(1)
		w.Uint32(2)
		w.Uint32(3)
		Expect(w.Len()).To(Equal(12))
	})

	It("Should write arbitrary bytes", func() {
		w := orc.NewWriter(0)
		w.Write([]byte{1, 2, 3, 4})
		Expect(w.Bytes()).To(Equal([]byte{1, 2, 3, 4}))
		w.Write([]byte{5, 6, 7, 8})
		Expect(w.Bytes()).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	})

	It("Should encode in big-endian byte order", func() {
		w := orc.NewWriter(0)
		w.Uint8(1)
		w.Uint32(1)
		w.Uint64(1)
		Expect(w.Bytes()).To(Equal([]byte{
			1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1,
		}))
	})

	It("Should write signed integers", func() {
		w := orc.NewWriter(0)
		w.Int8(-1)
		w.Int16(-256)
		w.Int32(-65536)
		w.Int64(-1)
		Expect(w.Len()).To(Equal(15))

		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(MustSucceed(r.Int8())).To(Equal(int8(-1)))
		Expect(MustSucceed(r.Int16())).To(Equal(int16(-256)))
		Expect(MustSucceed(r.Int32())).To(Equal(int32(-65536)))
		Expect(MustSucceed(r.Int64())).To(Equal(int64(-1)))
	})

	It("Should write floats", func() {
		w := orc.NewWriter(0)
		w.Float32(3.14)
		w.Float64(2.71828)
		Expect(w.Len()).To(Equal(12))

		r := orc.NewReader(bytesReader(w.Bytes()))
		f32 := MustSucceed(r.Float32())
		Expect(math.Abs(float64(f32) - 3.14)).To(BeNumerically("<", 0.001))
		f64 := MustSucceed(r.Float64())
		Expect(math.Abs(f64 - 2.71828)).To(BeNumerically("<", 0.00001))
	})

	It("Should write bool values", func() {
		w := orc.NewWriter(0)
		w.Bool(true)
		w.Bool(false)
		w.Bool(true)
		Expect(w.Bytes()).To(Equal([]byte{1, 0, 1}))
	})

	It("Should write length-prefixed strings", func() {
		w := orc.NewWriter(0)
		w.String("hello")
		w.String("")
		w.String("world")
		Expect(w.Len()).To(Equal(4 + 5 + 4 + 0 + 4 + 5))

		r := orc.NewReader(bytesReader(w.Bytes()))
		Expect(MustSucceed(r.String())).To(Equal("hello"))
		Expect(MustSucceed(r.String())).To(Equal(""))
		Expect(MustSucceed(r.String())).To(Equal("world"))
	})

	Describe("Reset", func() {
		It("Should clear the buffer for reuse", func() {
			w := orc.NewWriter(10)
			w.Write([]byte{1, 2, 3, 4})
			w.Reset()
			Expect(w.Len()).To(Equal(0))
			w.Write([]byte{5, 6, 7, 8})
			Expect(w.Bytes()).To(Equal([]byte{5, 6, 7, 8}))
		})
	})

	Describe("Resize", func() {
		It("Should ensure capacity without losing data", func() {
			w := orc.NewWriter(5)
			w.Write([]byte{1, 2, 3, 4, 5})
			w.Resize(20)
			w.Write([]byte{6, 7, 8, 9, 10})
			Expect(w.Bytes()).To(Equal([]byte{
				1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
			}))
		})

		It("Should be a no-op when capacity is sufficient", func() {
			w := orc.NewWriter(100)
			w.Write([]byte{1, 2, 3})
			w.Resize(50)
			Expect(w.Bytes()).To(Equal([]byte{1, 2, 3}))
		})
	})

	Describe("Copy", func() {
		It("Should return an owned copy of the buffer", func() {
			w := orc.NewWriter(0)
			w.Uint32(42)
			c := w.Copy()
			Expect(c).To(Equal(w.Bytes()))
			w.Reset()
			w.Uint32(99)
			Expect(c).To(Equal([]byte{0, 0, 0, 42}))
		})
	})
})
