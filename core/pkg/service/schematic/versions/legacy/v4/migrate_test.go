// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v4"
)

// createV3 builds a v3.Data with every field populated to a non-zero value so
// passthrough regressions surface.
func createV3() v3.Data {
	srcH, tgtH := "out", "in"
	return v3.Data{
		Version:         v3.Version,
		Editable:        true,
		FitViewOnResize: true,
		Snapshot:        true,
		RemoteCreated:   true,
		Viewport:        v0.Viewport{Position: v0.XY{X: 12, Y: 34}, Zoom: 1.5},
		Nodes: []v0.Node{
			{Key: "n1", Position: v0.XY{X: 1, Y: 2}},
			{Key: "n2", Position: v0.XY{X: 3, Y: 4}},
		},
		Edges: []v3.Edge{{
			Key:          "e1",
			Source:       "n1",
			Target:       "n2",
			SourceHandle: &srcH,
			TargetHandle: &tgtH,
			Segments:     []v3.Segment{{Direction: "x", Length: 10}},
			Data:         json.RawMessage(`{"color":"#ff0000"}`),
		}},
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
	It("Should set authority to 1", func() {
		out := v4.Migrate(createV3())
		Expect(out.Version).To(Equal(v4.Version))
		Expect(out.Authority).To(Equal(1.0))
	})

	It("Should pass every v3 field through unchanged", func() {
		in := createV3()
		out := v4.Migrate(in)
		Expect(out.Edges).To(Equal(in.Edges))
		Expect(out.Nodes).To(Equal(in.Nodes))
		Expect(out.Props).To(Equal(in.Props))
		Expect(out.Legend).To(Equal(in.Legend))
		Expect(out.Key).To(Equal(in.Key))
		Expect(out.Type).To(Equal(in.Type))
		Expect(out.ViewportMode).To(Equal(in.ViewportMode))
	})
})
