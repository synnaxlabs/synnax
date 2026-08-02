// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin/go/freeze"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("VersionsGate", func() {
	var (
		root  string
		write func(rel, content string)
	)

	const liveV0 = `
@go output "core/pkg/service/channel"

Channel struct {
	key uuid @key
	name string

	@go version 0
	@go marshal
}
`

	analyzeLive := func(source string) *pipeline.Result {
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), source,
			"schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		return &pipeline.Result{Resolutions: table}
	}

	run := func(p *pipeline.Result) check.GateReport {
		return check.VersionsGate{}.Run(
			GinkgoT().Context(), p, check.Env{RepoRoot: root},
		)
	}

	// canonicalV0 writes the v0 file exactly as freeze would emit it, so the
	// drift check starts clean.
	canonicalV0 := func(p *pipeline.Result) {
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		out := MustSucceed(freeze.Canonical(GinkgoT().Context(), freeze.Input{
			Live:     p.Resolutions,
			Resolver: resolver,
			Chain:    chains["schemas/synnax/channel"],
			N:        0,
		}))
		write("schemas/synnax/versions/channel/v0.oracle", out)
	}

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		write = func(rel, content string) {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
	})

	It("Should pass when no chains exist", func() {
		p := analyzeLive(liveV0)
		Expect(run(p).Status).To(Equal(check.StatusPass))
	})

	It("Should pass a clean chain", func() {
		write("schemas/synnax/versions/channel/v0.oracle", "placeholder = uuid\n")
		p := analyzeLive(liveV0)
		canonicalV0(p)
		report := run(p)
		Expect(report.Findings).To(BeEmpty())
		Expect(report.Status).To(Equal(check.StatusPass))
	})

	It("Should fail on drift between the live schema and the current file", func() {
		write("schemas/synnax/versions/channel/v0.oracle", "placeholder = uuid\n")
		p := analyzeLive(liveV0)
		canonicalV0(p)
		drifted := analyzeLive(`
@go output "core/pkg/service/channel"

Channel struct {
	key uuid @key
	name string
	virtual bool

	@go version 0
	@go marshal
}
`)
		report := run(drifted)
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Message).To(ContainSubstring("drifts"))
		Expect(report.Findings[0].FixHint).To(ContainSubstring("oracle migrate"))
	})

	It("Should fail a redeclaration identical to its predecessor", func() {
		write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
	key uuid @key
	name string

	@go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
Channel struct {
	key uuid @key
	name string

	@go marshal
}
`)
		p := analyzeLive(`
@go output "core/pkg/service/channel"

Channel struct {
	key uuid @key
	name string

	@go version 1
	@go marshal
}
`)
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		var minimality bool
		for _, f := range report.Findings {
			if f.FixHint == "use an alias: Channel = v0.Channel" {
				minimality = true
			}
		}
		Expect(minimality).To(BeTrue(), "expected a minimality finding")
	})

	It("Should fail versioned paths without a chain", func() {
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Message).
			To(ContainSubstring("has no version chain"))
	})
})
