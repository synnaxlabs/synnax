// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package toml_test

import (
	"bytes"
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xtoml "github.com/synnaxlabs/x/encoding/toml"
	. "github.com/synnaxlabs/x/testutil"
)

// tagged carries explicit toml tags. The codec reads these directly (go-toml would
// otherwise emit the verbatim "ShowChannelNames").
type tagged struct {
	ShowChannelNames bool `toml:"show_channel_names"`
	Precision        int  `toml:"precision"`
}

// flatEnvelope mimics imex.Envelope: it implements the codec's Marshaler/Unmarshaler
// interfaces to flatten the promoted fields alongside Data, since go-toml exposes no
// usable hook of its own.
type flatEnvelope struct {
	Version int
	Name    string
	Data    map[string]any
}

func (e flatEnvelope) TOMLValue() (any, error) {
	m := make(map[string]any, len(e.Data)+2)
	maps.Copy(m, e.Data)
	m["version"] = e.Version
	m["name"] = e.Name
	return m, nil
}

func (e *flatEnvelope) FromTOMLValue(m map[string]any) error {
	if v, ok := m["version"].(int64); ok {
		e.Version = int(v)
		delete(m, "version")
	}
	if v, ok := m["name"].(string); ok {
		e.Name = v
		delete(m, "name")
	}
	e.Data = m
	return nil
}

var _ = Describe("Codec", func() {
	It("Should have the correct content type", func() {
		Expect(xtoml.Codec.ContentType()).To(Equal("application/toml"))
	})
	It("Should round-trip honoring toml tags", func(ctx SpecContext) {
		in := tagged{ShowChannelNames: true, Precision: 4}
		b := MustSucceed(xtoml.Codec.Encode(ctx, in))
		Expect(string(b)).To(ContainSubstring("show_channel_names = true"))
		var out tagged
		Expect(xtoml.Codec.Decode(ctx, b, &out)).To(Succeed())
		Expect(out).To(Equal(in))
		var streamed tagged
		Expect(xtoml.Codec.DecodeStream(ctx, bytes.NewReader(b), &streamed)).To(Succeed())
		Expect(streamed).To(Equal(in))
	})
	It("Should emit integers as integers, not floats", func(ctx SpecContext) {
		b := MustSucceed(xtoml.Codec.Encode(ctx, tagged{Precision: 2}))
		Expect(string(b)).To(ContainSubstring("precision = 2"))
		Expect(string(b)).ToNot(ContainSubstring("precision = 2.0"))
	})
	It("Should honor the Marshaler interface that flattens fields", func(ctx SpecContext) {
		env := flatEnvelope{
			Version: 1,
			Name:    "Temperature Log",
			Data:    map[string]any{"remote_created": false},
		}
		b := MustSucceed(xtoml.Codec.Encode(ctx, env))
		var flat map[string]any
		Expect(xtoml.Codec.Decode(ctx, b, &flat)).To(Succeed())
		Expect(flat).To(HaveKeyWithValue("version", int64(1)))
		Expect(flat).To(HaveKeyWithValue("name", "Temperature Log"))
		Expect(flat).To(HaveKey("remote_created"))
		Expect(flat).ToNot(HaveKey("Data"))
		Expect(flat).ToNot(HaveKey("data"))
	})
	It("Should honor the Unmarshaler interface on decode", func(ctx SpecContext) {
		in := []byte("version = 2\nname = \"foo\"\nchannels = [1, 2]\n")
		var env flatEnvelope
		Expect(xtoml.Codec.Decode(ctx, in, &env)).To(Succeed())
		Expect(env.Version).To(Equal(2))
		Expect(env.Name).To(Equal("foo"))
		Expect(env.Data).To(HaveKey("channels"))
	})
	It("Should add error info on decoding failure", func(ctx SpecContext) {
		var out tagged
		Expect(xtoml.Codec.Decode(ctx, []byte("= = ="), &out)).To(
			MatchError(ContainSubstring("failed to decode")),
		)
	})
})
