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
		Entry("v7", "testdata/import_v7_states.json"),
		Entry("a Console export", "testdata/import_console_states.json"),
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
