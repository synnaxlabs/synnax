// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy"
	v4 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v4"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

func jsonMap(raw string) msgpack.EncodedJSON {
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

var _ = Describe("MigrateData", func() {
	Describe("version dispatch", func() {
		It("Should walk a v0 blob through every step to v4.Data", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "0.0.0",
				"axes": {"renderTrigger": 0, "hasHadChannelSet": false, "axes": {
					"x1": {"key": "x1", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"x2": {"key": "x2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y1": {"key": "y1", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y2": {"key": "y2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y3": {"key": "y3", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y4": {"key": "y4", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75}
				}}
			}`)))
			Expect(out.Version).To(Equal(v4.Version))
			Expect(out.Legend.Visible).To(BeTrue())
			Expect(out.Legend.Position.X).To(Equal(50.0))
			Expect(out.Axes.Axes.X1.Type).To(Equal("time"))
			Expect(out.Axes.Axes.Y1.LabelDirection).To(Equal("y"))
		})

		It("Should fall back to v0 when the blob has no version field", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{"title": {"level": "h4", "visible": false}}`)))
			Expect(out.Version).To(Equal(v4.Version))
			Expect(out.Title.Level).To(Equal("h4"))
		})

		It("Should walk a nil blob to a zero v4.Data", func() {
			out := MustSucceed(legacy.MigrateData(nil))
			Expect(out.Version).To(Equal(v4.Version))
		})

		It("Should error on an unknown declared version", func() {
			Expect(legacy.MigrateData(jsonMap(`{"version": "99.0.0"}`))).Error().
				To(MatchError(ContainSubstring("unknown line plot data version")))
		})

		It("Should produce a v4-shaped Data when given a v2 input", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "2.0.0",
				"title": {"level": "h3", "visible": true},
				"legend": {"visible": false, "position": {"x": 1, "y": 2}},
				"axes": {"renderTrigger": 0, "hasHadChannelSet": false, "axes": {
					"x1": {"key": "x1", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75, "type": "time"},
					"x2": {"key": "x2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75, "type": "time"},
					"y1": {"key": "y1", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y2": {"key": "y2", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y3": {"key": "y3", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75,"type": null},
					"y4": {"key": "y4", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75}
				}}
			}`)))
			Expect(out.Version).To(Equal(v4.Version))
			Expect(out.Title.Level).To(Equal("h3"))
			Expect(out.Legend.Visible).To(BeFalse())
			Expect(out.Legend.Position.X).To(Equal(1.0))
		})
	})
})
