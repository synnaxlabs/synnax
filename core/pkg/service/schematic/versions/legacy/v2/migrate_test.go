// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v2"
)

// createV1 builds a v1.Data with every field populated to a non-zero value so
// passthrough regressions surface.
func createV1() v1.Data {
	srcH, tgtH := "out", "in"
	return v1.Data{
		Version:         v1.Version,
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
		Legend:  v1.ZeroLegend,
	}
}

var _ = Describe("Migrate", func() {
	It("Should add the schematic type literal and the default viewport mode", func() {
		out := v2.Migrate(createV1())
		Expect(out.Version).To(Equal(v2.Version))
		Expect(out.Type).To(Equal("schematic"))
		Expect(out.ViewportMode).To(Equal("select"))
	})

	It("Should generate a fresh uuid key on every call", func() {
		in := createV1()
		a := v2.Migrate(in)
		b := v2.Migrate(in)
		Expect(a.Key).NotTo(BeEmpty())
		Expect(b.Key).NotTo(BeEmpty())
		Expect(a.Key).NotTo(Equal(b.Key))
	})

	It("Should pass every v1 field through unchanged", func() {
		in := createV1()
		out := v2.Migrate(in)
		Expect(out.Editable).To(Equal(in.Editable))
		Expect(out.FitViewOnResize).To(Equal(in.FitViewOnResize))
		Expect(out.Snapshot).To(Equal(in.Snapshot))
		Expect(out.RemoteCreated).To(Equal(in.RemoteCreated))
		Expect(out.Viewport).To(Equal(in.Viewport))
		Expect(out.Nodes).To(Equal(in.Nodes))
		Expect(out.Edges).To(Equal(in.Edges))
		Expect(out.Props).To(Equal(in.Props))
		Expect(out.Control).To(Equal(in.Control))
		Expect(out.Legend).To(Equal(in.Legend))
	})
})
