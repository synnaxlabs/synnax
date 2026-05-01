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
)

var _ = Describe("AnalyzeGate", func() {
	It("passes when diagnostics are empty", func(ctx SpecContext) {
		gate := check.NewAnalyzeGate(false)
		r := &pipeline.Result{Diagnostics: &diagnostics.Diagnostics{}}
		Expect(gate.Run(ctx, r, check.Env{}).Status).To(Equal(check.StatusPass))
	})

	It("fails when an error diagnostic is present", func(ctx SpecContext) {
		diag := &diagnostics.Diagnostics{}
		diag.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Message:  "boom",
			File:     "schemas/x.oracle",
		})
		r := &pipeline.Result{Diagnostics: diag}
		report := check.NewAnalyzeGate(false).Run(ctx, r, check.Env{})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings).To(HaveLen(1))
		Expect(report.Findings[0].Severity).To(Equal(check.SeverityError))
	})

	It("surfaces warnings without failing by default", func(ctx SpecContext) {
		diag := &diagnostics.Diagnostics{}
		diag.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityWarning,
			Message:  "soft",
			File:     "schemas/x.oracle",
		})
		r := &pipeline.Result{Diagnostics: diag}
		report := check.NewAnalyzeGate(false).Run(ctx, r, check.Env{})
		Expect(report.Status).To(Equal(check.StatusPass))
		Expect(report.Findings).To(HaveLen(1))
		Expect(report.Findings[0].Severity).To(Equal(check.SeverityWarning))
	})

	It("promotes warnings to errors when WarningsAsErrors is set", func(ctx SpecContext) {
		diag := &diagnostics.Diagnostics{}
		diag.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityWarning,
			Message:  "soft",
			File:     "schemas/x.oracle",
		})
		r := &pipeline.Result{Diagnostics: diag}
		report := check.NewAnalyzeGate(true).Run(ctx, r, check.Env{})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Severity).To(Equal(check.SeverityError))
	})

	It("includes hint from notes when present", func(ctx SpecContext) {
		diag := &diagnostics.Diagnostics{}
		diag.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityError,
			Message:  "boom",
			File:     "schemas/x.oracle",
			Notes:    []diagnostics.Note{{Message: "try this"}},
		})
		r := &pipeline.Result{Diagnostics: diag}
		report := check.NewAnalyzeGate(false).Run(ctx, r, check.Env{})
		Expect(report.Findings[0].FixHint).To(Equal("try this"))
	})
})
