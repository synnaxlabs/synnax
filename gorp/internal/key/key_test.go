// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package key_test

import (
	"math"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

type widget struct{}

type renamed struct{}

func (renamed) CustomTypeName() string { return "Gadget" }

type name string

func roundTrip[K interface {
	types.Primitive
	comparable
}](c key.Codec[K], k K) K {
	b := append([]byte(nil), c.Encode(k)...)
	return c.Decode(b)
}

var _ = Describe("Key", func() {
	Describe("Prefix", func() {
		It("Should start every table key with gorp. and the type name", func() {
			Expect(string(key.Prefix[widget]())).To(Equal("gorp.widget"))
		})

		It("Should use a custom type name", func() {
			Expect(string(key.Prefix[renamed]())).To(Equal("gorp.Gadget"))
		})
	})

	Describe("Migrations", func() {
		It("Should record a namespace's migrations under gorp.migration.", func() {
			Expect(string(key.Migrations("widget"))).To(Equal("gorp.migration.widget"))
		})
	})

	Describe("LegacyPrefix", func() {
		It("Should encode the type name with msgpack", func(ctx SpecContext) {
			Expect(key.LegacyPrefix(ctx, "Widget")).
				To(Equal(MustSucceed(msgpack.Codec.Encode(ctx, "Widget"))))
		})
	})

	Describe("Codec", func() {
		prefix := []byte("gorp.widget")

		It("Should return the table prefix", func() {
			c := key.NewCodec[uint32](prefix)
			Expect(c.Prefix()).To(Equal(prefix))
		})

		DescribeTable("Should encode fixed-size keys big-endian after the prefix",
			func(encode func() []byte, suffix []byte) {
				Expect(encode()).To(Equal(slices.Concat(prefix, suffix)))
			},
			Entry("uint8", func() []byte {
				c := key.NewCodec[uint8](prefix)
				return c.Encode(0xAB)
			}, []byte{0xAB}),
			Entry("uint16", func() []byte {
				c := key.NewCodec[uint16](prefix)
				return c.Encode(0x0102)
			}, []byte{0x01, 0x02}),
			Entry("uint32", func() []byte {
				c := key.NewCodec[uint32](prefix)
				return c.Encode(0x01020304)
			}, []byte{0x01, 0x02, 0x03, 0x04}),
			Entry("uint64", func() []byte {
				c := key.NewCodec[uint64](prefix)
				return c.Encode(0x0102030405060708)
			}, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}),
			Entry("int32 below zero", func() []byte {
				c := key.NewCodec[int32](prefix)
				return c.Encode(-1)
			}, []byte{0xFF, 0xFF, 0xFF, 0xFF}),
		)

		It("Should encode a string key as its raw bytes", func() {
			c := key.NewCodec[string](prefix)
			Expect(string(c.Encode("abc"))).To(Equal("gorp.widgetabc"))
		})

		It("Should reverse the bytes of a 16-byte key", func() {
			c := key.NewCodec[[16]byte](prefix)
			var k [16]byte
			for i := range k {
				k[i] = byte(i)
			}
			suffix := c.Encode(k)[len(prefix):]
			for i := range suffix {
				Expect(suffix[i]).To(Equal(byte(15 - i)))
			}
		})

		DescribeTable("Should decode what it encodes",
			func(check func()) { check() },
			Entry("int8", func() {
				Expect(roundTrip(key.NewCodec[int8](prefix), math.MinInt8)).
					To(BeEquivalentTo(math.MinInt8))
			}),
			Entry("int16", func() {
				Expect(roundTrip(key.NewCodec[int16](prefix), -12345)).
					To(BeEquivalentTo(-12345))
			}),
			Entry("int64", func() {
				Expect(roundTrip(key.NewCodec[int64](prefix), math.MaxInt64)).
					To(BeEquivalentTo(int64(math.MaxInt64)))
			}),
			Entry("float64", func() {
				Expect(roundTrip(key.NewCodec[float64](prefix), 3.5)).To(Equal(3.5))
			}),
			Entry("named string", func() {
				Expect(roundTrip(key.NewCodec[name](prefix), "a/b:c")).
					To(Equal(name("a/b:c")))
			}),
			Entry("empty string", func() {
				Expect(roundTrip(key.NewCodec[string](prefix), "")).To(BeEmpty())
			}),
			Entry("16-byte array", func() {
				k := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
				Expect(roundTrip(key.NewCodec[[16]byte](prefix), k)).To(Equal(k))
			}),
		)

		It("Should reuse its buffer across calls", func() {
			c := key.NewCodec[uint16](prefix)
			first := c.Encode(1)
			c.Encode(2)
			Expect(first[len(prefix):]).To(Equal([]byte{0x00, 0x02}))
		})

		It("Should grow its buffer for longer string keys", func() {
			c := key.NewCodec[string](prefix)
			c.Encode("a")
			Expect(string(c.Encode("a much longer key"))).
				To(Equal("gorp.widgeta much longer key"))
		})

		Describe("HasPrefix", func() {
			It("Should match every key against an empty prefix", func() {
				c := key.NewCodec[uint32](prefix)
				Expect(c.HasPrefix(7, nil)).To(BeTrue())
			})

			It("Should compare the leading bytes of a fixed-size encoding", func() {
				c := key.NewCodec[uint32](prefix)
				Expect(c.HasPrefix(0x01020304, []byte{0x01, 0x02})).To(BeTrue())
				Expect(c.HasPrefix(0x01020304, []byte{0x02})).To(BeFalse())
			})

			It("Should reject a prefix longer than a fixed-size key", func() {
				c := key.NewCodec[uint16](prefix)
				Expect(c.HasPrefix(0x0102, []byte{0x01, 0x02, 0x03})).To(BeFalse())
			})

			It("Should compare the leading bytes of a string key", func() {
				c := key.NewCodec[string](prefix)
				Expect(c.HasPrefix("rack/1", []byte("rack/"))).To(BeTrue())
				Expect(c.HasPrefix("rack/1", []byte("task/"))).To(BeFalse())
				Expect(c.HasPrefix("ra", []byte("rack/"))).To(BeFalse())
			})
		})
	})
})
