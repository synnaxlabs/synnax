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
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// hoverExamples returns one table entry for each Arc code block in the hover docs of
// the float members of the math module.
func hoverExamples() []TableEntry {
	// The math.avg reset example does not compile on this branch.
	floatMembers := set.New("is_nan", "is_infinite", "is_finite")
	var entries []TableEntry
	add := func(name string, d doc.Doc) {
		for i, block := range d.Blocks() {
			if code, ok := strings.CutPrefix(block.Render(), "```arc\n"); ok {
				entries = append(entries, Entry(
					fmt.Sprintf("%s block %d", name, i),
					strings.TrimSuffix(code, "\n```"),
				))
			}
		}
	}
	for _, s := range stl.NewSymbols() {
		if s.Name != "math" || s.Kind != symbol.KindModule {
			continue
		}
		for _, member := range s.Children() {
			if floatMembers.Contains(member.Name) {
				add("math."+member.Name, member.Doc)
			}
		}
	}
	return entries
}

// newExampleRoot returns a root with the channels the hover examples use.
func newExampleRoot() *symbol.Symbol {
	ambient := stl.NewSymbols()
	for i, name := range []string{
		"sensor", "output", "rate_output", "pump_speed", "pump_stopped",
	} {
		ambient = append(ambient, &symbol.Symbol{
			Name: name,
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   i + 1,
		})
	}
	return symbol.NewRoot(nil, ambient)
}

var _ = Describe("Hover examples", func() {
	DescribeTable("Should compile",
		func(ctx SpecContext, code string) {
			MustSucceed(arc.CompileText(
				ctx,
				arc.Text{Raw: "import math\n" + code},
				newExampleRoot(),
			))
		},
		hoverExamples(),
	)
})
