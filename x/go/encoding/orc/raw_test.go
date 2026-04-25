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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Raw", func() {
	Describe("NewRaw", func() {
		It("Should return a validation error when the bytes do not have the magic header", func() {
			Expect(orc.NewRaw([]byte{1, 2, 3})).Error().To(
				SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("data was not encoded using orc")),
				))
		})

		It("Should strip the magic header when it matches", func() {
			bytes := append(magic[:], 1, 2, 3)
			d := MustSucceed(orc.NewRaw(bytes))
			Expect(d).To(Equal(orc.Raw{1, 2, 3}))
		})
	})

	Describe("Skip", func() {
		It("Should skip fixed-size bytes", func() {
			data := orc.Raw([]byte{1, 2, 3, 4, 5})
			rest := data.Skip(3)
			Expect(rest).To(Equal(orc.Raw([]byte{4, 5})))
		})

		It("Should return nil when data is too short", func() {
			data := orc.Raw([]byte{1, 2})
			Expect(data.Skip(5)).To(BeNil())
		})
	})

	Describe("SkipString", func() {
		It("Should skip a length-prefixed string", func() {
			w := orc.NewWriter(0)
			w.String("hello")
			w.Uint32(42)
			r := orc.Raw(w.Bytes())
			rest := r.SkipString()
			Expect(rest).ToNot(BeNil())
			v, _ := rest.ReadUint32()
			Expect(v).To(Equal(uint32(42)))
		})

		It("Should return nil on truncated data", func() {
			Expect(orc.Raw([]byte{0, 0, 0, 10, 1, 2}).SkipString()).To(BeNil())
		})

		It("Should return nil on insufficient header", func() {
			Expect(orc.Raw([]byte{0, 0}).SkipString()).To(BeNil())
		})
	})

	Describe("SkipStrings", func() {
		It("Should skip multiple strings", func() {
			w := orc.NewWriter(0)
			w.String("one")
			w.String("two")
			w.String("three")
			w.Uint8(99)
			r := orc.Raw(w.Bytes())
			rest := r.SkipStrings(3)
			Expect(rest).ToNot(BeNil())
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(99)))
		})

		It("Should return nil if any string is truncated", func() {
			w := orc.NewWriter(0)
			w.String("ok")
			r := orc.Raw(w.Bytes())
			Expect(r.SkipStrings(2)).To(BeNil())
		})
	})

	Describe("SkipBool", func() {
		It("Should skip 1 byte", func() {
			data := orc.Raw([]byte{1, 42})
			rest := data.SkipBool()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipUint16", func() {
		It("Should skip 2 bytes", func() {
			data := orc.Raw([]byte{0, 1, 42})
			rest := data.SkipUint16()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipUint32", func() {
		It("Should skip 4 bytes", func() {
			data := orc.Raw([]byte{0, 0, 0, 1, 42})
			rest := data.SkipUint32()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipUint64", func() {
		It("Should skip 8 bytes", func() {
			data := orc.Raw([]byte{0, 0, 0, 0, 0, 0, 0, 1, 42})
			rest := data.SkipUint64()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipUint8", func() {
		It("Should skip 1 byte", func() {
			data := orc.Raw([]byte{99, 42})
			rest := data.SkipUint8()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipInt8", func() {
		It("Should skip 1 byte", func() {
			data := orc.Raw([]byte{0xFF, 42})
			rest := data.SkipInt8()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipInt16", func() {
		It("Should skip 2 bytes", func() {
			data := orc.Raw([]byte{0xFF, 0x00, 42})
			rest := data.SkipInt16()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipInt32", func() {
		It("Should skip 4 bytes", func() {
			data := orc.Raw([]byte{0, 0, 0, 1, 42})
			rest := data.SkipInt32()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipInt64", func() {
		It("Should skip 8 bytes", func() {
			data := orc.Raw([]byte{0, 0, 0, 0, 0, 0, 0, 1, 42})
			rest := data.SkipInt64()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipFloat32", func() {
		It("Should skip 4 bytes", func() {
			data := orc.Raw([]byte{0, 0, 0, 0, 42})
			rest := data.SkipFloat32()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("SkipFloat64", func() {
		It("Should skip 8 bytes", func() {
			data := orc.Raw([]byte{0, 0, 0, 0, 0, 0, 0, 0, 42})
			rest := data.SkipFloat64()
			v, _ := rest.ReadUint8()
			Expect(v).To(Equal(uint8(42)))
		})
	})

	Describe("ReadUint8", func() {
		It("Should return nil rest on empty data", func() {
			_, rest := orc.Raw(nil).ReadUint8()
			Expect(rest).To(BeNil())
		})
	})

	Describe("ReadUint16", func() {
		It("Should return nil rest on short data", func() {
			_, rest := orc.Raw([]byte{1}).ReadUint16()
			Expect(rest).To(BeNil())
		})
	})

	Describe("ReadUint32", func() {
		It("Should return nil rest on short data", func() {
			_, rest := orc.Raw([]byte{1, 2}).ReadUint32()
			Expect(rest).To(BeNil())
		})
	})

	Describe("ReadUint64", func() {
		It("Should return nil rest on short data", func() {
			_, rest := orc.Raw([]byte{1, 2, 3, 4}).ReadUint64()
			Expect(rest).To(BeNil())
		})
	})

	Describe("ReadLenPrefixed", func() {
		It("Should return nil on short header", func() {
			val, rest := orc.Raw([]byte{0, 0}).ReadLenPrefixed()
			Expect(val).To(BeNil())
			Expect(rest).To(BeNil())
		})

		It("Should return nil on truncated body", func() {
			val, rest := orc.Raw([]byte{0, 0, 0, 10, 1, 2}).ReadLenPrefixed()
			Expect(val).To(BeNil())
			Expect(rest).To(BeNil())
		})
	})

	Describe("ReadString", func() {
		It("Should read a length-prefixed string", func() {
			w := orc.NewWriter(0)
			w.String("hello")
			w.String("world")
			r := orc.Raw(w.Bytes())
			val, rest := r.ReadString()
			Expect(string(val)).To(Equal("hello"))
			val, rest = rest.ReadString()
			Expect(string(val)).To(Equal("world"))
			Expect(rest).To(BeEmpty())
		})
	})

	Describe("ReadBool", func() {
		It("Should read a boolean", func() {
			v, rest := orc.Raw([]byte{1, 0}).ReadBool()
			Expect(v).To(BeTrue())
			v, _ = rest.ReadBool()
			Expect(v).To(BeFalse())
		})

		It("Should return nil rest on empty data", func() {
			_, rest := orc.Raw(nil).ReadBool()
			Expect(rest).To(BeNil())
		})
	})

	Describe("ReadUint16", func() {
		It("Should read a big-endian uint16", func() {
			buf := make([]byte, 3)
			binary.BigEndian.PutUint16(buf, 1000)
			buf[2] = 42
			v, rest := orc.Raw(buf).ReadUint16()
			Expect(v).To(Equal(uint16(1000)))
			Expect(rest).To(HaveLen(1))
		})
	})

	Describe("ReadUint32", func() {
		It("Should read a big-endian uint32", func() {
			buf := make([]byte, 5)
			binary.BigEndian.PutUint32(buf, 100000)
			buf[4] = 42
			v, rest := orc.Raw(buf).ReadUint32()
			Expect(v).To(Equal(uint32(100000)))
			Expect(rest).To(HaveLen(1))
		})
	})

	Describe("ReadUint64", func() {
		It("Should read a big-endian uint64", func() {
			buf := make([]byte, 9)
			binary.BigEndian.PutUint64(buf, 123456789)
			buf[8] = 42
			v, rest := orc.Raw(buf).ReadUint64()
			Expect(v).To(Equal(uint64(123456789)))
			Expect(rest).To(HaveLen(1))
		})
	})

	Describe("ReadInt8", func() {
		It("Should read a signed int8", func() {
			v, _ := orc.Raw([]byte{0xFF}).ReadInt8()
			Expect(v).To(Equal(int8(-1)))
		})
	})

	Describe("ReadInt16", func() {
		It("Should read a signed int16", func() {
			w := orc.NewWriter(0)
			w.Int16(-256)
			v, _ := orc.Raw(w.Bytes()).ReadInt16()
			Expect(v).To(Equal(int16(-256)))
		})
	})

	Describe("ReadInt32", func() {
		It("Should read a signed int32", func() {
			w := orc.NewWriter(0)
			w.Int32(-65536)
			v, _ := orc.Raw(w.Bytes()).ReadInt32()
			Expect(v).To(Equal(int32(-65536)))
		})
	})

	Describe("ReadInt64", func() {
		It("Should read a signed int64", func() {
			w := orc.NewWriter(0)
			w.Int64(-1)
			v, _ := orc.Raw(w.Bytes()).ReadInt64()
			Expect(v).To(Equal(int64(-1)))
		})
	})

	Describe("ReadFloat32", func() {
		It("Should read a float32", func() {
			w := orc.NewWriter(0)
			w.Float32(3.14)
			v, _ := orc.Raw(w.Bytes()).ReadFloat32()
			Expect(v).To(BeNumerically("~", float32(3.14), 1e-6))
		})
	})

	Describe("ReadFloat64", func() {
		It("Should read a float64", func() {
			w := orc.NewWriter(0)
			w.Float64(2.718281828)
			v, _ := orc.Raw(w.Bytes()).ReadFloat64()
			Expect(v).To(BeNumerically("~", 2.718281828, 1e-9))
		})
	})

	Describe("Mixed field navigation", func() {
		It("Should navigate a record with mixed field types", func() {
			w := orc.NewWriter(0)
			w.String("name")
			w.Uint16(42)
			w.Bool(true)
			w.String("target")

			r := orc.Raw(w.Bytes())
			r = r.SkipString()
			r = r.SkipUint16()
			r = r.SkipBool()
			val, rest := r.ReadString()
			Expect(string(val)).To(Equal("target"))
			Expect(rest).To(BeEmpty())
		})
	})
})
