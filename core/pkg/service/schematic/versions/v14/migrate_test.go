// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14_test

import (
	"embed"
	"encoding/hex"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	v14 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v14"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
)

//go:embed testdata/*.hex
var fixtures embed.FS

// loadWire returns the Orc payload stored as hex in testdata. v13_released.hex is a v13
// schematic with a select and a state indicator, encoded by the core/v0.62.0 codec.
func loadWire(name string) []byte {
	GinkgoHelper()
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	return MustSucceed(hex.DecodeString(strings.ReplaceAll(string(raw), "\n", "")))
}

var _ = Describe("MigrateSchematic", func() {
	It(
		"Should lift a schematic that 0.62.0 stored with no enumeration",
		func(ctx SpecContext) {
			var stored v13.Schematic
			Expect(
				orc.Codec.Decode(loadWire("v13_released.hex"), &stored),
			).To(Succeed())
			out := MustSucceed(v14.MigrateSchematic(ctx, stored))
			Expect(out.Key).To(Equal(stored.Key))
			Expect(out.Name).To(Equal("Valve panel"))
			Expect(out.Nodes).To(HaveLen(2))
			sel, ok := out.Nodes[0].Config.Variant.(v14.SelectNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(sel.Enumeration).To(BeNil())
			Expect(sel.CommandChannel).To(BeEquivalentTo(1048577))
			Expect(sel.Options).To(HaveLen(2))
			Expect(sel.Options[1].Name).To(Equal("Open"))
			Expect(sel.Options[1].Color).ToNot(BeNil())
			ind, ok := out.Nodes[1].Config.Variant.(v14.StateIndicatorNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(ind.Enumeration).To(BeNil())
			Expect(ind.Channel).To(BeEquivalentTo(1048578))
			Expect(ind.Options).To(HaveLen(2))
		},
	)
})
