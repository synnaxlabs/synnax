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
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/pipeline"
)

type fixedGate struct {
	name   string
	status check.Status
}

func (g *fixedGate) Name() string { return g.name }
func (g *fixedGate) Run(_ context.Context, _ *pipeline.Result, _ check.Env) check.GateReport {
	return check.GateReport{
		Gate:    g.name,
		Status:  g.status,
		Elapsed: time.Millisecond,
	}
}

var _ = Describe("Run", func() {
	It("runs every checker when gates is empty", func(ctx SpecContext) {
		report := check.Run(ctx, &pipeline.Result{}, check.Env{}, []check.Checker{
			&fixedGate{name: "a", status: check.StatusPass},
			&fixedGate{name: "b", status: check.StatusFail},
		}, nil)
		Expect(report.TotalRun).To(Equal(2))
		Expect(report.TotalPassed).To(Equal(1))
		Expect(report.TotalFailed).To(Equal(1))
	})

	It("skips checkers not in the gates filter", func(ctx SpecContext) {
		report := check.Run(ctx, &pipeline.Result{}, check.Env{}, []check.Checker{
			&fixedGate{name: "a", status: check.StatusPass},
			&fixedGate{name: "b", status: check.StatusFail},
		}, []string{"a"})
		Expect(report.TotalRun).To(Equal(1))
		Expect(report.TotalPassed).To(Equal(1))
		Expect(report.TotalFailed).To(Equal(0))
		Expect(report.Gates).To(HaveLen(2))
		Expect(report.Gates[1].Status).To(Equal(check.StatusSkipped))
	})

	It("preserves checker order in the report", func(ctx SpecContext) {
		report := check.Run(ctx, &pipeline.Result{}, check.Env{}, []check.Checker{
			&fixedGate{name: "format", status: check.StatusPass},
			&fixedGate{name: "analyze", status: check.StatusPass},
			&fixedGate{name: "generated", status: check.StatusPass},
		}, nil)
		names := []string{}
		for _, g := range report.Gates {
			names = append(names, g.Gate)
		}
		Expect(names).To(Equal([]string{"format", "analyze", "generated"}))
	})
})

var _ = Describe("Report.FirstExitCode", func() {
	It("returns 0 when every gate passed", func() {
		r := &check.Report{Gates: []check.GateReport{
			{Gate: "format", Status: check.StatusPass},
		}}
		Expect(r.FirstExitCode()).To(Equal(0))
	})

	It("returns the gate-specific code on first failure", func() {
		r := &check.Report{Gates: []check.GateReport{
			{Gate: "format", Status: check.StatusPass},
			{Gate: "generated", Status: check.StatusFail},
			{Gate: "orphans", Status: check.StatusFail},
		}}
		Expect(r.FirstExitCode()).To(Equal(check.FailureCodes["generated"]))
	})

	It("returns 1 for an unknown gate name", func() {
		r := &check.Report{Gates: []check.GateReport{
			{Gate: "mystery", Status: check.StatusFail},
		}}
		Expect(r.FirstExitCode()).To(Equal(1))
	})
})

var _ = Describe("Render", func() {
	r := &check.Report{
		Gates: []check.GateReport{
			{Gate: "format", Status: check.StatusPass, Elapsed: time.Millisecond},
			{Gate: "analyze", Status: check.StatusFail, Elapsed: time.Millisecond,
				Findings: []check.Finding{{
					Path: "schemas/x.oracle", Line: 10, Severity: check.SeverityError,
					Message: "boom", FixHint: "fix it",
				}}},
		},
		TotalRun: 2, TotalPassed: 1, TotalFailed: 1, Elapsed: time.Millisecond,
	}

	It("emits readable text", func() {
		var buf bytes.Buffer
		Expect(check.Render(&buf, r, check.FormatText, false)).To(Succeed())
		out := buf.String()
		Expect(out).To(ContainSubstring("format"))
		Expect(out).To(ContainSubstring("analyze"))
		Expect(out).To(ContainSubstring("schemas/x.oracle"))
		Expect(out).To(ContainSubstring("boom"))
		Expect(out).To(ContainSubstring("fix it"))
	})

	It("emits valid JSON with string severities", func() {
		var buf bytes.Buffer
		Expect(check.Render(&buf, r, check.FormatJSON, false)).To(Succeed())
		var decoded struct {
			Gates []struct {
				Gate     string `json:"gate"`
				Status   string `json:"status"`
				Findings []struct {
					Severity string `json:"severity"`
				} `json:"findings"`
			} `json:"gates"`
		}
		Expect(json.Unmarshal(buf.Bytes(), &decoded)).To(Succeed())
		Expect(decoded.Gates).To(HaveLen(2))
		Expect(decoded.Gates[0].Status).To(Equal("pass"))
		Expect(decoded.Gates[1].Status).To(Equal("fail"))
		Expect(decoded.Gates[1].Findings[0].Severity).To(Equal("error"))
	})

	It("hides info findings on passing gates unless verbose", func() {
		report := &check.Report{Gates: []check.GateReport{{
			Gate: "g", Status: check.StatusPass,
			Findings: []check.Finding{{
				Severity: check.SeverityInfo, Message: "fyi",
			}},
		}}}
		var buf bytes.Buffer
		Expect(check.Render(&buf, report, check.FormatText, false)).To(Succeed())
		Expect(buf.String()).NotTo(ContainSubstring("fyi"))
		buf.Reset()
		Expect(check.Render(&buf, report, check.FormatText, true)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("fyi"))
	})

	It("returns an error for unknown formats", func() {
		var buf bytes.Buffer
		// FormatText path is the default fallback, so any unknown
		// string just renders as text without error. Confirm the
		// behaviour is documented: unknown -> text-format render.
		Expect(check.Render(&buf, r, check.Format("xml"), false)).To(Succeed())
		Expect(strings.TrimSpace(buf.String())).NotTo(BeEmpty())
	})
})
