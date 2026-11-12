// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/text"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Axes", func() {
	DescribeTable("Should keep the field defaults of a partial nested axis",
		func(decode func(in map[string]any, out *v5.Axes) error) {
			var axes v5.Axes
			Expect(decode(
				map[string]any{"y1": map[string]any{"label": "Pressure"}},
				&axes,
			)).To(Succeed())
			Expect(axes.Y1).To(Equal(v5.Axis{
				Key:            v5.AxisKeyY1,
				Label:          "Pressure",
				LabelDirection: spatial.DirectionY,
				LabelLevel:     text.LevelSmall,
				TickSpacing:    75,
			}))
			Expect(axes.X1.Key).To(Equal(v5.AxisKeyX1))
		},
		Entry("JSON", func(in map[string]any, out *v5.Axes) error {
			GinkgoHelper()
			return json.Unmarshal(MustSucceed(json.Marshal(in)), out)
		}),
		Entry("msgpack", func(in map[string]any, out *v5.Axes) error {
			GinkgoHelper()
			return msgpack.Unmarshal(MustSucceed(msgpack.Marshal(in)), out)
		}),
	)

	It("Should keep an explicit zero tick spacing", func() {
		var axes v5.Axes
		Expect(json.Unmarshal([]byte(`{"x1":{"tick_spacing":0}}`), &axes)).
			To(Succeed())
		Expect(axes.X1.TickSpacing).To(BeZero())
		Expect(axes.X2.TickSpacing).To(Equal(75.0))
	})
})
