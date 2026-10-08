// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v9_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v9 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v9"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("DecodeWithDefaults", func() {
	It("Should fill the schema defaults of every omitted field", func() {
		cfg := MustSucceed(v9.DecodeWithDefaults(msgpack.EncodedJSON{
			"variant": "switch",
		})).Variant.(v9.SwitchElementConfig)
		Expect(cfg.Scale).To(BeEquivalentTo(1))
		Expect(cfg.StalenessTimeout).To(BeEquivalentTo(5))
		Expect(cfg.Label.Level).To(BeEquivalentTo("h5"))
	})

	It("Should keep a carried value, zero included", func() {
		cfg := MustSucceed(v9.DecodeWithDefaults(msgpack.EncodedJSON{
			"variant": "line",
			"end":     map[string]any{"x": 0.0, "y": 40.0},
		})).Variant.(v9.LineElementConfig)
		Expect(cfg.End).To(Equal(spatial.XY{X: 0, Y: 40}))
		Expect(cfg.StrokeWidth).To(BeEquivalentTo(2))
	})

	It("Should keep a carried empty string", func() {
		cfg := MustSucceed(v9.DecodeWithDefaults(msgpack.EncodedJSON{
			"variant": "value",
			"units":   "",
		})).Variant.(v9.ValueElementConfig)
		Expect(cfg.Units).To(Equal(""))
	})

	It("Should fill the control state defaults when the config carries one", func() {
		cfg := MustSucceed(v9.DecodeWithDefaults(msgpack.EncodedJSON{
			"variant": "switch",
			"control": map[string]any{"authority": 10.0},
		})).Variant.(v9.SwitchElementConfig)
		Expect(cfg.Control).ToNot(BeNil())
		Expect(cfg.Control.Authority).To(HaveValue(BeEquivalentTo(10)))
		Expect(cfg.Control.Orientation).To(BeEquivalentTo("bottom"))
	})

	DescribeTable("Should reject a config that names no known variant",
		func(stored msgpack.EncodedJSON) {
			Expect(v9.DecodeWithDefaults(stored)).Error().To(
				MatchError(validate.ErrValidation),
			)
		},
		Entry("missing", msgpack.EncodedJSON{"scale": 2.0}),
		Entry("unknown", msgpack.EncodedJSON{"variant": "not-a-symbol"}),
	)
})
