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
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

type camelNode struct {
	Key    string `json:"key"`
	ZIndex int    `json:"z_index"`
}

type camelPayload struct {
	Name         string                         `json:"name"`
	StrokeWidth  float64                        `json:"stroke_width"`
	Nodes        []camelNode                    `json:"nodes"`
	Configs      map[string]msgpack.EncodedJSON `json:"configs"`
	ManualBounds map[string]camelNode           `json:"manual_bounds"`
}

func camelEnvelope(src string) imex.Envelope {
	var env imex.Envelope
	Expect(json.Unmarshal([]byte(src), &env)).To(Succeed())
	return env
}

var _ = Describe("DecodeCamel", func() {
	It("Should decode camelCase keys into snake_case tagged fields", func(ctx SpecContext) {
		env := camelEnvelope(`{
			"type": "schematic",
			"name": "n",
			"strokeWidth": 2.5,
			"nodes": [{"key": "a", "zIndex": 3}]
		}`)
		p := MustSucceed(imex.DecodeCamel[camelPayload](ctx, env))
		Expect(p.StrokeWidth).To(Equal(2.5))
		Expect(p.Nodes).To(HaveLen(1))
		Expect(p.Nodes[0].ZIndex).To(Equal(3))
	})

	It("Should decode snake_case keys unchanged", func(ctx SpecContext) {
		env := camelEnvelope(`{
			"type": "schematic",
			"name": "n",
			"stroke_width": 1.5,
			"nodes": [{"key": "a", "z_index": 7}]
		}`)
		p := MustSucceed(imex.DecodeCamel[camelPayload](ctx, env))
		Expect(p.StrokeWidth).To(Equal(1.5))
		Expect(p.Nodes[0].ZIndex).To(Equal(7))
	})

	It("Should leave map keys and opaque payloads untouched", func(ctx SpecContext) {
		env := camelEnvelope(`{
			"type": "schematic",
			"name": "n",
			"configs": {"myNode": {"strokeWidth": 4, "innerProps": {"fooBar": 1}}},
			"manualBounds": {"someKey": {"key": "b", "zIndex": 9}}
		}`)
		p := MustSucceed(imex.DecodeCamel[camelPayload](ctx, env))
		// The Configs map key is a user-chosen identifier and its EncodedJSON
		// value owns its wire form: both keep their original camelCase.
		Expect(p.Configs).To(HaveKey("myNode"))
		var cfg map[string]any
		Expect(p.Configs["myNode"].Unmarshal(&cfg)).To(Succeed())
		Expect(cfg).To(HaveKey("strokeWidth"))
		Expect(cfg["innerProps"]).To(HaveKeyWithValue("fooBar", 1.0))
		// Map values of struct type still normalize their own field keys.
		Expect(p.ManualBounds).To(HaveKey("someKey"))
		Expect(p.ManualBounds["someKey"].ZIndex).To(Equal(9))
	})

	It("Should error when no codec is bound", func() {
		Expect(imex.DecodeCamel[camelPayload](
			GinkgoT().Context(), imex.Envelope{},
		)).Error().To(MatchError(ContainSubstring("no codec bound")))
	})
})
