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
	"github.com/synnaxlabs/x/color"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Color", func() {
	Describe("FromHex", func() {
		It("parses 6-char hex without #", func() {
			c, err := color.FromHex("ff0000")
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 255, G: 0, B: 0, A: 1.0}))
		})

		It("parses 6-char hex with #", func() {
			c, err := color.FromHex("#00ff00")
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 0, G: 255, B: 0, A: 1.0}))
		})

		It("parses 8-char hex with alpha", func() {
			c, err := color.FromHex("#0000ff80")
			Expect(err).ToNot(HaveOccurred())
			Expect(c.R).To(Equal(uint8(0)))
			Expect(c.G).To(Equal(uint8(0)))
			Expect(c.B).To(Equal(uint8(255)))
			Expect(c.A).To(BeNumerically("~", 0.502, 0.01))
		})

		It("rejects invalid length", func() {
			_, err := color.FromHex("fff")
			Expect(err).To(HaveOccurred())
		})

		It("rejects invalid characters", func() {
			_, err := color.FromHex("gggggg")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Hex", func() {
		It("outputs 6-char when alpha is 1.0", func() {
			c := color.Color{R: 255, G: 128, B: 64, A: 1.0}
			Expect(c.Hex()).To(Equal("#ff8040"))
		})

		It("outputs 8-char when alpha is not 1.0", func() {
			c := color.Color{R: 255, G: 0, B: 0, A: 0.5}
			Expect(c.Hex()).To(HavePrefix("#ff0000"))
			Expect(c.Hex()).To(HaveLen(9))
		})
	})

	Describe("IsZero", func() {
		It("returns true for zero color", func() {
			c := color.Color{R: 0, G: 0, B: 0, A: 0}
			Expect(c.IsZero()).To(BeTrue())
		})

		It("returns false when any component is non-zero", func() {
			Expect(color.Color{R: 1, G: 0, B: 0, A: 0}.IsZero()).To(BeFalse())
			Expect(color.Color{R: 0, G: 0, B: 0, A: 1}.IsZero()).To(BeFalse())
		})
	})

	Describe("JSON", func() {
		It("unmarshals struct format", func() {
			var c color.Color
			err := json.Unmarshal([]byte(`{"r":255,"g":128,"b":64,"a":0.5}`), &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 255, G: 128, B: 64, A: 0.5}))
		})

		It("unmarshals zero color struct", func() {
			var c color.Color
			err := json.Unmarshal([]byte(`{"r":0,"g":0,"b":0,"a":0}`), &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 0, G: 0, B: 0, A: 0}))
		})

		It("unmarshals array format", func() {
			var c color.Color
			err := json.Unmarshal([]byte(`[255,0,0,0.5]`), &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 255, G: 0, B: 0, A: 0.5}))
		})

		It("unmarshals hex string", func() {
			var c color.Color
			err := json.Unmarshal([]byte(`"#ff0000"`), &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 255, G: 0, B: 0, A: 1.0}))
		})

		It("unmarshals null", func() {
			var c color.Color
			err := json.Unmarshal([]byte(`null`), &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{}))
		})

		It("rejects invalid array length", func() {
			var c color.Color
			err := json.Unmarshal([]byte(`[255,0,0]`), &c)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Msgpack", func() {
		It("decodes struct format", func() {
			data, _ := msgpack.Marshal(map[string]any{"r": 255, "g": 128, "b": 64, "a": 0.5})
			var c color.Color
			err := msgpack.Unmarshal(data, &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 255, G: 128, B: 64, A: 0.5}))
		})

		It("decodes zero color struct", func() {
			data, _ := msgpack.Marshal(map[string]any{"r": 0, "g": 0, "b": 0, "a": 0.0})
			var c color.Color
			err := msgpack.Unmarshal(data, &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 0, G: 0, B: 0, A: 0}))
		})

		It("decodes array format", func() {
			data, _ := msgpack.Marshal([]any{255, 0, 0, 0.5})
			var c color.Color
			err := msgpack.Unmarshal(data, &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 255, G: 0, B: 0, A: 0.5}))
		})

		It("decodes hex string", func() {
			data, _ := msgpack.Marshal("#ff0000")
			var c color.Color
			err := msgpack.Unmarshal(data, &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{R: 255, G: 0, B: 0, A: 1.0}))
		})

		It("decodes nil", func() {
			data, _ := msgpack.Marshal(nil)
			var c color.Color
			err := msgpack.Unmarshal(data, &c)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).To(Equal(color.Color{}))
		})

		It("rejects invalid array length", func() {
			data, _ := msgpack.Marshal([]any{255, 0, 0})
			var c color.Color
			err := msgpack.Unmarshal(data, &c)
			Expect(err).To(HaveOccurred())
		})
	})
})
