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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v5"
	"github.com/synnaxlabs/x/spatial"
)

// nonZeroV4 builds a v4.Data with every model field populated so passthrough
// regressions in v5.Migrate surface.
func nonZeroV4() v4.Data {
	return v4.Data{
		Version:  v4.Version,
		Snapshot: true,
		Nodes: []v0.Node{
			{Key: "n1", Position: spatial.XY{X: 5, Y: 6}, ZIndex: new(3)},
		},
		Edges: []v3.Edge{
			{Key: "e1", Source: "n1", Target: "n2", Segments: []v3.Segment{}},
		},
		Props: map[string]json.RawMessage{"n1": json.RawMessage(`{"k":1}`)},
	}
}

var _ = Describe("Migrate", func() {
	It("Should restamp the version and pass every model field through", func() {
		in := nonZeroV4()

		out := v5.Migrate(in)

		Expect(out.Version).To(Equal(v5.Version))
		Expect(out.Snapshot).To(Equal(in.Snapshot))
		Expect(out.Nodes).To(Equal(in.Nodes))
		Expect(out.Edges).To(Equal(in.Edges))
		Expect(out.Props).To(Equal(in.Props))
	})
})
