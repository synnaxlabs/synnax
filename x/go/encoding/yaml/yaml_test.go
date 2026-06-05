// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package yaml_test

import (
	"bytes"
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xyaml "github.com/synnaxlabs/x/encoding/yaml"
	. "github.com/synnaxlabs/x/testutil"
	"gopkg.in/yaml.v3"
)

// tagged carries explicit yaml tags. The codec reads these directly (yaml.v3 would
// otherwise lowercase the field name to "showchannelnames").
type tagged struct {
	ShowChannelNames bool `yaml:"show_channel_names"`
	Precision        int  `yaml:"precision"`
}

// flatEnvelope mimics imex.Envelope: its MarshalYAML/UnmarshalYAML flatten the promoted
// fields alongside Data. It proves the codec honors the standard yaml.Marshaler /
// yaml.Unmarshaler interfaces.
type flatEnvelope struct {
	Version int
	Name    string
	Data    map[string]any
}

func (e flatEnvelope) MarshalYAML() (any, error) {
	m := make(map[string]any, len(e.Data)+2)
	maps.Copy(m, e.Data)
	m["version"] = e.Version
	m["name"] = e.Name
	return m, nil
}

func (e *flatEnvelope) UnmarshalYAML(node *yaml.Node) error {
	var m map[string]any
	if err := node.Decode(&m); err != nil {
		return err
	}
	if v, ok := m["version"].(int); ok {
		e.Version = v
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
		Expect(xyaml.Codec.ContentType()).To(Equal("application/yaml"))
	})
	It("Should round-trip honoring yaml tags", func(ctx SpecContext) {
		in := tagged{ShowChannelNames: true, Precision: 4}
		b := MustSucceed(xyaml.Codec.Encode(ctx, in))
		Expect(string(b)).To(ContainSubstring("show_channel_names: true"))
		var out tagged
		Expect(xyaml.Codec.Decode(ctx, b, &out)).To(Succeed())
		Expect(out).To(Equal(in))
		var streamed tagged
		Expect(xyaml.Codec.DecodeStream(ctx, bytes.NewReader(b), &streamed)).To(Succeed())
		Expect(streamed).To(Equal(in))
	})
	It("Should emit integers as integers, not floats or strings", func(ctx SpecContext) {
		b := MustSucceed(xyaml.Codec.Encode(ctx, tagged{Precision: 2}))
		Expect(string(b)).To(ContainSubstring("precision: 2"))
		Expect(string(b)).ToNot(ContainSubstring("precision: 2.0"))
		Expect(string(b)).ToNot(ContainSubstring(`precision: "2"`))
	})
	It("Should honor yaml.Marshaler that flattens fields", func(ctx SpecContext) {
		env := flatEnvelope{
			Version: 1,
			Name:    "Temperature Log",
			Data:    map[string]any{"channels": []any{"a", "b"}},
		}
		b := MustSucceed(xyaml.Codec.Encode(ctx, env))
		var flat map[string]any
		Expect(xyaml.Codec.Decode(ctx, b, &flat)).To(Succeed())
		Expect(flat).To(HaveKeyWithValue("version", 1))
		Expect(flat).To(HaveKeyWithValue("name", "Temperature Log"))
		Expect(flat).To(HaveKey("channels"))
		Expect(flat).ToNot(HaveKey("Data"))
		Expect(flat).ToNot(HaveKey("data"))
	})
	It("Should honor yaml.Unmarshaler on decode", func(ctx SpecContext) {
		var env flatEnvelope
		in := []byte("version: 2\nname: foo\nchannels: [1, 2]\n")
		Expect(xyaml.Codec.Decode(ctx, in, &env)).To(Succeed())
		Expect(env.Version).To(Equal(2))
		Expect(env.Name).To(Equal("foo"))
		Expect(env.Data).To(HaveKey("channels"))
	})
	It("Should add error info on decoding failure", func(ctx SpecContext) {
		var out tagged
		Expect(xyaml.Codec.Decode(ctx, []byte("\tnot: [valid"), &out)).To(
			MatchError(ContainSubstring("failed to decode")),
		)
	})
})
