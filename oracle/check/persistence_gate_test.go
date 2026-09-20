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
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("PersistenceGate", func() {
	var root string

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		dir := filepath.Join(root, "schemas/synnax/versions/channel")
		Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "v0.oracle"), []byte(`
Entry struct {
	key uuid @key

	@go marshal
}
`), 0o644)).To(Succeed())
	})

	run := func(ctx context.Context, live string) check.GateReport {
		GinkgoHelper()
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			ctx, live, "schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag == nil || diag.Ok()).To(BeTrue())
		chains := MustSucceed(versions.Discover(root))
		r := &pipeline.Result{
			Resolutions: table,
			Chains:      chains,
			Versions: versions.NewResolver(
				chains, analyzer.NewStandardFileLoader(root),
			),
		}
		return check.NewPersistenceGate(false).Run(ctx, r, check.Env{})
	}

	messages := func(r check.GateReport) []string {
		out := make([]string, len(r.Findings))
		for i, f := range r.Findings {
			out[i] = f.Message
		}
		return out
	}

	It("Should pass a current-surface member", func(ctx SpecContext) {
		r := run(ctx, `
			@go output "out"
			Entry struct {
				key uuid @key

				@go marshal
			}
		`)
		Expect(r.Findings).To(BeEmpty())
	})

	It(
		"Should warn on a persisted type absent from the current version file",
		func(ctx SpecContext) {
			r := run(ctx, `
			@go output "out"
			Entry struct {
				key uuid @key
				sibling Sibling

				@go marshal
			}
			Sibling struct {
				name string
			}
		`)
			Expect(messages(r)).To(ContainElement(
				"channel.Sibling is persisted but absent from its resource's " +
					"current version file",
			))
		},
	)

	It("Should pass when the pipeline produced no resolutions", func(
		ctx SpecContext,
	) {
		r := check.NewPersistenceGate(true).Run(ctx, &pipeline.Result{}, check.Env{})
		Expect(r.Status).To(Equal(check.StatusPass))
		Expect(r.Findings).To(BeEmpty())
	})

	It("Should promote warnings to errors when configured", func(ctx SpecContext) {
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			ctx, `
			@go output "out"
			Entry struct {
				key uuid @key
				sibling Sibling

				@go marshal
			}
			Sibling struct {
				name string
			}
		`, "schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag == nil || diag.Ok()).To(BeTrue())
		chains := MustSucceed(versions.Discover(root))
		p := &pipeline.Result{
			Resolutions: table,
			Chains:      chains,
			Schemas:     []string{"schemas/synnax/channel.oracle"},
			Versions: versions.NewResolver(
				chains, analyzer.NewStandardFileLoader(root),
			),
		}
		r := check.NewPersistenceGate(true).Run(ctx, p, check.Env{})
		Expect(r.Status).To(Equal(check.StatusFail))
		Expect(r.Findings[0].Severity).To(Equal(check.SeverityError))
		Expect(r.Findings[0].Path).To(Equal("schemas/synnax/channel.oracle"))
	})

	It("Should fail when the chain survey errors", func(ctx SpecContext) {
		Expect(os.WriteFile(filepath.Join(
			root, "schemas/synnax/versions/channel/v0.oracle",
		), []byte("Entry struct {{{\n"), 0o644)).To(Succeed())
		r := run(ctx, `
			@go output "out"
			Entry struct {
				key uuid @key

				@go marshal
			}
		`)
		Expect(r.Status).To(Equal(check.StatusFail))
		Expect(r.Findings[0].Message).To(ContainSubstring("failed to parse"))
	})
})
