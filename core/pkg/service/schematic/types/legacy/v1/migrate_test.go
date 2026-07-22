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
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/types/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/types/legacy/v1"
)

// createV0 builds a v0.Data with every field populated to a non-zero value so
// passthrough regressions surface.
func createV0() v0.Data {
	srcH, tgtH := "out", "in"
	return v0.Data{
		Version:         v0.Version,
		Editable:        true,
		FitViewOnResize: true,
		Snapshot:        true,
		RemoteCreated:   true,
		Viewport:        v0.Viewport{Position: v0.XY{X: 12, Y: 34}, Zoom: 1.5},
		Nodes: []v0.Node{
			{Key: "n1", Position: v0.XY{X: 1, Y: 2}},
			{Key: "n2", Position: v0.XY{X: 3, Y: 4}},
		},
		Edges: []v0.Edge{{
			Key:          "e1",
			Source:       "n1",
			Target:       "n2",
			SourceHandle: &srcH,
			TargetHandle: &tgtH,
			Data:         json.RawMessage(`{"segments":[{"direction":"x","length":10}]}`),
		}},
		Props:   map[string]json.RawMessage{"n1": json.RawMessage(`{"key":"valve"}`)},
		Control: "released",
	}
}

var _ = Describe("Migrate", func() {
	It("Should attach the default legend", func() {
		out := v1.Migrate(createV0())
		Expect(out.Version).To(Equal(v1.Version))
		Expect(out.Legend).To(Equal(v1.ZeroLegend))
	})

	It("Should pass every v0 field through unchanged", func() {
		in := createV0()
		out := v1.Migrate(in)
		Expect(out.Editable).To(Equal(in.Editable))
		Expect(out.FitViewOnResize).To(Equal(in.FitViewOnResize))
		Expect(out.Snapshot).To(Equal(in.Snapshot))
		Expect(out.RemoteCreated).To(Equal(in.RemoteCreated))
		Expect(out.Viewport).To(Equal(in.Viewport))
		Expect(out.Nodes).To(Equal(in.Nodes))
		Expect(out.Edges).To(Equal(in.Edges))
		Expect(out.Props).To(Equal(in.Props))
		Expect(out.Control).To(Equal(in.Control))
	})

	It("Should produce a legend whose units default to px when no legend exists upstream", func() {
		out := v1.Migrate(v0.Data{})
		Expect(out.Legend.Visible).To(BeTrue())
		Expect(out.Legend.Position).To(Equal(v1.LegendPosition{
			X: 50, Y: 50,
			Units: &v1.LegendUnits{X: "px", Y: "px"},
		}))
		Expect(out.Legend.Colors).To(BeEmpty())
	})
})
