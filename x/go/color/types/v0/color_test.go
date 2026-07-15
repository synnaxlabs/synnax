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
	v0 "github.com/synnaxlabs/x/color/types/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Color", func() {
	Describe("Hex", func() {
		It("Should output a 6-character hex without alpha when alpha is 1", func() {
			c := v0.Color{R: 255, G: 0, B: 0, A: 1}
			Expect(c.Hex()).To(Equal("#ff0000"))
		})
		It("Should output an 8-character hex when alpha is not 1", func() {
			c := v0.Color{R: 255, G: 0, B: 0, A: 0.5}
			hex := c.Hex()
			Expect(hex).To(HavePrefix("#ff0000"))
			Expect(hex).To(HaveLen(9))
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
			Expect(json.Unmarshal([]byte(`{"r":255,"g":0,"b":0,"a":1}`), &c)).To(Succeed())
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
	})

	Describe("Msgpack", func() {
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
	})
})
