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

}
`

	const channelV0 = `
Channel struct {
	key uuid @key
	name string

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
		chains := MustSucceed(versions.Discover(root))
		var resolver *versions.Resolver
		if len(chains) > 0 {
			resolver = versions.NewResolver(
				chains, analyzer.NewStandardFileLoader(root),
			)
		}
		return &pipeline.Result{
			Resolutions: table,
			Chains:      chains,
			Versions:    resolver,
		}
	}

	run := func(p *pipeline.Result) check.GateReport {
		return check.VersionsGate{}.Run(
			GinkgoT().Context(), p, check.Env{RepoRoot: root},
		)
	}

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		write = func(rel, content string) {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("licenses/headers/template.txt",
			"Copyright {{YEAR}} Synnax Labs, Inc.\n")
	})

	It("Should pass when no chains exist", func() {
		p := analyzeLive(liveV0)
		Expect(run(p).Status).To(Equal(check.StatusPass))
	})

	It("Should pass when the pipeline produced no resolutions", func() {
		Expect(run(&pipeline.Result{}).Status).To(Equal(check.StatusPass))
	})

	It("Should pass when chains exist but the resolver is absent", func() {
		write("schemas/synnax/versions/channel/v0.oracle", channelV0)
		p := analyzeLive(liveV0)
		p.Versions = nil
		Expect(run(p).Status).To(Equal(check.StatusPass))
	})

	It("Should fail a version file the resolver cannot analyze", func() {
		write("schemas/synnax/versions/channel/v0.oracle", "Channel struct {{{\n")
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Path).
			To(Equal("schemas/synnax/versions/channel/v0.oracle"))
		Expect(report.Findings[0].Message).To(ContainSubstring("failed to parse"))
	})

	It("Should pass a clean chain", func() {
		write("schemas/synnax/versions/channel/v0.oracle", channelV0)
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Findings).To(BeEmpty())
		Expect(report.Status).To(Equal(check.StatusPass))
	})

	mergeLive := func(source string) []byte {
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		return []byte(MustSucceed(versions.MergeLive(
			GinkgoT().Context(), resolver,
			chains["schemas/synnax/channel"], source,
		)))
	}

	It("Should pass when the live file matches its merged projection", func() {
		write("schemas/synnax/versions/channel/v0.oracle", channelV0)
		merged := mergeLive(liveV0)
		write("schemas/synnax/channel.oracle", string(merged))
		p := analyzeLive(string(merged))
		p.MergedSources = map[string][]byte{
			"schemas/synnax/channel.oracle": merged,
		}
		report := run(p)
		Expect(report.Findings).To(BeEmpty())
		Expect(report.Status).To(Equal(check.StatusPass))
	})

	It("Should fail on drift between the live file and its version files", func() {
		write("schemas/synnax/versions/channel/v0.oracle", channelV0)
		// The on-disk live file lacks the version-owned marshal tag the
		// merged projection carries.
		write("schemas/synnax/channel.oracle", liveV0)
		p := analyzeLive(liveV0)
		p.MergedSources = map[string][]byte{
			"schemas/synnax/channel.oracle": mergeLive(liveV0),
		}
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Message).
			To(ContainSubstring("drifts from its version files"))
		Expect(report.Findings[0].FixHint).To(ContainSubstring("oracle sync"))
	})

	It("Should capture a diff on drift when the env requests it", func() {
		write("schemas/synnax/versions/channel/v0.oracle", channelV0)
		write("schemas/synnax/channel.oracle", liveV0)
		p := analyzeLive(liveV0)
		p.MergedSources = map[string][]byte{
			"schemas/synnax/channel.oracle": mergeLive(liveV0),
		}
		report := check.VersionsGate{}.Run(
			GinkgoT().Context(), p,
			check.Env{RepoRoot: root, IncludeDiffs: true},
		)
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Diff).To(ContainSubstring("@go marshal"))
	})

	It("Should fail drift when the live file is missing on disk", func() {
		write("schemas/synnax/versions/channel/v0.oracle", channelV0)
		p := analyzeLive(liveV0)
		p.MergedSources = map[string][]byte{
			"schemas/synnax/channel.oracle": mergeLive(liveV0),
		}
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Message).
			To(ContainSubstring("drifts from its version files"))
	})

	It("Should allow a stored reference to a chainless live schema", func() {
		write("schemas/arc/program.oracle", "Program struct {\n\twasm bytes\n}\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/arc/program"

Channel struct {
	key uuid @key
	compiled program.Program

	@go marshal
}
`)
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Findings).To(BeEmpty())
		Expect(report.Status).To(Equal(check.StatusPass))
	})

	It("Should fail a stored pin into an ended chain", func() {
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		write("schemas/x/versions/telem/v1.oracle", "// gone\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v0"

Channel struct {
	key uuid @key
	created telem.TimeStamp

	@go marshal
}
`)
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		var ended bool
		for _, f := range report.Findings {
			if f.Message == "schemas/synnax/channel's current surface stores "+
				"telem, but schemas/x/telem's chain ended at v1; nothing may "+
				"persist an ended resource" {
				ended = true
			}
		}
		Expect(ended).To(BeTrue(), "expected an ended-chain finding")
	})

	It("Should allow a lagging pin whose members alias the current shape", func() {
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		write(
			"schemas/x/versions/telem/v1.oracle",
			"TimeStamp = v0.TimeStamp\n\nExtra = int32\n",
		)
		write("schemas/x/telem.oracle", "TimeStamp = int64\nExtra = int32\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v0"

Channel struct {
	key uuid @key
	created telem.TimeStamp

	@go marshal
}
`)
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Findings).To(BeEmpty())
		Expect(report.Status).To(Equal(check.StatusPass))
	})

	It("Should walk stored references through every declaration form", func() {
		write(
			"schemas/x/versions/telem/v0.oracle",
			"TimeStamp = int64\n\nRange struct {\n\tstart TimeStamp\n}\n",
		)
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v0"

Stamp = telem.TimeStamp

Span telem.TimeStamp

Event union on kind {
	window telem.Range
}

Channel struct {
	key uuid @key
	stamp Stamp
	span Span
	event Event

	@go marshal
}
`)
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Findings).To(BeEmpty())
		Expect(report.Status).To(Equal(check.StatusPass))
	})

	It("Should fail a stored reference that imports a live schema", func() {
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		write("schemas/x/telem.oracle", "TimeStamp = int64\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/telem"

Channel struct {
	key uuid @key
	created telem.TimeStamp

	@go marshal
}
`)
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		var placement bool
		for _, f := range report.Findings {
			if f.Message == "schemas/synnax/channel v0 stores telem "+
				"but imports its live schema" {
				placement = true
			}
		}
		Expect(placement).To(BeTrue(), "expected a placement finding")
	})

	It("Should fail a resolved-only reference that pins", func() {
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		write("schemas/x/telem.oracle", "TimeStamp = int64\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v0"

Channel struct {
	key uuid @key
	seen telem.TimeStamp {
		@go marshal omit
	}

	@go marshal
}
`)
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		var placement bool
		for _, f := range report.Findings {
			if f.Message == "schemas/synnax/channel v0 only resolves telem "+
				"at read time but pins it" {
				placement = true
			}
		}
		Expect(placement).To(BeTrue(), "expected a placement finding")
	})

	It("Should fail a stored pin lagging its dependency's current version", func() {
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		write("schemas/x/versions/telem/v1.oracle", "TimeStamp = int32\n")
		write("schemas/x/telem.oracle", "TimeStamp = int32\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v0"

Channel struct {
	key uuid @key
	created telem.TimeStamp

	@go marshal
}
`)
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		var currency bool
		for _, f := range report.Findings {
			if f.Message == "schemas/synnax/channel's current surface resolves "+
				"telem at v0, but schemas/x/telem's current version is v1" {
				currency = true
			}
		}
		Expect(currency).To(BeTrue(), "expected a pin-currency finding")
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
})
