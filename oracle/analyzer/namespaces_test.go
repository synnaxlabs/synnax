// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Namespaces", func() {
	Describe("UsedNamespaces", func() {
		It("Should return each qualifier that the file references", func() {
			Expect(analyzer.UsedNamespaces(MustSucceed(parser.Parse(`
				Range struct extends base.Range {
					span telem.TimeRange
					labels label.Key[]
				}
			`)))).To(Equal(set.New("base", "telem", "label")))
		})

		It("Should add the filter namespace when the file declares a filter", func() {
			Expect(analyzer.UsedNamespaces(MustSucceed(parser.Parse(`
				Thing struct {
					name string
				}
				Where filter Thing {
					name
				}
			`)))).To(Equal(set.New(resolution.FilterNamespace)))
		})
	})

	Describe("QualifiedNames", func() {
		It(
			"Should return the qualified names of a fragment that does not parse",
			func() {
				Expect(analyzer.QualifiedNames(`action SetCode {
				code status.Code
				span telem.TimeSpan
			}`)).To(Equal(set.New("status.Code", "telem.TimeSpan")))
			},
		)

		It("Should skip a dotted name inside a string", func() {
			Expect(
				analyzer.QualifiedNames(`@doc value "see status.Code"`),
			).To(BeEmpty())
		})
	})
})
