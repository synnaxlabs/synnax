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
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

func jsonMap(raw string) msgpack.EncodedJSON {
	GinkgoHelper()
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

// stateAt renders a Console schematic state stamped at the given semver major. Every
// version in the chain shares the v0 field names, so one body exercises each arm.
func stateAt(version string) msgpack.EncodedJSON {
	GinkgoHelper()
	return jsonMap(`{
		"version": "` + version + `",
		"snapshot": true,
		"nodes": [{"key": "n1", "position": {"x": 4, "y": 5}, "zIndex": 7}],
		"edges": [{
			"key": "e1",
			"source": "n1",
			"target": "n2",
			"sourceHandle": "out",
			"targetHandle": "in",
			"data": {"color": "#00ff00"}
		}],
		"props": {"n1": {"key": "valve", "color": "#ff0000"}}
	}`)
}

var _ = Describe("MigrateData", func() {
	DescribeTable("Should lift every Console state version to the latest snapshot",
		func(version string) {
			d := MustSucceed(legacy.MigrateData(stateAt(version)))
			Expect(d.Version).To(Equal(legacy.DataVersion))
			Expect(d.Snapshot).To(BeTrue())
			Expect(d.Nodes).To(HaveLen(1))
			Expect(d.Nodes[0].Key).To(Equal("n1"))
			Expect(*d.Nodes[0].ZIndex).To(Equal(7))
			Expect(d.Edges).To(HaveLen(1))
			Expect(d.Edges[0].Key).To(Equal("e1"))
			Expect(*d.Edges[0].SourceHandle).To(Equal("out"))
			Expect(d.Props).To(HaveKey("n1"))
		},
		Entry("v0", "0.0.0"),
		Entry("v1", "1.0.0"),
		Entry("v2", "2.0.0"),
		Entry("v3", "3.0.0"),
		Entry("v4", "4.0.0"),
		Entry("v5", "5.0.0"),
	)

	It("Should carry the opaque edge data bag forward from a pre-v3 blob", func() {
		d := MustSucceed(legacy.MigrateData(stateAt("0.0.0")))
		Expect(d.Edges[0].Data).To(MatchJSON(`{"color": "#00ff00"}`))
		Expect(d.Edges[0].Segments).To(BeEmpty())
	})

	It("Should keep the segments a v3 blob already declares", func() {
		d := MustSucceed(legacy.MigrateData(jsonMap(`{
			"version": "3.0.0",
			"nodes": [],
			"edges": [{
				"key": "e1",
				"source": "n1",
				"target": "n2",
				"segments": [{"direction": "x", "length": 12}]
			}],
			"props": {}
		}`)))
		Expect(d.Edges[0].Segments).To(Equal([]legacy.Segment{
			{Direction: "x", Length: 12},
		}))
	})

	It("Should drop edges with an empty source or target", func() {
		d := MustSucceed(legacy.MigrateData(jsonMap(`{
			"version": "5.0.0",
			"nodes": [],
			"edges": [
				{"key": "keep", "source": "n1", "target": "n2"},
				{"key": "no-source", "source": "", "target": "n2"},
				{"key": "no-target", "source": "n1", "target": ""}
			],
			"props": {}
		}`)))
		Expect(d.Edges).To(HaveLen(1))
		Expect(d.Edges[0].Key).To(Equal("keep"))
	})

	It("Should fall back to v0 when the blob has no version field", func() {
		d := MustSucceed(legacy.MigrateData(jsonMap(`{
			"nodes": [{"key": "n1", "position": {"x": 1, "y": 2}}],
			"edges": [],
			"props": {}
		}`)))
		Expect(d.Version).To(Equal(legacy.DataVersion))
		Expect(d.Nodes).To(HaveLen(1))
	})

	It("Should decode a nil blob into a zero Data at the latest version", func() {
		d := MustSucceed(legacy.MigrateData(nil))
		Expect(d.Version).To(Equal(legacy.DataVersion))
		Expect(d.Nodes).To(BeEmpty())
	})

	It("Should error on an unknown declared version", func() {
		Expect(legacy.MigrateData(jsonMap(`{"version": "99.0.0"}`))).Error().
			To(MatchError(ContainSubstring("unknown schematic data version 99")))
	})

	It("Should error when the blob does not match its declared version", func() {
		Expect(legacy.MigrateData(jsonMap(`{
			"version": "0.0.0", "nodes": {"not": "an array"}
		}`))).Error().To(MatchError(ContainSubstring("decode v0 schematic data")))
	})
})

var _ = Describe("Versions", func() {
	It("Should report the Console's highest written version as LastVersion", func() {
		Expect(legacy.LastVersion).To(BeEquivalentTo(6))
	})

	It("Should terminate the state chain one version below LastVersion", func() {
		Expect(legacy.DataVersion).To(BeEquivalentTo(5))
	})
})
