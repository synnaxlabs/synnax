// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	xjson "github.com/synnaxlabs/x/encoding/json"
	xtoml "github.com/synnaxlabs/x/encoding/toml"
	xyaml "github.com/synnaxlabs/x/encoding/yaml"
	xhttp "github.com/synnaxlabs/x/http"
	. "github.com/synnaxlabs/x/testutil"
)

// The portable import/export endpoints negotiate JSON, YAML, and TOML for the envelope
// body. Envelope's flat wire shape is implemented only in its MarshalJSON/UnmarshalJSON,
// so these specs assert the YAML and TOML codecs reproduce that exact shape via the JSON
// bridge rather than emitting a nested struct layout.
var _ = Describe("Portable Codecs", func() {
	newEnvelope := func() imex.Envelope {
		return imex.Envelope{
			Version: 1,
			Type:    "log",
			Name:    "Temperature Log",
			Data: map[string]any{
				"channels": []any{
					map[string]any{"channel": 1, "color": "red", "precision": 2},
				},
				"remote_created":      false,
				"timestamp_precision": 1,
			},
		}
	}

	DescribeTable("Should serialize the envelope as a flat document",
		func(ctx SpecContext, codec xhttp.Codec) {
			b := MustSucceed(codec.Encode(ctx, newEnvelope()))
			var flat map[string]any
			Expect(codec.Decode(ctx, b, &flat)).To(Succeed())
			Expect(flat).To(HaveKeyWithValue("type", "log"))
			Expect(flat).To(HaveKeyWithValue("name", "Temperature Log"))
			Expect(flat).To(HaveKey("version"))
			Expect(flat).To(HaveKey("channels"))
			Expect(flat).To(HaveKey("remote_created"))
			Expect(flat).ToNot(HaveKey("data"))
			Expect(flat).ToNot(HaveKey("Data"))
		},
		Entry("YAML", xyaml.Codec),
		Entry("TOML", xtoml.Codec),
	)

	DescribeTable("Should round-trip the envelope into its promoted fields and data",
		func(ctx SpecContext, codec xhttp.Codec) {
			env := newEnvelope()
			b := MustSucceed(codec.Encode(ctx, env))
			var decoded imex.Envelope
			Expect(codec.Decode(ctx, b, &decoded)).To(Succeed())
			Expect(decoded.Version).To(Equal(imex.Version(1)))
			Expect(decoded.Type).To(Equal("log"))
			Expect(decoded.Name).To(Equal("Temperature Log"))
			expected := MustSucceed(json.Marshal(env.Data))
			actual := MustSucceed(json.Marshal(decoded.Data))
			Expect(actual).To(MatchJSON(expected))
		},
		Entry("YAML", xyaml.Codec),
		Entry("TOML", xtoml.Codec),
	)

	It("Should agree with the JSON codec on the decoded envelope", func(ctx SpecContext) {
		env := newEnvelope()
		var fromJSON, fromYAML, fromTOML imex.Envelope
		Expect(xjson.Codec.Decode(ctx, MustSucceed(xjson.Codec.Encode(ctx, env)), &fromJSON)).To(Succeed())
		Expect(xyaml.Codec.Decode(ctx, MustSucceed(xyaml.Codec.Encode(ctx, env)), &fromYAML)).To(Succeed())
		Expect(xtoml.Codec.Decode(ctx, MustSucceed(xtoml.Codec.Encode(ctx, env)), &fromTOML)).To(Succeed())
		// The promoted fields are typed identically across codecs; Data values differ
		// only in their concrete numeric Go type (json.Number vs int vs int64), so they
		// are compared after normalizing back through JSON.
		jsonData := MustSucceed(json.Marshal(fromJSON.Data))
		for _, other := range []imex.Envelope{fromYAML, fromTOML} {
			Expect(other.Version).To(Equal(fromJSON.Version))
			Expect(other.Type).To(Equal(fromJSON.Type))
			Expect(other.Name).To(Equal(fromJSON.Name))
			Expect(MustSucceed(json.Marshal(other.Data))).To(MatchJSON(jsonData))
		}
	})
})
