// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/doc"
)

// examples returns the Arc code blocks of d.
func examples(d doc.Doc) []string {
	var code []string
	for _, block := range d.Blocks() {
		if body, ok := strings.CutPrefix(block.Render(), "```arc\n"); ok {
			code = append(code, strings.TrimSuffix(body, "\n```"))
		}
	}
	return code
}

var _ = Describe("Hover examples", func() {
	It("Should compile every example in the math module", func(ctx SpecContext) {
		var code []string
		for _, s := range stl.NewSymbols() {
			if s.Name != "math" || s.Kind != symbol.KindModule {
				continue
			}
			code = append(code, examples(s.Doc)...)
			for _, member := range s.Children() {
				code = append(code, examples(member.Doc)...)
			}
		}
		Expect(code).ToNot(BeEmpty())
		channels := map[string]types.Type{
			"sensor":      types.F64(),
			"output":      types.F64(),
			"rate_output": types.F64(),
		}
		for _, example := range code {
			ambient := stl.NewSymbols()
			id := 1
			for name, t := range channels {
				ambient = append(ambient, &symbol.Symbol{
					Name: name,
					Kind: symbol.KindChannel,
					Type: types.Chan(t),
					ID:   id,
				})
				id++
			}
			_, err := arc.CompileText(
				ctx,
				arc.Text{Raw: "import math\n" + example},
				symbol.NewRoot(nil, ambient),
			)
			Expect(err).ToNot(HaveOccurred(), example)
		}
	})
})
