// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v3"
)

// createV2 builds a v2.Data with every field populated to a non-zero value so
// passthrough regressions surface.
func createV2() v2.Data {
	srcH, tgtH := "out", "in"
	return v2.Data{
		Version:         v2.Version,
		Editable:        true,
		FitViewOnResize: true,
		Snapshot:        true,
		RemoteCreated:   true,
		Viewport:        v0.Viewport{Position: v0.XY{X: 12, Y: 34}, Zoom: 1.5},
		Nodes: []v0.Node{
			{Key: "n1", Position: v0.XY{X: 1, Y: 2}},
			{Key: "n2", Position: v0.XY{X: 3, Y: 4}},
		},
		Edges: []v0.Edge{
			{
				Key:          "e1",
				Source:       "n1",
				Target:       "n2",
				SourceHandle: &srcH,
				TargetHandle: &tgtH,
				Data: json.RawMessage(
					`{"segments":[{"direction":"x","length":10}]}`,
				),
			},
		},
		Props: map[string]json.RawMessage{
			"n1": json.RawMessage(`{"key":"valve"}`),
		},
		Control:      "released",
		Legend:       v1.ZeroLegend,
		Key:          "3e8f9a52-2c1d-4a6b-9f27-c05561f7f2a4",
		Type:         "schematic",
		ViewportMode: "select",
	}
}

var _ = Describe("Migrate", func() {
	It("Should attach an empty segments slice to every edge", func() {
		out := v3.Migrate(createV2())
		Expect(out.Version).To(Equal(v3.Version))
		for _, e := range out.Edges {
			Expect(e.Segments).NotTo(BeNil())
			Expect(e.Segments).To(BeEmpty())
		}
	})

	It("Should preserve edge.Data so v6 can lift segments/color/variant", func() {
		in := createV2()
		out := v3.Migrate(in)
		Expect(out.Edges).To(HaveLen(len(in.Edges)))
		for i, e := range out.Edges {
			Expect(e.Data).To(Equal(in.Edges[i].Data))
		}
	})

	It("Should pass non-edge fields through unchanged", func() {
		in := createV2()
		out := v3.Migrate(in)
		Expect(out.Nodes).To(Equal(in.Nodes))
		Expect(out.Props).To(Equal(in.Props))
		Expect(out.Key).To(Equal(in.Key))
		Expect(out.Type).To(Equal(in.Type))
		Expect(out.ViewportMode).To(Equal(in.ViewportMode))
		Expect(out.Legend).To(Equal(in.Legend))
	})
})
