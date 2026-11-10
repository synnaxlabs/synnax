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
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		GinkgoHelper()
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
		return check.NewVersionsGate().Run(
			GinkgoT().Context(), p, check.Env{RepoRoot: root},
		)
	}

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		write = func(rel, content string) {
			GinkgoHelper()
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
		GinkgoHelper()
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
		report := check.NewVersionsGate().Run(
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

	placementFindings := func(report check.GateReport) []string {
		GinkgoHelper()
		var found []string
		for _, f := range report.Findings {
			if strings.Contains(f.Message, "but imports its live schema") {
				found = append(found, f.Message)
			}
		}
		return found
	}

	DescribeTable("Should fail a reference that imports a live schema",
		func(decls string) {
			write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
			write("schemas/x/telem.oracle", "TimeStamp = int64\n")
			write(
				"schemas/synnax/versions/channel/v0.oracle",
				"import \"schemas/x/telem\"\n\n"+decls,
			)
			report := run(analyzeLive(liveV0))
			Expect(report.Status).To(Equal(check.StatusFail))
			Expect(placementFindings(report)).To(ConsistOf(
				"schemas/synnax/channel v0 references telem but imports its live schema",
			))
		},
		Entry("when the reference is stored", `
Channel struct {
	key uuid @key
	created telem.TimeStamp

	@go marshal
}
`),
		Entry("when the reference is an omitted field", `
Channel struct {
	key uuid @key
	seen telem.TimeStamp {
		@go marshal omit
	}

	@go marshal
}
`),
		Entry("when an inline union variant holds the reference", `
Shape union on kind {
	circle {
		at telem.TimeStamp
	}

	@go marshal
}

Channel struct {
	key uuid @key
	shape Shape

	@go marshal
}
`),
	)

	It("Should accept an omitted field that pins its dependency", func() {
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
		Expect(placementFindings(run(analyzeLive(liveV0)))).To(BeEmpty())
	})

	It("Should accept a live import of an ended chain", func() {
		write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
		write("schemas/x/versions/telem/v1.oracle", "// Ended.\n")
		write("schemas/x/telem.oracle", "TimeStamp = int64\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/telem"

Channel struct {
	key uuid @key
	seen telem.TimeStamp {
		@go marshal omit
	}

	@go marshal
}
`)
		Expect(placementFindings(run(analyzeLive(liveV0)))).To(BeEmpty())
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

	It("Should fail an omitted-field pin lagging its dependency's current version",
		func() {
			write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
			write("schemas/x/versions/telem/v1.oracle", "TimeStamp = int32\n")
			write("schemas/x/telem.oracle", "TimeStamp = int32\n")
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
			report := run(analyzeLive(liveV0))
			Expect(report.Status).To(Equal(check.StatusFail))
			var messages []string
			for _, f := range report.Findings {
				messages = append(messages, f.Message)
			}
			Expect(messages).To(ContainElement(
				"schemas/synnax/channel's current surface resolves telem at v0 for " +
					"omitted fields, but schemas/x/telem's current version is v1",
			))
		},
	)

	DescribeTable(
		"Should fail a lagging omitted-field pin that the current surface uses",
		func(files map[string]string) {
			write("schemas/x/versions/telem/v0.oracle",
				"TimeStamp = int64\n\nStamp = int64\n")
			write("schemas/x/versions/telem/v1.oracle",
				"TimeStamp = int32\n\nStamp = v0.Stamp\n")
			write("schemas/x/telem.oracle", "TimeStamp = int32\n\nStamp = int64\n")
			for rel, content := range files {
				write(rel, content)
			}
			var messages []string
			for _, f := range run(analyzeLive(liveV0)).Findings {
				messages = append(messages, f.Message)
			}
			Expect(messages).To(ContainElement(ContainSubstring(
				"resolves telem at v0 for omitted fields",
			)))
		},
		Entry("when an earlier version file declares the member", map[string]string{
			"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/x/versions/telem/v0"

Channel struct {
	key uuid @key
	seen telem.TimeStamp {
		@go marshal omit
	}

	@go marshal
}
`,
			"schemas/synnax/versions/channel/v1.oracle": "Channel = v0.Channel\n",
		}),
		Entry("when an inline union variant holds the omitted field", map[string]string{
			"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/x/versions/telem/v0"

Shape union on kind {
	circle {
		seen telem.TimeStamp {
			@go marshal omit
		}
	}

	@go marshal
}

Channel struct {
	key uuid @key
	shape Shape

	@go marshal
}
`,
		}),
		Entry("when a stored reference also uses the dependency", map[string]string{
			"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/x/versions/telem/v0"

Channel struct {
	key uuid @key
	created telem.Stamp
	seen telem.TimeStamp {
		@go marshal omit
	}

	@go marshal
}
`,
		}),
	)

	It("Should leave a lagging stored member to the stored check", func() {
		write("schemas/x/versions/telem/v0.oracle",
			"TimeStamp = int64\n\nStamp = int64\n")
		write("schemas/x/versions/telem/v1.oracle",
			"TimeStamp = v0.TimeStamp\n\nStamp = int32\n")
		write("schemas/x/telem.oracle", "TimeStamp = int64\n\nStamp = int32\n")
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v0"

Channel struct {
	key uuid @key
	created telem.Stamp
	seen telem.TimeStamp {
		@go marshal omit
	}

	@go marshal
}
`)
		var messages []string
		for _, f := range run(analyzeLive(liveV0)).Findings {
			messages = append(messages, f.Message)
		}
		Expect(
			messages,
		).To(ContainElement(ContainSubstring("resolves telem at v0, but")))
		Expect(messages).ToNot(ContainElement(ContainSubstring("for omitted fields")))
	})

	It("Should accept an omitted-field pin at its dependency's current version",
		func() {
			write("schemas/x/versions/telem/v0.oracle", "TimeStamp = int64\n")
			write("schemas/x/versions/telem/v1.oracle", "TimeStamp = int32\n")
			write("schemas/x/telem.oracle", "TimeStamp = int32\n")
			write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/telem/v1"

Channel struct {
	key uuid @key
	seen telem.TimeStamp {
		@go marshal omit
	}

	@go marshal
}
`)
			for _, f := range run(analyzeLive(liveV0)).Findings {
				Expect(f.Message).ToNot(ContainSubstring("for omitted fields"))
			}
		},
	)

	It("Should fail a stored pin that only a borrowed enum reaches", func() {
		write("schemas/x/versions/auth/v0.oracle", `
Method enum {
	password = "password"
}
`)
		write("schemas/x/versions/auth/v1.oracle", `
Method enum {
	password = "password"
	key = "key"
}
`)
		write("schemas/x/auth.oracle", `
Method enum {
	password = "password"
	key = "key"
}
`)
		write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/auth/v0"

Channel struct {
	key uuid @key
	proof Proof

	@go marshal
}

Credentials struct { username string }

Proof union on method auth.Method {
	password Credentials
}
`)
		p := analyzeLive(liveV0)
		report := run(p)
		Expect(report.Status).To(Equal(check.StatusFail))
		var currency bool
		for _, f := range report.Findings {
			if f.Message == "schemas/synnax/channel's current surface resolves "+
				"auth at v0, but schemas/x/auth's current version is v1" {
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

	It("Should pass a redeclaration that references a type redeclared with it", func() {
		write("schemas/synnax/versions/channel/v0.oracle", `
Old struct {
	a int32

	@go marshal
}

Inner struct {
	o Old

	@go marshal
}

Channel struct {
	key uuid @key
	inner Inner

	@go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
New struct {
	a int32

	@go marshal
}

Inner struct {
	o New

	@go marshal
}

Channel struct {
	key uuid @key
	inner Inner

	@go marshal
}
`)
		p := analyzeLive(`
@go output "core/pkg/service/channel"

New struct {
	a int32
}

Inner struct {
	o New
}

Channel struct {
	key uuid @key
	inner Inner
}
`)
		report := run(p)
		for _, f := range report.Findings {
			Expect(f.FixHint).ToNot(Equal("use an alias: Channel = v0.Channel"))
		}
	})

	aliasHints := func(report check.GateReport) []string {
		var hints []string
		for _, f := range report.Findings {
			if f.FixHint != "" {
				hints = append(hints, f.FixHint)
			}
		}
		return hints
	}

	It("Should fail an unchanged union with an inline variant", func() {
		const window = `
Window union on variant {
	rolling {
		span int32
	}
	fixed {
		start int32
	}

	@go marshal
}
`
		write("schemas/synnax/versions/channel/v0.oracle", window+`
Channel struct {
	key uuid @key
	window Window

	@go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", window+`
Channel struct {
	key uuid @key
	window Window
	name string

	@go marshal
}
`)
		p := analyzeLive(`
@go output "core/pkg/service/channel"

Window union on variant {
	rolling {
		span int32
	}
	fixed {
		start int32
	}
}

Channel struct {
	key uuid @key
	window Window
	name string
}
`)
		Expect(aliasHints(run(p))).To(ConsistOf("use an alias: Window = v0.Window"))
	})

	It("Should fail two unchanged structs that reference each other", func() {
		const pair = `
A struct {
	bs B[]

	@go marshal
}

B struct {
	as A[]

	@go marshal
}
`
		write("schemas/synnax/versions/channel/v0.oracle", pair+`
Channel struct {
	key uuid @key
	a A

	@go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", pair+`
Channel struct {
	key uuid @key
	a A
	name string

	@go marshal
}
`)
		p := analyzeLive(`
@go output "core/pkg/service/channel"

A struct {
	bs B[]
}

B struct {
	as A[]
}

Channel struct {
	key uuid @key
	a A
	name string
}
`)
		Expect(aliasHints(run(p))).To(ConsistOf(
			"use an alias: A = v0.A",
			"use an alias: B = v0.B",
		))
	})

	It("Should pass an unchanged struct that references a changed one", func() {
		write("schemas/synnax/versions/channel/v0.oracle", `
Ranges struct {
	x1 string[]

	@go marshal
}

Channel struct {
	key uuid @key
	ranges Ranges

	@go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
Ranges struct {
	x1 string[]
	custom int32

	@go marshal
}

Channel struct {
	key uuid @key
	ranges Ranges

	@go marshal
}
`)
		p := analyzeLive(`
@go output "core/pkg/service/channel"

Ranges struct {
	x1 string[]
	custom int32
}

Channel struct {
	key uuid @key
	ranges Ranges
}
`)
		Expect(aliasHints(run(p))).To(BeEmpty())
	})

	It("Should pass a struct whose field types move to another resource", func() {
		const state = `
State enum {
	idle = "idle"
	running = "running"
}
`
		write("schemas/synnax/versions/shared/v0.oracle", state)
		write(
			"schemas/synnax/shared.oracle",
			"@go output \"core/pkg/service/shared\"\n"+
				state,
		)
		write("schemas/synnax/versions/channel/v0.oracle", state+`
Channel struct {
	key uuid @key
	state State
	states State[]

	@go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
import "schemas/synnax/versions/shared/v0"

Channel struct {
	key uuid @key
	state shared.State
	states shared.State[]

	@go marshal
}
`)
		p := analyzeLive(`
import "schemas/synnax/shared"

@go output "core/pkg/service/channel"

Channel struct {
	key uuid @key
	state shared.State
	states shared.State[]
}
`)
		Expect(aliasHints(run(p))).To(BeEmpty())
	})

	It("Should pass a union whose inline variant changes only a default", func() {
		window := func(span int) string {
			return fmt.Sprintf(`
Window union on variant {
	rolling {
		span int32 = %d
	}
	fixed {
		start int32
	}

	@go marshal
}

Channel struct {
	key uuid @key
	window Window

	@go marshal
}
`, span)
		}
		write("schemas/synnax/versions/channel/v0.oracle", window(1))
		write("schemas/synnax/versions/channel/v1.oracle", window(2))
		p := analyzeLive(`
@go output "core/pkg/service/channel"

Window union on variant {
	rolling {
		span int32 = 2
	}
	fixed {
		start int32
	}
}

Channel struct {
	key uuid @key
	window Window
}
`)
		Expect(aliasHints(run(p))).To(BeEmpty())
	})

	Describe("Rules on a type aliased from a frozen definer", func() {
		const channelV1 = `
Entry = v0.Entry

Window = v0.Window

Channel struct {
	key uuid @key
	entries Entry[]
	window Window
	name string

	@go marshal
}
`
		frozen := func(entryRules, spanRules string) string {
			return `
Entry struct {
	min_val float64 = 0 {` + entryRules + `
	}
	max_val float64 = 1 {` + entryRules + `
	}
}

Window union on variant {
	rolling {
		span int32 {` + spanRules + `
		}
	}
}

Channel struct {
	key uuid @key
	entries Entry[]
	window Window

	@go marshal
}
`
		}
		live := func(entryRules, spanRules, entryGo string) string {
			return `
@go output "core/pkg/service/channel"

Entry struct {
	min_val float64 = 0 {` + entryRules + `
	}
	max_val float64 = 1 {` + entryRules + `
	}
` + entryGo + `
}

Window union on variant {
	rolling {
		span int32 {` + spanRules + `
		}
	}
}

Channel struct {
	key uuid @key
	entries Entry[]
	window Window
	name string
}
`
		}
		const (
			validate = "\n\t\t@validate max 17"
			group    = "\n\t\t@default group \"range\""
			fields   = "\t@go fields \"Size int\""
		)

		DescribeTable("Should fail a live rule that the frozen definer lacks",
			func(entryRules, spanRules, entryGo, want string) {
				write("schemas/synnax/versions/channel/v0.oracle", frozen("", ""))
				write("schemas/synnax/versions/channel/v1.oracle", channelV1)
				report := run(analyzeLive(live(entryRules, spanRules, entryGo)))
				Expect(report.Status).To(Equal(check.StatusFail))
				Expect(report.Findings).ToNot(BeEmpty())
				Expect(report.Findings[0].Path).
					To(Equal("schemas/synnax/versions/channel/v1.oracle"))
				Expect(report.Findings[0].Message).To(ContainSubstring(want))
				Expect(report.Findings[0].FixHint).
					To(Equal("the Go alias uses v0's rules: restate the rule in v0"))
			},
			Entry("on a validate rule", validate, "", "",
				"v1 aliases Entry from v0, but the live @validate on "+
					"Entry.min_val does not match v0's"),
			Entry("on a default group", group, "", "",
				"the live @default on Entry.min_val"),
			Entry("on a Go method rule", "", "", fields,
				"the live @go method rules on Entry"),
			Entry("on an inline variant payload", "", validate, "",
				"v1 aliases Window from v0, but the live @validate on "),
		)

		It("Should pass when the frozen definer restates the live rules", func() {
			write(
				"schemas/synnax/versions/channel/v0.oracle",
				frozen(validate+group, validate),
			)
			write("schemas/synnax/versions/channel/v1.oracle", channelV1)
			report := run(analyzeLive(live(validate+group, validate, "")))
			Expect(report.Findings).To(BeEmpty())
		})

		// channel v1 aliases v0's Entry, which pins label v0, which pins tone v0.
		writeDependencies := func(labelRules, toneRules string) {
			GinkgoHelper()
			write("schemas/synnax/versions/tone/v0.oracle", `
Tone struct {
	level int32 {`+toneRules+`
	}

	@go marshal
}
`)
			write("schemas/synnax/tone.oracle", `
@go output "core/pkg/service/tone"

Tone struct {
	level int32 {
		@validate max 9
	}
}
`)
			write("schemas/synnax/versions/label/v0.oracle", `
import "schemas/synnax/versions/tone/v0"

Label struct {
	key uuid @key
	name string {`+labelRules+`
	}
	tone tone.Tone

	@go marshal
}
`)
			write("schemas/synnax/label.oracle", `
import "schemas/synnax/tone"

@go output "core/pkg/service/label"

Label struct {
	key uuid @key
	name string {
		@validate required
	}
	tone tone.Tone
}
`)
			write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/synnax/versions/label/v0"

Entry struct {
	labels label.Label[]
}

Channel struct {
	key uuid @key
	entries Entry[]

	@go marshal
}
`)
			write("schemas/synnax/versions/channel/v1.oracle", `
Entry = v0.Entry

Channel struct {
	key uuid @key
	entries Entry[]
	name string

	@go marshal
}
`)
			write("schemas/synnax/channel.oracle", `
import "schemas/synnax/label"

@go output "core/pkg/service/channel"

Entry struct {
	labels label.Label[]
}

Channel struct {
	key uuid @key
	entries Entry[]
	name string
}
`)
		}
		runDependencies := func() check.GateReport {
			GinkgoHelper()
			loader := analyzer.NewStandardFileLoader(root)
			table, diag := analyzer.Analyze(
				GinkgoT().Context(),
				[]string{
					"schemas/synnax/tone",
					"schemas/synnax/label",
					"schemas/synnax/channel",
				},
				loader,
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			chains := MustSucceed(versions.Discover(root))
			return run(&pipeline.Result{
				Resolutions: table,
				Chains:      chains,
				Versions:    versions.NewResolver(chains, loader),
			})
		}
		const (
			labelRule = "\n\t\t@validate required"
			toneRule  = "\n\t\t@validate max 9"
		)

		DescribeTable(
			"Should check the rules of every dependency a frozen definer reaches",
			func(labelRules, toneRules string, want []string) {
				writeDependencies(labelRules, toneRules)
				report := runDependencies()
				messages := make([]string, len(report.Findings))
				for i, f := range report.Findings {
					messages[i] = f.Path + ": " + f.Message
				}
				Expect(messages).To(Equal(want))
			},
			Entry("when a direct dependency's version file lacks a live rule",
				"", toneRule,
				[]string{"schemas/synnax/versions/label/v0.oracle: frozen " +
					"schemas/synnax/channel v0 reaches schemas/synnax/label v0, but " +
					"the live @validate on Label.name does not match v0's"}),
			Entry("when a transitive dependency's version file lacks a live rule",
				labelRule, "",
				[]string{"schemas/synnax/versions/tone/v0.oracle: frozen " +
					"schemas/synnax/channel v0 reaches schemas/synnax/tone v0, but " +
					"the live @validate on Tone.level does not match v0's"}),
			Entry("when every dependency restates its live rules",
				labelRule, toneRule, []string{}),
		)

		It("Should fail when a reached dependency's chain cannot be resolved", func() {
			writeDependencies(labelRule, toneRule)
			write("schemas/synnax/versions/label/v1.oracle", "Label struct {{{\n")
			report := runDependencies()
			Expect(report.Status).To(Equal(check.StatusFail))
			Expect(report.Findings).To(ContainElement(And(
				HaveField("Path", "schemas/synnax/versions/channel/v1.oracle"),
				HaveField("Message", ContainSubstring("failed to parse")),
			)))
		})
	})
})
