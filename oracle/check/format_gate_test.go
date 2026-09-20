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
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FormatGate", func() {
	It("passes when sources match formatted bytes", func(ctx SpecContext) {
		gate := check.NewFormatGate()
		r := &pipeline.Result{
			Schemas: []string{"schemas/x.oracle"},
			Sources: map[string][]byte{
				"schemas/x.oracle": []byte("body\n"),
			},
			FormattedSources: map[string][]byte{
				"schemas/x.oracle": []byte("body\n"),
			},
		}
		report := gate.Run(ctx, r, check.Env{})
		Expect(report.Status).To(Equal(check.StatusPass))
		Expect(report.Findings).To(BeEmpty())
	})

	It("fails when a source diverges from canonical", func(ctx SpecContext) {
		gate := check.NewFormatGate()
		r := &pipeline.Result{
			Schemas: []string{"schemas/x.oracle"},
			Sources: map[string][]byte{
				"schemas/x.oracle": []byte("uncanonical"),
			},
			FormattedSources: map[string][]byte{
				"schemas/x.oracle": []byte("canonical\n"),
			},
		}
		report := gate.Run(ctx, r, check.Env{})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings).To(HaveLen(1))
		Expect(report.Findings[0].Path).To(Equal("schemas/x.oracle"))
		Expect(report.Findings[0].FixHint).To(ContainSubstring("oracle fmt"))
		Expect(report.Findings[0].Diff).To(BeEmpty())
	})

	It("captures diff when env requests it", func(ctx SpecContext) {
		gate := check.NewFormatGate()
		r := &pipeline.Result{
			Schemas: []string{"schemas/x.oracle"},
			Sources: map[string][]byte{
				"schemas/x.oracle": []byte("a\nb\n"),
			},
			FormattedSources: map[string][]byte{
				"schemas/x.oracle": []byte("a\nB\n"),
			},
		}
		report := gate.Run(ctx, r, check.Env{IncludeDiffs: true})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Diff).To(ContainSubstring("-b"))
		Expect(report.Findings[0].Diff).To(ContainSubstring("+B"))
	})

	It("truncates long diffs and marks unequal tails", func(ctx SpecContext) {
		gate := check.NewFormatGate()
		var raw, canonical strings.Builder
		for i := range 60 {
			fmt.Fprintf(&raw, "line %d\n", i)
			fmt.Fprintf(&canonical, "LINE %d\n", i)
		}
		canonical.WriteString("extra tail\n")
		r := &pipeline.Result{
			Schemas: []string{"schemas/x.oracle"},
			Sources: map[string][]byte{
				"schemas/x.oracle": []byte(raw.String()),
			},
			FormattedSources: map[string][]byte{
				"schemas/x.oracle": []byte(canonical.String()),
			},
		}
		report := gate.Run(ctx, r, check.Env{IncludeDiffs: true})
		Expect(report.Findings[0].Diff).
			To(ContainSubstring("remain unprocessed"))
	})

	It("diffs pure insertions and deletions", func(ctx SpecContext) {
		gate := check.NewFormatGate()
		r := &pipeline.Result{
			Schemas: []string{"schemas/a.oracle", "schemas/b.oracle"},
			Sources: map[string][]byte{
				"schemas/a.oracle": []byte("a\n"),
				"schemas/b.oracle": []byte("a\nb\nc\n"),
			},
			FormattedSources: map[string][]byte{
				"schemas/a.oracle": []byte("a\nb\nc\n"),
				"schemas/b.oracle": []byte("a\n"),
			},
		}
		report := gate.Run(ctx, r, check.Env{IncludeDiffs: true})
		Expect(report.Findings).To(HaveLen(2))
		Expect(report.Findings[0].Diff).To(ContainSubstring("+b"))
		Expect(report.Findings[1].Diff).To(ContainSubstring("-b"))
	})

	Describe("version files", func() {
		var root string

		BeforeEach(func() {
			root = GinkgoT().TempDir()
		})

		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}

		result := func() *pipeline.Result {
			GinkgoHelper()
			return &pipeline.Result{
				Chains: MustSucceed(versions.Discover(root)),
			}
		}

		It("passes canonically formatted version files", func(ctx SpecContext) {
			write(
				"schemas/synnax/versions/channel/v0.oracle",
				MustSucceed(formatter.Format("Key = uuid\n")),
			)
			report := check.NewFormatGate().Run(
				ctx, result(), check.Env{RepoRoot: root},
			)
			Expect(report.Findings).To(BeEmpty())
			Expect(report.Status).To(Equal(check.StatusPass))
		})

		It("fails an unformatted version file", func(ctx SpecContext) {
			write(
				"schemas/synnax/versions/channel/v0.oracle",
				"Key    =     uuid\n",
			)
			report := check.NewFormatGate().Run(
				ctx, result(), check.Env{RepoRoot: root},
			)
			Expect(report.Status).To(Equal(check.StatusFail))
			Expect(report.Findings[0].FixHint).To(ContainSubstring("oracle fmt"))
		})

		It("fails a version file missing on disk", func(ctx SpecContext) {
			write("schemas/synnax/versions/channel/v0.oracle", "Key = uuid\n")
			p := result()
			Expect(os.Remove(filepath.Join(
				root, "schemas/synnax/versions/channel/v0.oracle",
			))).To(Succeed())
			report := check.NewFormatGate().Run(ctx, p, check.Env{RepoRoot: root})
			Expect(report.Status).To(Equal(check.StatusFail))
			Expect(report.Findings[0].Message).
				To(ContainSubstring("failed to read version file"))
		})
	})
})
