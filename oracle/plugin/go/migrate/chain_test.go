// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/migrate"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Chain migrate.gen.go", func() {
	It("Should emit auto-copies as a pure function of adjacent files", func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
	key uuid @key
	name string

	@go marshal
	@go migrate
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
Channel struct {
	key uuid @key
	name string
	virtual bool

	@go marshal
	@go migrate
}
`)
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), `
@go output "core/pkg/service/channel"

Channel struct {
	key uuid @key
	name string
	virtual bool

	@go version 1
	@go marshal
	@go migrate
}
`,
			"schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		resp := MustSucceed(migrate.New().Generate(&plugin.Request{
			Resolutions: table, RepoRoot: root, Versions: resolver,
		}))
		var content string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/channel/versions/v1/migrate.gen.go" {
				content = string(f.Content)
			}
		}
		Expect(content).ToNot(BeEmpty())
		Expect(content).To(ContainSubstring("package v1"))
		Expect(content).To(ContainSubstring("igrateChannel"))
	})
})
