// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package border_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/border"
	"github.com/synnaxlabs/x/spatial"
)

var _ = Describe("Radius", func() {
	DescribeTable("UnmarshalJSON accepts crude forms",
		func(input string, expected border.Radius) {
			var r border.Radius
			Expect(json.Unmarshal([]byte(input), &r)).To(Succeed())
			Expect(r).To(Equal(expected))
		},
		Entry("bare number", `4`, border.Radius{
			TopLeft:     spatial.XY{X: 4, Y: 4},
			TopRight:    spatial.XY{X: 4, Y: 4},
			BottomLeft:  spatial.XY{X: 4, Y: 4},
			BottomRight: spatial.XY{X: 4, Y: 4},
		}),
		Entry("direction pair", `{"x": 50, "y": 10}`, border.Radius{
			TopLeft:     spatial.XY{X: 50, Y: 10},
			TopRight:    spatial.XY{X: 50, Y: 10},
			BottomLeft:  spatial.XY{X: 50, Y: 10},
			BottomRight: spatial.XY{X: 50, Y: 10},
		}),
		Entry("per-corner numbers, camelCase", `{"topLeft": 1, "topRight": 2, "bottomLeft": 3, "bottomRight": 4}`, border.Radius{
			TopLeft:     spatial.XY{X: 1, Y: 1},
			TopRight:    spatial.XY{X: 2, Y: 2},
			BottomLeft:  spatial.XY{X: 3, Y: 3},
			BottomRight: spatial.XY{X: 4, Y: 4},
		}),
		Entry("canonical per-corner pairs, snake_case", `{"top_left": {"x": 1, "y": 9}, "top_right": {"x": 2, "y": 8}, "bottom_left": {"x": 3, "y": 7}, "bottom_right": {"x": 4, "y": 6}}`, border.Radius{
			TopLeft:     spatial.XY{X: 1, Y: 9},
			TopRight:    spatial.XY{X: 2, Y: 8},
			BottomLeft:  spatial.XY{X: 3, Y: 7},
			BottomRight: spatial.XY{X: 4, Y: 6},
		}),
		Entry("null", `null`, border.Radius{}),
	)

	It("Should round-trip the canonical form through MarshalJSON", func() {
		in := border.Radius{
			TopLeft:     spatial.XY{X: 1, Y: 9},
			TopRight:    spatial.XY{X: 2, Y: 8},
			BottomLeft:  spatial.XY{X: 3, Y: 7},
			BottomRight: spatial.XY{X: 4, Y: 6},
		}
		data, err := json.Marshal(in)
		Expect(err).To(BeNil())
		var out border.Radius
		Expect(json.Unmarshal(data, &out)).To(Succeed())
		Expect(out).To(Equal(in))
	})
})
