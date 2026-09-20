// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framework_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Collect helpers", func() {
	var (
		table *resolution.Table
		req   *plugin.Request
	)

	BeforeEach(func() {
		table = resolution.NewTable()
		req = &plugin.Request{Resolutions: table}
	})

	Describe("CollectStructs", func() {
		It("Should group structs by output path", func() {
			Expect(table.Add(newGoType("A", resolution.StructForm{}, "pkg/a"))).
				To(Succeed())
			Expect(table.Add(newGoType("B", resolution.StructForm{}, "pkg/a"))).
				To(Succeed())
			c := MustSucceed(framework.CollectStructs("go", req))
			Expect(c.Paths()).To(Equal([]string{"pkg/a"}))
			Expect(c.Get("pkg/a")).To(HaveLen(2))
		})

		It("Should reject an output path escaping the repository", func() {
			req.RepoRoot = GinkgoT().TempDir()
			Expect(table.Add(newGoType("A", resolution.StructForm{}, "../out"))).
				To(Succeed())
			Expect(framework.CollectStructs("go", req)).Error().
				To(MatchError(ContainSubstring("invalid output path for A")))
		})
	})

	Describe("CollectDistinct", func() {
		It("Should collect distinct types", func() {
			Expect(table.Add(newGoType("Key", resolution.DistinctForm{}, "pkg/a"))).
				To(Succeed())
			c := MustSucceed(framework.CollectDistinct("go", req))
			Expect(c.Get("pkg/a")).To(HaveLen(1))
		})

		It("Should reject an output path escaping the repository", func() {
			req.RepoRoot = GinkgoT().TempDir()
			Expect(table.Add(newGoType("Key", resolution.DistinctForm{}, "../out"))).
				To(Succeed())
			Expect(framework.CollectDistinct("go", req)).Error().
				To(MatchError(ContainSubstring("invalid output path for Key")))
		})
	})

	Describe("CollectAliases", func() {
		It("Should collect alias types", func() {
			Expect(table.Add(newGoType("Name", resolution.AliasForm{}, "pkg/a"))).
				To(Succeed())
			c := MustSucceed(framework.CollectAliases("go", req))
			Expect(c.Get("pkg/a")).To(HaveLen(1))
		})

		It("Should reject an output path escaping the repository", func() {
			req.RepoRoot = GinkgoT().TempDir()
			Expect(table.Add(newGoType("Name", resolution.AliasForm{}, "../out"))).
				To(Succeed())
			Expect(framework.CollectAliases("go", req)).Error().
				To(MatchError(ContainSubstring("invalid output path for Name")))
		})
	})
})
