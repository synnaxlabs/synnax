// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/color/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Color", func() {
	Describe("FromHex", func() {
		It("Should parse a 6-character hex string", func() {
			c := MustSucceed(v0.FromHex("#ff0000"))
			Expect(c).To(Equal(v0.Color{R: 255, G: 0, B: 0, A: 1}))
		})
		It("Should parse a 6-character hex string without hash", func() {
			c := MustSucceed(v0.FromHex("00ff00"))
			Expect(c).To(Equal(v0.Color{R: 0, G: 255, B: 0, A: 1}))
		})
		It("Should parse an 8-character hex string with alpha", func() {
			c := MustSucceed(v0.FromHex("#ff000080"))
			Expect(c.R).To(Equal(uint8(255)))
			Expect(c.G).To(Equal(uint8(0)))
			Expect(c.B).To(Equal(uint8(0)))
			Expect(c.A).To(BeNumerically("~", 128.0/255.0, 0.01))
		})
		It("Should return an error for an invalid hex string", func() {
			Expect(v0.FromHex("#xyz")).Error().To(MatchError(validate.ErrValidation))
		})
		It("Should return an error for wrong length", func() {
			Expect(v0.FromHex("#12345")).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("IsZero", func() {
		It("Should return true for the zero value", func() {
			Expect(v0.Color{}.IsZero()).To(BeTrue())
		})
		It("Should return false when R is non-zero", func() {
			Expect(v0.Color{R: 1}.IsZero()).To(BeFalse())
		})
		It("Should return false when A is non-zero", func() {
			Expect(v0.Color{A: 0.5}.IsZero()).To(BeFalse())
		})
	})

	Describe("JSON", func() {
		It("Should marshal to struct format", func() {
			c := v0.Color{R: 255, G: 128, B: 0, A: 1}
			data := MustSucceed(json.Marshal(c))
			Expect(string(data)).To(ContainSubstring(`"r":255`))
		})
		It("Should unmarshal from a hex string", func() {
			var c v0.Color
			Expect(json.Unmarshal([]byte(`"#ff8000"`), &c)).To(Succeed())
			Expect(c.R).To(Equal(uint8(255)))
			Expect(c.G).To(Equal(uint8(128)))
			Expect(c.B).To(Equal(uint8(0)))
			Expect(c.A).To(Equal(1.0))
		})
		It("Should unmarshal from an array", func() {
			var c v0.Color
			Expect(json.Unmarshal([]byte(`[255, 0, 0, 0.5]`), &c)).To(Succeed())
			Expect(c.R).To(Equal(uint8(255)))
			Expect(c.A).To(Equal(0.5))
		})
		It("Should unmarshal from an object", func() {
			var c v0.Color
			Expect(
				json.Unmarshal([]byte(`{"r":255,"g":0,"b":0,"a":1}`), &c),
			).To(Succeed())
			Expect(c.R).To(Equal(uint8(255)))
			Expect(c.A).To(Equal(1.0))
		})
		It("Should round-trip JSON correctly", func() {
			original := v0.Color{R: 100, G: 200, B: 50, A: 0.75}
			data := MustSucceed(json.Marshal(original))
			var decoded v0.Color
			Expect(json.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
		It("Should unmarshal an empty string as the zero color", func() {
			c := v0.Color{R: 1, G: 2, B: 3, A: 0.5}
			Expect(json.Unmarshal([]byte(`""`), &c)).To(Succeed())
			Expect(c).To(Equal(v0.Color{}))
		})
		It("Should unmarshal a JSON null as the zero color", func() {
			c := v0.Color{R: 1, G: 2, B: 3, A: 0.5}
			Expect(json.Unmarshal([]byte(`null`), &c)).To(Succeed())
			Expect(c).To(Equal(v0.Color{}))
		})
		It("Should unmarshal a legacy rgba255 object", func() {
			var c v0.Color
			Expect(
				json.Unmarshal([]byte(`{"rgba255":[5,10,15,0.8]}`), &c),
			).To(Succeed())
			Expect(c).To(Equal(v0.Color{R: 5, G: 10, B: 15, A: 0.8}))
		})
		It("Should unmarshal a 3-element legacy rgba255 object", func() {
			var c v0.Color
			Expect(json.Unmarshal([]byte(`{"rgba255":[5,10,15]}`), &c)).To(Succeed())
			Expect(c).To(Equal(v0.Color{R: 5, G: 10, B: 15, A: 1}))
		})

		DescribeTable("Should lift a legacy 0-255 alpha onto the 0-1 scale",
			func(data string, expected v0.Color) {
				var c v0.Color
				Expect(json.Unmarshal([]byte(data), &c)).To(Succeed())
				Expect(c.R).To(Equal(expected.R))
				Expect(c.G).To(Equal(expected.G))
				Expect(c.B).To(Equal(expected.B))
				Expect(c.A).To(BeNumerically("~", expected.A, 0.001))
			},
			Entry("array with alpha 255",
				`[28, 28, 28, 255]`, v0.Color{R: 28, G: 28, B: 28, A: 1}),
			Entry("array with mid-range alpha",
				`[255, 0, 0, 128]`, v0.Color{R: 255, G: 0, B: 0, A: 128.0 / 255}),
			Entry("array with alpha just above 1 clamps to opaque",
				`[255, 0, 0, 1.5]`, v0.Color{R: 255, G: 0, B: 0, A: 1}),
			Entry("array with alpha at the clamp boundary",
				`[255, 0, 0, 2]`, v0.Color{R: 255, G: 0, B: 0, A: 1}),
			Entry("array with alpha just above the clamp boundary divides",
				`[255, 0, 0, 2.01]`, v0.Color{R: 255, G: 0, B: 0, A: 2.01 / 255}),
			Entry("array with alpha 1 stays opaque",
				`[255, 0, 0, 1]`, v0.Color{R: 255, G: 0, B: 0, A: 1}),
			Entry("array with fractional alpha untouched",
				`[255, 0, 0, 0.25]`, v0.Color{R: 255, G: 0, B: 0, A: 0.25}),
			Entry("object with alpha 255",
				`{"r":5,"g":5,"b":5,"a":255}`, v0.Color{R: 5, G: 5, B: 5, A: 1}),
			Entry("rgba255 object with alpha 255",
				`{"rgba255":[5,5,5,255]}`, v0.Color{R: 5, G: 5, B: 5, A: 1}),
		)

		DescribeTable("Should reject an alpha above the 0-255 scale",
			func(data string) {
				var c v0.Color
				Expect(json.Unmarshal([]byte(data), &c)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("array", `[255, 0, 0, 300]`),
			Entry("object", `{"r":5,"g":5,"b":5,"a":300}`),
			Entry("rgba255 object", `{"rgba255":[5,5,5,300]}`),
		)
	})

	Describe("MessagePack", func() {
		It("Should decode from a string (backwards compat)", func() {
			encoded := MustSucceed(msgpack.Marshal("#ff0000"))
			var c v0.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c).To(Equal(v0.Color{R: 255, G: 0, B: 0, A: 1}))
		})
		It("Should decode from a string with alpha (backwards compat)", func() {
			encoded := MustSucceed(msgpack.Marshal("#00ff0080"))
			var c v0.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c.R).To(Equal(uint8(0)))
			Expect(c.G).To(Equal(uint8(255)))
			Expect(c.B).To(Equal(uint8(0)))
			Expect(c.A).To(BeNumerically("~", 128.0/255.0, 0.01))
		})
		It("Should round-trip msgpack correctly", func() {
			original := v0.Color{R: 100, G: 200, B: 50, A: 0.75}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.Color
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
		It("Should decode #000000 as black with full alpha", func() {
			encoded := MustSucceed(msgpack.Marshal("#000000"))
			var c v0.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c).To(Equal(v0.Color{R: 0, G: 0, B: 0, A: 1}))
		})
		It("Should decode an empty string as zero color", func() {
			encoded := MustSucceed(msgpack.Marshal(""))
			var c v0.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c).To(Equal(v0.Color{}))
		})
		It("Should decode nil as zero color", func() {
			encoded := MustSucceed(msgpack.Marshal(nil))
			var c v0.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c).To(Equal(v0.Color{}))
		})
		It("Should lift a legacy 0-255 alpha in an array", func() {
			encoded := MustSucceed(msgpack.Marshal([]any{28, 28, 28, 255}))
			var c v0.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c).To(Equal(v0.Color{R: 28, G: 28, B: 28, A: 1}))
		})
		It("Should lift a legacy 0-255 alpha in a map", func() {
			encoded := MustSucceed(msgpack.Marshal(
				map[string]any{"r": 5, "g": 5, "b": 5, "a": 255},
			))
			var c v0.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c).To(Equal(v0.Color{R: 5, G: 5, B: 5, A: 1}))
		})
	})
})
