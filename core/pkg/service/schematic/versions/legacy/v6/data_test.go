// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v6 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v6"
)

var _ = Describe("Data", func() {
	It("Should stamp the version the Console wrote", func() {
		Expect(v6.Version).To(BeEquivalentTo(6))
	})

	It("Should decode the Console's camelCase node keys", func() {
		var d v6.Data
		Expect(json.Unmarshal([]byte(`{
			"snapshot": true,
			"nodes": [{"key": "n1", "position": {"x": 3, "y": 4}, "zIndex": 2}],
			"edges": [{
				"key": "e1",
				"source": {"node": "n1", "param": "out"},
				"target": {"node": "n2", "param": "in"}
			}],
			"configs": {"n1": {"variant": "tank"}}
		}`), &d)).To(Succeed())
		Expect(d.Snapshot).To(BeTrue())
		Expect(d.Nodes).To(HaveLen(1))
		Expect(d.Nodes[0].ZIndex).To(Equal(int16(2)))
		Expect(d.Nodes[0].Position.X).To(Equal(3.0))
		Expect(d.Edges[0].Source).To(Equal(v6.Handle{Node: "n1", Param: "out"}))
		Expect(d.Edges[0].Target).To(Equal(v6.Handle{Node: "n2", Param: "in"}))
		Expect(d.Configs).To(HaveKey("n1"))
	})

	It("Should leave ZIndex zero when the key is snake_case", func() {
		var d v6.Data
		Expect(json.Unmarshal(
			[]byte(`{"nodes": [{"key": "n1", "z_index": 9}]}`), &d,
		)).To(Succeed())
		Expect(d.Nodes[0].ZIndex).To(BeZero())
	})
})
