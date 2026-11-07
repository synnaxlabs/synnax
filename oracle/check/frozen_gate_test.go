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
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/frozen"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FrozenGate", func() {
	const (
		sumFile = "schemas/synnax/frozen/doc.sum"
		v0Dir   = "core/pkg/service/doc/versions/v0"
		v0File  = v0Dir + "/types.gen.go"
		v1File  = "core/pkg/service/doc/versions/v1/types.gen.go"
	)
	var root string
	write := func(rel, content string) {
		GinkgoHelper()
		full := filepath.Join(root, rel)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
	}
	result := func(ctx context.Context, outputs ...string) *pipeline.Result {
		GinkgoHelper()
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(ctx, []string{"schemas/synnax/doc"}, loader)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		files := make([]plugin.File, 0, len(outputs))
		for _, path := range outputs {
			files = append(files, plugin.File{Path: path})
		}
		return &pipeline.Result{
			Resolutions: table,
			Versions:    versions.NewResolver(chains, loader),
			Outputs:     map[string][]plugin.File{"stub": files},
		}
	}
	run := func(ctx context.Context, res *pipeline.Result) check.GateReport {
		GinkgoHelper()
		return check.NewFrozenGate().Run(ctx, res, check.Env{RepoRoot: root})
	}

	BeforeEach(func(ctx SpecContext) {
		root = GinkgoT().TempDir()
		write("schemas/synnax/versions/doc/v0.oracle", `
Doc struct {
	key uuid @key

	@go marshal
}
`)
		write("schemas/synnax/versions/doc/v1.oracle", `
Doc = v0.Doc
`)
		write("schemas/synnax/doc.oracle", `
@go output "core/pkg/service/doc"

Doc struct {
	key uuid @key

	@go marshal
}
`)
		write(v0File, "v0")
		write(v1File, "v1")
		chains := MustSucceed(frozen.Compute(ctx, result(ctx, v0File, v1File), root))
		Expect(frozen.Update(chains, root, false)).To(Succeed())
	})

	It("Should be named frozen", func() {
		Expect(check.NewFrozenGate().Name()).To(Equal("frozen"))
	})

	It("Should pass when the disk matches the sums", func(ctx SpecContext) {
		r := run(ctx, result(ctx, v0File, v1File))
		Expect(r.Status).To(Equal(check.StatusPass))
		Expect(r.Elapsed).To(BeNumerically(">", 0))
	})

	It("Should pass with no version chains", func(ctx SpecContext) {
		Expect(run(ctx, &pipeline.Result{}).Status).To(Equal(check.StatusPass))
	})

	It("Should fail when a frozen file changes", func(ctx SpecContext) {
		write(v0File, "edited")
		r := run(ctx, result(ctx, v0File, v1File))
		Expect(r.Status).To(Equal(check.StatusFail))
		Expect(r.Findings).To(HaveLen(1))
		Expect(r.Findings[0].Path).To(Equal(v0Dir))
		Expect(r.Findings[0].FixHint).To(ContainSubstring("--refreeze"))
	})

	It("Should fail when a frozen package stops generating", func(ctx SpecContext) {
		r := run(ctx, result(ctx, v1File))
		Expect(r.Status).To(Equal(check.StatusFail))
		Expect(r.Findings).To(HaveLen(1))
		Expect(r.Findings[0].Path).To(Equal(v0Dir))
	})

	It("Should ask for a sync when the current package changes", func(ctx SpecContext) {
		write(v1File, "edited")
		r := run(ctx, result(ctx, v0File, v1File))
		Expect(r.Findings).To(HaveLen(1))
		Expect(r.Findings[0].Path).To(Equal(sumFile))
		Expect(r.Findings[0].FixHint).To(Equal("run `oracle sync`"))
	})

	It("Should ask for a sync when no sum file exists", func(ctx SpecContext) {
		Expect(os.Remove(filepath.Join(root, sumFile))).To(Succeed())
		r := run(ctx, result(ctx, v0File, v1File))
		Expect(r.Status).To(Equal(check.StatusFail))
		Expect(r.Findings).To(HaveLen(2))
		Expect(r.Findings[0].FixHint).To(Equal("run `oracle sync`"))
	})

	It("Should report a malformed sum file", func(ctx SpecContext) {
		write(sumFile, "broken\n")
		r := run(ctx, result(ctx, v0File, v1File))
		Expect(r.Status).To(Equal(check.StatusFail))
		Expect(r.Findings[0].Path).To(Equal(sumFile))
	})

	It("Should pass after a sync with a template-only change", func(ctx SpecContext) {
		res := result(ctx, v0File, v1File)
		for i := range res.Outputs["stub"] {
			res.Outputs["stub"][i].Content = []byte("new template")
		}
		Expect(frozen.Pin(ctx, res, root)).To(Succeed())
		for _, f := range res.Outputs["stub"] {
			write(f.Path, string(f.Content))
		}
		chains := MustSucceed(frozen.Compute(ctx, res, root))
		Expect(frozen.Update(chains, root, false)).To(Succeed())
		Expect(run(ctx, res).Status).To(Equal(check.StatusPass))
		Expect(os.ReadFile(filepath.Join(root, v0File))).To(Equal([]byte("v0")))
	})

	It("Should fail on a removed line of a frozen package", func(ctx SpecContext) {
		recorded := MustSucceed(frozen.Read(sumFile, root))
		write(sumFile, fmt.Sprintf(
			"v1 %s %s %s\n", recorded[1].Schema, recorded[1].Live, recorded[1].Output,
		))
		r := run(ctx, result(ctx, v0File, v1File))
		Expect(r.Status).To(Equal(check.StatusFail))
		Expect(r.Findings).To(HaveLen(1))
		Expect(r.Findings[0].Path).To(Equal(v0Dir))
		Expect(r.Findings[0].FixHint).To(ContainSubstring("--refreeze"))
	})

	It("Should fail on a sum file that no chain owns", func(ctx SpecContext) {
		const orphan = "schemas/synnax/frozen/gone.sum"
		write(orphan, "v0 s l o\n")
		r := run(ctx, result(ctx, v0File, v1File))
		Expect(r.Status).To(Equal(check.StatusFail))
		Expect(r.Findings).To(HaveLen(1))
		Expect(r.Findings[0].Path).To(Equal(orphan))
		Expect(r.Findings[0].FixHint).To(Equal("run `oracle sync`"))
	})
})
