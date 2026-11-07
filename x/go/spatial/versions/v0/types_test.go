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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/spatial/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Viewport", func() {
	DescribeTable("Should decode over the schema defaults",
		func(in map[string]any, expected v0.Viewport) {
			var fromJSON, fromMsgpack v0.Viewport
			Expect(json.Unmarshal(MustSucceed(json.Marshal(in)), &fromJSON)).
				To(Succeed())
			Expect(fromJSON).To(Equal(expected))
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(in)), &fromMsgpack)).
				To(Succeed())
			Expect(fromMsgpack).To(Equal(expected))
		},
		Entry("absent zoom takes its default", map[string]any{}, v0.Viewport{Zoom: 1}),
		Entry(
			"explicit zero zoom stays",
			map[string]any{"zoom": 0},
			v0.Viewport{Zoom: 0},
		),
		Entry(
			"explicit zoom stays",
			map[string]any{"zoom": 2, "position": map[string]any{"x": 3}},
			v0.Viewport{Zoom: 2, Position: v0.XY{X: 3}},
		),
	)

	It("Should merge the input into a value that already holds data", func() {
		vp := v0.Viewport{Zoom: 3, Position: v0.XY{X: 4}}
		Expect(json.Unmarshal([]byte(`{"position":{"y":5}}`), &vp)).To(Succeed())
		Expect(vp).To(Equal(v0.Viewport{Zoom: 3, Position: v0.XY{X: 4, Y: 5}}))
	})
})

var _ = Describe("StickyXY", func() {
	It("Should keep a supplied struct member and default an absent one", func() {
		var s v0.StickyXY
		Expect(json.Unmarshal(
			[]byte(`{"x":5,"root":{"x":"right","y":"bottom"}}`),
			&s,
		)).To(Succeed())
		Expect(s).To(Equal(v0.StickyXY{
			X:     5,
			Root:  v0.CornerLocation{X: v0.XLocationRight, Y: v0.YLocationBottom},
			Units: v0.StickyUnits{X: v0.StickyUnitPx, Y: v0.StickyUnitPx},
		}))
	})
})
