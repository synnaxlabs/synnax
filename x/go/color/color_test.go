// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/synnaxlabs/x/color"
)

var _ = Describe("Color", func() {
	Describe("FromHex", func() {
		It("Should parse a 6-character hex string", func() {
			c, err := color.FromHex("#ff0000")
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 255, G: 0, B: 0, A: 1}))
		})
		It("Should parse a 6-character hex string without hash", func() {
			c, err := color.FromHex("00ff00")
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 0, G: 255, B: 0, A: 1}))
		})
		It("Should parse an 8-character hex string with alpha", func() {
			c, err := color.FromHex("#ff000080")
			Expect(err).ToNot(HaveOccurred())
			Expect(c.R).To(Equal(uint8(255)))
			Expect(c.G).To(Equal(uint8(0)))
			Expect(c.B).To(Equal(uint8(0)))
			Expect(c.A).To(BeNumerically("~", 128.0/255.0, 0.01))
		})
		It("Should return an error for an invalid hex string", func() {
			_, err := color.FromHex("#xyz")
			Expect(err).To(HaveOccurred())
		})
		It("Should return an error for wrong length", func() {
			_, err := color.FromHex("#12345")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("MustFromHex", func() {
		It("Should parse a valid hex string", func() {
			c := color.MustFromHex("#0000ff")
			Expect(c).To(Equal(color.Color{R: 0, G: 0, B: 255, A: 1}))
		})
		It("Should panic on an invalid hex string", func() {
			Expect(func() { color.MustFromHex("invalid") }).To(Panic())
		})
	})

	Describe("Hex", func() {
		It("Should output a 6-character hex without alpha when alpha is 1", func() {
			c := color.Color{R: 255, G: 0, B: 0, A: 1}
			Expect(c.Hex()).To(Equal("#ff0000"))
		})
		It("Should output an 8-character hex when alpha is not 1", func() {
			c := color.Color{R: 255, G: 0, B: 0, A: 0.5}
			hex := c.Hex()
			Expect(hex).To(HavePrefix("#ff0000"))
			Expect(len(hex)).To(Equal(9))
		})
	})

	Describe("IsZero", func() {
		It("Should return true for the zero value", func() {
			Expect(color.Color{}.IsZero()).To(BeTrue())
		})
		It("Should return false when R is non-zero", func() {
			Expect(color.Color{R: 1}.IsZero()).To(BeFalse())
		})
		It("Should return false when A is non-zero", func() {
			Expect(color.Color{A: 0.5}.IsZero()).To(BeFalse())
		})
	})

	Describe("JSON", func() {
		It("Should marshal to struct format", func() {
			c := color.Color{R: 255, G: 128, B: 0, A: 1}
			data, err := json.Marshal(c)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`"r":255`))
		})
		It("Should unmarshal from a hex string", func() {
			var c color.Color
			Expect(json.Unmarshal([]byte(`"#ff8000"`), &c)).To(Succeed())
			Expect(c.R).To(Equal(uint8(255)))
			Expect(c.G).To(Equal(uint8(128)))
			Expect(c.B).To(Equal(uint8(0)))
			Expect(c.A).To(Equal(1.0))
		})
		It("Should unmarshal from an array", func() {
			var c color.Color
			Expect(json.Unmarshal([]byte(`[255, 0, 0, 0.5]`), &c)).To(Succeed())
			Expect(c.R).To(Equal(uint8(255)))
			Expect(c.A).To(Equal(0.5))
		})
		It("Should unmarshal from an object", func() {
			var c color.Color
			Expect(json.Unmarshal([]byte(`{"r":255,"g":0,"b":0,"a":1}`), &c)).To(Succeed())
			Expect(c.R).To(Equal(uint8(255)))
			Expect(c.A).To(Equal(1.0))
		})
		It("Should round-trip JSON correctly", func() {
			original := color.Color{R: 100, G: 200, B: 50, A: 0.75}
			data, err := json.Marshal(original)
			Expect(err).ToNot(HaveOccurred())
			var decoded color.Color
			Expect(json.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
	})

	Describe("Msgpack", func() {
		It("Should decode from a string (backwards compat)", func() {
			encoded, err := msgpack.Marshal("#ff0000")
			Expect(err).ToNot(HaveOccurred())
			var c color.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c).To(Equal(color.Color{R: 255, G: 0, B: 0, A: 1}))
		})
		It("Should decode from a string with alpha (backwards compat)", func() {
			encoded, err := msgpack.Marshal("#00ff0080")
			Expect(err).ToNot(HaveOccurred())
			var c color.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c.R).To(Equal(uint8(0)))
			Expect(c.G).To(Equal(uint8(255)))
			Expect(c.B).To(Equal(uint8(0)))
			Expect(c.A).To(BeNumerically("~", 128.0/255.0, 0.01))
		})
		It("Should round-trip msgpack correctly", func() {
			original := color.Color{R: 100, G: 200, B: 50, A: 0.75}
			data, err := msgpack.Marshal(original)
			Expect(err).ToNot(HaveOccurred())
			var decoded color.Color
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
		It("Should decode an empty string as zero color", func() {
			encoded, err := msgpack.Marshal("#000000")
			Expect(err).ToNot(HaveOccurred())
			var c color.Color
			Expect(msgpack.Unmarshal(encoded, &c)).To(Succeed())
			Expect(c).To(Equal(color.Color{R: 0, G: 0, B: 0, A: 1}))
		})
	})
})
