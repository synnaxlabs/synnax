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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

func fileDiag(d diagnostics.Diagnostic) *diagnostics.Files {
	diag := diagnostics.NewFiles()
	diag.Report("schemas/x.oracle", d)
	return diag
}

var _ = Describe("AnalyzeGate", func() {
	It("passes when diagnostics are empty", func(ctx SpecContext) {
		gate := check.NewAnalyzeGate(false)
		r := &pipeline.Result{Diagnostics: diagnostics.NewFiles()}
		Expect(gate.Run(ctx, r, check.Env{}).Status).To(Equal(check.StatusPass))
	})

	It("fails when an error diagnostic is present", func(ctx SpecContext) {
		r := &pipeline.Result{Diagnostics: fileDiag(diagnostics.Diagnostic{
			Severity: protocol.DiagnosticSeverityError,
			Message:  "boom",
		})}
		report := check.NewAnalyzeGate(false).Run(ctx, r, check.Env{})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings).To(HaveLen(1))
		Expect(report.Findings[0].Severity).To(Equal(check.SeverityError))
		Expect(report.Findings[0].Path).To(Equal("schemas/x.oracle"))
	})

	It("surfaces warnings without failing by default", func(ctx SpecContext) {
		r := &pipeline.Result{Diagnostics: fileDiag(diagnostics.Diagnostic{
			Severity: protocol.DiagnosticSeverityWarning,
			Message:  "soft",
		})}
		report := check.NewAnalyzeGate(false).Run(ctx, r, check.Env{})
		Expect(report.Status).To(Equal(check.StatusPass))
		Expect(report.Findings).To(HaveLen(1))
		Expect(report.Findings[0].Severity).To(Equal(check.SeverityWarning))
	})

	It(
		"promotes warnings to errors when WarningsAsErrors is set",
		func(ctx SpecContext) {
			r := &pipeline.Result{Diagnostics: fileDiag(diagnostics.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  "soft",
			})}
			report := check.NewAnalyzeGate(true).Run(ctx, r, check.Env{})
			Expect(report.Status).To(Equal(check.StatusFail))
			Expect(report.Findings[0].Severity).To(Equal(check.SeverityError))
		},
	)

	It("includes hint from notes when present", func(ctx SpecContext) {
		r := &pipeline.Result{Diagnostics: fileDiag(diagnostics.Diagnostic{
			Severity: protocol.DiagnosticSeverityError,
			Message:  "boom",
			Notes:    []protocol.DiagnosticRelatedInformation{{Message: "try this"}},
		})}
		report := check.NewAnalyzeGate(false).Run(ctx, r, check.Env{})
		Expect(report.Findings[0].FixHint).To(Equal("try this"))
	})
})
