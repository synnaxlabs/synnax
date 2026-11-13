// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/color/versions/v0"
	v2 "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

func alphaEntries() []TableEntry {
	entries := make([]TableEntry, 256)
	for a := range entries {
		entries[a] = Entry(nil, a)
	}
	return entries
}

var _ = Describe("Color", func() {
	Describe("Parse", func() {
		DescribeTable("Should parse a hex string",
			func(s string, expected v2.Color) {
				Expect(v2.Parse(s)).To(Equal(expected))
			},
			Entry("6 digits", "#ff8000", v2.Color{R: 255, G: 128, A: 255}),
			Entry("6 digits without a hash", "00ff00", v2.Color{G: 255, A: 255}),
			Entry("8 digits", "#ff000080", v2.Color{R: 255, A: 128}),
			Entry("8 digits upper case", "#FF0000FF", v2.Color{R: 255, A: 255}),
			Entry("8 digits zero", "#00000000", v2.Color{}),
		)
		DescribeTable("Should reject an invalid hex string",
			func(s string) {
				Expect(v2.Parse(s)).Error().To(MatchError(validate.ErrValidation))
			},
			Entry("odd length", "#12345"),
			Entry("not hex", "#xyzxyz"),
		)
	})

	Describe("Hex", func() {
		DescribeTable("Should write lower case hex",
			func(c v2.Color, expected string) { Expect(c.Hex()).To(Equal(expected)) },
			Entry("opaque", v2.Color{R: 255, G: 128, B: 1, A: 255}, "#ff8001"),
			Entry("translucent", v2.Color{R: 10, G: 11, B: 12, A: 128}, "#0a0b0c80"),
			Entry("zero", v2.Color{}, "#00000000"),
		)
	})

	Describe("Uint32", func() {
		It("Should pack the color as 0xRRGGBBAA", func() {
			c := v2.Color{R: 0x12, G: 0x34, B: 0x56, A: 0x78}
			Expect(c.Uint32()).To(Equal(uint32(0x12345678)))
			Expect(v2.FromUint32(0x12345678)).To(Equal(c))
		})
	})

	Describe("Orc", func() {
		It("Should store the color as one big-endian packed uint32", func() {
			w := orc.NewWriter(0)
			c := v2.Color{R: 0x12, G: 0x34, B: 0x56, A: 0x78}
			Expect(c.EncodeOrc(w)).To(Succeed())
			Expect(w.Bytes()).To(HaveLen(4))
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			var decoded v2.Color
			Expect(decoded.DecodeOrc(r)).To(Succeed())
			Expect(decoded).To(Equal(c))
		})
	})

	Describe("JSON", func() {
		It("Should marshal to a hex string", func() {
			c := v2.Color{R: 255, G: 128, A: 255}
			Expect(json.Marshal(c)).To(Equal([]byte(`"#ff8000"`)))
		})
		It("Should unmarshal null into a nil pointer", func() {
			v := struct {
				C *v2.Color `json:"c"`
			}{C: &v2.Color{R: 1}}
			Expect(json.Unmarshal([]byte(`{"c":null}`), &v)).To(Succeed())
			Expect(v.C).To(BeNil())
		})
		It("Should marshal a nil pointer as null", func() {
			var c *v2.Color
			Expect(json.Marshal(c)).To(Equal([]byte(`null`)))
		})
		DescribeTable("Should round-trip every alpha",
			func(a int) {
				original := v2.Color{R: 1, G: 2, B: 3, A: uint8(a)}
				var decoded v2.Color
				Expect(json.Unmarshal(MustSucceed(json.Marshal(original)), &decoded)).
					To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			alphaEntries(),
		)
		DescribeTable(
			"Should unmarshal",
			func(data string, expected v2.Color) {
				c := v2.Color{R: 9, G: 9, B: 9, A: 9}
				Expect(json.Unmarshal([]byte(data), &c)).To(Succeed())
				Expect(c).To(Equal(expected))
			},
			Entry("6 digit hex", `"#ff8000"`, v2.Color{R: 255, G: 128, A: 255}),
			Entry("8 digit hex", `"#ff800080"`, v2.Color{R: 255, G: 128, A: 128}),
		)
		DescribeTable("Should reject",
			func(data string) {
				var c v2.Color
				Expect(json.Unmarshal([]byte(data), &c)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("null", `null`),
			Entry("an empty string", `""`),
			Entry("an empty object", `{}`),
			Entry("a tuple", `[1, 2, 3, 0.5]`),
			Entry("an object", `{"r":1,"g":2,"b":3,"a":0.5}`),
			Entry("a 5 digit hex string", `"#12345"`),
			Entry("a number", `7`),
		)
		It("Should decode with the frozen v0 decoder", func() {
			c := v2.Color{R: 255, G: 128, B: 0, A: 128}
			var old v0.Color
			Expect(json.Unmarshal(MustSucceed(json.Marshal(c)), &old)).To(Succeed())
			Expect(old.R).To(Equal(uint8(255)))
			Expect(old.G).To(Equal(uint8(128)))
			Expect(old.A).To(BeNumerically("~", 128.0/255, 1e-9))
		})
	})

	Describe("MarshalText", func() {
		It("Should return the hex string", func() {
			Expect(v2.Color{R: 1, A: 255}.MarshalText()).To(Equal([]byte("#010000")))
		})
	})

	Describe("MessagePack", func() {
		It("Should encode as an 8 digit hex string", func() {
			c := v2.Color{R: 255, G: 128, A: 64}
			var s string
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(c)), &s)).To(Succeed())
			Expect(s).To(Equal("#ff800040"))
		})
		It("Should round-trip a color", func() {
			original := v2.Color{R: 100, G: 200, B: 50, A: 191}
			var decoded v2.Color
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(original)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(original))
		})
		DescribeTable(
			"Should decode",
			func(value any, expected v2.Color) {
				c := v2.Color{R: 9, G: 9, B: 9, A: 9}
				Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(value)), &c)).
					To(Succeed())
				Expect(c).To(Equal(expected))
			},
			Entry("6 digit hex", "#000000", v2.Color{A: 255}),
		)
		DescribeTable("Should reject",
			func(value any) {
				var c v2.Color
				Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(value)), &c)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("an empty string", ""),
			Entry("a string that is not hex", "red"),
			Entry("a tuple", []any{1, 2, 3, 0.5}),
			Entry("a map", map[string]any{"r": 1, "g": 2, "b": 3, "a": 1}),
		)
	})
})
