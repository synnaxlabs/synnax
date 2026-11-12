// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package formatter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/formatter"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	xlsp "github.com/synnaxlabs/x/lsp"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

// rootOf returns the root scope of the analyzed source.
func rootOf(ctx SpecContext, source string) *symbol.Symbol {
	GinkgoHelper()
	aCtx := acontext.NewRoot(ctx, MustSucceed(parser.Parse(source)), NewRoot(nil))
	analyzer.AnalyzeProgram(aCtx)
	return aCtx.Scope.Root()
}

var _ = Describe("AddImport", func() {
	It("Should add a loose import to a file with no import", func(ctx SpecContext) {
		edits := formatter.AddImport(rootOf(ctx, "func f() {}\n"), "time")
		Expect(edits).To(Equal([]protocol.TextEdit{{NewText: "import time\n\n"}}))
	})

	It("Should add a loose import with no root scope", func() {
		Expect(formatter.AddImport(nil, "time")).
			To(Equal([]protocol.TextEdit{{NewText: "import time\n\n"}}))
	})

	It(
		"Should put an existing import and the new one in one block",
		func(ctx SpecContext) {
			edits := formatter.AddImport(rootOf(ctx, "import math as m\n"), "time")
			Expect(edits).To(Equal([]protocol.TextEdit{{
				Range:   protocol.Range{End: protocol.Position{Character: 16}},
				NewText: "import (\n    math as m\n    time\n)\n",
			}}))
		},
	)

	It("Should remove each import statement after the first", func(ctx SpecContext) {
		edits := formatter.AddImport(
			rootOf(ctx, "import math\nimport stable\n"),
			"time",
		)
		Expect(edits).To(Equal([]protocol.TextEdit{
			{
				Range:   protocol.Range{End: protocol.Position{Character: 11}},
				NewText: "import (\n    math\n    stable\n    time\n)\n",
			},
			{Range: protocol.Range{
				Start: protocol.Position{Line: 1},
				End:   protocol.Position{Line: 2},
			}},
		}))
	})

	DescribeTable("Should give no edit for a module the file imports",
		func(ctx SpecContext, source string) {
			Expect(formatter.AddImport(rootOf(ctx, source), "time")).To(BeEmpty())
		},
		Entry("by its name", "import time\n"),
		Entry("with an alias", "import time as t\n"),
	)
})

var _ = Describe("RemoveImport", func() {
	DescribeTable("Should remove the import",
		func(ctx SpecContext, source, alias, expected string) {
			item := rootOf(ctx, source).FindChild(alias).AST.(parser.IImportItemContext)
			edit := formatter.RemoveImport(item)
			Expect(xlsp.ApplyIncrementalChange(
				source,
				&protocol.TextDocumentContentChangePartial{
					Range: edit.Range,
					Text:  edit.NewText,
				},
			)).To(Equal(expected))
		},
		Entry(
			"with its statement when it is alone",
			"import time\n\nfunc f() {}\n",
			"time",
			"\nfunc f() {}\n",
		),
		Entry(
			"when it is the first of a block",
			"import (\n    math\n    time\n)\n",
			"math",
			"import (\n    time\n)\n",
		),
		Entry(
			"when it is the last of a block",
			"import (\n    math\n    time\n)\n",
			"time",
			"import (\n    math\n)\n",
		),
	)
})
