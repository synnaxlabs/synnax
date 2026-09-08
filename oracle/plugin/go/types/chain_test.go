// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

const chainLiveSource = `
@go output "core/pkg/service/channel"

Key = uuid

Channel struct {
	key Key @key
	name string
	virtual bool
}
`

var _ = Describe("Chain-driven alias split", func() {
	var req *plugin.Request

	BeforeEach(func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/channel/v0.oracle", `
Key = uuid

Channel struct {
	key Key @key
	name string

	@go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
Key = v0.Key

Channel struct {
	key Key @key
	name string
	virtual bool

	@go marshal
}
`)
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), chainLiveSource,
			"schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req = &plugin.Request{Resolutions: table, RepoRoot: root, Versions: resolver}
	})

	It("Should alias per the current file's enumeration", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		var current string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/channel/versions/v1/types.gen.go" {
				current = string(f.Content)
			}
		}
		Expect(current).ToNot(BeEmpty())
		Expect(current).To(ContainSubstring("type Key = v0.Key"))
		Expect(current).To(ContainSubstring("type Channel struct"))
	})

	It("Should regenerate frozen packages from the version files", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		var frozen string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/channel/versions/v0/types.gen.go" {
				frozen = string(f.Content)
			}
		}
		Expect(frozen).ToNot(BeEmpty())
		Expect(frozen).To(ContainSubstring("package v0"))
		Expect(frozen).To(ContainSubstring("type Key = uuid.UUID"))
		Expect(frozen).To(ContainSubstring("type Channel struct"))
		// The frozen shape is v0's: no Virtual field, no alias lines.
		Expect(frozen).ToNot(ContainSubstring("Virtual"))
		Expect(frozen).ToNot(ContainSubstring("= v"))
	})
})

var _ = Describe("Ended chain", func() {
	var req *plugin.Request

	BeforeEach(func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/thing/v0.oracle", `
Thing struct {
	key uuid @key
	name string

	@go marshal
}
`)
		write("schemas/synnax/versions/thing/v1.oracle", "// gone\n")
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), `
@go output "core/thing"

Thing struct {
	key uuid @key
	name string
}
`,
			"schemas/synnax/thing.oracle", "thing",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req = &plugin.Request{Resolutions: table, RepoRoot: root, Versions: resolver}
	})

	It("Should keep frozen packages generating without a current package", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		byPath := make(map[string]string, len(resp.Files))
		for _, f := range resp.Files {
			byPath[f.Path] = string(f.Content)
		}
		frozen := byPath["core/thing/versions/v0/types.gen.go"]
		Expect(frozen).To(ContainSubstring("package v0"))
		Expect(frozen).To(ContainSubstring("type Thing struct"))
		// The live type generates concretely at the package root; no package
		// exists for the tombstone and no root alias files point into versions.
		Expect(byPath["core/thing/types.gen.go"]).
			To(ContainSubstring("type Thing struct"))
		for path := range byPath {
			Expect(path).ToNot(ContainSubstring("thing/versions/v1"))
			Expect(path).ToNot(Equal("core/thing/versions/types.gen.go"))
		}
	})
})
