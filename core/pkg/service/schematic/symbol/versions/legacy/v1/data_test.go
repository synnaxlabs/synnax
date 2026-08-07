// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/legacy/v1"
	"github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Data", func() {
	It("Should stamp the version the Console wrote", func() {
		Expect(v1.Version).To(BeEquivalentTo(1))
	})

	It("Should decode the Console's camelCase spec keys", func() {
		var d v1.Data
		Expect(json.Unmarshal([]byte(`{
			"key": "11111111-2222-3333-4444-555555555555",
			"version": 1,
			"name": "Valve",
			"data": {
				"svg": "<svg/>",
				"variant": "valve",
				"scale": 1.5,
				"scaleStroke": true,
				"previewViewport": {
					"position": {"x": 1, "y": 2},
					"dimensions": {"width": 3, "height": 4}
				},
				"handles": [{
					"key": "h1",
					"position": {"x": 5, "y": 6},
					"orientation": "top"
				}],
				"states": [{
					"key": "s1",
					"name": "Open",
					"regions": [{
						"key": "r1",
						"name": "Body",
						"selectors": ["#body"],
						"strokeColor": "#ff0000",
						"fillColor": "#00ff00"
					}]
				}]
			}
		}`), &d)).To(Succeed())
		Expect(d.Spec.SVG).To(Equal("<svg/>"))
		Expect(d.Spec.Variant).To(Equal("valve"))
		Expect(d.Spec.Scale).To(Equal(1.5))
		Expect(d.Spec.ScaleStroke).To(BeTrue())
		Expect(d.Spec.PreviewViewport).ToNot(BeNil())
		Expect(d.Spec.Handles).To(HaveLen(1))
		Expect(d.Spec.Handles[0].Orientation).To(BeEquivalentTo("top"))
		Expect(d.Spec.States).To(HaveLen(1))
		Expect(d.Spec.States[0].Regions[0].Selectors).To(Equal([]string{"#body"}))
	})

	It("Should parse the hex region colors the Console persisted", func() {
		var d v1.Data
		Expect(json.Unmarshal([]byte(`{"data": {"states": [{"regions": [
			{"strokeColor": "#ff0000", "fillColor": "#0000ff"}
		]}]}}`), &d)).To(Succeed())
		r := d.Spec.States[0].Regions[0]
		Expect(*r.StrokeColor).To(Equal(MustSucceed(color.FromHex("#ff0000"))))
		Expect(*r.FillColor).To(Equal(MustSucceed(color.FromHex("#0000ff"))))
	})

	It("Should leave region colors nil when the Console omitted them", func() {
		var d v1.Data
		Expect(json.Unmarshal(
			[]byte(`{"data": {"states": [{"regions": [{"key": "r1"}]}]}}`), &d,
		)).To(Succeed())
		r := d.Spec.States[0].Regions[0]
		Expect(r.StrokeColor).To(BeNil())
		Expect(r.FillColor).To(BeNil())
	})

	It("Should leave the spec zero when the key is snake_case", func() {
		var d v1.Data
		Expect(json.Unmarshal(
			[]byte(`{"data": {"scale_stroke": true, "preview_viewport": {}}}`), &d,
		)).To(Succeed())
		Expect(d.Spec.ScaleStroke).To(BeFalse())
		Expect(d.Spec.PreviewViewport).To(BeNil())
	})
})
