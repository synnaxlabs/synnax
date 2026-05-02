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
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

type erroringFormatter struct{}

func (erroringFormatter) Format(_ context.Context, _ []byte, _ string) ([]byte, error) {
	return nil, errors.New("synthetic formatter failure")
}

var _ = Describe("GeneratedGate", func() {
	var (
		repoRoot   string
		formatters *format.Registry
	)

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "gen"))
		DeferCleanup(func() { Expect(os.RemoveAll(repoRoot)).To(Succeed()) })
		formatters = format.NewRegistry()
	})

	resultWith := func(files []plugin.File) *pipeline.Result {
		return &pipeline.Result{
			Outputs: map[string][]plugin.File{"stub": files},
		}
	}

	It("passes when on-disk content matches generated output", func(ctx SpecContext) {
		Expect(os.MkdirAll(filepath.Join(repoRoot, "out"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(repoRoot, "out", "x.gen.go"), []byte("hi"), 0644)).To(Succeed())
		gate := check.NewGeneratedGate(formatters, 1)
		report := gate.Run(ctx, resultWith([]plugin.File{
			{Path: "out/x.gen.go", Content: []byte("hi")},
		}), check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusPass))
	})

	It("flags missing files", func(ctx SpecContext) {
		gate := check.NewGeneratedGate(formatters, 1)
		report := gate.Run(ctx, resultWith([]plugin.File{
			{Path: "out/missing.gen.go", Content: []byte("hi")},
		}), check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Message).To(ContainSubstring("missing on disk"))
	})

	It("flags content mismatch and includes diff when requested", func(ctx SpecContext) {
		Expect(os.MkdirAll(filepath.Join(repoRoot, "out"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(repoRoot, "out", "x.gen.go"), []byte("old"), 0644)).To(Succeed())
		gate := check.NewGeneratedGate(formatters, 1)
		report := gate.Run(ctx, resultWith([]plugin.File{
			{Path: "out/x.gen.go", Content: []byte("new")},
		}), check.Env{RepoRoot: repoRoot, IncludeDiffs: true})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Message).To(ContainSubstring("does not match"))
		Expect(report.Findings[0].Diff).To(ContainSubstring("-old"))
		Expect(report.Findings[0].Diff).To(ContainSubstring("+new"))
	})

	It("captures per-file formatter failures without aborting the gate", func(ctx SpecContext) {
		Expect(os.MkdirAll(filepath.Join(repoRoot, "out"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(repoRoot, "out", "ok.gen.go"), []byte("hi"), 0644)).To(Succeed())
		formatters.Register(".bad", &erroringFormatter{})
		gate := check.NewGeneratedGate(formatters, 1)
		report := gate.Run(ctx, resultWith([]plugin.File{
			{Path: "out/ok.gen.go", Content: []byte("hi")},
			{Path: "out/x.bad", Content: []byte("hi")},
		}), check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings).To(HaveLen(1))
		Expect(report.Findings[0].Message).To(ContainSubstring("formatter failed"))
	})
})
