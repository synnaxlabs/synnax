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

Key = uuid {
	@go version 1
}

Channel struct {
	key Key @key
	name string
	virtual bool

	@go version 1
	@go marshal
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
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
Key = v0.Key

Channel struct {
	key Key @key
	name string
	virtual bool
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

	It("Should error when the chain and @go version disagree", func() {
		disagreeing := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(),
			`
@go output "core/pkg/service/channel"

Channel struct {
	key uuid @key

	@go version 2
	@go marshal
}
`,
			"schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(req.RepoRoot), disagreeing,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		badReq := &plugin.Request{
			Resolutions: disagreeing,
			RepoRoot:    req.RepoRoot,
			Versions:    req.Versions,
		}
		Expect(gotypes.New(gotypes.DefaultOptions()).Generate(badReq)).Error().
			To(MatchError(ContainSubstring("chain's current file is v1")))
	})
})
