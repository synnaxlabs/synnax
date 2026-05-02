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
})
