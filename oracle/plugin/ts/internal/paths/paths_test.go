// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package paths_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/paths"
)

func TestPaths(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TS Paths Suite")
}

var _ = Describe("FindPackage", func() {
	It("Should return nil for paths outside any known workspace", func() {
		Expect(paths.FindPackage("vendor/foo")).To(BeNil())
	})
	It("Should return the matching mapping for a client/ts path", func() {
		pkg := paths.FindPackage("client/ts/src/schematic")
		Expect(pkg).ToNot(BeNil())
		Expect(pkg.PackageName).To(Equal("@synnaxlabs/client"))
	})
	It("Should match the longest applicable prefix", func() {
		pkg := paths.FindPackage("pluto/src/vis/diagram")
		Expect(pkg).ToNot(BeNil())
		Expect(pkg.PackageName).To(Equal("@synnaxlabs/pluto"))
	})
})

var _ = Describe("CalculateImport", func() {
	It("Should produce an internal alias when both paths share a workspace", func() {
		got := paths.CalculateImport(
			"client/ts/src/schematic",
			"client/ts/src/schematic/types.gen",
		)
		Expect(got).To(Equal("@/schematic/types.gen"))
	})
	It("Should produce a package name when the target is a different workspace", func() {
		got := paths.CalculateImport(
			"client/ts/src/schematic",
			"x/ts/src/spatial",
		)
		Expect(got).To(Equal("@synnaxlabs/x"))
	})
	It("Should fall back to a relative import when the source is unknown", func() {
		got := paths.CalculateImport("vendor/foo", "vendor/bar")
		Expect(got).To(Equal("../bar"))
	})
	It("Should fall back to a relative import when the target is unknown", func() {
		got := paths.CalculateImport("client/ts/src/schematic", "vendor/bar")
		Expect(got).To(HavePrefix("../"))
	})
	It("Should normalize relative paths to slash separators", func() {
		got := paths.CalculateImport("vendor/a/b", "vendor/c/d")
		Expect(got).To(ContainSubstring("/"))
		Expect(got).ToNot(ContainSubstring(`\`))
	})
})
