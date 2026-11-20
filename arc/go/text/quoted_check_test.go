// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Quoted channel names, checker cases", func() {
	chans := StaticResolver{
		{Name: "math", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10},
		{Name: "outf", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 20},
	}
	analyze := func(ctx SpecContext, source string) string {
		GinkgoHelper()
		prog, err := text.Parse(text.Text{Raw: source})
		if err != nil {
			return err.Error()
		}
		_, diagnostics := text.Analyze(
			ctx,
			prog,
			symbol.NewRoot(chans, stl.NewSymbols()),
		)
		return diagnostics.String()
	}

	It("Should not accept an empty quoted channel name", func(ctx SpecContext) {
		Expect(
			analyze(ctx, "func f() f64 {\n\treturn ''\n}"),
		).ToNot(Equal("analysis successful"))
	})

	It("Should still flag an import that only a quoted channel name spells", func(
		ctx SpecContext,
	) {
		Expect(analyze(ctx, "import math\n\n'math' * 2.0 -> 'outf'")).
			To(ContainSubstring(`unused import: "math"`))
	})
})
