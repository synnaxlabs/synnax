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
	"encoding/json/jsontext"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"
	"github.com/synnaxlabs/x/spatial"
)

// nonZeroV0 builds a v0.Data with every model field populated so passthrough
// regressions in v1.Migrate surface.
func nonZeroV0() v0.Data {
	return v0.Data{
		Version:  v0.Version,
		Snapshot: true,
		Nodes: []v0.Node{
			{Key: "n1", Position: spatial.XY{X: 5, Y: 6}, ZIndex: new(3)},
		},
		Edges: []v0.Edge{
			{Key: "e1", Source: "n1", Target: "n2", SourceHandle: new("a")},
		},
		Props: map[string]jsontext.Value{"n1": jsontext.Value(`{"k":1}`)},
	}
}

var _ = Describe("Migrate", func() {
	It("Should restamp the version and pass every model field through", func() {
		in := nonZeroV0()

		out := v1.Migrate(in)

		Expect(out.Version).To(Equal(v1.Version))
		Expect(out.Snapshot).To(Equal(in.Snapshot))
		Expect(out.Nodes).To(Equal(in.Nodes))
		Expect(out.Edges).To(Equal(in.Edges))
		Expect(out.Props).To(Equal(in.Props))
	})
})
