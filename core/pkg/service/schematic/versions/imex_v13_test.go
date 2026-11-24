// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	color "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DecodeV13", func() {
	options := func(n v13.Node) []v13.StateMapping {
		GinkgoHelper()
		switch cfg := n.Config.Variant.(type) {
		case v13.SelectNodeConfig:
			return cfg.Options
		case v13.StateIndicatorNodeConfig:
			return cfg.Options
		}
		Fail("not a select or a state indicator")
		return nil
	}

	DescribeTable("Should keep the options of each select of an old file",
		func(ctx SpecContext, path string) {
			s, ok := MustSucceed2(versions.DecodeV13(ctx, LoadEnvelope(path)))
			Expect(ok).To(BeTrue())
			Expect(options(s.Nodes[0])).To(Equal([]v13.StateMapping{
				{
					Key:   "off",
					Name:  "Off",
					Value: 0,
					Color: &color.Color{R: 255, A: 255},
				},
				{Key: "on", Name: "On", Value: 1, Color: &color.Color{G: 255, A: 255}},
			}))
		},
		Entry("v13", "testdata/import_v13_states.json"),
		Entry("v12", "testdata/import_v12_states.json"),
		Entry("v9", "testdata/import_v9_states.json"),
		Entry("v7", "testdata/import_v7_states.json"),
		Entry("a Console export", "testdata/import_console_states.json"),
	)

	DescribeTable("Should keep each node at the index that DecodeImExEnvelope gives it",
		func(ctx SpecContext, path string) {
			env := LoadEnvelope(path)
			old, ok := MustSucceed2(versions.DecodeV13(ctx, env))
			Expect(ok).To(BeTrue())
			current := MustSucceed(versions.DecodeImExEnvelope(ctx, env))
			keys := make([]string, len(old.Nodes))
			for i, n := range old.Nodes {
				keys[i] = n.Key
			}
			currentKeys := make([]string, len(current.Nodes))
			for i, n := range current.Nodes {
				currentKeys[i] = n.Key
			}
			Expect(keys).ToNot(BeEmpty())
			Expect(keys).To(Equal(currentKeys))
		},
		Entry("v13", "testdata/import_v13.json"),
		Entry("v11", "testdata/import_v11.json"),
		Entry("v10", "testdata/import_v10_button.json"),
		Entry("v8", "testdata/import_v8.json"),
		Entry("v7", "testdata/import_v7.json"),
		Entry("a v6 Console export", "testdata/import_typed_console.json"),
		Entry("a v3 Console state", "testdata/import_v3_state.json"),
		Entry("a v0 Console state", "testdata/import_v0_state.json"),
	)

	DescribeTable("Should fail on a file that its version cannot read",
		func(ctx SpecContext, path, msg string) {
			Expect(versions.DecodeV13(ctx, LoadEnvelope(path))).
				Error().To(MatchError(ContainSubstring(msg)))
		},
		Entry("a rejected node", "testdata/import_v8_rejected.json", "node n1"),
		Entry(
			"bad nodes",
			"testdata/import_v12_bad_nodes.json",
			`field "nodes" expects []v8.Node, got JSON string`,
		),
		Entry(
			"bad Console nodes",
			"testdata/import_console_bad_nodes.json",
			`field "nodes" expects []v6.Node, got JSON string`,
		),
	)

	It("Should report a file of v14 or newer as not old", func(ctx SpecContext) {
		env := imex.Envelope{Version: 14, Type: "schematic"}
		s, ok := MustSucceed2(versions.DecodeV13(ctx, env))
		Expect(ok).To(BeFalse())
		Expect(s).To(BeZero())
	})

	It("Should fail on a Console file with no nodes", func(ctx SpecContext) {
		Expect(versions.DecodeV13(ctx, LoadEnvelope(
			"testdata/import_unrecognized.json",
		))).Error().To(MatchError(ContainSubstring(
			`file is not a schematic: no "props" field`,
		)))
	})
})
